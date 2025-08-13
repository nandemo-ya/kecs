package converters_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/cloudwatch"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

var _ = Describe("TaskConverter LogConfiguration", func() {
	var (
		converter *converters.TaskConverter
		pod       *corev1.Pod
		taskDef   *storage.TaskDefinition
	)

	BeforeEach(func() {
		cloudWatchIntegration := &cloudwatch.MockIntegration{}
		converter = converters.NewTaskConverterWithCloudWatch("us-east-1", "123456789012", cloudWatchIntegration)

		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-pod",
				Annotations: make(map[string]string),
			},
		}

		taskDef = &storage.TaskDefinition{
			ARN:    "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
			Family: "test-task",
		}
	})

	Describe("applyCloudWatchLogsConfiguration", func() {
		Context("when container has awslogs driver", func() {
			It("should add correct annotations for basic awslogs configuration", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name: ptr.To("nginx"),
						LogConfiguration: &types.LogConfiguration{
							LogDriver: ptr.To("awslogs"),
							Options: map[string]string{
								"awslogs-group":         "/ecs/nginx-app",
								"awslogs-region":        "us-east-1",
								"awslogs-stream-prefix": "nginx",
							},
						},
					},
				}

				converter.ApplyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)

				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/cloudwatch-logs-enabled", "true"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-nginx-logs-driver", "awslogs"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-nginx-logs-group", "/ecs/nginx-app"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-nginx-logs-stream-prefix", "nginx"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-nginx-logs-region", "us-east-1"))
			})

			It("should use container name as stream prefix when not specified", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name: ptr.To("app"),
						LogConfiguration: &types.LogConfiguration{
							LogDriver: ptr.To("awslogs"),
							Options: map[string]string{
								"awslogs-group":  "/ecs/app",
								"awslogs-region": "us-west-2",
							},
						},
					},
				}

				converter.ApplyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)

				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-app-logs-stream-prefix", "app"))
			})

			It("should use default region when not specified", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name: ptr.To("app"),
						LogConfiguration: &types.LogConfiguration{
							LogDriver: ptr.To("awslogs"),
							Options: map[string]string{
								"awslogs-group": "/ecs/app",
							},
						},
					},
				}

				converter.ApplyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)

				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-app-logs-region", "us-east-1"))
			})

			It("should handle multiple containers with different log configurations", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name: ptr.To("nginx"),
						LogConfiguration: &types.LogConfiguration{
							LogDriver: ptr.To("awslogs"),
							Options: map[string]string{
								"awslogs-group":         "/ecs/nginx",
								"awslogs-stream-prefix": "web",
							},
						},
					},
					{
						Name: ptr.To("app"),
						LogConfiguration: &types.LogConfiguration{
							LogDriver: ptr.To("awslogs"),
							Options: map[string]string{
								"awslogs-group":         "/ecs/app",
								"awslogs-stream-prefix": "backend",
								"awslogs-region":        "eu-west-1",
							},
						},
					},
				}

				converter.ApplyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)

				// Check nginx container annotations
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-nginx-logs-driver", "awslogs"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-nginx-logs-group", "/ecs/nginx"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-nginx-logs-stream-prefix", "web"))

				// Check app container annotations
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-app-logs-driver", "awslogs"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-app-logs-group", "/ecs/app"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-app-logs-stream-prefix", "backend"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-app-logs-region", "eu-west-1"))
			})

			It("should add additional awslogs options as annotations", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name: ptr.To("app"),
						LogConfiguration: &types.LogConfiguration{
							LogDriver: ptr.To("awslogs"),
							Options: map[string]string{
								"awslogs-group":             "/ecs/app",
								"awslogs-create-group":      "true",
								"awslogs-datetime-format":   "%Y-%m-%d %H:%M:%S",
								"awslogs-multiline-pattern": "^\\[",
							},
						},
					},
				}

				converter.ApplyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)

				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-app-logs-create-group", "true"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-app-logs-datetime-format", "%Y-%m-%d %H:%M:%S"))
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-app-logs-multiline-pattern", "^\\["))
			})
		})

		Context("when container does not have awslogs driver", func() {
			It("should not add annotations for other log drivers", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name: ptr.To("app"),
						LogConfiguration: &types.LogConfiguration{
							LogDriver: ptr.To("json-file"),
							Options: map[string]string{
								"max-size": "10m",
								"max-file": "3",
							},
						},
					},
				}

				converter.ApplyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)

				Expect(pod.Annotations).NotTo(HaveKey("kecs.dev/cloudwatch-logs-enabled"))
				Expect(pod.Annotations).NotTo(HaveKey("kecs.dev/container-app-logs-driver"))
			})

			It("should handle mixed log drivers correctly", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name: ptr.To("nginx"),
						LogConfiguration: &types.LogConfiguration{
							LogDriver: ptr.To("awslogs"),
							Options: map[string]string{
								"awslogs-group": "/ecs/nginx",
							},
						},
					},
					{
						Name: ptr.To("sidecar"),
						LogConfiguration: &types.LogConfiguration{
							LogDriver: ptr.To("json-file"),
						},
					},
				}

				converter.ApplyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)

				// Should have CloudWatch enabled because at least one container uses awslogs
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/cloudwatch-logs-enabled", "true"))

				// Should have annotations for nginx
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/container-nginx-logs-driver", "awslogs"))

				// Should NOT have annotations for sidecar
				Expect(pod.Annotations).NotTo(HaveKey("kecs.dev/container-sidecar-logs-driver"))
			})
		})

		Context("when log configuration is nil", func() {
			It("should handle nil LogConfiguration gracefully", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name:             ptr.To("app"),
						LogConfiguration: nil,
					},
				}

				converter.ApplyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)

				Expect(pod.Annotations).NotTo(HaveKey("kecs.dev/cloudwatch-logs-enabled"))
			})

			It("should handle nil container name gracefully", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name: nil,
						LogConfiguration: &types.LogConfiguration{
							LogDriver: ptr.To("awslogs"),
							Options: map[string]string{
								"awslogs-group": "/ecs/app",
							},
						},
					},
				}

				// Should not panic
				converter.ApplyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)
			})
		})
	})
})
