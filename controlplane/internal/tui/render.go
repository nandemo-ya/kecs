package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/nandemo-ya/kecs/controlplane/internal/version"
)

var (
	// Layout styles
	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e1e2e")).
			Foreground(lipgloss.Color("#cdd6f4")).
			PaddingLeft(1).
			PaddingRight(1).
			Bold(true)

	breadcrumbStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#313244")).
			Foreground(lipgloss.Color("#bac2de")).
			PaddingLeft(1).
			PaddingRight(1).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#585b70"))

	footerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1a1a")).
			Foreground(lipgloss.Color("#808080")).
			Padding(0, 1)

	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	statusActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00ff00"))

	statusInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ff0000"))

	statusErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ff0000"))

	// Carousel styles
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#89dceb")).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cdd6f4"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#585b70"))

	emptyStateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a6adc8")).
			Italic(true)

	// New styles for enhanced layout
	navigationPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#585b70")).
				Padding(1, 2)

	resourcePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#585b70")).
				Padding(1, 2)

	summaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#aaaaaa")).
			Padding(0, 1)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#333333"))

	// Row highlight style for selected items
	selectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#005577")).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true)

	dimmedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080"))
)

// getHeaderShortcuts returns context-appropriate shortcuts for the header
func (m Model) getHeaderShortcuts() string {
	// Check if we're in a special mode
	if m.searchMode {
		return statusActiveStyle.Render("Search Mode")
	}
	if m.commandMode {
		return statusActiveStyle.Render("Command Mode")
	}
	if m.currentView == ViewCommandPalette {
		return statusActiveStyle.Render("Command Palette")
	}
	if m.currentView == ViewInstanceCreate {
		return statusActiveStyle.Render("Create Instance")
	}

	// Show context-specific shortcuts
	shortcuts := []string{}

	// Style for k9s-like shortcuts
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))

	switch m.currentView {
	case ViewInstances:
		// Show only the most important shortcuts in the header
		shortcuts = []string{
			keyStyle.Render("↵") + sepStyle.Render(" Select"),
			keyStyle.Render("n") + sepStyle.Render(" New"),
			keyStyle.Render("s") + sepStyle.Render(" Start/Stop"),
			keyStyle.Render("?") + sepStyle.Render(" Help"),
		}
	case ViewClusters:
		shortcuts = []string{
			keyStyle.Render("↵") + sepStyle.Render(" Select"),
			keyStyle.Render("n") + sepStyle.Render(" Create"),
			keyStyle.Render("T") + sepStyle.Render(" Tasks"),
			keyStyle.Render("ESC") + sepStyle.Render(" Back"),
			keyStyle.Render("?") + sepStyle.Render(" Help"),
		}
	case ViewServices:
		shortcuts = []string{
			keyStyle.Render("s") + sepStyle.Render(" Scale"),
			keyStyle.Render("u") + sepStyle.Render(" Update"),
			keyStyle.Render("l") + sepStyle.Render(" Logs"),
			keyStyle.Render("ESC") + sepStyle.Render(" Back"),
		}
	case ViewTasks:
		shortcuts = []string{
			keyStyle.Render("↵") + sepStyle.Render(" Describe"),
			keyStyle.Render("l") + sepStyle.Render(" Logs"),
			keyStyle.Render("s") + sepStyle.Render(" Stop"),
			keyStyle.Render("ESC") + sepStyle.Render(" Back"),
		}
	case ViewLogs:
		shortcuts = []string{
			keyStyle.Render("f") + sepStyle.Render(" Toggle"),
			keyStyle.Render("s") + sepStyle.Render(" Save"),
			keyStyle.Render("ESC") + sepStyle.Render(" Back"),
		}
	case ViewTaskDefinitionFamilies:
		shortcuts = []string{
			keyStyle.Render("↵") + sepStyle.Render(" Select"),
			keyStyle.Render("N") + sepStyle.Render(" New"),
			keyStyle.Render("C") + sepStyle.Render(" Copy"),
			keyStyle.Render("ESC") + sepStyle.Render(" Back"),
		}
	case ViewTaskDefinitionRevisions:
		shortcuts = []string{
			keyStyle.Render("↵") + sepStyle.Render(" JSON"),
			keyStyle.Render("e") + sepStyle.Render(" Edit"),
			keyStyle.Render("y") + sepStyle.Render(" Yank"),
			keyStyle.Render("ESC") + sepStyle.Render(" Back"),
		}
	default:
		shortcuts = []string{
			keyStyle.Render(":") + sepStyle.Render(" Cmd"),
			keyStyle.Render("/") + sepStyle.Render(" Search"),
			keyStyle.Render("?") + sepStyle.Render(" Help"),
		}
	}

	// Join shortcuts with spaces
	return strings.Join(shortcuts, "  ")
}

// renderInstanceCarousel renders the instance list as a horizontal carousel
func (m Model) renderInstanceCarousel() string {
	// Empty state
	if len(m.instances) == 0 {
		return emptyStateStyle.Render("No KECS instances | Press i to create instance")
	}

	// Calculate visible instances (read-only in render)
	avgInstanceWidth := 20 // Average width per instance name + spacing
	indicatorWidth := 4    // Space for "◀ " and " ▶"
	padding := 4           // General padding

	availableWidth := m.width - indicatorWidth - padding
	maxVisibleInstances := availableWidth / avgInstanceWidth
	if maxVisibleInstances < 1 {
		maxVisibleInstances = 1
	}

	// Calculate carousel offset for current selection
	selectedIdx := m.getSelectedInstanceIndex()
	carouselOffset := m.instanceCarouselOffset

	// Adjust offset if needed (but don't modify state)
	if selectedIdx < carouselOffset {
		carouselOffset = selectedIdx
	}
	if selectedIdx >= carouselOffset+maxVisibleInstances {
		carouselOffset = selectedIdx - maxVisibleInstances + 1
	}

	var items []string

	// Add left navigation indicator if needed
	if carouselOffset > 0 {
		items = append(items, dimStyle.Render("◀"))
	}

	// Determine the range of instances to display
	startIdx := carouselOffset
	endIdx := startIdx + maxVisibleInstances
	if endIdx > len(m.instances) {
		endIdx = len(m.instances)
	}

	// Render visible instances
	for i := startIdx; i < endIdx; i++ {
		inst := m.instances[i]

		// Status indicator
		var statusIcon string
		switch strings.ToLower(inst.Status) {
		case "running":
			statusIcon = statusActiveStyle.Render("●")
		case "unhealthy":
			statusIcon = statusErrorStyle.Render("●")
		default:
			statusIcon = statusInactiveStyle.Render("○")
		}

		// Format instance name with selection indicator
		instDisplay := fmt.Sprintf("%s %s", statusIcon, inst.Name)
		if inst.Name == m.selectedInstance {
			// Highlight selected instance
			instDisplay = selectedItemStyle.Render(fmt.Sprintf("[ %s ]", instDisplay))
		} else {
			instDisplay = normalItemStyle.Render(instDisplay)
		}

		items = append(items, instDisplay)
	}

	// Add right navigation indicator if needed
	if endIdx < len(m.instances) {
		items = append(items, dimStyle.Render("▶"))
	}

	// Always show the create instance shortcut at the end
	createShortcut := dimStyle.Render("│ Press i to create")
	items = append(items, createShortcut)

	// Join items with spacing
	return strings.Join(items, "  ")
}

func (m Model) renderHeader() string {
	// Build the header with KECS version and instance carousel
	headerLeft := fmt.Sprintf("KECS %s", version.GetVersion())

	// Render instance carousel on the same line
	instanceCarousel := m.renderInstanceCarousel()

	// Combine header and carousel
	if instanceCarousel != "" {
		return headerStyle.Render(fmt.Sprintf("%s | %s", headerLeft, instanceCarousel))
	}

	return headerStyle.Render(headerLeft)
}

func (m Model) renderBreadcrumb() string {
	parts := []string{}

	// Build breadcrumb based on current navigation
	// Skip instance part since it's shown in the header

	// Clusters breadcrumb
	if m.currentView == ViewClusters {
		parts = append(parts, "[Clusters]")
	} else if m.selectedCluster != "" {
		// Format: Cluster(name) - don't truncate here, let the full-width logic handle it
		parts = append(parts, fmt.Sprintf("Cluster(%s)", m.selectedCluster))
	}

	// Services breadcrumb
	if m.currentView == ViewServices && len(parts) > 0 {
		parts = append(parts, ">", "[Services]")
	} else if m.selectedService != "" {
		if len(parts) == 0 && m.selectedCluster != "" {
			// Add cluster if it's not already there
			parts = append(parts, fmt.Sprintf("Cluster(%s)", m.selectedCluster))
		}
		if len(parts) > 0 {
			parts = append(parts, ">")
		}
		// Format: Service(name) - don't truncate here, let the full-width logic handle it
		parts = append(parts, fmt.Sprintf("Service(%s)", m.selectedService))
	}

	// Show Tasks breadcrumb whether service is selected or not
	if m.currentView == ViewTasks {
		if len(parts) > 0 {
			parts = append(parts, ">")
		}
		if m.selectedService == "" {
			// Showing all tasks in cluster
			parts = append(parts, "[All Tasks]")
		} else {
			// Showing tasks for specific service
			parts = append(parts, "[Tasks]")
		}
	}

	// Task Describe view
	if m.currentView == ViewTaskDescribe && m.selectedTask != "" {
		if len(parts) > 0 {
			parts = append(parts, ">")
		}
		// Show full task ID without truncation
		parts = append(parts, fmt.Sprintf("Task(%s)", m.selectedTask))
	}

	// Logs view
	if m.currentView == ViewLogs {
		if m.selectedTask != "" && m.currentView != ViewTaskDescribe {
			if len(parts) > 0 {
				parts = append(parts, ">")
			}
			// Show full task ID without truncation
			parts = append(parts, fmt.Sprintf("Task(%s)", m.selectedTask))
		}
		if len(parts) > 0 {
			parts = append(parts, ">")
		}
		parts = append(parts, "[Logs]")
	}

	// Task Definition navigation
	if m.currentView == ViewTaskDefinitionFamilies || m.currentView == ViewTaskDefinitionRevisions {
		parts = append(parts, "[Task Definitions]")

		if m.currentView == ViewTaskDefinitionRevisions && m.selectedFamily != "" {
			parts = append(parts, ">", m.selectedFamily)
		}
	}

	// Handle empty breadcrumb
	if len(parts) == 0 {
		// Don't show anything if there's no navigation context
		return ""
	}

	breadcrumb := strings.Join(parts, " ")

	// Let the breadcrumb flow naturally without width constraints
	return breadcrumbStyle.Render(breadcrumb)
}

