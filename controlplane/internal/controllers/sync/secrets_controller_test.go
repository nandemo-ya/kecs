package sync_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nandemo-ya/kecs/controlplane/internal/controllers/sync"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/secretsmanager"
)

func TestSecretsController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Secrets Controller Suite")
}

var _ = Describe("SecretsController", func() {
	var (
		ctx            context.Context
		kubeClient     *fake.Clientset
	)

	BeforeEach(func() {
		ctx = context.Background()
		kubeClient = fake.NewSimpleClientset()
		
		// Create kecs-system namespace
		_, err := kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kecs-system",
			},
		}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Create user namespace
		_, err = kubeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default-us-east-1",
			},
		}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Centralized Secret Management", func() {
		Context("when syncing Secrets Manager secrets", func() {
			It("should create secret in kecs-system and replicate to user namespace", func() {
				// Create a pod with secret annotation
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default-us-east-1",
						Labels: map[string]string{
							"kecs.dev/managed-by": "kecs",
						},
						Annotations: map[string]string{
							"kecs.dev/secret-count": "1",
							"kecs.dev/secret-0-arn": "app:DB_PASSWORD:arn:aws:secretsmanager:us-east-1:123456789012:secret:db-password",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "nginx",
							},
						},
					},
				}

				_, err := kubeClient.CoreV1().Pods("default-us-east-1").Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Simulate the controller handling the pod
				// In real implementation, this would be triggered by the informer
				
				// Check that secret was created in kecs-system
				Eventually(func() bool {
					_, err := kubeClient.CoreV1().Secrets("kecs-system").Get(ctx, "sm-db-password", metav1.GetOptions{})
					return err == nil
				}).Should(BeTrue())

				// Check that secret was replicated to user namespace
				Eventually(func() bool {
					secret, err := kubeClient.CoreV1().Secrets("default-us-east-1").Get(ctx, "sm-db-password", metav1.GetOptions{})
					if err != nil {
						return false
					}
					// Check for replication labels
					return secret.Labels["kecs.io/replicated-from"] == "kecs-system"
				}).Should(BeTrue())
			})
		})

		Context("when syncing SSM parameters", func() {
			It("should create ConfigMap for non-sensitive parameters in kecs-system", func() {
				// Create a pod with SSM parameter annotation for non-sensitive data
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default-us-east-1",
						Labels: map[string]string{
							"kecs.dev/managed-by": "kecs",
						},
						Annotations: map[string]string{
							"kecs.dev/secret-count": "1",
							"kecs.dev/secret-0-arn": "app:DATABASE_URL:arn:aws:ssm:us-east-1:123456789012:parameter/app/database/url",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "nginx",
							},
						},
					},
				}

				_, err := kubeClient.CoreV1().Pods("default-us-east-1").Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Check that ConfigMap was created in kecs-system
				Eventually(func() bool {
					_, err := kubeClient.CoreV1().ConfigMaps("kecs-system").Get(ctx, "ssm-app-database-url", metav1.GetOptions{})
					return err == nil
				}).Should(BeTrue())

				// Check that ConfigMap was replicated to user namespace
				Eventually(func() bool {
					cm, err := kubeClient.CoreV1().ConfigMaps("default-us-east-1").Get(ctx, "ssm-app-database-url", metav1.GetOptions{})
					if err != nil {
						return false
					}
					return cm.Labels["kecs.io/replicated-from"] == "kecs-system"
				}).Should(BeTrue())
			})

			It("should create Secret for sensitive SSM parameters in kecs-system", func() {
				// Create a pod with SSM parameter annotation for sensitive data
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default-us-east-1",
						Labels: map[string]string{
							"kecs.dev/managed-by": "kecs",
						},
						Annotations: map[string]string{
							"kecs.dev/secret-count": "1",
							"kecs.dev/secret-0-arn": "app:DB_PASSWORD:arn:aws:ssm:us-east-1:123456789012:parameter/app/db_password",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "app",
								Image: "nginx",
							},
						},
					},
				}

				_, err := kubeClient.CoreV1().Pods("default-us-east-1").Create(ctx, pod, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Check that Secret was created in kecs-system (not ConfigMap)
				Eventually(func() bool {
					_, err := kubeClient.CoreV1().Secrets("kecs-system").Get(ctx, "ssm-app-db-password", metav1.GetOptions{})
					return err == nil
				}).Should(BeTrue())
			})
		})
	})

	Describe("Secret Replicator", func() {
		var replicator *sync.SecretsReplicator

		BeforeEach(func() {
			replicator = sync.NewSecretsReplicator(kubeClient)
		})

		Context("when replicating secrets", func() {
			It("should copy secret from kecs-system to target namespace", func() {
				// Create source secret in kecs-system
				sourceSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "kecs-system",
						Labels: map[string]string{
							"kecs.io/managed-by": "kecs",
							"kecs.io/source":     "secretsmanager",
						},
					},
					Data: map[string][]byte{
						"value": []byte("secret-value"),
					},
				}
				_, err := kubeClient.CoreV1().Secrets("kecs-system").Create(ctx, sourceSecret, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Replicate to user namespace
				err = replicator.ReplicateSecretToNamespace(ctx, "test-secret", "default-us-east-1")
				Expect(err).NotTo(HaveOccurred())

				// Verify replicated secret
				replicatedSecret, err := kubeClient.CoreV1().Secrets("default-us-east-1").Get(ctx, "test-secret", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(replicatedSecret.Data).To(Equal(sourceSecret.Data))
				Expect(replicatedSecret.Labels["kecs.io/replicated-from"]).To(Equal("kecs-system"))
			})
		})

		Context("when cleaning up orphaned replicas", func() {
			It("should delete replicas that no longer exist in kecs-system", func() {
				// Create an orphaned replica in user namespace
				orphanedSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "orphaned-secret",
						Namespace: "default-us-east-1",
						Labels: map[string]string{
							"kecs.io/replicated-from": "kecs-system",
						},
					},
					Data: map[string][]byte{
						"value": []byte("orphaned"),
					},
				}
				_, err := kubeClient.CoreV1().Secrets("default-us-east-1").Create(ctx, orphanedSecret, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Run cleanup
				err = replicator.CleanupOrphanedReplicas(ctx, "default-us-east-1")
				Expect(err).NotTo(HaveOccurred())

				// Verify orphaned secret was deleted
				_, err = kubeClient.CoreV1().Secrets("default-us-east-1").Get(ctx, "orphaned-secret", metav1.GetOptions{})
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

// Mock implementations for testing
type mockSecretsManagerIntegration struct{}

func (m *mockSecretsManagerIntegration) GetSecret(ctx context.Context, secretName string) (*secretsmanager.Secret, error) {
	return &secretsmanager.Secret{
		Name:  secretName,
		Value: "mock-secret-value",
	}, nil
}

func (m *mockSecretsManagerIntegration) CreateOrUpdateSecret(ctx context.Context, secret *secretsmanager.Secret, jsonKey string, namespace string) error {
	return nil
}

func (m *mockSecretsManagerIntegration) DeleteSecret(ctx context.Context, secretName string, namespace string) error {
	return nil
}

func (m *mockSecretsManagerIntegration) GetSecretNameForSecret(secretName string) string {
	return "sm-" + secretName
}

func (m *mockSecretsManagerIntegration) WatchForChanges(ctx context.Context) <-chan interface{} {
	ch := make(chan interface{})
	close(ch)
	return ch
}

type mockSSMIntegration struct{}

func (m *mockSSMIntegration) SyncParameter(ctx context.Context, parameterName string, namespace string) error {
	return nil
}

func (m *mockSSMIntegration) GetSecretNameForParameter(parameterName string) string {
	return "ssm-" + parameterName
}

func (m *mockSSMIntegration) CreateOrUpdateConfigMap(ctx context.Context, name string, data map[string]string, namespace string) error {
	return nil
}

func (m *mockSSMIntegration) CreateOrUpdateSecret(ctx context.Context, name string, data map[string][]byte, namespace string) error {
	return nil
}

func (m *mockSSMIntegration) DeleteConfigMap(ctx context.Context, name string, namespace string) error {
	return nil
}

func (m *mockSSMIntegration) DeleteSecret(ctx context.Context, name string, namespace string) error {
	return nil
}

func (m *mockSSMIntegration) GetParameter(ctx context.Context, parameterName string) (string, error) {
	return "mock-parameter-value", nil
}

func (m *mockSSMIntegration) IsParameterSecure(parameterName string) bool {
	return false
}