package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

	// Check if running in test mode
	testMode := getEnvOrDefault("KECS_TEST_MODE", "true")
	// Enable container mode for proper cluster management
	containerMode := getEnvOrDefault("KECS_CONTAINER_MODE", "true")
	// Get cluster provider (k3d or kind)
	clusterProvider := getEnvOrDefault("KECS_CLUSTER_PROVIDER", "k3d")
	
	// Debug: Print environment variable
	fmt.Printf("DEBUG: KECS_TEST_MODE from environment: %s\n", testMode)
	fmt.Printf("DEBUG: KECS_CONTAINER_MODE from environment: %s\n", containerMode)
	fmt.Printf("DEBUG: KECS_CLUSTER_PROVIDER from environment: %s\n", clusterProvider)

	// Create temporary directory for kubeconfig if it doesn't exist
	kubeconfigHostPath := "/tmp/kecs-kubeconfig"
	os.MkdirAll(kubeconfigHostPath, 0755)

	// Container request configuration
	req := testcontainers.ContainerRequest{
		Image:        getEnvOrDefault("KECS_IMAGE", "kecs:test"),
		ExposedPorts: []string{"8080/tcp", "8081/tcp"},
		Env: map[string]string{
			"LOG_LEVEL":              getEnvOrDefault("KECS_LOG_LEVEL", "debug"),
			"KECS_TEST_MODE":         testMode,
			"KECS_CONTAINER_MODE":    containerMode,
			"KECS_CLUSTER_PROVIDER":  clusterProvider,
			"KECS_KUBECONFIG_PATH":   "/kecs/kubeconfig",
			"KECS_K3D_OPTIMIZED":     "true",
		},
		Mounts: testcontainers.ContainerMounts{
			{
				Source: testcontainers.GenericBindMountSource{
					HostPath: "/var/run/docker.sock",
				},
				Target:   "/var/run/docker.sock",
				ReadOnly: false,
			},
			{
				Source: testcontainers.GenericBindMountSource{
					HostPath: kubeconfigHostPath,
				},
				Target:   "/kecs/kubeconfig",
				ReadOnly: false,
			},
		},
		WaitingFor: wait.ForHTTP("/health").
			WithPort("8081/tcp").
			WithStartupTimeout(120*time.Second),
	}
	
	// Debug: Print environment being set
	fmt.Printf("DEBUG: Setting container env KECS_TEST_MODE=%s\n", testMode)
	fmt.Printf("DEBUG: Setting container env KECS_CONTAINER_MODE=%s\n", containerMode)
	fmt.Printf("DEBUG: Setting container env KECS_CLUSTER_PROVIDER=%s\n", clusterProvider)

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
	if testMode != "true" && containerMode == "true" {
		// Wait for cluster creation in container mode
		if clusterProvider == "k3d" {
			// k3d is faster to start
			time.Sleep(15 * time.Second)
		} else {
			// Kind needs more time
			time.Sleep(30 * time.Second)
		}
	} else if testMode != "true" {
		// Wait for cluster creation in normal mode
		if clusterProvider == "k3d" {
			// k3d is faster
			time.Sleep(5 * time.Second)
		} else {
			// Kind needs more time
			time.Sleep(10 * time.Second)
		}
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

	exitCode, reader, err := k.container.Exec(ctx, append([]string{"kecs"}, args...))
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}
	
	// Read output from reader
	buf := make([]byte, 4096)
	n, _ := reader.Read(buf)
	output := string(buf[:n])
	
	if exitCode != 0 {
		return output, fmt.Errorf("command exited with code %d: %s", exitCode, output)
	}
	return output, nil
}

// Cleanup terminates the container and cleans up resources
func (k *KECSContainer) Cleanup() error {
	var err error
	
	// Clean up any clusters created during tests
	if os.Getenv("KECS_CONTAINER_MODE") == "true" {
		clusterProvider := getEnvOrDefault("KECS_CLUSTER_PROVIDER", "k3d")
		
		if clusterProvider == "k3d" {
			// List and clean up any kecs-* k3d clusters
			fmt.Println("Cleaning up k3d clusters created during tests...")
			// For simplicity, just try to delete any kecs-* clusters
			// k3d doesn't have a simple "list names only" like kind
			deleteCmd := exec.Command("bash", "-c", "k3d cluster list -o json | jq -r '.[].name' | grep '^kecs-' | xargs -I {} k3d cluster delete {}")
			if err := deleteCmd.Run(); err != nil {
				// Fallback: try direct deletion of common test cluster names
				for _, clusterName := range []string{"kecs-default", "kecs-test", "kecs-cluster1", "kecs-cluster2"} {
					deleteCmd := exec.Command("k3d", "cluster", "delete", clusterName)
					deleteCmd.Run() // Ignore errors, cluster might not exist
				}
			}
		} else {
			// List and clean up any kecs-* Kind clusters
			fmt.Println("Cleaning up Kind clusters created during tests...")
			cmd := exec.Command("kind", "get", "clusters")
			output, _ := cmd.Output()
			clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
			
			for _, cluster := range clusters {
				if strings.HasPrefix(cluster, "kecs-") {
					fmt.Printf("Deleting Kind cluster: %s\n", cluster)
					deleteCmd := exec.Command("kind", "delete", "cluster", "--name", cluster)
					if deleteErr := deleteCmd.Run(); deleteErr != nil {
						fmt.Printf("Warning: failed to delete Kind cluster %s: %v\n", cluster, deleteErr)
					}
				}
			}
		}
		
		// Clean up kubeconfig directory
		kubeconfigPath := "/tmp/kecs-kubeconfig"
		if removeErr := os.RemoveAll(kubeconfigPath); removeErr != nil {
			fmt.Printf("Warning: failed to remove kubeconfig directory: %v\n", removeErr)
		}
	}
	
	// Give KECS some time to complete async k3d cluster deletion
	// This is important because DeleteCluster in KECS deletes k3d clusters asynchronously
	fmt.Println("Waiting for async k3d cluster deletion to complete...")
	time.Sleep(5 * time.Second)
	
	if k.container != nil {
		err = k.container.Terminate(k.ctx)
	}
	return err
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