func (m Model) renderFooter() string {
	// Check if we should show clipboard notification
	if m.clipboardMsg != "" && time.Since(m.clipboardMsgTime) < 3*time.Second {
		notification := successStyle.Render("📋 " + m.clipboardMsg)
		return footerStyle.Width(m.width).Render(notification)
	}

	// Check if we should show command result
	if m.commandPalette != nil && m.commandPalette.ShouldShowResult() {
		// Show command result for a few seconds
		result := successStyle.Render("✓ " + m.commandPalette.lastResult)
		return footerStyle.Width(m.width).Render(result)
	}

	// Show mode-specific input
	if m.searchMode {
		input := fmt.Sprintf("Search: %s_", m.searchQuery)
		help := dimmedStyle.Render("[Enter] Apply  [Esc] Cancel")
		content := fmt.Sprintf("%s  %s", input, help)
		return footerStyle.Width(m.width).Render(content)
	} else if m.commandMode {
		input := fmt.Sprintf("Command: %s_", m.commandInput)
		help := dimmedStyle.Render("[Enter] Execute  [Tab] Palette  [Esc] Cancel")
		content := fmt.Sprintf("%s  %s", input, help)
		return footerStyle.Width(m.width).Render(content)
	}

	// Default footer is empty (no status info)
	return footerStyle.Width(m.width).Render("")
}

// renderNavigationPanel renders the top navigation panel (30% height)
func (m Model) renderNavigationPanel() string {
	// Calculate height for navigation panel (30% of available height)
	navHeight := int(float64(m.height) * 0.3)

	// Ensure minimum height for all content (including shortcuts)
	// Need at least: header(1) + breadcrumb(1) + separator(1) + summary(1-2) + separator(1) + view shortcuts(1) + global shortcuts(1) = 7-8 lines
	minRequiredHeight := 9
	if navHeight < minRequiredHeight {
		navHeight = minRequiredHeight
	}

	// Calculate actual available width inside the panel
	panelWidth := m.width - 6 // Account for panel borders and padding

	// Header and breadcrumb use full width
	header := m.renderHeader()
	breadcrumb := m.renderBreadcrumb()
	summary := m.renderSummary()

	// Add separator line - match the actual panel width
	separatorWidth := panelWidth
	if separatorWidth > m.width-8 {
		separatorWidth = m.width - 8 // Ensure it doesn't overflow
	}
	topSeparator := separatorStyle.Render(strings.Repeat("─", separatorWidth))

	// Render shortcuts as single lines at the bottom
	viewShortcuts := m.renderViewShortcutsLine(panelWidth)
	globalShortcuts := m.renderGlobalShortcutsLine(panelWidth)

	// Stack all components vertically
	var components []string
	components = append(components, header)
	if breadcrumb != "" {
		components = append(components, breadcrumb)
	}
	components = append(components, topSeparator)
	components = append(components, summary)
	// Only one separator line before shortcuts
	if viewShortcuts != "" {
		components = append(components, viewShortcuts)
	}
	if globalShortcuts != "" {
		components = append(components, globalShortcuts)
	}

	navContent := lipgloss.JoinVertical(lipgloss.Top, components...)

	// Don't restrict height too much - let content determine minimum
	actualContentHeight := len(components) + 2 // Add padding
	if navHeight < actualContentHeight {
		navHeight = actualContentHeight
	}

	return navigationPanelStyle.
		Width(m.width - 4). // Account for borders and padding
		Height(navHeight - 4).
		Render(navContent)
}

// renderResourcePanel renders the bottom resource panel (70% height)
func (m Model) renderResourcePanel() string {
	// Calculate height for resource panel (70% of available height)
	resourceHeight := int(float64(m.height) * 0.7)
	if resourceHeight < 10 {
		resourceHeight = 10 // Minimum height
	}

	var content string

	// Render view-specific content
	switch m.currentView {
	case ViewInstances:
		// Only show instances view when no instances exist (for creation prompt)
		if len(m.instances) == 0 {
			content = m.renderNoInstancesView()
		} else {
			// Should not normally reach here as we auto-select instances
			content = m.renderClustersList(resourceHeight - 4)
		}
	case ViewClusters:
		content = m.renderClustersList(resourceHeight - 4)
	case ViewServices:
		content = m.renderServicesList(resourceHeight - 4)
	case ViewTasks:
		content = m.renderTasksList(resourceHeight - 4)
	case ViewTaskDescribe:
		content = m.renderTaskDescribe()
	case ViewLogs:
		// Always show log viewer in fullscreen if active
		if m.logViewer != nil {
			return m.logViewer.View()
		} else {
			content = m.renderLogsContent(resourceHeight - 4)
		}
	case ViewHelp:
		content = m.renderHelpContent(resourceHeight - 4)
	case ViewTaskDefinitionFamilies:
		content = m.renderTaskDefFamiliesList(resourceHeight - 4)
	case ViewTaskDefinitionRevisions:
		if m.showTaskDefJSON {
			content = m.renderTaskDefRevisionsTwoColumn(resourceHeight - 4)
		} else {
			content = m.renderTaskDefRevisionsList(resourceHeight-4, m.width-8)
		}
	}

	// Apply resource panel style with fixed height
	return resourcePanelStyle.
		Width(m.width - 4).         // Account for borders and padding
		Height(resourceHeight - 4). // Account for borders and padding
		Render(content)
}

