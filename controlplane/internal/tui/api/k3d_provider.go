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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/nandemo-ya/kecs/controlplane/internal/instance"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
)

// K3dInstanceProvider provides instance information directly from k3d clusters
type K3dInstanceProvider struct {
	k3dManager      *kubernetes.K3dClusterManager
	dockerClient    *client.Client
	instanceManager *instance.Manager
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

	// Create instance manager for start/stop operations
	instanceManager, err := instance.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create instance manager: %w", err)
	}

	return &K3dInstanceProvider{
		k3dManager:      k3dManager,
		dockerClient:    dockerClient,
		instanceManager: instanceManager,
	}, nil
}

// ListInstances returns all KECS instances from k3d clusters
func (p *K3dInstanceProvider) ListInstances(ctx context.Context) ([]Instance, error) {
	// Get all KECS clusters from k3d
	clusters, err := p.k3dManager.ListClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list k3d clusters: %w", err)
	}

	var instances []Instance
	for _, cluster := range clusters {
		instance, err := p.getInstanceInfo(ctx, cluster.Name)
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
	inst := Instance{
		Name:      name,
		Status:    "Unknown",
		Clusters:  0,
		Services:  0,
		Tasks:     0,
		APIPort:   8080,
		CreatedAt: time.Now(), // Default value
	}

	// Try to load saved configuration
	if config, err := instance.LoadInstanceConfig(name); err == nil {
		inst.APIPort = config.APIPort
		inst.AdminPort = config.AdminPort
		inst.LocalStack = config.LocalStack
		inst.Traefik = config.Traefik
		inst.DevMode = config.DevMode
		inst.CreatedAt = config.CreatedAt
	}

	// Check if cluster is running
	running, err := p.k3dManager.IsClusterRunning(ctx, name)
	if err != nil {
		return inst, err
	}

	if running {
		inst.Status = "Running"
	} else {
		inst.Status = "Stopped"
	}

	// Get container info for more details
	containerName := fmt.Sprintf("kecs-%s", name)
	containerInfo, err := p.getContainerInfo(ctx, containerName)
	if err == nil && containerInfo != nil {
		// Extract port mapping
		for _, port := range containerInfo.Ports {
			if port.PrivatePort == 8080 {
				inst.APIPort = int(port.PublicPort)
				break
			}
		}

		// Get creation time
		if containerInfo.Created > 0 {
			inst.CreatedAt = time.Unix(containerInfo.Created, 0)
		}

		// Update status based on container state
		switch containerInfo.State {
		case "running":
			inst.Status = "Running"
		case "exited":
			inst.Status = "Stopped"
		case "created":
			inst.Status = "Created"
		default:
			inst.Status = strings.Title(containerInfo.State)
		}
	}

	// If instance is running, try to get resource counts from API
	if inst.Status == "Running" && inst.APIPort > 0 {
		// Get cluster, service, and task counts from the instance's API
		clusters, services, tasks := p.getInstanceCounts(inst.APIPort)
		inst.Clusters = clusters
		inst.Services = services
		inst.Tasks = tasks
	}

	return inst, nil
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

// CreateInstance creates a new KECS instance using instance.Manager
func (p *K3dInstanceProvider) CreateInstance(ctx context.Context, opts CreateInstanceOptions) (*Instance, error) {
	// Convert TUI options to instance.StartOptions
	startOpts := instance.StartOptions{
		InstanceName: opts.Name,
		ApiPort:      opts.APIPort,
		AdminPort:    opts.AdminPort,
		NoLocalStack: !opts.LocalStack,
		NoTraefik:    !opts.Traefik,
		DevMode:      opts.DevMode,
	}

	// Use the instance manager to start the instance
	if err := p.instanceManager.Start(ctx, startOpts); err != nil {
		return nil, fmt.Errorf("failed to start instance: %w", err)
	}

	// Return the created instance info
	inst, err := p.getInstanceInfo(ctx, opts.Name)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// GetInstance retrieves a specific KECS instance information
func (p *K3dInstanceProvider) GetInstance(ctx context.Context, name string) (*Instance, error) {
	// Check if the instance exists
	exists, err := p.k3dManager.ClusterExists(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check instance existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("instance not found: %s", name)
	}

	inst, err := p.getInstanceInfo(ctx, name)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// GetInstanceLogs retrieves logs from the KECS instance container
func (p *K3dInstanceProvider) GetInstanceLogs(ctx context.Context, name string, follow bool) (<-chan LogEntry, error) {
	// Get container name
	containerName := fmt.Sprintf("kecs-%s", name)

	// Find container ID
	containers, err := p.dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containerID string
	for _, c := range containers {
		for _, n := range c.Names {
			if strings.TrimPrefix(n, "/") == containerName {
				containerID = c.ID
				break
			}
		}
		if containerID != "" {
			break
		}
	}

	if containerID == "" {
		return nil, fmt.Errorf("container %s not found", containerName)
	}

	// Create log channel
	logChan := make(chan LogEntry, 100)

	// Start goroutine to read logs
	go func() {
		defer close(logChan)

		options := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     follow,
			Timestamps: true,
		}

		reader, err := p.dockerClient.ContainerLogs(ctx, containerID, options)
		if err != nil {
			logChan <- LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to get logs: %v", err),
			}
			return
		}
		defer reader.Close()

		// Parse Docker multiplexed stream
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()

			// Parse timestamp and message
			// Docker log format with timestamps: "2025-01-08T10:30:45.123456789Z message"
			parts := strings.SplitN(line, " ", 2)

			var timestamp time.Time
			var message string

			if len(parts) >= 2 {
				// Try to parse timestamp
				if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
					timestamp = t
					message = parts[1]
				} else {
					// If parsing fails, use current time and whole line as message
					timestamp = time.Now()
					message = line
				}
			} else {
				timestamp = time.Now()
				message = line
			}

			// Determine log level from message content
			level := "INFO"
			lowerMsg := strings.ToLower(message)
			if strings.Contains(lowerMsg, "error") || strings.Contains(lowerMsg, "failed") {
				level = "ERROR"
			} else if strings.Contains(lowerMsg, "warn") {
				level = "WARN"
			} else if strings.Contains(lowerMsg, "debug") {
				level = "DEBUG"
			}

			logChan <- LogEntry{
				Timestamp: timestamp,
				Level:     level,
				Message:   message,
			}
		}

		if err := scanner.Err(); err != nil {
			logChan <- LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Error reading logs: %v", err),
			}
		}
	}()

	return logChan, nil
}

