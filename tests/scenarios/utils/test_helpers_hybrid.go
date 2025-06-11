package utils

import (
	"os"
	"os/exec"
	"testing"
)

// TestWithBothClients runs a test function with both curl and AWS CLI clients
func TestWithBothClients(t *testing.T, testName string, testFunc func(t *testing.T, client ECSClientInterface, mode ClientMode)) {
	t.Run(testName+"_Curl", func(t *testing.T) {
		// Start KECS container
		kecs := StartKECS(t)
		defer kecs.Cleanup()
		
		// Create curl client
		client := NewECSClient(kecs.Endpoint(), CurlMode)
		testFunc(t, client, CurlMode)
	})
	
	// Only run AWS CLI tests if explicitly enabled
	if os.Getenv("TEST_WITH_AWS_CLI") == "true" {
		t.Run(testName+"_AWSCLI", func(t *testing.T) {
			// Check if AWS CLI is installed
			if !IsAWSCLIInstalled() {
				t.Skip("AWS CLI not installed")
			}
			
			// Start KECS container
			kecs := StartKECS(t)
			defer kecs.Cleanup()
			
			// Create AWS CLI client
			client := NewECSClient(kecs.Endpoint(), AWSCLIMode)
			testFunc(t, client, AWSCLIMode)
		})
	}
}

// IsAWSCLIInstalled checks if AWS CLI is available
func IsAWSCLIInstalled() bool {
	_, err := exec.LookPath("aws")
	return err == nil
}

// GetTestClient returns an ECS client based on environment variable
func GetTestClient(endpoint string) ECSClientInterface {
	if os.Getenv("USE_AWS_CLI") == "true" && IsAWSCLIInstalled() {
		return NewECSClient(endpoint, AWSCLIMode)
	}
	return NewECSClient(endpoint, CurlMode)
}