// renderSummary renders contextual summary information
func (m Model) renderSummary() string {
	var summary string

	switch m.currentView {
	case ViewInstances:
		active := 0
		total := len(m.instances)
		for _, inst := range m.instances {
			// Check for various active states
			if inst.Status == "ACTIVE" || inst.Status == "Running" || inst.Status == "running" {
				active++
			}
		}
		summary = fmt.Sprintf("Total Instances: %d | Active: %d | Stopped: %d",
			total, active, total-active)

	case ViewClusters:
		if m.selectedInstance != "" {
			totalServices := 0
			totalTasks := 0
			for _, cluster := range m.clusters {
				totalServices += cluster.Services
				totalTasks += cluster.Tasks
			}

			// Find instance configuration
			var apiPort, adminPort int
			for _, inst := range m.instances {
				if inst.Name == m.selectedInstance {
					apiPort = inst.APIPort
					adminPort = inst.AdminPort
					break
				}
			}

			// Build vertical layout with proper alignment
			line1 := fmt.Sprintf("Clusters: %-4d  API:   %d", len(m.clusters), apiPort)
			line2 := fmt.Sprintf("Services: %-4d  Admin: %d", totalServices, adminPort)
			line3 := fmt.Sprintf("Tasks:    %-4d", totalTasks)

			// Create multi-line summary with consistent styling
			summary = summaryStyle.Render(line1) + "\n" +
				summaryStyle.Render(line2) + "\n" +
				summaryStyle.Render(line3)
		}

	case ViewServices:
		if m.selectedCluster != "" {
			totalDesired := 0
			totalRunning := 0
			for _, svc := range m.services {
				totalDesired += svc.Desired
				totalRunning += svc.Running
			}

			summary = fmt.Sprintf("Cluster: %s | Services: %d | Desired Tasks: %d | Running Tasks: %d",
				m.selectedCluster, len(m.services), totalDesired, totalRunning)
		}

	case ViewTasks:
		running := 0
		healthy := 0
		for _, task := range m.tasks {
			if task.Status == "RUNNING" {
				running++
				if task.Health == "HEALTHY" {
					healthy++
				}
			}
		}

		if m.selectedService != "" {
			// Tasks for specific service
			summary = fmt.Sprintf("Service: %s | Tasks: %d | Running: %d | Healthy: %d",
				m.selectedService, len(m.tasks), running, healthy)
		} else {
			// All tasks in cluster
			summary = fmt.Sprintf("Cluster: %s | All Tasks: %d | Running: %d | Healthy: %d",
				m.selectedCluster, len(m.tasks), running, healthy)
		}

	case ViewLogs:
		if m.selectedTask != "" {
			// Keep log view summary short and on one line
			summary = fmt.Sprintf("Log entries: %d", len(m.logs))
		}

	case ViewTaskDefinitionFamilies:
		if m.selectedInstance != "" {
			active := 0
			total := len(m.taskDefFamilies)
			for _, family := range m.taskDefFamilies {
				if family.ActiveCount > 0 {
					active++
				}
			}
			summary = fmt.Sprintf("Instance: %s | Task Definition Families: %d | Active: %d",
				m.selectedInstance, total, active)
		}

	case ViewTaskDefinitionRevisions:
		if m.selectedFamily != "" {
			active := 0
			total := len(m.taskDefRevisions)
			for _, rev := range m.taskDefRevisions {
				if rev.Status == "ACTIVE" {
					active++
				}
			}
			latestRev := 0
			if len(m.taskDefRevisions) > 0 {
				latestRev = m.taskDefRevisions[0].Revision
			}
			summary = fmt.Sprintf("Family: %s | Revisions: %d | Active: %d | Latest: %d",
				m.selectedFamily, total, active, latestRev)
		}

	case ViewLoadBalancers:
		if m.selectedInstance != "" {
			active := 0
			for _, lb := range m.loadBalancers {
				if lb.State == "active" {
					active++
				}
			}
			summary = fmt.Sprintf("Load Balancers: %d | Active: %d | Instance: %s",
				len(m.loadBalancers), active, m.selectedInstance)
		}

	case ViewTargetGroups:
		if m.selectedInstance != "" {
			healthy := 0
			unhealthy := 0
			for _, tg := range m.targetGroups {
				if tg.HealthyTargetCount > 0 && tg.UnhealthyTargetCount == 0 {
					healthy++
				} else if tg.UnhealthyTargetCount > 0 {
					unhealthy++
				}
			}
			summary = fmt.Sprintf("Target Groups: %d | Healthy: %d | Unhealthy: %d | Instance: %s",
				len(m.targetGroups), healthy, unhealthy, m.selectedInstance)
		}

	case ViewListeners:
		if m.selectedInstance != "" && m.selectedLB != "" {
			// Extract LB name from ARN if possible
			lbName := m.selectedLB
			if len(m.loadBalancers) > 0 {
				for _, lb := range m.loadBalancers {
					if lb.ARN == m.selectedLB {
						lbName = lb.Name
						break
					}
				}
			}
			summary = fmt.Sprintf("Listeners: %d | Load Balancer: %s",
				len(m.listeners), lbName)
		}
	}

	if summary == "" {
		summary = "No resources selected"
	}

	// Add separator line - make sure it fits within the left column width
	totalWidth := m.width - 4 // Account for panel borders
	leftColumnWidth := int(float64(totalWidth) * 0.7)
	separatorWidth := leftColumnWidth - 2 // Account for padding
	if separatorWidth < 20 {
		separatorWidth = 20
	}
	separator := strings.Repeat("─", separatorWidth)

	// For multi-line summaries (which contain newlines), we need to handle them differently
	if strings.Contains(summary, "\n") {
		// Multi-line summary - add style and separator at the end
		return lipgloss.JoinVertical(
			lipgloss.Top,
			summary, // Already styled per line
			separatorStyle.Render(separator),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Top,
		summaryStyle.Render(summary),
		separatorStyle.Render(separator),
	)
}

// View-specific render methods

// renderNoInstancesView renders a view when no instances exist
func (m Model) renderNoInstancesView() string {
	// Use the full available height
	height := m.height

	// Create styled welcome message components
	emojiStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f9e2af")).
		Bold(true)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f5c2e7")).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8"))

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6e3a1")).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9399b2"))

	// Build the welcome content
	lines := []string{
		emojiStyle.Render("🚀") + " " + titleStyle.Render("Welcome to KECS"),
		"",
		messageStyle.Render("No KECS instances found."),
		"",
		"Press " + keyStyle.Render("'i'") + " " + descStyle.Render("to create your first instance"),
		"Press " + keyStyle.Render("'?'") + " " + descStyle.Render("for help"),
	}

	content := lipgloss.JoinVertical(lipgloss.Center, lines...)

	// Center the content both horizontally and vertically
	return lipgloss.Place(
		m.width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderInstancesList renders the instances list with the given height constraint
func (m Model) renderInstancesList(maxHeight int) string {
	// Colors for instances
	activeColor := lipgloss.Color("#00ff00")
	stoppedColor := lipgloss.Color("#ff0000")
	headerColor := lipgloss.Color("#808080")

	// Styles for instances
	instHeaderStyle := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true)

	activeStyle := lipgloss.NewStyle().
		Foreground(activeColor)

	stoppedStyle := lipgloss.NewStyle().
		Foreground(stoppedColor)

	// Calculate column widths based on available width
	availableWidth := m.width - 8 // Account for padding and borders
	nameWidth := int(float64(availableWidth) * 0.25)
	statusWidth := int(float64(availableWidth) * 0.12)
	clustersWidth := int(float64(availableWidth) * 0.10)
	servicesWidth := int(float64(availableWidth) * 0.10)
	tasksWidth := int(float64(availableWidth) * 0.10)
	portWidth := int(float64(availableWidth) * 0.12)
	ageWidth := int(float64(availableWidth) * 0.10)

	// Header
	header := fmt.Sprintf(
		"%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		nameWidth, "NAME",
		statusWidth, "STATUS",
		clustersWidth, "CLUSTERS",
		servicesWidth, "SERVICES",
		tasksWidth, "TASKS",
		portWidth, "API PORT",
		ageWidth, "AGE",
	)
	header = instHeaderStyle.Render(header)

	// Rows
	rows := []string{header}

	// Get filtered instances
	filteredInstances := m.filterInstances(m.instances)

	// Calculate visible range with scrolling
	visibleRows := maxHeight - 2 // Account for header and potential scroll indicator
	startIdx := 0
	endIdx := len(filteredInstances)

	// Adjust cursor if it's out of bounds
	if m.instanceCursor >= len(filteredInstances) {
		m.instanceCursor = len(filteredInstances) - 1
		if m.instanceCursor < 0 {
			m.instanceCursor = 0
		}
	}

	// Implement scrolling if needed
	if m.instanceCursor >= visibleRows {
		startIdx = m.instanceCursor - visibleRows + 1
	}
	if endIdx > startIdx+visibleRows {
		endIdx = startIdx + visibleRows
	}

	for i := startIdx; i < endIdx; i++ {
		instance := filteredInstances[i]
		// Format values
		name := instance.Name
		status := formatInstanceStatus(instance.Status)
		clusters := fmt.Sprintf("%d", instance.Clusters)
		services := fmt.Sprintf("%d", instance.Services)
		tasks := fmt.Sprintf("%d", instance.Tasks)
		port := fmt.Sprintf("%d", instance.APIPort)
		age := formatDuration(instance.Age)

		// Truncate long values
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}

		// Create row
		row := fmt.Sprintf(
			"%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			nameWidth, name,
			statusWidth, status,
			clustersWidth, clusters,
			servicesWidth, services,
			tasksWidth, tasks,
			portWidth, port,
			ageWidth, age,
		)

		// Apply styles
		if i == m.instanceCursor {
			// Apply full-row highlight with consistent width
			row = selectedRowStyle.Width(availableWidth).Render("▸ " + row)
		} else {
			row = "  " + row
			switch instance.Status {
			case "running", "ACTIVE":
				row = activeStyle.Render(row)
			case "stopped", "STOPPED":
				row = stoppedStyle.Render(row)
			case "pending":
				row = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render(row)
			case "unhealthy":
				row = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff8800")).Render(row)
			}
		}

		rows = append(rows, row)
	}

	// Add scroll indicator if needed
	if len(filteredInstances) > visibleRows {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d instances]", startIdx+1, endIdx, len(filteredInstances))
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(scrollInfo))
	}

	// Add search indicator if searching
	if m.searchMode || m.searchQuery != "" {
		searchInfo := fmt.Sprintf("\n[Search: %s]", m.searchQuery)
		if m.searchMode {
			searchInfo += "_"
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render(searchInfo))
	}

	return strings.Join(rows, "\n")
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	} else {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}

// formatInstanceStatus formats instance status with icons
func formatInstanceStatus(status string) string {
	switch status {
	case "running", "Running", "ACTIVE":
		return "● Running"
	case "stopped", "Stopped", "STOPPED":
		return "○ Stopped"
	case "pending", "Pending":
		return "◐ Pending"
	case "unhealthy", "Unhealthy":
		return "▲ Unhealthy"
	case "starting", "Starting":
		return "◉ Starting"
	case "stopping", "Stopping":
		return "◉ Stopping"
	default:
		return status
	}
}

// renderClustersList renders the clusters list with the given height constraint
func (m Model) renderClustersList(maxHeight int) string {
	if m.selectedInstance == "" {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render("No instance selected. Press 'i' to go to instances.")
	}

	// Colors for clusters
	activeColor := lipgloss.Color("#00ff00")
	headerColor := lipgloss.Color("#808080")

	// Styles for clusters
	clusterHeaderStyle := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true)

	activeStyle := lipgloss.NewStyle().
		Foreground(activeColor)

	// Calculate column widths based on available width
	availableWidth := m.width - 8
	nameWidth := int(float64(availableWidth) * 0.30)
	statusWidth := int(float64(availableWidth) * 0.12)
	regionWidth := int(float64(availableWidth) * 0.15)
	servicesWidth := int(float64(availableWidth) * 0.13)
	tasksWidth := int(float64(availableWidth) * 0.13)
	ageWidth := int(float64(availableWidth) * 0.17)

	// Header
	header := fmt.Sprintf(
		"%-*s %-*s %-*s %-*s %-*s %-*s",
		nameWidth, "NAME",
		statusWidth, "STATUS",
		regionWidth, "REGION",
		servicesWidth, "SERVICES",
		tasksWidth, "TASKS",
		ageWidth, "AGE",
	)
	header = clusterHeaderStyle.Render(header)

	// Rows
	rows := []string{header}

	// Get filtered clusters
	filteredClusters := m.filterClusters(m.clusters)

	// Calculate visible range with scrolling
	visibleRows := maxHeight - 2
	startIdx := 0
	endIdx := len(filteredClusters)

	// Adjust cursor if it's out of bounds
	if m.clusterCursor >= len(filteredClusters) {
		m.clusterCursor = len(filteredClusters) - 1
		if m.clusterCursor < 0 {
			m.clusterCursor = 0
		}
	}

	if m.clusterCursor >= visibleRows {
		startIdx = m.clusterCursor - visibleRows + 1
	}
	if endIdx > startIdx+visibleRows {
		endIdx = startIdx + visibleRows
	}

	for i := startIdx; i < endIdx; i++ {
		cluster := filteredClusters[i]
		// Format values
		name := cluster.Name
		status := cluster.Status
		region := cluster.Region
		services := fmt.Sprintf("%d", cluster.Services)
		tasks := fmt.Sprintf("%d", cluster.Tasks)
		age := formatDuration(cluster.Age)

		// Truncate long values
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}
		if len(region) > regionWidth {
			region = region[:regionWidth-3] + "..."
		}

		// Create row
		row := fmt.Sprintf(
			"%-*s %-*s %-*s %-*s %-*s %-*s",
			nameWidth, name,
			statusWidth, status,
			regionWidth, region,
			servicesWidth, services,
			tasksWidth, tasks,
			ageWidth, age,
		)

		// Apply styles
		if i == m.clusterCursor {
			// Apply full-row highlight with consistent width
			row = selectedRowStyle.Width(availableWidth).Render("▸ " + row)
		} else {
			row = "  " + row
			if cluster.Status == "ACTIVE" {
				row = activeStyle.Render(row)
			}
		}

		rows = append(rows, row)
	}

	// Add scroll indicator if needed
	if len(filteredClusters) > visibleRows {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d clusters]", startIdx+1, endIdx, len(filteredClusters))
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(scrollInfo))
	}

	// Add search indicator if searching
	if m.searchMode || m.searchQuery != "" {
		searchInfo := fmt.Sprintf("\n[Search: %s]", m.searchQuery)
		if m.searchMode {
			searchInfo += "_"
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render(searchInfo))
	}

	return strings.Join(rows, "\n")
}

