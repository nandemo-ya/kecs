package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderDialogTitle renders a dialog title with an optional close button
// Returns the formatted title line with right-aligned close button
func renderDialogTitle(title string, showCloseButton bool, closeButtonFocused bool, width int) string {
	if !showCloseButton {
		return formTitleStyle.Render(title)
	}

	// Close button styling
	closeBtn := "[×]"
	if closeButtonFocused {
		closeBtn = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6b6b")).
			Bold(true).
			Render("[×]")
	} else {
		closeBtn = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Render("[×]")
	}

	// Calculate spacing for right-aligned close button
	titleWidth := lipgloss.Width(title)
	closeBtnWidth := 3 // [×]
	spaces := width - titleWidth - closeBtnWidth
	if spaces < 1 {
		spaces = 1
	}

	return formTitleStyle.Render(title) + strings.Repeat(" ", spaces) + closeBtn
}
