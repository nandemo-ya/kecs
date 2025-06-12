package kecs_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestKECS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "KECS Testcontainers Suite")
}