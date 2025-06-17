package admission_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nandemo-ya/kecs/controlplane/internal/admission"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

var _ = Describe("Sidecar Injection Integration", func() {
	var (
		taskConverter *converters.TaskConverter
		sidecarInj    *admission.SidecarInjector
		mockLocalStackMgr *mockLocalStackManager
	)

	BeforeEach(func() {
		taskConverter = converters.NewTaskConverter("us-east-1", "123456789012")
		mockLocalStackMgr = &mockLocalStackManager{
			enabled:         true,
			endpoint:        "http://localstack:4566",
			enabledServices: []string{"s3", "dynamodb", "sqs", "sns", "ssm", "secretsmanager"},
		}
		sidecarInj = admission.NewSidecarInjector("kecs/aws-proxy:latest", mockLocalStackMgr)
	})

	Describe("Task to Pod conversion with sidecar injection", func() {
		It("should add sidecar injection annotation for tasks using AWS services", func() {
			// Create task definition with AWS service usage
			taskDef := &storage.TaskDefinition{
				ARN:      "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
				Family:   "test-task",
				Revision: 1,
				ContainerDefinitions: `[{
					"name": "app",
					"image": "myapp:latest",
					"environment": [
						{"name": "AWS_REGION", "value": "us-east-1"},
						{"name": "S3_BUCKET", "value": "my-bucket"}
					],
					"secrets": [{
						"name": "DB_PASSWORD",
						"valueFrom": "arn:aws:secretsmanager:us-east-1:123456789012:secret:db-password-AbCdEf"
					}],
					"logConfiguration": {
						"logDriver": "awslogs",
						"options": {
							"awslogs-group": "/ecs/test-task",
							"awslogs-region": "us-east-1"
						}
					}
				}]`,
			}

			runTaskReq := types.RunTaskRequest{
				Cluster:        &[]string{"test-cluster"}[0],
				TaskDefinition: &[]string{"test-task:1"}[0],
			}
			runTaskReqJSON, _ := json.Marshal(runTaskReq)

			cluster := &storage.Cluster{
				Name:   "test-cluster",
				Region: "us-east-1",
			}

			// Convert task to pod
			pod, err := taskConverter.ConvertTaskToPod(taskDef, runTaskReqJSON, cluster, "task-123")
			Expect(err).NotTo(HaveOccurred())

			// Check that sidecar injection annotation is added
			Expect(pod.Annotations["kecs.io/inject-aws-proxy"]).To(Equal("true"))
			
			// Check that proxy services are detected correctly
			Expect(pod.Annotations["kecs.io/proxy-services"]).To(ContainSubstring("s3"))
			Expect(pod.Annotations["kecs.io/proxy-services"]).To(ContainSubstring("secretsmanager"))
			Expect(pod.Annotations["kecs.io/proxy-services"]).To(ContainSubstring("cloudwatch"))
		})

		It("should inject sidecar when webhook processes the pod", func() {
			// Create a pod with injection annotation
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Annotations: map[string]string{
						"kecs.io/inject-aws-proxy": "true",
						"kecs.io/proxy-services":   "s3,dynamodb",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "myapp:latest",
							Env: []corev1.EnvVar{
								{Name: "AWS_REGION", Value: "us-east-1"},
							},
						},
					},
				},
			}

			podJSON, _ := json.Marshal(pod)
			req := &admissionv1.AdmissionRequest{
				UID:       "test-uid",
				Name:      "test-pod",
				Namespace: "default",
				Object: runtime.RawExtension{
					Raw: podJSON,
				},
			}

			// Process through webhook
			resp := sidecarInj.Handle(context.Background(), req)
			Expect(resp.Allowed).To(BeTrue())
			Expect(resp.Patch).NotTo(BeNil())

			// Apply patches to verify sidecar was added
			var patches []map[string]interface{}
			err := json.Unmarshal(resp.Patch, &patches)
			Expect(err).NotTo(HaveOccurred())

			// Find sidecar container patch
			var sidecarFound bool
			for _, patch := range patches {
				if patch["path"] == "/spec/containers/-" && patch["op"] == "add" {
					container := patch["value"].(map[string]interface{})
					if container["name"] == "aws-sdk-proxy" {
						sidecarFound = true
						
						// Verify sidecar configuration
						Expect(container["image"]).To(Equal("kecs/aws-proxy:latest"))
						
						// Check environment variables
						envVars := container["env"].([]interface{})
						var hasEndpoint, hasServices bool
						for _, env := range envVars {
							envMap := env.(map[string]interface{})
							if envMap["name"] == "LOCALSTACK_ENDPOINT" {
								hasEndpoint = true
								Expect(envMap["value"]).To(Equal("http://localstack:4566"))
							}
							if envMap["name"] == "PROXY_SERVICES" {
								hasServices = true
								Expect(envMap["value"]).To(Equal("s3,dynamodb"))
							}
						}
						Expect(hasEndpoint).To(BeTrue())
						Expect(hasServices).To(BeTrue())
					}
				}
			}
			Expect(sidecarFound).To(BeTrue())

			// Check that main container has proxy environment variables
			var proxyEnvFound bool
			for _, patch := range patches {
				if patch["op"] == "add" && patch["path"].(string) == "/spec/containers/0/env/-" {
					env := patch["value"].(map[string]interface{})
					if env["name"] == "AWS_ENDPOINT_URL" {
						proxyEnvFound = true
						Expect(env["value"]).To(Equal("http://localhost:8080"))
					}
				}
			}
			Expect(proxyEnvFound).To(BeTrue())
		})
	})

	Describe("End-to-end webhook integration", func() {
		It("should start webhook server and handle requests", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Create fake Kubernetes client
			fakeClient := fake.NewSimpleClientset()

			// Create webhook integration
			webhookInt := admission.NewWebhookIntegration(
				fakeClient,
				"kecs-system",
				8443,
				"kecs/aws-proxy:latest",
				mockLocalStackMgr,
			)

			// Note: In a real test, we would:
			// 1. Start the webhook server
			// 2. Create a test pod through the Kubernetes API
			// 3. Verify the pod was mutated with sidecar
			// However, this requires a full Kubernetes test environment
			
			// For now, we just verify the integration can be created
			Expect(webhookInt).NotTo(BeNil())
		})
	})
})

// Mock LocalStackManager for testing
type mockLocalStackManager struct {
	enabled         bool
	endpoint        string
	enabledServices []string
}

func (m *mockLocalStackManager) IsEnabled() bool {
	return m.enabled
}

func (m *mockLocalStackManager) GetEndpoint() string {
	return m.endpoint
}

func (m *mockLocalStackManager) GetEnabledServices() []string {
	return m.enabledServices
}

func TestAdmissionIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Admission Integration Suite")
}