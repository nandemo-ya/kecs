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
		}
		if m.selectedInstance != "" {
			shortcuts = append(shortcuts, keyStyle.Render("<T>")+sepStyle.Render(" TaskDefs"))
		}
		shortcuts = append(shortcuts,
			keyStyle.Render("<:>")+sepStyle.Render(" Cmd"),
			keyStyle.Render("</>")+sepStyle.Render(" Search"),
			keyStyle.Render("<?>")+sepStyle.Render(" Help"),
		)
	case ViewClusters:
		shortcuts = []string{
			keyStyle.Render("<↵>") + sepStyle.Render(" Select"),
			keyStyle.Render("<n>") + sepStyle.Render(" Create"),
			keyStyle.Render("<ESC>") + sepStyle.Render(" Back"),
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
			keyStyle.Render("<↵>") + sepStyle.Render(" Describe"),
			keyStyle.Render("<l>") + sepStyle.Render(" Logs"),
			keyStyle.Render("<ESC>") + sepStyle.Render(" Back"),
			keyStyle.Render("<:>") + sepStyle.Render(" Cmd"),
		}
	case ViewLogs:
		shortcuts = []string{
			keyStyle.Render("<f>") + sepStyle.Render(" Follow"),
			keyStyle.Render("<s>") + sepStyle.Render(" Save"),
			keyStyle.Render("<Esc>") + sepStyle.Render(" Back"),
		}
	case ViewTaskDefinitionFamilies:
		shortcuts = []string{
			keyStyle.Render("<↵>") + sepStyle.Render(" Select"),
			keyStyle.Render("<N>") + sepStyle.Render(" New"),
			keyStyle.Render("<ESC>") + sepStyle.Render(" Back"),
			keyStyle.Render("</>") + sepStyle.Render(" Search"),
		}
	case ViewTaskDefinitionRevisions:
		shortcuts = []string{
			keyStyle.Render("<↵>") + sepStyle.Render(" JSON"),
			keyStyle.Render("<e>") + sepStyle.Render(" Edit"),
			keyStyle.Render("<ESC>") + sepStyle.Render(" Back"),
		}
		if m.showTaskDefJSON {
			shortcuts = append(shortcuts,
				keyStyle.Render("<^U/^D>")+sepStyle.Render(" Scroll"),
			)
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
	headerText := fmt.Sprintf("KECS %s", version.GetVersion())
	if m.selectedInstance != "" {
		headerText = fmt.Sprintf("KECS %s | Instance: %s", version.GetVersion(), m.selectedInstance)
	}

	// Check if we need to truncate
	if lipgloss.Width(headerText) > maxContentWidth {
		if m.selectedInstance != "" {
			// Shorten instance name
			instanceName := m.selectedInstance
			prefix := fmt.Sprintf("KECS %s | ", version.GetVersion())
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

	// Task Definition navigation
	if m.currentView == ViewTaskDefinitionFamilies || m.currentView == ViewTaskDefinitionRevisions {
		parts = append(parts, ">", "[Task Definitions]")

		if m.currentView == ViewTaskDefinitionRevisions && m.selectedFamily != "" {
			parts = append(parts, ">", m.selectedFamily)
		}
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

	// Default footer shows status info
	left := ""
	right := ""

	// Show instance status on the left
	if m.selectedInstance != "" {
		status := "Unknown"
		for _, inst := range m.instances {
			if inst.Name == m.selectedInstance {
				// Check for various active states
				if inst.Status == "ACTIVE" || inst.Status == "Running" || inst.Status == "running" {
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
	case ViewTaskDefinitionFamilies:
		right = fmt.Sprintf("Families: %d", len(m.filterTaskDefFamilies(m.taskDefFamilies)))
	case ViewTaskDefinitionRevisions:
		right = fmt.Sprintf("Revisions: %d", len(m.taskDefRevisions))
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

	// Add separator line after breadcrumb
	separatorWidth := leftColumnWidth - 2 // Account for padding
	if separatorWidth < 20 {
		separatorWidth = 20
	}
	topSeparator := separatorStyle.Render(strings.Repeat("─", separatorWidth))

	// Combine left column elements
	leftColumn := lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		breadcrumb,
		topSeparator,
		summary,
	)

	// Right column: shortcuts (vertical list)
	rightColumn := m.renderShortcutsColumn(rightColumnWidth, navHeight-4)

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
	case ViewTaskDescribe:
		content = m.renderTaskDescribe()
	case ViewLogs:
		if m.logViewer != nil {
			// If log viewer is active, use its render
			return m.logViewer.View()
		}
		content = m.renderLogsContent(resourceHeight - 4)
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
			var features []string
			var apiPort, adminPort int
			for _, inst := range m.instances {
				if inst.Name == m.selectedInstance {
					apiPort = inst.APIPort
					adminPort = inst.AdminPort
					if inst.LocalStack {
						features = append(features, "LocalStack")
					}
					if inst.Traefik {
						features = append(features, "Traefik")
					}
					if inst.DevMode {
						features = append(features, "DevMode")
					}
					break
				}
			}

			// Build vertical layout with proper alignment
			line1 := fmt.Sprintf("Clusters: %-4d  API:   %d", len(m.clusters), apiPort)
			line2 := fmt.Sprintf("Services: %-4d  Admin: %d", totalServices, adminPort)
			line3 := fmt.Sprintf("Tasks:    %-4d", totalTasks)

			// Add features on the same line if they exist
			if len(features) > 0 {
				line3 += "  " + strings.Join(features, ", ")
			}

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

			// Add instance configuration info
			var instanceInfo string
			if m.selectedInstance != "" {
				var features []string
				for _, inst := range m.instances {
					if inst.Name == m.selectedInstance {
						if inst.LocalStack {
							features = append(features, "LocalStack")
						}
						if inst.Traefik {
							features = append(features, "Traefik")
						}
						if inst.DevMode {
							features = append(features, "DevMode")
						}
						break
					}
				}
				if len(features) > 0 {
					instanceInfo = " | " + strings.Join(features, ", ")
				}
			}

			summary = fmt.Sprintf("Cluster: %s | Services: %d | Desired Tasks: %d | Running Tasks: %d%s",
				m.selectedCluster, len(m.services), totalDesired, totalRunning, instanceInfo)
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
		// Only show Task defs shortcut if an instance is selected
		if m.selectedInstance != "" {
			shortcuts = append(shortcuts,
				keyStyle.Render("T")+" "+descStyle.Render("Task defs"),
			)
		}
	case ViewClusters:
		shortcuts = append(shortcuts,
			keyStyle.Render("↵")+" "+descStyle.Render("Select"),
			keyStyle.Render("n")+" "+descStyle.Render("Create cluster"),
			keyStyle.Render("ESC")+" "+descStyle.Render("Back"),
			keyStyle.Render("i")+" "+descStyle.Render("Instances"),
			keyStyle.Render("s")+" "+descStyle.Render("Services"),
			keyStyle.Render("T")+" "+descStyle.Render("Task defs"),
		)
	case ViewServices:
		shortcuts = append(shortcuts,
			keyStyle.Render("↵")+" "+descStyle.Render("Select"),
			keyStyle.Render("S")+" "+descStyle.Render("Scale"),
			keyStyle.Render("r")+" "+descStyle.Render("Restart"),
			keyStyle.Render("u")+" "+descStyle.Render("Update"),
			keyStyle.Render("x")+" "+descStyle.Render("Stop"),
			keyStyle.Render("l")+" "+descStyle.Render("Logs"),
			keyStyle.Render("T")+" "+descStyle.Render("Task defs"),
		)
	case ViewTasks:
		shortcuts = append(shortcuts,
			keyStyle.Render("↵")+" "+descStyle.Render("Describe"),
			keyStyle.Render("l")+" "+descStyle.Render("Logs"),
			keyStyle.Render("ESC")+" "+descStyle.Render("Back"),
			keyStyle.Render("T")+" "+descStyle.Render("Task defs"),
		)
	case ViewLogs:
		shortcuts = append(shortcuts,
			keyStyle.Render("f")+" "+descStyle.Render("Follow"),
			keyStyle.Render("s")+" "+descStyle.Render("Save"),
			keyStyle.Render("ESC")+" "+descStyle.Render("Back"),
		)
	case ViewTaskDescribe:
		shortcuts = append(shortcuts,
			keyStyle.Render("ESC")+" "+descStyle.Render("Back"),
			keyStyle.Render("l")+" "+descStyle.Render("View Logs"),
			keyStyle.Render("r")+" "+descStyle.Render("Restart"),
			keyStyle.Render("s")+" "+descStyle.Render("Stop"),
		)
	case ViewTaskDefinitionFamilies:
		shortcuts = append(shortcuts,
			keyStyle.Render("↵")+" "+descStyle.Render("Select"),
			keyStyle.Render("N")+" "+descStyle.Render("New"),
			keyStyle.Render("C")+" "+descStyle.Render("Copy latest"),
			keyStyle.Render("ESC")+" "+descStyle.Render("Back"),
		)
	case ViewTaskDefinitionRevisions:
		shortcuts = append(shortcuts,
			keyStyle.Render("↵")+" "+descStyle.Render("Toggle JSON"),
			keyStyle.Render("e")+" "+descStyle.Render("Edit"),
			keyStyle.Render("c")+" "+descStyle.Render("Copy"),
			keyStyle.Render("d")+" "+descStyle.Render("Deregister"),
			keyStyle.Render("ESC")+" "+descStyle.Render("Back"),
		)
		if m.showTaskDefJSON {
			shortcuts = append(shortcuts,
				keyStyle.Render("^U")+" "+descStyle.Render("Scroll up"),
				keyStyle.Render("^D")+" "+descStyle.Render("Scroll down"),
			)
		}
	}

	// Add common shortcuts
	shortcuts = append(shortcuts,
		"", // Empty line for separation
		keyStyle.Render(":")+" "+descStyle.Render("Command"),
		keyStyle.Render("/")+" "+descStyle.Render("Search"),
		keyStyle.Render("?")+" "+descStyle.Render("Help"),
		keyStyle.Render("Ctrl+C")+" "+descStyle.Render("Quit"),
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
  Ctrl-C      Quit application
  Esc         Go back / Cancel
  /           Search in current view
  ↑, k        Move up
  ↓, j        Move down
  Enter       Select/Drill down
  ESC         Go back to parent view

Clipboard Operations:
  y           Copy selected item name/ID to clipboard
  Y           Copy full details to clipboard

Resource Navigation:
  i           Go to instances
  c           Go to clusters
  s           Go to services
  t           Go to tasks
  T           Go to task definitions (from any view with instance selected)

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

Task Definition Operations:
  N           Create new task definition (Families view)
  C           Copy family's latest revision (Families view)
  Enter       Toggle JSON view (Revisions view)
  e           Edit as new revision (Revisions view)
  c           Copy to clipboard (Revisions view)
  d           Deregister revision (Revisions view)
  a           Activate revision (Revisions view)
  D           Diff mode (Revisions view)
  Ctrl+U      Scroll JSON up (JSON view)
  Ctrl+D      Scroll JSON down (JSON view)

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
		Background(lipgloss.Color("#2a2a4a")).
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

	headers := fmt.Sprintf("%-4s %-10s %-10s %-12s",
		"REV", "STATUS", "CPU/MEM", "CREATED")

	// Styles
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2a2a4a")).
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
		row := fmt.Sprintf("%-4d %-10s %-10s %-12s",
			rev.Revision,
			rev.Status,
			cpuMem,
			formatDuration(time.Since(rev.CreatedAt)),
		)

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
