package phase1_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("K3D Cluster Integration", Serial, func() {
	var (
		client utils.ECSClientInterface
		logger *utils.TestLogger
	)

	BeforeEach(func() {
		// Use shared resources from suite
		client = sharedClient
		logger = sharedLogger
	})

	Describe("K3D Cluster Full Lifecycle", func() {
		Context("when creating a k3d-backed cluster", func() {
			var clusterName string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("k3d-test")
			})

			It("should create and delete a k3d cluster successfully", func() {
				logger.Info("Testing full k3d cluster lifecycle for: %s", clusterName)

				// Step 1: Create the cluster
				logger.Info("Step 1: Creating k3d cluster: %s", clusterName)
				err := client.CreateCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to create k3d cluster")

				// Step 2: Verify cluster exists and is active
				logger.Info("Step 2: Verifying cluster exists and is active")
				cluster, err := client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.ClusterName).To(Equal(clusterName))
				Expect(cluster.Status).To(Equal("ACTIVE"))

				// Step 3: List clusters and verify our cluster is present
				logger.Info("Step 3: Listing clusters to verify presence")
				clusters, err := client.ListClusters()
				Expect(err).NotTo(HaveOccurred())
				
				found := false
				for _, arn := range clusters {
					if containsClusterName(arn, clusterName) {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), "Cluster should be present in list")

				// Step 4: Wait a bit to ensure k3d cluster is fully initialized
				// k3d cluster creation can take 20-30 seconds
				logger.Info("Step 4: Waiting for k3d cluster to be fully initialized (30s)")
				time.Sleep(30 * time.Second)

				// Step 5: Describe cluster again to verify it's still active
				logger.Info("Step 5: Re-verifying cluster status after initialization")
				cluster, err = client.DescribeCluster(clusterName)
				Expect(err).NotTo(HaveOccurred())
				Expect(cluster.Status).To(Equal("ACTIVE"))

				// Step 6: Delete the cluster
				logger.Info("Step 6: Deleting k3d cluster: %s", clusterName)
				err = client.DeleteCluster(clusterName)
				Expect(err).NotTo(HaveOccurred(), "Failed to delete k3d cluster")

				// Step 7: Wait for k3d cluster deletion to complete
				logger.Info("Step 7: Waiting for k3d cluster deletion to complete (10s)")
				time.Sleep(10 * time.Second)

				// Step 8: Verify cluster is deleted via describe (the most important check)
				logger.Info("Step 8: Verifying cluster is deleted via describe")
				_, err = client.DescribeCluster(clusterName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))

				// Note: We're not checking ListClusters after deletion due to eventual consistency
				// issues. The DescribeCluster check above is sufficient to verify deletion.

				logger.Info("K3D cluster lifecycle test completed successfully")
			})
		})
	})
})

// Helper function to check if a cluster ARN contains the given cluster name
func containsClusterName(arn, clusterName string) bool {
	// ARN format: arn:aws:ecs:region:account:cluster/cluster-name
	// We need to check the last part after the last slash
	parts := splitARN(arn)
	if len(parts) > 0 {
		return parts[len(parts)-1] == clusterName
	}
	return false
}

// Split ARN by slashes
func splitARN(arn string) []string {
	var parts []string
	current := ""
	for _, char := range arn {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}