package utils

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// NativeKECSManager manages KECS instances running directly on Docker host
type NativeKECSManager struct {
	mu               sync.Mutex
	instances        map[string]*NativeKECSInstance
	allocatedPorts   map[int]string // port -> instance name mapping
	baseAPIPort      int
	baseAdminPort    int
	imageTag         string
	localBuild       bool
}

// NativeKECSInstance represents a running KECS instance
type NativeKECSInstance struct {
	Name           string
	APIPort        int
	AdminPort      int
	DataDir        string
	ContainerName  string
	Endpoint       string
	AdminEndpoint  string
	Started        time.Time
}

// NewNativeKECSManager creates a new native KECS manager
func NewNativeKECSManager() *NativeKECSManager {
	// Use high port numbers to avoid conflicts
	baseAPIPort := 35000
	baseAdminPort := 36000
	
	// Check environment variables for custom base ports
	if envPort := os.Getenv("KECS_TEST_BASE_API_PORT"); envPort != "" {
		if port, err := parsePort(envPort); err == nil {
			baseAPIPort = port
		}
	}
	if envPort := os.Getenv("KECS_TEST_BASE_ADMIN_PORT"); envPort != "" {
		if port, err := parsePort(envPort); err == nil {
			baseAdminPort = port
		}
	}
	
	// In CI, we pre-build the image, so default to false
	defaultLocalBuild := "false"
	if _, ci := os.LookupEnv("CI"); !ci {
		// Local development defaults to true
		defaultLocalBuild = "true"
	}
	
	return &NativeKECSManager{
		instances:      make(map[string]*NativeKECSInstance),
		allocatedPorts: make(map[int]string),
		baseAPIPort:    baseAPIPort,
		baseAdminPort:  baseAdminPort,
		imageTag:       getEnvOrDefault("KECS_IMAGE", "ghcr.io/nandemo-ya/kecs:latest"),
		localBuild:     getEnvOrDefault("KECS_LOCAL_BUILD", defaultLocalBuild) == "true",
	}
}

// StartKECS starts a new KECS instance with automatic port allocation
func (m *NativeKECSManager) StartKECS(testName string) (*NativeKECSInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Generate unique instance name
	instanceID := fmt.Sprintf("test-%s-%d", sanitizeName(testName), time.Now().UnixNano()/1e6)
	containerName := fmt.Sprintf("kecs-%s", instanceID)
	
	// Find available ports
	apiPort, adminPort, err := m.findAvailablePorts()
	if err != nil {
		return nil, fmt.Errorf("failed to find available ports: %w", err)
	}
	
	// Create temporary data directory
	dataDir, err := os.MkdirTemp("", fmt.Sprintf("kecs-%s-*", instanceID))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp data directory: %w", err)
	}
	
	// Mark ports as allocated
	m.allocatedPorts[apiPort] = containerName
	m.allocatedPorts[adminPort] = containerName
	
	// Build the kecs start command
	args := []string{
		"start",
		"--name", containerName,
		"--image", m.imageTag,
		"--api-port", fmt.Sprintf("%d", apiPort),
		"--admin-port", fmt.Sprintf("%d", adminPort),
		"--data-dir", dataDir,
		"--detach",
	}
	
	if m.localBuild {
		args = append(args, "--local-build")
	}
	
	// Execute kecs start command
	cmd := exec.Command("kecs", args...)
	
	// Set environment variables
	cmd.Env = append(os.Environ(),
		"KECS_SECURITY_ACKNOWLEDGED=true",
		"KECS_LOG_LEVEL=debug",
		"KECS_TEST_MODE=false", // Run in normal mode for integration tests
		"KECS_CONTAINER_MODE=true",
		"KECS_K3D_OPTIMIZED=true",
		"KECS_LOCALSTACK_ENABLED=true",
		"KECS_LOCALSTACK_USE_TRAEFIK=true",
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up on failure
		m.releasePortsLocked(apiPort, adminPort)
		os.RemoveAll(dataDir)
		return nil, fmt.Errorf("failed to start KECS: %w\nOutput: %s", err, output)
	}
	
	// Create instance object
	instance := &NativeKECSInstance{
		Name:          instanceID,
		APIPort:       apiPort,
		AdminPort:     adminPort,
		DataDir:       dataDir,
		ContainerName: containerName,
		Endpoint:      fmt.Sprintf("http://127.0.0.1:%d", apiPort),
		AdminEndpoint: fmt.Sprintf("http://127.0.0.1:%d", adminPort),
		Started:       time.Now(),
	}
	
	// Store instance
	m.instances[containerName] = instance
	
	// Wait for instance to be ready
	if err := m.waitForReady(instance); err != nil {
		// Clean up on failure
		m.StopKECS(instance)
		return nil, fmt.Errorf("KECS instance failed to become ready: %w", err)
	}
	
	return instance, nil
}

