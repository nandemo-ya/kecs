package advanced_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAdvancedIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Advanced Integration Test Suite")
}