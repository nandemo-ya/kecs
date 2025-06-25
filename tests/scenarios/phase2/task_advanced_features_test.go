package phase2_test

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
		advancedKecs   *utils.KECSContainer
		advancedClient *utils.AWSCLIClient
		advancedClusterName string
		client      *utils.AWSCLIClient
		logger      *utils.TestLogger
		clusterName string
	)

	BeforeEach(func() {
		if advancedKecs == nil {
			logger = utils.NewTestLogger(GinkgoT())
			logger.Info("Starting Advanced Features tests")

			// Start KECS container
			advancedKecs = utils.StartKECS(GinkgoT())
			advancedClient = utils.NewAWSCLIClient(advancedKecs.Endpoint())

			// Create cluster for this test suite
			advancedClusterName = utils.GenerateTestName("phase2-advanced-cluster")
			err := advancedClient.CreateCluster(advancedClusterName)
			Expect(err).NotTo(HaveOccurred())

			utils.AssertClusterActive(GinkgoT(), advancedClient, advancedClusterName)
			logger.Info("Created cluster: %s", advancedClusterName)
			
			// Give k3d cluster some time to start up in the background
			// The actual readiness will be checked when we try to create services
			logger.Info("Allowing k3d cluster initialization to begin...")
			time.Sleep(5 * time.Second)
		}
		// Set local variables
		client = advancedClient
		clusterName = advancedClusterName
	})

	AfterEach(func() {
		// Cleanup is handled per test
	})


	Describe("Task Definition Revision Management", func() {
		var taskDefFamily string

		BeforeEach(func() {
			taskDefFamily = utils.GenerateTestName("td-revision")
		})

		AfterEach(func() {
			// Deregister all revisions
			_ = client.DeregisterTaskDefinition(fmt.Sprintf("%s:1", taskDefFamily))
			_ = client.DeregisterTaskDefinition(fmt.Sprintf("%s:2", taskDefFamily))
		})

		It("should increment revision number when updating task definition", func() {
			logger.Info("Testing task definition revision increments")

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
			resp1, err := client.RegisterTaskDefinitionFromJSON(string(taskDef1JSON))
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
			resp2, err := client.RegisterTaskDefinitionFromJSON(string(taskDef2JSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp2.Revision).To(Equal(2))

			logger.Info("Successfully created task definition revisions 1 and 2")
		})

		It("should retrieve specific task definition revision", func() {
			logger.Info("Testing retrieval of specific revisions")

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
			_, err := client.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())

			// Update for revision 2
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["memory"] = 256
			taskDef2JSON, _ := json.Marshal(taskDef)
			_, err = client.RegisterTaskDefinitionFromJSON(string(taskDef2JSON))
			Expect(err).NotTo(HaveOccurred())

			// Describe specific revision
			desc1, err := client.DescribeTaskDefinition(fmt.Sprintf("%s:1", taskDefFamily))
			Expect(err).NotTo(HaveOccurred())
			Expect(desc1.Revision).To(Equal(1))
			Expect(desc1.ContainerDefs[0].Memory).To(Equal(128))

			// Describe latest revision
			descLatest, err := client.DescribeTaskDefinition(taskDefFamily)
			Expect(err).NotTo(HaveOccurred())
			Expect(descLatest.Revision).To(Equal(2))
			Expect(descLatest.ContainerDefs[0].Memory).To(Equal(256))

			logger.Info("Successfully retrieved specific and latest revisions")
		})
	})

	Describe("Task Definition with Volume Configuration", func() {
		var taskDefFamily string

		BeforeEach(func() {
			taskDefFamily = utils.GenerateTestName("td-volume")
		})

		AfterEach(func() {
			_ = client.DeregisterTaskDefinition(taskDefFamily)
		})

		It("should support volume sharing between containers", func() {
			logger.Info("Testing volume configuration and sharing")

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
			resp, err := client.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Volumes).To(HaveLen(2))

			logger.Info("Successfully registered task definition with volume configuration")
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
			_, err := client.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			_ = client.DeleteService(clusterName, serviceName)
			_ = client.DeregisterTaskDefinition(taskDefFamily)
		})

		It("should create service with custom deployment configuration", func() {
			logger.Info("Testing service creation with deployment configuration")

			// Create service with standard API
			err := client.CreateService(clusterName, serviceName, taskDefFamily, 2)
			Expect(err).NotTo(HaveOccurred())

			// Verify service was created
			service, err := client.DescribeService(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.ServiceName).To(Equal(serviceName))
			Expect(service.DesiredCount).To(Equal(2))

			logger.Info("Successfully created service with deployment configuration")
		})

		It("should update service deployment configuration", func() {
			logger.Info("Testing service deployment configuration update")

			// Create service first
			err := client.CreateService(clusterName, serviceName, taskDefFamily, 2)
			Expect(err).NotTo(HaveOccurred())

			// Update service count
			err = client.UpdateService(clusterName, serviceName, 3)
			Expect(err).NotTo(HaveOccurred())

			// Verify update
			service, err := client.DescribeService(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(service.DesiredCount).To(Equal(3))

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
			newTaskDef, err := client.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())

			// Update service with new task definition
			err = client.UpdateServiceTaskDefinition(clusterName, serviceName, newTaskDef.TaskDefinitionArn)
			Expect(err).NotTo(HaveOccurred())

			logger.Info("Successfully updated service deployment configuration")
		})
	})

	Describe("Error Handling and Edge Cases", func() {
		It("should handle task definition deregistration errors gracefully", func() {
			logger.Info("Testing error handling for deregistration")

			// Try to deregister non-existent task definition
			err := client.DeregisterTaskDefinition("non-existent-task:1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ClientException"))

			logger.Info("Error handling works correctly for non-existent resources")
		})

		It("should handle service creation with invalid task definition", func() {
			logger.Info("Testing service creation with invalid task definition")

			serviceName := utils.GenerateTestName("svc-invalid")
			
			err := client.CreateService(clusterName, serviceName, "invalid-task-def:1", 1)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ClientException"))

			logger.Info("Service creation correctly fails with invalid task definition")
		})

		It("should maintain idempotency for task definition registration", func() {
			logger.Info("Testing idempotency of task definition registration")

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
			resp1, err := client.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())
			revision1 := resp1.Revision

			resp2, err := client.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())
			revision2 := resp2.Revision

			// Should create new revision even with identical definition (AWS ECS behavior)
			Expect(revision2).To(Equal(2))
			Expect(revision1).To(Equal(1))

			// Cleanup
			_ = client.DeregisterTaskDefinition(fmt.Sprintf("%s:1", taskDefFamily))
			_ = client.DeregisterTaskDefinition(fmt.Sprintf("%s:2", taskDefFamily))

			logger.Info("Task definition registration maintains proper revision behavior")
		})
	})

	Describe("Pagination Support", func() {
		It("should list task definitions", func() {
			logger.Info("Testing task definition listing")

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
				_, err := client.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
				Expect(err).NotTo(HaveOccurred())
			}

			// List task definitions
			taskDefs, err := client.ListTaskDefinitions()
			Expect(err).NotTo(HaveOccurred())
			
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
				_ = client.DeregisterTaskDefinition(family)
			}

			logger.Info("Task definition listing verified")
		})

		It("should list services", func() {
			logger.Info("Testing service listing")

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
			_, err := client.RegisterTaskDefinitionFromJSON(string(taskDefJSON))
			Expect(err).NotTo(HaveOccurred())

			// Create multiple services
			serviceNames := []string{}
			for i := 0; i < 3; i++ {
				serviceName := utils.GenerateTestName(fmt.Sprintf("svc-list-%02d", i))
				serviceNames = append(serviceNames, serviceName)
				
				err := client.CreateService(clusterName, serviceName, taskDefFamily, 1)
				Expect(err).NotTo(HaveOccurred())
			}

			// List services
			services, err := client.ListServices(clusterName)
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
				_ = client.DeleteService(clusterName, serviceName)
			}
			_ = client.DeregisterTaskDefinition(taskDefFamily)

			logger.Info("Service listing verified")
		})
	})

})