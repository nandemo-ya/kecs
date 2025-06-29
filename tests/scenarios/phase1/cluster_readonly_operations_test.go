package phase1

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster Read-Only Operations with Shared Clusters", Serial, func() {
	var (
		client      utils.ECSClientInterface
		logger      *utils.TestLogger
		clusterName string
	)

	BeforeEach(func() {
		// Use shared resources from suite
		client = sharedClient
		logger = sharedLogger

		// Get or create a shared cluster for read-only operations
		var err error
		clusterName, err = sharedClusterManager.GetOrCreateCluster("readonly-test")
		Expect(err).NotTo(HaveOccurred())
		
		logger.Info("Using shared cluster for read-only tests: %s", clusterName)
	})

	AfterEach(func() {
		// Release the cluster for other tests to use
		sharedClusterManager.ReleaseCluster(clusterName)
	})

	Describe("Describe Cluster Operations", func() {
		Context("when describing a cluster by name", func() {
			It("should return the cluster details", func() {
				logger.Info("Describing shared cluster by name: %s", clusterName)

				cluster, err := client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.ClusterName).To(Equal(clusterName))
				Expect(cluster.Status).To(Equal("ACTIVE"))
				Expect(cluster.ClusterArn).To(ContainSubstring(clusterName))
			})
		})

		Context("when describing a cluster by ARN", func() {
			It("should return the cluster details", func() {
				// First get the ARN
				cluster, err := client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
				arn := cluster.ClusterArn

				logger.Info("Describing shared cluster by ARN: %s", arn)

				// Now describe by ARN (using the same method)
				clusterByArn, err := client.DescribeCluster(arn)
				Expect(err).NotTo(HaveOccurred())
				Expect(clusterByArn.ClusterName).To(Equal(clusterName))
				Expect(clusterByArn.ClusterArn).To(Equal(arn))
			})
		})

		Context("when describing multiple clusters", func() {
			var secondClusterName string

			BeforeEach(func() {
				// Get another shared cluster
				var err error
				secondClusterName, err = sharedClusterManager.GetOrCreateCluster("readonly-test-2")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				sharedClusterManager.ReleaseCluster(secondClusterName)
			})

			It("should return details for both clusters", func() {
				logger.Info("Describing multiple clusters: %s, %s", clusterName, secondClusterName)

				// Describe each cluster individually
				cluster1, err := client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
				
				cluster2, err := client.DescribeCluster(secondClusterName)
				Expect(err).NotTo(HaveOccurred())

				// Verify both clusters exist and are active
				Expect(cluster1.ClusterName).To(Equal(clusterName))
				Expect(cluster2.ClusterName).To(Equal(secondClusterName))
				Expect(cluster1.Status).To(Equal("ACTIVE"))
				Expect(cluster2.Status).To(Equal("ACTIVE"))
			})
		})
	})

	Describe("List Operations", func() {
		Context("when listing clusters", func() {
			It("should include the shared cluster", func() {
				logger.Info("Listing clusters, expecting to find: %s", clusterName)

				// Retry logic to handle eventual consistency
				var clusters []string
				var err error
				Eventually(func() bool {
					clusters, err = client.ListClusters()
					if err != nil {
						logger.Info("Error listing clusters (will retry): %v", err)
						return false
					}
					
					// Check if our cluster is in the list
					for _, arn := range clusters {
						if strings.Contains(arn, clusterName) {
							logger.Info("Found cluster %s in list of %d clusters", clusterName, len(clusters))
							return true
						}
					}
					
					logger.Info("Cluster %s not found yet in list of %d clusters", clusterName, len(clusters))
					return false
				}, 10*time.Second, 500*time.Millisecond).Should(BeTrue(), 
					"Shared cluster %s should eventually appear in cluster list", clusterName)
			})
		})

		Context("when listing services in the cluster", func() {
			It("should list services (empty for new cluster)", func() {
				logger.Info("Listing services in shared cluster: %s", clusterName)

				services, err := client.ListServices(clusterName)
				Expect(err).NotTo(HaveOccurred())
				// New cluster should have no services
				Expect(services).To(BeEmpty())
			})
		})
	})

	Describe("Cluster Status Operations", func() {
		Context("when checking cluster attributes", func() {
			It("should return empty attributes for new cluster", func() {
				logger.Info("Checking attributes for shared cluster: %s", clusterName)

				// This is a hypothetical operation - adjust based on actual API
				cluster, err := client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
				
				// Verify default state
				Expect(cluster.Status).To(Equal("ACTIVE"))
				Expect(cluster.ActiveServicesCount).To(Equal(0))
				Expect(cluster.RunningTasksCount).To(Equal(0))
			})
		})
	})
})