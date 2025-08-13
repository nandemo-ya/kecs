package phase2

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Task Definition: Simple Web Application", Serial, func() {
	var (
		webClient      utils.ECSClientInterface
		webLogger      *utils.TestLogger
		webClusterName string
	)

	BeforeEach(func() {
		// Use shared resources from suite
		webClient = sharedClient
		webLogger = sharedLogger
		
		// Get or create a shared cluster for web tests
		var err error
		webClusterName, err = sharedClusterManager.GetOrCreateCluster("phase2-web")
		Expect(err).NotTo(HaveOccurred())
		
		webLogger.Info("Using shared cluster: %s", webClusterName)
	})

	AfterEach(func() {
		// Cleanup is done in the last test
	})

	Describe("Nginx Web Server Deployment", func() {
		var (
			taskDefFamily string
			serviceName   string
		)

		BeforeEach(func() {
			taskDefFamily = utils.GenerateTestName("simple-web-td")
			serviceName = utils.GenerateTestName("simple-web-svc")
		})

		AfterEach(func() {
			// Clean up service
			if serviceName != "" && webClient != nil {
				webLogger.Info("Deleting service: %s", serviceName)
				_ = webClient.DeleteService(webClusterName, serviceName)
				
				// Wait for tasks to stop
				Eventually(func() int {
					tasks, _ := webClient.ListTasks(webClusterName, serviceName)
					return len(tasks)
				}, 60*time.Second, 5*time.Second).Should(Equal(0))
			}

			// Clean up task definition
			if taskDefFamily != "" && webClient != nil {
				webLogger.Info("Deregistering task definition: %s", taskDefFamily)
				_ = webClient.DeregisterTaskDefinition(taskDefFamily)
			}
		})

		It("should register a simple nginx task definition", func() {
			webLogger.Info("Registering task definition: %s", taskDefFamily)

			// Load task definition template
			taskDefJSON, err := os.ReadFile("templates/single-container/simple-web.json")
			Expect(err).NotTo(HaveOccurred())

			// Update family name in the template
			var taskDef map[string]interface{}
			err = json.Unmarshal(taskDefJSON, &taskDef)
			Expect(err).NotTo(HaveOccurred())
			taskDef["family"] = taskDefFamily

			// Register task definition
			updatedJSON, err := json.Marshal(taskDef)
			Expect(err).NotTo(HaveOccurred())

			registeredTaskDef, err := webClient.RegisterTaskDefinitionFromJSON(string(updatedJSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(registeredTaskDef.Family).To(Equal(taskDefFamily))
			Expect(registeredTaskDef.Revision).To(Equal(1))
			
			webLogger.Info("Successfully registered task definition: %s:%d", taskDefFamily, registeredTaskDef.Revision)
		})

		It("should create a service and run nginx containers", func() {
			// First register the task definition
			taskDefJSON, err := os.ReadFile("templates/single-container/simple-web.json")
			Expect(err).NotTo(HaveOccurred())

			var taskDef map[string]interface{}
			err = json.Unmarshal(taskDefJSON, &taskDef)
			Expect(err).NotTo(HaveOccurred())
			taskDef["family"] = taskDefFamily

			updatedJSON, err := json.Marshal(taskDef)
			Expect(err).NotTo(HaveOccurred())

			_, err = webClient.RegisterTaskDefinitionFromJSON(string(updatedJSON))
			Expect(err).NotTo(HaveOccurred())

			// Create service with 2 desired tasks
			webLogger.Info("Creating service: %s with 2 tasks", serviceName)
			
			// Retry service creation in case k3d cluster is still initializing
			Eventually(func() error {
				return webClient.CreateService(webClusterName, serviceName, taskDefFamily, 2)
			}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

			// Wait for tasks to be running
			webLogger.Info("Waiting for tasks to reach RUNNING state")
			Eventually(func() int {
				tasks, err := webClient.ListTasks(webClusterName, serviceName)
				if err != nil {
					return 0
				}
				
				runningCount := 0
				for _, taskArn := range tasks {
					task, err := webClient.DescribeTask(webClusterName, taskArn)
					if err == nil && task.LastStatus == "RUNNING" {
						runningCount++
					}
				}
				return runningCount
			}, 120*time.Second, 5*time.Second).Should(Equal(2))

			webLogger.Info("All tasks are running")
		})

		It("should serve HTTP requests from nginx containers", func() {
			Skip("HTTP connectivity tests require task IP discovery - implement after task networking")
			
			// This test would:
			// 1. Get task IPs
			// 2. Make HTTP requests to nginx
			// 3. Verify response
		})

		It("should handle task scaling", func() {
			// Register task definition
			taskDefJSON, err := os.ReadFile("templates/single-container/simple-web.json")
			Expect(err).NotTo(HaveOccurred())

			var taskDef map[string]interface{}
			err = json.Unmarshal(taskDefJSON, &taskDef)
			Expect(err).NotTo(HaveOccurred())
			taskDef["family"] = taskDefFamily

			updatedJSON, err := json.Marshal(taskDef)
			Expect(err).NotTo(HaveOccurred())

			_, err = webClient.RegisterTaskDefinitionFromJSON(string(updatedJSON))
			Expect(err).NotTo(HaveOccurred())

			// Create service with 1 task
			webLogger.Info("Creating service with 1 task")
			
			// Retry service creation in case k3d cluster is still initializing
			Eventually(func() error {
				return webClient.CreateService(webClusterName, serviceName, taskDefFamily, 1)
			}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

			// Wait for initial task
			Eventually(func() int {
				tasks, _ := webClient.ListTasks(webClusterName, serviceName)
				return len(tasks)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))

			// Scale up to 3 tasks
			webLogger.Info("Scaling service up to 3 tasks")
			err = webClient.UpdateService(webClusterName, serviceName, 3)
			Expect(err).NotTo(HaveOccurred())

			// Wait for scale up
			Eventually(func() int {
				tasks, _ := webClient.ListTasks(webClusterName, serviceName)
				return len(tasks)
			}, 60*time.Second, 5*time.Second).Should(Equal(3))

			// Scale down to 1 task
			webLogger.Info("Scaling service down to 1 task")
			err = webClient.UpdateService(webClusterName, serviceName, 1)
			Expect(err).NotTo(HaveOccurred())

			// Wait for scale down
			Eventually(func() int {
				tasks, _ := webClient.ListTasks(webClusterName, serviceName)
				return len(tasks)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))

			webLogger.Info("Scaling test completed successfully")
		})

		It("should update task definition and deploy new version", func() {
			// Register initial task definition
			taskDefJSON, err := os.ReadFile("templates/single-container/simple-web.json")
			Expect(err).NotTo(HaveOccurred())

			var taskDef map[string]interface{}
			err = json.Unmarshal(taskDefJSON, &taskDef)
			Expect(err).NotTo(HaveOccurred())
			taskDef["family"] = taskDefFamily

			updatedJSON, err := json.Marshal(taskDef)
			Expect(err).NotTo(HaveOccurred())

			taskDefV1, err := webClient.RegisterTaskDefinitionFromJSON(string(updatedJSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDefV1.Revision).To(Equal(1))

			// Create service
			webLogger.Info("Creating service with task definition v1")
			
			// Retry service creation in case k3d cluster is still initializing
			Eventually(func() error {
				return webClient.CreateService(webClusterName, serviceName, taskDefFamily, 1)
			}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

			// Wait for initial deployment
			Eventually(func() int {
				tasks, _ := webClient.ListTasks(webClusterName, serviceName)
				return len(tasks)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))

			// Update task definition (change nginx version)
			containers := taskDef["containerDefinitions"].([]interface{})
			container := containers[0].(map[string]interface{})
			container["image"] = "nginx:1.25-alpine"  // Update to specific version

			updatedJSON, err = json.Marshal(taskDef)
			Expect(err).NotTo(HaveOccurred())

			taskDefV2, err := webClient.RegisterTaskDefinitionFromJSON(string(updatedJSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDefV2.Revision).To(Equal(2))

			// Update service to use new task definition
			webLogger.Info("Updating service to use task definition v2")
			err = webClient.UpdateServiceTaskDefinition(webClusterName, serviceName, fmt.Sprintf("%s:2", taskDefFamily))
			Expect(err).NotTo(HaveOccurred())

			// Verify new task is using v2
			Eventually(func() bool {
				tasks, err := webClient.ListTasks(webClusterName, serviceName)
				if err != nil || len(tasks) == 0 {
					return false
				}
				
				task, err := webClient.DescribeTask(webClusterName, tasks[0])
				if err != nil {
					return false
				}
				
				return strings.Contains(task.TaskDefinitionArn, ":2")
			}, 120*time.Second, 5*time.Second).Should(BeTrue())

			webLogger.Info("Successfully updated to new task definition version")
		})

		It("should cleanup web resources", Label("cleanup"), func() {
			// Release the shared cluster
			if sharedClusterManager != nil && webClusterName != "" {
				sharedClusterManager.ReleaseCluster(webClusterName)
			}
		})
	})
})

// Helper function to make HTTP request (placeholder for future implementation)
func makeHTTPRequest(url string) (*http.Response, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	return client.Get(url)
}

// Helper function to read response body
func readResponseBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}