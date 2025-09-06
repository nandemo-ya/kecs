// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package instance

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/host/k3d"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// StartOptions contains options for starting a KECS instance
type StartOptions struct {
	InstanceName                 string
	DataDir                      string
	ConfigFile                   string
	AdditionalLocalStackServices string // Comma-separated list of additional LocalStack services
	ApiPort                      int
	AdminPort                    int
}

// CreationStatus represents the status of instance creation
type CreationStatus struct {
	Step    string // Current step name
	Status  string // "pending", "running", "done", "failed"
	Message string // Optional message
}

// Manager handles KECS instance lifecycle
type Manager struct {
	k3dManager *k3d.K3dClusterManager

	// Creation status tracking
	statusMu       sync.RWMutex
	creationStatus map[string]*CreationStatus // map[instanceName]*CreationStatus
}

// NewManager creates a new instance manager
func NewManager() (*Manager, error) {
	// Default configuration - registry is always enabled for all instances
	k3dConfig := &k3d.ClusterManagerConfig{
		Provider:       "k3d",
		EnableRegistry: true,  // Always enabled for local development
		ContainerMode:  false, // TUI mode is not container mode
	}
	k3dManager, err := k3d.NewK3dClusterManager(k3dConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k3d manager: %w", err)
	}

	return &Manager{
		k3dManager:     k3dManager,
		creationStatus: make(map[string]*CreationStatus),
	}, nil
}

// updateStatus updates the creation status for an instance
func (m *Manager) updateStatus(instanceName, step, status string, message ...string) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()

	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}

	m.creationStatus[instanceName] = &CreationStatus{
		Step:    step,
		Status:  status,
		Message: msg,
	}
}

// GetCreationStatus returns the current creation status for an instance
func (m *Manager) GetCreationStatus(instanceName string) *CreationStatus {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()

	if status, ok := m.creationStatus[instanceName]; ok {
		// Return a copy to avoid race conditions
		return &CreationStatus{
			Step:    status.Step,
			Status:  status.Status,
			Message: status.Message,
		}
	}
	return nil
}

