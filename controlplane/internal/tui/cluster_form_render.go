package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderClusterCreateOverlay renders the cluster creation form as an overlay
func (m Model) renderClusterCreateOverlay() string {
	// Render the form
	form := m.renderClusterForm()

	// Create overlay with the form centered
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		form,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#000000")),
	)
}

// renderClusterForm renders the cluster creation form
func (m Model) renderClusterForm() string {
	if m.clusterForm == nil {
		return ""
	}

	f := m.clusterForm
	var content []string

	// Title with close button
	title := "Create ECS Cluster"
	closeBtn := "[×]"
	if f.focusedField == FieldCloseButton {
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
	totalWidth := 61   // Form width
	spaces := totalWidth - titleWidth - closeBtnWidth
	if spaces < 1 {
		spaces = 1
	}

	titleLine := formTitleStyle.Render(title) + strings.Repeat(" ", spaces) + closeBtn
	content = append(content, titleLine)
	content = append(content, "")

	// Instance info
	instanceInfo := formLabelStyle.Render(fmt.Sprintf("Instance: %s", m.selectedInstance))
	content = append(content, instanceInfo)
	content = append(content, "")

	// Cluster name field
	nameLabel := formLabelStyle.Width(14).Render("Cluster Name:")
	nameInput := m.renderFormInput(f.clusterName.Value(), f.focusedField == FieldClusterName)
	content = append(content, fmt.Sprintf("%s %s", nameLabel, nameInput))
	if f.nameError != "" {
		content = append(content, "  "+formErrorStyle.Render(f.nameError))
	}
	content = append(content, "")

	// Region selector
	regionLabel := formLabelStyle.Width(14).Render("Region:")
	regionSelector := m.renderRegionSelector(f)
	content = append(content, fmt.Sprintf("%s %s", regionLabel, regionSelector))
	content = append(content, "")

	// Buttons (centered)
	createBtnLabel := "Create"
	if f.isCreating {
		createBtnLabel = "Creating..."
	}
	createBtn := m.renderFormButton(createBtnLabel, f.focusedField == FieldCreateButton && !f.isCreating)
	cancelBtn := m.renderFormButton("Cancel", f.focusedField == FieldCancelButton)
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, createBtn, cancelBtn)

	// Center the buttons
	buttonsWidth := lipgloss.Width(buttons)
	formWidth := 61 // Width of form content area (65 - 2*2 padding)
	if buttonsWidth < formWidth {
		padding := (formWidth - buttonsWidth) / 2
		buttons = strings.Repeat(" ", padding) + buttons
	}
	content = append(content, buttons)

	// Show creation status with checkmarks
	if f.isCreating && len(f.creationSteps) > 0 {
		content = append(content, "")
		for _, step := range f.creationSteps {
			var icon string
			var style lipgloss.Style
			switch step.Status {
			case "done":
				icon = "✅"
				style = formSuccessStyle
			case "running":
				icon = "⏳"
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00"))
			case "failed":
				icon = "❌"
				style = formErrorStyle
			default:
				icon = "○"
				style = formLabelStyle
			}
			content = append(content, style.Render(fmt.Sprintf("%s %s", icon, step.Name)))
		}
		if f.creationElapsed != "" {
			content = append(content, formLabelStyle.Render(fmt.Sprintf("⏱  %s elapsed", f.creationElapsed)))
		}
	}

	// Error or success message
	if !f.isCreating && f.errorMsg != "" {
		content = append(content, "")
		content = append(content, formErrorStyle.Render("Error: "+f.errorMsg))
	} else if !f.isCreating && f.successMsg != "" {
		content = append(content, "")
		content = append(content, formSuccessStyle.Render("✓ "+f.successMsg))
	}

	// Help text
	help := formHelpStyle.Render("[Tab] Navigate  [↑/↓] Select Region  [Esc] Cancel")
	content = append(content, "")
	content = append(content, help)

	// Join all content
	formContent := strings.Join(content, "\n")

	// Apply form style
	return formStyle.Render(formContent)
}

// renderRegionSelector renders the region selector dropdown
func (m Model) renderRegionSelector(f *ClusterForm) string {
	width := 35
	focused := f.focusedField == FieldRegion

	// Build the selector display
	selectedRegion := awsRegions[f.regionIndex]

	if focused {
		// Show dropdown-like view when focused
		startIdx := f.regionIndex - 1
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := f.regionIndex + 2
		if endIdx > len(awsRegions) {
			endIdx = len(awsRegions)
		}

		// Build lines for the dropdown
		var lines []string
		for i := startIdx; i < endIdx && i < len(awsRegions); i++ {
			var line string
			if i == f.regionIndex {
				// Highlighted selection
				line = lipgloss.NewStyle().
					Background(lipgloss.Color("#3a3a5a")).
					Foreground(lipgloss.Color("#ffffff")).
					Width(width - 4). // Account for border and padding
					Render("▸ " + awsRegions[i])
			} else {
				// Other options
				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#808080")).
					Width(width - 4). // Account for border and padding
					Render("  " + awsRegions[i])
			}
			lines = append(lines, line)
		}

		// Join lines and apply border
		content := strings.Join(lines, "\n")
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#3a3a5a")).
			Width(width).
			Padding(0, 1). // Add horizontal padding
			Render(content)
	} else {
		// Show only selected region when not focused
		return formInputStyle.Render(selectedRegion)
	}
}
