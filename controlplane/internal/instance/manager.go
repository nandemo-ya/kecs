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
	"os"
	"path/filepath"
	"sync"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	kecs "github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
)

// StartOptions contains options for starting a KECS instance
type StartOptions struct {
	InstanceName string
	DataDir      string
	ConfigFile   string
	NoLocalStack bool
	NoTraefik    bool
	ApiPort      int
	AdminPort    int
	DevMode      bool
}

// CreationStatus represents the status of instance creation
type CreationStatus struct {
	Step    string // Current step name
	Status  string // "pending", "running", "done", "failed"
	Message string // Optional message
}

// Manager handles KECS instance lifecycle
type Manager struct {
	k3dManager *kecs.K3dClusterManager
	
	// Creation status tracking
	statusMu       sync.RWMutex
	creationStatus map[string]*CreationStatus // map[instanceName]*CreationStatus
}

// NewManager creates a new instance manager
func NewManager() (*Manager, error) {
	// Create k3d manager
	k3dManager, err := kecs.NewK3dClusterManager(nil)
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

	// Load configuration
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override with options
	if opts.NoLocalStack {
		cfg.LocalStack.Enabled = false
	}
	if opts.NoTraefik {
		cfg.Features.Traefik = false
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
	errChan := make(chan error, 3)

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

	// Deploy Traefik if enabled
	if cfg.Features.Traefik {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.updateStatus(opts.InstanceName, "Configuring Traefik", "running")
			if err := m.deployTraefik(ctx, opts.InstanceName, cfg, opts.ApiPort); err != nil {
				m.updateStatus(opts.InstanceName, "Configuring Traefik", "failed", err.Error())
				errChan <- fmt.Errorf("failed to deploy Traefik: %w", err)
				return
			}
			m.updateStatus(opts.InstanceName, "Configuring Traefik", "done")
		}()
	}

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
	// Implementation will be moved from stop.go
	return nil
}

// List lists all KECS instances
func (m *Manager) List(ctx context.Context) ([]InstanceInfo, error) {
	// Implementation will be moved from existing code
	return nil, nil
}

// InstanceInfo contains information about a KECS instance
type InstanceInfo struct {
	Name      string
	Status    string
	CreatedAt string
	ApiPort   int
}

// Helper functions will be implemented below...