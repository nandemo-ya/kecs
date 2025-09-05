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
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// hasDependencies checks if any container has dependencies
func hasDependencies(containers []ContainerDetail) bool {
	for _, container := range containers {
		if len(container.DependsOn) > 0 {
			return true
		}
	}
	return false
}

// renderDependencyGraph renders a visual dependency graph
func (m Model) renderDependencyGraph(containers []ContainerDetail) string {
	graphStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginLeft(2)

	var lines []string
	lines = append(lines, lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true).
		Render("  Dependency Flow:"))
	lines = append(lines, "")

	// Simple approach: show each dependency relationship
	for _, container := range containers {
		if len(container.DependsOn) > 0 {
			containerStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("220"))

			for _, dep := range container.DependsOn {
				// Color code condition
				conditionColor := "245"
				switch dep.Condition {
				case "START":
					conditionColor = "220"
				case "COMPLETE":
					conditionColor = "214"
				case "SUCCESS":
					conditionColor = "82"
				case "HEALTHY":
					conditionColor = "46"
				}
				condStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color(conditionColor)).
					Italic(true)

				lines = append(lines, fmt.Sprintf("  %s ─→ %s %s",
					containerStyle.Render(dep.ContainerName),
					containerStyle.Render(container.Name),
					condStyle.Render(fmt.Sprintf("[%s]", dep.Condition))))
			}
		}
	}

	// If no dependencies exist, show containers without dependencies
	if len(lines) == 2 { // Only header and empty line
		containerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("220"))

		for _, container := range containers {
			lines = append(lines, fmt.Sprintf("  %s %s",
				containerStyle.Render(container.Name),
				lipgloss.NewStyle().
					Foreground(lipgloss.Color("245")).
					Italic(true).
					Render("(independent)")))
		}
	}

	return graphStyle.Render(strings.Join(lines, "\n"))
}

// TaskDetail represents detailed task information
type TaskDetail struct {
	// Basic Info
	TaskARN         string
	TaskDefinition  string
	ClusterARN      string
	ServiceName     string
	LaunchType      string
	PlatformVersion string

	// Status
	LastStatus    string
	DesiredStatus string
	HealthStatus  string
	StopCode      string
	StoppedReason string

	// Resources
	CPU    string
	Memory string

	// Network
	NetworkMode string
	IPs         []string
	DNSNames    []string

	// Timestamps
	CreatedAt  time.Time
	StartedAt  *time.Time
	StoppedAt  *time.Time
	StoppingAt *time.Time

	// Containers
	Containers []ContainerDetail
}

// ContainerDetail represents container information within a task
type ContainerDetail struct {
	Name            string
	Image           string
	Status          string
	ExitCode        *int
	Reason          string
	CPU             string
	Memory          string
	Essential       bool
	HealthStatus    string
	NetworkBindings []NetworkBinding
	DependsOn       []ContainerDependency // Dependencies on other containers
}

// ContainerDependency represents a dependency on another container
type ContainerDependency struct {
	ContainerName string
	Condition     string // START, COMPLETE, SUCCESS, HEALTHY
}

// NetworkBinding represents port mappings
type NetworkBinding struct {
	ContainerPort int
	HostPort      int
	Protocol      string
}

