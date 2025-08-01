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

// Manager handles KECS instance lifecycle
type Manager struct {
	k3dManager *kecs.K3dClusterManager
}

// NewManager creates a new instance manager
func NewManager() (*Manager, error) {
	// Create k3d manager
	k3dManager, err := kecs.NewK3dClusterManager(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create k3d manager: %w", err)
	}

	return &Manager{
		k3dManager: k3dManager,
	}, nil
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
	if err := m.createCluster(ctx, opts.InstanceName, cfg, opts); err != nil {
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}

	// Step 2: Create namespace
	if err := m.createNamespace(ctx, opts.InstanceName); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Step 3: Deploy components in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// Deploy Control Plane
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := m.deployControlPlane(ctx, opts.InstanceName, cfg, opts); err != nil {
			errChan <- fmt.Errorf("failed to deploy control plane: %w", err)
			return
		}
	}()

	// Deploy LocalStack if enabled
	if cfg.LocalStack.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.deployLocalStack(ctx, opts.InstanceName, cfg); err != nil {
				errChan <- fmt.Errorf("failed to deploy LocalStack: %w", err)
				return
			}
		}()
	}

	// Deploy Traefik if enabled
	if cfg.Features.Traefik {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.deployTraefik(ctx, opts.InstanceName, cfg, opts.ApiPort); err != nil {
				errChan <- fmt.Errorf("failed to deploy Traefik: %w", err)
				return
			}
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
	if err := m.waitForReady(ctx, opts.InstanceName, cfg); err != nil {
		return fmt.Errorf("components failed to become ready: %w", err)
	}

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