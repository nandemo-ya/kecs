package three_tier_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Three Tier Application Integration", func() {
	var (
		kecs       *utils.KECSContainer
		ecsClient  utils.ECSClientInterface
		clusterName string
	)

	BeforeEach(func() {
		// Start KECS
		kecs = utils.StartKECS(GinkgoT())

		// Create ECS client
		ecsClient = utils.NewCurlClient(kecs.Endpoint())

		// Create cluster
		clusterName = fmt.Sprintf("three-tier-cluster-%d", time.Now().Unix())
		err := ecsClient.CreateCluster(clusterName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if kecs != nil {
			// Cleanup services and tasks
			services, _ := ecsClient.ListServices(clusterName)
			for _, serviceName := range services {
				ecsClient.DeleteService(clusterName, serviceName)
			}

			// Delete cluster
			if clusterName != "" {
				ecsClient.DeleteCluster(clusterName)
			}

			// Stop KECS
			kecs.Cleanup()
		}
	})

	Describe("Full Stack Deployment", func() {
		var (
			dbTaskDefARN       string
			backendTaskDefARN  string
			frontendTaskDefARN string
			albDNS             string
		)

		BeforeEach(func() {
			// Build Docker images
			buildDockerImages()

			// Setup LocalStack resources
			setupLocalStackResources(kecs)

			// Register task definitions
			var err error
			var td *utils.TaskDefinition
			td, err = registerTaskDefinition(ecsClient, "database-task.json")
			Expect(err).NotTo(HaveOccurred())
			dbTaskDefARN = td.Family + ":" + fmt.Sprintf("%d", td.Revision)

			td, err = registerTaskDefinition(ecsClient, "backend-task.json")
			Expect(err).NotTo(HaveOccurred())
			backendTaskDefARN = td.Family + ":" + fmt.Sprintf("%d", td.Revision)

			td, err = registerTaskDefinition(ecsClient, "frontend-task.json")
			Expect(err).NotTo(HaveOccurred())
			frontendTaskDefARN = td.Family + ":" + fmt.Sprintf("%d", td.Revision)
		})

		It("should deploy and integrate all three tiers", func() {
			By("Creating database service with service discovery")
			createServiceWithDiscovery(ecsClient, clusterName, "database-service", dbTaskDefARN, map[string]string{
				"postgres": "5432",
				"redis":    "6379",
			})

			By("Waiting for database to be healthy")
			Eventually(func() bool {
				return isServiceHealthy(ecsClient, clusterName, "database-service")
			}, 2*time.Minute, 10*time.Second).Should(BeTrue())

			By("Creating backend service with service discovery")
			createServiceWithDiscovery(ecsClient, clusterName, "backend-service", backendTaskDefARN, map[string]string{
				"backend": "3000",
			})

			By("Creating frontend service with load balancer")
			_, albDNS = createServiceWithALB(ecsClient, clusterName, "frontend-service", frontendTaskDefARN)

			By("Waiting for all services to be healthy")
			Eventually(func() bool {
				return isServiceHealthy(ecsClient, clusterName, "backend-service") &&
					isServiceHealthy(ecsClient, clusterName, "frontend-service")
			}, 3*time.Minute, 10*time.Second).Should(BeTrue())

			By("Testing frontend health check")
			Eventually(func() int {
				resp, err := http.Get(fmt.Sprintf("http://%s/health", albDNS))
				if err != nil {
					return 0
				}
				defer resp.Body.Close()
				return resp.StatusCode
			}, 1*time.Minute, 5*time.Second).Should(Equal(200))

			By("Testing backend API through frontend proxy")
			Eventually(func() bool {
				resp, err := http.Get(fmt.Sprintf("http://%s/api/health", albDNS))
				if err != nil {
					return false
				}
				defer resp.Body.Close()

				var health map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&health)
				
				return health["status"] == "healthy" &&
					health["database"] == "connected" &&
					health["cache"] == "connected"
			}, 2*time.Minute, 5*time.Second).Should(BeTrue())

			By("Testing data persistence through cache")
			// Add a user
			addUserResponse := addUser(albDNS, "Test User", "test@example.com")
			Expect(addUserResponse).To(Equal(201))

			// First load should be from database
			users1 := loadUsers(albDNS)
			Expect(users1["source"]).To(Equal("database"))
			Expect(users1["data"]).To(HaveLen(1))

			// Second load should be from cache
			users2 := loadUsers(albDNS)
			Expect(users2["source"]).To(Equal("cache"))
			Expect(users2["data"]).To(HaveLen(1))

			By("Testing S3 integration")
			// Upload test file to S3
			uploadTestFile(kecs, "test-bucket", "test-file.txt", "Hello from KECS!")

			// List files through API
			files := listS3Files(albDNS)
			Expect(files["files"]).To(HaveLen(1))

			By("Testing service discovery")
			services := checkServices(albDNS)
			Expect(services["services"]).To(HaveKey("backend"))
			Expect(services["services"]).To(HaveKey("database"))
			Expect(services["services"]).To(HaveKey("cache"))
		})

		It("should handle rolling updates", func() {
			// Deploy initial version
			deployFullStack()

			By("Updating backend with new version")
			// Modify backend task definition with new image tag
			updatedBackendTaskDef := updateTaskDefinitionVersion(backendTaskDefARN, "v2")
			
			// Update service
			err := ecsClient.UpdateService(clusterName, "backend-service", nil, updatedBackendTaskDef)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying zero-downtime deployment")
			// Monitor health during update
			healthCheckPassed := true
			done := make(chan bool)
			
			go func() {
				for {
					select {
					case <-done:
						return
					default:
						resp, err := http.Get(fmt.Sprintf("http://%s/api/health", albDNS))
						if err != nil || resp.StatusCode != 200 {
							healthCheckPassed = false
						}
						time.Sleep(1 * time.Second)
					}
				}
			}()

			// Wait for update to complete
			Eventually(func() bool {
				return isServiceUpdated(ecsClient, clusterName, "backend-service", updatedBackendTaskDef)
			}, 5*time.Minute, 10*time.Second).Should(BeTrue())

			close(done)
			Expect(healthCheckPassed).To(BeTrue(), "Health check failed during rolling update")
		})

		It("should scale services under load", func() {
			deployFullStack()

			By("Scaling backend service to 3 instances")
			desiredCount := 3
			err := ecsClient.UpdateService(clusterName, "backend-service", &desiredCount, "")
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for scale-out to complete")
			Eventually(func() int {
				return getRunningTaskCount(ecsClient, clusterName, "backend-service")
			}, 2*time.Minute, 10*time.Second).Should(Equal(3))

			By("Verifying load distribution")
			// Collect backend hostnames from multiple requests
			hostnames := make(map[string]int)
			for i := 0; i < 30; i++ {
				services := checkServices(albDNS)
				if backend, ok := services["services"].(map[string]interface{})["backend"].(map[string]interface{}); ok {
					if hostname, ok := backend["host"].(string); ok {
						hostnames[hostname]++
					}
				}
				time.Sleep(100 * time.Millisecond)
			}

			// Should have hit multiple backend instances
			Expect(len(hostnames)).To(BeNumerically(">=", 2), "Load not distributed across instances")
		})

		It("should recover from failures", func() {
			deployFullStack()

			By("Simulating database failure")
			// Stop database tasks
			stopServiceTasks(ecsClient, clusterName, "database-service")

			By("Verifying backend reports unhealthy")
			Eventually(func() string {
				resp, err := http.Get(fmt.Sprintf("http://%s/api/health", albDNS))
				if err != nil {
					return "error"
				}
				defer resp.Body.Close()

				var health map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&health)
				return health["status"].(string)
			}, 1*time.Minute, 5*time.Second).Should(Equal("unhealthy"))

			By("Waiting for ECS to restart database")
			Eventually(func() bool {
				return isServiceHealthy(ecsClient, clusterName, "database-service")
			}, 3*time.Minute, 10*time.Second).Should(BeTrue())

			By("Verifying system recovers")
			Eventually(func() string {
				resp, err := http.Get(fmt.Sprintf("http://%s/api/health", albDNS))
				if err != nil {
					return "error"
				}
				defer resp.Body.Close()

				var health map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&health)
				return health["status"].(string)
			}, 2*time.Minute, 5*time.Second).Should(Equal("healthy"))
		})
	})
})

