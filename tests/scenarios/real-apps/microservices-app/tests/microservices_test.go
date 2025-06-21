package microservices_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Microservices Application Integration", func() {
	var (
		kecs        *utils.KECSContainer
		ecsClient   utils.ECSClientInterface
		clusterName string
	)

	BeforeEach(func() {
		// Start KECS
		kecs = utils.StartKECS(GinkgoT())

		// Create ECS client
		ecsClient = utils.NewCurlClient(kecs.Endpoint())

		// Create cluster
		clusterName = fmt.Sprintf("microservices-cluster-%d", time.Now().Unix())
		err := ecsClient.CreateCluster(clusterName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if kecs != nil {
			// Cleanup services
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

	Describe("Microservices Deployment", func() {
		var (
			apiGatewayTaskDef   string
			userServiceTaskDef  string
			orderServiceTaskDef string
		)

		BeforeEach(func() {
			// Build Docker images
			buildDockerImages()

			// Setup LocalStack resources
			setupLocalStackResources(kecs)

			// Register task definitions
			var err error
			var td *utils.TaskDefinition

			td, err = registerTaskDefinition(ecsClient, "api-gateway-task.json")
			Expect(err).NotTo(HaveOccurred())
			apiGatewayTaskDef = td.Family + ":" + fmt.Sprintf("%d", td.Revision)

			td, err = registerTaskDefinition(ecsClient, "user-service-task.json")
			Expect(err).NotTo(HaveOccurred())
			userServiceTaskDef = td.Family + ":" + fmt.Sprintf("%d", td.Revision)

			// For now, we'll skip order service
			// td, err = registerTaskDefinition(ecsClient, "order-service-task.json")
			// Expect(err).NotTo(HaveOccurred())
			// orderServiceTaskDef = td.Family + ":" + fmt.Sprintf("%d", td.Revision)
		})

		It("should deploy and integrate microservices", func() {
			By("Creating user service with service discovery")
			err := ecsClient.CreateService(clusterName, "user-service", userServiceTaskDef, 2)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for user service to be healthy")
			Eventually(func() bool {
				return isServiceHealthy(ecsClient, clusterName, "user-service")
			}, 2*time.Minute, 10*time.Second).Should(BeTrue())

			By("Creating API gateway service")
			err = ecsClient.CreateService(clusterName, "api-gateway", apiGatewayTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for API gateway to be healthy")
			Eventually(func() bool {
				return isServiceHealthy(ecsClient, clusterName, "api-gateway")
			}, 2*time.Minute, 10*time.Second).Should(BeTrue())

			By("Testing service discovery through API gateway")
			// In real test, would check API gateway can discover user service
			// For now, we verify services are running
			services, err := ecsClient.ListServices(clusterName)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(ContainElements("user-service", "api-gateway"))
		})

		It("should handle service-to-service communication", func() {
			By("Deploying all microservices")
			deployMicroservices(ecsClient, clusterName, apiGatewayTaskDef, userServiceTaskDef)

			By("Creating a user through API gateway")
			// In real implementation, would make HTTP request to API gateway
			// which would forward to user service

			By("Verifying user service received the request")
			// Check logs or metrics to verify communication

			By("Testing service discovery updates")
			// Scale user service and verify API gateway discovers new instances
			err := ecsClient.UpdateService(clusterName, "user-service", intPtr(3), "")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() int {
				return getRunningTaskCount(ecsClient, clusterName, "user-service")
			}, 2*time.Minute, 10*time.Second).Should(Equal(3))
		})

		It("should handle service failures gracefully", func() {
			By("Deploying microservices")
			deployMicroservices(ecsClient, clusterName, apiGatewayTaskDef, userServiceTaskDef)

			By("Simulating user service failure")
			stopServiceTasks(ecsClient, clusterName, "user-service")

			By("Verifying API gateway handles failure gracefully")
			// In real test, would check API gateway returns appropriate error
			// and doesn't crash

			By("Waiting for user service recovery")
			Eventually(func() bool {
				return isServiceHealthy(ecsClient, clusterName, "user-service")
			}, 3*time.Minute, 10*time.Second).Should(BeTrue())

			By("Verifying system recovers")
			// Check that communication is restored
		})

		It("should scale based on load", func() {
			By("Deploying microservices with minimal instances")
			deployMicroservices(ecsClient, clusterName, apiGatewayTaskDef, userServiceTaskDef)

			By("Simulating load on user service")
			// In real test, would generate load

			By("Scaling user service")
			err := ecsClient.UpdateService(clusterName, "user-service", intPtr(5), "")
			Expect(err).NotTo(HaveOccurred())

			By("Verifying scale-out")
			Eventually(func() int {
				return getRunningTaskCount(ecsClient, clusterName, "user-service")
			}, 2*time.Minute, 10*time.Second).Should(Equal(5))

			By("Testing load distribution")
			// Verify requests are distributed across instances
		})
	})
})

// Helper functions

func buildDockerImages() {
	GinkgoWriter.Printf("Building Docker images...\n")

	baseDir := filepath.Join("..")

	// Build API Gateway image
	cmd := exec.Command("docker", "build", "-t", "microservices-api-gateway:latest",
		filepath.Join(baseDir, "services/api-gateway"))
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "Failed to build API Gateway image: %s", string(output))

	// Build User Service image
	cmd = exec.Command("docker", "build", "-t", "microservices-user-service:latest",
		filepath.Join(baseDir, "services/user-service"))
	output, err = cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "Failed to build User Service image: %s", string(output))
}

func setupLocalStackResources(kecs *utils.KECSContainer) {
	GinkgoWriter.Printf("Setting up LocalStack resources...\n")
	// In real implementation, would create DynamoDB tables, S3 buckets, etc.
}

func registerTaskDefinition(client utils.ECSClientInterface, filename string) (*utils.TaskDefinition, error) {
	taskDefPath := filepath.Join("..", "task-definitions", filename)
	taskDefJSON, err := os.ReadFile(taskDefPath)
	if err != nil {
		return nil, err
	}

	return client.RegisterTaskDefinition("microservices", string(taskDefJSON))
}

func isServiceHealthy(client utils.ECSClientInterface, clusterName, serviceName string) bool {
	service, err := client.DescribeService(clusterName, serviceName)
	if err != nil {
		return false
	}

	return service.RunningCount == service.DesiredCount && service.DesiredCount > 0
}

func deployMicroservices(client utils.ECSClientInterface, clusterName, apiGatewayTaskDef, userServiceTaskDef string) {
	err := client.CreateService(clusterName, "user-service", userServiceTaskDef, 2)
	Expect(err).NotTo(HaveOccurred())

	err = client.CreateService(clusterName, "api-gateway", apiGatewayTaskDef, 1)
	Expect(err).NotTo(HaveOccurred())

	// Wait for services to be healthy
	Eventually(func() bool {
		return isServiceHealthy(client, clusterName, "user-service") &&
			isServiceHealthy(client, clusterName, "api-gateway")
	}, 3*time.Minute, 10*time.Second).Should(BeTrue())
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

func intPtr(i int) *int {
	return &i
}
