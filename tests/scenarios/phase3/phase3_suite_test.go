package phase3

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

func TestPhase3(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Phase3 Suite - LocalStack Transparent Communication")
}

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
