package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderServiceUpdateDialog renders the service update dialog
func (m Model) renderServiceUpdateDialog() string {
	if m.serviceUpdateDialog == nil {
		return ""
	}

	d := m.serviceUpdateDialog
	var content []string

	// Title with close button
	titleLine := renderDialogTitle("Update Service", true, d.IsCloseFocused(), 76)
	content = append(content, titleLine)
	content = append(content, "")

	// Service info
	content = append(content, d.GetMessage())
	content = append(content, "")

	if !d.confirmMode {
		// Selection mode - show task definition list
		content = append(content, formLabelStyle.Render("Select Task Definition:"))
		content = append(content, "")

		// List of task definitions
		listStyle := formInputStyle
		if d.IsListFocused() {
			listStyle = formInputFocusedStyle
		}

		// Show up to 5 items with scroll indication
		startIdx := 0
		endIdx := len(d.availableTaskDefs)
		maxVisible := 5

		if len(d.availableTaskDefs) > maxVisible {
			// Center the selection
			startIdx = d.selectedTaskDefIdx - 2
			if startIdx < 0 {
				startIdx = 0
			}
			endIdx = startIdx + maxVisible
			if endIdx > len(d.availableTaskDefs) {
				endIdx = len(d.availableTaskDefs)
				startIdx = endIdx - maxVisible
			}
		}

		// Build list content
		var listItems []string
		for i := startIdx; i < endIdx; i++ {
			taskDef := d.availableTaskDefs[i]
			prefix := "  "
			if i == d.selectedTaskDefIdx {
				prefix = "▸ "
			}

			// Highlight current task def
			item := prefix + taskDef
			if taskDef == d.currentTaskDef {
				item += " (current)"
			}

			// Apply highlight style to selected item
			if i == d.selectedTaskDefIdx {
				itemStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("#005577")).
					Foreground(lipgloss.Color("#ffffff")).
					Width(74) // Width of list minus borders
				listItems = append(listItems, itemStyle.Render(item))
			} else {
				listItems = append(listItems, item)
			}
		}

		// Add scroll indicators
		if startIdx > 0 {
			listItems[0] = "↑ " + listItems[0][2:]
		}
		if endIdx < len(d.availableTaskDefs) {
			lastIdx := len(listItems) - 1
			listItems[lastIdx] = "↓ " + listItems[lastIdx][2:]
		}

		listContent := strings.Join(listItems, "\n")
		content = append(content, listStyle.Width(76).Render(listContent))

		// Error message
		if d.errorMsg != "" {
			content = append(content, "")
			content = append(content, formErrorStyle.Render(d.errorMsg))
		}
		content = append(content, "")

		// Buttons
		okBtn := m.renderFormButton("Update", d.IsOKFocused())
		cancelBtn := m.renderFormButton("Cancel", d.IsCancelFocused())
		buttons := lipgloss.JoinHorizontal(lipgloss.Top, okBtn, cancelBtn)

		// Center the buttons
		buttonsWidth := lipgloss.Width(buttons)
		formWidth := 76 // Width of form content area
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
		formWidth := 76
		if buttonsWidth < formWidth {
			padding := (formWidth - buttonsWidth) / 2
			buttons = strings.Repeat(" ", padding) + buttons
		}
		content = append(content, buttons)
	}

	// Help text
	content = append(content, "")
	help := formHelpStyle.Render("[↑/↓] Select  [Tab] Navigate  [Enter] Confirm  [Esc] Cancel")
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
		Width(80).
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

// renderUpdatingProgress renders the updating progress overlay
func (m Model) renderUpdatingProgress() string {
	var content []string

	content = append(content, "")
	content = append(content, formSuccessStyle.Render("⏳ Updating Service..."))
	content = append(content, "")
	content = append(content, formLabelStyle.Render(fmt.Sprintf("Service: %s", m.updatingServiceName)))

	if m.updatingTaskDef != "" {
		content = append(content, formLabelStyle.Render(fmt.Sprintf("Task Definition: %s", m.updatingTaskDef)))
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
		Width(60).
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
