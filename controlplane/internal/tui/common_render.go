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

	// Style the title without margin
	styledTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff00")).
		Bold(true).
		Render(title)

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
	titleWidth := lipgloss.Width(styledTitle)
	closeBtnWidth := 3 // [×]
	spaces := width - titleWidth - closeBtnWidth
	if spaces < 1 {
		spaces = 1
	}

	// Create the complete line without margin
	completeLine := styledTitle + strings.Repeat(" ", spaces) + closeBtn

	// Apply margin to the whole line
	return lipgloss.NewStyle().
		MarginBottom(1).
		Render(completeLine)
}
