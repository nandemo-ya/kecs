package basic_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBasicIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Basic Integration Test Suite")
}