// Start starts a KECS instance with the given options
func (m *Manager) Start(ctx context.Context, opts StartOptions) error {
	// Generate instance name if not provided
	if opts.InstanceName == "" {
		opts.InstanceName = generateInstanceName()
	}

	// Handle automatic port allocation
	if opts.ApiPort == 0 || opts.AdminPort == 0 {
		allocatedApiPort, allocatedAdminPort, err := m.allocatePorts(ctx, opts.ApiPort, opts.AdminPort)
		if err != nil {
			return fmt.Errorf("failed to allocate ports: %w", err)
		}
		if opts.ApiPort == 0 {
			opts.ApiPort = allocatedApiPort
		}
		if opts.AdminPort == 0 {
			opts.AdminPort = allocatedAdminPort
		}
	}

	// Check if instance already exists and is stopped
	exists, err := m.k3dManager.ClusterExists(ctx, opts.InstanceName)
	if err != nil {
		return fmt.Errorf("failed to check cluster existence: %w", err)
	}

	if exists {
		// Check if it's running
		running, err := m.k3dManager.IsClusterRunning(ctx, opts.InstanceName)
		if err != nil {
			return fmt.Errorf("failed to check cluster status: %w", err)
		}

		if running {
			return fmt.Errorf("instance '%s' is already running", opts.InstanceName)
		}

		// Instance exists but is stopped - restart it
		return m.restartInstance(ctx, opts)
	}

	// Load configuration
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// LocalStack is always enabled
	cfg.LocalStack.Enabled = true

	// Add additional services if specified
	if opts.AdditionalLocalStackServices != "" {
		additionalServices := strings.Split(opts.AdditionalLocalStackServices, ",")
		for i := range additionalServices {
			additionalServices[i] = strings.TrimSpace(additionalServices[i])
		}
		cfg.LocalStack.Services = mergeLocalStackServices(cfg.LocalStack.Services, additionalServices)
	}

	// Set up data directory
	if opts.DataDir == "" {
		home, _ := os.UserHomeDir()
		opts.DataDir = filepath.Join(home, ".kecs", "instances", opts.InstanceName, "data")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(opts.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Save instance configuration
	if err := SaveInstanceConfig(opts.InstanceName, opts); err != nil {
		// Log warning but don't fail - config saving is not critical
		// TODO: Add proper logging here
	}

	// Step 1: Create k3d cluster
	m.updateStatus(opts.InstanceName, "Creating k3d cluster", "running")
	if err := m.createCluster(ctx, opts.InstanceName, cfg, opts); err != nil {
		m.updateStatus(opts.InstanceName, "Creating k3d cluster", "failed", err.Error())
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}
	m.updateStatus(opts.InstanceName, "Creating k3d cluster", "done")

	// Step 2: Create namespace
	m.updateStatus(opts.InstanceName, "Creating namespace", "running")
	if err := m.createNamespace(ctx, opts.InstanceName); err != nil {
		m.updateStatus(opts.InstanceName, "Creating namespace", "failed", err.Error())
		return fmt.Errorf("failed to create namespace: %w", err)
	}
	m.updateStatus(opts.InstanceName, "Creating namespace", "done")

	// Step 3: Deploy components in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, 4) // Increased channel size for Vector

	// Deploy Control Plane
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.updateStatus(opts.InstanceName, "Deploying control plane", "running")
		if err := m.deployControlPlane(ctx, opts.InstanceName, cfg, opts); err != nil {
			m.updateStatus(opts.InstanceName, "Deploying control plane", "failed", err.Error())
			errChan <- fmt.Errorf("failed to deploy control plane: %w", err)
			return
		}
		m.updateStatus(opts.InstanceName, "Deploying control plane", "done")
	}()

	// Deploy LocalStack if enabled
	if cfg.LocalStack.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.updateStatus(opts.InstanceName, "Starting LocalStack", "running")
			if err := m.deployLocalStack(ctx, opts.InstanceName, cfg); err != nil {
				m.updateStatus(opts.InstanceName, "Starting LocalStack", "failed", err.Error())
				errChan <- fmt.Errorf("failed to deploy LocalStack: %w", err)
				return
			}
			m.updateStatus(opts.InstanceName, "Starting LocalStack", "done")
		}()
	}

	// Deploy Vector for log aggregation
	// Vector is always deployed for CloudWatch Logs support
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.updateStatus(opts.InstanceName, "Deploying Vector", "running")
		if err := m.deployVector(ctx, opts.InstanceName, cfg); err != nil {
			m.updateStatus(opts.InstanceName, "Deploying Vector", "failed", err.Error())
			// Vector deployment failure is not critical, just log warning
			logging.Warn("Failed to deploy Vector DaemonSet", "error", err)
			// Don't send to errChan to avoid failing the entire startup
		} else {
			m.updateStatus(opts.InstanceName, "Deploying Vector", "done")
		}
	}()

	// Wait for deployments
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return err
	}

	// Step 4: Wait for readiness
	m.updateStatus(opts.InstanceName, "Finalizing", "running")
	if err := m.waitForReady(ctx, opts.InstanceName, cfg); err != nil {
		m.updateStatus(opts.InstanceName, "Finalizing", "failed", err.Error())
		return fmt.Errorf("components failed to become ready: %w", err)
	}
	m.updateStatus(opts.InstanceName, "Finalizing", "done")

	// Clear status after successful creation
	m.statusMu.Lock()
	delete(m.creationStatus, opts.InstanceName)
	m.statusMu.Unlock()

	return nil
}

