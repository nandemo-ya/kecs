package ssm_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	ssmIntegration "github.com/nandemo-ya/kecs/controlplane/internal/integrations/ssm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("SSM Integration", func() {
	var (
		integration ssmIntegration.Integration
		kubeClient  *fake.Clientset
		mockSSM     *mockSSMClient
		config      *ssmIntegration.Config
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()
		mockSSM = &mockSSMClient{
			parameters: make(map[string]*ssm.GetParameterOutput),
		}
		config = &ssmIntegration.Config{
			LocalStackEndpoint: "http://localhost:4566",
			SecretPrefix:       "ssm-",
			KubeNamespace:      "default",
			SyncRetries:        3,
			CacheTTL:           5 * time.Minute,
		}
		integration = ssmIntegration.NewIntegrationWithClient(kubeClient, mockSSM, config)
	})

	Describe("GetParameter", func() {
		It("should retrieve a parameter from SSM", func() {
			// Setup mock parameter
			paramName := "/app/database/password"
			paramValue := "secret-password-123"
			mockSSM.parameters[paramName] = &ssm.GetParameterOutput{
				Parameter: &types.Parameter{
					Name:             aws.String(paramName),
					Value:            aws.String(paramValue),
					Type:             types.ParameterTypeSecureString,
					Version:          1,
					LastModifiedDate: aws.Time(time.Now()),
				},
			}

			// Get parameter
			param, err := integration.GetParameter(context.Background(), paramName)
			Expect(err).NotTo(HaveOccurred())
			Expect(param).NotTo(BeNil())
			Expect(param.Name).To(Equal(paramName))
			Expect(param.Value).To(Equal(paramValue))
			Expect(param.Type).To(Equal("SecureString"))
			Expect(param.Version).To(Equal(int64(1)))
		})

		It("should return error for non-existent parameter", func() {
			_, err := integration.GetParameter(context.Background(), "/non/existent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parameter not found"))
		})

		It("should cache parameters", func() {
			// Setup mock parameter
			paramName := "/app/cache/test"
			paramValue := "cached-value"
			mockSSM.parameters[paramName] = &ssm.GetParameterOutput{
				Parameter: &types.Parameter{
					Name:             aws.String(paramName),
					Value:            aws.String(paramValue),
					Type:             types.ParameterTypeString,
					Version:          1,
					LastModifiedDate: aws.Time(time.Now()),
				},
			}

			// First call
			param1, err := integration.GetParameter(context.Background(), paramName)
			Expect(err).NotTo(HaveOccurred())
			Expect(param1.Value).To(Equal(paramValue))

			// Update mock to return different value
			mockSSM.parameters[paramName] = &ssm.GetParameterOutput{
				Parameter: &types.Parameter{
					Name:             aws.String(paramName),
					Value:            aws.String("new-value"),
					Type:             types.ParameterTypeString,
					Version:          2,
					LastModifiedDate: aws.Time(time.Now()),
				},
			}

			// Second call should return cached value
			param2, err := integration.GetParameter(context.Background(), paramName)
			Expect(err).NotTo(HaveOccurred())
			Expect(param2.Value).To(Equal(paramValue)) // Still the old cached value
		})
	})

	Describe("GetSecretNameForParameter", func() {
		It("should generate valid Kubernetes secret names", func() {
			testCases := []struct {
				parameterName string
				expectedName  string
			}{
				{"/app/database/password", "ssm-app-database-password"},
				{"app/database/password", "ssm-app-database-password"},
				{"/app/database/connection_string", "ssm-app-database-connection-string"},
				{"/APP/Database/Password", "ssm-app-database-password"},
				{"/app//database///password", "ssm-app-database-password"},
				{"/app@database#password$", "ssm-app-database-password"},
			}

			for _, tc := range testCases {
				secretName := integration.GetSecretNameForParameter(tc.parameterName)
				Expect(secretName).To(Equal(tc.expectedName))
			}
		})
	})

	Describe("CreateOrUpdateSecret", func() {
		It("should create a new Kubernetes secret", func() {
			param := &ssmIntegration.Parameter{
				Name:         "/app/test/secret",
				Value:        "test-value",
				Type:         "String",
				Version:      1,
				LastModified: time.Now(),
			}

			err := integration.CreateOrUpdateSecret(context.Background(), param, "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify secret was created
			secretName := integration.GetSecretNameForParameter(param.Name)
			secret, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secretName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(secret).NotTo(BeNil())
			Expect(string(secret.Data["value"])).To(Equal("test-value"))
			Expect(secret.Annotations[ssmIntegration.SecretAnnotations.ParameterName]).To(Equal(param.Name))
			Expect(secret.Annotations[ssmIntegration.SecretAnnotations.ParameterVersion]).To(Equal("1"))
			Expect(secret.Labels[ssmIntegration.SecretLabels.ManagedBy]).To(Equal("kecs"))
			Expect(secret.Labels[ssmIntegration.SecretLabels.Source]).To(Equal("ssm"))
		})

		It("should update an existing Kubernetes secret", func() {
			// Create initial secret
			secretName := "ssm-app-existing-secret"
			initialSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
					Annotations: map[string]string{
						ssmIntegration.SecretAnnotations.ParameterVersion: "1",
					},
				},
				Data: map[string][]byte{
					"value": []byte("old-value"),
				},
			}
			_, err := kubeClient.CoreV1().Secrets("default").Create(context.Background(), initialSecret, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Update with new parameter
			param := &ssmIntegration.Parameter{
				Name:         "/app/existing/secret",
				Value:        "new-value",
				Type:         "String",
				Version:      2,
				LastModified: time.Now(),
			}

			err = integration.CreateOrUpdateSecret(context.Background(), param, "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify secret was updated
			updatedSecret, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secretName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(updatedSecret.Data["value"])).To(Equal("new-value"))
			Expect(updatedSecret.Annotations[ssmIntegration.SecretAnnotations.ParameterVersion]).To(Equal("2"))
		})
	})

	Describe("SyncParameter", func() {
		It("should sync a parameter from SSM to Kubernetes", func() {
			// Setup mock parameter
			paramName := "/app/sync/test"
			paramValue := "sync-test-value"
			mockSSM.parameters[paramName] = &ssm.GetParameterOutput{
				Parameter: &types.Parameter{
					Name:             aws.String(paramName),
					Value:            aws.String(paramValue),
					Type:             types.ParameterTypeString,
					Version:          1,
					LastModifiedDate: aws.Time(time.Now()),
				},
			}

			// Sync parameter
			err := integration.SyncParameter(context.Background(), paramName, "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify secret was created
			secretName := integration.GetSecretNameForParameter(paramName)
			secret, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secretName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(secret.Data["value"])).To(Equal(paramValue))
		})

		It("should handle sync errors gracefully", func() {
			// Try to sync non-existent parameter
			err := integration.SyncParameter(context.Background(), "/non/existent", "default")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get parameter"))
		})
	})

	Describe("DeleteSecret", func() {
		It("should delete a synchronized secret", func() {
			// Create a secret first
			paramName := "/app/delete/test"
			secretName := integration.GetSecretNameForParameter(paramName)
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					"value": []byte("to-be-deleted"),
				},
			}
			_, err := kubeClient.CoreV1().Secrets("default").Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Delete the secret
			err = integration.DeleteSecret(context.Background(), paramName, "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify secret was deleted
			_, err = kubeClient.CoreV1().Secrets("default").Get(context.Background(), secretName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should not error when deleting non-existent secret", func() {
			err := integration.DeleteSecret(context.Background(), "/non/existent", "default")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SyncParameters", func() {
		It("should sync multiple parameters", func() {
			// Setup mock parameters
			params := []string{
				"/app/batch/param1",
				"/app/batch/param2",
				"/app/batch/param3",
			}
			for i, paramName := range params {
				mockSSM.parameters[paramName] = &ssm.GetParameterOutput{
					Parameter: &types.Parameter{
						Name:             aws.String(paramName),
						Value:            aws.String(fmt.Sprintf("value-%d", i+1)),
						Type:             types.ParameterTypeString,
						Version:          1,
						LastModifiedDate: aws.Time(time.Now()),
					},
				}
			}

			// Sync all parameters
			err := integration.SyncParameters(context.Background(), params, "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify all secrets were created
			for i, paramName := range params {
				secretName := integration.GetSecretNameForParameter(paramName)
				secret, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secretName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(string(secret.Data["value"])).To(Equal(fmt.Sprintf("value-%d", i+1)))
			}
		})

		It("should report errors for failed syncs", func() {
			params := []string{
				"/app/batch/exists",
				"/app/batch/notexists",
			}
			// Only setup one parameter
			mockSSM.parameters[params[0]] = &ssm.GetParameterOutput{
				Parameter: &types.Parameter{
					Name:             aws.String(params[0]),
					Value:            aws.String("exists-value"),
					Type:             types.ParameterTypeString,
					Version:          1,
					LastModifiedDate: aws.Time(time.Now()),
				},
			}

			// Sync parameters
			err := integration.SyncParameters(context.Background(), params, "default")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to sync 1 parameters"))

			// Verify successful sync still created secret
			secretName := integration.GetSecretNameForParameter(params[0])
			secret, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secretName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(secret.Data["value"])).To(Equal("exists-value"))
		})
	})
})

// mockSSMClient is a mock implementation of SSMClient for testing
type mockSSMClient struct {
	parameters map[string]*ssm.GetParameterOutput
}

func (m *mockSSMClient) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if params.Name == nil {
		return nil, fmt.Errorf("parameter name is required")
	}
	
	output, exists := m.parameters[*params.Name]
	if !exists {
		return nil, &types.ParameterNotFound{
			Message: aws.String(fmt.Sprintf("parameter not found: %s", *params.Name)),
		}
	}
	
	return output, nil
}

func (m *mockSSMClient) GetParameters(ctx context.Context, params *ssm.GetParametersInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersOutput, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSSMClient) PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSSMClient) DeleteParameter(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
	return nil, fmt.Errorf("not implemented")
}