package phase1_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPhase1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Phase 1 - Cluster Operations Suite")
}