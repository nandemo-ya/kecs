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
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
)

// K3dInstanceProvider provides instance information directly from k3d clusters
type K3dInstanceProvider struct {
	k3dManager    *kubernetes.K3dClusterManager
	dockerClient  *client.Client
}

// NewK3dInstanceProvider creates a new k3d-based instance provider
func NewK3dInstanceProvider() (*K3dInstanceProvider, error) {
	// Create k3d manager
	k3dManager, err := kubernetes.NewK3dClusterManager(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create k3d manager: %w", err)
	}

	// Create Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &K3dInstanceProvider{
		k3dManager:   k3dManager,
		dockerClient: dockerClient,
	}, nil
}

// ListInstances returns all KECS instances from k3d clusters
func (p *K3dInstanceProvider) ListInstances(ctx context.Context) ([]Instance, error) {
	// Get all KECS clusters from k3d
	clusterNames, err := p.k3dManager.ListClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list k3d clusters: %w", err)
	}

	var instances []Instance
	for _, name := range clusterNames {
		instance, err := p.getInstanceInfo(ctx, name)
		if err != nil {
			// Log error but continue with other instances
			continue
		}
		instances = append(instances, instance)
	}

	return instances, nil
}

// getInstanceInfo retrieves detailed information about a KECS instance
func (p *K3dInstanceProvider) getInstanceInfo(ctx context.Context, name string) (Instance, error) {
	instance := Instance{
		Name:      name,
		Status:    "Unknown",
		Clusters:  0,
		Services:  0,
		Tasks:     0,
		APIPort:   8080,
		CreatedAt: time.Now(), // Default value
	}

	// Check if cluster is running
	running, err := p.k3dManager.IsClusterRunning(ctx, name)
	if err != nil {
		return instance, err
	}

	if running {
		instance.Status = "Running"
	} else {
		instance.Status = "Stopped"
	}

	// Get container info for more details
	containerName := fmt.Sprintf("kecs-%s", name)
	containerInfo, err := p.getContainerInfo(ctx, containerName)
	if err == nil && containerInfo != nil {
		// Extract port mapping
		for _, port := range containerInfo.Ports {
			if port.PrivatePort == 8080 {
				instance.APIPort = int(port.PublicPort)
				break
			}
		}

		// Get creation time
		if containerInfo.Created > 0 {
			instance.CreatedAt = time.Unix(containerInfo.Created, 0)
		}

		// Update status based on container state
		switch containerInfo.State {
		case "running":
			instance.Status = "Running"
		case "exited":
			instance.Status = "Stopped"
		case "created":
			instance.Status = "Created"
		default:
			instance.Status = strings.Title(containerInfo.State)
		}
	}

	// If instance is running, try to get resource counts from API
	if instance.Status == "Running" {
		// This will be done through the API client if available
		// For now, we just return the basic info
	}

	return instance, nil
}

// getContainerInfo retrieves Docker container information
func (p *K3dInstanceProvider) getContainerInfo(ctx context.Context, containerName string) (*container.Summary, error) {
	containers, err := p.dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		for _, name := range container.Names {
			if strings.TrimPrefix(name, "/") == containerName {
				return &container, nil
			}
		}
	}

	return nil, fmt.Errorf("container %s not found", containerName)
}

// Close cleans up resources
func (p *K3dInstanceProvider) Close() error {
	if p.dockerClient != nil {
		return p.dockerClient.Close()
	}
	return nil
}