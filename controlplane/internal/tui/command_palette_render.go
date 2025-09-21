package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Command palette styles
	overlayStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#000000")).
			Foreground(lipgloss.Color("#ffffff"))

	paletteStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Background(lipgloss.Color("#1a1a1a")).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(1, 2).
			MarginTop(5)

	paletteHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00ff00")).
				Bold(true)

	paletteInputStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#2a2a2a")).
				Foreground(lipgloss.Color("#ffffff")).
				Padding(0, 1)

	commandListStyle = lipgloss.NewStyle().
				MarginTop(1).
				MarginBottom(1)

	commandItemStyle = lipgloss.NewStyle().
				PaddingLeft(2)

	selectedCommandStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#005577")).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true).
				PaddingLeft(1).
				PaddingRight(1)

	commandNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00ff00")).
				Bold(true)

	commandDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#808080"))

	commandShortcutStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffff00"))

	commandCategoryStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00ffff")).
				Bold(true).
				Underline(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff0000")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff00")).
			Bold(true)
)

func (m Model) renderCommandPaletteOverlay() string {
	// Create the command palette overlay
	palette := m.renderCommandPalette()

	// Create a semi-transparent overlay background
	_ = overlayStyle.Width(m.width).Height(m.height).Render("")

	// Place the palette in the center-top of the overlay
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Top,
		palette,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#000000")),
	)
}

func (m Model) renderNormalView() string {
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

	// Render navigation panel (30% height)
	navigationPanel := m.renderNavigationPanelWithHeight(navPanelHeight)

	// Render resource panel (70% height)
	resourcePanel := m.renderResourcePanelWithHeight(resourcePanelHeight)

	// Combine panels - they should exactly fill the terminal height
	return lipgloss.JoinVertical(
		lipgloss.Top,
		navigationPanel,
		resourcePanel,
	)
}

func (m Model) renderCommandPalette() string {
	cp := m.commandPalette
	maxWidth := 70
	// Dynamic max height based on terminal size
	maxHeight := m.height - 10
	if maxHeight > 25 {
		maxHeight = 25
	}
	if maxHeight < 10 {
		maxHeight = 10
	}

	// Build header
	header := paletteHeaderStyle.Render("Command Palette")

	// Build input line
	prompt := "> "
	inputContent := prompt + cp.query
	if m.commandMode {
		inputContent += "_"
	}
	input := paletteInputStyle.Width(maxWidth - 4).Render(inputContent)

	// Build command list
	var commandList []string

	// Group commands by category if no query
	if cp.query == "" {
		categoryMap := make(map[CommandCategory][]Command)
		for _, cmd := range cp.filteredCmds {
			categoryMap[cmd.Category] = append(categoryMap[cmd.Category], cmd)
		}

		// Display in order
		categories := []CommandCategory{
			CommandCategoryGeneral,
			CommandCategoryNavigation,
			CommandCategoryCreate,
			CommandCategoryManage,
			CommandCategoryScale,
			CommandCategoryDebug,
			CommandCategoryExport,
		}

		for _, cat := range categories {
			if cmds, ok := categoryMap[cat]; ok && len(cmds) > 0 {
				commandList = append(commandList, commandCategoryStyle.Render(string(cat)))
				for _, cmd := range cmds {
					// Check if this is the selected command
					selected := false
					for j, filteredCmd := range cp.filteredCmds {
						if filteredCmd.Name == cmd.Name && j == cp.selectedIndex {
							selected = true
							break
						}
					}
					commandList = append(commandList, m.renderCommandItem(cmd, selected))
				}
				commandList = append(commandList, "") // Add spacing
			}
		}
	} else {
		// Show filtered results
		for i, cmd := range cp.filteredCmds {
			commandList = append(commandList, m.renderCommandItem(cmd, i == cp.selectedIndex))
		}
	}

	// Calculate available space for commands
	// Account for: header (1), input (1), help text (1), padding (4), borders (2)
	reservedLines := 9
	availableLines := m.height - reservedLines
	if availableLines < 5 {
		availableLines = 5
	}
	if availableLines > 15 {
		availableLines = 15
	}

	// Limit the number of commands shown
	if len(commandList) > availableLines {
		commandList = commandList[:availableLines]
		commandList = append(commandList, commandDescStyle.Render("  ... and more"))
	}

	// Build the full list
	listContent := commandListStyle.Render(strings.Join(commandList, "\n"))

	// Show result or error
	var statusLine string
	if m.err != nil {
		statusLine = errorStyle.Render("Error: " + m.err.Error())
		m.err = nil // Clear error after displaying
	} else if cp.showResult && cp.lastResult != "" {
		statusLine = successStyle.Render("✓ " + cp.lastResult)
	}

	// Build footer with help text
	helpText := commandDescStyle.Render("↑/↓ Navigate • Enter Execute • Tab Autocomplete • Esc Cancel")

	// Combine all parts
	content := []string{
		header,
		input,
		listContent,
	}

	if statusLine != "" {
		content = append(content, statusLine)
	}

	content = append(content, helpText)

	// Join and apply palette style
	paletteContent := strings.Join(content, "\n")

	// Apply width constraint
	palette := paletteStyle.Width(maxWidth).MaxHeight(maxHeight).Render(paletteContent)

	return palette
}

func (m Model) renderCommandItem(cmd Command, selected bool) string {
	// Build command display
	nameAndShortcut := cmd.Name
	if cmd.Shortcut != "" {
		nameAndShortcut += " [" + cmd.Shortcut + "]"
	}

	desc := " - " + cmd.Description

	// Truncate if too long
	maxLineWidth := 65
	fullLine := nameAndShortcut + desc
	if len(fullLine) > maxLineWidth {
		// Truncate description to fit
		maxDescLen := maxLineWidth - len(nameAndShortcut) - 3 // 3 for "..."
		if maxDescLen > 0 {
			desc = desc[:maxDescLen] + "..."
		}
	}

	// Format with proper spacing
	if selected {
		// Apply selection style to the entire line
		line := "▶ " + nameAndShortcut + desc
		return selectedCommandStyle.Width(65).Render(line)
	} else {
		// Apply name style only to name, description style to description
		namePart := commandNameStyle.Render(nameAndShortcut)
		descPart := commandDescStyle.Render(desc)
		return commandItemStyle.Render(namePart + descPart)
	}
}

// Add a method to display command result in footer
func (m Model) renderCommandResult() string {
	if m.commandPalette.showResult && m.commandPalette.lastResult != "" {
		return successStyle.Render(" " + m.commandPalette.lastResult + " ")
	}
	if m.err != nil {
		return errorStyle.Render(" Error: " + m.err.Error() + " ")
	}
	return ""
}
