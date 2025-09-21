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
	"sort"
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
	Environment     []EnvironmentVariable // Environment variables
	MountPoints     []MountPoint          // Volume mount points
	HealthCheck     *HealthCheck          // Health check configuration
	LogConfig       *LogConfiguration     // Logging configuration
	Secrets         []Secret              // Secrets from Parameter Store or Secrets Manager
	Command         []string              // Command override
	EntryPoint      []string              // Entry point override
	WorkingDir      string                // Working directory
	User            string                // User to run as
}

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Name  string
	Value string
}

// MountPoint represents a volume mount point
type MountPoint struct {
	SourceVolume  string
	ContainerPath string
	ReadOnly      bool
}

// HealthCheck represents container health check configuration
type HealthCheck struct {
	Command     []string
	Interval    int
	Timeout     int
	Retries     int
	StartPeriod int
}

// LogConfiguration represents container logging configuration
type LogConfiguration struct {
	LogDriver string
	Options   map[string]string
}

// Secret represents a secret from Parameter Store or Secrets Manager
type Secret struct {
	Name      string
	ValueFrom string
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

	// Calculate available width for 3-column layout
	availableWidth := m.width - 10 // Account for padding and borders
	columnWidth := availableWidth / 3

	// Render Overview, Network, and Timestamps in 3 columns
	overviewSection := m.renderTaskOverviewCompact(detail, columnWidth)
	networkSection := m.renderTaskNetworkCompact(detail, columnWidth)
	timestampsSection := m.renderTaskTimestampsCompact(detail, columnWidth)

	// Join the three sections horizontally
	topSection := lipgloss.JoinHorizontal(
		lipgloss.Top,
		overviewSection,
		networkSection,
		timestampsSection,
	)

	// Render containers section with horizontal layout
	containersSection := m.renderTaskContainersHorizontal(detail, availableWidth)

	// Join sections vertically
	sections := []string{
		topSection,
		containersSection,
	}

	// Join sections with spacing
	content := strings.Join(sections, "\n\n")

	// Don't add padding here since it's now handled by the resource panel
	// content = contentStyle.Render(content)

	// Calculate available height for content within resource panel
	// Use a more conservative estimate for the resource panel context
	contentHeight := 20 // Default height for resource panel content
	if m.height > 10 {
		contentHeight = m.height - 9 // Leave space for navigation and header
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

// renderTaskOverviewCompact renders a compact overview for 3-column layout
func (m Model) renderTaskOverviewCompact(detail *TaskDetail, width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	columnStyle := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Padding(0, 1)

	var lines []string
	lines = append(lines, titleStyle.Render("Overview"))

	// Task Definition - compact format
	taskDefDisplay := detail.TaskDefinition
	if taskDefDisplay != "" {
		parts := strings.Split(taskDefDisplay, "/")
		if len(parts) > 1 {
			taskDefDisplay = parts[len(parts)-1]
		}
	} else {
		taskDefDisplay = "-"
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("Task Definition:"),
		valueStyle.Render(taskDefDisplay)))

	// Launch Type
	launchType := detail.LaunchType
	if launchType == "" {
		launchType = "FARGATE"
	}
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("Launch Type:"),
		valueStyle.Render(launchType)))

	// Resources
	cpu := detail.CPU
	if cpu == "" {
		cpu = "-"
	}
	memory := detail.Memory
	if memory == "" {
		memory = "-"
	}
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("Resources:"),
		valueStyle.Render(fmt.Sprintf("CPU: %s, Memory: %s", cpu, memory))))

	// Health
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

	return columnStyle.Render(strings.Join(lines, "\n"))
}

// renderTaskNetworkCompact renders a compact network section for 3-column layout
func (m Model) renderTaskNetworkCompact(detail *TaskDetail, width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	columnStyle := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Padding(0, 1)

	var lines []string
	lines = append(lines, titleStyle.Render("Network"))

	// Network Mode
	networkMode := detail.NetworkMode
	if networkMode == "" {
		networkMode = "awsvpc"
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("Mode:"),
		valueStyle.Render(networkMode)))

	// IPs
	ipsValue := "-"
	if len(detail.IPs) > 0 {
		ipsValue = strings.Join(detail.IPs, ", ")
	}
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("IPs:"),
		valueStyle.Render(ipsValue)))

	return columnStyle.Render(strings.Join(lines, "\n"))
}