// DeleteInstance deletes a KECS instance
func (p *K3dInstanceProvider) DeleteInstance(ctx context.Context, name string) error {
	// Use instance manager to destroy the instance (without deleting data)
	return p.instanceManager.Destroy(ctx, name, false)
}

// GetInstanceCreationStatus returns the creation status for an instance
func (p *K3dInstanceProvider) GetInstanceCreationStatus(ctx context.Context, name string) (*CreationStatus, error) {
	// Get status from the local instance manager
	status := p.instanceManager.GetCreationStatus(name)
	if status == nil {
		// If no creation status, check if instance exists
		exists, err := p.k3dManager.ClusterExists(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to check instance existence: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("instance not found: %s", name)
		}

		// Instance exists but no creation status - it's already created
		return &CreationStatus{
			Step:    "Completed",
			Status:  "done",
			Message: "Instance is ready",
		}, nil
	}

	// Convert from instance.CreationStatus to api.CreationStatus
	return &CreationStatus{
		Step:    status.Step,
		Status:  status.Status,
		Message: status.Message,
	}, nil
}

// getInstanceCounts retrieves cluster, service, and task counts from an instance's API
func (p *K3dInstanceProvider) getInstanceCounts(apiPort int) (clusters, services, tasks int) {
	// Call ListClusters API
	url := fmt.Sprintf("http://localhost:%d/v1/ListClusters", apiPort)

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		return 0, 0, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, 0, 0
	}

	// Parse cluster response
	if clusterArns, ok := result["clusterArns"].([]interface{}); ok {
		clusters = len(clusterArns)

		// For each cluster, get services and tasks
		for _, arnInterface := range clusterArns {
			if arn, ok := arnInterface.(string); ok {
				// Extract cluster name from ARN
				clusterName := extractClusterName(arn)

				// Get services count
				servicesURL := fmt.Sprintf("http://localhost:%d/v1/ListServices", apiPort)
				servicesBody := fmt.Sprintf(`{"cluster":"%s"}`, clusterName)
				servicesResp, err := client.Post(servicesURL, "application/json", strings.NewReader(servicesBody))
				if err == nil && servicesResp.StatusCode == http.StatusOK {
					var servicesResult map[string]interface{}
					if err := json.NewDecoder(servicesResp.Body).Decode(&servicesResult); err == nil {
						if serviceArns, ok := servicesResult["serviceArns"].([]interface{}); ok {
							services += len(serviceArns)
						}
					}
					servicesResp.Body.Close()
				}

				// Get tasks count
				tasksURL := fmt.Sprintf("http://localhost:%d/v1/ListTasks", apiPort)
				tasksBody := fmt.Sprintf(`{"cluster":"%s"}`, clusterName)
				tasksResp, err := client.Post(tasksURL, "application/json", strings.NewReader(tasksBody))
				if err == nil && tasksResp.StatusCode == http.StatusOK {
					var tasksResult map[string]interface{}
					if err := json.NewDecoder(tasksResp.Body).Decode(&tasksResult); err == nil {
						if taskArns, ok := tasksResult["taskArns"].([]interface{}); ok {
							tasks += len(taskArns)
						}
					}
					tasksResp.Body.Close()
				}
			}
		}
	}

	return clusters, services, tasks
}

// extractClusterName extracts the cluster name from an ARN
func extractClusterName(arn string) string {
	// ARN format: arn:aws:ecs:region:account:cluster/name
	parts := strings.Split(arn, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	// If not in ARN format, assume it's already the cluster name
	return arn
}

// Close cleans up resources
func (p *K3dInstanceProvider) Close() error {
	if p.dockerClient != nil {
		return p.dockerClient.Close()
	}
	return nil
}
