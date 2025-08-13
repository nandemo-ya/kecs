package converters_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

var _ = Describe("TaskConverter Health Check", func() {
	var (
		converter *converters.TaskConverter
		cluster   *storage.Cluster
	)

	BeforeEach(func() {
		converter = converters.NewTaskConverter("us-east-1", "123456789012")
		cluster = &storage.Cluster{
			Name:   "test-cluster",
			Region: "us-east-1",
			Status: "ACTIVE",
		}
	})

	Context("when task definition has health check", func() {
		It("should convert CMD-SHELL health check to exec probe", func() {
			containerDef := types.ContainerDefinition{
				Name:  strPtr("app"),
				Image: strPtr("nginx:latest"),
				HealthCheck: &types.HealthCheck{
					Command:     []string{"CMD-SHELL", "wget -q -O - http://localhost/health || exit 1"},
					Interval:    int32Ptr(30),
					Timeout:     int32Ptr(5),
					Retries:     int32Ptr(3),
					StartPeriod: int32Ptr(60),
				},
			}

			taskDef := &storage.TaskDefinition{
				Family:               "test-family",
				Revision:             1,
				ContainerDefinitions: mustMarshalContainerDefs([]types.ContainerDefinition{containerDef}),
			}

			pod, err := converter.ConvertTaskToPod(taskDef, []byte("{}"), cluster, "test-task-123")
			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())
			Expect(pod.Spec.Containers).To(HaveLen(1))

			container := pod.Spec.Containers[0]

			// Check liveness probe
			Expect(container.LivenessProbe).NotTo(BeNil())
			Expect(container.LivenessProbe.Exec).NotTo(BeNil())
			Expect(container.LivenessProbe.Exec.Command).To(Equal([]string{
				"sh", "-c", "wget -q -O - http://localhost/health || exit 1",
			}))
			Expect(container.LivenessProbe.PeriodSeconds).To(Equal(int32(30)))
			Expect(container.LivenessProbe.TimeoutSeconds).To(Equal(int32(5)))
			Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(3)))
			Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(60)))

			// Check readiness probe
			Expect(container.ReadinessProbe).NotTo(BeNil())
			Expect(container.ReadinessProbe.Exec).NotTo(BeNil())
			// Readiness probe should have shorter initial delay
			Expect(container.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(10)))
			Expect(container.ReadinessProbe.PeriodSeconds).To(Equal(int32(30)))
		})

		It("should convert HTTP health check to HTTP probe", func() {
			containerDef := types.ContainerDefinition{
				Name:  strPtr("app"),
				Image: strPtr("nginx:latest"),
				HealthCheck: &types.HealthCheck{
					Command:     []string{"HTTP", "/health", "8080"},
					Interval:    int32Ptr(15),
					Timeout:     int32Ptr(3),
					Retries:     int32Ptr(2),
					StartPeriod: int32Ptr(45),
				},
			}

			taskDef := &storage.TaskDefinition{
				Family:               "test-family",
				Revision:             1,
				ContainerDefinitions: mustMarshalContainerDefs([]types.ContainerDefinition{containerDef}),
			}

			pod, err := converter.ConvertTaskToPod(taskDef, []byte("{}"), cluster, "test-task-123")
			Expect(err).NotTo(HaveOccurred())

			container := pod.Spec.Containers[0]

			// Check liveness probe
			Expect(container.LivenessProbe).NotTo(BeNil())
			Expect(container.LivenessProbe.HTTPGet).NotTo(BeNil())
			Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/health"))
			Expect(container.LivenessProbe.HTTPGet.Port).To(Equal(intstr.FromInt(8080)))
			Expect(container.LivenessProbe.PeriodSeconds).To(Equal(int32(15)))
			Expect(container.LivenessProbe.TimeoutSeconds).To(Equal(int32(3)))
			Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(2)))
			Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(45)))

			// Check readiness probe
			Expect(container.ReadinessProbe).NotTo(BeNil())
			Expect(container.ReadinessProbe.HTTPGet).NotTo(BeNil())
			Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/health"))
			Expect(container.ReadinessProbe.HTTPGet.Port).To(Equal(intstr.FromInt(8080)))
			Expect(container.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(10))) // Shorter for readiness
		})

		It("should convert CMD health check to exec probe", func() {
			containerDef := types.ContainerDefinition{
				Name:  strPtr("app"),
				Image: strPtr("nginx:latest"),
				HealthCheck: &types.HealthCheck{
					Command:  []string{"CMD", "/bin/health-check", "--verbose"},
					Interval: int32Ptr(20),
					Timeout:  int32Ptr(10),
				},
			}

			taskDef := &storage.TaskDefinition{
				Family:               "test-family",
				Revision:             1,
				ContainerDefinitions: mustMarshalContainerDefs([]types.ContainerDefinition{containerDef}),
			}

			pod, err := converter.ConvertTaskToPod(taskDef, []byte("{}"), cluster, "test-task-123")
			Expect(err).NotTo(HaveOccurred())

			container := pod.Spec.Containers[0]

			// Check liveness probe
			Expect(container.LivenessProbe).NotTo(BeNil())
			Expect(container.LivenessProbe.Exec).NotTo(BeNil())
			Expect(container.LivenessProbe.Exec.Command).To(Equal([]string{
				"/bin/health-check", "--verbose",
			}))
			Expect(container.LivenessProbe.PeriodSeconds).To(Equal(int32(20)))
			Expect(container.LivenessProbe.TimeoutSeconds).To(Equal(int32(10)))
			// Check defaults
			Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(3)))
			Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(30)))
		})

		It("should handle containers without health check", func() {
			containerDef := types.ContainerDefinition{
				Name:  strPtr("app"),
				Image: strPtr("nginx:latest"),
				// No HealthCheck field
			}

			taskDef := &storage.TaskDefinition{
				Family:               "test-family",
				Revision:             1,
				ContainerDefinitions: mustMarshalContainerDefs([]types.ContainerDefinition{containerDef}),
			}

			pod, err := converter.ConvertTaskToPod(taskDef, []byte("{}"), cluster, "test-task-123")
			Expect(err).NotTo(HaveOccurred())

			container := pod.Spec.Containers[0]

			// Check no probes are set
			Expect(container.LivenessProbe).To(BeNil())
			Expect(container.ReadinessProbe).To(BeNil())
		})

		It("should use defaults when timing parameters are not specified", func() {
			containerDef := types.ContainerDefinition{
				Name:  strPtr("app"),
				Image: strPtr("nginx:latest"),
				HealthCheck: &types.HealthCheck{
					Command: []string{"CMD-SHELL", "echo ok"},
					// No timing parameters specified
				},
			}

			taskDef := &storage.TaskDefinition{
				Family:               "test-family",
				Revision:             1,
				ContainerDefinitions: mustMarshalContainerDefs([]types.ContainerDefinition{containerDef}),
			}

			pod, err := converter.ConvertTaskToPod(taskDef, []byte("{}"), cluster, "test-task-123")
			Expect(err).NotTo(HaveOccurred())

			container := pod.Spec.Containers[0]

			// Check liveness probe uses defaults
			Expect(container.LivenessProbe).NotTo(BeNil())
			Expect(container.LivenessProbe.PeriodSeconds).To(Equal(int32(30)))
			Expect(container.LivenessProbe.TimeoutSeconds).To(Equal(int32(5)))
			Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(3)))
			Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(30)))
			Expect(container.LivenessProbe.SuccessThreshold).To(Equal(int32(1)))
		})

		It("should handle multiple containers with different health checks", func() {
			containerDefs := []types.ContainerDefinition{
				{
					Name:  strPtr("app"),
					Image: strPtr("app:latest"),
					HealthCheck: &types.HealthCheck{
						Command:  []string{"CMD-SHELL", "curl -f http://localhost:3000/health"},
						Interval: int32Ptr(30),
					},
				},
				{
					Name:  strPtr("sidecar"),
					Image: strPtr("sidecar:latest"),
					HealthCheck: &types.HealthCheck{
						Command:  []string{"HTTP", "/status", "9090"},
						Interval: int32Ptr(10),
					},
				},
				{
					Name:  strPtr("no-health"),
					Image: strPtr("simple:latest"),
					// No health check
				},
			}

			taskDef := &storage.TaskDefinition{
				Family:               "test-family",
				Revision:             1,
				ContainerDefinitions: mustMarshalContainerDefs(containerDefs),
			}

			pod, err := converter.ConvertTaskToPod(taskDef, []byte("{}"), cluster, "test-task-123")
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Spec.Containers).To(HaveLen(3))

			// Check app container
			appContainer := pod.Spec.Containers[0]
			Expect(appContainer.LivenessProbe).NotTo(BeNil())
			Expect(appContainer.LivenessProbe.Exec).NotTo(BeNil())
			Expect(appContainer.LivenessProbe.Exec.Command).To(Equal([]string{
				"sh", "-c", "curl -f http://localhost:3000/health",
			}))

			// Check sidecar container
			sidecarContainer := pod.Spec.Containers[1]
			Expect(sidecarContainer.LivenessProbe).NotTo(BeNil())
			Expect(sidecarContainer.LivenessProbe.HTTPGet).NotTo(BeNil())
			Expect(sidecarContainer.LivenessProbe.HTTPGet.Path).To(Equal("/status"))
			Expect(sidecarContainer.LivenessProbe.HTTPGet.Port).To(Equal(intstr.FromInt(9090)))

			// Check no-health container
			noHealthContainer := pod.Spec.Containers[2]
			Expect(noHealthContainer.LivenessProbe).To(BeNil())
			Expect(noHealthContainer.ReadinessProbe).To(BeNil())
		})
	})
})

func strPtr(v string) *string {
	return &v
}

func int32Ptr(v int) *int {
	return &v
}

func mustMarshalContainerDefs(defs []types.ContainerDefinition) string {
	data, err := json.Marshal(defs)
	if err != nil {
		panic(err)
	}
	return string(data)
}
