package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
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
			Background(lipgloss.Color("#2a2a4a")).
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
		shortcuts = []string{
			keyStyle.Render("<N>") + sepStyle.Render(" New"),
			keyStyle.Render("<:>") + sepStyle.Render(" Cmd"),
			keyStyle.Render("</>") + sepStyle.Render(" Search"),
			keyStyle.Render("<?>") + sepStyle.Render(" Help"),
		}
	case ViewClusters:
		shortcuts = []string{
			keyStyle.Render("<↵>") + sepStyle.Render(" Select"),
			keyStyle.Render("<←>") + sepStyle.Render(" Back"),
			keyStyle.Render("<:>") + sepStyle.Render(" Cmd"),
			keyStyle.Render("<?>") + sepStyle.Render(" Help"),
		}
	case ViewServices:
		shortcuts = []string{
			keyStyle.Render("<S>") + sepStyle.Render(" Scale"),
			keyStyle.Render("<r>") + sepStyle.Render(" Restart"),
			keyStyle.Render("<l>") + sepStyle.Render(" Logs"),
			keyStyle.Render("<:>") + sepStyle.Render(" Cmd"),
		}
	case ViewTasks:
		shortcuts = []string{
			keyStyle.Render("<l>") + sepStyle.Render(" Logs"),
			keyStyle.Render("<D>") + sepStyle.Render(" Describe"),
			keyStyle.Render("<←>") + sepStyle.Render(" Back"),
			keyStyle.Render("<:>") + sepStyle.Render(" Cmd"),
		}
	case ViewLogs:
		shortcuts = []string{
			keyStyle.Render("<f>") + sepStyle.Render(" Follow"),
			keyStyle.Render("<s>") + sepStyle.Render(" Save"),
			keyStyle.Render("<Esc>") + sepStyle.Render(" Back"),
		}
	default:
		shortcuts = []string{
			keyStyle.Render("<:>") + sepStyle.Render(" Cmd"),
			keyStyle.Render("</>") + sepStyle.Render(" Search"),
			keyStyle.Render("<?>") + sepStyle.Render(" Help"),
		}
	}
	
	// Join shortcuts with spaces
	return strings.Join(shortcuts, "  ")
}

func (m Model) renderHeader() string {
	// Calculate the actual available width for left column
	totalWidth := m.width - 4 // Account for panel borders
	leftColumnWidth := int(float64(totalWidth) * 0.7)
	maxContentWidth := leftColumnWidth - 2 // Account for header padding
	
	// Build the header content
	headerText := "KECS v0.0.1-alpha"
	if m.selectedInstance != "" {
		headerText = fmt.Sprintf("KECS v0.0.1-alpha | Instance: %s", m.selectedInstance)
	}
	
	// Check if we need to truncate
	if lipgloss.Width(headerText) > maxContentWidth {
		if m.selectedInstance != "" {
			// Shorten instance name
			instanceName := m.selectedInstance
			prefix := "KECS v0.0.1-alpha | "
			maxInstanceWidth := maxContentWidth - len(prefix) - 3 // -3 for "..."
			if maxInstanceWidth > 0 && len(instanceName) > maxInstanceWidth {
				instanceName = instanceName[:maxInstanceWidth] + "..."
			}
			headerText = prefix + instanceName
		}
	}
	
	// Apply header style with proper width
	return headerStyle.Width(maxContentWidth).Render(headerText)
}

