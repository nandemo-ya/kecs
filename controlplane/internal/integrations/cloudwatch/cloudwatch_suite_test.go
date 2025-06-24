package cloudwatch_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCloudwatch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CloudWatch Integration Suite")
}
