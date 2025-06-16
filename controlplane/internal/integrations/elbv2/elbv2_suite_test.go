package elbv2_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestELBv2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ELBv2 Integration Suite")
}