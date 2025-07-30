package tui2

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Layout styles
	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1a1a")).
			Foreground(lipgloss.Color("#ffffff")).
			PaddingLeft(1).
			PaddingRight(1)
			
	breadcrumbStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2a2a2a")).
			Foreground(lipgloss.Color("#cccccc")).
			Padding(0, 1)
			
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
			BorderForeground(lipgloss.Color("#444444")).
			Padding(1, 2)
	
	resourcePanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
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
)

func (m Model) renderHeader() string {
	// Build the left side content
	left := "KECS v1.0.0"
	if m.selectedInstance != "" {
		left = fmt.Sprintf("KECS v1.0.0 | Instance: %s", m.selectedInstance)
	}
	
	// Build the right side content (status)
	right := ""
	if m.selectedInstance != "" {
		// Check if the selected instance is active
		status := "● Active"
		statusStyle := statusActiveStyle
		for _, inst := range m.instances {
			if inst.Name == m.selectedInstance && inst.Status != "ACTIVE" {
				status = "○ Inactive"
				statusStyle = statusInactiveStyle
				break
			}
		}
		right = statusStyle.Render(status)
	}
	
	// Calculate the actual available width for content
	// navigationPanelStyle sets Width to m.width - 4
	// headerStyle has Padding(0, 1) which takes 2 more characters
	maxContentWidth := m.width - 4 - 2
	
	// If no instance selected, just show KECS version
	if m.selectedInstance == "" {
		return headerStyle.Render(left)
	}
	
	// Check if content fits
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	minGap := 2 // Minimum gap between left and right
	
	totalWidth := leftWidth + minGap + rightWidth
	
	if totalWidth > maxContentWidth {
		// Content is too wide, try shorter formats
		instanceName := m.selectedInstance
		if len(instanceName) > 15 {
			instanceName = instanceName[:12] + "..."
			left = fmt.Sprintf("KECS v1.0.0 | Instance: %s", instanceName)
			leftWidth = lipgloss.Width(left)
			totalWidth = leftWidth + minGap + rightWidth
		}
		
		// If still too wide, use shorter format
		if totalWidth > maxContentWidth {
			left = fmt.Sprintf("KECS | %s", instanceName)
			leftWidth = lipgloss.Width(left)
			totalWidth = leftWidth + minGap + rightWidth
		}
		
		// If still too wide, just show left part
		if totalWidth > maxContentWidth {
			headerContent := left
			if lipgloss.Width(headerContent) > maxContentWidth {
				headerContent = headerContent[:maxContentWidth-3] + "..."
			}
			return headerStyle.Render(headerContent)
		}
	}
	
	// Calculate actual gap
	gap := maxContentWidth - leftWidth - rightWidth
	if gap < minGap {
		gap = minGap
	}
	
	// Build the full header
	headerContent := left + strings.Repeat(" ", gap) + right
	
	// Apply header style
	return headerStyle.Render(headerContent)
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
	
	// Ensure breadcrumb doesn't exceed available width
	maxWidth := m.width - 2 // Account for padding
	if lipgloss.Width(breadcrumb) > maxWidth {
		// Truncate from the beginning if too long
		for lipgloss.Width(breadcrumb) > maxWidth && len(breadcrumb) > 0 {
			// Find first space and remove everything before it
			if idx := strings.Index(breadcrumb, " > "); idx >= 0 {
				breadcrumb = "..." + breadcrumb[idx+2:]
			} else {
				// If no separator found, just truncate
				breadcrumb = "..." + breadcrumb[3:]
			}
		}
	}
	
	return breadcrumbStyle.Render(breadcrumb)
}

func (m Model) renderFooter() string {
	shortcuts := []string{}
	
	// Context-specific shortcuts
	switch m.currentView {
	case ViewInstances:
		shortcuts = []string{
			"[i] Instances", "[c] Clusters", "[s] Services", "[t] Tasks",
			"[/] Search", "[?] Help",
		}
	case ViewClusters:
		shortcuts = []string{
			"[↑↓] Navigate", "[Enter] Select", "[Backspace] Back",
			"[i] Instances", "[R] Refresh", "[?] Help",
		}
	case ViewServices:
		shortcuts = []string{
			"[↑↓] Navigate", "[Enter] Select", "[Backspace] Back",
			"[r] Restart", "[S] Scale", "[l] Logs", "[?] Help",
		}
	case ViewTasks:
		shortcuts = []string{
			"[↑↓] Navigate", "[l] Logs", "[D] Describe",
			"[Backspace] Back", "[R] Refresh", "[?] Help",
		}
	case ViewLogs:
		shortcuts = []string{
			"[Esc] Back", "[f] Follow", "[/] Filter", "[s] Save",
			"[↑↓] Scroll",
		}
	}
	
	// Add mode indicators
	if m.searchMode {
		shortcuts = []string{"Search: " + m.searchQuery + "_", "[Enter] Apply", "[Esc] Cancel"}
	} else if m.commandMode {
		shortcuts = []string{"Command: " + m.commandInput + "_", "[Enter] Execute", "[Esc] Cancel"}
	}
	
	footer := strings.Join(shortcuts, "  ")
	return footerStyle.Width(m.width).Render(footer)
}