// Stop stops a KECS instance
func (m *Manager) Stop(ctx context.Context, instanceName string) error {
	// Check if instance exists
	exists, err := m.k3dManager.ClusterExists(ctx, instanceName)
	if err != nil {
		return fmt.Errorf("failed to check instance existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("instance '%s' does not exist", instanceName)
	}

	// Check if instance is running
	running, err := m.k3dManager.IsClusterRunning(ctx, instanceName)
	if err != nil {
		return fmt.Errorf("failed to check instance status: %w", err)
	}

	if !running {
		return fmt.Errorf("instance '%s' is not running", instanceName)
	}

	// Stop the k3d cluster
	if err := m.k3dManager.StopCluster(ctx, instanceName); err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	return nil
}

// Destroy destroys a KECS instance
func (m *Manager) Destroy(ctx context.Context, instanceName string, deleteData bool) error {
	// Check if instance exists
	exists, err := m.k3dManager.ClusterExists(ctx, instanceName)
	if err != nil {
		return fmt.Errorf("failed to check instance existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("instance '%s' does not exist", instanceName)
	}

	// Delete the k3d cluster
	if err := m.k3dManager.DeleteCluster(ctx, instanceName); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	// Delete data if requested
	if deleteData {
		home, _ := os.UserHomeDir()
		dataDir := filepath.Join(home, ".kecs", "instances", instanceName, "data")

		if err := os.RemoveAll(dataDir); err != nil {
			// Non-fatal error - just log it
			// TODO: Add proper logging here
		}

		// Also delete the instance directory if it's empty
		instanceDir := filepath.Join(home, ".kecs", "instances", instanceName)
		os.Remove(instanceDir) // This will only succeed if directory is empty
	}

	return nil
}

// List lists all KECS instances
func (m *Manager) List(ctx context.Context) ([]InstanceInfo, error) {
	// Get list of k3d clusters
	clusters, err := m.k3dManager.ListClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	instances := make([]InstanceInfo, 0, len(clusters))
	for _, clusterInfo := range clusters {
		// Check if cluster is running
		running, _ := m.k3dManager.IsClusterRunning(ctx, clusterInfo.Name)
		status := "STOPPED"
		if running {
			status = "RUNNING"
		}

		// Load instance config to get ports
		cfg, _ := LoadInstanceConfig(clusterInfo.Name)

		// Check for data directory
		home, _ := os.UserHomeDir()
		dataDir := filepath.Join(home, ".kecs", "instances", clusterInfo.Name, "data")
		hasData := false
		if _, err := os.Stat(dataDir); err == nil {
			hasData = true
		}

		instances = append(instances, InstanceInfo{
			Name:       clusterInfo.Name,
			Status:     status,
			ApiPort:    cfg.APIPort,
			AdminPort:  cfg.AdminPort,
			HasData:    hasData,
			LocalStack: cfg.LocalStack,
		})
	}

	return instances, nil
}

// InstanceInfo contains information about a KECS instance
type InstanceInfo struct {
	Name       string
	Status     string
	ApiPort    int
	AdminPort  int
	HasData    bool
	LocalStack bool
}

// IsRunning checks if an instance is running
func (m *Manager) IsRunning(ctx context.Context, instanceName string) (bool, error) {
	exists, err := m.k3dManager.ClusterExists(ctx, instanceName)
	if err != nil {
		return false, fmt.Errorf("failed to check instance existence: %w", err)
	}

	if !exists {
		return false, nil
	}

	return m.k3dManager.IsClusterRunning(ctx, instanceName)
}

// Restart restarts a stopped instance (deprecated - use Start instead)
func (m *Manager) Restart(ctx context.Context, instanceName string) error {
	// Check if instance exists
	exists, err := m.k3dManager.ClusterExists(ctx, instanceName)
	if err != nil {
		return fmt.Errorf("failed to check instance existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("instance '%s' does not exist", instanceName)
	}

	// Check if instance is running
	running, err := m.k3dManager.IsClusterRunning(ctx, instanceName)
	if err != nil {
		return fmt.Errorf("failed to check instance status: %w", err)
	}

	if running {
		return fmt.Errorf("instance '%s' is already running", instanceName)
	}

	// Load saved instance configuration
	savedConfig, err := LoadInstanceConfig(instanceName)
	if err != nil {
		// If no saved config, use defaults
		savedConfig = &InstanceConfig{
			APIPort:    4566,
			AdminPort:  5374,
			LocalStack: true,
		}
	}

	// Convert to StartOptions
	opts := StartOptions{
		InstanceName: instanceName,
		ApiPort:      savedConfig.APIPort,
		AdminPort:    savedConfig.AdminPort,
		// LocalStack is always enabled, no additional services for restart
		AdditionalLocalStackServices: "",
	}

	// Use restartInstance to handle the restart
	return m.restartInstance(ctx, opts)
}

// restartInstance restarts a stopped instance and redeploys all components
func (m *Manager) restartInstance(ctx context.Context, opts StartOptions) error {

	// Load configuration
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// LocalStack is always enabled
	cfg.LocalStack.Enabled = true

	// Add additional services if specified
	if opts.AdditionalLocalStackServices != "" {
		additionalServices := strings.Split(opts.AdditionalLocalStackServices, ",")
		for i := range additionalServices {
			additionalServices[i] = strings.TrimSpace(additionalServices[i])
		}
		cfg.LocalStack.Services = mergeLocalStackServices(cfg.LocalStack.Services, additionalServices)
	}

	// Load saved instance config if available
	savedConfig, err := LoadInstanceConfig(opts.InstanceName)
	if err == nil {
		// Use saved config values if not overridden by command line
		if opts.ApiPort == 0 {
			opts.ApiPort = savedConfig.APIPort
		}
		if opts.AdminPort == 0 {
			opts.AdminPort = savedConfig.AdminPort
		}
		if opts.DataDir == "" {
			opts.DataDir = savedConfig.DataDir
		}
	}

	// Set up data directory
	if opts.DataDir == "" {
		home, _ := os.UserHomeDir()
		opts.DataDir = filepath.Join(home, ".kecs", "instances", opts.InstanceName, "data")
	}

	// Step 1: Start the k3d cluster
	m.updateStatus(opts.InstanceName, "Starting k3d cluster", "running")
	if err := m.k3dManager.StartCluster(ctx, opts.InstanceName); err != nil {
		m.updateStatus(opts.InstanceName, "Starting k3d cluster", "failed", err.Error())
		return fmt.Errorf("failed to start k3d cluster: %w", err)
	}
	m.updateStatus(opts.InstanceName, "Starting k3d cluster", "done")

	// Step 2: Wait for cluster to be ready
	m.updateStatus(opts.InstanceName, "Waiting for cluster", "running")
	if err := m.k3dManager.WaitForClusterReady(ctx, opts.InstanceName); err != nil {
		m.updateStatus(opts.InstanceName, "Waiting for cluster", "failed", err.Error())
		return fmt.Errorf("cluster did not become ready: %w", err)
	}
	m.updateStatus(opts.InstanceName, "Waiting for cluster", "done")

	// Step 3: Recreate namespace (in case it was deleted)
	m.updateStatus(opts.InstanceName, "Creating namespace", "running")
	if err := m.createOrUpdateNamespace(ctx, opts.InstanceName); err != nil {
		m.updateStatus(opts.InstanceName, "Creating namespace", "failed", err.Error())
		return fmt.Errorf("failed to create namespace: %w", err)
	}
	m.updateStatus(opts.InstanceName, "Creating namespace", "done")

	// Step 4: Deploy components in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, 4)

	// Deploy Control Plane
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.updateStatus(opts.InstanceName, "Deploying control plane", "running")
		if err := m.deployControlPlane(ctx, opts.InstanceName, cfg, opts); err != nil {
			m.updateStatus(opts.InstanceName, "Deploying control plane", "failed", err.Error())
			errChan <- fmt.Errorf("failed to deploy control plane: %w", err)
			return
		}
		m.updateStatus(opts.InstanceName, "Deploying control plane", "done")
	}()

	// Deploy LocalStack if enabled
	if cfg.LocalStack.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.updateStatus(opts.InstanceName, "Starting LocalStack", "running")
			if err := m.deployLocalStack(ctx, opts.InstanceName, cfg); err != nil {
				m.updateStatus(opts.InstanceName, "Starting LocalStack", "failed", err.Error())
				errChan <- fmt.Errorf("failed to deploy LocalStack: %w", err)
				return
			}
			m.updateStatus(opts.InstanceName, "Starting LocalStack", "done")
		}()
	}

	// Deploy Vector for log aggregation
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.updateStatus(opts.InstanceName, "Deploying Vector", "running")
		if err := m.deployVector(ctx, opts.InstanceName, cfg); err != nil {
			m.updateStatus(opts.InstanceName, "Deploying Vector", "failed", err.Error())
			// Vector deployment failure is not critical, just log warning
			logging.Warn("Failed to deploy Vector DaemonSet", "error", err)
		} else {
			m.updateStatus(opts.InstanceName, "Deploying Vector", "done")
		}
	}()

	// Wait for deployments
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return err
	}

	// Step 5: Wait for readiness
	m.updateStatus(opts.InstanceName, "Finalizing", "running")
	if err := m.waitForReady(ctx, opts.InstanceName, cfg); err != nil {
		m.updateStatus(opts.InstanceName, "Finalizing", "failed", err.Error())
		return fmt.Errorf("components failed to become ready: %w", err)
	}
	m.updateStatus(opts.InstanceName, "Finalizing", "done")

	// Clear status after successful restart
	m.statusMu.Lock()
	delete(m.creationStatus, opts.InstanceName)
	m.statusMu.Unlock()

	// Save updated instance configuration
	if err := SaveInstanceConfig(opts.InstanceName, opts); err != nil {
		// Log warning but don't fail - config saving is not critical
		// TODO: Add proper logging here
	}

	return nil
}