// renderTaskDescribe renders the task describe view
func (m Model) renderTaskDescribe() string {
	if m.selectedTaskDetail == nil {
		// Load task details
		return m.renderLoadingTaskDetails()
	}

	detail := m.selectedTaskDetail

	// Create sections with proper spacing
	sections := []string{
		m.renderTaskOverview(detail),
		m.renderTaskContainers(detail),
		m.renderTaskNetwork(detail),
		m.renderTaskTimestamps(detail),
	}

	// Join sections with more spacing
	content := strings.Join(sections, "\n\n")

	// Don't add padding here since it's now handled by the resource panel
	// content = contentStyle.Render(content)

	// Calculate available height for content within resource panel
	// Use a more conservative estimate for the resource panel context
	contentHeight := 20 // Default height for resource panel content
	if m.height > 10 {
		contentHeight = m.height - 10 // Leave space for navigation, header, footer
	}

	// Apply scrolling if needed
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Calculate max scroll position
	maxScroll := totalLines - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Clamp scroll position to valid range (read-only check)
	scrollPos := m.taskDescribeScroll
	if scrollPos < 0 {
		scrollPos = 0
	}
	if scrollPos > maxScroll {
		scrollPos = maxScroll
	}

	// Extract visible lines
	if totalLines > contentHeight {
		start := scrollPos
		end := start + contentHeight

		// Safety checks to prevent panic
		if start < 0 {
			start = 0
		}
		if start >= totalLines {
			start = totalLines - 1
			if start < 0 {
				start = 0
			}
		}
		if end > totalLines {
			end = totalLines
		}
		if end <= start {
			end = start + 1
			if end > totalLines {
				end = totalLines
			}
		}

		// Only join if we have valid indices
		if start < totalLines && end <= totalLines && start < end {
			content = strings.Join(lines[start:end], "\n")
		} else {
			// Fallback to show something
			if totalLines > 0 {
				content = strings.Join(lines[0:min(contentHeight, totalLines)], "\n")
			}
		}
	}

	// Return content only (no header/footer since it's in the resource panel now)
	return content
}

func (m Model) renderTaskOverview(detail *TaskDetail) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(16) // Fixed width for alignment

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	sectionStyle := lipgloss.NewStyle().
		MarginBottom(1)

	var lines []string
	lines = append(lines, titleStyle.Render("Overview"))
	lines = append(lines, "") // Empty line after title

	// Task Definition - extract family:revision from ARN
	taskDefDisplay := detail.TaskDefinition
	if taskDefDisplay != "" {
		// Try to extract family:revision from ARN format
		// Example: arn:aws:ecs:region:account:task-definition/family:revision
		parts := strings.Split(taskDefDisplay, "/")
		if len(parts) > 1 {
			taskDefDisplay = parts[len(parts)-1]
		}
	} else {
		taskDefDisplay = "-"
	}
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("Task Definition:"),
		valueStyle.Render(taskDefDisplay)))

	// Service
	if detail.ServiceName != "" {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("Service:"),
			valueStyle.Render(detail.ServiceName)))
	}

	// Launch Type
	if detail.LaunchType != "" {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("Launch Type:"),
			valueStyle.Render(detail.LaunchType)))
	} else {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("Launch Type:"),
			valueStyle.Render("FARGATE"))) // Default to FARGATE
	}

	// Resources
	cpu := detail.CPU
	if cpu == "" {
		cpu = "-"
	}
	memory := detail.Memory
	if memory == "" {
		memory = "-"
	}
	lines = append(lines, fmt.Sprintf("%s CPU: %s, Memory: %s",
		labelStyle.Render("Resources:"),
		valueStyle.Render(cpu),
		valueStyle.Render(memory)))

	// Health Status
	if detail.HealthStatus != "" {
		healthColor := "244"
		if detail.HealthStatus == "HEALTHY" {
			healthColor = "82"
		} else if detail.HealthStatus == "UNHEALTHY" {
			healthColor = "196"
		}
		healthStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(healthColor))
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("Health:"),
			healthStyle.Render(detail.HealthStatus)))
	}

	// Stopped Reason
	if detail.StoppedReason != "" {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("Stopped Reason:"),
			valueStyle.Render(detail.StoppedReason)))
	}

	return sectionStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) renderTaskContainers(detail *TaskDetail) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("237")).
		Padding(1).
		MarginBottom(1).
		MarginLeft(2) // Indent containers

	sectionStyle := lipgloss.NewStyle().
		MarginBottom(1)

	var sections []string
	sections = append(sections, titleStyle.Render(fmt.Sprintf("Containers (%d)", len(detail.Containers))))
	sections = append(sections, "") // Empty line after title

	// First, render dependency graph if there are dependencies
	if hasDependencies(detail.Containers) {
		sections = append(sections, m.renderDependencyGraph(detail.Containers))
		sections = append(sections, "") // Empty line after graph
	}

	for _, container := range detail.Containers {
		var lines []string

		// Container name and status
		statusColor := "244"
		switch strings.ToUpper(container.Status) {
		case "RUNNING":
			statusColor = "82"
		case "PENDING":
			statusColor = "226"
		case "STOPPED":
			statusColor = "196"
		}

		nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))

		header := fmt.Sprintf("%s [%s]",
			nameStyle.Render(container.Name),
			statusStyle.Render(container.Status))

		if container.Essential {
			header += " " + lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Render("(essential)")
		}

		lines = append(lines, header)
		lines = append(lines, "") // Empty line after header

		// Image
		if container.Image != "" {
			lines = append(lines, fmt.Sprintf("Image: %s", container.Image))
		} else {
			lines = append(lines, "Image: -")
		}

		// Resources
		cpu := container.CPU
		if cpu == "" {
			cpu = "-"
		}
		memory := container.Memory
		if memory == "" {
			memory = "-"
		}
		lines = append(lines, fmt.Sprintf("Resources: CPU=%s, Memory=%s", cpu, memory))

		// Exit code and reason
		if container.ExitCode != nil {
			exitColor := "82"
			if *container.ExitCode != 0 {
				exitColor = "196"
			}
			exitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(exitColor))
			lines = append(lines, fmt.Sprintf("Exit Code: %s",
				exitStyle.Render(fmt.Sprintf("%d", *container.ExitCode))))

			if container.Reason != "" {
				lines = append(lines, fmt.Sprintf("Reason: %s", container.Reason))
			}
		}

		// Dependencies
		if len(container.DependsOn) > 0 {
			var deps []string
			for _, dep := range container.DependsOn {
				conditionColor := "244"
				switch dep.Condition {
				case "START":
					conditionColor = "220"
				case "COMPLETE":
					conditionColor = "214"
				case "SUCCESS":
					conditionColor = "82"
				case "HEALTHY":
					conditionColor = "46"
				}
				condStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(conditionColor))
				deps = append(deps, fmt.Sprintf("%s (%s)",
					dep.ContainerName, condStyle.Render(dep.Condition)))
			}
			lines = append(lines, fmt.Sprintf("Depends On: %s", strings.Join(deps, ", ")))
		}

		// Network bindings
		if len(container.NetworkBindings) > 0 {
			var ports []string
			for _, binding := range container.NetworkBindings {
				ports = append(ports, fmt.Sprintf("%d:%d/%s",
					binding.HostPort, binding.ContainerPort, binding.Protocol))
			}
			lines = append(lines, fmt.Sprintf("Ports: %s", strings.Join(ports, ", ")))
		}

		sections = append(sections, containerStyle.Render(strings.Join(lines, "\n")))
	}

	return sectionStyle.Render(strings.Join(sections, "\n"))
}

