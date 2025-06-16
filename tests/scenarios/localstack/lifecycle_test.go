package localstack_test

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/tests/scenarios/localstack/helpers"
	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("LocalStack Lifecycle", func() {
	var (
		kecs   *utils.KECSContainer
		client utils.ECSClientInterface
		testClusterName string
	)

	BeforeEach(func() {
		// Start KECS with LocalStack enabled
		kecs = utils.StartKECS(GinkgoT())
		DeferCleanup(func() {
			if kecs != nil {
				kecs.Cleanup()
			}
		})

		// Create ECS client
		client = utils.NewECSClient(kecs.Endpoint())
		
		// Create a test cluster
		testClusterName = fmt.Sprintf("test-localstack-%d", time.Now().Unix())
		err := client.CreateCluster(testClusterName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Clean up cluster
		if client != nil && testClusterName != "" {
			client.DeleteCluster(testClusterName)
		}
	})

	Describe("Starting and Stopping LocalStack", func() {
		It("should start LocalStack successfully", func() {
			// Start LocalStack with default services
			helpers.StartLocalStack(GinkgoT(), kecs, nil)

			// Wait for LocalStack to be ready
			helpers.WaitForLocalStackReady(GinkgoT(), client, testClusterName, 30*time.Second)

			// Check status
			status := helpers.GetLocalStackStatus(GinkgoT(), kecs)
			Expect(status).To(ContainSubstring("Running: true"))
			Expect(status).To(ContainSubstring("Healthy: true"))
		})

		It("should stop LocalStack successfully", func() {
			// Start LocalStack first
			helpers.StartLocalStack(GinkgoT(), kecs, nil)
			helpers.WaitForLocalStackReady(GinkgoT(), client, testClusterName, 30*time.Second)

			// Stop LocalStack
			helpers.StopLocalStack(GinkgoT(), kecs)

			// Give it a moment to stop
			time.Sleep(2 * time.Second)

			// Check status
			status := helpers.GetLocalStackStatus(GinkgoT(), kecs)
			Expect(status).To(ContainSubstring("Running: false"))
		})

		It("should restart LocalStack successfully", func() {
			// Start LocalStack
			helpers.StartLocalStack(GinkgoT(), kecs, nil)
			helpers.WaitForLocalStackReady(GinkgoT(), client, testClusterName, 30*time.Second)

			// Get initial status
			initialStatus := helpers.GetLocalStackStatus(GinkgoT(), kecs)
			Expect(initialStatus).To(ContainSubstring("Running: true"))

			// Restart LocalStack
			helpers.RestartLocalStack(GinkgoT(), kecs)

			// Wait for it to be ready again
			helpers.WaitForLocalStackReady(GinkgoT(), client, testClusterName, 30*time.Second)

			// Check status after restart
			status := helpers.GetLocalStackStatus(GinkgoT(), kecs)
			Expect(status).To(ContainSubstring("Running: true"))
			Expect(status).To(ContainSubstring("Healthy: true"))
		})
	})

	Describe("Service Management", func() {
		BeforeEach(func() {
			// Start LocalStack with minimal services
			helpers.StartLocalStack(GinkgoT(), kecs, []string{"iam"})
			helpers.WaitForLocalStackReady(GinkgoT(), client, testClusterName, 30*time.Second)
		})

		It("should enable additional services", func() {
			// Enable S3 service
			helpers.EnableLocalStackService(GinkgoT(), kecs, "s3")

			// Give it a moment to initialize
			time.Sleep(5 * time.Second)

			// Check that S3 is now enabled
			status := helpers.GetLocalStackStatus(GinkgoT(), kecs)
			Expect(status).To(ContainSubstring("s3"))
		})

		It("should disable services", func() {
			// First enable S3
			helpers.EnableLocalStackService(GinkgoT(), kecs, "s3")
			time.Sleep(5 * time.Second)

			// Now disable it
			helpers.DisableLocalStackService(GinkgoT(), kecs, "s3")
			time.Sleep(2 * time.Second)

			// Check that S3 is no longer in the enabled services
			status := helpers.GetLocalStackStatus(GinkgoT(), kecs)
			lines := strings.Split(status, "\n")
			
			// Find the enabled services section
			inServicesSection := false
			s3Found := false
			for _, line := range lines {
				if strings.Contains(line, "Enabled Services:") {
					inServicesSection = true
					continue
				}
				if inServicesSection && strings.TrimSpace(line) == "" {
					break
				}
				if inServicesSection && strings.Contains(line, "s3") {
					s3Found = true
					break
				}
			}
			
			Expect(s3Found).To(BeFalse(), "S3 should not be in enabled services")
		})

		It("should list available services", func() {
			output, err := kecs.ExecuteCommand("localstack", "services")
			Expect(err).NotTo(HaveOccurred())
			
			// Check that common services are listed
			Expect(output).To(ContainSubstring("iam"))
			Expect(output).To(ContainSubstring("s3"))
			Expect(output).To(ContainSubstring("dynamodb"))
			Expect(output).To(ContainSubstring("cloudwatchlogs"))
		})
	})

	Describe("Persistence", func() {
		It("should maintain data after restart", func() {
			Skip("Persistence test requires LocalStack Pro or specific configuration")
			
			// This test would:
			// 1. Start LocalStack with persistence enabled
			// 2. Create some resources (e.g., S3 bucket)
			// 3. Restart LocalStack
			// 4. Verify resources still exist
		})
	})

	Describe("Health Monitoring", func() {
		It("should report unhealthy state when LocalStack is not running", func() {
			// Don't start LocalStack
			status := helpers.GetLocalStackStatus(GinkgoT(), kecs)
			Expect(status).To(ContainSubstring("Running: false"))
		})

		It("should report service-level health", func() {
			// Start LocalStack with multiple services
			helpers.StartLocalStack(GinkgoT(), kecs, []string{"iam", "s3", "dynamodb"})
			helpers.WaitForLocalStackReady(GinkgoT(), client, testClusterName, 30*time.Second)

			// Get detailed status
			status := helpers.GetLocalStackStatus(GinkgoT(), kecs)
			
			// Check for service health section
			Expect(status).To(ContainSubstring("Service Health:"))
			Expect(status).To(ContainSubstring("iam"))
			Expect(status).To(ContainSubstring("s3"))
			Expect(status).To(ContainSubstring("dynamodb"))
		})
	})
})