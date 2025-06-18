package basic_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Service Lifecycle", func() {
	var (
		kecs        *utils.KECSContainer
		ecsClient   utils.ECSClientInterface
		clusterName string
		taskDefArn  string
	)

	BeforeEach(func() {
		// Start KECS
		kecs = utils.StartKECS(GinkgoT())
		ecsClient = utils.NewCurlClient(kecs.Endpoint())

		// Create test cluster
		clusterName = fmt.Sprintf("service-test-%d", time.Now().Unix())
		err := ecsClient.CreateCluster(clusterName)
		Expect(err).NotTo(HaveOccurred())

		// Register a simple task definition
		taskDef, err := ecsClient.RegisterTaskDefinition("test-task", createSimpleTaskDefinition())
		Expect(err).NotTo(HaveOccurred())
		taskDefArn = taskDef.TaskDefinitionArn
	})

	AfterEach(func() {
		if ecsClient != nil && clusterName != "" {
			// Cleanup services
			services, _ := ecsClient.ListServices(clusterName)
			for _, serviceName := range services {
				ecsClient.DeleteService(clusterName, serviceName)
			}

			// Delete cluster
			ecsClient.DeleteCluster(clusterName)
		}

		if kecs != nil {
			kecs.Cleanup()
		}
	})

	Describe("Basic Service Operations", func() {
		It("should create and delete a service", func() {
			serviceName := fmt.Sprintf("test-service-%d", time.Now().Unix())

			By("Creating a service")
			err := ecsClient.CreateService(clusterName, serviceName, taskDefArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service exists")
			service, err := ecsClient.DescribeService(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.ServiceName).To(Equal(serviceName))
			Expect(service.DesiredCount).To(Equal(1))

			By("Listing services")
			services, err := ecsClient.ListServices(clusterName)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(ContainElement(ContainSubstring(serviceName)))

			By("Deleting the service")
			err = ecsClient.DeleteService(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service is deleted")
			Eventually(func() error {
				_, err := ecsClient.DescribeService(clusterName, serviceName)
				return err
			}, 30*time.Second, 2*time.Second).Should(HaveOccurred())
		})

		It("should scale a service", func() {
			serviceName := fmt.Sprintf("scale-test-%d", time.Now().Unix())

			By("Creating a service with 1 instance")
			err := ecsClient.CreateService(clusterName, serviceName, taskDefArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for service to be stable")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				return err == nil && service.RunningCount == 1
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Scaling service to 3 instances")
			desiredCount := 3
			err = ecsClient.UpdateService(clusterName, serviceName, &desiredCount, "")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service scaled up")
			Eventually(func() int {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				if err != nil {
					return 0
				}
				return service.RunningCount
			}, 120*time.Second, 10*time.Second).Should(Equal(3))

			By("Scaling service down to 0")
			desiredCount = 0
			err = ecsClient.UpdateService(clusterName, serviceName, &desiredCount, "")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service scaled down")
			Eventually(func() int {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				if err != nil {
					return -1
				}
				return service.RunningCount
			}, 60*time.Second, 5*time.Second).Should(Equal(0))

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})

		It("should update service task definition", func() {
			serviceName := fmt.Sprintf("update-test-%d", time.Now().Unix())

			By("Creating a service")
			err := ecsClient.CreateService(clusterName, serviceName, taskDefArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Registering new task definition")
			newTaskDef, err := ecsClient.RegisterTaskDefinition("test-task-v2", createSimpleTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Updating service with new task definition")
			err = ecsClient.UpdateService(clusterName, serviceName, nil, newTaskDef.TaskDefinitionArn)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service updated")
			Eventually(func() string {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				if err != nil {
					return ""
				}
				return service.TaskDefinition
			}, 60*time.Second, 5*time.Second).Should(Equal(newTaskDef.TaskDefinitionArn))

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})
	})

	Describe("Service Configuration", func() {
		It("should create service with different configurations", func() {
			configs := []struct {
				name         string
				desiredCount int
				description  string
			}{
				{"single-instance", 1, "Single instance service"},
				{"multi-instance", 3, "Multi-instance service"},
				{"zero-instance", 0, "Zero instance service"},
			}

			for _, config := range configs {
				serviceName := fmt.Sprintf("%s-%d", config.name, time.Now().Unix())

				By(fmt.Sprintf("Creating %s", config.description))
				err := ecsClient.CreateService(clusterName, serviceName, taskDefArn, config.desiredCount)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Verifying %s configuration", config.description))
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				Expect(err).NotTo(HaveOccurred())
				Expect(service.DesiredCount).To(Equal(config.desiredCount))

				By(fmt.Sprintf("Cleanup %s", config.description))
				ecsClient.DeleteService(clusterName, serviceName)
			}
		})
	})

	Describe("Service Error Scenarios", func() {
		It("should handle invalid cluster name", func() {
			serviceName := fmt.Sprintf("invalid-cluster-test-%d", time.Now().Unix())

			By("Attempting to create service in non-existent cluster")
			err := ecsClient.CreateService("non-existent-cluster", serviceName, taskDefArn, 1)
			Expect(err).To(HaveOccurred())
		})

		It("should handle invalid task definition", func() {
			serviceName := fmt.Sprintf("invalid-taskdef-test-%d", time.Now().Unix())

			By("Attempting to create service with non-existent task definition")
			err := ecsClient.CreateService(clusterName, serviceName, "non-existent:1", 1)
			Expect(err).To(HaveOccurred())
		})

		It("should handle duplicate service names", func() {
			serviceName := fmt.Sprintf("duplicate-test-%d", time.Now().Unix())

			By("Creating first service")
			err := ecsClient.CreateService(clusterName, serviceName, taskDefArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Attempting to create duplicate service")
			err = ecsClient.CreateService(clusterName, serviceName, taskDefArn, 1)
			Expect(err).To(HaveOccurred())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})
	})
})

func createSimpleTaskDefinition() string {
	return `{
		"family": "test-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "test-container",
				"image": "nginx:alpine",
				"portMappings": [
					{
						"containerPort": 80,
						"protocol": "tcp"
					}
				],
				"essential": true,
				"memory": 256
			}
		]
	}`
}