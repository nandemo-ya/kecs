package phase2

import (
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

func TestPhase2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Phase 2: Task Definitions and Services")
}

