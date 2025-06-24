package ssm_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSSM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SSM Integration Suite")
}
