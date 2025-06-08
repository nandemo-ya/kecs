package utils

import (
	"context"
	"fmt"
	"os"
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

	// Container request configuration
	req := testcontainers.ContainerRequest{
		Image:        "kecs:test",
		ExposedPorts: []string{"8080/tcp", "8081/tcp"},
		Env: map[string]string{
			"LOG_LEVEL":                 getEnvOrDefault("KECS_LOG_LEVEL", "info"),
			"KECS_DISABLE_KIND_CLUSTER": "false", // Enable kind cluster creation
		},
		WaitingFor: wait.ForHTTP("/health").
			WithPort("8081/tcp").
			WithStartupTimeout(30*time.Second),
	}

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
	time.Sleep(5 * time.Second)

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
	logs, err := k.container.Logs(k.ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	
	buf := make([]byte, 10000)
	n, _ := logs.Read(buf)
	return string(buf[:n]), nil
}

// Cleanup terminates the container and cleans up resources
func (k *KECSContainer) Cleanup() error {
	if k.container != nil {
		return k.container.Terminate(k.ctx)
	}
	return nil
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