// renderServicesList renders the services list with the given height constraint
func (m Model) renderServicesList(maxHeight int) string {
	if m.selectedInstance == "" || m.selectedCluster == "" {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render("No cluster selected. Press 'c' to go to clusters.")
	}

	// Colors for services
	activeColor := lipgloss.Color("#00ff00")
	inactiveColor := lipgloss.Color("#0000ff")
	updatingColor := lipgloss.Color("#ffff00")
	provisioningColor := lipgloss.Color("#ff8800")
	headerColor := lipgloss.Color("#808080")

	// Styles for services
	serviceHeaderStyle := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true)

	activeStyle := lipgloss.NewStyle().
		Foreground(activeColor)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(inactiveColor)

	updatingStyle := lipgloss.NewStyle().
		Foreground(updatingColor)

	provisioningStyle := lipgloss.NewStyle().
		Foreground(provisioningColor)

	// Calculate column widths based on available width
	availableWidth := m.width - 8
	nameWidth := int(float64(availableWidth) * 0.20)
	desiredWidth := int(float64(availableWidth) * 0.08)
	runningWidth := int(float64(availableWidth) * 0.08)
	pendingWidth := int(float64(availableWidth) * 0.08)
	statusWidth := int(float64(availableWidth) * 0.10)
	taskDefWidth := int(float64(availableWidth) * 0.40)
	ageWidth := int(float64(availableWidth) * 0.06)

	// Header
	header := fmt.Sprintf(
		"%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		nameWidth, "NAME",
		desiredWidth, "DESIRED",
		runningWidth, "RUNNING",
		pendingWidth, "PENDING",
		statusWidth, "STATUS",
		taskDefWidth, "TASK DEF",
		ageWidth, "AGE",
	)
	header = serviceHeaderStyle.Render(header)

	// Rows
	rows := []string{header}

	// Get filtered services
	filteredServices := m.filterServices(m.services)

	// Calculate visible range with scrolling
	visibleRows := maxHeight - 2
	startIdx := 0
	endIdx := len(filteredServices)

	// Adjust cursor if it's out of bounds
	if m.serviceCursor >= len(filteredServices) {
		m.serviceCursor = len(filteredServices) - 1
		if m.serviceCursor < 0 {
			m.serviceCursor = 0
		}
	}

	if m.serviceCursor >= visibleRows {
		startIdx = m.serviceCursor - visibleRows + 1
	}
	if endIdx > startIdx+visibleRows {
		endIdx = startIdx + visibleRows
	}

	for i := startIdx; i < endIdx; i++ {
		service := filteredServices[i]
		// Format values
		name := service.Name
		desired := fmt.Sprintf("%d", service.Desired)
		running := fmt.Sprintf("%d", service.Running)
		pending := fmt.Sprintf("%d", service.Pending)
		status := service.Status
		taskDef := service.TaskDef
		age := formatDuration(service.Age)

		// Extract task definition name and revision from ARN
		// ARN format: arn:aws:ecs:region:account:task-definition/name:revision
		if strings.HasPrefix(taskDef, "arn:") {
			parts := strings.Split(taskDef, "/")
			if len(parts) > 1 {
				taskDef = parts[len(parts)-1] // Gets "name:revision"
			}
		}

		// Truncate long values
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}
		if len(taskDef) > taskDefWidth {
			taskDef = taskDef[:taskDefWidth-3] + "..."
		}

		// Create row
		row := fmt.Sprintf(
			"%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			nameWidth, name,
			desiredWidth, desired,
			runningWidth, running,
			pendingWidth, pending,
			statusWidth, status,
			taskDefWidth, taskDef,
			ageWidth, age,
		)

		// Apply styles
		if i == m.serviceCursor {
			// Apply full-row highlight with consistent width
			row = selectedRowStyle.Width(availableWidth).Render("▸ " + row)
		} else {
			row = "  " + row
			switch service.Status {
			case "ACTIVE":
				row = activeStyle.Render(row)
			case "INACTIVE":
				row = inactiveStyle.Render(row)
			case "UPDATING":
				row = updatingStyle.Render(row)
			case "PROVISIONING":
				row = provisioningStyle.Render(row)
			}
		}

		rows = append(rows, row)
	}

	// Add scroll indicator if needed
	if len(filteredServices) > visibleRows {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d services]", startIdx+1, endIdx, len(filteredServices))
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(scrollInfo))
	}

	// Add search indicator if searching
	if m.searchMode || m.searchQuery != "" {
		searchInfo := fmt.Sprintf("\n[Search: %s]", m.searchQuery)
		if m.searchMode {
			searchInfo += "_"
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render(searchInfo))
	}

	return strings.Join(rows, "\n")
}

// renderTasksList renders the tasks list with the given height constraint
func (m Model) renderTasksList(maxHeight int) string {
	if m.selectedInstance == "" || m.selectedCluster == "" {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render("No cluster selected.")
	}

	// Colors for tasks
	runningColor := lipgloss.Color("#00ff00")
	pendingColor := lipgloss.Color("#ffff00")
	stoppingColor := lipgloss.Color("#ff8800")
	failedColor := lipgloss.Color("#ff0000")
	healthyColor := lipgloss.Color("#00ff00")
	unhealthyColor := lipgloss.Color("#ff0000")
	unknownColor := lipgloss.Color("#808080")
	headerColor := lipgloss.Color("#808080")

	// Styles for tasks
	taskHeaderStyle := lipgloss.NewStyle().
		Foreground(headerColor).
		Bold(true)

	runningStyle := lipgloss.NewStyle().
		Foreground(runningColor)

	pendingStyle := lipgloss.NewStyle().
		Foreground(pendingColor)

	stoppingStyle := lipgloss.NewStyle().
		Foreground(stoppingColor)

	failedStyle := lipgloss.NewStyle().
		Foreground(failedColor)

	// Calculate column widths based on available width
	availableWidth := m.width - 8
	idWidth := int(float64(availableWidth) * 0.45)
	statusWidth := int(float64(availableWidth) * 0.12)
	healthWidth := int(float64(availableWidth) * 0.10)
	cpuWidth := int(float64(availableWidth) * 0.08)
	memoryWidth := int(float64(availableWidth) * 0.10)
	ipWidth := int(float64(availableWidth) * 0.10)
	ageWidth := int(float64(availableWidth) * 0.05)

	// Header
	header := fmt.Sprintf(
		"%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		idWidth, "TASK ID",
		statusWidth, "STATUS",
		healthWidth, "HEALTH",
		cpuWidth, "CPU",
		memoryWidth, "MEMORY",
		ipWidth, "IP",
		ageWidth, "AGE",
	)
	header = taskHeaderStyle.Render(header)

	// Rows
	rows := []string{header}

	// Get filtered tasks
	filteredTasks := m.filterTasks(m.tasks)

	// If no tasks, show appropriate message
	if len(filteredTasks) == 0 {
		emptyMsg := ""
		if m.selectedService == "" {
			emptyMsg = "No tasks in this cluster. Press 'r' to refresh."
		} else {
			emptyMsg = "No tasks for this service. Press 'r' to refresh."
		}
		rows = append(rows, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render(emptyMsg))
		return strings.Join(rows, "\n")
	}

	// Calculate visible range with scrolling
	visibleRows := maxHeight - 2
	startIdx := 0
	endIdx := len(filteredTasks)

	// Adjust cursor if it's out of bounds
	if m.taskCursor >= len(filteredTasks) {
		m.taskCursor = len(filteredTasks) - 1
		if m.taskCursor < 0 {
			m.taskCursor = 0
		}
	}

	if m.taskCursor >= visibleRows {
		startIdx = m.taskCursor - visibleRows + 1
	}
	if endIdx > startIdx+visibleRows {
		endIdx = startIdx + visibleRows
	}

	for i := startIdx; i < endIdx; i++ {
		task := filteredTasks[i]
		// Format values
		id := task.ID
		status := task.Status
		health := task.Health
		cpu := fmt.Sprintf("%.1f", task.CPU)
		memory := task.Memory
		ip := task.IP
		age := formatDuration(task.Age)

		// Handle pending tasks
		if status == "PENDING" {
			cpu = "-"
			memory = "-"
			ip = "-"
		}

		// Truncate long values
		if len(id) > idWidth {
			id = id[:idWidth-3] + "..."
		}

		// Create row
		row := fmt.Sprintf(
			"%-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			idWidth, id,
			statusWidth, status,
			healthWidth, health,
			cpuWidth, cpu,
			memoryWidth, memory,
			ipWidth, ip,
			ageWidth, age,
		)

		// Apply styles
		if i == m.taskCursor {
			// Apply full-row highlight with consistent width
			row = selectedRowStyle.Width(availableWidth).Render("▸ " + row)
		} else {
			row = "  " + row
			// Color based on status and health
			switch status {
			case "RUNNING":
				switch health {
				case "HEALTHY":
					row = lipgloss.NewStyle().Foreground(healthyColor).Render(row)
				case "UNHEALTHY":
					row = lipgloss.NewStyle().Foreground(unhealthyColor).Render(row)
				default:
					row = runningStyle.Render(row)
				}
			case "PENDING":
				row = pendingStyle.Render(row)
			case "STOPPING":
				row = stoppingStyle.Render(row)
			case "FAILED":
				row = failedStyle.Render(row)
			default:
				row = lipgloss.NewStyle().Foreground(unknownColor).Render(row)
			}
		}

		rows = append(rows, row)
	}

	// Add scroll indicator if needed
	if len(filteredTasks) > visibleRows {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d tasks]", startIdx+1, endIdx, len(filteredTasks))
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(scrollInfo))
	}

	// Add search indicator if searching
	if m.searchMode || m.searchQuery != "" {
		searchInfo := fmt.Sprintf("\n[Search: %s]", m.searchQuery)
		if m.searchMode {
			searchInfo += "_"
		}
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render(searchInfo))
	}

	return strings.Join(rows, "\n")
}