// StartKECSWithDataDir starts a new KECS instance with a specific data directory
func (m *NativeKECSManager) StartKECSWithDataDir(testName string, dataDir string) (*NativeKECSInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Generate unique instance name
	instanceID := fmt.Sprintf("test-%s-%d", sanitizeName(testName), time.Now().UnixNano()/1e6)
	containerName := fmt.Sprintf("kecs-%s", instanceID)
	
	// Find available ports
	apiPort, adminPort, err := m.findAvailablePorts()
	if err != nil {
		return nil, fmt.Errorf("failed to find available ports: %w", err)
	}
	
	// Mark ports as allocated
	m.allocatedPorts[apiPort] = containerName
	m.allocatedPorts[adminPort] = containerName
	
	// Build the kecs start command
	args := []string{
		"start",
		"--name", containerName,
		"--image", m.imageTag,
		"--api-port", fmt.Sprintf("%d", apiPort),
		"--admin-port", fmt.Sprintf("%d", adminPort),
		"--data-dir", dataDir,
		"--detach",
	}
	
	if m.localBuild {
		args = append(args, "--local-build")
	}
	
	// Execute kecs start command
	fmt.Printf("Starting KECS with command: kecs %s\n", strings.Join(args, " "))
	fmt.Printf("Using image: %s\n", m.imageTag)
	fmt.Printf("Local build: %v\n", m.localBuild)
	
	// First check if kecs binary exists
	if _, err := exec.LookPath("kecs"); err != nil {
		return nil, fmt.Errorf("kecs binary not found in PATH: %w", err)
	}
	
	// Check if the Docker image exists (only if not local build)
	if !m.localBuild {
		checkImageCmd := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}", m.imageTag)
		imageOutput, _ := checkImageCmd.CombinedOutput()
		fmt.Printf("Available Docker images matching %s:\n%s\n", m.imageTag, string(imageOutput))
		
		// List all images for debugging
		allImagesCmd := exec.Command("docker", "images", "--format", "table {{.Repository}}:{{.Tag}}\t{{.Size}}")
		allImagesOutput, _ := allImagesCmd.CombinedOutput()
		fmt.Printf("All available Docker images:\n%s\n", string(allImagesOutput))
	}
	
	cmd := exec.Command("kecs", args...)
	
	// Set environment variables
	cmd.Env = append(os.Environ(),
		"KECS_SECURITY_ACKNOWLEDGED=true",
		"KECS_LOG_LEVEL=debug",
		"KECS_TEST_MODE=false", // Run in normal mode for integration tests
		"KECS_CONTAINER_MODE=true",
		"KECS_K3D_OPTIMIZED=true",
		"KECS_LOCALSTACK_ENABLED=true",
		"KECS_LOCALSTACK_USE_TRAEFIK=true",
		"KECS_AUTO_RECOVER_STATE=true", // Enable auto recovery for persistence tests
	)
	
	// Add project root if in CI
	if projectRoot := os.Getenv("KECS_PROJECT_ROOT"); projectRoot != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("KECS_PROJECT_ROOT=%s", projectRoot))
	}
	
	output, err := cmd.CombinedOutput()
	fmt.Printf("KECS start command completed with error: %v\n", err)
	fmt.Printf("KECS start command output:\n%s\n", string(output))
	
	if err != nil {
		// Clean up on failure
		m.releasePortsLocked(apiPort, adminPort)
		return nil, fmt.Errorf("failed to start KECS: %w\nOutput: %s", err, output)
	}
	
	// Verify container is actually running
	checkCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	checkOutput, checkErr := checkCmd.CombinedOutput()
	if checkErr != nil || !strings.Contains(string(checkOutput), containerName) {
		// List all containers to debug
		allCmd := exec.Command("docker", "ps", "-a", "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}")
		allOutput, _ := allCmd.CombinedOutput()
		return nil, fmt.Errorf("container %s is not running after start command.\nDocker ps -a output:\n%s", containerName, allOutput)
	}
	
	fmt.Printf("Container %s is running\n", containerName)
	
	// Create instance object
	instance := &NativeKECSInstance{
		Name:          instanceID,
		APIPort:       apiPort,
		AdminPort:     adminPort,
		DataDir:       dataDir,
		ContainerName: containerName,
		Endpoint:      fmt.Sprintf("http://127.0.0.1:%d", apiPort),
		AdminEndpoint: fmt.Sprintf("http://127.0.0.1:%d", adminPort),
		Started:       time.Now(),
	}
	
	// Store instance
	m.instances[containerName] = instance
	
	// Wait for instance to be ready
	if err := m.waitForReady(instance); err != nil {
		// Clean up on failure
		m.StopKECS(instance)
		return nil, fmt.Errorf("KECS instance failed to become ready: %w", err)
	}
	
	return instance, nil
}

