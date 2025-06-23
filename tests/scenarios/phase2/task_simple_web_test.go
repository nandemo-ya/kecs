package phase2_test

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

// Global variables for this test file
var (
	kecs        *utils.KECSContainer
	client      utils.ECSClientInterface
	logger      *utils.TestLogger
	clusterName string
)

var _ = BeforeSuite(func() {
	logger = utils.NewTestLogger(GinkgoT())
	logger.Info("Starting Simple Web Application tests")

	// Start KECS container
	kecs = utils.StartKECS(GinkgoT())
	client = utils.NewECSClientInterface(kecs.Endpoint())

	// Create cluster for this test file
	clusterName = utils.GenerateTestName("phase2-web-cluster")
	err := client.CreateCluster(clusterName)
	Expect(err).NotTo(HaveOccurred())

	utils.AssertClusterActive(GinkgoT(), client, clusterName)
	logger.Info("Created cluster: %s", clusterName)
	
	// Wait for k3d cluster to be created and ready
	// The cluster is created asynchronously, so we need to wait for it to be fully ready
	logger.Info("Waiting for k3d cluster to be created and ready...")
	
	// First wait a bit for async cluster creation to start
	time.Sleep(10 * time.Second)
	
	// Then actively check for cluster readiness by attempting to create a test service
	logger.Info("Checking k3d cluster readiness with test service creation...")
	testServiceName := utils.GenerateTestName("readiness-probe")
	testTaskDefFamily := utils.GenerateTestName("readiness-td")
	
	// Register a simple task definition for readiness check
	testTaskDefJSON := `{
		"family": "` + testTaskDefFamily + `",
		"containerDefinitions": [{
			"name": "test",
			"image": "busybox:latest",
			"cpu": 128,
			"memory": 128,
			"essential": true
		}]
	}`
	
	// First, try to delete any existing test service from previous runs
	_ = client.DeleteService(clusterName, testServiceName)
	
	// Wait for service deletion to complete
	Eventually(func() bool {
		services, _ := client.ListServices(clusterName)
		for _, svc := range services {
			if svc.ServiceName == testServiceName {
				return false // Service still exists
			}
		}
		return true // Service is gone
	}, 30*time.Second, 2*time.Second).Should(BeTrue(), "Failed to delete existing test service")
	
	Eventually(func() error {
		// Try to register task definition and create service
		_, err := client.RegisterTaskDefinitionFromJSON(testTaskDefJSON)
		if err != nil {
			return fmt.Errorf("failed to register test task definition: %w", err)
		}
		
		err = client.CreateService(clusterName, testServiceName, testTaskDefFamily, 1)
		if err != nil {
			// If it's still a duplicate key error, try deleting again
			if strings.Contains(err.Error(), "Duplicate key") {
				_ = client.DeleteService(clusterName, testServiceName)
				time.Sleep(5 * time.Second)
				// Don't return error, let it retry in next iteration
				return fmt.Errorf("service still exists, retrying: %w", err)
			}
			return fmt.Errorf("failed to create test service: %w", err)
		}
		
		// Success - clean up test resources
		_ = client.DeleteService(clusterName, testServiceName)
		_ = client.DeregisterTaskDefinition(testTaskDefFamily + ":1")
		return nil
	}, 120*time.Second, 10*time.Second).Should(Succeed(), "k3d cluster failed to become ready")
	
	logger.Info("k3d cluster is ready for service creation")
})

var _ = AfterSuite(func() {
	if client != nil && clusterName != "" {
		logger.Info("Cleaning up cluster: %s", clusterName)
		_ = client.DeleteCluster(clusterName)
	}
	if kecs != nil {
		kecs.Cleanup()
	}
})

