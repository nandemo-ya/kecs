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

package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/mock"
)

// loadMockDataCmd loads mock data based on current selections
func (m Model) loadMockDataCmd() tea.Cmd {
	if m.useMockData {
		return mock.LoadAllData(m.selectedInstance, m.selectedCluster, m.selectedService, m.selectedTask)
	}
	
	// Use API client to load data
	return m.loadDataFromAPI()
}

// loadDataFromAPI loads data from the API
func (m Model) loadDataFromAPI() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Load instances
		instances, err := m.apiClient.ListInstances(ctx)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to list instances: %w", err)}
		}

		// Convert API instances to TUI instances
		tuiInstances := make([]Instance, len(instances))
		for i, inst := range instances {
			tuiInstances[i] = Instance{
				Name:     inst.Name,
				Status:   inst.Status,
				Clusters: inst.Clusters,
				Services: inst.Services,
				Tasks:    inst.Tasks,
				APIPort:  inst.APIPort,
				Age:      time.Since(inst.CreatedAt),
			}
		}

		// Load clusters if instance is selected
		var tuiClusters []Cluster
		if m.selectedInstance != "" {
			clusterArns, err := m.apiClient.ListClusters(ctx, m.selectedInstance)
			if err == nil && len(clusterArns) > 0 {
				clusters, err := m.apiClient.DescribeClusters(ctx, m.selectedInstance, clusterArns)
				if err == nil {
					tuiClusters = make([]Cluster, len(clusters))
					for i, cluster := range clusters {
						tuiClusters[i] = Cluster{
							Name:     cluster.ClusterName,
							Status:   cluster.Status,
							Services: cluster.ActiveServicesCount,
							Tasks:    cluster.RunningTasksCount,
							Age:      24 * time.Hour, // Mock age for now
						}
					}
				}
			}
		}

		// Load services if cluster is selected
		var tuiServices []Service
		if m.selectedInstance != "" && m.selectedCluster != "" {
			serviceArns, err := m.apiClient.ListServices(ctx, m.selectedInstance, m.selectedCluster)
			if err == nil && len(serviceArns) > 0 {
				services, err := m.apiClient.DescribeServices(ctx, m.selectedInstance, m.selectedCluster, serviceArns)
				if err == nil {
					tuiServices = make([]Service, len(services))
					for i, service := range services {
						tuiServices[i] = Service{
							Name:    service.ServiceName,
							Status:  service.Status,
							Desired: service.DesiredCount,
							Running: service.RunningCount,
							Pending: service.PendingCount,
							TaskDef: service.TaskDefinition,
							Age:     time.Since(service.CreatedAt),
						}
					}
				}
			}
		}

		// Load tasks if service is selected
		var tuiTasks []Task
		if m.selectedInstance != "" && m.selectedCluster != "" && m.selectedService != "" {
			taskArns, err := m.apiClient.ListTasks(ctx, m.selectedInstance, m.selectedCluster, m.selectedService)
			if err == nil && len(taskArns) > 0 {
				tasks, err := m.apiClient.DescribeTasks(ctx, m.selectedInstance, m.selectedCluster, taskArns)
				if err == nil {
					tuiTasks = make([]Task, len(tasks))
					for i, task := range tasks {
						tuiTasks[i] = Task{
							ID:         extractTaskID(task.TaskArn),
							Status:     task.LastStatus,
							CPU:        parseCPU(task.Cpu),
							Memory:     task.Memory,
							Age:        time.Since(task.CreatedAt),
						}
					}
				}
			}
		}

		return dataLoadedMsg{
			instances: tuiInstances,
			clusters:  tuiClusters,
			services:  tuiServices,
			tasks:     tuiTasks,
		}
	}
}

// createInstanceCmd creates a new instance via API
func (m Model) createInstanceCmd(opts api.CreateInstanceOptions) tea.Cmd {
	if m.useMockData {
		// Mock creation
		return func() tea.Msg {
			time.Sleep(1 * time.Second) // Simulate API delay
			return instanceCreatedMsg{
				instance: Instance{
					Name:     opts.Name,
					Status:   "pending",
					Clusters: 0,
					Services: 0,
					Tasks:    0,
					APIPort:  opts.APIPort,
					Age:      0,
				},
			}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		instance, err := m.apiClient.CreateInstance(ctx, opts)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to create instance: %w", err)}
		}

		return instanceCreatedMsg{
			instance: Instance{
				Name:     instance.Name,
				Status:   instance.Status,
				Clusters: instance.Clusters,
				Services: instance.Services,
				Tasks:    instance.Tasks,
				APIPort:  instance.APIPort,
				Age:      time.Since(instance.CreatedAt),
			},
		}
	}
}

// Message types for API operations
type dataLoadedMsg struct {
	instances []Instance
	clusters  []Cluster
	services  []Service
	tasks     []Task
}

type instanceCreatedMsg struct {
	instance Instance
}

type errMsg struct {
	err error
}

// instanceStatusUpdateMsg is sent when instance statuses are updated
type instanceStatusUpdateMsg struct {
	instances []Instance
}

// Helper functions
func parseCPU(cpuStr string) float64 {
	// Parse CPU string to float64
	// Example: "256" -> 256.0
	var cpu float64
	fmt.Sscanf(cpuStr, "%f", &cpu)
	return cpu
}

func extractTaskID(taskArn string) string {
	// Extract task ID from ARN
	// Example: arn:aws:ecs:us-east-1:123456789012:task/default/1234567890123456789
	// Returns: 1234567890123456789
	parts := make([]string, 0)
	current := ""
	for _, char := range taskArn {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return taskArn
}

// updateInstanceStatusCmd updates the status of all instances
func (m Model) updateInstanceStatusCmd() tea.Cmd {
	if m.useMockData {
		// For mock data, simulate status changes
		return func() tea.Msg {
			// No status changes in mock mode
			return instanceStatusUpdateMsg{instances: m.instances}
		}
	}
	
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		instances, err := m.apiClient.ListInstances(ctx)
		if err != nil {
			// Don't show error for status updates, just return current state
			return instanceStatusUpdateMsg{instances: m.instances}
		}
		
		// Convert API instances to TUI instances
		tuiInstances := make([]Instance, len(instances))
		for i, inst := range instances {
			// Check health status if instance is running
			if inst.Status == "running" {
				err := m.apiClient.HealthCheck(ctx, inst.Name)
				if err != nil {
					inst.Status = "unhealthy"
				}
			}
			
			tuiInstances[i] = Instance{
				Name:     inst.Name,
				Status:   inst.Status,
				Clusters: inst.Clusters,
				Services: inst.Services,
				Tasks:    inst.Tasks,
				APIPort:  inst.APIPort,
				Age:      time.Since(inst.CreatedAt),
			}
		}
		
		return instanceStatusUpdateMsg{instances: tuiInstances}
	}
}