// StopKECS stops and cleans up a KECS instance
func (m *NativeKECSManager) StopKECS(instance *NativeKECSInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var errors []error
	
	// Stop the container
	cmd := exec.Command("kecs", "stop", "--name", instance.ContainerName)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Log but don't fail - container might already be stopped
		fmt.Printf("Warning: failed to stop container %s: %v\nOutput: %s\n", 
			instance.ContainerName, err, output)
	}
	
	// Clean up k3d clusters created by this instance
	// List all k3d clusters and delete those with kecs-* prefix
	cmd = exec.Command("k3d", "cluster", "list", "-o", "json")
	if output, err := cmd.Output(); err == nil && len(output) > 0 {
		// Simple approach: delete all kecs-* clusters
		// In production, we might want to track which clusters belong to which instance
		deleteCmd := exec.Command("bash", "-c", 
			"k3d cluster list -o json | jq -r '.[].name' | grep '^kecs-' | xargs -r -I {} k3d cluster delete {}")
		if err := deleteCmd.Run(); err != nil {
			fmt.Printf("Warning: failed to clean up k3d clusters: %v\n", err)
		}
	}
	
	// Clean up data directory
	if instance.DataDir != "" && strings.HasPrefix(instance.DataDir, os.TempDir()) {
		if err := os.RemoveAll(instance.DataDir); err != nil {
			errors = append(errors, fmt.Errorf("failed to remove data dir: %w", err))
		}
	}
	
	// Release ports
	m.releasePortsLocked(instance.APIPort, instance.AdminPort)
	
	// Remove from instances map
	delete(m.instances, instance.ContainerName)
	
	if len(errors) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errors)
	}
	
	return nil
}

// StopAll stops all managed KECS instances
func (m *NativeKECSManager) StopAll() error {
	m.mu.Lock()
	// Create a copy of instances to avoid modifying map during iteration
	instances := make([]*NativeKECSInstance, 0, len(m.instances))
	for _, inst := range m.instances {
		instances = append(instances, inst)
	}
	m.mu.Unlock()
	
	var errors []error
	for _, inst := range instances {
		if err := m.StopKECS(inst); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop %s: %w", inst.ContainerName, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors stopping instances: %v", errors)
	}
	
	return nil
}

// GetEndpoint returns the API endpoint for an instance
func (i *NativeKECSInstance) GetEndpoint() string {
	return i.Endpoint
}

// GetAdminEndpoint returns the admin endpoint for an instance
func (i *NativeKECSInstance) GetAdminEndpoint() string {
	return i.AdminEndpoint
}