func (m Model) renderBreadcrumb() string {
	parts := []string{}
	
	// Build breadcrumb based on current navigation
	if m.currentView == ViewInstances || m.selectedInstance != "" {
		if m.currentView == ViewInstances {
			parts = append(parts, "[Instances]")
		} else {
			parts = append(parts, "Instances")
		}
	}
	
	if m.selectedInstance != "" {
		parts = append(parts, ">", m.selectedInstance)
		
		if m.currentView == ViewClusters {
			parts = append(parts, ">", "[Clusters]")
		} else if m.selectedCluster != "" {
			parts = append(parts, ">", "Clusters")
		}
	}
	
	if m.selectedCluster != "" {
		parts = append(parts, ">", m.selectedCluster)
		
		if m.currentView == ViewServices {
			parts = append(parts, ">", "[Services]")
		} else if m.selectedService != "" {
			parts = append(parts, ">", "Services")
		}
	}
	
	if m.selectedService != "" {
		parts = append(parts, ">", m.selectedService)
		
		if m.currentView == ViewTasks {
			parts = append(parts, ">", "[Tasks]")
		}
	}
	
	if m.currentView == ViewLogs {
		if m.selectedTask != "" {
			// Truncate long task IDs to prevent overflow
			taskID := m.selectedTask
			maxTaskIDLength := 20
			if len(taskID) > maxTaskIDLength {
				taskID = taskID[:maxTaskIDLength-3] + "..."
			}
			parts = append(parts, ">", taskID)
		}
		parts = append(parts, ">", "[Logs]")
	}
	
	breadcrumb := strings.Join(parts, " ")
	
	// Calculate the same width as header for consistency
	totalWidth := m.width - 4 // Account for panel borders
	leftColumnWidth := int(float64(totalWidth) * 0.7)
	maxContentWidth := leftColumnWidth - 2 // Account for breadcrumb padding
	
	// Ensure breadcrumb doesn't exceed available width
	if lipgloss.Width(breadcrumb) > maxContentWidth {
		// Truncate from the beginning if too long
		for lipgloss.Width(breadcrumb) > maxContentWidth && len(breadcrumb) > 0 {
			// Find first space and remove everything before it
			if idx := strings.Index(breadcrumb, " > "); idx >= 0 {
				breadcrumb = "..." + breadcrumb[idx+2:]
			} else {
				// If no separator found, just truncate
				breadcrumb = "..." + breadcrumb[3:]
			}
		}
	}
	
	return breadcrumbStyle.Width(maxContentWidth).Render(breadcrumb)
}

func (m Model) renderFooter() string {
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
	
	// Default footer shows status info
	left := ""
	right := ""
	
	// Show instance status on the left
	if m.selectedInstance != "" {
		status := "Unknown"
		for _, inst := range m.instances {
			if inst.Name == m.selectedInstance {
				if inst.Status == "ACTIVE" {
					status = statusActiveStyle.Render("● Active")
				} else {
					status = statusInactiveStyle.Render("○ Inactive")
				}
				break
			}
		}
		left = fmt.Sprintf("Instance: %s", status)
	}
	
	// Show selection count on the right based on current view
	switch m.currentView {
	case ViewInstances:
		right = fmt.Sprintf("Instances: %d", len(m.instances))
	case ViewClusters:
		right = fmt.Sprintf("Clusters: %d", len(m.filterClusters(m.clusters)))
	case ViewServices:
		right = fmt.Sprintf("Services: %d", len(m.filterServices(m.services)))
	case ViewTasks:
		right = fmt.Sprintf("Tasks: %d", len(m.filterTasks(m.tasks)))
	case ViewLogs:
		right = fmt.Sprintf("Logs: %d", len(m.filterLogs(m.logs)))
	}
	
	// Calculate spacing
	if left == "" && right == "" {
		return footerStyle.Width(m.width).Render("")
	}
	
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := m.width - leftWidth - rightWidth - 4 // -4 for padding
	if gap < 2 {
		gap = 2
	}
	
	content := left + strings.Repeat(" ", gap) + right
	return footerStyle.Width(m.width).Render(content)
}

