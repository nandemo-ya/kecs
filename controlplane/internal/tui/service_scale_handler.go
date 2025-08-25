package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// handleServiceScaleDialogKeys handles key events for the service scale dialog
func (m Model) handleServiceScaleDialogKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.serviceScaleDialog == nil {
		return m, nil
	}

	d := m.serviceScaleDialog

	switch msg.String() {
	case "esc":
		// Cancel dialog
		m.serviceScaleDialog = nil
		return m, nil

	case "tab":
		// Move focus between buttons
		d.MoveFocus()

	case "enter":
		if !d.IsOKFocused() && !d.IsCancelFocused() {
			// In input field, move to OK button
			d.focusedButton = 0
		} else if d.IsOKFocused() {
			if !d.confirmMode {
				// Validate and show confirmation
				if d.Validate() {
					d.SetConfirmMode(true)
				}
			} else {
				// Execute scaling
				return m.executeServiceScale()
			}
		} else if d.IsCancelFocused() {
			// Cancel dialog
			m.serviceScaleDialog = nil
			return m, nil
		}

	case "backspace":
		if !d.IsOKFocused() && !d.IsCancelFocused() && !d.confirmMode {
			d.RemoveLastChar()
		}

	default:
		// Handle text input
		if !d.IsOKFocused() && !d.IsCancelFocused() && !d.confirmMode {
			if len(msg.String()) == 1 {
				d.UpdateInput(d.desiredCount + msg.String())
			}
		}
	}

	return m, nil
}

// executeServiceScale executes the service scaling operation
func (m Model) executeServiceScale() (Model, tea.Cmd) {
	if m.serviceScaleDialog == nil {
		return m, nil
	}

	d := m.serviceScaleDialog
	desiredCount := d.GetDesiredCount()
	serviceName := d.serviceName

	// Close dialog and show progress
	m.serviceScaleDialog = nil
	m.scalingInProgress = true
	m.scalingServiceName = serviceName
	m.scalingTargetCount = desiredCount

	// Find the service to update
	var serviceArn string
	for _, svc := range m.services {
		if svc.Name == serviceName {
			// Construct service ARN
			serviceArn = fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s",
				"us-east-1",    // Use default region
				"000000000000", // Use default account
				m.selectedCluster,
				serviceName)
			break
		}
	}

	if serviceArn == "" {
		m.scalingInProgress = false
		return m, nil
	}

	// Create the scale command
	return m, m.scaleService(serviceArn, desiredCount)
}

// scaleService creates a command to scale the service
func (m Model) scaleService(serviceArn string, desiredCount int) tea.Cmd {
	return func() tea.Msg {
		// Call UpdateService API
		err := m.apiClient.UpdateServiceDesiredCount(
			m.selectedInstance,
			m.selectedCluster,
			serviceArn,
			desiredCount,
		)

		if err != nil {
			return ServiceScaledMsg{
				Success: false,
				Error:   err,
			}
		}

		// Wait a bit to show progress
		time.Sleep(1 * time.Second)

		return ServiceScaledMsg{
			Success:      true,
			ServiceName:  m.scalingServiceName,
			DesiredCount: desiredCount,
		}
	}
}

// ServiceScaledMsg is sent when a service scaling operation completes
type ServiceScaledMsg struct {
	Success      bool
	ServiceName  string
	DesiredCount int
	Error        error
}
