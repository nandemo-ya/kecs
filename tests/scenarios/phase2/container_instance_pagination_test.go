package phase2_test

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Phase2: Container Instance and Attributes Pagination", func() {
	var (
		kecs   *utils.KECSContainer
		client *utils.ECSClient
	)

	BeforeEach(func() {
		// Start KECS container
		kecs = utils.StartKECS(GinkgoT())
		DeferCleanup(kecs.Cleanup)

		// Create ECS client
		client = utils.NewECSClient(kecs.Endpoint())
	})

	Describe("Container Instance Pagination", func() {
		var clusterName string

		BeforeEach(func() {
			clusterName = fmt.Sprintf("test-ci-pagination-%d", time.Now().Unix())
			err := client.CreateCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				err := client.DeleteCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when listing container instances with pagination", func() {
			It("should handle empty list correctly", func() {
				// List with pagination parameters
				cmd := []string{
					"aws", "ecs", "list-container-instances",
					"--cluster", clusterName,
					"--max-results", "5",
					"--endpoint-url", kecs.Endpoint(),
					"--region", "us-east-1",
				}

				output, err := kecs.RunCommand(cmd...)
				if err != nil || output == "" || len(output) < 10 {
					// Get container logs for debugging
					logs, _ := kecs.GetLogs()
					GinkgoWriter.Printf("Container logs:\n%s\n", logs)
					GinkgoWriter.Printf("Command output:\n%s\n", output)
					GinkgoWriter.Printf("Command error: %v\n", err)
				}
				Expect(err).NotTo(HaveOccurred())

				// Should return empty list since no instances are registered
				Expect(output).To(ContainSubstring("containerInstanceArns"))
				// AWS returns empty array without space: []
				Expect(output).To(MatchRegexp(`"containerInstanceArns"\s*:\s*\[\s*\]`))
			})

			It("should accept pagination parameters via direct API", func() {
				// Test direct API call
				payload := fmt.Sprintf(`{"cluster":"%s","maxResults":5}`, clusterName)
				
				cmd := []string{
					"curl", "-s", "-X", "POST",
					"-H", "Content-Type: application/x-amz-json-1.1",
					"-H", "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListContainerInstances",
					"-d", payload,
					kecs.APIEndpoint(),
				}

				output, err := kecs.RunCommand(cmd...)
				// Debug output
				GinkgoWriter.Printf("curl command: %v\n", cmd)
				GinkgoWriter.Printf("curl output: %s\n", output)
				GinkgoWriter.Printf("curl error: %v\n", err)
				
				// Get container logs on empty response
				if output == "{}" || output == "" || len(output) < 10 {
					logs, _ := kecs.GetLogs()
					GinkgoWriter.Printf("Container logs:\n%s\n", logs)
				}
				
				Expect(err).NotTo(HaveOccurred())

				// Should return valid JSON response
				Expect(output).To(ContainSubstring("containerInstanceArns"))
				Expect(strings.ToLower(output)).NotTo(ContainSubstring("error"))
			})

			It("should accept status filter", func() {
				// Test with status filter
				cmd := []string{
					"aws", "ecs", "list-container-instances",
					"--cluster", clusterName,
					"--status", "DRAINING",
					"--endpoint-url", kecs.Endpoint(),
					"--region", "us-east-1",
				}

				output, err := kecs.RunCommand(cmd...)
				Expect(err).NotTo(HaveOccurred())

				// Should return empty list since no DRAINING instances exist
				Expect(output).To(ContainSubstring("containerInstanceArns"))
				Expect(output).To(MatchRegexp(`"containerInstanceArns"\s*:\s*\[\s*\]`))
			})
		})
	})

	Describe("Attributes Pagination", func() {
		var clusterName string

		BeforeEach(func() {
			clusterName = fmt.Sprintf("test-attr-pagination-%d", time.Now().Unix())
			err := client.CreateCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				err := client.DeleteCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when listing attributes with pagination", func() {
			It("should handle empty list correctly", func() {
				// List with pagination parameters
				cmd := []string{
					"aws", "ecs", "list-attributes",
					"--cluster", clusterName,
					"--target-type", "container-instance",
					"--max-results", "5",
					"--endpoint-url", kecs.Endpoint(),
					"--region", "us-east-1",
				}

				output, err := kecs.RunCommand(cmd...)
				Expect(err).NotTo(HaveOccurred())

				// Should return empty list since no attributes exist
				Expect(output).To(ContainSubstring("attributes"))
				// AWS returns empty array
				Expect(output).To(MatchRegexp(`"attributes"\s*:\s*\[\s*\]`))
			})

			It("should accept pagination parameters via direct API", func() {
				// Test direct API call
				payload := fmt.Sprintf(`{"cluster":"%s","targetType":"container-instance","maxResults":5}`, clusterName)
				
				cmd := []string{
					"curl", "-s", "-X", "POST",
					"-H", "Content-Type: application/x-amz-json-1.1",
					"-H", "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListAttributes",
					"-d", payload,
					kecs.APIEndpoint(),
				}

				output, err := kecs.RunCommand(cmd...)
				Expect(err).NotTo(HaveOccurred())

				// Should return valid JSON response
				Expect(output).To(ContainSubstring("attributes"))
				Expect(strings.ToLower(output)).NotTo(ContainSubstring("error"))
			})

			It("should work without cluster parameter", func() {
				// AWS ECS allows listing attributes without specifying cluster
				cmd := []string{
					"aws", "ecs", "list-attributes",
					"--target-type", "container-instance",
					"--endpoint-url", kecs.Endpoint(),
					"--region", "us-east-1",
				}

				output, err := kecs.RunCommand(cmd...)
				Expect(err).NotTo(HaveOccurred())

				// Should return empty list
				Expect(output).To(ContainSubstring("attributes"))
				Expect(output).To(ContainSubstring("[]"))
			})
		})
	})
})