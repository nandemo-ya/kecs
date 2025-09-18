package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderELBv2View renders the ELBv2 resources view with tabs
func (m Model) renderELBv2View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Calculate heights
	footerHeight := 1
	availableHeight := m.height - footerHeight
	tabBarHeight := 3
	contentHeight := availableHeight - tabBarHeight

	// Render tab bar
	tabBar := m.renderELBv2TabBar()

	// Render content based on selected sub-view
	var content string
	switch m.elbv2SubView {
	case 0: // Load Balancers
		content = m.renderLoadBalancersList(contentHeight - 4)
	case 1: // Target Groups
		content = m.renderTargetGroupsList(contentHeight - 4)
	case 2: // Listeners
		content = m.renderListenersList(contentHeight - 4)
	default:
		content = m.renderLoadBalancersList(contentHeight - 4)
	}

	// Wrap content in a bordered panel
	contentPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#585b70")).
		Width(m.width - 2).
		Height(contentHeight).
		Render(content)

	// Render footer
	footer := m.renderFooter()

	// Combine all components
	return lipgloss.JoinVertical(
		lipgloss.Top,
		tabBar,
		contentPanel,
		footer,
	)
}

// renderELBv2TabBar renders the tab bar for ELBv2 views
func (m Model) renderELBv2TabBar() string {
	tabs := []string{"Load Balancers", "Target Groups", "Listeners"}
	var tabsRendered []string

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#667eea")).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 2).
		Bold(true)

	unselectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2d3748")).
		Foreground(lipgloss.Color("#a0aec0")).
		Padding(0, 2)

	for i, tab := range tabs {
		if i == m.elbv2SubView {
			tabsRendered = append(tabsRendered, selectedStyle.Render(tab))
		} else {
			tabsRendered = append(tabsRendered, unselectedStyle.Render(tab))
		}
	}

	tabLine := lipgloss.JoinHorizontal(lipgloss.Left, tabsRendered...)

	// Add navigation hints
	hints := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#718096")).
		Render("  Tab: Switch view • esc: Back")

	// Create header with instance/cluster info
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a0aec0")).
		Bold(true)

	header := headerStyle.Render(fmt.Sprintf("ELBv2 Resources - Instance: %s", m.selectedInstance))

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		tabLine+hints,
		"", // Empty line for spacing
	)
}