func (m Model) renderTaskNetwork(detail *TaskDetail) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(16) // Fixed width for alignment

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	sectionStyle := lipgloss.NewStyle().
		MarginBottom(1)

	var lines []string
	lines = append(lines, titleStyle.Render("Network"))
	lines = append(lines, "") // Empty line after title

	// Network Mode
	networkMode := detail.NetworkMode
	if networkMode == "" {
		networkMode = "awsvpc"
	}
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("Mode:"),
		valueStyle.Render(networkMode)))

	// IPs
	if len(detail.IPs) > 0 {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("IPs:"),
			valueStyle.Render(strings.Join(detail.IPs, ", "))))
	} else {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("IPs:"),
			valueStyle.Render("-")))
	}

	// DNS Names
	if len(detail.DNSNames) > 0 {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("DNS Names:"),
			valueStyle.Render(strings.Join(detail.DNSNames, ", "))))
	}

	return sectionStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) renderTaskTimestamps(detail *TaskDetail) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(16) // Fixed width for alignment

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	durationStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)

	sectionStyle := lipgloss.NewStyle().
		MarginBottom(1)

	var lines []string
	lines = append(lines, titleStyle.Render("Timestamps"))
	lines = append(lines, "") // Empty line after title

	// Created
	lines = append(lines, fmt.Sprintf("%s %s %s",
		labelStyle.Render("Created:"),
		valueStyle.Render(detail.CreatedAt.Format("2006-01-02 15:04:05")),
		durationStyle.Render(fmt.Sprintf("(%s ago)", formatDuration(time.Since(detail.CreatedAt))))))

	// Started
	if detail.StartedAt != nil && !detail.StartedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("%s %s %s",
			labelStyle.Render("Started:"),
			valueStyle.Render(detail.StartedAt.Format("2006-01-02 15:04:05")),
			durationStyle.Render(fmt.Sprintf("(%s ago)", formatDuration(time.Since(*detail.StartedAt))))))
	}

	// Stopping
	if detail.StoppingAt != nil && !detail.StoppingAt.IsZero() {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("Stopping:"),
			valueStyle.Render(detail.StoppingAt.Format("2006-01-02 15:04:05"))))
	}

	// Stopped
	if detail.StoppedAt != nil && !detail.StoppedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("Stopped:"),
			valueStyle.Render(detail.StoppedAt.Format("2006-01-02 15:04:05"))))

		// Show duration if we have both started and stopped times
		if detail.StartedAt != nil && !detail.StartedAt.IsZero() {
			duration := detail.StoppedAt.Sub(*detail.StartedAt)
			lines = append(lines, fmt.Sprintf("%s %s",
				labelStyle.Render("Duration:"),
				durationStyle.Render(formatDuration(duration))))
		}
	}

	return sectionStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) renderLoadingTaskDetails() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Bold(true).
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	return loadingStyle.Render("Loading task details...")
}