// renderViewShortcutsLine renders view-specific shortcuts in a single line
func (m Model) renderViewShortcutsLine(width int) string {
	// Style for shortcuts
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f5c2e7")).Bold(true)

	// Get view-specific shortcuts
	viewBindings, _ := m.keyBindings.GetAllBindingsForView(m.currentView, m)

	if len(viewBindings) == 0 {
		return ""
	}

	// Build shortcuts string
	var shortcuts []string
	for _, binding := range viewBindings {
		if binding.Condition == nil || binding.Condition(m) {
			keyStr := FormatKeyString(binding.Keys)
			shortcut := keyStyle.Render(keyStr) + sepStyle.Render(" "+binding.Description)
			shortcuts = append(shortcuts, shortcut)
		}
	}

	// Limit to fit on one line
	line := headerStyle.Render("View: ") + strings.Join(shortcuts, "  ")

	// Truncate if too long
	if lipgloss.Width(line) > width {
		// Show only the most important shortcuts
		if len(shortcuts) > 3 {
			shortcuts = shortcuts[:3]
			line = headerStyle.Render("View: ") + strings.Join(shortcuts, "  ") + sepStyle.Render("  ...")
		}
	}

	return line
}

// renderGlobalShortcutsLine renders global shortcuts in a single line
func (m Model) renderGlobalShortcutsLine(width int) string {
	// Style for shortcuts
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086"))
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f5c2e7")).Bold(true)

	// Get global shortcuts
	_, globalBindings := m.keyBindings.GetAllBindingsForView(m.currentView, m)

	// Build shortcuts string - focus on most important ones
	var shortcuts []string

	// Priority order for global shortcuts to show
	priorityActions := []KeyAction{
		ActionMoveUp,
		ActionMoveDown,
		ActionBack,
		ActionToggleInstance,
		ActionDeleteInstance,
		ActionHelp,
		ActionRefresh,
		ActionQuit,
	}

	for _, action := range priorityActions {
		for _, binding := range globalBindings {
			if binding.Action == action {
				keyStr := FormatKeyString(binding.Keys)
				shortcut := keyStyle.Render(keyStr) + sepStyle.Render(" "+binding.Description)
				shortcuts = append(shortcuts, shortcut)
				break
			}
		}
	}

	line := headerStyle.Render("Global: ") + strings.Join(shortcuts, "  ")

	// Truncate if too long
	if lipgloss.Width(line) > width {
		// Show fewer shortcuts
		if len(shortcuts) > 4 {
			shortcuts = shortcuts[:4]
			line = headerStyle.Render("Global: ") + strings.Join(shortcuts, "  ") + sepStyle.Render("  ...")
		}
	}

	return line
}

// renderShortcutsColumn renders the shortcuts column for the navigation panel
func (m Model) renderShortcutsColumn(width int) string {
	// Style for k9s-like shortcuts
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9399b2"))

	// Container style for shortcuts column
	columnStyle := lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#45475a")).
		PaddingLeft(1).
		Width(width)

	// Get all shortcuts
	viewBindings, globalBindings := m.keyBindings.GetAllBindingsForView(m.currentView, m)

	// For narrow columns (< 80), use single column layout
	if width < 80 {
		var allShortcuts []string

		// Add view-specific shortcuts first
		if len(viewBindings) > 0 {
			// Add section header
			allShortcuts = append(allShortcuts, lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f5c2e7")).
				Bold(true).
				Render("View Shortcuts"))

			for _, binding := range viewBindings {
				if binding.Condition == nil || binding.Condition(m) {
					keyStr := FormatKeyString(binding.Keys)
					shortcut := fmt.Sprintf("%s %s",
						keyStyle.Render(keyStr),
						descStyle.Render(binding.Description))
					allShortcuts = append(allShortcuts, shortcut)
				}
			}
		}

		// Add global shortcuts header
		allShortcuts = append(allShortcuts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f5c2e7")).
			Bold(true).
			Render("Global Shortcuts"))

		// Format global shortcuts in 2 columns if there are enough
		if len(globalBindings) >= 6 && width >= 60 {
			// Split into two columns
			mid := (len(globalBindings) + 1) / 2
			colWidth := (width - 10) / 2 // Leave some padding

			for i := 0; i < mid; i++ {
				// Left column item
				leftBinding := globalBindings[i]
				leftKeyStr := FormatKeyString(leftBinding.Keys)
				leftShortcut := fmt.Sprintf("%s %s",
					keyStyle.Render(leftKeyStr),
					descStyle.Render(leftBinding.Description))

				// Right column item (if exists)
				rightShortcut := ""
				if i+mid < len(globalBindings) {
					rightBinding := globalBindings[i+mid]
					rightKeyStr := FormatKeyString(rightBinding.Keys)
					rightShortcut = fmt.Sprintf("%s %s",
						keyStyle.Render(rightKeyStr),
						descStyle.Render(rightBinding.Description))
				}

				// Combine both columns
				line := lipgloss.JoinHorizontal(
					lipgloss.Top,
					lipgloss.NewStyle().Width(colWidth).Render(leftShortcut),
					lipgloss.NewStyle().Width(colWidth).Render(rightShortcut),
				)
				allShortcuts = append(allShortcuts, line)
			}
		} else {
			// Single column for narrow terminals or few shortcuts
			for _, binding := range globalBindings {
				keyStr := FormatKeyString(binding.Keys)
				shortcut := fmt.Sprintf("%s %s",
					keyStyle.Render(keyStr),
					descStyle.Render(binding.Description))
				allShortcuts = append(allShortcuts, shortcut)
			}
		}

		// Special case shortcuts for task def JSON view
		if m.currentView == ViewTaskDefinitionRevisions && m.showTaskDefJSON {
			allShortcuts = append(allShortcuts,
				keyStyle.Render("^U")+" "+descStyle.Render("Scroll up"),
				keyStyle.Render("^D")+" "+descStyle.Render("Scroll down"),
			)
		}

		content := strings.Join(allShortcuts, "\n")
		// Don't restrict height, let it flow naturally
		return columnStyle.Render(content)
	}

	// For wider columns, show shortcuts in a single column with sections
	var allShortcuts []string

	// Add view-specific shortcuts first
	if len(viewBindings) > 0 {
		// Add header for view shortcuts
		allShortcuts = append(allShortcuts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f5c2e7")).
			Bold(true).
			Render("View Shortcuts"))

		for _, binding := range viewBindings {
			if binding.Condition == nil || binding.Condition(m) {
				keyStr := FormatKeyString(binding.Keys)
				shortcut := fmt.Sprintf("%s %s",
					keyStyle.Render(keyStr),
					descStyle.Render(binding.Description))
				allShortcuts = append(allShortcuts, shortcut)
			}
		}

		// Add special case shortcuts that aren't in the registry yet
		if m.currentView == ViewTaskDefinitionRevisions && m.showTaskDefJSON {
			allShortcuts = append(allShortcuts,
				keyStyle.Render("^U")+" "+descStyle.Render("Scroll up"),
				keyStyle.Render("^D")+" "+descStyle.Render("Scroll down"),
			)
		}

		// Add a blank line between sections
		allShortcuts = append(allShortcuts, "")
	}

	// Add header for global shortcuts
	allShortcuts = append(allShortcuts, lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f5c2e7")).
		Bold(true).
		Render("Global Shortcuts"))

	// Format global shortcuts in 2 columns if space allows
	effectiveWidth := width - 4 // Account for padding and borders
	if len(globalBindings) >= 6 && effectiveWidth >= 60 {
		// Split into two columns
		mid := (len(globalBindings) + 1) / 2
		colWidth := effectiveWidth / 2

		for i := 0; i < mid; i++ {
			// Left column item
			leftBinding := globalBindings[i]
			leftKeyStr := FormatKeyString(leftBinding.Keys)
			leftShortcut := fmt.Sprintf("%s %s",
				keyStyle.Render(leftKeyStr),
				descStyle.Render(leftBinding.Description))

			// Right column item (if exists)
			rightShortcut := ""
			if i+mid < len(globalBindings) {
				rightBinding := globalBindings[i+mid]
				rightKeyStr := FormatKeyString(rightBinding.Keys)
				rightShortcut = fmt.Sprintf("%s %s",
					keyStyle.Render(rightKeyStr),
					descStyle.Render(rightBinding.Description))
			}

			// Combine both columns with proper spacing
			line := lipgloss.JoinHorizontal(
				lipgloss.Top,
				lipgloss.NewStyle().Width(colWidth).MaxWidth(colWidth).Render(leftShortcut),
				lipgloss.NewStyle().Width(colWidth).MaxWidth(colWidth).Render(rightShortcut),
			)
			allShortcuts = append(allShortcuts, line)
		}
	} else {
		// Single column for narrow space or few shortcuts
		for _, binding := range globalBindings {
			keyStr := FormatKeyString(binding.Keys)
			shortcut := fmt.Sprintf("%s %s",
				keyStyle.Render(keyStr),
				descStyle.Render(binding.Description))
			allShortcuts = append(allShortcuts, shortcut)
		}
	}

	content := strings.Join(allShortcuts, "\n")
	// Don't restrict height, let it flow naturally
	return columnStyle.Render(content)
}

