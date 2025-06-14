package cluster_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster List Operations", func() {
	var (
		kecs   *utils.KECSContainer
		client *utils.ECSClient
		logger *utils.TestLogger
	)

	BeforeEach(func() {
		// Start KECS container
		kecs = utils.StartKECS(GinkgoT())
		DeferCleanup(kecs.Cleanup)

		// Create ECS client
		client = utils.NewECSClient(kecs.Endpoint())
		logger = utils.NewTestLogger(GinkgoT())
	})

	Describe("Listing Clusters", func() {
		var clusterNames []string

		BeforeEach(func() {
			// Create test clusters
			clusterNames = []string{
				utils.GenerateTestName("list-cluster-1"),
				utils.GenerateTestName("list-cluster-2"),
				utils.GenerateTestName("list-cluster-3"),
			}

			// Ensure cleanup
			DeferCleanup(func() {
				for _, name := range clusterNames {
					utils.CleanupCluster(GinkgoT(), client, name)
				}
			})
		})

		Context("when no clusters have been created", func() {
			It("should return an empty list or only existing clusters", func() {
				// Initial list should be empty or contain only existing clusters
				clusters, err := client.ListClusters()
				Expect(err).NotTo(HaveOccurred(), "Failed to list clusters")

				initialCount := len(clusters)
				logger.Info("Initial cluster count: %d", initialCount)
			})
		})

		Context("when multiple clusters have been created", func() {
			BeforeEach(func() {
				// Create clusters
				for _, name := range clusterNames {
					logger.Info("Creating cluster: %s", name)
					err := client.CreateCluster(name)
					Expect(err).NotTo(HaveOccurred(), "Failed to create cluster %s", name)
				}
			})

			It("should list all created clusters", func() {
				// List clusters
				clusters, err := client.ListClusters()
				Expect(err).NotTo(HaveOccurred(), "Failed to list clusters")

				// Verify all created clusters are in the list
				// ListClusters returns ARNs, so we need to check for ARN format
				for _, expectedName := range clusterNames {
					// Convert cluster name to expected ARN format
					expectedArn := fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:cluster/%s", expectedName)
					Expect(clusters).To(ContainElement(expectedArn),
						"Cluster ARN for %s not found in list", expectedName)
				}

				logger.Info("Successfully listed %d clusters", len(clusters))
			})

			It("should update the list when a cluster is deleted", func() {
				// Delete one cluster
				deletedCluster := clusterNames[0]
				logger.Info("Deleting cluster: %s", deletedCluster)
				err := client.DeleteCluster(deletedCluster)
				Expect(err).NotTo(HaveOccurred(), "Failed to delete cluster")

				// List clusters again
				clusters, err := client.ListClusters()
				Expect(err).NotTo(HaveOccurred(), "Failed to list clusters")

				// Verify deleted cluster is not in the list
				deletedArn := fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:cluster/%s", deletedCluster)
				Expect(clusters).NotTo(ContainElement(deletedArn),
					"Deleted cluster ARN should not appear in list")

				// Verify remaining clusters are still in the list
				for i := 1; i < len(clusterNames); i++ {
					expectedArn := fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:cluster/%s", clusterNames[i])
					Expect(clusters).To(ContainElement(expectedArn),
						"Cluster ARN for %s not found in list", clusterNames[i])
				}
			})
		})
	})

	Describe("Cluster List Pagination", func() {
		PContext("when there are more clusters than the page size", func() {
			It("should support pagination", func() {
				// This test would verify pagination works correctly
				// when there are more clusters than the page size
			})
		})
	})

	Describe("Cluster List Consistency", func() {
		Context("when listing clusters multiple times", func() {
			It("should return consistent results", func() {
				// Create a set of clusters
				numClusters := 5
				clusterNames := make([]string, numClusters)
				for i := 0; i < numClusters; i++ {
					clusterNames[i] = utils.GenerateTestName(fmt.Sprintf("consistency-cluster-%d", i))
				}

				// Ensure cleanup
				DeferCleanup(func() {
					for _, name := range clusterNames {
						utils.CleanupCluster(GinkgoT(), client, name)
					}
				})

				// Create all clusters
				for _, name := range clusterNames {
					err := client.CreateCluster(name)
					Expect(err).NotTo(HaveOccurred(), "Failed to create cluster %s", name)
				}

				// List clusters multiple times to ensure consistency
				for i := 0; i < 3; i++ {
					logger.Info("List attempt %d", i+1)

					clusters, err := client.ListClusters()
					Expect(err).NotTo(HaveOccurred(), "Failed to list clusters on attempt %d", i+1)

					// Count our test clusters
					foundCount := 0
					for _, clusterArn := range clusters {
						for _, expectedName := range clusterNames {
							expectedArn := fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:cluster/%s", expectedName)
							if clusterArn == expectedArn {
								foundCount++
								break
							}
						}
					}

					Expect(foundCount).To(Equal(numClusters),
						"Inconsistent cluster count on attempt %d: expected %d, found %d",
						i+1, numClusters, foundCount)
				}
			})
		})
	})
})