// renderNavigationPanel renders the top navigation panel (30% height)
func (m Model) renderNavigationPanel() string {
	// Calculate height for navigation panel (30% of available height)
	navHeight := int(float64(m.height-1) * 0.3) // -1 for footer
	if navHeight < 10 {
		navHeight = 10 // Minimum height for navigation content
	}
	
	// Calculate column widths (7:3 ratio)
	totalWidth := m.width - 4 // Account for panel borders
	leftColumnWidth := int(float64(totalWidth) * 0.7)
	rightColumnWidth := totalWidth - leftColumnWidth - 1 // -1 for gap between columns
	
	// Left column: header, breadcrumb, and summary
	header := m.renderHeader()
	breadcrumb := m.renderBreadcrumb()
	summary := m.renderSummary()
	
	// Combine left column elements
	leftColumn := lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		breadcrumb,
		summary,
	)
	
	// Right column: shortcuts (vertical list)
	rightColumn := m.renderShortcutsColumn(rightColumnWidth, navHeight - 4)
	
	// Join columns horizontally
	navContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(leftColumnWidth).Render(leftColumn),
		lipgloss.NewStyle().Width(rightColumnWidth).Render(rightColumn),
	)
	
	// Apply navigation panel style with fixed height
	return navigationPanelStyle.
		Width(m.width - 4). // Account for borders and padding
		Height(navHeight - 4).
		MaxHeight(navHeight - 4).
		Render(navContent)
}

