package cluster_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster Operations with Generated Types", func() {
	var (
		kecs   *utils.KECSContainer
		client utils.ECSClientInterface
		logger *utils.TestLogger
	)

	BeforeEach(func() {
		// Start KECS container
		kecs = utils.StartKECS(GinkgoT())
		DeferCleanup(kecs.Cleanup)

		// Create ECS client using generated types
		client = utils.NewECSClientInterface(kecs.Endpoint(), utils.GeneratedMode)
		logger = utils.NewTestLogger(GinkgoT())
	})

	Describe("Cluster CRUD Operations", func() {
		Context("when using generated types", func() {
			var clusterName string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("gen-cluster")
				DeferCleanup(func() {
					// Clean up cluster if it exists
					_ = client.DeleteCluster(clusterName)
				})
			})

			It("should create and describe cluster with generated types", func() {
				logger.Info("Creating cluster with generated types: %s", clusterName)

				// Create cluster
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to create cluster")

				// Describe cluster
				cluster, err := client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to describe cluster")

				// Verify cluster properties
				Expect(cluster.ClusterName).To(Equal(clusterName))
				Expect(cluster.Status).To(Equal("ACTIVE"))
				Expect(cluster.ClusterArn).To(ContainSubstring("arn:aws:ecs:"))
				Expect(cluster.ClusterArn).To(ContainSubstring("cluster/" + clusterName))

				// Initial counts should be zero
				Expect(cluster.RegisteredContainerInstancesCount).To(Equal(0))
				Expect(cluster.RunningTasksCount).To(Equal(0))
				Expect(cluster.PendingTasksCount).To(Equal(0))
				Expect(cluster.ActiveServicesCount).To(Equal(0))
			})

			It("should list clusters with generated types", func() {
				// Create cluster first
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())

				// List clusters
				clusterArns, err := client.ListClusters()
				Expect(err).NotTo(HaveOccurred())
				Expect(clusterArns).To(ContainElement(ContainSubstring(clusterName)))
			})

			It("should delete cluster with generated types", func() {
				// Create cluster first
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())

				// Delete cluster
				err = client.DeleteCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())

				// Verify cluster is deleted
				_, err = client.DescribeCluster(clusterName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("when testing JSON marshaling compatibility", func() {
			It("should handle cluster settings correctly", func() {
				clusterName := utils.GenerateTestName("settings-cluster")
				DeferCleanup(func() {
					_ = client.DeleteCluster(clusterName)
				})

				// Create cluster (settings would be set via API if supported)
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())

				// Describe cluster to check settings
				cluster, err := client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())

				// Settings should be parsed correctly (even if empty)
				Expect(cluster.Settings).NotTo(BeNil())
			})
		})
	})

	Describe("Error Handling with Generated Types", func() {
		It("should handle cluster not found error", func() {
			nonExistentCluster := "non-existent-cluster"
			
			_, err := client.DescribeCluster(nonExistentCluster)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should handle duplicate cluster creation", func() {
			clusterName := utils.GenerateTestName("dup-cluster")
			DeferCleanup(func() {
				_ = client.DeleteCluster(clusterName)
			})

			// Create cluster
			err := client.CreateCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())

			// Try to create again - should not error (idempotent)
			err = client.CreateCluster(clusterName)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})