// Helper functions

func buildDockerImages() {
	GinkgoWriter.Printf("Building Docker images...\n")
	
	// The test is running from tests/ directory, so backend and frontend are at ../
	baseDir := filepath.Join("..")
	
	// Build backend image
	cmd := exec.Command("docker", "build", "-t", "three-tier-backend:latest", filepath.Join(baseDir, "backend"))
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "Failed to build backend image: %s", string(output))
	
	// Build frontend image
	cmd = exec.Command("docker", "build", "-t", "three-tier-frontend:latest", filepath.Join(baseDir, "frontend"))
	output, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "Failed to build frontend image: %s", string(output))
}

func setupLocalStackResources(kecs *utils.KECSContainer) {
	GinkgoWriter.Printf("Setting up LocalStack resources...\n")
	// In real implementation, this would use LocalStack endpoint
	// For now, we'll skip this as LocalStack integration is separate
}

func registerTaskDefinition(client utils.ECSClientInterface, filename string) (*utils.TaskDefinition, error) {
	// The test is running from tests/ directory, so task-definitions is at ../task-definitions/
	taskDefPath := filepath.Join("..", "task-definitions", filename)
	taskDefJSON, err := os.ReadFile(taskDefPath)
	if err != nil {
		return nil, err
	}
	
	return client.RegisterTaskDefinition("three-tier", string(taskDefJSON))
}

