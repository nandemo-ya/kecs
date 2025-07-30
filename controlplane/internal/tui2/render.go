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
			Padding(0, 1)
			
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
)

func (m Model) renderHeader() string {
	status := "● Active"
	statusStyle := statusActiveStyle
	
	environment := "development"
	if m.selectedInstance != "" {
		environment = m.selectedInstance
		// Check if the selected instance is active
		for _, inst := range m.instances {
			if inst.Name == m.selectedInstance && inst.Status != "ACTIVE" {
				status = "○ Inactive"
				statusStyle = statusInactiveStyle
				break
			}
		}
	}
	
	left := fmt.Sprintf("KECS v1.0.0 | Environment: %s", environment)
	right := statusStyle.Render(status)
	
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4 // 4 for padding
	if gap < 0 {
		gap = 0
	}
	
	header := left + strings.Repeat(" ", gap) + right
	return headerStyle.Width(m.width).Render(header)
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
			parts = append(parts, ">", m.selectedTask)
		}
		parts = append(parts, ">", "[Logs]")
	}
	
	breadcrumb := strings.Join(parts, " ")
	return breadcrumbStyle.Width(m.width).Render(breadcrumb)
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

// View-specific render methods

func (m Model) renderInstancesView() string {
	// Colors for instances
	activeColor := lipgloss.Color("#00ff00")
	stoppedColor := lipgloss.Color("#ff0000")
	selectedColor := lipgloss.Color("#00ffff")
	headerColor := lipgloss.Color("#808080")
	
	// Styles for instances
	instHeaderStyle := lipgloss.NewStyle().
			Foreground(headerColor).
			Bold(true)
			
	selectedStyle := lipgloss.NewStyle().
			Foreground(selectedColor).
			Bold(true)
			
	activeStyle := lipgloss.NewStyle().
			Foreground(activeColor)
			
	stoppedStyle := lipgloss.NewStyle().
			Foreground(stoppedColor)
	
	// Calculate column widths
	nameWidth := 20
	statusWidth := 10
	clustersWidth := 10
	servicesWidth := 10
	tasksWidth := 10
	portWidth := 10
	ageWidth := 10
	
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
	
	for i, instance := range m.instances {
		// Format values
		name := instance.Name
		status := instance.Status
		clusters := fmt.Sprintf("%d", instance.Clusters)
		services := fmt.Sprintf("%d", instance.Services)
		tasks := fmt.Sprintf("%d", instance.Tasks)
		port := fmt.Sprintf("%d", instance.APIPort)
		age := formatDuration(instance.Age)
		
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
			row = selectedStyle.Render(row)
		} else {
			switch instance.Status {
			case "ACTIVE":
				row = activeStyle.Render(row)
			case "STOPPED":
				row = stoppedStyle.Render(row)
			}
		}
		
		rows = append(rows, row)
	}
	
	// Calculate available height for content
	contentHeight := m.height - 4 // Header, breadcrumb, footer
	if len(rows) > contentHeight {
		rows = rows[:contentHeight]
	}
	
	content := strings.Join(rows, "\n")
	return contentStyle.Render(content)
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