// renderShortcutsInline renders shortcuts in a compact inline format for narrow terminals
func (m Model) renderShortcutsInline(width int) string {
	// Style for inline shortcuts
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6e3a1")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9399b2"))

	// Get all shortcuts
	viewBindings, globalBindings := m.keyBindings.GetAllBindingsForView(m.currentView, m)

	// Build inline shortcuts string
	var shortcuts []string

	// Add most important view-specific shortcuts (limit to 3-4)
	count := 0
	for _, binding := range viewBindings {
		if count >= 3 {
			break
		}
		if binding.Condition == nil || binding.Condition(m) {
			keyStr := FormatKeyString(binding.Keys)
			shortcuts = append(shortcuts, fmt.Sprintf("%s %s",
				keyStyle.Render(keyStr),
				descStyle.Render(binding.Description)))
			count++
		}
	}

	// Add essential global shortcuts
	essentialGlobals := []KeyAction{ActionBack, ActionHelp, ActionQuit}
	for _, action := range essentialGlobals {
		for _, binding := range globalBindings {
			if binding.Action == action {
				keyStr := FormatKeyString(binding.Keys)
				shortcuts = append(shortcuts, fmt.Sprintf("%s %s",
					keyStyle.Render(keyStr),
					descStyle.Render(binding.Description)))
				break
			}
		}
	}

	// Join with separator
	return strings.Join(shortcuts, " │ ")
}

// renderLogsContent renders the logs content with the given height constraint
func (m Model) renderLogsContent(maxHeight int) string {
	if m.selectedTask == "" {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render("No task selected for log viewing. Select a task and press 'l' to view logs.")
	}

	// Colors for logs
	infoColor := lipgloss.Color("#ffffff")
	warnColor := lipgloss.Color("#ffff00")
	errorColor := lipgloss.Color("#ff0000")
	debugColor := lipgloss.Color("#808080")
	timestampColor := lipgloss.Color("#00ffff")

	// Styles for logs
	infoStyle := lipgloss.NewStyle().
		Foreground(infoColor)

	warnStyle := lipgloss.NewStyle().
		Foreground(warnColor)

	errorStyle := lipgloss.NewStyle().
		Foreground(errorColor)

	debugStyle := lipgloss.NewStyle().
		Foreground(debugColor)

	timestampStyle := lipgloss.NewStyle().
		Foreground(timestampColor)

	// Calculate available space
	availableWidth := m.width - 8 // Account for padding and borders
	availableHeight := maxHeight

	// Build log lines
	lines := []string{}

	// Get filtered logs
	filteredLogs := m.filterLogs(m.logs)

	// Adjust cursor if it's out of bounds
	if m.logCursor >= len(filteredLogs) {
		m.logCursor = len(filteredLogs) - 1
		if m.logCursor < 0 {
			m.logCursor = 0
		}
	}

	// Process logs
	startIdx := m.logCursor
	endIdx := startIdx + availableHeight
	if endIdx > len(filteredLogs) {
		endIdx = len(filteredLogs)
	}

	for i := startIdx; i < endIdx && i < len(filteredLogs); i++ {
		log := filteredLogs[i]

		// Format timestamp
		timestamp := log.Timestamp.Format("2006-01-02 15:04:05")
		timestampStr := timestampStyle.Render(timestamp)

		// Format level
		levelStr := fmt.Sprintf("[%-5s]", log.Level)
		switch log.Level {
		case "INFO":
			levelStr = infoStyle.Render(levelStr)
		case "WARN":
			levelStr = warnStyle.Render(levelStr)
		case "ERROR":
			levelStr = errorStyle.Render(levelStr)
		case "DEBUG":
			levelStr = debugStyle.Render(levelStr)
		}

		// Format message
		message := log.Message
		maxMessageWidth := availableWidth - len(timestamp) - len("[DEBUG]") - 4
		if len(message) > maxMessageWidth {
			message = message[:maxMessageWidth-3] + "..."
		}

		// Combine parts
		logLine := fmt.Sprintf("%s %s %s", timestampStr, levelStr, message)

		// Apply selection if this is the current line
		if i == m.logCursor {
			// Apply full-row highlight with consistent width
			logLine = selectedRowStyle.Width(availableWidth).Render("▸ " + logLine)
		} else {
			logLine = "  " + logLine
		}

		lines = append(lines, logLine)
	}

	// Add scroll indicator if needed
	if len(filteredLogs) > availableHeight {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d log entries]", startIdx+1, endIdx, len(filteredLogs))
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(scrollInfo))
	}

	// Add search indicator if searching
	if m.searchMode || m.searchQuery != "" {
		searchInfo := fmt.Sprintf("\n[Search: %s]", m.searchQuery)
		if m.searchMode {
			searchInfo += "_"
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render(searchInfo))
	}

	return strings.Join(lines, "\n")
}

// renderTaskDescribeView renders the task describe view
func (m Model) renderTaskDescribeView() string {
	// Find the selected task
	var selectedTask *Task
	for i := range m.tasks {
		if m.tasks[i].ID == m.selectedTask {
			selectedTask = &m.tasks[i]
			break
		}
	}

	if selectedTask == nil {
		return "Task not found"
	}

	// Header style
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1e1e2e")).
		Foreground(lipgloss.Color("#cdd6f4")).
		Padding(0, 1).
		Bold(true)

	// Section style
	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6e3a1")).
		Bold(true).
		MarginTop(1)

	// Label style
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9399b2"))

	// Value style
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdd6f4"))

	// Status styles
	runningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6e3a1"))
	stoppedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f38ba8"))
	pendingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fab387"))

	// Health styles
	healthyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6e3a1"))
	unhealthyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f38ba8"))
	unknownStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9399b2"))

	// Build content
	content := []string{}

	// Header
	header := headerStyle.Width(m.width).Render("Task Details - Press ESC to go back")
	content = append(content, header)

	// Basic Information
	content = append(content, sectionStyle.Render("\n▸ Basic Information"))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Task ID:"), valueStyle.Render(selectedTask.ID)))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Service:"), valueStyle.Render(selectedTask.Service)))

	// Status
	statusStr := selectedTask.Status
	switch selectedTask.Status {
	case "RUNNING":
		statusStr = runningStyle.Render("● " + statusStr)
	case "STOPPED":
		statusStr = stoppedStyle.Render("○ " + statusStr)
	case "PENDING":
		statusStr = pendingStyle.Render("◐ " + statusStr)
	}
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Status:"), statusStr))

	// Health
	healthStr := selectedTask.Health
	switch selectedTask.Health {
	case "HEALTHY":
		healthStr = healthyStyle.Render("✓ " + healthStr)
	case "UNHEALTHY":
		healthStr = unhealthyStyle.Render("✗ " + healthStr)
	default:
		healthStr = unknownStyle.Render("? " + healthStr)
	}
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Health:"), healthStr))

	// Resource Information
	content = append(content, sectionStyle.Render("\n▸ Resource Usage"))
	content = append(content, fmt.Sprintf("  %s %.2f%%", labelStyle.Render("CPU:"), selectedTask.CPU))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Memory:"), valueStyle.Render(selectedTask.Memory)))

	// Network Information
	content = append(content, sectionStyle.Render("\n▸ Network Information"))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("IP Address:"), valueStyle.Render(selectedTask.IP)))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Port Mappings:"), valueStyle.Render("80:8080, 443:8443")))

	// Container Information (Mock)
	content = append(content, sectionStyle.Render("\n▸ Container Information"))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Image:"), valueStyle.Render("nginx:latest")))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Container ID:"), valueStyle.Render("abc123def456")))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Runtime:"), valueStyle.Render("docker")))

	// Environment Variables (Mock)
	content = append(content, sectionStyle.Render("\n▸ Environment Variables"))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("ENV:"), valueStyle.Render("production")))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("LOG_LEVEL:"), valueStyle.Render("info")))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("DATABASE_URL:"), valueStyle.Render("postgres://...")))

	// Timestamps
	content = append(content, sectionStyle.Render("\n▸ Timestamps"))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Age:"), valueStyle.Render(selectedTask.Age.String())))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Created:"), valueStyle.Render("2024-01-15 10:30:00")))
	content = append(content, fmt.Sprintf("  %s %s", labelStyle.Render("Started:"), valueStyle.Render("2024-01-15 10:30:15")))

	// Footer with shortcuts
	footerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#313244")).
		Foreground(lipgloss.Color("#9399b2")).
		Padding(0, 1).
		MarginTop(2)

	footer := footerStyle.Width(m.width).Render("ESC: Back  |  l: View Logs  |  r: Restart  |  s: Stop")
	content = append(content, footer)

	// Join all content
	return lipgloss.JoinVertical(lipgloss.Top, content...)
}

