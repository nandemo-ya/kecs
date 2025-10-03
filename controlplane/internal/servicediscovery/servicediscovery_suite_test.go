package servicediscovery

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestServiceDiscovery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServiceDiscovery Suite")
}