// renderResourcePanel renders the bottom resource panel (70% height)
func (m Model) renderResourcePanel() string {
	// Calculate height for resource panel (70% of available height)
	resourceHeight := int(float64(m.height-1) * 0.7) // -1 for footer
	if resourceHeight < 10 {
		resourceHeight = 10 // Minimum height
	}
	
	var content string
	
	// Render view-specific content
	switch m.currentView {
	case ViewInstances:
		content = m.renderInstancesList(resourceHeight - 4) // Account for borders/padding
	case ViewClusters:
		content = m.renderClustersList(resourceHeight - 4)
	case ViewServices:
		content = m.renderServicesList(resourceHeight - 4)
	case ViewTasks:
		content = m.renderTasksList(resourceHeight - 4)
	case ViewLogs:
		content = m.renderLogsContent(resourceHeight - 4)
	case ViewHelp:
		content = m.renderHelpContent(resourceHeight - 4)
	}
	
	// Apply resource panel style with fixed height
	return resourcePanelStyle.
		Width(m.width - 4). // Account for borders and padding
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
			if inst.Status == "ACTIVE" {
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
			summary = fmt.Sprintf("Instance: %s | Clusters: %d | Total Services: %d | Total Tasks: %d",
				m.selectedInstance, len(m.clusters), totalServices, totalTasks)
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
		if m.selectedService != "" {
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
			summary = fmt.Sprintf("Service: %s | Tasks: %d | Running: %d | Healthy: %d",
				m.selectedService, len(m.tasks), running, healthy)
		}
		
	case ViewLogs:
		if m.selectedTask != "" {
			// Keep log view summary short and on one line
			summary = fmt.Sprintf("Log entries: %d", len(m.logs))
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
	
	return lipgloss.JoinVertical(
		lipgloss.Top,
		summaryStyle.Render(summary),
		separatorStyle.Render(separator),
	)
}

// View-specific render methods

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
	if endIdx > startIdx + visibleRows {
		endIdx = startIdx + visibleRows
	}
	
	for i := startIdx; i < endIdx; i++ {
		instance := filteredInstances[i]
		// Format values
		name := instance.Name
		status := instance.Status
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
			case "ACTIVE":
				row = activeStyle.Render(row)
			case "STOPPED":
				row = stoppedStyle.Render(row)
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
	warningColor := lipgloss.Color("#ffff00")
	
	// Styles for clusters
	clusterHeaderStyle := lipgloss.NewStyle().
			Foreground(headerColor).
			Bold(true)
			
	activeStyle := lipgloss.NewStyle().
			Foreground(activeColor)
			
	warningStyle := lipgloss.NewStyle().
			Foreground(warningColor)
	
	// Calculate column widths based on available width
	availableWidth := m.width - 8
	nameWidth := int(float64(availableWidth) * 0.20)
	statusWidth := int(float64(availableWidth) * 0.10)
	servicesWidth := int(float64(availableWidth) * 0.10)
	tasksWidth := int(float64(availableWidth) * 0.10)
	cpuWidth := int(float64(availableWidth) * 0.12)
	memoryWidth := int(float64(availableWidth) * 0.12)
	namespaceWidth := int(float64(availableWidth) * 0.20)
	ageWidth := int(float64(availableWidth) * 0.10)
	
	// Header
	header := fmt.Sprintf(
		"%-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		nameWidth, "NAME",
		statusWidth, "STATUS",
		servicesWidth, "SERVICES",
		tasksWidth, "TASKS",
		cpuWidth, "CPU",
		memoryWidth, "MEMORY",
		namespaceWidth, "NAMESPACE",
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
	if endIdx > startIdx + visibleRows {
		endIdx = startIdx + visibleRows
	}
	
	for i := startIdx; i < endIdx; i++ {
		cluster := filteredClusters[i]
		// Format values
		name := cluster.Name
		status := cluster.Status
		services := fmt.Sprintf("%d", cluster.Services)
		tasks := fmt.Sprintf("%d", cluster.Tasks)
		cpu := fmt.Sprintf("%.1f/%.1f", cluster.CPUUsed, cluster.CPUTotal)
		memory := fmt.Sprintf("%s/%s", cluster.MemoryUsed, cluster.MemoryTotal)
		namespace := cluster.Namespace
		age := formatDuration(cluster.Age)
		
		// Truncate long values
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}
		if len(namespace) > namespaceWidth {
			namespace = namespace[:namespaceWidth-3] + "..."
		}
		
		// Create row
		row := fmt.Sprintf(
			"%-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			nameWidth, name,
			statusWidth, status,
			servicesWidth, services,
			tasksWidth, tasks,
			cpuWidth, cpu,
			memoryWidth, memory,
			namespaceWidth, namespace,
			ageWidth, age,
		)
		
		// Apply styles
		if i == m.clusterCursor {
			// Apply full-row highlight with consistent width
			row = selectedRowStyle.Width(availableWidth).Render("▸ " + row)
		} else {
			row = "  " + row
			// Check CPU usage for warning
			cpuUsagePercent := (cluster.CPUUsed / cluster.CPUTotal) * 100
			if cpuUsagePercent > 80 {
				row = warningStyle.Render(row)
			} else if cluster.Status == "ACTIVE" {
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
	nameWidth := int(float64(availableWidth) * 0.25)
	desiredWidth := int(float64(availableWidth) * 0.10)
	runningWidth := int(float64(availableWidth) * 0.10)
	pendingWidth := int(float64(availableWidth) * 0.10)
	statusWidth := int(float64(availableWidth) * 0.15)
	taskDefWidth := int(float64(availableWidth) * 0.20)
	ageWidth := int(float64(availableWidth) * 0.10)
	
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
	if endIdx > startIdx + visibleRows {
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
	if m.selectedInstance == "" || m.selectedCluster == "" || m.selectedService == "" {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			Render("No service selected. Press 's' to go to services.")
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
	idWidth := int(float64(availableWidth) * 0.20)
	serviceWidth := int(float64(availableWidth) * 0.20)
	statusWidth := int(float64(availableWidth) * 0.10)
	healthWidth := int(float64(availableWidth) * 0.10)
	cpuWidth := int(float64(availableWidth) * 0.08)
	memoryWidth := int(float64(availableWidth) * 0.10)
	ipWidth := int(float64(availableWidth) * 0.15)
	ageWidth := int(float64(availableWidth) * 0.10)
	
	// Header
	header := fmt.Sprintf(
		"%-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		idWidth, "ID",
		serviceWidth, "SERVICE",
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
	if endIdx > startIdx + visibleRows {
		endIdx = startIdx + visibleRows
	}
	
	for i := startIdx; i < endIdx; i++ {
		task := filteredTasks[i]
		// Format values
		id := task.ID
		service := task.Service
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
		if len(service) > serviceWidth {
			service = service[:serviceWidth-3] + "..."
		}
		
		// Create row
		row := fmt.Sprintf(
			"%-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
			idWidth, id,
			serviceWidth, service,
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

// renderShortcutsColumn renders the shortcuts column for the navigation panel
func (m Model) renderShortcutsColumn(width, height int) string {
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
	
	shortcuts := []string{}
	
	// Add view-specific shortcuts
	switch m.currentView {
	case ViewInstances:
		shortcuts = append(shortcuts,
			keyStyle.Render("N")+" "+descStyle.Render("New instance"),
			keyStyle.Render("↵")+" "+descStyle.Render("Select"),
			keyStyle.Render("S")+" "+descStyle.Render("Start/Stop"),
			keyStyle.Render("D")+" "+descStyle.Render("Delete"),
		)
	case ViewClusters:
		shortcuts = append(shortcuts,
			keyStyle.Render("↵")+" "+descStyle.Render("Select"),
			keyStyle.Render("←")+" "+descStyle.Render("Back"),
			keyStyle.Render("i")+" "+descStyle.Render("Instances"),
			keyStyle.Render("s")+" "+descStyle.Render("Services"),
		)
	case ViewServices:
		shortcuts = append(shortcuts,
			keyStyle.Render("↵")+" "+descStyle.Render("Select"),
			keyStyle.Render("S")+" "+descStyle.Render("Scale"),
			keyStyle.Render("r")+" "+descStyle.Render("Restart"),
			keyStyle.Render("u")+" "+descStyle.Render("Update"),
			keyStyle.Render("x")+" "+descStyle.Render("Stop"),
			keyStyle.Render("l")+" "+descStyle.Render("Logs"),
		)
	case ViewTasks:
		shortcuts = append(shortcuts,
			keyStyle.Render("↵")+" "+descStyle.Render("View logs"),
			keyStyle.Render("l")+" "+descStyle.Render("Logs"),
			keyStyle.Render("D")+" "+descStyle.Render("Describe"),
			keyStyle.Render("←")+" "+descStyle.Render("Back"),
		)
	case ViewLogs:
		shortcuts = append(shortcuts,
			keyStyle.Render("f")+" "+descStyle.Render("Follow"),
			keyStyle.Render("s")+" "+descStyle.Render("Save"),
			keyStyle.Render("←")+" "+descStyle.Render("Back"),
		)
	case ViewTaskDescribe:
		shortcuts = append(shortcuts,
			keyStyle.Render("ESC")+" "+descStyle.Render("Back"),
			keyStyle.Render("l")+" "+descStyle.Render("View Logs"),
			keyStyle.Render("r")+" "+descStyle.Render("Restart"),
			keyStyle.Render("s")+" "+descStyle.Render("Stop"),
		)
	}
	
	// Add common shortcuts
	shortcuts = append(shortcuts,
		"", // Empty line for separation
		keyStyle.Render(":")+" "+descStyle.Render("Command"),
		keyStyle.Render("/")+" "+descStyle.Render("Search"),
		keyStyle.Render("?")+" "+descStyle.Render("Help"),
		keyStyle.Render("q")+" "+descStyle.Render("Quit"),
	)
	
	// Join shortcuts vertically
	shortcutsContent := strings.Join(shortcuts, "\n")
	
	return columnStyle.Height(height).Render(shortcutsContent)
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
	helpText := `KECS TUI Help

Global Navigation:
  ?           Show/hide this help
  q, Ctrl-C   Quit
  /           Search in current view
  Esc         Cancel/Back
  ↑, k        Move up
  ↓, j        Move down
  Enter       Select/Drill down
  Backspace   Go back to parent view

Resource Navigation:
  i           Go to instances
  c           Go to clusters
  s           Go to services
  t           Go to tasks
  d           Go to task definitions

Instance Operations:
  N           Create new instance (Instances view)
  S           Stop/Start instance (Instance selected)
  D           Delete instance (Instance selected)
  Ctrl+I      Quick switch instance (Any view)

Service Operations:
  r           Restart service (Service selected)
  S           Scale service (Service selected)
  u           Update service (Service selected)
  x           Stop service (Service selected)

Common Operations:
  l           View logs (Task/Service selected)
  D           Describe resource (Any resource)
  R           Refresh view (Any view)
  M           Multi-instance overview (Any view)

Command Mode:
  :           Enter command mode
  Enter       Execute command
  Esc         Cancel command

Press any key to close help...`

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(helpText)
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

