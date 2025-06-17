package secretsmanager_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	sm "github.com/nandemo-ya/kecs/controlplane/internal/integrations/secretsmanager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Secrets Manager Integration", func() {
	var (
		integration sm.Integration
		kubeClient  *fake.Clientset
		mockSM      *mockSecretsManagerClient
		config      *sm.Config
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()
		mockSM = &mockSecretsManagerClient{
			secrets: make(map[string]*secretsmanager.GetSecretValueOutput),
		}
		config = &sm.Config{
			LocalStackEndpoint: "http://localhost:4566",
			SecretPrefix:       "sm-",
			KubeNamespace:      "default",
			SyncRetries:        3,
			CacheTTL:           5 * time.Minute,
		}
		integration = sm.NewIntegrationWithClient(kubeClient, mockSM, config)
	})

	Describe("GetSecret", func() {
		It("should retrieve a string secret from Secrets Manager", func() {
			// Setup mock secret
			secretName := "my-app/database/password"
			secretValue := "super-secret-password"
			mockSM.secrets[secretName] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String(secretName),
				SecretString:  aws.String(secretValue),
				VersionId:     aws.String("version-123"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}

			// Get secret
			secret, err := integration.GetSecret(context.Background(), secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret).NotTo(BeNil())
			Expect(secret.Name).To(Equal(secretName))
			Expect(secret.Value).To(Equal(secretValue))
			Expect(secret.Type).To(Equal("String"))
			Expect(secret.VersionId).To(Equal("version-123"))
			Expect(secret.VersionStage).To(Equal([]string{"AWSCURRENT"}))
		})

		It("should retrieve a binary secret from Secrets Manager", func() {
			// Setup mock binary secret
			secretName := "my-app/certificate"
			binaryData := []byte("binary-certificate-data")
			mockSM.secrets[secretName] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String(secretName),
				SecretBinary:  binaryData,
				VersionId:     aws.String("version-456"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}

			// Get secret
			secret, err := integration.GetSecret(context.Background(), secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret).NotTo(BeNil())
			Expect(secret.Name).To(Equal(secretName))
			Expect(secret.Value).To(Equal(string(binaryData)))
			Expect(secret.Type).To(Equal("Binary"))
		})

		It("should return error for non-existent secret", func() {
			_, err := integration.GetSecret(context.Background(), "non-existent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("secret not found"))
		})

		It("should cache secrets", func() {
			// Setup mock secret
			secretName := "my-app/cache/test"
			secretValue := "cached-value"
			mockSM.secrets[secretName] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String(secretName),
				SecretString:  aws.String(secretValue),
				VersionId:     aws.String("v1"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}

			// First call
			secret1, err := integration.GetSecret(context.Background(), secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret1.Value).To(Equal(secretValue))
			Expect(secret1.VersionId).To(Equal("v1"))

			// Update mock to return different value
			mockSM.secrets[secretName] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String(secretName),
				SecretString:  aws.String("new-value"),
				VersionId:     aws.String("v2"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}

			// Second call should return cached value
			secret2, err := integration.GetSecret(context.Background(), secretName)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret2.Value).To(Equal(secretValue)) // Still the old cached value
			Expect(secret2.VersionId).To(Equal("v1"))    // Still the old version
		})
	})

	Describe("GetSecretWithKey", func() {
		It("should extract JSON key from secret", func() {
			// Setup mock JSON secret
			secretName := "my-app/config"
			jsonValue := `{"username": "admin", "password": "secret123", "host": "localhost"}`
			mockSM.secrets[secretName] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String(secretName),
				SecretString:  aws.String(jsonValue),
				VersionId:     aws.String("v1"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}

			// Get specific key
			value, err := integration.GetSecretWithKey(context.Background(), secretName, "password")
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal("secret123"))
		})

		It("should return full value when no key specified", func() {
			// Setup mock secret
			secretName := "my-app/simple"
			secretValue := "simple-value"
			mockSM.secrets[secretName] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String(secretName),
				SecretString:  aws.String(secretValue),
				VersionId:     aws.String("v1"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}

			// Get without key
			value, err := integration.GetSecretWithKey(context.Background(), secretName, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(secretValue))
		})

		It("should return error for non-existent JSON key", func() {
			// Setup mock JSON secret
			secretName := "my-app/config"
			jsonValue := `{"username": "admin"}`
			mockSM.secrets[secretName] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String(secretName),
				SecretString:  aws.String(jsonValue),
				VersionId:     aws.String("v1"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}

			// Try to get non-existent key
			_, err := integration.GetSecretWithKey(context.Background(), secretName, "password")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("key password not found"))
		})
	})

	Describe("GetSecretNameForSecret", func() {
		It("should generate valid Kubernetes secret names", func() {
			testCases := []struct {
				secretName   string
				expectedName string
			}{
				{"my-app/database/password", "sm-my-app-database-password"},
				{"my-app/database/password-AbCdEf", "sm-my-app-database-password"},
				{"MY-APP/Database/Password", "sm-my-app-database-password"},
				{"app/db//password", "sm-app-db-password"},
				{"app@db#password$", "sm-app-db-password"},
				{"/app/database/password/", "sm-app-database-password"},
			}

			for _, tc := range testCases {
				secretName := integration.GetSecretNameForSecret(tc.secretName)
				Expect(secretName).To(Equal(tc.expectedName))
			}
		})
	})

	Describe("CreateOrUpdateSecret", func() {
		It("should create a new Kubernetes secret", func() {
			secret := &sm.Secret{
				Name:         "my-app/test/secret",
				Value:        "test-value",
				Type:         "String",
				VersionId:    "v1",
				VersionStage: []string{"AWSCURRENT"},
				CreatedDate:  time.Now(),
			}

			err := integration.CreateOrUpdateSecret(context.Background(), secret, "", "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify secret was created
			secretName := integration.GetSecretNameForSecret(secret.Name)
			k8sSecret, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secretName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sSecret).NotTo(BeNil())
			Expect(string(k8sSecret.Data["value"])).To(Equal("test-value"))
			Expect(k8sSecret.Annotations[sm.SecretAnnotations.SecretName]).To(Equal(secret.Name))
			Expect(k8sSecret.Annotations[sm.SecretAnnotations.VersionId]).To(Equal("v1"))
			Expect(k8sSecret.Labels[sm.SecretLabels.ManagedBy]).To(Equal("kecs"))
			Expect(k8sSecret.Labels[sm.SecretLabels.Source]).To(Equal("secretsmanager"))
		})

		It("should create a secret with specific JSON key", func() {
			jsonSecret := &sm.Secret{
				Name:         "my-app/config",
				Value:        `{"username": "admin", "password": "secret123"}`,
				Type:         "String",
				VersionId:    "v1",
				VersionStage: []string{"AWSCURRENT"},
				CreatedDate:  time.Now(),
			}

			err := integration.CreateOrUpdateSecret(context.Background(), jsonSecret, "password", "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify secret was created with JSON key
			secretName := integration.GetSecretNameForSecret(jsonSecret.Name)
			k8sSecret, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secretName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(k8sSecret.Data["password"])).To(Equal("secret123"))
			Expect(k8sSecret.Annotations[sm.SecretAnnotations.JSONKey]).To(Equal("password"))
		})

		It("should update an existing Kubernetes secret", func() {
			// Create initial secret
			secretName := "sm-my-app-existing-secret"
			initialSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
					Annotations: map[string]string{
						sm.SecretAnnotations.VersionId: "v1",
					},
				},
				Data: map[string][]byte{
					"value": []byte("old-value"),
				},
			}
			_, err := kubeClient.CoreV1().Secrets("default").Create(context.Background(), initialSecret, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Update with new secret
			secret := &sm.Secret{
				Name:         "my-app/existing/secret",
				Value:        "new-value",
				Type:         "String",
				VersionId:    "v2",
				VersionStage: []string{"AWSCURRENT"},
				CreatedDate:  time.Now(),
			}

			err = integration.CreateOrUpdateSecret(context.Background(), secret, "", "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify secret was updated
			updatedSecret, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secretName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(updatedSecret.Data["value"])).To(Equal("new-value"))
			Expect(updatedSecret.Annotations[sm.SecretAnnotations.VersionId]).To(Equal("v2"))
		})
	})

	Describe("SyncSecret", func() {
		It("should sync a secret from Secrets Manager to Kubernetes", func() {
			// Setup mock secret
			secretName := "my-app/sync/test"
			secretValue := "sync-test-value"
			mockSM.secrets[secretName] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String(secretName),
				SecretString:  aws.String(secretValue),
				VersionId:     aws.String("v1"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}

			// Sync secret
			err := integration.SyncSecret(context.Background(), secretName, "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify secret was created
			k8sSecretName := integration.GetSecretNameForSecret(secretName)
			secret, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), k8sSecretName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(secret.Data["value"])).To(Equal(secretValue))
		})

		It("should handle sync errors gracefully", func() {
			// Try to sync non-existent secret
			err := integration.SyncSecret(context.Background(), "non/existent", "default")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get secret"))
		})
	})

	Describe("DeleteSecret", func() {
		It("should delete a synchronized secret", func() {
			// Create a secret first
			secretName := "my-app/delete/test"
			k8sSecretName := integration.GetSecretNameForSecret(secretName)
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k8sSecretName,
					Namespace: "default",
				},
				Data: map[string][]byte{
					"value": []byte("to-be-deleted"),
				},
			}
			_, err := kubeClient.CoreV1().Secrets("default").Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Delete the secret
			err = integration.DeleteSecret(context.Background(), secretName, "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify secret was deleted
			_, err = kubeClient.CoreV1().Secrets("default").Get(context.Background(), k8sSecretName, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should not error when deleting non-existent secret", func() {
			err := integration.DeleteSecret(context.Background(), "non/existent", "default")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SyncSecrets", func() {
		It("should sync multiple secrets", func() {
			// Setup mock secrets
			secrets := []sm.SecretReference{
				{SecretName: "my-app/batch/secret1"},
				{SecretName: "my-app/batch/secret2", JSONKey: "password"},
				{SecretName: "my-app/batch/secret3"},
			}
			
			// Add mock data
			mockSM.secrets["my-app/batch/secret1"] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String("my-app/batch/secret1"),
				SecretString:  aws.String("value-1"),
				VersionId:     aws.String("v1"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}
			mockSM.secrets["my-app/batch/secret2"] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String("my-app/batch/secret2"),
				SecretString:  aws.String(`{"username": "admin", "password": "secret123"}`),
				VersionId:     aws.String("v1"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}
			mockSM.secrets["my-app/batch/secret3"] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String("my-app/batch/secret3"),
				SecretString:  aws.String("value-3"),
				VersionId:     aws.String("v1"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}

			// Sync all secrets
			err := integration.SyncSecrets(context.Background(), secrets, "default")
			Expect(err).NotTo(HaveOccurred())

			// Verify all secrets were created
			secret1Name := integration.GetSecretNameForSecret("my-app/batch/secret1")
			secret1, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secret1Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(secret1.Data["value"])).To(Equal("value-1"))

			secret2Name := integration.GetSecretNameForSecret("my-app/batch/secret2")
			secret2, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secret2Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(secret2.Data["password"])).To(Equal("secret123"))

			secret3Name := integration.GetSecretNameForSecret("my-app/batch/secret3")
			secret3, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secret3Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(secret3.Data["value"])).To(Equal("value-3"))
		})

		It("should report errors for failed syncs", func() {
			secrets := []sm.SecretReference{
				{SecretName: "my-app/batch/exists"},
				{SecretName: "my-app/batch/notexists"},
			}
			
			// Only setup one secret
			mockSM.secrets["my-app/batch/exists"] = &secretsmanager.GetSecretValueOutput{
				Name:          aws.String("my-app/batch/exists"),
				SecretString:  aws.String("exists-value"),
				VersionId:     aws.String("v1"),
				VersionStages: []string{"AWSCURRENT"},
				CreatedDate:   aws.Time(time.Now()),
			}

			// Sync secrets
			err := integration.SyncSecrets(context.Background(), secrets, "default")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to sync 1 secrets"))

			// Verify successful sync still created secret
			secretName := integration.GetSecretNameForSecret("my-app/batch/exists")
			secret, err := kubeClient.CoreV1().Secrets("default").Get(context.Background(), secretName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(string(secret.Data["value"])).To(Equal("exists-value"))
		})
	})
})

// mockSecretsManagerClient is a mock implementation of SecretsManagerClient for testing
type mockSecretsManagerClient struct {
	secrets map[string]*secretsmanager.GetSecretValueOutput
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	if params.SecretId == nil {
		return nil, fmt.Errorf("secret ID is required")
	}
	
	output, exists := m.secrets[*params.SecretId]
	if !exists {
		return nil, fmt.Errorf("secret not found: %s", *params.SecretId)
	}
	
	return output, nil
}

func (m *mockSecretsManagerClient) CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSecretsManagerClient) UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSecretsManagerClient) DeleteSecret(ctx context.Context, params *secretsmanager.DeleteSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DeleteSecretOutput, error) {
	return nil, fmt.Errorf("not implemented")
}