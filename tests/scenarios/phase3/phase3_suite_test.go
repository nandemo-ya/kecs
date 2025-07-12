package phase3

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

func TestPhase3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Phase3 Suite - LocalStack Transparent Communication")
}

var _ = BeforeSuite(func() {
	// In simple mode, KECS is already running (managed by Makefile)
	if utils.IsSimpleMode() {
		sharedKECS = utils.StartKECSSimple(GinkgoT())
	} else {
		// Original behavior for backward compatibility
		sharedKECS = utils.StartKECSForTest(GinkgoT(), "phase3-suite")
	}
	
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

	// In simple mode, don't clean up the KECS instance (managed by Makefile)
	if !utils.IsSimpleMode() {
		// Clean up container after all tests
		if sharedKECS != nil {
			sharedKECS.Cleanup()
		}
		
		// Clean up any orphaned resources in native mode
		if err := utils.CleanupTestResources(); err != nil {
			GinkgoT().Logf("Warning: failed to cleanup test resources: %v", err)
		}
	}
})
