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
	"encoding/json"
	"fmt"
	"strings"
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
				Name:       inst.Name,
				Status:     inst.Status,
				Clusters:   inst.Clusters,
				Services:   inst.Services,
				Tasks:      inst.Tasks,
				APIPort:    inst.APIPort,
				AdminPort:  inst.AdminPort,
				LocalStack: inst.LocalStack,
				Traefik:    inst.Traefik,
				Age:        time.Since(inst.CreatedAt),
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
						// Extract region from ClusterArn (format: arn:aws:ecs:region:account:cluster/name)
						region := extractRegionFromArn(cluster.ClusterArn)
						tuiClusters[i] = Cluster{
							Name:     cluster.ClusterName,
							Status:   cluster.Status,
							Region:   region,
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
							ID:     extractTaskID(task.TaskArn),
							Status: task.LastStatus,
							CPU:    parseCPU(task.Cpu),
							Memory: task.Memory,
							Age:    time.Since(task.CreatedAt),
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

// loadTaskLogsCmd loads logs for the selected task
func (m Model) loadTaskLogsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.selectedTask == "" {
			return errMsg{err: fmt.Errorf("no task selected")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get logs from API
		logs, err := m.apiClient.GetTaskLogs(ctx, m.selectedInstance, m.selectedCluster, m.selectedTask, 100)
		if err != nil {
			// If API fails, use mock data
			logs = generateMockLogs(m.selectedTask)
		}

		// Convert API logs to TUI logs
		tuiLogs := make([]LogEntry, len(logs))
		for i, log := range logs {
			tuiLogs[i] = LogEntry{
				Timestamp: log.Timestamp,
				Level:     log.Level,
				Message:   log.Message,
			}
		}

		return logsLoadedMsg{
			logs: tuiLogs,
		}
	}
}

// generateMockLogs generates mock log entries for testing
func generateMockLogs(taskID string) []api.LogEntry {
	now := time.Now()
	return []api.LogEntry{
		{
			Timestamp: now.Add(-5 * time.Minute),
			Level:     "INFO",
			Message:   fmt.Sprintf("Starting task %s", taskID),
		},
		{
			Timestamp: now.Add(-4 * time.Minute),
			Level:     "INFO",
			Message:   "Pulling container image...",
		},
		{
			Timestamp: now.Add(-3 * time.Minute),
			Level:     "INFO",
			Message:   "Container started successfully",
		},
		{
			Timestamp: now.Add(-2 * time.Minute),
			Level:     "INFO",
			Message:   "Health check passed",
		},
		{
			Timestamp: now.Add(-1 * time.Minute),
			Level:     "WARN",
			Message:   "High memory usage detected (85%)",
		},
		{
			Timestamp: now.Add(-30 * time.Second),
			Level:     "INFO",
			Message:   "Request received: GET /health",
		},
		{
			Timestamp: now.Add(-10 * time.Second),
			Level:     "ERROR",
			Message:   "Failed to connect to database: timeout",
		},
		{
			Timestamp: now.Add(-5 * time.Second),
			Level:     "INFO",
			Message:   "Retrying database connection...",
		},
		{
			Timestamp: now,
			Level:     "INFO",
			Message:   "Database connection restored",
		},
	}
}

// createInstanceCmd creates a new instance via API
func (m Model) createInstanceCmd(opts api.CreateInstanceOptions) tea.Cmd {
	if m.useMockData {
		// Mock creation with steps simulation
		return func() tea.Msg {
			// Simulate step-by-step creation
			time.Sleep(500 * time.Millisecond)
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
		// Increase timeout to 3 minutes for LocalStack and other components to start
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		instance, err := m.apiClient.CreateInstance(ctx, opts)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to create instance: %w", err)}
		}

		return instanceCreatedMsg{
			instance: Instance{
				Name:       instance.Name,
				Status:     instance.Status,
				Clusters:   instance.Clusters,
				Services:   instance.Services,
				Tasks:      instance.Tasks,
				APIPort:    instance.APIPort,
				AdminPort:  instance.AdminPort,
				LocalStack: instance.LocalStack,
				Traefik:    instance.Traefik,
				Age:        time.Since(instance.CreatedAt),
			},
		}
	}
}

// monitorInstanceCreation monitors the creation progress and sends status updates
func (m Model) monitorInstanceCreation(instanceName string, hasLocalStack bool) tea.Cmd {
	return func() tea.Msg {
		// Start a background monitoring goroutine
		startTime := time.Now()

		// Define initial steps
		steps := []CreationStep{
			{Name: "Creating k3d cluster", Status: "running"},
			{Name: "Deploying control plane", Status: "pending"},
		}

		if hasLocalStack {
			steps = append(steps, CreationStep{Name: "Starting LocalStack", Status: "pending"})
		}
		steps = append(steps, CreationStep{Name: "Finalizing", Status: "pending"})

		// Return initial status immediately
		elapsed := time.Since(startTime)
		return instanceCreationStatusMsg{
			steps:   steps,
			elapsed: fmt.Sprintf("%.0fs", elapsed.Seconds()),
		}
	}
}

// updateCreationStatusCmd generates periodic updates for instance creation status
func (m Model) updateCreationStatusCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return creationStatusTickMsg(t)
	})
}

