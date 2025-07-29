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
	// TODO: Implement clusters view
	return contentStyle.Render("Clusters view - TODO")
}

func (m Model) renderServicesView() string {
	// TODO: Implement services view
	return contentStyle.Render("Services view - TODO")
}

func (m Model) renderTasksView() string {
	// TODO: Implement tasks view
	return contentStyle.Render("Tasks view - TODO")
}

func (m Model) renderLogsView() string {
	// TODO: Implement logs view
	return contentStyle.Render("Logs view - TODO")
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