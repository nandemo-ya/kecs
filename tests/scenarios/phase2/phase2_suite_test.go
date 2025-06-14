package phase2_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPhase2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Phase2 Suite")
}