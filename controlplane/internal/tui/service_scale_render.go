package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderServiceScaleDialog renders the service scale dialog
func (m Model) renderServiceScaleDialog() string {
	if m.serviceScaleDialog == nil {
		return ""
	}

	d := m.serviceScaleDialog
	var content []string

	// Title with close button
	titleLine := renderDialogTitle("Scale Service", true, d.IsCloseFocused(), 66)
	content = append(content, titleLine)
	content = append(content, "")

	// Service info
	content = append(content, d.GetMessage())
	content = append(content, "")

	if !d.confirmMode {
		// Input mode
		content = append(content, "Desired count:")

		// Input field
		inputStyle := formInputStyle
		if d.IsInputFocused() {
			inputStyle = formInputFocusedStyle
		}
		inputValue := d.desiredCount
		if d.IsInputFocused() {
			inputValue += "_"
		}
		content = append(content, inputStyle.Width(30).Render(inputValue))

		// Error message
		if d.errorMsg != "" {
			content = append(content, formErrorStyle.Render(d.errorMsg))
		}
		content = append(content, "")

		// Buttons
		okBtn := m.renderFormButton("Scale", d.IsOKFocused())
		cancelBtn := m.renderFormButton("Cancel", d.IsCancelFocused())
		buttons := lipgloss.JoinHorizontal(lipgloss.Top, okBtn, cancelBtn)

		// Center the buttons
		buttonsWidth := lipgloss.Width(buttons)
		formWidth := 66 // Width of form content area (70 - 2*2 padding)
		if buttonsWidth < formWidth {
			padding := (formWidth - buttonsWidth) / 2
			buttons = strings.Repeat(" ", padding) + buttons
		}
		content = append(content, buttons)
	} else {
		// Confirmation mode
		content = append(content, "")

		// Buttons
		confirmBtn := m.renderFormButton("Confirm", d.IsOKFocused())
		cancelBtn := m.renderFormButton("Cancel", d.IsCancelFocused())
		buttons := lipgloss.JoinHorizontal(lipgloss.Top, confirmBtn, cancelBtn)

		// Center the buttons
		buttonsWidth := lipgloss.Width(buttons)
		formWidth := 66
		if buttonsWidth < formWidth {
			padding := (formWidth - buttonsWidth) / 2
			buttons = strings.Repeat(" ", padding) + buttons
		}
		content = append(content, buttons)
	}

	// Help text
	content = append(content, "")
	help := formHelpStyle.Render("[Tab] Navigate  [Enter] Select  [Esc] Cancel")
	content = append(content, help)

	// Join all content
	dialogContent := strings.Join(content, "\n")

	// Apply dialog style
	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#1a1a1a")).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(1, 2).
		Width(70).
		Render(dialogContent)

	// Center the dialog
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#000000")),
	)
}

// renderScalingProgress renders the scaling progress overlay
func (m Model) renderScalingProgress() string {
	var content []string

	content = append(content, "")
	content = append(content, formSuccessStyle.Render("⏳ Scaling Service..."))
	content = append(content, "")
	content = append(content, formLabelStyle.Render(fmt.Sprintf("Service: %s", m.scalingServiceName)))

	if m.scalingTargetCount >= 0 {
		content = append(content, formLabelStyle.Render(fmt.Sprintf("Target: %d task(s)", m.scalingTargetCount)))
	}

	content = append(content, "")
	content = append(content, formHelpStyle.Render("Please wait..."))

	// Join all content
	progressContent := strings.Join(content, "\n")

	// Apply dialog style
	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#666666")).
		Background(lipgloss.Color("#1a1a1a")).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(1, 2).
		Width(50).
		Render(progressContent)

	// Center the dialog
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#000000")),
	)
}