// renderHelpContent renders the help content with the given height constraint
func (m Model) renderHelpContent(maxHeight int) string {
	// Style definitions
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#cdd6f4")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#89b4fa")).
		MarginTop(1).
		MarginBottom(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6e3a1")).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9399b2"))

	subsectionStyle := lipgloss.NewStyle().
		Bold(true).
		Underline(true)

	// Calculate column widths
	totalWidth := m.width - 8 // Account for padding
	columnWidth := totalWidth/2 - 2

	// Build left column: View-specific operations
	leftContent := []string{
		sectionStyle.Render("View-Specific Operations"),
		"",
	}

	// Group view-specific bindings by view
	viewGroups := map[string][]KeyBinding{
		"Instances":        {},
		"Clusters":         {},
		"Services":         {},
		"Tasks":            {},
		"Task Definitions": {},
		"Logs":             {},
	}

	// Define view mappings in a stable order
	viewMappings := []struct {
		view     ViewType
		viewName string
	}{
		{ViewInstances, "Instances"},
		{ViewClusters, "Clusters"},
		{ViewServices, "Services"},
		{ViewTasks, "Tasks"},
		{ViewTaskDefinitionFamilies, "Task Definitions"},
		{ViewTaskDefinitionRevisions, "Task Definitions"},
		{ViewLogs, "Logs"},
	}

	// Collect bindings for each view in stable order
	for _, mapping := range viewMappings {
		bindings := m.keyBindings.GetViewBindings(mapping.view)
		for _, binding := range bindings {
			// Filter out navigation bindings that will be shown in global
			if binding.Action == ActionMoveUp || binding.Action == ActionMoveDown ||
				binding.Action == ActionBack || binding.Action == ActionQuit {
				continue
			}
			viewGroups[mapping.viewName] = append(viewGroups[mapping.viewName], binding)
		}
	}

	// Add each view's bindings
	for _, viewName := range []string{"Instances", "Clusters", "Services", "Tasks", "Task Definitions", "Logs"} {
		bindings := viewGroups[viewName]
		if len(bindings) > 0 {
			leftContent = append(leftContent, subsectionStyle.Render(viewName+" Operations:"))
			for _, binding := range bindings {
				keyStr := FormatKeyString(binding.Keys)
				// Pad key string for alignment
				formattedKey := fmt.Sprintf("%-12s", keyStyle.Render(keyStr))
				leftContent = append(leftContent, formattedKey+"  "+descStyle.Render(binding.Description))
			}
			leftContent = append(leftContent, "")
		}
	}

	// Build right column: Global operations
	rightContent := []string{
		sectionStyle.Render("Global Shortcuts"),
		"",
	}

	// Get global bindings
	globalBindings := m.keyBindings.GetGlobalBindings()

	// Group global bindings by category
	navigationBindings := []KeyBinding{}
	instanceBindings := []KeyBinding{}
	utilityBindings := []KeyBinding{}
	applicationBindings := []KeyBinding{}

	for _, binding := range globalBindings {
		switch binding.Action {
		case ActionMoveUp, ActionMoveDown, ActionBack, ActionGoHome:
			navigationBindings = append(navigationBindings, binding)
		case ActionToggleInstance, ActionSwitchInstance, ActionDeleteInstance:
			instanceBindings = append(instanceBindings, binding)
		case ActionHelp, ActionSearch, ActionCommand, ActionRefresh:
			utilityBindings = append(utilityBindings, binding)
		case ActionQuit:
			applicationBindings = append(applicationBindings, binding)
		}
	}

	// Add navigation bindings
	if len(navigationBindings) > 0 {
		rightContent = append(rightContent, subsectionStyle.Render("Navigation:"))
		for _, binding := range navigationBindings {
			keyStr := FormatKeyString(binding.Keys)
			formattedKey := fmt.Sprintf("%-12s", keyStyle.Render(keyStr))
			rightContent = append(rightContent, formattedKey+"  "+descStyle.Render(binding.Description))
		}
		rightContent = append(rightContent, "")
	}

	// Add instance management bindings
	if len(instanceBindings) > 0 {
		rightContent = append(rightContent, subsectionStyle.Render("Instance Management:"))
		for _, binding := range instanceBindings {
			keyStr := FormatKeyString(binding.Keys)
			formattedKey := fmt.Sprintf("%-12s", keyStyle.Render(keyStr))
			rightContent = append(rightContent, formattedKey+"  "+descStyle.Render(binding.Description))
		}
		rightContent = append(rightContent, "")
	}

	// Add utility bindings
	if len(utilityBindings) > 0 {
		rightContent = append(rightContent, subsectionStyle.Render("Utilities:"))
		for _, binding := range utilityBindings {
			keyStr := FormatKeyString(binding.Keys)
			formattedKey := fmt.Sprintf("%-12s", keyStyle.Render(keyStr))
			rightContent = append(rightContent, formattedKey+"  "+descStyle.Render(binding.Description))
		}
		rightContent = append(rightContent, "")
	}

	// Add application bindings
	if len(applicationBindings) > 0 {
		rightContent = append(rightContent, subsectionStyle.Render("Application:"))
		for _, binding := range applicationBindings {
			keyStr := FormatKeyString(binding.Keys)
			formattedKey := fmt.Sprintf("%-12s", keyStyle.Render(keyStr))
			rightContent = append(rightContent, formattedKey+"  "+descStyle.Render(binding.Description))
		}
		rightContent = append(rightContent, "")
	}

	// Add special shortcuts not in the registry
	rightContent = append(rightContent, subsectionStyle.Render("Special Keys:"))
	rightContent = append(rightContent, fmt.Sprintf("%-12s", keyStyle.Render("Tab"))+"  "+descStyle.Render("Switch to next instance"))
	rightContent = append(rightContent, fmt.Sprintf("%-12s", keyStyle.Render("Shift+Tab"))+"  "+descStyle.Render("Switch to previous instance"))

	// Create styled columns
	leftColumn := lipgloss.NewStyle().
		Width(columnWidth).
		Padding(0, 2).
		Render(strings.Join(leftContent, "\n"))

	rightColumn := lipgloss.NewStyle().
		Width(columnWidth).
		Padding(0, 2).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#45475a")).
		Render(strings.Join(rightContent, "\n"))

	// Join columns
	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		rightColumn,
	)

	// Add title and footer
	title := titleStyle.Render("KECS TUI Help - Keyboard Shortcuts")
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6c7086")).
		MarginTop(2).
		Render("Press ? to close help")

	// Combine all parts
	helpView := lipgloss.JoinVertical(
		lipgloss.Top,
		title,
		content,
		footer,
	)

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(helpView)
}

func (m Model) renderHelpView() string {
	return m.renderHelpContent(m.height - 1)
}

// renderInstancesView renders the original instances view (for backward compatibility)
func (m Model) renderInstancesView() string {
	return m.renderInstancesList(m.height - 4)
}

// renderClustersView renders the original clusters view (for backward compatibility)
func (m Model) renderClustersView() string {
	return m.renderClustersList(m.height - 4)
}

// renderServicesView renders the original services view (for backward compatibility)
func (m Model) renderServicesView() string {
	return m.renderServicesList(m.height - 4)
}

// renderTasksView renders the original tasks view (for backward compatibility)
func (m Model) renderTasksView() string {
	return m.renderTasksList(m.height - 4)
}

// renderLogsView renders the original logs view (for backward compatibility)
func (m Model) renderLogsView() string {
	return m.renderLogsContent(m.height - 4)
}

// renderConfirmDialogOverlay renders the confirmation dialog as an overlay
func (m Model) renderConfirmDialogOverlay() string {
	if m.confirmDialog == nil {
		// Fallback to normal view if no dialog
		return m.View()
	}

	// Simply render the dialog centered on screen
	return m.confirmDialog.Render(m.width, m.height)
}

// renderInstanceSwitcherOverlay renders the instance switcher as an overlay
func (m Model) renderInstanceSwitcherOverlay() string {
	if m.instanceSwitcher == nil {
		// Safety check - fallback to regular view
		return m.View()
	}

	// Simply render the switcher centered on screen
	return m.instanceSwitcher.Render(m.width, m.height)
}

// renderDeletingOverlay renders the deletion progress overlay
func (m Model) renderDeletingOverlay() string {
	// Create overlay style
	overlayStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center)

	// Create dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#585b70")).
		Padding(2, 4).
		Background(lipgloss.Color("#1e1e2e"))

	// Create spinner with message
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		m.spinner.View()+" "+m.deletingMessage,
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Render("Please wait..."),
	)

	dialog := dialogStyle.Render(content)
	return overlayStyle.Render(dialog)
}

// renderTaskDefFamiliesList renders the list of task definition families
func (m Model) renderTaskDefFamiliesList(maxHeight int) string {
	if len(m.taskDefFamilies) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render("No task definition families found.")
	}

	// Filter families
	filteredFamilies := m.filterTaskDefFamilies(m.taskDefFamilies)
	if len(filteredFamilies) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render("No families match the search criteria.")
	}

	// Column headers
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00ff00")).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder())

	headers := fmt.Sprintf("%-30s %4s %6s %6s %12s",
		"FAMILY NAME", "REV", "ACTIVE", "TOTAL", "UPDATED")

	// Styles
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#005577")).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true)

	normalStyle := lipgloss.NewStyle()

	// Build rows
	var rows []string
	rows = append(rows, headerStyle.Render(headers))

	visibleRows := maxHeight - 2 // Header and spacing
	startIdx := 0
	if m.taskDefFamilyCursor >= visibleRows {
		startIdx = m.taskDefFamilyCursor - visibleRows + 1
	}

	endIdx := startIdx + visibleRows
	if endIdx > len(filteredFamilies) {
		endIdx = len(filteredFamilies)
	}

	for i := startIdx; i < endIdx; i++ {
		family := filteredFamilies[i]

		// Format row
		row := fmt.Sprintf("%-30s %4d %6d %6d %12s",
			truncateString(family.Family, 30),
			family.LatestRevision,
			family.ActiveCount,
			family.TotalCount,
			formatDuration(time.Since(family.LastUpdated)),
		)

		// Apply style
		if i == m.taskDefFamilyCursor {
			row = selectedStyle.Render("▸ " + row)
		} else {
			row = normalStyle.Render("  " + row)
		}

		rows = append(rows, row)
	}

	// Add scroll indicator
	if len(filteredFamilies) > visibleRows {
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d families", startIdx+1, endIdx, len(filteredFamilies))
		rows = append(rows, "", lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Render(scrollInfo))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderTaskDefinitionRevisionsView renders the task definition revisions view

// renderTaskDefRevisionsTwoColumn renders two column view with JSON
func (m Model) renderTaskDefRevisionsTwoColumn(maxHeight int) string {
	// Calculate dimensions
	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth - 1 // -1 for border

	// Render components
	leftColumn := m.renderTaskDefRevisionsList(maxHeight, leftWidth)
	rightColumn := m.renderTaskDefJSON(maxHeight, rightWidth)

	// Combine columns
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
		lipgloss.NewStyle().
			Height(maxHeight).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			Render(""),
		rightColumn,
	)
}