func createServiceWithDiscovery(client utils.ECSClientInterface, clusterName, serviceName, taskDefArn string, servicePorts map[string]string) string {
	// This would integrate with the service discovery implementation
	// For now, we'll create a regular service
	err := client.CreateService(clusterName, serviceName, taskDefArn, 1)
	Expect(err).NotTo(HaveOccurred())
	return serviceName
}

func createServiceWithALB(client utils.ECSClientInterface, clusterName, serviceName, taskDefArn string) (string, string) {
	// This would integrate with the ALB implementation
	// For now, we'll create a regular service and return a mock ALB DNS
	err := client.CreateService(clusterName, serviceName, taskDefArn, 1)
	Expect(err).NotTo(HaveOccurred())
	
	// In real implementation, this would return the actual ALB DNS
	albDNS := "localhost:8080"
	
	return serviceName, albDNS
}

func isServiceHealthy(client utils.ECSClientInterface, clusterName, serviceName string) bool {
	service, err := client.DescribeService(clusterName, serviceName)
	if err != nil {
		return false
	}
	
	return service.RunningCount == service.DesiredCount && service.DesiredCount > 0
}

func deployFullStack() {
	// Helper to deploy all services
	// Implementation would reuse the deployment logic from the main test
}

func addUser(albDNS, name, email string) int {
	payload := fmt.Sprintf(`{"name":"%s","email":"%s"}`, name, email)
	resp, err := http.Post(
		fmt.Sprintf("http://%s/api/users", albDNS),
		"application/json",
		strings.NewReader(payload),
	)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func loadUsers(albDNS string) map[string]interface{} {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/users", albDNS))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func listS3Files(albDNS string) map[string]interface{} {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/files", albDNS))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func checkServices(albDNS string) map[string]interface{} {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/services", albDNS))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result
}

func uploadTestFile(kecs *utils.KECSContainer, bucket, key, content string) {
	// In real implementation, this would use LocalStack endpoint
	// For now, we'll skip this as LocalStack integration is separate
	GinkgoWriter.Printf("Would upload file %s to bucket %s\n", key, bucket)
}

func updateTaskDefinitionVersion(taskDefArn, version string) string {
	// In real implementation, this would update the task definition
	// with a new image tag and return the new ARN
	return taskDefArn + ":2"
}

func isServiceUpdated(client utils.ECSClientInterface, clusterName, serviceName, newTaskDefArn string) bool {
	service, err := client.DescribeService(clusterName, serviceName)
	if err != nil {
		return false
	}
	
	return service.TaskDefinition == newTaskDefArn && 
		service.RunningCount == service.DesiredCount
}

func getRunningTaskCount(client utils.ECSClientInterface, clusterName, serviceName string) int {
	service, err := client.DescribeService(clusterName, serviceName)
	if err != nil {
		return 0
	}
	return service.RunningCount
}

func stopServiceTasks(client utils.ECSClientInterface, clusterName, serviceName string) {
	tasks, err := client.ListTasks(clusterName, serviceName)
	if err != nil {
		return
	}
	
	for _, taskArn := range tasks {
		client.StopTask(clusterName, taskArn, "Simulating failure")
	}
}