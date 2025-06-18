package three_tier_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestThreeTierApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Three Tier Application Test Suite")
}