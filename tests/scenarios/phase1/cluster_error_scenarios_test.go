package phase1

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster Error Scenarios", Serial, func() {
	var (
		client utils.ECSClientInterface
		logger *utils.TestLogger
	)

	BeforeEach(func() {
		// Use shared resources from suite
		client = sharedClient
		logger = sharedLogger
	})

	Describe("Resource Conflicts", func() {
		Context("when deleting a cluster with active services", func() {
			var clusterName string
			var serviceName string
			var taskDefArn string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("conflict-cluster")
				serviceName = utils.GenerateTestName("conflict-service")

				// Create cluster
				Expect(client.CreateCluster(clusterName)).To(Succeed())

				// Register a task definition
				taskDef := `{
					"family": "conflict-task",
					"containerDefinitions": [{
						"name": "app",
						"image": "nginx:latest",
						"memory": 128,
						"essential": true
					}]
				}`
				td, err := client.RegisterTaskDefinition("conflict-task", taskDef)
				Expect(err).NotTo(HaveOccurred())
				taskDefArn = td.TaskDefinitionArn

				// Create a service with unique name
				err = client.CreateService(clusterName, serviceName, taskDefArn, 1)
				Expect(err).NotTo(HaveOccurred())

				// Verify service was created using Eventually
				Eventually(func() (int, error) {
					services, err := client.ListServices(clusterName)
					if err != nil {
						return 0, err
					}
					return len(services), nil
				}, 10*time.Second, 500*time.Millisecond).Should(BeNumerically(">", 0), "Service should be created")

				DeferCleanup(func() {
					// Clean up service first
					_ = client.DeleteService(clusterName, serviceName)
					// Then cluster
					_ = client.DeleteCluster(clusterName)
					// Deregister task definition
					_ = client.DeregisterTaskDefinition(taskDefArn)
				})
			})

			It("should fail to delete cluster with active service", func() {
				logger.Info("Attempting to delete cluster with active service: %s", clusterName)

				err := client.DeleteCluster(clusterName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Or(
					ContainSubstring("active"),
					ContainSubstring("services are active"),
				))
			})
		})

		Context("when deleting a cluster with running tasks", func() {
			var clusterName string
			var taskDefArn string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("task-conflict")

				// Create cluster
				Expect(client.CreateCluster(clusterName)).To(Succeed())

				// Register a task definition
				taskDef := `{
					"family": "running-task",
					"containerDefinitions": [{
						"name": "app",
						"image": "busybox",
						"command": ["sleep", "300"],
						"memory": 128,
						"essential": true
					}]
				}`
				td, err := client.RegisterTaskDefinition("running-task", taskDef)
				Expect(err).NotTo(HaveOccurred())
				taskDefArn = td.TaskDefinitionArn

				// Run a task
				runResp, err := client.RunTask(clusterName, taskDefArn, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(runResp.Tasks)).To(BeNumerically(">", 0))

				// Wait for task to be in running state
				Eventually(func() (int, error) {
					tasks, err := client.ListTasks(clusterName, "")
					if err != nil {
						return 0, err
					}
					return len(tasks), nil
				}, 10*time.Second, 500*time.Millisecond).Should(BeNumerically(">", 0), "Task should be running")

				DeferCleanup(func() {
					// Stop any running tasks
					tasks, _ := client.ListTasks(clusterName, "")
					for _, taskArn := range tasks {
						_ = client.StopTask(clusterName, taskArn, "cleanup")
					}
					// Delete cluster
					_ = client.DeleteCluster(clusterName)
					// Deregister task definition
					_ = client.DeregisterTaskDefinition(taskDefArn)
				})
			})

			It("should fail to delete cluster with running tasks", func() {
				logger.Info("Attempting to delete cluster with running tasks: %s", clusterName)

				err := client.DeleteCluster(clusterName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Or(
					ContainSubstring("tasks"),
					ContainSubstring("active"),
				))
			})
		})
	})

	Describe("Operation on Non-existent Resources", func() {
		Context("when operating on non-existent clusters", func() {
			nonExistentCluster := "non-existent-cluster-ops"

			It("should fail to create service on non-existent cluster", func() {
				logger.Info("Testing create service on non-existent cluster")

				err := client.CreateService(nonExistentCluster, "test-service", "nginx:latest", 1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not exist"))
			})

			It("should fail to run task on non-existent cluster", func() {
				logger.Info("Testing run task on non-existent cluster")

				_, err := client.RunTask(nonExistentCluster, "nginx:latest", 1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not exist"))
			})
		})
	})
})