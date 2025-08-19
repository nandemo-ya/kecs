package converters_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("TaskSetConverter", func() {
	var (
		converter *converters.TaskSetConverter
		taskSet   *storage.TaskSet
		service   *storage.Service
		taskDef   *storage.TaskDefinition
	)

	BeforeEach(func() {
		taskConverter := converters.NewTaskConverter("us-east-1", "000000000000")
		converter = converters.NewTaskSetConverter(taskConverter)

		// Create test TaskSet
		taskSet = &storage.TaskSet{
			ID:                   "ts-12345678",
			ARN:                  "arn:aws:ecs:us-east-1:000000000000:task-set/default/test-service/ts-12345678",
			ServiceARN:           "arn:aws:ecs:us-east-1:000000000000:service/default/test-service",
			ClusterARN:           "arn:aws:ecs:us-east-1:000000000000:cluster/default",
			ExternalID:           "blue-deployment",
			TaskDefinition:       "webapp:1",
			LaunchType:           "FARGATE",
			Status:               "ACTIVE",
			StabilityStatus:      "STEADY_STATE",
			ComputedDesiredCount: 2,
			Region:               "us-east-1",
			AccountID:            "000000000000",
		}

		// Create test Service
		service = &storage.Service{
			ServiceName:  "test-service",
			DesiredCount: 2,
		}

		// Create test TaskDefinition
		containerDefs := []generated.ContainerDefinition{
			{
				Name:  taskSetStrPtr("webapp"),
				Image: taskSetStrPtr("nginx:latest"),
				PortMappings: []generated.PortMapping{
					{
						ContainerPort: taskSetInt32Ptr(80),
						Protocol:      (*generated.TransportProtocol)(taskSetStrPtr("tcp")),
					},
				},
			},
		}
		containerDefsJSON, _ := json.Marshal(containerDefs)

		taskDef = &storage.TaskDefinition{
			ARN:                  "arn:aws:ecs:us-east-1:000000000000:task-definition/webapp:1",
			Family:               "webapp",
			Revision:             1,
			ContainerDefinitions: string(containerDefsJSON),
			NetworkMode:          "awsvpc",
			TaskRoleARN:          "arn:aws:iam::000000000000:role/ecsTaskRole",
			ExecutionRoleARN:     "arn:aws:iam::000000000000:role/ecsTaskExecutionRole",
		}
	})

	Describe("ConvertTaskSetToDeployment", func() {
		It("should convert TaskSet to Deployment", func() {
			deployment, err := converter.ConvertTaskSetToDeployment(taskSet, service, taskDef, "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(deployment).NotTo(BeNil())

			// Check deployment metadata
			Expect(deployment.Name).To(Equal("test-service-ts-12345678"))
			Expect(deployment.Namespace).To(Equal("default-us-east-1"))
			Expect(deployment.Labels["kecs.io/taskset"]).To(Equal("ts-12345678"))
			Expect(deployment.Labels["kecs.io/service"]).To(Equal("test-service"))
			Expect(deployment.Labels["kecs.io/taskset-external-id"]).To(Equal("blue-deployment"))

			// Check replicas
			Expect(*deployment.Spec.Replicas).To(Equal(int32(2)))

			// Check selector
			Expect(deployment.Spec.Selector.MatchLabels["kecs.io/taskset"]).To(Equal("ts-12345678"))
		})

		It("should handle TaskSet with scale configuration", func() {
			scale := generated.Scale{
				Value: taskSetFloat64Ptr(50.0),
				Unit:  (*generated.ScaleUnit)(taskSetStrPtr("PERCENT")),
			}
			scaleJSON, _ := json.Marshal(scale)
			taskSet.Scale = string(scaleJSON)

			deployment, err := converter.ConvertTaskSetToDeployment(taskSet, service, taskDef, "default")
			Expect(err).NotTo(HaveOccurred())

			// 50% of 2 desired count = 1 replica
			Expect(*deployment.Spec.Replicas).To(Equal(int32(1)))
		})
	})

	Describe("ConvertTaskSetToService", func() {
		It("should create Service for TaskSet with port mappings", func() {
			k8sService, err := converter.ConvertTaskSetToService(taskSet, service, taskDef, "default", false)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sService).NotTo(BeNil())

			// Check service metadata
			Expect(k8sService.Name).To(Equal("test-service-ts-12345678-svc"))
			Expect(k8sService.Namespace).To(Equal("default-us-east-1"))
			Expect(k8sService.Labels["kecs.io/taskset"]).To(Equal("ts-12345678"))

			// Check ports
			Expect(k8sService.Spec.Ports).To(HaveLen(1))
			Expect(k8sService.Spec.Ports[0].Port).To(Equal(int32(80)))
			Expect(k8sService.Spec.Ports[0].Name).To(Equal("tcp-80"))

			// Check selector
			Expect(k8sService.Spec.Selector["kecs.io/taskset"]).To(Equal("ts-12345678"))
		})

		It("should return nil for TaskSet without port mappings", func() {
			// TaskDefinition without port mappings
			containerDefs := []generated.ContainerDefinition{
				{
					Name:  taskSetStrPtr("worker"),
					Image: taskSetStrPtr("busybox:latest"),
				},
			}
			containerDefsJSON, _ := json.Marshal(containerDefs)
			taskDef.ContainerDefinitions = string(containerDefsJSON)

			k8sService, err := converter.ConvertTaskSetToService(taskSet, service, taskDef, "default", false)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sService).To(BeNil())
		})
	})

	Describe("GetReplicas", func() {
		It("should calculate replicas from percentage scale", func() {
			scale := generated.Scale{
				Value: taskSetFloat64Ptr(75.0),
				Unit:  (*generated.ScaleUnit)(taskSetStrPtr("PERCENT")),
			}
			scaleJSON, _ := json.Marshal(scale)
			taskSet.Scale = string(scaleJSON)
			service.DesiredCount = 4

			replicas := converter.GetReplicas(taskSet, service)
			Expect(*replicas).To(Equal(int32(3))) // 75% of 4 = 3
		})

		It("should use absolute count scale", func() {
			scale := generated.Scale{
				Value: taskSetFloat64Ptr(5.0),
				Unit:  (*generated.ScaleUnit)(taskSetStrPtr("COUNT")),
			}
			scaleJSON, _ := json.Marshal(scale)
			taskSet.Scale = string(scaleJSON)

			replicas := converter.GetReplicas(taskSet, service)
			Expect(*replicas).To(Equal(int32(5)))
		})

		It("should use computed desired count when no scale", func() {
			taskSet.ComputedDesiredCount = 3
			replicas := converter.GetReplicas(taskSet, service)
			Expect(*replicas).To(Equal(int32(3)))
		})
	})

	Describe("GetTaskSetStatusFromDeployment", func() {
		It("should extract status from deployment", func() {
			deployment := &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Replicas:      5,
					ReadyReplicas: 3,
				},
			}

			runningCount, pendingCount := converter.GetTaskSetStatusFromDeployment(deployment)
			Expect(runningCount).To(Equal(int64(3)))
			Expect(pendingCount).To(Equal(int64(2)))
		})
	})

	Describe("Helper Methods", func() {
		It("should generate valid deployment name", func() {
			name := converter.GetDeploymentName("my_service-name", "ts-abc123")
			Expect(name).To(Equal("my-service-name-ts-abc123"))
		})

		It("should generate valid service name", func() {
			name := converter.GetServiceName("my-service", "ts-abc123")
			Expect(name).To(Equal("my-service-ts-abc123-svc"))
		})

		It("should return correct namespace", func() {
			namespace := converter.GetNamespace("test-cluster", "us-west-2")
			Expect(namespace).To(Equal("test-cluster-us-west-2"))
		})
	})
})

// Helper functions
func taskSetStrPtr(s string) *string {
	return &s
}

func taskSetInt32Ptr(i int32) *int32 {
	return &i
}

func taskSetFloat64Ptr(f float64) *float64 {
	return &f
}