// renderTaskTimestampsCompact renders a compact timestamps section for 3-column layout
func (m Model) renderTaskTimestampsCompact(detail *TaskDetail, width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	columnStyle := lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Padding(0, 1)

	var lines []string
	lines = append(lines, titleStyle.Render("Timestamps"))

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("%s %s",
		labelStyle.Render("Created:"),
		valueStyle.Render(detail.CreatedAt.Format("15:04:05"))))
	lines = append(lines, fmt.Sprintf("%s",
		valueStyle.Render(fmt.Sprintf("(%s ago)", formatDuration(time.Since(detail.CreatedAt))))))

	// Started
	if detail.StartedAt != nil && !detail.StartedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("Started:"),
			valueStyle.Render(detail.StartedAt.Format("15:04:05"))))
	}

	// Stopped
	if detail.StoppedAt != nil && !detail.StoppedAt.IsZero() {
		lines = append(lines, fmt.Sprintf("%s %s",
			labelStyle.Render("Stopped:"),
			valueStyle.Render(detail.StoppedAt.Format("15:04:05"))))
	}

	return columnStyle.Render(strings.Join(lines, "\n"))
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

// renderTaskContainersHorizontal renders containers in a 2-column layout with selectable list
func (m Model) renderTaskContainersHorizontal(detail *TaskDetail, width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	sectionStyle := lipgloss.NewStyle().
		MarginBottom(1)

	// Title
	title := titleStyle.Render(fmt.Sprintf("Containers (%d)", len(detail.Containers)))

	numContainers := len(detail.Containers)
	if numContainers == 0 {
		return sectionStyle.Render(title + "\n\nNo containers")
	}

	// Calculate widths for 2-column layout
	listWidth := 25                      // Fixed width for container list (reduced since no border)
	detailWidth := width - listWidth - 4 // -4 for margin/padding between columns

	// Ensure minimum widths
	if detailWidth < 40 {
		detailWidth = 40
	}

	// Build container list (left column)
	listStyle := lipgloss.NewStyle().
		Width(listWidth).
		Padding(0, 1)

	var listItems []string
	for i, container := range detail.Containers {
		// Container status color
		statusColor := "244"
		switch strings.ToUpper(container.Status) {
		case "RUNNING":
			statusColor = "82"
		case "PENDING":
			statusColor = "226"
		case "STOPPED":
			statusColor = "196"
		}

		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor))

		// Format container name with selection indicator
		nameDisplay := container.Name
		if len(nameDisplay) > listWidth-6 { // Leave space for status indicator
			nameDisplay = nameDisplay[:listWidth-9] + "..."
		}

		itemStyle := lipgloss.NewStyle()
		if i == m.selectedContainer {
			// Highlight selected container
			itemStyle = itemStyle.
				Background(lipgloss.Color("237")).
				Foreground(lipgloss.Color("220")).
				Bold(true)
			listItems = append(listItems, itemStyle.Render(fmt.Sprintf("▶ %s [%s]",
				nameDisplay,
				statusStyle.Render(strings.ToUpper(container.Status[:1])))))
		} else {
			listItems = append(listItems, fmt.Sprintf("  %s [%s]",
				nameDisplay,
				statusStyle.Render(strings.ToUpper(container.Status[:1]))))
		}
	}

	containerList := listStyle.Render(strings.Join(listItems, "\n"))

	// Build container detail (right column)
	detailStyle := lipgloss.NewStyle().
		Width(detailWidth).
		Padding(0, 1)

	// Ensure selected container index is valid
	if m.selectedContainer >= len(detail.Containers) {
		m.selectedContainer = 0
	}

	selectedContainer := detail.Containers[m.selectedContainer]
	detailContent := m.renderContainerDetail(selectedContainer, detailWidth-2) // -2 for padding

	containerDetail := detailStyle.Render(detailContent)

	// Join the two columns
	columnsRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		containerList,
		lipgloss.NewStyle().Width(2).Render("  "), // Spacer
		containerDetail,
	)

	// Combine title and columns
	return sectionStyle.Render(title + "\n\n" + columnsRow)
}