// checkCreationStatusCmd checks the actual creation status from API
func (m Model) checkCreationStatusCmd(instanceName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		status, err := m.apiClient.GetInstanceCreationStatus(ctx, instanceName)
		if err != nil {
			// Ignore errors during status check
			return nil
		}

		if status == nil {
			// No status means creation is complete or not started
			return nil
		}

		return actualCreationStatusMsg{
			instanceName: instanceName,
			status:       status,
		}
	}
}

// creationStatusTickMsg is sent every second during instance creation
type creationStatusTickMsg time.Time

// actualCreationStatusMsg contains actual status from API
type actualCreationStatusMsg struct {
	instanceName string
	status       *api.CreationStatus
}

// closeFormMsg is sent to close the instance creation form
type closeFormMsg struct{}

// Message types for API operations
type dataLoadedMsg struct {
	instances []Instance
	clusters  []Cluster
	services  []Service
	tasks     []Task
}

type logsLoadedMsg struct {
	logs []LogEntry
}

type instanceCreatedMsg struct {
	instance Instance
}

// Instance creation status messages
type instanceCreationStatusMsg struct {
	steps   []CreationStep
	elapsed string
}

type instanceCreationTimeoutMsg struct {
	elapsed string
}

type instanceCreationContinueMsg struct {
	continueWaiting bool // true to continue, false to abort
}

type errMsg struct {
	err error
}

// instanceStatusUpdateMsg is sent when instance statuses are updated
type instanceStatusUpdateMsg struct {
	instances []Instance
}

// instanceDeletingMsg indicates that an instance deletion is in progress
type instanceDeletingMsg struct {
	name string
}

// instanceDeletedMsg indicates that an instance deletion has completed
type instanceDeletedMsg struct {
	name string
	err  error
}

// Helper functions
func parseCPU(cpuStr string) float64 {
	// Parse CPU string to float64
	// Example: "256" -> 256.0
	var cpu float64
	fmt.Sscanf(cpuStr, "%f", &cpu)
	return cpu
}

func extractRegionFromArn(arn string) string {
	// Extract region from ARN
	// Example: arn:aws:ecs:us-east-1:123456789012:cluster/default
	// Returns: us-east-1
	parts := strings.Split(arn, ":")
	if len(parts) >= 4 {
		return parts[3]
	}
	return "unknown"
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
				Name:       inst.Name,
				Status:     inst.Status,
				Clusters:   inst.Clusters,
				Services:   inst.Services,
				Tasks:      inst.Tasks,
				APIPort:    inst.APIPort,
				AdminPort:  inst.AdminPort,
				LocalStack: inst.LocalStack,
				Traefik:    inst.Traefik,
				Age:        time.Since(inst.CreatedAt),
			}
		}

		return instanceStatusUpdateMsg{instances: tuiInstances}
	}
}

// loadTaskDefinitionFamiliesCmd loads task definition families
func (m Model) loadTaskDefinitionFamiliesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Get families list
		families, err := m.apiClient.ListTaskDefinitionFamilies(ctx, m.selectedInstance)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to list task definition families: %w", err)}
		}

		// Get details for each family
		taskDefFamilies := make([]TaskDefinitionFamily, 0, len(families))
		for _, family := range families {
			// Get revisions for this family
			revisions, err := m.apiClient.ListTaskDefinitionRevisions(ctx, m.selectedInstance, family)
			if err != nil {
				continue // Skip on error
			}

			if len(revisions) == 0 {
				continue
			}

			// Count active revisions
			activeCount := 0
			for _, rev := range revisions {
				if rev.Status == "ACTIVE" {
					activeCount++
				}
			}

			// Latest revision is the first one (assuming sorted by revision desc)
			latestRevision := revisions[0].Revision
			lastUpdated := revisions[0].CreatedAt

			taskDefFamilies = append(taskDefFamilies, TaskDefinitionFamily{
				Family:         family,
				LatestRevision: latestRevision,
				ActiveCount:    activeCount,
				TotalCount:     len(revisions),
				LastUpdated:    lastUpdated,
			})
		}

		return taskDefFamiliesLoadedMsg{families: taskDefFamilies}
	}
}

