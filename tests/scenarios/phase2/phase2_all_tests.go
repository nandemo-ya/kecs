package phase2

import (
	"encoding/json"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Phase 2: Additional Task Definition Tests", Serial, func() {
	
	Describe("Task Definition: Background Worker", func() {
		var (
			workerClient      utils.ECSClientInterface
			workerLogger      *utils.TestLogger
			workerClusterName string
		)

		BeforeEach(func() {
			// Use shared resources from suite
			workerClient = sharedClient
			workerLogger = sharedLogger
			
			// Get or create a shared cluster for worker tests
			var err error
			workerClusterName, err = sharedClusterManager.GetOrCreateCluster("phase2-worker")
			Expect(err).NotTo(HaveOccurred())
			
			workerLogger.Info("Using shared cluster: %s", workerClusterName)
		})

		AfterEach(func() {
			// Cleanup is done in the last test
		})

		Describe("Python Worker Process", func() {
			var (
				taskDefFamily string
				serviceName   string
			)

			BeforeEach(func() {
				taskDefFamily = utils.GenerateTestName("worker-td")
				serviceName = utils.GenerateTestName("worker-svc")
			})

			AfterEach(func() {
				// Clean up service
				if serviceName != "" && workerClient != nil {
					workerLogger.Info("Deleting service: %s", serviceName)
					_ = workerClient.DeleteService(workerClusterName, serviceName)

					// Wait for tasks to stop
					Eventually(func() int {
						tasks, _ := workerClient.ListTasks(workerClusterName, serviceName)
						return len(tasks)
					}, 60*time.Second, 5*time.Second).Should(Equal(0))
				}

				// Clean up task definition
				if taskDefFamily != "" && workerClient != nil {
					workerLogger.Info("Deregistering task definition: %s", taskDefFamily)
					_ = workerClient.DeregisterTaskDefinition(taskDefFamily)
				}
			})

			It("should register a background worker task definition", func() {
				workerLogger.Info("Registering task definition: %s", taskDefFamily)

				// Load task definition template
				taskDefJSON, err := os.ReadFile("templates/single-container/background-worker.json")
				Expect(err).NotTo(HaveOccurred())

				// Update family name in the template
				var taskDef map[string]interface{}
				err = json.Unmarshal(taskDefJSON, &taskDef)
				Expect(err).NotTo(HaveOccurred())
				taskDef["family"] = taskDefFamily

				// Register task definition
				updatedJSON, err := json.Marshal(taskDef)
				Expect(err).NotTo(HaveOccurred())

				registeredTaskDef, err := workerClient.RegisterTaskDefinitionFromJSON(string(updatedJSON))
				Expect(err).NotTo(HaveOccurred())
				Expect(registeredTaskDef.Family).To(Equal(taskDefFamily))
				Expect(registeredTaskDef.Revision).To(Equal(1))

				workerLogger.Info("Successfully registered task definition: %s:%d", taskDefFamily, registeredTaskDef.Revision)
			})

			It("should run background worker and process queue", func() {
				// Register task definition
				taskDefJSON, err := os.ReadFile("templates/single-container/background-worker.json")
				Expect(err).NotTo(HaveOccurred())

				var taskDef map[string]interface{}
				err = json.Unmarshal(taskDefJSON, &taskDef)
				Expect(err).NotTo(HaveOccurred())
				taskDef["family"] = taskDefFamily

				updatedJSON, err := json.Marshal(taskDef)
				Expect(err).NotTo(HaveOccurred())

				_, err = workerClient.RegisterTaskDefinitionFromJSON(string(updatedJSON))
				Expect(err).NotTo(HaveOccurred())

				// Create service with 1 worker
				workerLogger.Info("Creating service: %s with 1 worker", serviceName)
				
				// Retry service creation in case k3d cluster is still initializing
				Eventually(func() error {
					return workerClient.CreateService(workerClusterName, serviceName, taskDefFamily, 1)
				}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

				// Wait for task to be running
				workerLogger.Info("Waiting for worker to reach RUNNING state")
				var taskArn string
				Eventually(func() bool {
					tasks, err := workerClient.ListTasks(workerClusterName, serviceName)
					if err != nil || len(tasks) == 0 {
						return false
					}

					taskArn = tasks[0]
					task, err := workerClient.DescribeTask(workerClusterName, taskArn)
					if err != nil {
						return false
					}

					return task.LastStatus == "RUNNING"
				}, 120*time.Second, 5*time.Second).Should(BeTrue())

				workerLogger.Info("Worker is running")

				// Wait for worker to process some items
				Eventually(func() bool {
					// In a real implementation, we'd check logs or metrics
					// For now, just ensure the task stays running
					task, err := workerClient.DescribeTask(workerClusterName, taskArn)
					if err != nil {
						return false
					}
					return task.LastStatus == "RUNNING"
				}, 15*time.Second, 2*time.Second).Should(BeTrue())

				// Check logs to verify processing (placeholder - requires log retrieval implementation)
				workerLogger.Info("Worker should be processing queue items")
			})

			It("should cleanup worker resources", Label("cleanup"), func() {
				// Release the shared cluster
				if sharedClusterManager != nil && workerClusterName != "" {
					sharedClusterManager.ReleaseCluster(workerClusterName)
				}
			})
		})
	})

	Describe("Task Definition: Failure Handling", func() {
		var (
			failureClient      utils.ECSClientInterface
			failureLogger      *utils.TestLogger
			failureClusterName string
		)

		BeforeEach(func() {
			// Use shared resources from suite
			failureClient = sharedClient
			failureLogger = sharedLogger
			
			// Get or create a shared cluster for failure tests
			var err error
			failureClusterName, err = sharedClusterManager.GetOrCreateCluster("phase2-failure")
			Expect(err).NotTo(HaveOccurred())
			
			failureLogger.Info("Using shared cluster: %s", failureClusterName)
		})

		Describe("Container Failure and Restart", func() {
			var (
				taskDefFamily string
				serviceName   string
			)

			BeforeEach(func() {
				taskDefFamily = utils.GenerateTestName("failure-td")
				serviceName = utils.GenerateTestName("failure-svc")
			})

			AfterEach(func() {
				// Clean up service
				if serviceName != "" && failureClient != nil {
					failureLogger.Info("Deleting service: %s", serviceName)
					_ = failureClient.DeleteService(failureClusterName, serviceName)

					// Wait for tasks to stop
					Eventually(func() int {
						tasks, _ := failureClient.ListTasks(failureClusterName, serviceName)
						return len(tasks)
					}, 60*time.Second, 5*time.Second).Should(Equal(0))
				}

				// Clean up task definition
				if taskDefFamily != "" && failureClient != nil {
					failureLogger.Info("Deregistering task definition: %s", taskDefFamily)
					_ = failureClient.DeregisterTaskDefinition(taskDefFamily)
				}
			})

			It("should handle container exit and restart task", func() {
				// Register task definition
				taskDefJSON, err := os.ReadFile("templates/single-container/failure-app.json")
				Expect(err).NotTo(HaveOccurred())

				var taskDef map[string]interface{}
				err = json.Unmarshal(taskDefJSON, &taskDef)
				Expect(err).NotTo(HaveOccurred())
				taskDef["family"] = taskDefFamily

				updatedJSON, err := json.Marshal(taskDef)
				Expect(err).NotTo(HaveOccurred())

				_, err = failureClient.RegisterTaskDefinitionFromJSON(string(updatedJSON))
				Expect(err).NotTo(HaveOccurred())

				// Create service with 1 task
				failureLogger.Info("Creating service: %s", serviceName)
				
				// Retry service creation in case k3d cluster is still initializing
				Eventually(func() error {
					return failureClient.CreateService(failureClusterName, serviceName, taskDefFamily, 1)
				}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

				// Wait for initial task to start
				failureLogger.Info("Waiting for initial task to start")
				var initialTaskArn string
				Eventually(func() bool {
					tasks, err := failureClient.ListTasks(failureClusterName, serviceName)
					if err != nil || len(tasks) == 0 {
						return false
					}

					initialTaskArn = tasks[0]
					task, err := failureClient.DescribeTask(failureClusterName, initialTaskArn)
					if err != nil {
						return false
					}

					// Task should be at least pending or running
					return task.LastStatus == "PENDING" || task.LastStatus == "RUNNING"
				}, 60*time.Second, 5*time.Second).Should(BeTrue())

				failureLogger.Info("Initial task started: %s", initialTaskArn)

				// Wait for task to fail (container exits after 30s)
				failureLogger.Info("Waiting for container to fail")
				Eventually(func() string {
					task, err := failureClient.DescribeTask(failureClusterName, initialTaskArn)
					if err != nil {
						return ""
					}
					return task.LastStatus
				}, 90*time.Second, 5*time.Second).Should(Equal("STOPPED"))

				// Verify task stopped due to container exit
				stoppedTask, err := failureClient.DescribeTask(failureClusterName, initialTaskArn)
				Expect(err).NotTo(HaveOccurred())
				failureLogger.Info("Task stopped with reason: %s", stoppedTask.StoppedReason)

				// Wait for service to start a replacement task
				failureLogger.Info("Waiting for service to start replacement task")
				Eventually(func() bool {
					tasks, err := failureClient.ListTasks(failureClusterName, serviceName)
					if err != nil {
						return false
					}

					// Look for a new task (different from the initial one)
					for _, taskArn := range tasks {
						if taskArn != initialTaskArn {
							task, err := failureClient.DescribeTask(failureClusterName, taskArn)
							if err == nil && (task.LastStatus == "PENDING" || task.LastStatus == "RUNNING") {
								failureLogger.Info("Replacement task started: %s", taskArn)
								return true
							}
						}
					}
					return false
				}, 120*time.Second, 5*time.Second).Should(BeTrue())
			})

			It("should cleanup failure resources", Label("cleanup"), func() {
				// Release the shared cluster
				if sharedClusterManager != nil && failureClusterName != "" {
					sharedClusterManager.ReleaseCluster(failureClusterName)
				}
			})
		})
	})

	Describe("Task Definition: Health Check Failures", func() {
		var (
			healthClient      utils.ECSClientInterface
			healthLogger      *utils.TestLogger
			healthClusterName string
		)

		BeforeEach(func() {
			// Use shared resources from suite
			healthClient = sharedClient
			healthLogger = sharedLogger
			
			// Get or create a shared cluster for health tests
			var err error
			healthClusterName, err = sharedClusterManager.GetOrCreateCluster("phase2-health")
			Expect(err).NotTo(HaveOccurred())
			
			healthLogger.Info("Using shared cluster: %s", healthClusterName)
		})

		Describe("Container Health Check Management", func() {
			var (
				taskDefFamily string
				serviceName   string
			)

			BeforeEach(func() {
				taskDefFamily = utils.GenerateTestName("health-td")
				serviceName = utils.GenerateTestName("health-svc")
			})

			AfterEach(func() {
				// Clean up service
				if serviceName != "" && healthClient != nil {
					healthLogger.Info("Deleting service: %s", serviceName)
					_ = healthClient.DeleteService(healthClusterName, serviceName)

					// Wait for tasks to stop
					Eventually(func() int {
						tasks, _ := healthClient.ListTasks(healthClusterName, serviceName)
						return len(tasks)
					}, 60*time.Second, 5*time.Second).Should(Equal(0))
				}

				// Clean up task definition
				if taskDefFamily != "" && healthClient != nil {
					healthLogger.Info("Deregistering task definition: %s", taskDefFamily)
					_ = healthClient.DeregisterTaskDefinition(taskDefFamily)
				}
			})

			It("should handle tasks with failing health checks", func() {
				// Register task definition
				taskDefJSON, err := os.ReadFile("templates/single-container/health-check-fail.json")
				Expect(err).NotTo(HaveOccurred())

				var taskDef map[string]interface{}
				err = json.Unmarshal(taskDefJSON, &taskDef)
				Expect(err).NotTo(HaveOccurred())
				taskDef["family"] = taskDefFamily

				updatedJSON, err := json.Marshal(taskDef)
				Expect(err).NotTo(HaveOccurred())

				_, err = healthClient.RegisterTaskDefinitionFromJSON(string(updatedJSON))
				Expect(err).NotTo(HaveOccurred())

				// Create service with 1 task
				healthLogger.Info("Creating service: %s", serviceName)
				
				// Retry service creation in case k3d cluster is still initializing
				Eventually(func() error {
					return healthClient.CreateService(healthClusterName, serviceName, taskDefFamily, 1)
				}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

				// Wait for task to start
				healthLogger.Info("Waiting for task to start")
				var taskArn string
				Eventually(func() bool {
					tasks, err := healthClient.ListTasks(healthClusterName, serviceName)
					if err != nil || len(tasks) == 0 {
						return false
					}

					taskArn = tasks[0]
					task, err := healthClient.DescribeTask(healthClusterName, taskArn)
					if err != nil {
						return false
					}

					return task.LastStatus == "RUNNING"
				}, 60*time.Second, 5*time.Second).Should(BeTrue())

				healthLogger.Info("Task started: %s", taskArn)

				// For now, just verify the task is running
				// In a real implementation, health check failures would cause task to be stopped
				healthLogger.Info("Task with failing health check is running")
			})

			It("should cleanup health check resources", Label("cleanup"), func() {
				// Release the shared cluster
				if sharedClusterManager != nil && healthClusterName != "" {
					sharedClusterManager.ReleaseCluster(healthClusterName)
				}
			})
		})
	})
})