// renderContainerDetail renders detailed information for a single container
func (m Model) renderContainerDetail(container ContainerDetail, width int) string {
	var lines []string

	// Container name and status header
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
		imageLabel := lipgloss.NewStyle().Bold(true).Render("Image:")
		lines = append(lines, fmt.Sprintf("%s %s", imageLabel, container.Image))
	} else {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Image:")+" -")
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
	resourceLabel := lipgloss.NewStyle().Bold(true).Render("Resources:")
	lines = append(lines, fmt.Sprintf("%s CPU=%s, Memory=%s", resourceLabel, cpu, memory))

	// Exit code and reason
	if container.ExitCode != nil {
		exitColor := "82"
		if *container.ExitCode != 0 {
			exitColor = "196"
		}
		exitStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(exitColor))
		exitLabel := lipgloss.NewStyle().Bold(true).Render("Exit Code:")
		lines = append(lines, fmt.Sprintf("%s %s", exitLabel,
			exitStyle.Render(fmt.Sprintf("%d", *container.ExitCode))))

		if container.Reason != "" {
			reasonLabel := lipgloss.NewStyle().Bold(true).Render("Reason:")
			lines = append(lines, fmt.Sprintf("%s %s", reasonLabel, container.Reason))
		}
	}

	// Environment variables (if present, show first few)
	if len(container.Environment) > 0 {
		lines = append(lines, "")
		envLabel := lipgloss.NewStyle().Bold(true).Render("Environment:")
		lines = append(lines, envLabel)
		maxEnvVars := 5
		for i, env := range container.Environment {
			if i >= maxEnvVars {
				lines = append(lines, fmt.Sprintf("  ... and %d more", len(container.Environment)-maxEnvVars))
				break
			}
			lines = append(lines, fmt.Sprintf("  %s=%s", env.Name, env.Value))
		}
	}

	// Dependencies
	if len(container.DependsOn) > 0 {
		lines = append(lines, "")
		depsLabel := lipgloss.NewStyle().Bold(true).Render("Dependencies:")
		lines = append(lines, depsLabel)
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
			lines = append(lines, fmt.Sprintf("  %s (%s)",
				dep.ContainerName, condStyle.Render(dep.Condition)))
		}
	}

	// Network bindings
	if len(container.NetworkBindings) > 0 {
		lines = append(lines, "")
		portsLabel := lipgloss.NewStyle().Bold(true).Render("Port Mappings:")
		lines = append(lines, portsLabel)
		for _, binding := range container.NetworkBindings {
			lines = append(lines, fmt.Sprintf("  %d:%d/%s",
				binding.HostPort, binding.ContainerPort, binding.Protocol))
		}
	}

	// Mount points
	if len(container.MountPoints) > 0 {
		lines = append(lines, "")
		mountsLabel := lipgloss.NewStyle().Bold(true).Render("Mount Points:")
		lines = append(lines, mountsLabel)
		for _, mount := range container.MountPoints {
			readOnly := ""
			if mount.ReadOnly {
				readOnly = " (ro)"
			}
			lines = append(lines, fmt.Sprintf("  %s → %s%s",
				mount.SourceVolume, mount.ContainerPath, readOnly))
		}
	}

	// Health Check
	if container.HealthCheck != nil {
		lines = append(lines, "")
		healthLabel := lipgloss.NewStyle().Bold(true).Render("Health Check:")
		lines = append(lines, healthLabel)
		if len(container.HealthCheck.Command) > 0 {
			lines = append(lines, fmt.Sprintf("  Command: %s", strings.Join(container.HealthCheck.Command, " ")))
		}
		if container.HealthCheck.Interval > 0 {
			lines = append(lines, fmt.Sprintf("  Interval: %ds", container.HealthCheck.Interval))
		}
		if container.HealthCheck.Timeout > 0 {
			lines = append(lines, fmt.Sprintf("  Timeout: %ds", container.HealthCheck.Timeout))
		}
		if container.HealthCheck.Retries > 0 {
			lines = append(lines, fmt.Sprintf("  Retries: %d", container.HealthCheck.Retries))
		}
		if container.HealthCheck.StartPeriod > 0 {
			lines = append(lines, fmt.Sprintf("  Start Period: %ds", container.HealthCheck.StartPeriod))
		}
	}

	// Health Status
	if container.HealthStatus != "" && container.HealthStatus != "UNKNOWN" {
		if container.HealthCheck == nil {
			lines = append(lines, "")
		}
		healthStatusLabel := lipgloss.NewStyle().Bold(true).Render("Health Status:")
		healthStatusColor := "244"
		switch container.HealthStatus {
		case "HEALTHY":
			healthStatusColor = "82"
		case "UNHEALTHY":
			healthStatusColor = "196"
		}
		healthStatusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(healthStatusColor))
		lines = append(lines, fmt.Sprintf("%s %s", healthStatusLabel, healthStatusStyle.Render(container.HealthStatus)))
	}

	// Log Configuration
	if container.LogConfig != nil {
		lines = append(lines, "")
		logLabel := lipgloss.NewStyle().Bold(true).Render("Log Configuration:")
		lines = append(lines, logLabel)
		lines = append(lines, fmt.Sprintf("  Driver: %s", container.LogConfig.LogDriver))
		if len(container.LogConfig.Options) > 0 {
			lines = append(lines, "  Options:")
			// Sort keys for consistent display order
			var keys []string
			for key := range container.LogConfig.Options {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				lines = append(lines, fmt.Sprintf("    %s: %s", key, container.LogConfig.Options[key]))
			}
		}
	}

	// Secrets
	if len(container.Secrets) > 0 {
		lines = append(lines, "")
		secretsLabel := lipgloss.NewStyle().Bold(true).Render("Secrets:")
		lines = append(lines, secretsLabel)
		for _, secret := range container.Secrets {
			lines = append(lines, fmt.Sprintf("  %s: %s", secret.Name, secret.ValueFrom))
		}
	}

	// Command and EntryPoint
	if len(container.Command) > 0 {
		lines = append(lines, "")
		cmdLabel := lipgloss.NewStyle().Bold(true).Render("Command:")
		lines = append(lines, fmt.Sprintf("%s %s", cmdLabel, strings.Join(container.Command, " ")))
	}
	if len(container.EntryPoint) > 0 {
		lines = append(lines, "")
		entryLabel := lipgloss.NewStyle().Bold(true).Render("Entry Point:")
		lines = append(lines, fmt.Sprintf("%s %s", entryLabel, strings.Join(container.EntryPoint, " ")))
	}

	// Working Directory and User
	if container.WorkingDir != "" {
		lines = append(lines, "")
		workDirLabel := lipgloss.NewStyle().Bold(true).Render("Working Dir:")
		lines = append(lines, fmt.Sprintf("%s %s", workDirLabel, container.WorkingDir))
	}
	if container.User != "" {
		lines = append(lines, "")
		userLabel := lipgloss.NewStyle().Bold(true).Render("User:")
		lines = append(lines, fmt.Sprintf("%s %s", userLabel, container.User))
	}

	return strings.Join(lines, "\n")
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
		var containerHealthChecks map[string]*HealthCheck
		var containerLogConfigs map[string]*LogConfiguration
		var containerSecrets map[string][]Secret
		var containerEnvironments map[string][]EnvironmentVariable
		var containerMountPoints map[string][]MountPoint
		var containerCommands map[string][]string
		var containerEntryPoints map[string][]string
		var containerWorkingDirs map[string]string
		var containerUsers map[string]string

		// Initialize maps regardless of task definition fetch result
		containerDependencies = make(map[string][]ContainerDependency)
		containerEssential = make(map[string]bool)
		containerImages = make(map[string]string)
		containerResources = make(map[string]struct{ CPU, Memory string })
		containerHealthChecks = make(map[string]*HealthCheck)
		containerLogConfigs = make(map[string]*LogConfiguration)
		containerSecrets = make(map[string][]Secret)
		containerEnvironments = make(map[string][]EnvironmentVariable)
		containerMountPoints = make(map[string][]MountPoint)
		containerCommands = make(map[string][]string)
		containerEntryPoints = make(map[string][]string)
		containerWorkingDirs = make(map[string]string)
		containerUsers = make(map[string]string)

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

						// Store health check
						if containerDef.HealthCheck != nil {
							containerHealthChecks[containerDef.Name] = &HealthCheck{
								Command:     containerDef.HealthCheck.Command,
								Interval:    containerDef.HealthCheck.Interval,
								Timeout:     containerDef.HealthCheck.Timeout,
								Retries:     containerDef.HealthCheck.Retries,
								StartPeriod: containerDef.HealthCheck.StartPeriod,
							}
						}

						// Store log configuration
						if containerDef.LogConfiguration != nil {
							containerLogConfigs[containerDef.Name] = &LogConfiguration{
								LogDriver: containerDef.LogConfiguration.LogDriver,
								Options:   containerDef.LogConfiguration.Options,
							}
						}

						// Store secrets
						if len(containerDef.Secrets) > 0 {
							var secrets []Secret
							for _, s := range containerDef.Secrets {
								secrets = append(secrets, Secret{
									Name:      s.Name,
									ValueFrom: s.ValueFrom,
								})
							}
							containerSecrets[containerDef.Name] = secrets
						}

						// Store environment variables
						if len(containerDef.Environment) > 0 {
							var envVars []EnvironmentVariable
							for _, e := range containerDef.Environment {
								envVars = append(envVars, EnvironmentVariable{
									Name:  e.Name,
									Value: e.Value,
								})
							}
							containerEnvironments[containerDef.Name] = envVars
						}

						// Store mount points
						if len(containerDef.MountPoints) > 0 {
							var mounts []MountPoint
							for _, m := range containerDef.MountPoints {
								mounts = append(mounts, MountPoint{
									SourceVolume:  m.SourceVolume,
									ContainerPath: m.ContainerPath,
									ReadOnly:      m.ReadOnly,
								})
							}
							containerMountPoints[containerDef.Name] = mounts
						}

						// Store command and entrypoint
						if len(containerDef.Command) > 0 {
							containerCommands[containerDef.Name] = containerDef.Command
						}
						if len(containerDef.EntryPoint) > 0 {
							containerEntryPoints[containerDef.Name] = containerDef.EntryPoint
						}

						// Store working directory and user
						if containerDef.WorkingDirectory != "" {
							containerWorkingDirs[containerDef.Name] = containerDef.WorkingDirectory
						}
						if containerDef.User != "" {
							containerUsers[containerDef.Name] = containerDef.User
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

			// Add health check if available
			if healthCheck, ok := containerHealthChecks[container.Name]; ok {
				containerDetail.HealthCheck = healthCheck
			}

			// Add log configuration if available
			if logConfig, ok := containerLogConfigs[container.Name]; ok {
				containerDetail.LogConfig = logConfig
			}

			// Add secrets if available
			if secrets, ok := containerSecrets[container.Name]; ok {
				containerDetail.Secrets = secrets
			}

			// Add environment variables if available
			if envVars, ok := containerEnvironments[container.Name]; ok {
				containerDetail.Environment = envVars
			}

			// Add mount points if available
			if mounts, ok := containerMountPoints[container.Name]; ok {
				containerDetail.MountPoints = mounts
			}

			// Add command if available
			if command, ok := containerCommands[container.Name]; ok {
				containerDetail.Command = command
			}

			// Add entrypoint if available
			if entryPoint, ok := containerEntryPoints[container.Name]; ok {
				containerDetail.EntryPoint = entryPoint
			}

			// Add working directory if available
			if workingDir, ok := containerWorkingDirs[container.Name]; ok {
				containerDetail.WorkingDir = workingDir
			}

			// Add user if available
			if user, ok := containerUsers[container.Name]; ok {
				containerDetail.User = user
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