// Commands for task describe view

func (m Model) loadTaskDetailsCmd() tea.Cmd {
	return func() tea.Msg {
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("loadTaskDetailsCmd", "Starting to load task details for task: %s, instance: %s, cluster: %s",
				m.selectedTask, m.selectedInstance, m.selectedCluster)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Debug: Log that we're loading task details
		// This will help us see if the function is being called
		// Get full task details from API
		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("loadTaskDetailsCmd", "Calling DescribeTasks API")
		}
		tasks, err := m.apiClient.DescribeTasks(ctx, m.selectedInstance, m.selectedCluster, []string{m.selectedTask})
		if err != nil || len(tasks) == 0 {
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("loadTaskDetailsCmd", "DescribeTasks failed: err=%v, tasks_len=%d", err, len(tasks))
			}
			return errMsg{err: fmt.Errorf("failed to load task details: %w", err)}
		}

		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("loadTaskDetailsCmd", "DescribeTasks succeeded, got %d tasks", len(tasks))
		}

		task := tasks[0]

		// Convert API task to TaskDetail
		detail := &TaskDetail{
			TaskARN:        task.TaskArn,
			TaskDefinition: task.TaskDefinitionArn,
			ClusterARN:     task.ClusterArn,
			LastStatus:     task.LastStatus,
			DesiredStatus:  task.DesiredStatus,
			HealthStatus:   task.HealthStatus,
			CPU:            task.Cpu,
			Memory:         task.Memory,
			CreatedAt:      task.CreatedAt,
		}

		// Parse service name if available
		if task.ServiceName != "" {
			detail.ServiceName = task.ServiceName
		}

		// Set timestamps
		if task.StartedAt != nil && !task.StartedAt.IsZero() {
			detail.StartedAt = task.StartedAt
		}
		if task.StoppedAt != nil && !task.StoppedAt.IsZero() {
			detail.StoppedAt = task.StoppedAt
		}

		// Get task definition to extract container details
		var containerDependencies map[string][]ContainerDependency
		var containerEssential map[string]bool
		var containerImages map[string]string
		var containerResources map[string]struct{ CPU, Memory string }

		// Initialize maps regardless of task definition fetch result
		containerDependencies = make(map[string][]ContainerDependency)
		containerEssential = make(map[string]bool)
		containerImages = make(map[string]string)
		containerResources = make(map[string]struct{ CPU, Memory string })

		if task.TaskDefinitionArn != "" {
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("loadTaskDetailsCmd", "Task has TaskDefinitionArn: %s", task.TaskDefinitionArn)
			}
			// First, try to extract family:revision from the ARN itself as fallback
			// Format: arn:aws:ecs:region:account-id:task-definition/family:revision
			if parts := strings.Split(task.TaskDefinitionArn, "/"); len(parts) >= 2 {
				detail.TaskDefinition = parts[len(parts)-1]
				if debugLogger := GetDebugLogger(); debugLogger != nil {
					debugLogger.LogWithCaller("loadTaskDetailsCmd", "Extracted task definition from ARN: %s", detail.TaskDefinition)
				}
			}

			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("loadTaskDetailsCmd", "Calling DescribeTaskDefinition API")
			}
			taskDef, err := m.apiClient.DescribeTaskDefinition(ctx, m.selectedInstance, task.TaskDefinitionArn)
			if err == nil && taskDef != nil {
				if debugLogger := GetDebugLogger(); debugLogger != nil {
					debugLogger.LogWithCaller("loadTaskDetailsCmd", "DescribeTaskDefinition succeeded: Family=%s, Revision=%d", taskDef.Family, taskDef.Revision)
				}
				// Update task definition display to family:revision format if we got valid data
				if taskDef.Family != "" && taskDef.Revision > 0 {
					detail.TaskDefinition = fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision)
				}

				if taskDef.ContainerDefinitions != nil {
					for _, containerDef := range taskDef.ContainerDefinitions {
						// Store essential flag
						containerEssential[containerDef.Name] = containerDef.Essential

						// Store image
						if containerDef.Image != "" {
							containerImages[containerDef.Name] = containerDef.Image
						}

						// Store resource limits
						cpu := "-"
						memory := "-"
						if containerDef.Cpu > 0 {
							cpu = fmt.Sprintf("%d", containerDef.Cpu)
						}
						if containerDef.Memory > 0 {
							memory = fmt.Sprintf("%d", containerDef.Memory)
						} else if containerDef.MemoryReservation > 0 {
							memory = fmt.Sprintf("%d", containerDef.MemoryReservation)
						}
						containerResources[containerDef.Name] = struct{ CPU, Memory string }{cpu, memory}

						// Store dependencies
						if len(containerDef.DependsOn) > 0 {
							var deps []ContainerDependency
							for _, dep := range containerDef.DependsOn {
								deps = append(deps, ContainerDependency{
									ContainerName: dep.ContainerName,
									Condition:     dep.Condition,
								})
							}
							containerDependencies[containerDef.Name] = deps
						}
					}
				}
			} else {
				if debugLogger := GetDebugLogger(); debugLogger != nil {
					debugLogger.LogWithCaller("loadTaskDetailsCmd", "DescribeTaskDefinition failed: err=%v", err)
				}
			}
		} else {
			if debugLogger := GetDebugLogger(); debugLogger != nil {
				debugLogger.LogWithCaller("loadTaskDetailsCmd", "Task has no TaskDefinitionArn")
			}
		}

		// Convert containers
		for _, container := range task.Containers {
			containerDetail := ContainerDetail{
				Name:   container.Name,
				Status: container.LastStatus,
				Reason: container.Reason,
			}

			// Add image from task definition
			if image, ok := containerImages[container.Name]; ok {
				containerDetail.Image = image
			} else {
				containerDetail.Image = "-"
			}

			// Add resource information
			if resources, ok := containerResources[container.Name]; ok {
				containerDetail.CPU = resources.CPU
				containerDetail.Memory = resources.Memory
			} else {
				containerDetail.CPU = "-"
				containerDetail.Memory = "-"
			}

			if container.ExitCode != nil {
				containerDetail.ExitCode = container.ExitCode
			}

			// Add dependsOn information if available
			if deps, ok := containerDependencies[container.Name]; ok {
				containerDetail.DependsOn = deps
			}

			// Add essential flag if available
			if essential, ok := containerEssential[container.Name]; ok {
				containerDetail.Essential = essential
			}

			detail.Containers = append(detail.Containers, containerDetail)
		}

		// Set network mode (default to awsvpc for now)
		detail.NetworkMode = "awsvpc"

		if debugLogger := GetDebugLogger(); debugLogger != nil {
			debugLogger.LogWithCaller("loadTaskDetailsCmd", "Returning task detail with TaskDefinition=%s, Containers=%d",
				detail.TaskDefinition, len(detail.Containers))
		}

		return taskDetailLoadedMsg{detail: detail}
	}
}

// Message types
type taskDetailLoadedMsg struct {
	detail *TaskDetail
}
