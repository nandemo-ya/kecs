package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// KECSContainer represents a running KECS instance for testing
type KECSContainer struct {
	container testcontainers.Container
	endpoint  string
	adminPort string
	ctx       context.Context
}

// StartKECS starts a new KECS container for testing
func StartKECS(t TestingT) *KECSContainer {
	ctx := context.Background()

	// Check if running in test mode (default to true for tests)
	testMode := getEnvOrDefault("KECS_TEST_MODE", "true")
	disableKind := "true" // Always disable Kind cluster in test containers
	
	// Debug: Print environment variable
	fmt.Printf("DEBUG: KECS_TEST_MODE from environment: %s\n", testMode)

	// Container request configuration
	req := testcontainers.ContainerRequest{
		Image:        getEnvOrDefault("KECS_IMAGE", "kecs:test"),
		ExposedPorts: []string{"8080/tcp", "8081/tcp"},
		Env: map[string]string{
			"LOG_LEVEL":                 getEnvOrDefault("KECS_LOG_LEVEL", "debug"),
			"KECS_DISABLE_KIND_CLUSTER": disableKind,
			"KECS_TEST_MODE":            testMode,
		},
		WaitingFor: wait.ForHTTP("/health").
			WithPort("8081/tcp").
			WithStartupTimeout(60*time.Second),
	}
	
	// Debug: Print environment being set
	fmt.Printf("DEBUG: Setting container env KECS_TEST_MODE=%s\n", testMode)

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start KECS container: %v", err)
	}

	// Get API endpoint
	apiHost, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to get container host: %v", err)
	}

	apiPort, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to get API port: %v", err)
	}

	adminPort, err := container.MappedPort(ctx, "8081/tcp")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to get admin port: %v", err)
	}

	endpoint := fmt.Sprintf("http://%s:%s", apiHost, apiPort.Port())

	// Wait a bit for KECS to be fully ready
	if testMode != "true" {
		// Wait for Kind cluster creation
		time.Sleep(10 * time.Second)
	} else {
		// Shorter wait in test mode
		time.Sleep(2 * time.Second)
	}

	return &KECSContainer{
		container: container,
		endpoint:  endpoint,
		adminPort: adminPort.Port(),
		ctx:       ctx,
	}
}

// Endpoint returns the KECS API endpoint URL
func (k *KECSContainer) Endpoint() string {
	return k.endpoint
}

// AdminEndpoint returns the KECS admin endpoint URL
func (k *KECSContainer) AdminEndpoint() string {
	host, _ := k.container.Host(k.ctx)
	return fmt.Sprintf("http://%s:%s", host, k.adminPort)
}

// GetLogs returns the container logs
func (k *KECSContainer) GetLogs() (string, error) {
	// Use docker logs directly to get all logs
	containerID := k.container.GetContainerID()
	cmd := exec.Command("docker", "logs", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to original method
		logs, err := k.container.Logs(k.ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get logs: %w", err)
		}
		
		// Read all available logs
		buf := make([]byte, 100000)
		n, _ := logs.Read(buf)
		return string(buf[:n]), nil
	}
	return string(output), nil
}

// ExecuteCommand executes a command inside the KECS container
func (k *KECSContainer) ExecuteCommand(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(k.ctx, 30*time.Second)
	defer cancel()

	exitCode, output, err := k.container.Exec(ctx, append([]string{"kecs"}, args...))
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}
	if exitCode != 0 {
		return string(output), fmt.Errorf("command exited with code %d: %s", exitCode, string(output))
	}
	return string(output), nil
}

// Cleanup terminates the container and cleans up resources
func (k *KECSContainer) Cleanup() error {
	if k.container != nil {
		return k.container.Terminate(k.ctx)
	}
	return nil
}

// RunCommand executes a command in the container
func (k *KECSContainer) RunCommand(command ...string) (string, error) {
	cmd := exec.Command(command[0], command[1:]...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// APIEndpoint returns the KECS API endpoint URL (same as Endpoint)
func (k *KECSContainer) APIEndpoint() string {
	return k.endpoint
}


// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := getEnv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnv is a wrapper for getting environment variables (for testing)
var getEnv = func(key string) string {
	return os.Getenv(key)
}