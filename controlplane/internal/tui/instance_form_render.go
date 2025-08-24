package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Instance form styles
	formOverlayStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#000000")).
				Foreground(lipgloss.Color("#ffffff"))

	formStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Background(lipgloss.Color("#1a1a1a")).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(1, 2).
			Width(65)

	formTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ff00")).
			Bold(true).
			MarginBottom(1)

	formLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080"))

	formInputStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2a2a2a")).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(0, 1).
			Width(35)

	formInputFocusedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#3a3a5a")).
				Foreground(lipgloss.Color("#ffffff")).
				Padding(0, 1).
				Width(35)

	formCheckboxStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00ff00"))

	formCheckboxUncheckedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#666666"))

	formButtonStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2a2a2a")).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(0, 2).
			MarginRight(2)

	formButtonFocusedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#00ff00")).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Padding(0, 2).
				MarginRight(2)

	formErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff0000")).
			Italic(true)

	formSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00ff00")).
				Bold(true)

	formHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)
)

// renderInstanceCreateOverlay renders the instance creation form as an overlay
func (m Model) renderInstanceCreateOverlay() string {
	// Render the form
	form := m.renderInstanceForm()

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

// renderInstanceForm renders the instance creation form
func (m Model) renderInstanceForm() string {
	if m.instanceForm == nil {
		return ""
	}

	f := m.instanceForm
	var content []string

	// Title
	title := formTitleStyle.Render("Create New Instance")
	content = append(content, title)
	content = append(content, "")

	// Instance name field
	nameLabel := formLabelStyle.Render("Instance Name:")
	nameInput := m.renderFormInput(f.instanceName, f.focusedField == FieldInstanceName)
	randomBtn := m.renderRandomButton(f.focusedField == FieldInstanceName)
	nameLine := fmt.Sprintf("%s %s %s", nameLabel, nameInput, randomBtn)
	content = append(content, nameLine)
	if f.nameError != "" {
		content = append(content, "  "+formErrorStyle.Render(f.nameError))
	}
	content = append(content, "")

	// API Port field
	apiLabel := formLabelStyle.Width(14).Render("API Port:")
	apiInput := m.renderFormInput(f.apiPort, f.focusedField == FieldAPIPort)
	content = append(content, fmt.Sprintf("%s %s", apiLabel, apiInput))
	if f.apiPortError != "" {
		content = append(content, "  "+formErrorStyle.Render(f.apiPortError))
	}

	// Admin Port field
	adminLabel := formLabelStyle.Width(14).Render("Admin Port:")
	adminInput := m.renderFormInput(f.adminPort, f.focusedField == FieldAdminPort)
	content = append(content, fmt.Sprintf("%s %s", adminLabel, adminInput))
	if f.adminPortError != "" {
		content = append(content, "  "+formErrorStyle.Render(f.adminPortError))
	}
	content = append(content, "")

	// Checkboxes
	content = append(content, m.renderCheckbox("Enable LocalStack", f.localStack, f.focusedField == FieldLocalStack))
	content = append(content, m.renderCheckbox("Enable Traefik Gateway", f.traefik, f.focusedField == FieldTraefik))
	content = append(content, m.renderCheckbox("Developer Mode", f.devMode, f.focusedField == FieldDevMode))
	content = append(content, "")

	// Buttons (centered)
	createBtnLabel := "Create"
	if f.isCreating {
		createBtnLabel = "Creating..."
	}
	createBtn := m.renderFormButton(createBtnLabel, f.focusedField == FieldSubmit && !f.isCreating)
	cancelBtn := m.renderFormButton("Cancel", f.focusedField == FieldCancel)
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

	// Show timeout prompt
	if f.showTimeoutPrompt {
		content = append(content, "")
		content = append(content, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffff00")).
			Render("⚠️  Creation is taking longer than expected"))
		content = append(content, formLabelStyle.Render("LocalStack may need to download images (~500MB)"))
		content = append(content, formLabelStyle.Render("Press 'c' to continue waiting or 'ESC' to abort"))
	}

	// Error or success message
	if !f.isCreating && f.errorMsg != "" {
		content = append(content, "")
		content = append(content, formErrorStyle.Render("Error: "+f.errorMsg))
	} else if !f.isCreating && f.successMsg != "" {
		content = append(content, "")
		content = append(content, formSuccessStyle.Render("✓ "+f.successMsg))
	}

	// Help text (shortened to fit in one line)
	help := formHelpStyle.Render("[Tab] Navigate  [Space] Toggle  [Esc] Cancel")
	content = append(content, "")
	content = append(content, help)

	// Join all content
	formContent := strings.Join(content, "\n")

	// Apply form style
	return formStyle.Render(formContent)
}

// renderFormInput renders a text input field
func (m Model) renderFormInput(value string, focused bool) string {
	display := value
	if focused {
		display += "_"
		return formInputFocusedStyle.Render(display)
	}
	return formInputStyle.Render(display)
}

// renderRandomButton renders the random name generator button
func (m Model) renderRandomButton(focused bool) string {
	if focused {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffff00")).
			Bold(true).
			Render("🎲")
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Render("🎲")
}

// renderCheckbox renders a checkbox field
func (m Model) renderCheckbox(label string, checked bool, focused bool) string {
	checkbox := "☐"
	style := formCheckboxUncheckedStyle

	if checked {
		checkbox = "☑"
		style = formCheckboxStyle
	}

	if focused {
		style = style.Background(lipgloss.Color("#2a2a4a"))
	}

	return fmt.Sprintf("%s %s", style.Render(checkbox), label)
}

// renderFormButton renders a form button
func (m Model) renderFormButton(label string, focused bool) string {
	if focused {
		return formButtonFocusedStyle.Render(label)
	}
	return formButtonStyle.Render(label)
}
