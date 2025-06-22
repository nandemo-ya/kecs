package phase2_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

// Shared resources for Phase 2 tests
var (
	// Logger instance
	sharedLogger *utils.TestLogger
)

// Resources that are unique per test file
var (
	// KECS container - initialized per test file
	sharedKECS *utils.KECSContainer
	
	// ECS client - initialized per test file
	sharedClient utils.ECSClientInterface
)

func TestPhase2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Phase 2: Task Definitions and Services")
}