// GetLogs retrieves logs from the KECS instance
func (i *NativeKECSInstance) GetLogs() (string, error) {
	cmd := exec.Command("kecs", "logs", "--name", i.ContainerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	return string(output), nil
}

// Helper functions

func (m *NativeKECSManager) findAvailablePorts() (apiPort, adminPort int, err error) {
	// Start searching from base ports
	apiPort = m.baseAPIPort
	adminPort = m.baseAdminPort
	
	// Find next available API port
	for attempts := 0; attempts < 1000; attempts++ {
		if m.isPortAvailable(apiPort) {
			break
		}
		apiPort++
	}
	
	// Find next available admin port
	for attempts := 0; attempts < 1000; attempts++ {
		if adminPort != apiPort && m.isPortAvailable(adminPort) {
			break
		}
		adminPort++
	}
	
	// Verify we found valid ports
	if !m.isPortAvailable(apiPort) || !m.isPortAvailable(adminPort) {
		return 0, 0, fmt.Errorf("could not find available ports after 1000 attempts")
	}
	
	return apiPort, adminPort, nil
}

func (m *NativeKECSManager) isPortAvailable(port int) bool {
	// Check if already allocated
	if _, allocated := m.allocatedPorts[port]; allocated {
		return false
	}
	
	// Check if port is actually available
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func (m *NativeKECSManager) releasePortsLocked(ports ...int) {
	for _, port := range ports {
		delete(m.allocatedPorts, port)
	}
}

func (m *NativeKECSManager) waitForReady(instance *NativeKECSInstance) error {
	client := &http.Client{Timeout: 5 * time.Second}
	// Increase timeout for CI environments
	timeoutDuration := 120 * time.Second
	if os.Getenv("CI") != "" {
		timeoutDuration = 300 * time.Second // 5 minutes in CI
	}
	deadline := time.Now().Add(timeoutDuration)
	
	fmt.Printf("Waiting for KECS instance %s to be ready (timeout: %v)...\n", instance.ContainerName, timeoutDuration)
	checkInterval := 2 * time.Second
	attempts := 0
	
	for time.Now().Before(deadline) {
		attempts++
		// Check admin health endpoint
		resp, err := client.Get(instance.AdminEndpoint + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Printf("KECS instance %s is ready after %d attempts\n", instance.ContainerName, attempts)
				// Give it a bit more time to fully initialize
				time.Sleep(2 * time.Second)
				return nil
			}
			fmt.Printf("Health check returned status %d (attempt %d)\n", resp.StatusCode, attempts)
		} else {
			if attempts % 10 == 0 {
				fmt.Printf("Health check failed (attempt %d): %v\n", attempts, err)
			}
		}
		time.Sleep(checkInterval)
	}
	
	// Get logs for debugging
	logs, _ := instance.GetLogs()
	
	// Also check container status
	statusCmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", instance.ContainerName), "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}")
	statusOutput, _ := statusCmd.CombinedOutput()
	
	return fmt.Errorf("instance did not become ready within %v. Container status:\n%s\n\nLogs:\n%s", timeoutDuration, statusOutput, logs)
}

func sanitizeName(name string) string {
	// Replace non-alphanumeric characters with hyphens
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, name)
	
	// Remove leading/trailing hyphens and collapse multiple hyphens
	result = strings.Trim(result, "-")
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	
	// Limit length
	if len(result) > 20 {
		result = result[:20]
	}
	
	return strings.ToLower(result)
}

func parsePort(s string) (int, error) {
	var port int
	_, err := fmt.Sscanf(s, "%d", &port)
	if err != nil || port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port: %s", s)
	}
	return port, nil
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// CleanupOrphanedResources cleans up any orphaned KECS test resources
func CleanupOrphanedResources() error {
	// List all kecs containers with test prefix
	cmd := exec.Command("docker", "ps", "-a", "--filter", "label=com.kecs.managed=true", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}
	
	var errors []error
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, name := range lines {
		if strings.HasPrefix(name, "kecs-test-") {
			// Stop and remove the container
			if err := exec.Command("docker", "stop", name).Run(); err != nil {
				fmt.Printf("Warning: failed to stop %s: %v\n", name, err)
			}
			if err := exec.Command("docker", "rm", name).Run(); err != nil {
				errors = append(errors, fmt.Errorf("failed to remove %s: %w", name, err))
			}
		}
	}
	
	// Clean up temp directories
	tempDir := os.TempDir()
	entries, err := os.ReadDir(tempDir)
	if err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "kecs-test-") && entry.IsDir() {
				dirPath := filepath.Join(tempDir, entry.Name())
				if err := os.RemoveAll(dirPath); err != nil {
					errors = append(errors, fmt.Errorf("failed to remove %s: %w", dirPath, err))
				}
			}
		}
	}
	
	// Clean up k3d clusters
	cmd = exec.Command("k3d", "cluster", "list", "-o", "json")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		// Parse k3d cluster list and remove test clusters
		// This would require JSON parsing, keeping it simple for now
		cmd = exec.Command("bash", "-c", "k3d cluster list -o json | jq -r '.[].name' | grep '^kecs-test-' | xargs -I {} k3d cluster delete {}")
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: failed to clean up k3d clusters: %v\n", err)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errors)
	}
	
	return nil
}