func (m Model) renderClustersView() string {
	// Colors for clusters
	activeColor := lipgloss.Color("#00ff00")
	selectedColor := lipgloss.Color("#00ffff")
	headerColor := lipgloss.Color("#808080")
	warningColor := lipgloss.Color("#ffff00")
	
	// Styles for clusters
	clusterHeaderStyle := lipgloss.NewStyle().
			Foreground(headerColor).
			Bold(true)
			
	selectedStyle := lipgloss.NewStyle().
			Foreground(selectedColor).
			Bold(true)
			
	activeStyle := lipgloss.NewStyle().
			Foreground(activeColor)
			
	warningStyle := lipgloss.NewStyle().
			Foreground(warningColor)
	
	// Calculate column widths
	nameWidth := 20
	statusWidth := 10
	servicesWidth := 10
	tasksWidth := 10
	cpuWidth := 12
	memoryWidth := 12
	namespaceWidth := 25
	ageWidth := 10
	
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
	
	if m.selectedInstance == "" {
		return contentStyle.Render("No instance selected")
	}
	
	for i, cluster := range m.clusters {
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
			row = selectedStyle.Render(row)
		} else {
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
	
	// Calculate available height for content
	contentHeight := m.height - 4 // Header, breadcrumb, footer
	if len(rows) > contentHeight {
		rows = rows[:contentHeight]
	}
	
	// Add instance info
	instanceInfo := fmt.Sprintf("\nInstance: %s\n\n", m.selectedInstance)
	
	content := instanceInfo + strings.Join(rows, "\n")
	return contentStyle.Render(content)
}

func (m Model) renderServicesView() string {
	// Colors for services
	activeColor := lipgloss.Color("#00ff00")
	inactiveColor := lipgloss.Color("#0000ff")
	updatingColor := lipgloss.Color("#ffff00")
	provisioningColor := lipgloss.Color("#ff8800")
	selectedColor := lipgloss.Color("#00ffff")
	headerColor := lipgloss.Color("#808080")
	
	// Styles for services
	serviceHeaderStyle := lipgloss.NewStyle().
			Foreground(headerColor).
			Bold(true)
			
	selectedStyle := lipgloss.NewStyle().
			Foreground(selectedColor).
			Bold(true)
			
	activeStyle := lipgloss.NewStyle().
			Foreground(activeColor)
			
	inactiveStyle := lipgloss.NewStyle().
			Foreground(inactiveColor)
			
	updatingStyle := lipgloss.NewStyle().
			Foreground(updatingColor)
			
	provisioningStyle := lipgloss.NewStyle().
			Foreground(provisioningColor)
	
	// Calculate column widths
	nameWidth := 25
	desiredWidth := 10
	runningWidth := 10
	pendingWidth := 10
	statusWidth := 15
	taskDefWidth := 20
	ageWidth := 10
	
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
	
	if m.selectedInstance == "" || m.selectedCluster == "" {
		return contentStyle.Render("No cluster selected")
	}
	
	for i, service := range m.services {
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
			row = selectedStyle.Render(row)
		} else {
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
	
	// Calculate available height for content
	contentHeight := m.height - 4 // Header, breadcrumb, footer
	if len(rows) > contentHeight {
		rows = rows[:contentHeight]
	}
	
	// Add context info
	contextInfo := fmt.Sprintf("\nInstance: %s > Cluster: %s\n\n", m.selectedInstance, m.selectedCluster)
	
	content := contextInfo + strings.Join(rows, "\n")
	return contentStyle.Render(content)
}

func (m Model) renderTasksView() string {
	// Colors for tasks
	runningColor := lipgloss.Color("#00ff00")
	pendingColor := lipgloss.Color("#ffff00")
	stoppingColor := lipgloss.Color("#ff8800")
	failedColor := lipgloss.Color("#ff0000")
	healthyColor := lipgloss.Color("#00ff00")
	unhealthyColor := lipgloss.Color("#ff0000")
	unknownColor := lipgloss.Color("#808080")
	selectedColor := lipgloss.Color("#00ffff")
	headerColor := lipgloss.Color("#808080")
	
	// Styles for tasks
	taskHeaderStyle := lipgloss.NewStyle().
			Foreground(headerColor).
			Bold(true)
			
	selectedStyle := lipgloss.NewStyle().
			Foreground(selectedColor).
			Bold(true)
			
	runningStyle := lipgloss.NewStyle().
			Foreground(runningColor)
			
	pendingStyle := lipgloss.NewStyle().
			Foreground(pendingColor)
			
	stoppingStyle := lipgloss.NewStyle().
			Foreground(stoppingColor)
			
	failedStyle := lipgloss.NewStyle().
			Foreground(failedColor)
	
	// Calculate column widths
	idWidth := 20
	serviceWidth := 20
	statusWidth := 10
	healthWidth := 10
	cpuWidth := 8
	memoryWidth := 10
	ipWidth := 15
	ageWidth := 10
	
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
	
	if m.selectedInstance == "" || m.selectedCluster == "" || m.selectedService == "" {
		return contentStyle.Render("No service selected")
	}
	
	for i, task := range m.tasks {
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
			row = selectedStyle.Render(row)
		} else {
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
	
	// Calculate available height for content
	contentHeight := m.height - 4 // Header, breadcrumb, footer
	if len(rows) > contentHeight {
		rows = rows[:contentHeight]
	}
	
	// Add context info
	contextInfo := fmt.Sprintf("\nInstance: %s > Cluster: %s > Service: %s\n\n", 
		m.selectedInstance, m.selectedCluster, m.selectedService)
	
	content := contextInfo + strings.Join(rows, "\n")
	return contentStyle.Render(content)
}

func (m Model) renderLogsView() string {
	// Colors for logs
	infoColor := lipgloss.Color("#ffffff")
	warnColor := lipgloss.Color("#ffff00")
	errorColor := lipgloss.Color("#ff0000")
	debugColor := lipgloss.Color("#808080")
	timestampColor := lipgloss.Color("#00ffff")
	selectedColor := lipgloss.Color("#00ffff")
	
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
			
	selectedStyle := lipgloss.NewStyle().
			Foreground(selectedColor).
			Bold(true)
	
	// Container style for logs
	logContainerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#808080")).
		Padding(1, 2)
	
	// Calculate available space
	availableWidth := m.width - 6 // Account for padding and borders
	availableHeight := m.height - 8 // Account for header, breadcrumb, footer, borders
	
	// Build log lines
	lines := []string{}
	
	if m.selectedTask == "" {
		lines = append(lines, "No task selected for log viewing")
	} else {
		// Add task info header
		taskInfo := fmt.Sprintf("Task: %s", m.selectedTask)
		lines = append(lines, taskInfo, "")
		
		// Process logs
		startIdx := m.logCursor
		endIdx := startIdx + availableHeight - 3 // Account for header lines
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
			maxMessageWidth := availableWidth - len(timestamp) - len("[DEBUG]") - 2
			if len(message) > maxMessageWidth {
				message = message[:maxMessageWidth-3] + "..."
			}
			
			// Combine parts
			logLine := fmt.Sprintf("%s %s %s", timestampStr, levelStr, message)
			
			// Apply selection if this is the current line
			if i == m.logCursor {
				logLine = selectedStyle.Render("> " + logLine)
			} else {
				logLine = "  " + logLine
			}
			
			lines = append(lines, logLine)
		}
		
		// Add scroll indicator if needed
		if len(m.logs) > availableHeight-3 {
			scrollInfo := fmt.Sprintf("\n[%d-%d of %d logs]", startIdx+1, endIdx, len(m.logs))
			lines = append(lines, scrollInfo)
		}
	}
	
	// Join lines and apply container style
	content := strings.Join(lines, "\n")
	styledContent := logContainerStyle.
		Width(availableWidth).
		Height(availableHeight).
		Render(content)
	
	// Add context info
	var contextInfo string
	if m.selectedTask != "" {
		contextInfo = fmt.Sprintf("\nLogs: %s > %s > %s > %s\n", 
			m.selectedInstance, m.selectedCluster, m.selectedService, m.selectedTask)
	} else {
		contextInfo = "\nLog Viewer\n"
	}
	
	return contextInfo + styledContent
}

func (m Model) renderHelpView() string {
	help := `
KECS TUI Help

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

Press any key to close help...
`
	return contentStyle.Render(help)
}