// loadTaskDefinitionRevisionsCmd loads revisions for a task definition family
func (m Model) loadTaskDefinitionRevisionsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		apiRevisions, err := m.apiClient.ListTaskDefinitionRevisions(ctx, m.selectedInstance, m.selectedFamily)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to list task definition revisions: %w", err)}
		}

		// Convert API revisions to TUI revisions
		tuiRevisions := make([]TaskDefinitionRevision, len(apiRevisions))
		for i, rev := range apiRevisions {
			tuiRevisions[i] = TaskDefinitionRevision{
				Family:    rev.Family,
				Revision:  rev.Revision,
				Status:    rev.Status,
				CPU:       rev.Cpu,
				Memory:    rev.Memory,
				CreatedAt: rev.CreatedAt,
			}
		}

		return taskDefRevisionsLoadedMsg{revisions: tuiRevisions}
	}
}

// startInstanceCmd starts a stopped instance asynchronously
func (m Model) startInstanceCmd(instanceName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := m.apiClient.StartInstance(ctx, instanceName)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to start instance %s: %w", instanceName, err)}
		}

		// Wait a bit before refreshing to allow the instance to transition
		time.Sleep(2 * time.Second)
		return refreshInstancesMsg{}
	}
}

// stopInstanceCmd stops a running instance asynchronously
func (m Model) stopInstanceCmd(instanceName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := m.apiClient.StopInstance(ctx, instanceName)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to stop instance %s: %w", instanceName, err)}
		}

		// Wait a bit before refreshing to allow the instance to transition
		time.Sleep(2 * time.Second)
		return refreshInstancesMsg{}
	}
}

// Message types for task definition operations
type taskDefFamiliesLoadedMsg struct {
	families []TaskDefinitionFamily
}

type taskDefRevisionsLoadedMsg struct {
	revisions []TaskDefinitionRevision
}

type taskDefJSONLoadedMsg struct {
	revision int
	json     string
}

// loadTaskDefinitionJSONCmd loads the JSON for a specific task definition revision
func (m Model) loadTaskDefinitionJSONCmd(taskDefArn string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		taskDef, err := m.apiClient.DescribeTaskDefinition(ctx, m.selectedInstance, taskDefArn)
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to describe task definition: %w", err)}
		}

		// Convert to JSON
		jsonBytes, err := json.MarshalIndent(taskDef, "", "  ")
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to marshal task definition: %w", err)}
		}

		// Find revision number from ARN
		revision := 0
		for _, rev := range m.taskDefRevisions {
			if rev.Family+":"+fmt.Sprintf("%d", rev.Revision) == taskDefArn ||
				fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s:%d",
					"us-east-1", "123456789012", rev.Family, rev.Revision) == taskDefArn {
				revision = rev.Revision
				break
			}
		}

		return taskDefJSONLoadedMsg{
			revision: revision,
			json:     string(jsonBytes),
		}
	}
}

// viewTaskLogsCmd opens the log viewer for a specific task
func (m Model) viewTaskLogsCmd(taskArn string, containerName string) tea.Cmd {
	return func() tea.Msg {
		// Create log API client
		var apiClient LogAPIClient
		if m.useMockData {
			apiClient = NewMockLogAPIClient()
		} else {
			// Use real API client with the correct endpoint
			// Logs API is now on admin port (8081)
			var adminPort int = 8081 // default admin port
			for _, inst := range m.instances {
				if inst.Name == m.selectedInstance {
					adminPort = inst.AdminPort
					break
				}
			}
			baseURL := fmt.Sprintf("http://localhost:%d", adminPort)
			apiClient = NewLogAPIClient(baseURL)
		}

		// Create log viewer
		logViewer := NewLogViewer(taskArn, containerName, apiClient)

		return logViewerCreatedMsg{
			viewer:    logViewer,
			taskArn:   taskArn,
			container: containerName,
		}
	}
}

// logViewerCreatedMsg is sent when a log viewer is created
type logViewerCreatedMsg struct {
	viewer    LogViewerModel
	taskArn   string
	container string
}

// deleteInstanceCmd initiates instance deletion
func (m Model) deleteInstanceCmd(instanceName string) tea.Cmd {
	return func() tea.Msg {
		// Start deleting
		return instanceDeletingMsg{name: instanceName}
	}
}

// performInstanceDeletionCmd performs the actual deletion
func (m Model) performInstanceDeletionCmd(instanceName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.apiClient.DeleteInstance(ctx, instanceName)
		return instanceDeletedMsg{name: instanceName, err: err}
	}
}

// createClusterCmd creates an ECS cluster via API
func (m Model) createClusterCmd(clusterName, region string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Note: Region is specified in the form but not yet used by the API
		// In the future, this could be used to configure region-specific settings
		cluster, err := m.apiClient.CreateCluster(ctx, m.selectedInstance, clusterName)

		if err != nil {
			return clusterCreatedMsg{
				clusterName: clusterName,
				region:      region,
				err:         err,
			}
		}

		// Success
		return clusterCreatedMsg{
			clusterName: cluster.ClusterName,
			region:      region,
			err:         nil,
		}
	}
}