var _ = Describe("Task Definition: Simple Web Application", Serial, func() {

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
			if serviceName != "" {
				logger.Info("Deleting service: %s", serviceName)
				_ = client.DeleteService(clusterName, serviceName)
				
				// Wait for tasks to stop
				Eventually(func() int {
					tasks, _ := client.ListTasks(clusterName, serviceName)
					return len(tasks)
				}, 60*time.Second, 5*time.Second).Should(Equal(0))
			}

			// Clean up task definition
			if taskDefFamily != "" {
				logger.Info("Deregistering task definition: %s", taskDefFamily)
				_ = client.DeregisterTaskDefinition(taskDefFamily)
			}
		})

		It("should register a simple nginx task definition", func() {
			logger.Info("Registering task definition: %s", taskDefFamily)

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

			registeredTaskDef, err := client.RegisterTaskDefinitionFromJSON(string(updatedJSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(registeredTaskDef.Family).To(Equal(taskDefFamily))
			Expect(registeredTaskDef.Revision).To(Equal(1))
			
			logger.Info("Successfully registered task definition: %s:%d", taskDefFamily, registeredTaskDef.Revision)
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

			_, err = client.RegisterTaskDefinitionFromJSON(string(updatedJSON))
			Expect(err).NotTo(HaveOccurred())

			// Create service with 2 desired tasks
			logger.Info("Creating service: %s with 2 tasks", serviceName)
			
			// Retry service creation in case k3d cluster is still initializing
			Eventually(func() error {
				return client.CreateService(clusterName, serviceName, taskDefFamily, 2)
			}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

			// Wait for tasks to be running
			logger.Info("Waiting for tasks to reach RUNNING state")
			Eventually(func() int {
				tasks, err := client.ListTasks(clusterName, serviceName)
				if err != nil {
					return 0
				}
				
				runningCount := 0
				for _, taskArn := range tasks {
					task, err := client.DescribeTask(clusterName, taskArn)
					if err == nil && task.LastStatus == "RUNNING" {
						runningCount++
					}
				}
				return runningCount
			}, 120*time.Second, 5*time.Second).Should(Equal(2))

			logger.Info("All tasks are running")
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

			_, err = client.RegisterTaskDefinitionFromJSON(string(updatedJSON))
			Expect(err).NotTo(HaveOccurred())

			// Create service with 1 task
			logger.Info("Creating service with 1 task")
			
			// Retry service creation in case k3d cluster is still initializing
			Eventually(func() error {
				return client.CreateService(clusterName, serviceName, taskDefFamily, 1)
			}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

			// Wait for initial task
			Eventually(func() int {
				tasks, _ := client.ListTasks(clusterName, serviceName)
				return len(tasks)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))

			// Scale up to 3 tasks
			logger.Info("Scaling service up to 3 tasks")
			err = client.UpdateService(clusterName, serviceName, 3)
			Expect(err).NotTo(HaveOccurred())

			// Wait for scale up
			Eventually(func() int {
				tasks, _ := client.ListTasks(clusterName, serviceName)
				return len(tasks)
			}, 60*time.Second, 5*time.Second).Should(Equal(3))

			// Scale down to 1 task
			logger.Info("Scaling service down to 1 task")
			err = client.UpdateService(clusterName, serviceName, 1)
			Expect(err).NotTo(HaveOccurred())

			// Wait for scale down
			Eventually(func() int {
				tasks, _ := client.ListTasks(clusterName, serviceName)
				return len(tasks)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))

			logger.Info("Scaling test completed successfully")
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

			taskDefV1, err := client.RegisterTaskDefinitionFromJSON(string(updatedJSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDefV1.Revision).To(Equal(1))

			// Create service
			logger.Info("Creating service with task definition v1")
			
			// Retry service creation in case k3d cluster is still initializing
			Eventually(func() error {
				return client.CreateService(clusterName, serviceName, taskDefFamily, 1)
			}, 60*time.Second, 5*time.Second).Should(Succeed(), "Failed to create service after retries")

			// Wait for initial deployment
			Eventually(func() int {
				tasks, _ := client.ListTasks(clusterName, serviceName)
				return len(tasks)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))

			// Update task definition (change nginx version)
			containers := taskDef["containerDefinitions"].([]interface{})
			container := containers[0].(map[string]interface{})
			container["image"] = "nginx:1.25-alpine"  // Update to specific version

			updatedJSON, err = json.Marshal(taskDef)
			Expect(err).NotTo(HaveOccurred())

			taskDefV2, err := client.RegisterTaskDefinitionFromJSON(string(updatedJSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDefV2.Revision).To(Equal(2))

			// Update service to use new task definition
			logger.Info("Updating service to use task definition v2")
			err = client.UpdateServiceTaskDefinition(clusterName, serviceName, fmt.Sprintf("%s:2", taskDefFamily))
			Expect(err).NotTo(HaveOccurred())

			// Verify new task is using v2
			Eventually(func() bool {
				tasks, err := client.ListTasks(clusterName, serviceName)
				if err != nil || len(tasks) == 0 {
					return false
				}
				
				task, err := client.DescribeTask(clusterName, tasks[0])
				if err != nil {
					return false
				}
				
				return strings.Contains(task.TaskDefinitionArn, ":2")
			}, 120*time.Second, 5*time.Second).Should(BeTrue())

			logger.Info("Successfully updated to new task definition version")
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