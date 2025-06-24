package ecs_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestECS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ECS Client Suite")
}