// renderLoadBalancersList renders the list of load balancers
func (m Model) renderLoadBalancersList(height int) string {
	if len(m.loadBalancers) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#718096")).
			Align(lipgloss.Center, lipgloss.Center).
			Width(m.width - 4).
			Height(height)
		return emptyStyle.Render("No load balancers found")
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a0aec0")).
		Bold(true)

	header := fmt.Sprintf("%-30s %-15s %-15s %-10s %-50s %s",
		"NAME", "TYPE", "SCHEME", "STATE", "DNS NAME", "AGE")

	// Rows
	var rows []string
	rows = append(rows, headerStyle.Render(header))
	rows = append(rows, strings.Repeat("─", m.width-4))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2d3748")).
		Foreground(lipgloss.Color("#ffffff"))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cbd5e0"))

	// State colors
	stateColors := map[string]lipgloss.Color{
		"active":       lipgloss.Color("#48bb78"), // green
		"provisioning": lipgloss.Color("#f6e05e"), // yellow
		"failed":       lipgloss.Color("#f56565"), // red
	}

	for i, lb := range m.loadBalancers {
		// Format age
		age := formatAge(time.Since(lb.CreatedAt))

		// Truncate DNS name if too long
		dnsName := lb.DNSName
		if len(dnsName) > 48 {
			dnsName = dnsName[:45] + "..."
		}

		// Color state
		stateStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cbd5e0"))
		if color, ok := stateColors[lb.State]; ok {
			stateStyle = stateStyle.Foreground(color)
		}
		state := stateStyle.Render(fmt.Sprintf("%-10s", lb.State))

		row := fmt.Sprintf("%-30s %-15s %-15s %s %-50s %s",
			truncate(lb.Name, 30),
			lb.Type,
			lb.Scheme,
			state,
			dnsName,
			age)

		if i == m.lbCursor {
			rows = append(rows, selectedStyle.Render(row))
		} else {
			rows = append(rows, normalStyle.Render(row))
		}
	}

	// Join rows and limit to available height
	content := strings.Join(rows, "\n")
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// renderTargetGroupsList renders the list of target groups
func (m Model) renderTargetGroupsList(height int) string {
	if len(m.targetGroups) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#718096")).
			Align(lipgloss.Center, lipgloss.Center).
			Width(m.width - 4).
			Height(height)
		return emptyStyle.Render("No target groups found")
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a0aec0")).
		Bold(true)

	header := fmt.Sprintf("%-30s %-8s %-8s %-10s %-20s %s",
		"NAME", "PORT", "PROTOCOL", "TYPE", "HEALTH", "TARGETS")

	// Rows
	var rows []string
	rows = append(rows, headerStyle.Render(header))
	rows = append(rows, strings.Repeat("─", m.width-4))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2d3748")).
		Foreground(lipgloss.Color("#ffffff"))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cbd5e0"))

	for i, tg := range m.targetGroups {
		// Format health status
		healthStyle := lipgloss.NewStyle()
		health := fmt.Sprintf("%d/%d healthy", tg.HealthyTargetCount, tg.RegisteredTargetsCount)
		if tg.UnhealthyTargetCount > 0 {
			healthStyle = healthStyle.Foreground(lipgloss.Color("#f6e05e")) // yellow for warning
			if tg.HealthyTargetCount == 0 {
				healthStyle = healthStyle.Foreground(lipgloss.Color("#f56565")) // red for critical
			}
		} else {
			healthStyle = healthStyle.Foreground(lipgloss.Color("#48bb78")) // green for healthy
		}
		healthFormatted := healthStyle.Render(fmt.Sprintf("%-20s", health))

		row := fmt.Sprintf("%-30s %-8d %-8s %-10s %s %d",
			truncate(tg.Name, 30),
			tg.Port,
			tg.Protocol,
			tg.TargetType,
			healthFormatted,
			tg.RegisteredTargetsCount)

		if i == m.tgCursor {
			rows = append(rows, selectedStyle.Render(row))
		} else {
			rows = append(rows, normalStyle.Render(row))
		}
	}

	// Join rows and limit to available height
	content := strings.Join(rows, "\n")
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// renderListenersList renders the list of listeners
func (m Model) renderListenersList(height int) string {
	if len(m.listeners) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#718096")).
			Align(lipgloss.Center, lipgloss.Center).
			Width(m.width - 4).
			Height(height)

		if m.selectedLB == "" {
			return emptyStyle.Render("Select a load balancer to view its listeners")
		}
		return emptyStyle.Render("No listeners found for selected load balancer")
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a0aec0")).
		Bold(true)

	header := fmt.Sprintf("%-8s %-10s %-15s %-40s %s",
		"PORT", "PROTOCOL", "ACTION", "TARGET GROUP", "RULES")

	// Rows
	var rows []string
	rows = append(rows, headerStyle.Render(header))
	rows = append(rows, strings.Repeat("─", m.width-4))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2d3748")).
		Foreground(lipgloss.Color("#ffffff"))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cbd5e0"))

	for i, listener := range m.listeners {
		// Get default action and target
		action := "unknown"
		targetGroup := "-"
		if len(listener.DefaultActions) > 0 {
			action = listener.DefaultActions[0].Type
			if listener.DefaultActions[0].TargetGroupArn != "" {
				// Extract target group name from ARN
				parts := strings.Split(listener.DefaultActions[0].TargetGroupArn, "/")
				if len(parts) >= 2 {
					targetGroup = parts[1]
				}
			}
		}

		row := fmt.Sprintf("%-8d %-10s %-15s %-40s %d",
			listener.Port,
			listener.Protocol,
			action,
			truncate(targetGroup, 40),
			listener.RuleCount)

		if i == m.listenerCursor {
			rows = append(rows, selectedStyle.Render(row))
		} else {
			rows = append(rows, normalStyle.Render(row))
		}
	}

	// Join rows and limit to available height
	content := strings.Join(rows, "\n")
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}

	return strings.Join(lines, "\n")
}

// Helper function to format age
func formatAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	return fmt.Sprintf("%dd", days)
}

// Helper function to truncate strings
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}
