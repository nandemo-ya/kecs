package phase1

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

// Suite-level shared resources
var (
	sharedKECS           utils.KECSContainerInterface
	sharedClient         utils.ECSClientInterface
	sharedLogger         *utils.TestLogger
	sharedClusterManager *utils.SharedClusterManager
)

var _ = BeforeSuite(func() {
	// Start KECS container once for the entire suite using factory
	sharedKECS = utils.StartKECSForTest(GinkgoT(), "phase1-suite")
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
	
	// Clean up any orphaned resources in native mode
	if err := utils.CleanupTestResources(); err != nil {
		GinkgoT().Logf("Warning: failed to cleanup test resources: %v", err)
	}
})

func TestPhase1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Phase 1 - Cluster Operations Suite")
}

