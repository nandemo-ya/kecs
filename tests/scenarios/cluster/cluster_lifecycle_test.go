package cluster_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster Lifecycle", func() {
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

	Describe("Creating and Deleting Clusters", func() {
		Context("when creating a new cluster", func() {
			var clusterName string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("test-cluster")
				DeferCleanup(func() {
					utils.CleanupCluster(GinkgoT(), client, clusterName)
				})
			})

			It("should create the cluster successfully", func() {
				logger.Info("Creating cluster: %s", clusterName)

				// Create cluster
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to create cluster")

				// Verify cluster is created and active
				utils.AssertClusterActive(GinkgoT(), client, clusterName)

				// Describe cluster to get details
				cluster, err := client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to describe cluster")

				Expect(cluster.ClusterName).To(Equal(clusterName))
				Expect(cluster.Status).To(Equal("ACTIVE"))
				Expect(cluster.RegisteredContainerInstancesCount).To(Equal(0))
				Expect(cluster.RunningTasksCount).To(Equal(0))
				Expect(cluster.ActiveServicesCount).To(Equal(0))

				logger.Info("Cluster created successfully: %s", cluster.ClusterArn)
			})

			It("should delete the cluster successfully", func() {
				// First create a cluster
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
				utils.AssertClusterActive(GinkgoT(), client, clusterName)

				logger.Info("Deleting cluster: %s", clusterName)

				// Delete cluster
				err = client.DeleteCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to delete cluster")

				// Verify cluster is deleted
				utils.AssertClusterDeleted(GinkgoT(), client, clusterName)

				logger.Info("Cluster deleted successfully")
			})
		})
	})

	Describe("Duplicate Cluster Creation", func() {
		Context("when attempting to create a duplicate cluster", func() {
			var clusterName string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("duplicate-cluster")
				DeferCleanup(func() {
					utils.CleanupCluster(GinkgoT(), client, clusterName)
				})
			})

			It("should be idempotent and not return an error", func() {
				// Create first cluster
				logger.Info("Creating first cluster: %s", clusterName)
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to create first cluster")

				// Try to create duplicate cluster
				logger.Info("Attempting to create duplicate cluster: %s", clusterName)
				err = client.CreateCluster(clusterName)

				// AWS ECS behavior: CreateCluster is idempotent, should succeed
				Expect(err).NotTo(HaveOccurred(), "Creating duplicate cluster should be idempotent")

				// Verify only one cluster exists
				cluster, err := client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.ClusterName).To(Equal(clusterName))
			})
		})
	})

	Describe("Non-existent Cluster Operations", func() {
		Context("when describing a non-existent cluster", func() {
			It("should return a not found error", func() {
				// Try to describe non-existent cluster
				nonExistentCluster := "non-existent-cluster-12345"
				logger.Info("Attempting to describe non-existent cluster: %s", nonExistentCluster)

				_, err := client.DescribeCluster(nonExistentCluster)
				Expect(err).To(HaveOccurred(), "Expected error for non-existent cluster")
				Expect(err.Error()).To(ContainSubstring("not found"), "Error should indicate cluster not found")
			})
		})
	})
})