// renderTaskDefRevisionsList renders the list of revisions
func (m Model) renderTaskDefRevisionsList(maxHeight int, width int) string {
	if len(m.taskDefRevisions) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Width(width).
			Render("No revisions found.")
	}

	// Column headers
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00ff00")).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder())

	// Adjust column widths to include family
	var headers string
	if width > 60 {
		// Wide view with family column
		headers = fmt.Sprintf("%-30s %-4s %-8s %-10s %-10s",
			"FAMILY", "REV", "STATUS", "CPU/MEM", "CREATED")
	} else {
		// Narrow view without family (for two-column mode)
		headers = fmt.Sprintf("%-4s %-10s %-10s %-12s",
			"REV", "STATUS", "CPU/MEM", "CREATED")
	}

	// Styles
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#005577")).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true).
		Width(width)

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff00"))

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999"))

	normalStyle := lipgloss.NewStyle()

	// Build rows
	var rows []string
	rows = append(rows, headerStyle.Width(width).Render(headers))

	visibleRows := maxHeight - 2
	startIdx := 0
	if m.taskDefRevisionCursor >= visibleRows {
		startIdx = m.taskDefRevisionCursor - visibleRows + 1
	}

	endIdx := startIdx + visibleRows
	if endIdx > len(m.taskDefRevisions) {
		endIdx = len(m.taskDefRevisions)
	}

	for i := startIdx; i < endIdx; i++ {
		rev := m.taskDefRevisions[i]

		// Format row
		cpuMem := fmt.Sprintf("%s/%s", rev.CPU, rev.Memory)
		var row string
		if width > 60 {
			// Wide view with family column
			familyName := rev.Family
			if len(familyName) > 28 {
				familyName = familyName[:25] + "..."
			}
			row = fmt.Sprintf("%-30s %-4d %-8s %-10s %-10s",
				familyName,
				rev.Revision,
				rev.Status,
				cpuMem,
				formatDuration(time.Since(rev.CreatedAt)),
			)
		} else {
			// Narrow view without family (for two-column mode)
			row = fmt.Sprintf("%-4d %-10s %-10s %-12s",
				rev.Revision,
				rev.Status,
				cpuMem,
				formatDuration(time.Since(rev.CreatedAt)),
			)
		}

		// Apply style
		if i == m.taskDefRevisionCursor {
			if m.showTaskDefJSON {
				row = selectedStyle.Render("▸" + row + " ◀")
			} else {
				row = selectedStyle.Render("▸ " + row)
			}
		} else {
			style := normalStyle
			if rev.Status == "ACTIVE" {
				style = activeStyle
			} else {
				style = inactiveStyle
			}
			row = style.Width(width).Render("  " + row)
		}

		rows = append(rows, row)
	}

	// Add help text
	if !m.showTaskDefJSON {
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
		rows = append(rows, "", helpStyle.Render("[Enter] View JSON"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// renderTaskDefJSON renders the JSON view
func (m Model) renderTaskDefJSON(maxHeight int, width int) string {
	if m.taskDefRevisionCursor >= len(m.taskDefRevisions) {
		return ""
	}

	selectedRev := m.taskDefRevisions[m.taskDefRevisionCursor]

	// Get JSON from cache or show loading message
	jsonContent, cached := m.taskDefJSONCache[selectedRev.Revision]
	if !cached {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Width(width).
			Height(maxHeight).
			Padding(1).
			Render("Loading task definition JSON...")
	}

	// JSON style
	jsonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cccccc")).
		Width(width - 2).
		Height(maxHeight - 2).
		Padding(1)

	// Add scroll indicator if needed
	lines := strings.Split(jsonContent, "\n")
	visibleLines := maxHeight - 4 // Leave room for scroll indicator

	// Adjust scroll position
	maxScroll := len(lines) - visibleLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.taskDefJSONScroll > maxScroll {
		m.taskDefJSONScroll = maxScroll
	}
	if m.taskDefJSONScroll < 0 {
		m.taskDefJSONScroll = 0
	}

	endLine := m.taskDefJSONScroll + visibleLines
	if endLine > len(lines) {
		endLine = len(lines)
	}

	visibleJSON := strings.Join(lines[m.taskDefJSONScroll:endLine], "\n")

	// Add scroll indicator
	if len(lines) > visibleLines {
		scrollPercent := 0
		if maxScroll > 0 {
			scrollPercent = (m.taskDefJSONScroll * 100) / maxScroll
		}
		scrollInfo := fmt.Sprintf(" Lines %d-%d of %d (%d%%) | Scroll: J/K, PgUp/PgDn, g/G, Ctrl+U/D ",
			m.taskDefJSONScroll+1, endLine, len(lines), scrollPercent)

		scrollStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Align(lipgloss.Center).
			Width(width - 2)

		visibleJSON = visibleJSON + "\n" + scrollStyle.Render(scrollInfo)
	}

	return jsonStyle.Render(visibleJSON)
}

// Helper function to truncate strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// renderNavigationPanelWithHeight renders the navigation panel with specified height
func (m Model) renderNavigationPanelWithHeight(panelHeight int) string {
	// Calculate actual available width inside the panel
	panelWidth := m.width - 6 // Account for panel borders and padding

	// Header and breadcrumb use full width
	header := m.renderHeader()
	breadcrumb := m.renderBreadcrumb()
	summary := m.renderSummary()

	// Add separator line - match the actual panel width
	separatorWidth := panelWidth
	if separatorWidth > m.width-8 {
		separatorWidth = m.width - 8 // Ensure it doesn't overflow
	}
	topSeparator := separatorStyle.Render(strings.Repeat("─", separatorWidth))

	// Render shortcuts as single lines at the bottom
	viewShortcuts := m.renderViewShortcutsLine(panelWidth)
	globalShortcuts := m.renderGlobalShortcutsLine(panelWidth)

	// Stack all components vertically
	var components []string
	components = append(components, header)
	if breadcrumb != "" {
		components = append(components, breadcrumb)
	}
	components = append(components, topSeparator)
	components = append(components, summary)
	// Only one separator line before shortcuts
	if viewShortcuts != "" {
		components = append(components, viewShortcuts)
	}
	if globalShortcuts != "" {
		components = append(components, globalShortcuts)
	}

	navContent := lipgloss.JoinVertical(lipgloss.Top, components...)

	// lipgloss.Height() includes padding but NOT borders
	// So if we want total height of panelHeight:
	// - Borders add 2 lines (top + bottom)
	// - Height() should be panelHeight - 2
	contentHeight := panelHeight - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	return navigationPanelStyle.
		Width(m.width - 4). // Account for borders and padding
		Height(contentHeight).
		Render(navContent)
}

// renderResourcePanelWithHeight renders the resource panel with specified height
func (m Model) renderResourcePanelWithHeight(panelHeight int) string {
	var content string

	// lipgloss.Height() includes padding but NOT borders
	// So if we want total height of panelHeight:
	// - Borders add 2 lines (top + bottom)
	// - Height() should be panelHeight - 2
	// - contentHeight for rendering lists should be panelHeight - 2 - 2(padding) = panelHeight - 4
	styleHeight := panelHeight - 2
	contentHeight := panelHeight - 4 // For rendering lists (excludes borders and padding)

	// Render view-specific content
	switch m.currentView {
	case ViewInstances:
		// Only show instances view when no instances exist (for creation prompt)
		if len(m.instances) == 0 {
			content = m.renderNoInstancesView()
		} else {
			// Should not normally reach here as we auto-select instances
			content = m.renderClustersList(contentHeight)
		}
	case ViewClusters:
		content = m.renderClustersList(contentHeight)
	case ViewServices:
		content = m.renderServicesList(contentHeight)
	case ViewTasks:
		content = m.renderTasksList(contentHeight)
	case ViewTaskDescribe:
		content = m.renderTaskDescribe()
	case ViewLogs:
		// Always show log viewer in fullscreen
		if m.logViewer != nil {
			content = m.logViewer.View()
		} else {
			// Log viewer not initialized yet
			content = "Initializing log viewer..."
		}
	case ViewTaskDefinitionFamilies:
		content = m.renderTaskDefFamiliesList(contentHeight)
	case ViewTaskDefinitionRevisions:
		if m.showTaskDefJSON {
			// Show two-column view with JSON
			content = m.renderTaskDefRevisionsTwoColumn(contentHeight)
		} else {
			// Show only the revisions list
			content = m.renderTaskDefRevisionsList(contentHeight, m.width-8)
		}
	default:
		content = "View not implemented"
	}

	return resourcePanelStyle.
		Width(m.width - 4).  // Account for borders and padding
		Height(styleHeight). // Height including padding (borders added automatically)
		Render(content)
}