// Compatibility interface to match existing KECSContainer

// NativeKECSAdapter adapts NativeKECSInstance to work with existing test code
type NativeKECSAdapter struct {
	instance *NativeKECSInstance
	manager  *NativeKECSManager
	DataDir  string // For persistence tests
}

// NewNativeKECSAdapter creates an adapter that implements the KECSContainer interface
func NewNativeKECSAdapter(instance *NativeKECSInstance, manager *NativeKECSManager) *NativeKECSAdapter {
	return &NativeKECSAdapter{
		instance: instance,
		manager:  manager,
	}
}

// Endpoint returns the API endpoint
func (a *NativeKECSAdapter) Endpoint() string {
	return a.instance.Endpoint
}

// AdminEndpoint returns the admin endpoint
func (a *NativeKECSAdapter) AdminEndpoint() string {
	return a.instance.AdminEndpoint
}

// GetLogs returns container logs
func (a *NativeKECSAdapter) GetLogs() (string, error) {
	return a.instance.GetLogs()
}

// Cleanup stops and cleans up the instance
func (a *NativeKECSAdapter) Cleanup() error {
	return a.manager.StopKECS(a.instance)
}

// APIEndpoint returns the API endpoint (compatibility method)
func (a *NativeKECSAdapter) APIEndpoint() string {
	return a.instance.Endpoint
}

// Stop stops the container without cleanup
func (a *NativeKECSAdapter) Stop() error {
	cmd := exec.Command("docker", "stop", a.instance.ContainerName)
	return cmd.Run()
}

// RunCommand is not supported in native mode
func (a *NativeKECSAdapter) RunCommand(command ...string) (string, error) {
	return "", fmt.Errorf("RunCommand is not supported in native mode")
}

// ExecuteCommand is not supported in native mode
func (a *NativeKECSAdapter) ExecuteCommand(args ...string) (string, error) {
	return "", fmt.Errorf("ExecuteCommand is not supported in native mode")
}

// KECSContainer is an alias for compatibility with existing tests
type KECSContainer = NativeKECSAdapter

// StartKECSWithPersistence starts a KECS instance with persistent data directory
func StartKECSWithPersistence(t TestingT) *KECSContainer {
	if globalNativeManager == nil {
		globalNativeManager = NewNativeKECSManager()
		t.Cleanup(func() {
			if err := globalNativeManager.StopAll(); err != nil {
				t.Logf("Warning: failed to stop all instances: %v", err)
			}
			if err := CleanupOrphanedResources(); err != nil {
				t.Logf("Warning: failed to cleanup orphaned resources: %v", err)
			}
		})
	}
	
	// Create a persistent data directory (not in temp)
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".kecs-test", fmt.Sprintf("persistence-%d", time.Now().Unix()))
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("Failed to create persistent data directory: %v", err)
	}
	
	instance, err := globalNativeManager.StartKECSWithDataDir("persistence-test", dataDir)
	if err != nil {
		t.Fatalf("Failed to start KECS with persistence: %v", err)
	}
	
	adapter := NewNativeKECSAdapter(instance, globalNativeManager)
	adapter.DataDir = dataDir
	return adapter
}

// RestartKECSWithPersistence restarts KECS with the same data directory
func RestartKECSWithPersistence(t TestingT, dataDir string) *KECSContainer {
	if globalNativeManager == nil {
		globalNativeManager = NewNativeKECSManager()
		t.Cleanup(func() {
			if err := globalNativeManager.StopAll(); err != nil {
				t.Logf("Warning: failed to stop all instances: %v", err)
			}
			if err := CleanupOrphanedResources(); err != nil {
				t.Logf("Warning: failed to cleanup orphaned resources: %v", err)
			}
		})
	}
	
	instance, err := globalNativeManager.StartKECSWithDataDir("persistence-restart", dataDir)
	if err != nil {
		t.Fatalf("Failed to restart KECS with persistence: %v", err)
	}
	
	adapter := NewNativeKECSAdapter(instance, globalNativeManager)
	adapter.DataDir = dataDir
	return adapter
}