// allocatePorts allocates available ports for API and Admin services
// If a port is already specified (non-zero), it will be used as-is
// Returns (apiPort, adminPort, error)
func (m *Manager) allocatePorts(ctx context.Context, requestedApiPort, requestedAdminPort int) (int, int, error) {
	// Default ports
	defaultApiPort := 5373
	defaultAdminPort := 5374

	// If both ports are specified, return them
	if requestedApiPort != 0 && requestedAdminPort != 0 {
		return requestedApiPort, requestedAdminPort, nil
	}

	// Get list of existing instances to check port usage
	clusters, err := m.k3dManager.ListClusters(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to list clusters: %w", err)
	}

	// Build a map of used ports from existing instances
	usedPorts := make(map[int]bool)
	for _, cluster := range clusters {
		// Try to load the saved config for each instance
		if config, err := LoadInstanceConfig(cluster.Name); err == nil {
			usedPorts[config.APIPort] = true
			usedPorts[config.AdminPort] = true
		}
	}

	// Determine API port
	apiPort := requestedApiPort
	if apiPort == 0 {
		// Check if default is available
		if !usedPorts[defaultApiPort] && isPortAvailable(defaultApiPort) {
			apiPort = defaultApiPort
		} else {
			// Find an available port starting from default + 10
			apiPort = findAvailablePort(defaultApiPort+10, usedPorts)
		}
	}

	// Determine Admin port
	adminPort := requestedAdminPort
	if adminPort == 0 {
		// Check if default is available
		if !usedPorts[defaultAdminPort] && adminPort != apiPort && isPortAvailable(defaultAdminPort) {
			adminPort = defaultAdminPort
		} else {
			// Find an available port starting from default + 10
			adminPort = findAvailablePort(defaultAdminPort+10, usedPorts)
			// Make sure it's different from API port
			if adminPort == apiPort {
				adminPort = findAvailablePort(adminPort+1, usedPorts)
			}
		}
	}

	// Mark these ports as used for this allocation
	usedPorts[apiPort] = true
	usedPorts[adminPort] = true

	return apiPort, adminPort, nil
}

// isPortAvailable checks if a port is available on the local system
func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// findAvailablePort finds an available port starting from the given port
func findAvailablePort(startPort int, usedPorts map[int]bool) int {
	for port := startPort; port < 65535; port++ {
		if !usedPorts[port] && isPortAvailable(port) {
			return port
		}
	}
	// Fallback - this should rarely happen
	return startPort
}

// mergeLocalStackServices merges required services with additional services
func mergeLocalStackServices(baseServices []string, additionalServices []string) []string {
	// Define required services that are always included
	requiredServices := []string{"iam", "logs", "ssm", "secretsmanager"}

	// Create a map to track unique services
	serviceMap := make(map[string]bool)

	// Add required services
	for _, service := range requiredServices {
		serviceMap[service] = true
	}

	// Add base services from configuration
	for _, service := range baseServices {
		serviceMap[service] = true
	}

	// Add additional services
	for _, service := range additionalServices {
		if service != "" {
			serviceMap[service] = true
		}
	}

	// Convert map back to slice
	result := make([]string, 0, len(serviceMap))
	for service := range serviceMap {
		result = append(result, service)
	}

	return result
}
