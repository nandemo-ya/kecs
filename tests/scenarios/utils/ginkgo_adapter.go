package utils

import "testing"

// GinkgoTAdapter adapts testing.T to work with our TestingT interface
type GinkgoTAdapter struct {
	T *testing.T
}

// Logf logs a formatted message
func (g *GinkgoTAdapter) Logf(format string, args ...interface{}) {
	g.T.Logf(format, args...)
}

// Fatalf logs a formatted message and marks the test as failed
func (g *GinkgoTAdapter) Fatalf(format string, args ...interface{}) {
	g.T.Fatalf(format, args...)
}

// Cleanup registers a function to be called when the test completes
func (g *GinkgoTAdapter) Cleanup(f func()) {
	g.T.Cleanup(f)
}