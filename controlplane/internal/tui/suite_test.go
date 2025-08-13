package tui_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTui2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TUI v2 Suite")
}