// renderNavigationPanel renders the top navigation panel (30% height)
func (m Model) renderNavigationPanel() string {
	// Calculate height for navigation panel (30% of available height)
	navHeight := int(float64(m.height-1) * 0.3) // -1 for footer
	if navHeight < 10 {
		navHeight = 10 // Minimum height for navigation content
	}
	
	// Render header
	header := m.renderHeader()
	
	// Render breadcrumb
	breadcrumb := m.renderBreadcrumb()
	
	// Render summary/overview based on current view
	summary := m.renderSummary()
	
	// Calculate exact heights
	headerHeight := 1
	breadcrumbHeight := 1
	summaryHeight := lipgloss.Height(summary)
	contentTotalHeight := headerHeight + breadcrumbHeight + summaryHeight
	
	// If content is too tall, constrain the summary
	if contentTotalHeight > navHeight - 4 {
		// Just show header and breadcrumb
		navContent := lipgloss.JoinVertical(
			lipgloss.Top,
			header,
			breadcrumb,
		)
		return navigationPanelStyle.
			Width(m.width - 4).
			Height(navHeight - 4).
			MaxHeight(navHeight - 4).
			Render(navContent)
	}
	
	// Combine navigation elements
	navContent := lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		breadcrumb,
		summary,
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
	
	// Add separator line - make sure it fits within the panel width
	separatorWidth := m.width - 8
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
	
	// Calculate visible range with scrolling
	visibleRows := maxHeight - 2 // Account for header and potential scroll indicator
	startIdx := 0
	endIdx := len(m.instances)
	
	// Implement scrolling if needed
	if m.instanceCursor >= visibleRows {
		startIdx = m.instanceCursor - visibleRows + 1
	}
	if endIdx > startIdx + visibleRows {
		endIdx = startIdx + visibleRows
	}
	
	for i := startIdx; i < endIdx; i++ {
		instance := m.instances[i]
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
	if len(m.instances) > visibleRows {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d instances]", startIdx+1, endIdx, len(m.instances))
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(scrollInfo))
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
	
	// Calculate visible range with scrolling
	visibleRows := maxHeight - 2
	startIdx := 0
	endIdx := len(m.clusters)
	
	if m.clusterCursor >= visibleRows {
		startIdx = m.clusterCursor - visibleRows + 1
	}
	if endIdx > startIdx + visibleRows {
		endIdx = startIdx + visibleRows
	}
	
	for i := startIdx; i < endIdx; i++ {
		cluster := m.clusters[i]
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
	if len(m.clusters) > visibleRows {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d clusters]", startIdx+1, endIdx, len(m.clusters))
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(scrollInfo))
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
	
	// Calculate visible range with scrolling
	visibleRows := maxHeight - 2
	startIdx := 0
	endIdx := len(m.services)
	
	if m.serviceCursor >= visibleRows {
		startIdx = m.serviceCursor - visibleRows + 1
	}
	if endIdx > startIdx + visibleRows {
		endIdx = startIdx + visibleRows
	}
	
	for i := startIdx; i < endIdx; i++ {
		service := m.services[i]
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
	if len(m.services) > visibleRows {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d services]", startIdx+1, endIdx, len(m.services))
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(scrollInfo))
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
	
	// Calculate visible range with scrolling
	visibleRows := maxHeight - 2
	startIdx := 0
	endIdx := len(m.tasks)
	
	if m.taskCursor >= visibleRows {
		startIdx = m.taskCursor - visibleRows + 1
	}
	if endIdx > startIdx + visibleRows {
		endIdx = startIdx + visibleRows
	}
	
	for i := startIdx; i < endIdx; i++ {
		task := m.tasks[i]
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
	if len(m.tasks) > visibleRows {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d tasks]", startIdx+1, endIdx, len(m.tasks))
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(scrollInfo))
	}
	
	return strings.Join(rows, "\n")
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
	
	// Process logs
	startIdx := m.logCursor
	endIdx := startIdx + availableHeight
	if endIdx > len(m.logs) {
		endIdx = len(m.logs)
	}
	
	for i := startIdx; i < endIdx && i < len(m.logs); i++ {
		log := m.logs[i]
		
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
	if len(m.logs) > availableHeight {
		scrollInfo := fmt.Sprintf("\n[Showing %d-%d of %d log entries]", startIdx+1, endIdx, len(m.logs))
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(scrollInfo))
	}
	
	return strings.Join(lines, "\n")
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

