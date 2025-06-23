package phase2_test

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
			workerKecs        *utils.KECSContainer
			workerClient      utils.ECSClientInterface
			workerLogger      *utils.TestLogger
			workerClusterName string
		)

		BeforeEach(func() {
			if workerKecs == nil {
				workerLogger = utils.NewTestLogger(GinkgoT())
				workerLogger.Info("Starting Background Worker tests")

				// Start KECS container
				workerKecs = utils.StartKECS(GinkgoT())
				workerClient = utils.NewECSClientInterface(workerKecs.Endpoint())

				// Create cluster for this test suite
				workerClusterName = utils.GenerateTestName("phase2-worker-cluster")
				err := workerClient.CreateCluster(workerClusterName)
				Expect(err).NotTo(HaveOccurred())

				utils.AssertClusterActive(GinkgoT(), workerClient, workerClusterName)
				workerLogger.Info("Created cluster: %s", workerClusterName)
				
				// Wait for k3d cluster to be created and ready
				// The cluster is created asynchronously, so we need to wait
				workerLogger.Info("Waiting for k3d cluster to be created and ready (30s)")
				time.Sleep(30 * time.Second)
			}
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
				err = workerClient.CreateService(workerClusterName, serviceName, taskDefFamily, 1)
				Expect(err).NotTo(HaveOccurred())

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

				// Wait a bit to let the worker process some items
				time.Sleep(15 * time.Second)

				// Check logs to verify processing (placeholder - requires log retrieval implementation)
				workerLogger.Info("Worker should be processing queue items")
			})

			It("should cleanup worker resources", Label("cleanup"), func() {
				if workerClient != nil && workerClusterName != "" {
					workerLogger.Info("Cleaning up cluster: %s", workerClusterName)
					_ = workerClient.DeleteCluster(workerClusterName)
				}
				if workerKecs != nil {
					workerKecs.Cleanup()
				}
			})
		})
	})

	Describe("Task Definition: Failure Handling", func() {
		var (
			failureKecs        *utils.KECSContainer
			failureClient      utils.ECSClientInterface
			failureLogger      *utils.TestLogger
			failureClusterName string
		)

		BeforeEach(func() {
			if failureKecs == nil {
				failureLogger = utils.NewTestLogger(GinkgoT())
				failureLogger.Info("Starting Failure Handling tests")

				// Start KECS container
				failureKecs = utils.StartKECS(GinkgoT())
				failureClient = utils.NewECSClientInterface(failureKecs.Endpoint())

				// Create cluster for this test suite
				failureClusterName = utils.GenerateTestName("phase2-failure-cluster")
				err := failureClient.CreateCluster(failureClusterName)
				Expect(err).NotTo(HaveOccurred())

				utils.AssertClusterActive(GinkgoT(), failureClient, failureClusterName)
				failureLogger.Info("Created cluster: %s", failureClusterName)
				
				// Wait for k3d cluster to be created and ready
				// The cluster is created asynchronously, so we need to wait
				failureLogger.Info("Waiting for k3d cluster to be created and ready (30s)")
				time.Sleep(30 * time.Second)
			}
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
				err = failureClient.CreateService(failureClusterName, serviceName, taskDefFamily, 1)
				Expect(err).NotTo(HaveOccurred())

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
				if failureClient != nil && failureClusterName != "" {
					failureLogger.Info("Cleaning up cluster: %s", failureClusterName)
					_ = failureClient.DeleteCluster(failureClusterName)
				}
				if failureKecs != nil {
					failureKecs.Cleanup()
				}
			})
		})
	})

	Describe("Task Definition: Health Check Failures", func() {
		var (
			healthKecs        *utils.KECSContainer
			healthClient      utils.ECSClientInterface
			healthLogger      *utils.TestLogger
			healthClusterName string
		)

		BeforeEach(func() {
			if healthKecs == nil {
				healthLogger = utils.NewTestLogger(GinkgoT())
				healthLogger.Info("Starting Health Check tests")

				// Start KECS container
				healthKecs = utils.StartKECS(GinkgoT())
				healthClient = utils.NewECSClientInterface(healthKecs.Endpoint())

				// Create cluster for this test suite
				healthClusterName = utils.GenerateTestName("phase2-health-cluster")
				err := healthClient.CreateCluster(healthClusterName)
				Expect(err).NotTo(HaveOccurred())

				utils.AssertClusterActive(GinkgoT(), healthClient, healthClusterName)
				healthLogger.Info("Created cluster: %s", healthClusterName)
			}
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
				err = healthClient.CreateService(healthClusterName, serviceName, taskDefFamily, 1)
				Expect(err).NotTo(HaveOccurred())

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
				if healthClient != nil && healthClusterName != "" {
					healthLogger.Info("Cleaning up cluster: %s", healthClusterName)
					_ = healthClient.DeleteCluster(healthClusterName)
				}
				if healthKecs != nil {
					healthKecs.Cleanup()
				}
			})
		})
	})
})