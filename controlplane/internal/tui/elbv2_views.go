package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderELBv2View renders the ELBv2 resources view with navigation panel
func (m Model) renderELBv2View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Calculate exact heights for panels to fill entire screen
	totalHeight := m.height

	// Calculate base heights (30/70 split)
	navPanelHeight := int(float64(totalHeight) * 0.3)
	resourcePanelHeight := totalHeight - navPanelHeight

	// Ensure minimum heights
	if navPanelHeight < 10 {
		navPanelHeight = 10
	}
	if resourcePanelHeight < 10 {
		resourcePanelHeight = 10
	}

	// Adjust to ensure they exactly fill the screen
	if navPanelHeight+resourcePanelHeight < totalHeight {
		// Add any remaining height to the resource panel
		resourcePanelHeight = totalHeight - navPanelHeight
	}

	// Render navigation panel (30% height) - use height-specific version
	navigationPanel := m.renderNavigationPanelWithHeight(navPanelHeight)

	// Render content based on current ELBv2 view
	var content string
	contentHeight := resourcePanelHeight - 4 // Account for borders

	switch m.currentView {
	case ViewLoadBalancers:
		content = m.renderLoadBalancersList(contentHeight)
	case ViewTargetGroups:
		content = m.renderTargetGroupsList(contentHeight)
	case ViewListeners:
		content = m.renderListenersList(contentHeight)
	default:
		content = m.renderLoadBalancersList(contentHeight)
	}

	// Wrap content in a bordered panel
	resourcePanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#585b70")).
		Width(m.width - 2).
		Height(resourcePanelHeight).
		Render(content)

	// Combine all components (no footer now)
	return lipgloss.JoinVertical(
		lipgloss.Top,
		navigationPanel,
		resourcePanel,
	)
}

// renderLoadBalancersList renders the list of load balancers
func (m Model) renderLoadBalancersList(height int) string {
	if len(m.loadBalancers) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#718096")).
			Align(lipgloss.Center, lipgloss.Center).
			Width(m.width-8).
			Height(height).
			Padding(1, 2)
		return emptyStyle.Render("No load balancers found")
	}

	// Calculate column widths based on available width (similar to ECS views)
	availableWidth := m.width - 8
	nameWidth := int(float64(availableWidth) * 0.25)
	typeWidth := int(float64(availableWidth) * 0.12)
	schemeWidth := int(float64(availableWidth) * 0.15)
	stateWidth := int(float64(availableWidth) * 0.12)
	dnsWidth := int(float64(availableWidth) * 0.28)
	ageWidth := int(float64(availableWidth) * 0.08)

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a0aec0")).
		Bold(true)

	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s",
		nameWidth, "NAME",
		typeWidth, "TYPE",
		schemeWidth, "SCHEME",
		stateWidth, "STATE",
		dnsWidth, "DNS NAME",
		ageWidth, "AGE")

	// Rows
	var rows []string
	rows = append(rows, "  "+headerStyle.Render(header))
	rows = append(rows, "  "+strings.Repeat("─", availableWidth-2))

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
		state := stateStyle.Render(fmt.Sprintf("%-*s", stateWidth, lb.State))

		row := fmt.Sprintf("%-*s %-*s %-*s %s %-*s %-*s",
			nameWidth, truncate(lb.Name, nameWidth),
			typeWidth, truncate(lb.Type, typeWidth),
			schemeWidth, lb.Scheme,
			state,
			dnsWidth, truncate(dnsName, dnsWidth),
			ageWidth, age)

		if i == m.lbCursor {
			rows = append(rows, selectedStyle.Width(availableWidth).Render("▸ "+row))
		} else {
			rows = append(rows, "  "+normalStyle.Render(row))
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
			Width(m.width-8).
			Height(height).
			Padding(1, 2)
		return emptyStyle.Render("No target groups found")
	}

	// Calculate column widths based on available width
	availableWidth := m.width - 8
	nameWidth := int(float64(availableWidth) * 0.30)
	portWidth := int(float64(availableWidth) * 0.10)
	protocolWidth := int(float64(availableWidth) * 0.12)
	typeWidth := int(float64(availableWidth) * 0.15)
	healthWidth := int(float64(availableWidth) * 0.20)
	targetsWidth := int(float64(availableWidth) * 0.13)

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a0aec0")).
		Bold(true)

	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s",
		nameWidth, "NAME",
		portWidth, "PORT",
		protocolWidth, "PROTOCOL",
		typeWidth, "TYPE",
		healthWidth, "HEALTH",
		targetsWidth, "TARGETS")

	// Rows
	var rows []string
	rows = append(rows, "  "+headerStyle.Render(header))
	rows = append(rows, "  "+strings.Repeat("─", availableWidth-2))

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
		healthFormatted := healthStyle.Render(fmt.Sprintf("%-*s", healthWidth, health))

		row := fmt.Sprintf("%-*s %-*d %-*s %-*s %s %-*d",
			nameWidth, truncate(tg.Name, nameWidth),
			portWidth, tg.Port,
			protocolWidth, tg.Protocol,
			typeWidth, tg.TargetType,
			healthFormatted,
			targetsWidth, tg.RegisteredTargetsCount)

		if i == m.tgCursor {
			rows = append(rows, selectedStyle.Width(availableWidth).Render("▸ "+row))
		} else {
			rows = append(rows, "  "+normalStyle.Render(row))
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
			Width(m.width-8).
			Height(height).
			Padding(1, 2)

		if m.selectedLB == "" {
			return emptyStyle.Render("Select a load balancer to view its listeners")
		}
		return emptyStyle.Render("No listeners found for selected load balancer")
	}

	// Calculate column widths based on available width
	availableWidth := m.width - 8
	portWidth := int(float64(availableWidth) * 0.10)
	protocolWidth := int(float64(availableWidth) * 0.12)
	actionWidth := int(float64(availableWidth) * 0.18)
	targetGroupWidth := int(float64(availableWidth) * 0.50)
	rulesWidth := int(float64(availableWidth) * 0.10)

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a0aec0")).
		Bold(true)

	header := fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s",
		portWidth, "PORT",
		protocolWidth, "PROTOCOL",
		actionWidth, "ACTION",
		targetGroupWidth, "TARGET GROUP",
		rulesWidth, "RULES")

	// Rows
	var rows []string
	rows = append(rows, "  "+headerStyle.Render(header))
	rows = append(rows, "  "+strings.Repeat("─", availableWidth-2))

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

		row := fmt.Sprintf("%-*d %-*s %-*s %-*s %-*d",
			portWidth, listener.Port,
			protocolWidth, listener.Protocol,
			actionWidth, truncate(action, actionWidth),
			targetGroupWidth, truncate(targetGroup, targetGroupWidth),
			rulesWidth, listener.RuleCount)

		if i == m.listenerCursor {
			rows = append(rows, selectedStyle.Width(availableWidth).Render("▸ "+row))
		} else {
			rows = append(rows, "  "+normalStyle.Render(row))
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
