package phase1_test

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

// Suite-level shared resources
var (
	sharedKECS           *utils.KECSContainer
	sharedClient         utils.ECSClientInterface
	sharedLogger         *utils.TestLogger
	sharedClusterManager *utils.SharedClusterManager
)

var _ = BeforeSuite(func() {
	// Start KECS container once for the entire suite
	sharedKECS = utils.StartKECS(GinkgoT())
	sharedClient = utils.NewECSClientInterface(sharedKECS.Endpoint(), utils.AWSCLIMode)
	sharedLogger = utils.NewTestLogger(GinkgoT())
	
	// Initialize shared cluster manager
	sharedClusterManager = utils.NewSharedClusterManager(sharedClient, true)
})

var _ = AfterSuite(func() {
	// Clean up shared clusters first
	if sharedClusterManager != nil {
		sharedClusterManager.CleanupAll()
	}
	
	// Clean up container after all tests
	if sharedKECS != nil {
		sharedKECS.Cleanup()
	}
})

func TestPhase1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Phase 1 - Cluster Operations Suite")
}

// cleanupAllClusters is a helper function to remove all clusters
func cleanupAllClusters() {
	clusters, err := sharedClient.ListClusters()
	Expect(err).NotTo(HaveOccurred())
	
	for _, clusterArn := range clusters {
		// Extract cluster name from ARN
		parts := strings.Split(clusterArn, "/")
		if len(parts) > 0 {
			clusterName := parts[len(parts)-1]
			err := sharedClient.DeleteCluster(clusterName)
			if err != nil {
				// Log but don't fail - cluster might have resources
				GinkgoWriter.Printf("Warning: Failed to delete cluster %s: %v\n", clusterName, err)
			}
		}
	}
	
	// Verify cleanup worked
	clustersAfter, _ := sharedClient.ListClusters()
	if len(clustersAfter) > 0 {
		GinkgoWriter.Printf("Warning: %d clusters remain after cleanup\n", len(clustersAfter))
	}
}