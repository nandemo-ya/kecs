package phase1_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster Pagination", Serial, func() {
	var (
		kecs   *utils.KECSContainer
		client utils.ECSClientInterface
		logger *utils.TestLogger
	)

	BeforeEach(func() {
		// Start KECS container
		kecs = utils.StartKECS(GinkgoT())
		DeferCleanup(kecs.Cleanup)

		// Create ECS client using AWS CLI
		client = utils.NewECSClientInterface(kecs.Endpoint(), utils.AWSCLIMode)
		logger = utils.NewTestLogger(GinkgoT())
	})

	Describe("List Operations with Pagination", func() {
		Context("when testing pagination with various page sizes", func() {
			var createdClusters []string

			BeforeEach(func() {
				// Create 15 clusters for pagination testing
				createdClusters = make([]string, 0, 15)
				for i := 0; i < 15; i++ {
					clusterName := fmt.Sprintf("page-test-%03d", i)
					err := client.CreateCluster(clusterName)
					Expect(err).NotTo(HaveOccurred())
					createdClusters = append(createdClusters, clusterName)
				}

				DeferCleanup(func() {
					// Clean up all created clusters
					for _, cluster := range createdClusters {
						_ = client.DeleteCluster(cluster)
					}
				})
			})

			It("should list clusters with maxResults=1", func() {
				logger.Info("Testing pagination with maxResults=1")

				// First page
				awsClient := client.(*utils.AWSCLIClient)
				clusters, nextToken, err := awsClient.ListClustersWithPagination(1, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(len(clusters)).To(Equal(1))
				Expect(nextToken).NotTo(BeEmpty())

				// Second page
				clusters2, nextToken2, err := awsClient.ListClustersWithPagination(1, nextToken)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(clusters2)).To(Equal(1))
				Expect(clusters2[0]).NotTo(Equal(clusters[0])) // Different cluster
				Expect(nextToken2).NotTo(BeEmpty())
			})

			It("should list clusters with maxResults=10", func() {
				logger.Info("Testing pagination with maxResults=10")

				awsClient := client.(*utils.AWSCLIClient)
				clusters, nextToken, err := awsClient.ListClustersWithPagination(10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(len(clusters)).To(BeNumerically("<=", 10))
				
				if len(createdClusters) > 10 {
					Expect(nextToken).NotTo(BeEmpty())
				}
			})

			It("should list clusters with maxResults=50", func() {
				logger.Info("Testing pagination with maxResults=50")

				awsClient := client.(*utils.AWSCLIClient)
				clusters, nextToken, err := awsClient.ListClustersWithPagination(50, "")
				Expect(err).NotTo(HaveOccurred())
				
				// Should return all 15 clusters
				Expect(len(clusters)).To(BeNumerically(">=", 15))
				Expect(nextToken).To(BeEmpty()) // No more pages
			})

			It("should list clusters with maxResults=100", func() {
				logger.Info("Testing pagination with maxResults=100")

				awsClient := client.(*utils.AWSCLIClient)
				clusters, nextToken, err := awsClient.ListClustersWithPagination(100, "")
				Expect(err).NotTo(HaveOccurred())
				
				// Should return all clusters
				Expect(len(clusters)).To(BeNumerically(">=", 15))
				Expect(nextToken).To(BeEmpty()) // No more pages
			})
		})

		Context("when testing next token handling", func() {
			var createdClusters []string

			BeforeEach(func() {
				// Create 25 clusters to ensure pagination
				createdClusters = make([]string, 0, 25)
				for i := 0; i < 25; i++ {
					clusterName := fmt.Sprintf("token-test-%03d", i)
					err := client.CreateCluster(clusterName)
					Expect(err).NotTo(HaveOccurred())
					createdClusters = append(createdClusters, clusterName)
				}

				DeferCleanup(func() {
					for _, cluster := range createdClusters {
						_ = client.DeleteCluster(cluster)
					}
				})
			})

			It("should handle pagination tokens correctly", func() {
				logger.Info("Testing pagination token flow")

				awsClient := client.(*utils.AWSCLIClient)
				allClusters := make(map[string]bool)
				pageCount := 0
				nextToken := ""

				// Paginate through all results with small page size
				for {
					clusters, newToken, err := awsClient.ListClustersWithPagination(5, nextToken)
					Expect(err).NotTo(HaveOccurred())
					
					pageCount++
					logger.Info("Page %d: Got %d clusters", pageCount, len(clusters))
					
					// Add clusters to our map
					for _, cluster := range clusters {
						allClusters[cluster] = true
					}
					
					if newToken == "" {
						break
					}
					
					// Ensure token changes
					if nextToken != "" {
						Expect(newToken).NotTo(Equal(nextToken))
					}
					nextToken = newToken
				}

				// Verify we got all our clusters
				foundCount := 0
				for _, clusterName := range createdClusters {
					for arn := range allClusters {
						if strings.Contains(arn, clusterName) {
							foundCount++
							break
						}
					}
				}
				Expect(foundCount).To(Equal(25), "Should find all 25 created clusters")
				Expect(pageCount).To(BeNumerically(">=", 5), "Should have at least 5 pages with maxResults=5")
			})

			It("should handle invalid next token gracefully", func() {
				logger.Info("Testing invalid next token")

				awsClient := client.(*utils.AWSCLIClient)
				// AWS typically returns an error for invalid tokens
				_, _, err := awsClient.ListClustersWithPagination(10, "invalid-token-12345")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when testing pagination consistency", func() {
			var createdClusters []string

			BeforeEach(func() {
				// Create exactly 20 clusters
				createdClusters = make([]string, 0, 20)
				for i := 0; i < 20; i++ {
					clusterName := fmt.Sprintf("consistency-test-%03d", i)
					err := client.CreateCluster(clusterName)
					Expect(err).NotTo(HaveOccurred())
					createdClusters = append(createdClusters, clusterName)
				}

				DeferCleanup(func() {
					for _, cluster := range createdClusters {
						_ = client.DeleteCluster(cluster)
					}
				})
			})

			It("should return consistent results across pages", func() {
				logger.Info("Testing pagination consistency")

				awsClient := client.(*utils.AWSCLIClient)
				
				// Get all clusters in one request
				allClusters, _, err := awsClient.ListClustersWithPagination(100, "")
				Expect(err).NotTo(HaveOccurred())
				
				// Now get them in smaller pages
				pagedClusters := make(map[string]bool)
				nextToken := ""
				
				for {
					clusters, newToken, err := awsClient.ListClustersWithPagination(7, nextToken)
					Expect(err).NotTo(HaveOccurred())
					
					for _, cluster := range clusters {
						// Check for duplicates
						Expect(pagedClusters[cluster]).To(BeFalse(), "Found duplicate cluster in pagination")
						pagedClusters[cluster] = true
					}
					
					if newToken == "" {
						break
					}
					nextToken = newToken
				}
				
				// Verify we got the same number of clusters
				Expect(len(pagedClusters)).To(Equal(len(allClusters)))
			})
		})
	})

	Describe("Large Scale Pagination Testing", func() {
		Context("when handling 150+ clusters", func() {
			var createdClusters []string
			const totalClusters = 150

			BeforeEach(func() {
				if !shouldRunLargeTests() {
					Skip("Skipping large scale test")
				}

				logger.Info("Creating %d clusters for large scale test", totalClusters)
				createdClusters = make([]string, 0, totalClusters)
				
				// Create clusters in batches for better performance
				for i := 0; i < totalClusters; i++ {
					clusterName := fmt.Sprintf("large-scale-%03d", i)
					err := client.CreateCluster(clusterName)
					Expect(err).NotTo(HaveOccurred())
					createdClusters = append(createdClusters, clusterName)
					
					if i%10 == 0 {
						logger.Info("Created %d/%d clusters", i, totalClusters)
					}
				}

				DeferCleanup(func() {
					logger.Info("Cleaning up %d clusters", len(createdClusters))
					for i, cluster := range createdClusters {
						_ = client.DeleteCluster(cluster)
						if i%10 == 0 {
							logger.Info("Deleted %d/%d clusters", i, len(createdClusters))
						}
					}
				})
			})

			It("should paginate through all 150 clusters correctly", func() {
				logger.Info("Testing pagination with 150 clusters")

				awsClient := client.(*utils.AWSCLIClient)
				allClusters := make(map[string]bool)
				pageCount := 0
				nextToken := ""
				
				// AWS ECS has a max of 100 results per page
				for {
					clusters, newToken, err := awsClient.ListClustersWithPagination(100, nextToken)
					Expect(err).NotTo(HaveOccurred())
					
					pageCount++
					logger.Info("Page %d: Got %d clusters", pageCount, len(clusters))
					
					// Verify page size constraints
					if newToken != "" {
						Expect(len(clusters)).To(Equal(100), "Should get exactly 100 clusters when more pages exist")
					} else {
						Expect(len(clusters)).To(BeNumerically("<=", 100), "Last page should have <= 100 clusters")
					}
					
					// Add clusters to our map
					for _, cluster := range clusters {
						Expect(allClusters[cluster]).To(BeFalse(), "Found duplicate cluster ARN")
						allClusters[cluster] = true
					}
					
					if newToken == "" {
						break
					}
					nextToken = newToken
				}
				
				// Verify we found all our clusters
				foundCount := 0
				for _, clusterName := range createdClusters {
					for arn := range allClusters {
						if strings.Contains(arn, clusterName) {
							foundCount++
							break
						}
					}
				}
				
				Expect(foundCount).To(Equal(totalClusters), fmt.Sprintf("Should find all %d created clusters", totalClusters))
				Expect(pageCount).To(Equal(2), "Should have exactly 2 pages for 150 clusters with maxResults=100")
			})

			It("should handle different page sizes correctly", func() {
				logger.Info("Testing various page sizes with 150 clusters")

				awsClient := client.(*utils.AWSCLIClient)
				
				// Test different page sizes
				pageSizes := []int{25, 50, 75, 100}
				
				for _, pageSize := range pageSizes {
					logger.Info("Testing with pageSize=%d", pageSize)
					
					clusterCount := 0
					pageCount := 0
					nextToken := ""
					seenClusters := make(map[string]bool)
					
					for {
						clusters, newToken, err := awsClient.ListClustersWithPagination(pageSize, nextToken)
						Expect(err).NotTo(HaveOccurred())
						
						pageCount++
						clusterCount += len(clusters)
						
						// Check for duplicates
						for _, cluster := range clusters {
							Expect(seenClusters[cluster]).To(BeFalse(), "Found duplicate with pageSize=%d", pageSize)
							seenClusters[cluster] = true
						}
						
						if newToken == "" {
							break
						}
						nextToken = newToken
					}
					
					// Verify we got all clusters
					Expect(clusterCount).To(BeNumerically(">=", totalClusters), "Should get at least %d clusters with pageSize=%d", totalClusters, pageSize)
					
					// Calculate expected pages (AWS caps at 100, so adjust expectation)
					effectivePageSize := pageSize
					if pageSize > 100 {
						effectivePageSize = 100
					}
					expectedPages := (totalClusters + effectivePageSize - 1) / effectivePageSize
					Expect(pageCount).To(BeNumerically(">=", expectedPages), "Should have at least %d pages with pageSize=%d", expectedPages, pageSize)
				}
			})
		})
	})
})

// Helper function to determine if large tests should run
func shouldRunLargeTests() bool {
	// Can be controlled by environment variable
	return GinkgoT().Name() == "Large Scale Pagination Testing when handling 150+ clusters should paginate through all 150 clusters correctly" ||
		GinkgoT().Name() == "Large Scale Pagination Testing when handling 150+ clusters should handle different page sizes correctly"
}