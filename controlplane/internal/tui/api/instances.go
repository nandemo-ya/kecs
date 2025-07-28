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

package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
)

// InstanceStatus represents the status of a KECS instance
type InstanceStatus string

const (
	InstanceRunning  InstanceStatus = "running"
	InstanceStopped  InstanceStatus = "stopped"
	InstanceNotFound InstanceStatus = "notfound"
)

// Instance represents a KECS instance
type Instance struct {
	Name       string         `json:"name"`
	Status     InstanceStatus `json:"status"`
	Running    bool           `json:"running"`
	DataExists bool           `json:"dataExists"`
	APIPort    int            `json:"apiPort"`
	AdminPort  int            `json:"adminPort"`
	CreatedAt  time.Time      `json:"createdAt"`
	StartedAt  *time.Time     `json:"startedAt,omitempty"`
	Resources  ResourceSummary `json:"resources"`
	DataDir    string         `json:"dataDir"`
}

// ResourceSummary contains counts of resources in an instance
type ResourceSummary struct {
	Clusters  int `json:"clusters"`
	Services  int `json:"services"`
	Tasks     int `json:"tasks"`
}

// CreateInstanceRequest represents a request to create a new instance
type CreateInstanceRequest struct {
	Name          string `json:"name"`
	APIPort       int    `json:"apiPort"`
	AdminPort     int    `json:"adminPort"`
	NoLocalStack  bool   `json:"noLocalStack"`
	NoTraefik     bool   `json:"noTraefik"`
	DevMode       bool   `json:"devMode"`
}

// InstanceManager provides instance management operations
type InstanceManager struct {
	manager *kubernetes.K3dClusterManager
}

// NewInstanceManager creates a new instance manager
func NewInstanceManager() (*InstanceManager, error) {
	manager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster manager: %w", err)
	}
	
	return &InstanceManager{
		manager: manager,
	}, nil
}

// ListInstances returns all KECS instances
func (im *InstanceManager) ListInstances(ctx context.Context) ([]Instance, error) {
	clusters, err := im.manager.ListClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	
	instances := make([]Instance, 0, len(clusters))
	for _, clusterName := range clusters {
		instance, err := im.GetInstance(ctx, clusterName)
		if err != nil {
			// Log error but continue with other instances
			continue
		}
		instances = append(instances, *instance)
	}
	
	return instances, nil
}

// GetInstance returns details about a specific instance
func (im *InstanceManager) GetInstance(ctx context.Context, name string) (*Instance, error) {
	exists, err := im.manager.ClusterExists(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check cluster existence: %w", err)
	}
	
	instance := &Instance{
		Name:   name,
		Status: InstanceNotFound,
	}
	
	if !exists {
		return instance, nil
	}
	
	// Check if running
	running, err := im.manager.IsClusterRunning(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check cluster status: %w", err)
	}
	
	instance.Running = running
	if running {
		instance.Status = InstanceRunning
		now := time.Now()
		instance.StartedAt = &now // TODO: Get actual start time from k3d
	} else {
		instance.Status = InstanceStopped
	}
	
	// Check for data directory
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".kecs", "instances", name, "data")
	instance.DataDir = dataDir
	
	if _, err := os.Stat(dataDir); err == nil {
		instance.DataExists = true
	}
	
	// Get port mappings from container labels or config
	instance.APIPort = im.getInstanceAPIPort(name)
	instance.AdminPort = instance.APIPort + 1 // Admin port is typically API port + 1
	
	// TODO: Get resource counts if instance is running
	// This would require connecting to the instance API
	
	return instance, nil
}

// CreateInstance creates a new KECS instance
func (im *InstanceManager) CreateInstance(ctx context.Context, req CreateInstanceRequest) error {
	// This would typically call the start command logic
	// For TUI, we'll need to adapt the start.go logic to be callable programmatically
	return fmt.Errorf("not implemented yet - use CLI command for now")
}

// StartInstance starts a stopped instance
func (im *InstanceManager) StartInstance(ctx context.Context, name string) error {
	return im.manager.StartCluster(ctx, name)
}

// StopInstance stops a running instance
func (im *InstanceManager) StopInstance(ctx context.Context, name string) error {
	return im.manager.StopCluster(ctx, name)
}

// DestroyInstance destroys an instance
func (im *InstanceManager) DestroyInstance(ctx context.Context, name string, deleteData bool) error {
	// Delete the cluster
	if err := im.manager.DeleteCluster(ctx, name); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}
	
	// Delete data if requested
	if deleteData {
		home, _ := os.UserHomeDir()
		dataDir := filepath.Join(home, ".kecs", "instances", name, "data")
		
		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("failed to delete data directory: %w", err)
		}
		
		// Also delete the instance directory if it's empty
		instanceDir := filepath.Join(home, ".kecs", "instances", name)
		os.Remove(instanceDir) // This will only succeed if directory is empty
	}
	
	return nil
}

// GetCurrentInstance returns the name of the currently connected instance
func (im *InstanceManager) GetCurrentInstance(endpoint string) string {
	// Extract instance name from endpoint if it's a local instance
	// Format: http://localhost:PORT or http://127.0.0.1:PORT
	if strings.Contains(endpoint, "localhost") || strings.Contains(endpoint, "127.0.0.1") {
		// Extract port from endpoint
		parts := strings.Split(endpoint, ":")
		if len(parts) >= 3 {
			portStr := strings.TrimSuffix(parts[2], "/")
			port, err := strconv.Atoi(portStr)
			if err == nil {
				// Find instance by port
				ctx := context.Background()
				instances, err := im.ListInstances(ctx)
				if err == nil {
					for _, instance := range instances {
						if instance.Status == InstanceRunning && instance.APIPort == port {
							return instance.Name
						}
					}
				}
			}
		}
		return "unknown"
	}
	
	return "remote"
}

// getInstanceAPIPort returns the API port for a given instance
func (im *InstanceManager) getInstanceAPIPort(instanceName string) int {
	// TODO: Check if we have a config file for this instance
	// home, _ := os.UserHomeDir()
	// configFile := filepath.Join(home, ".kecs", "instances", instanceName, "config.yaml")
	
	// For now, use a simple port allocation scheme
	// This should be replaced with actual port detection from k3d
	switch instanceName {
	case "dev":
		return 8080
	case "staging":
		return 8090
	case "test":
		return 8100
	case "local":
		return 8110
	case "prod":
		return 8200
	default:
		// Default port allocation: hash instance name to get a consistent port
		hash := 0
		for _, c := range instanceName {
			hash = hash*31 + int(c)
		}
		// Map to port range 8300-8999
		return 8300 + (hash % 700)
	}
}