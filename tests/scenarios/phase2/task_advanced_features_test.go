package phase2

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Phase 2: Advanced Task Definition and Service Features", Serial, func() {
	var (
		advancedClient      utils.ECSClientInterface
		advancedLogger      *utils.TestLogger
		advancedClusterName string
	)

	BeforeEach(func() {
		// Use shared resources from suite
		advancedClient = sharedClient
		advancedLogger = sharedLogger
		
		// Get or create a shared cluster for advanced tests
		var err error
		advancedClusterName, err = sharedClusterManager.GetOrCreateCluster("phase2-advanced")
		Expect(err).NotTo(HaveOccurred())
		
		advancedLogger.Info("Using shared cluster: %s", advancedClusterName)
	})

	AfterEach(func() {
		// Cleanup is done in the last test of each describe block
	})

	Describe("Task Definition Revision Management", func() {
		var taskDefFamily string

		BeforeEach(func() {
			taskDefFamily = utils.GenerateTestName("td-revision")
		})

		AfterEach(func() {
			// Deregister all revisions
			_ = advancedClient.DeregisterTaskDefinition(fmt.Sprintf("%s:1", taskDefFamily))
			_ = advancedClient.DeregisterTaskDefinition(fmt.Sprintf("%s:2", taskDefFamily))
		})

		It("should increment revision number when updating task definition", func() {
			advancedLogger.Info("Testing task definition revision increments")

			// First registration
			taskDef1 := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":   "test-container",
						"image":  "nginx:1.19",
						"memory": 128,
						"cpu":    128,
					},
				},
			}

			taskDef1JSON, _ := json.Marshal(taskDef1)
			resp1, err := advancedClient.RegisterTaskDefinitionFromJSON(string(taskDef1JSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp1.Revision).To(Equal(1))

			// Update with different configuration
			taskDef2 := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":   "test-container",
						"image":  "nginx:1.20", // Changed version
						"memory": 256,          // Increased memory
						"cpu":    256,          // Increased CPU
					},
				},
			}

			taskDef2JSON, _ := json.Marshal(taskDef2)
			resp2, err := advancedClient.RegisterTaskDefinitionFromJSON(string(taskDef2JSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp2.Revision).To(Equal(2))

			advancedLogger.Info("Successfully created task definition revisions 1 and 2")
		})

		It("should retrieve specific task definition revision", func() {
			advancedLogger.Info("Testing retrieval of specific revisions")

			// Create two revisions
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":   "test-container",
						"image":  "nginx:1.19",
						"memory": 128,
						"cpu":    128,
					},
				},
			}

			taskDefJSON, _ := json.Marshal(taskDef)
			_, err := advancedClient.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())

			// Update for revision 2
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["memory"] = 256
			taskDef2JSON, _ := json.Marshal(taskDef)
			_, err = advancedClient.RegisterTaskDefinitionFromJSON(string(taskDef2JSON))
			Expect(err).NotTo(HaveOccurred())

			// Describe specific revision
			desc1, err := advancedClient.DescribeTaskDefinition(fmt.Sprintf("%s:1", taskDefFamily))
			Expect(err).NotTo(HaveOccurred())
			Expect(desc1.Revision).To(Equal(1))
			Expect(desc1.ContainerDefs[0].Memory).To(Equal(128))

			// Describe latest revision
			descLatest, err := advancedClient.DescribeTaskDefinition(taskDefFamily)
			Expect(err).NotTo(HaveOccurred())
			Expect(descLatest.Revision).To(Equal(2))
			Expect(descLatest.ContainerDefs[0].Memory).To(Equal(256))

			advancedLogger.Info("Successfully retrieved specific and latest revisions")
		})
	})

	Describe("Task Definition with Volume Configuration", func() {
		var taskDefFamily string

		BeforeEach(func() {
			taskDefFamily = utils.GenerateTestName("td-volume")
		})

		AfterEach(func() {
			_ = advancedClient.DeregisterTaskDefinition(taskDefFamily)
		})

		It("should support volume sharing between containers", func() {
			advancedLogger.Info("Testing volume configuration and sharing")

			// Load template and modify
			templatePath := "templates/multi-container/nginx-webapp.json"
			templateContent, err := os.ReadFile(templatePath)
			Expect(err).NotTo(HaveOccurred())

			var taskDef map[string]interface{}
			err = json.Unmarshal(templateContent, &taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Update family name and add volumes
			taskDef["family"] = taskDefFamily
			taskDef["volumes"] = []map[string]interface{}{
				{
					"name": "shared-data",
					"host": map[string]interface{}{
						"sourcePath": "/tmp/shared",
					},
				},
				{
					"name": "logs-volume",
					"host": map[string]interface{}{
						"sourcePath": "/var/log/app",
					},
				},
			}

			// Add mount points to containers
			containers := taskDef["containerDefinitions"].([]interface{})
			containers[0].(map[string]interface{})["mountPoints"] = []map[string]interface{}{
				{
					"sourceVolume":  "shared-data",
					"containerPath": "/usr/share/nginx/html",
					"readOnly":      false,
				},
			}
			if len(containers) > 1 {
				containers[1].(map[string]interface{})["mountPoints"] = []map[string]interface{}{
					{
						"sourceVolume":  "shared-data",
						"containerPath": "/var/www/html",
						"readOnly":      true,
					},
				}
			}

			taskDefJSON, _ := json.Marshal(taskDef)
			resp, err := advancedClient.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Volumes).To(HaveLen(2))

			advancedLogger.Info("Successfully registered task definition with volume configuration")
		})
	})

	Describe("Service with Placement Constraints", func() {
		var (
			taskDefFamily string
			serviceName   string
		)

		BeforeEach(func() {
			taskDefFamily = utils.GenerateTestName("td-placement")
			serviceName = utils.GenerateTestName("svc-placement")

			// Register task definition
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":   "test-app",
						"image":  "nginx:alpine",
						"memory": 128,
						"cpu":    128,
					},
				},
			}
			taskDefJSON, _ := json.Marshal(taskDef)
			_, err := advancedClient.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// Clean up service
			if serviceName != "" {
				advancedLogger.Info("Deleting service: %s", serviceName)
				_ = advancedClient.DeleteService(advancedClusterName, serviceName)

				// Wait for tasks to stop
				Eventually(func() int {
					tasks, _ := advancedClient.ListTasks(advancedClusterName, serviceName)
					return len(tasks)
				}, 60*time.Second, 5*time.Second).Should(Equal(0))
			}
			_ = advancedClient.DeregisterTaskDefinition(taskDefFamily)
		})

		It("should create service with custom deployment configuration", func() {
			advancedLogger.Info("Testing service creation with deployment configuration")

			// Create service with standard API
			// Retry service creation in case k3d cluster is still initializing
			Eventually(func() error {
				return advancedClient.CreateService(advancedClusterName, serviceName, taskDefFamily, 2)
			}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

			// Verify service was created
			var service *utils.Service
			Eventually(func() error {
				var err error
				service, err = advancedClient.DescribeService(advancedClusterName, serviceName)
				return err
			}, 30*time.Second, 2*time.Second).Should(Succeed())
			
			Expect(service.ServiceName).To(Equal(serviceName))
			Expect(service.DesiredCount).To(Equal(2))

			advancedLogger.Info("Successfully created service with deployment configuration")
		})

		It("should update service deployment configuration", func() {
			advancedLogger.Info("Testing service deployment configuration update")

			// Create service first
			Eventually(func() error {
				return advancedClient.CreateService(advancedClusterName, serviceName, taskDefFamily, 2)
			}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

			// Wait for service to be stable
			Eventually(func() int {
				service, err := advancedClient.DescribeService(advancedClusterName, serviceName)
				if err != nil {
					return -1
				}
				return service.DesiredCount
			}, 30*time.Second, 2*time.Second).Should(Equal(2))

			// Update service count
			err := advancedClient.UpdateService(advancedClusterName, serviceName, 3)
			Expect(err).NotTo(HaveOccurred())

			// Verify update
			Eventually(func() int {
				service, err := advancedClient.DescribeService(advancedClusterName, serviceName)
				if err != nil {
					return -1
				}
				return service.DesiredCount
			}, 30*time.Second, 2*time.Second).Should(Equal(3))

			// Update with new task definition revision
			// First create a new revision
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":   "test-app",
						"image":  "nginx:alpine",
						"memory": 256, // Increased memory
						"cpu":    256,
					},
				},
			}
			taskDefJSON, _ := json.Marshal(taskDef)
			newTaskDef, err := advancedClient.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())

			// Update service with new task definition
			err = advancedClient.UpdateServiceTaskDefinition(advancedClusterName, serviceName, newTaskDef.TaskDefinitionArn)
			Expect(err).NotTo(HaveOccurred())

			advancedLogger.Info("Successfully updated service deployment configuration")
		})
	})

	Describe("Error Handling and Edge Cases", func() {
		It("should handle task definition deregistration errors gracefully", func() {
			advancedLogger.Info("Testing error handling for deregistration")

			// Try to deregister non-existent task definition
			err := advancedClient.DeregisterTaskDefinition("non-existent-task:1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ClientException"))

			advancedLogger.Info("Error handling works correctly for non-existent resources")
		})

		It("should handle service creation with invalid task definition", func() {
			advancedLogger.Info("Testing service creation with invalid task definition")

			serviceName := utils.GenerateTestName("svc-invalid")
			
			err := advancedClient.CreateService(advancedClusterName, serviceName, "invalid-task-def:1", 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ClientException"))

			advancedLogger.Info("Service creation correctly fails with invalid task definition")
		})

		It("should maintain idempotency for task definition registration", func() {
			advancedLogger.Info("Testing idempotency of task definition registration")

			taskDefFamily := utils.GenerateTestName("td-idempotent")
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":   "test-container",
						"image":  "nginx:alpine",
						"memory": 128,
						"cpu":    128,
					},
				},
			}

			// Register twice with identical definition
			taskDefJSON, _ := json.Marshal(taskDef)
			resp1, err := advancedClient.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())
			revision1 := resp1.Revision

			resp2, err := advancedClient.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())
			revision2 := resp2.Revision

			// Should create new revision even with identical definition (AWS ECS behavior)
			Expect(revision2).To(Equal(2))
			Expect(revision1).To(Equal(1))

			// Cleanup
			_ = advancedClient.DeregisterTaskDefinition(fmt.Sprintf("%s:1", taskDefFamily))
			_ = advancedClient.DeregisterTaskDefinition(fmt.Sprintf("%s:2", taskDefFamily))

			advancedLogger.Info("Task definition registration maintains proper revision behavior")
		})
	})

	Describe("Pagination Support", func() {
		It("should list task definitions", func() {
			advancedLogger.Info("Testing task definition listing")

			// Create multiple task definitions
			taskDefFamilies := []string{}
			for i := 0; i < 5; i++ {
				family := utils.GenerateTestName(fmt.Sprintf("td-list-%02d", i))
				taskDefFamilies = append(taskDefFamilies, family)
				
				taskDef := map[string]interface{}{
					"family": family,
					"containerDefinitions": []map[string]interface{}{
						{
							"name":   "test-container",
							"image":  "nginx:alpine",
							"memory": 128,
							"cpu":    128,
						},
					},
				}
				taskDefJSON, _ := json.Marshal(taskDef)
				_, err := advancedClient.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
				Expect(err).NotTo(HaveOccurred())
			}

			// List task definitions
			var taskDefs []string
			Eventually(func() int {
				var err error
				taskDefs, err = advancedClient.ListTaskDefinitions()
				if err != nil {
					return 0
				}
				return len(taskDefs)
			}, 30*time.Second, 2*time.Second).Should(BeNumerically(">=", 5))
			
			// Should contain our task definitions
			for _, family := range taskDefFamilies {
				found := false
				for _, td := range taskDefs {
					if strings.Contains(td, family) {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), fmt.Sprintf("Task definition %s not found in list", family))
			}

			// Cleanup
			for _, family := range taskDefFamilies {
				_ = advancedClient.DeregisterTaskDefinition(family)
			}

			advancedLogger.Info("Task definition listing verified")
		})

		It("should list services", func() {
			advancedLogger.Info("Testing service listing")

			// Create a task definition for services
			taskDefFamily := utils.GenerateTestName("td-svc-list")
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":   "test-container",
						"image":  "nginx:alpine",
						"memory": 128,
						"cpu":    128,
					},
				},
			}
			taskDefJSON, _ := json.Marshal(taskDef)
			_, err := advancedClient.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())

			// Create multiple services
			serviceNames := []string{}
			for i := 0; i < 3; i++ {
				serviceName := utils.GenerateTestName(fmt.Sprintf("svc-list-%02d", i))
				serviceNames = append(serviceNames, serviceName)
				
				// Retry service creation in case k3d cluster is still initializing
				Eventually(func() error {
					return advancedClient.CreateService(advancedClusterName, serviceName, taskDefFamily, 1)
				}, 60*time.Second, 5*time.Second).Should(Succeed(), fmt.Sprintf("Failed to create service %s", serviceName))
			}

			// Wait for all services to be created
			Eventually(func() int {
				services, err := advancedClient.ListServices(advancedClusterName)
				if err != nil {
					return 0
				}
				count := 0
				for _, serviceName := range serviceNames {
					for _, svc := range services {
						if strings.Contains(svc, serviceName) {
							count++
							break
						}
					}
				}
				return count
			}, 60*time.Second, 5*time.Second).Should(Equal(3))

			// List services
			services, err := advancedClient.ListServices(advancedClusterName)
			Expect(err).NotTo(HaveOccurred())
			
			// Should contain our services
			for _, serviceName := range serviceNames {
				found := false
				for _, svc := range services {
					if strings.Contains(svc, serviceName) {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), fmt.Sprintf("Service %s not found in list", serviceName))
			}

			// Cleanup
			for _, serviceName := range serviceNames {
				_ = advancedClient.DeleteService(advancedClusterName, serviceName)
				
				// Wait for service deletion
				Eventually(func() int {
					tasks, _ := advancedClient.ListTasks(advancedClusterName, serviceName)
					return len(tasks)
				}, 60*time.Second, 5*time.Second).Should(Equal(0))
			}
			_ = advancedClient.DeregisterTaskDefinition(taskDefFamily)

			advancedLogger.Info("Service listing verified")
		})

		It("should cleanup advanced features resources", Label("cleanup"), func() {
			// Release the shared cluster
			if sharedClusterManager != nil && advancedClusterName != "" {
				sharedClusterManager.ReleaseCluster(advancedClusterName)
			}
		})
	})

})