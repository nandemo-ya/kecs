package ssm_test

import (
	"context"
	"github.com/nandemo-ya/kecs/controlplane/internal/common"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	ssmIntegration "github.com/nandemo-ya/kecs/controlplane/internal/integrations/ssm"
	ssmapi "github.com/nandemo-ya/kecs/controlplane/internal/ssm/generated"
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
			parameters: make(map[string]*ssmapi.GetParameterResult),
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
			now := time.Now()
			version := int64(1)
			paramType := ssmapi.ParameterType("SecureString")
			mockSSM.parameters[paramName] = &ssmapi.GetParameterResult{
				Parameter: &ssmapi.Parameter{
					Name:             &paramName,
					Value:            &paramValue,
					Type:             &paramType,
					Version:          &version,
					LastModifiedDate: &common.UnixTime{Time: now},
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
			Expect(err.Error()).To(ContainSubstring("ParameterNotFound"))
		})

		It("should cache parameters", func() {
			// Setup mock parameter
			paramName := "/app/cache/test"
			paramValue := "cached-value"
			now := time.Now()
			version := int64(1)
			paramType := ssmapi.ParameterType("String")
			mockSSM.parameters[paramName] = &ssmapi.GetParameterResult{
				Parameter: &ssmapi.Parameter{
					Name:             &paramName,
					Value:            &paramValue,
					Type:             &paramType,
					Version:          &version,
					LastModifiedDate: &common.UnixTime{Time: now},
				},
			}

			// First call
			param1, err := integration.GetParameter(context.Background(), paramName)
			Expect(err).NotTo(HaveOccurred())
			Expect(param1.Value).To(Equal(paramValue))

			// Update mock to return different value
			now2 := time.Now()
			version2 := int64(2)
			newValue := "new-value"
			paramType2 := ssmapi.ParameterType("String")
			mockSSM.parameters[paramName] = &ssmapi.GetParameterResult{
				Parameter: &ssmapi.Parameter{
					Name:             &paramName,
					Value:            &newValue,
					Type:             &paramType2,
					Version:          &version2,
					LastModifiedDate: &common.UnixTime{Time: now2},
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
			now := time.Now()
			version := int64(1)
			paramType := ssmapi.ParameterType("String")
			mockSSM.parameters[paramName] = &ssmapi.GetParameterResult{
				Parameter: &ssmapi.Parameter{
					Name:             &paramName,
					Value:            &paramValue,
					Type:             &paramType,
					Version:          &version,
					LastModifiedDate: &common.UnixTime{Time: now},
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
				value := fmt.Sprintf("value-%d", i+1)
				now := time.Now()
				version := int64(1)
				paramType := ssmapi.ParameterType("String")
				paramNameCopy := paramName
				mockSSM.parameters[paramName] = &ssmapi.GetParameterResult{
					Parameter: &ssmapi.Parameter{
						Name:             &paramNameCopy,
						Value:            &value,
						Type:             &paramType,
						Version:          &version,
						LastModifiedDate: &common.UnixTime{Time: now},
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
			paramName0 := params[0]
			value0 := "exists-value"
			now0 := time.Now()
			version0 := int64(1)
			paramType0 := ssmapi.ParameterType("String")
			mockSSM.parameters[params[0]] = &ssmapi.GetParameterResult{
				Parameter: &ssmapi.Parameter{
					Name:             &paramName0,
					Value:            &value0,
					Type:             &paramType0,
					Version:          &version0,
					LastModifiedDate: &common.UnixTime{Time: now0},
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
	parameters map[string]*ssmapi.GetParameterResult
}

func (m *mockSSMClient) GetParameter(ctx context.Context, params *ssmapi.GetParameterRequest) (*ssmapi.GetParameterResult, error) {
	if params.Name == "" {
		return nil, fmt.Errorf("parameter name is required")
	}

	output, exists := m.parameters[params.Name]
	if !exists {
		msg := fmt.Sprintf("parameter not found: %s", params.Name)
		return nil, &ssmapi.ParameterNotFound{
			Message: &msg,
		}
	}

	return output, nil
}

func (m *mockSSMClient) GetParameters(ctx context.Context, params *ssmapi.GetParametersRequest) (*ssmapi.GetParametersResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSSMClient) PutParameter(ctx context.Context, params *ssmapi.PutParameterRequest) (*ssmapi.PutParameterResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSSMClient) DeleteParameter(ctx context.Context, params *ssmapi.DeleteParameterRequest) (*ssmapi.DeleteParameterResult, error) {
	return nil, fmt.Errorf("not implemented")
}
