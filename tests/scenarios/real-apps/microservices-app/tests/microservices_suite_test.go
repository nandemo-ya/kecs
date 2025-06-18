package microservices_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMicroservicesApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Microservices Application Test Suite")
}