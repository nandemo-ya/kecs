// simple_kecs.go - Simplified KECS client for single-instance testing
//
// This file provides simple utilities for tests running with a single
// KECS instance managed by TestMain.

package utils

import (
	"fmt"
	"os"
)

// SimpleKECSClient provides simplified access to the single KECS instance
type SimpleKECSClient struct {
	endpoint      string
	adminEndpoint string
}

// globalSimpleClient is the singleton instance for simple mode
var globalSimpleClient *SimpleKECSClient

// GetSimpleKECSClient returns the global simple KECS client
func GetSimpleKECSClient() *SimpleKECSClient {
	if globalSimpleClient == nil {
		globalSimpleClient = &SimpleKECSClient{
			endpoint:      os.Getenv("KECS_ENDPOINT"),
			adminEndpoint: os.Getenv("KECS_ADMIN_ENDPOINT"),
		}
		
		// Validate endpoints are set
		if globalSimpleClient.endpoint == "" {
			globalSimpleClient.endpoint = "http://localhost:8080"
		}
		if globalSimpleClient.adminEndpoint == "" {
			globalSimpleClient.adminEndpoint = "http://localhost:8081"
		}
	}
	return globalSimpleClient
}

// GetEndpoint returns the API endpoint
func (c *SimpleKECSClient) GetEndpoint() string {
	return c.endpoint
}

// GetAdminEndpoint returns the admin endpoint
func (c *SimpleKECSClient) GetAdminEndpoint() string {
	return c.adminEndpoint
}

// IsSimpleMode checks if we're running in simple mode
func IsSimpleMode() bool {
	return os.Getenv("KECS_TEST_MODE") == "simple"
}

// StartKECSSimple returns a mock container interface for simple mode
// This allows existing tests to work without modification
func StartKECSSimple(t TestingT) KECSContainerInterface {
	if !IsSimpleMode() {
		// Fall back to original implementation
		return StartKECSForTest(t, "fallback")
	}
	
	// Return a simple adapter that uses the global instance
	client := GetSimpleKECSClient()
	return &SimpleKECSAdapter{
		endpoint:      client.endpoint,
		adminEndpoint: client.adminEndpoint,
	}
}

// SimpleKECSAdapter implements KECSContainerInterface for simple mode
type SimpleKECSAdapter struct {
	endpoint      string
	adminEndpoint string
}

// Endpoint returns the API endpoint
func (a *SimpleKECSAdapter) Endpoint() string {
	return a.endpoint
}

// AdminEndpoint returns the admin endpoint
func (a *SimpleKECSAdapter) AdminEndpoint() string {
	return a.adminEndpoint
}

// APIEndpoint returns the API endpoint (compatibility)
func (a *SimpleKECSAdapter) APIEndpoint() string {
	return a.endpoint
}

// GetLogs returns empty logs in simple mode
func (a *SimpleKECSAdapter) GetLogs() (string, error) {
	return "", fmt.Errorf("logs not available in simple mode - check main process output")
}

// Cleanup does nothing in simple mode (process managed by TestMain)
func (a *SimpleKECSAdapter) Cleanup() error {
	// No-op in simple mode
	return nil
}

// Stop does nothing in simple mode
func (a *SimpleKECSAdapter) Stop() error {
	// No-op in simple mode
	return nil
}

// RunCommand is not supported in simple mode
func (a *SimpleKECSAdapter) RunCommand(command ...string) (string, error) {
	return "", fmt.Errorf("RunCommand is not supported in simple mode")
}

// ExecuteCommand is not supported in simple mode
func (a *SimpleKECSAdapter) ExecuteCommand(args ...string) (string, error) {
	return "", fmt.Errorf("ExecuteCommand is not supported in simple mode")
}

// UpdateStartKECSForTest modifies the existing function to use simple mode when enabled
func init() {
	// Override the global function if in simple mode
	if IsSimpleMode() {
		// This will be called when the package is imported
		// We can't directly override functions, but we can provide alternatives
	}
}