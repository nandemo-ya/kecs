package converters_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("ServiceConverter Health Check", func() {
	var (
		converter *converters.ServiceConverter
		service   *storage.Service
		cluster   *storage.Cluster
	)

	BeforeEach(func() {
		converter = converters.NewServiceConverter("us-east-1", "123456789012")
		service = &storage.Service{
			ServiceName:  "test-service",
			DesiredCount: 1,
			LaunchType:   "FARGATE",
			ARN:          "arn:aws:ecs:us-east-1:123456789012:service/test-service",
		}
		cluster = &storage.Cluster{
			Name:   "test-cluster",
			Region: "us-east-1",
		}
	})

	Context("when task definition has health check", func() {
		It("should convert CMD-SHELL health check to exec probe", func() {
			taskDef := &storage.TaskDefinition{
				Family:   "test-family",
				Revision: 1,
				ContainerDefinitions: mustMarshal([]map[string]interface{}{
					{
						"name":  "app",
						"image": "nginx:latest",
						"healthCheck": map[string]interface{}{
							"command":     []interface{}{"CMD-SHELL", "wget -q -O - http://localhost:8080/health || exit 1"},
							"interval":    float64(30),
							"timeout":     float64(5),
							"retries":     float64(3),
							"startPeriod": float64(30),
						},
					},
				}),
			}

			deployment, _, err := converter.ConvertServiceToDeployment(service, taskDef, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployment).NotTo(BeNil())
			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))

			container := deployment.Spec.Template.Spec.Containers[0]

			// Check liveness probe
			Expect(container.LivenessProbe).NotTo(BeNil())
			Expect(container.LivenessProbe.Exec).NotTo(BeNil())
			Expect(container.LivenessProbe.Exec.Command).To(Equal([]string{
				"sh", "-c", "wget -q -O - http://localhost:8080/health || exit 1",
			}))
			Expect(container.LivenessProbe.PeriodSeconds).To(Equal(int32(30)))
			Expect(container.LivenessProbe.TimeoutSeconds).To(Equal(int32(5)))
			Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(3)))
			Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(30)))

			// Check readiness probe
			Expect(container.ReadinessProbe).NotTo(BeNil())
			Expect(container.ReadinessProbe.Exec).NotTo(BeNil())
			Expect(container.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(10))) // Shorter for readiness
		})

		It("should convert HTTP health check to HTTP probe", func() {
			taskDef := &storage.TaskDefinition{
				Family:   "test-family",
				Revision: 1,
				ContainerDefinitions: mustMarshal([]map[string]interface{}{
					{
						"name":  "app",
						"image": "nginx:latest",
						"healthCheck": map[string]interface{}{
							"command":     []interface{}{"HTTP", "/health", "8080"},
							"interval":    float64(15),
							"timeout":     float64(3),
							"retries":     float64(2),
							"startPeriod": float64(60),
						},
					},
				}),
			}

			deployment, _, err := converter.ConvertServiceToDeployment(service, taskDef, cluster)
			Expect(err).NotTo(HaveOccurred())

			container := deployment.Spec.Template.Spec.Containers[0]

			// Check liveness probe
			Expect(container.LivenessProbe).NotTo(BeNil())
			Expect(container.LivenessProbe.HTTPGet).NotTo(BeNil())
			Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/health"))
			Expect(container.LivenessProbe.HTTPGet.Port).To(Equal(intstr.FromInt(8080)))
			Expect(container.LivenessProbe.PeriodSeconds).To(Equal(int32(15)))
			Expect(container.LivenessProbe.TimeoutSeconds).To(Equal(int32(3)))
			Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(2)))
			Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(60)))
		})

		It("should convert CMD health check to exec probe", func() {
			taskDef := &storage.TaskDefinition{
				Family:   "test-family",
				Revision: 1,
				ContainerDefinitions: mustMarshal([]map[string]interface{}{
					{
						"name":  "app",
						"image": "nginx:latest",
						"healthCheck": map[string]interface{}{
							"command":  []interface{}{"CMD", "curl", "-f", "http://localhost/health"},
							"interval": float64(20),
						},
					},
				}),
			}

			deployment, _, err := converter.ConvertServiceToDeployment(service, taskDef, cluster)
			Expect(err).NotTo(HaveOccurred())

			container := deployment.Spec.Template.Spec.Containers[0]

			// Check liveness probe
			Expect(container.LivenessProbe).NotTo(BeNil())
			Expect(container.LivenessProbe.Exec).NotTo(BeNil())
			Expect(container.LivenessProbe.Exec.Command).To(Equal([]string{
				"curl", "-f", "http://localhost/health",
			}))
			Expect(container.LivenessProbe.PeriodSeconds).To(Equal(int32(20)))
			// Check defaults
			Expect(container.LivenessProbe.TimeoutSeconds).To(Equal(int32(5)))
			Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(3)))
			Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(30)))
		})

		It("should handle containers without health check", func() {
			taskDef := &storage.TaskDefinition{
				Family:   "test-family",
				Revision: 1,
				ContainerDefinitions: mustMarshal([]map[string]interface{}{
					{
						"name":  "app",
						"image": "nginx:latest",
						// No healthCheck field
					},
				}),
			}

			deployment, _, err := converter.ConvertServiceToDeployment(service, taskDef, cluster)
			Expect(err).NotTo(HaveOccurred())

			container := deployment.Spec.Template.Spec.Containers[0]

			// Check no probes are set
			Expect(container.LivenessProbe).To(BeNil())
			Expect(container.ReadinessProbe).To(BeNil())
		})

		It("should handle multiple containers with different health checks", func() {
			taskDef := &storage.TaskDefinition{
				Family:   "test-family",
				Revision: 1,
				ContainerDefinitions: mustMarshal([]map[string]interface{}{
					{
						"name":  "app",
						"image": "app:latest",
						"healthCheck": map[string]interface{}{
							"command":  []interface{}{"CMD-SHELL", "curl -f http://localhost:3000/health"},
							"interval": float64(30),
						},
					},
					{
						"name":  "sidecar",
						"image": "sidecar:latest",
						"healthCheck": map[string]interface{}{
							"command":  []interface{}{"HTTP", "/ready", "9090"},
							"interval": float64(10),
						},
					},
				}),
			}

			deployment, _, err := converter.ConvertServiceToDeployment(service, taskDef, cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(2))

			// Check app container
			appContainer := deployment.Spec.Template.Spec.Containers[0]
			Expect(appContainer.LivenessProbe).NotTo(BeNil())
			Expect(appContainer.LivenessProbe.Exec).NotTo(BeNil())
			Expect(appContainer.LivenessProbe.Exec.Command).To(Equal([]string{
				"sh", "-c", "curl -f http://localhost:3000/health",
			}))

			// Check sidecar container
			sidecarContainer := deployment.Spec.Template.Spec.Containers[1]
			Expect(sidecarContainer.LivenessProbe).NotTo(BeNil())
			Expect(sidecarContainer.LivenessProbe.HTTPGet).NotTo(BeNil())
			Expect(sidecarContainer.LivenessProbe.HTTPGet.Path).To(Equal("/ready"))
			Expect(sidecarContainer.LivenessProbe.HTTPGet.Port).To(Equal(intstr.FromInt(9090)))
		})
	})
})

func mustMarshal(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
