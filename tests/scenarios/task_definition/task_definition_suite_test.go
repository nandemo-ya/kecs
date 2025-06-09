package task_definition_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTaskDefinition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Task Definition Suite")
}