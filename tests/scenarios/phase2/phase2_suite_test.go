package phase2

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

// Shared resources are defined in shared_test.go

var _ = BeforeSuite(func() {
	// In simple mode, KECS is already running (managed by Makefile)
	if utils.IsSimpleMode() {
		sharedKECS = utils.StartKECSSimple(GinkgoT())
	} else {
		// Original behavior for backward compatibility
		sharedKECS = utils.StartKECSForTest(GinkgoT(), "phase2-suite")
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

func TestPhase2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Phase 2: Task Definitions and Services")
}

