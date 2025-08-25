package tui

import (
	"log"
	"os"
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
		// Move focus between input field and buttons
		d.MoveFocus()

	case "enter":
		if d.IsInputFocused() {
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
		} else if d.IsCancelFocused() || d.IsCloseFocused() {
			// Cancel dialog
			m.serviceScaleDialog = nil
			return m, nil
		}

	case "backspace":
		if d.IsInputFocused() && !d.confirmMode {
			d.RemoveLastChar()
		}

	default:
		// Handle text input
		if d.IsInputFocused() && !d.confirmMode {
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

	// Log to file for debugging (optional)
	if logFile := os.Getenv("KECS_TUI_DEBUG_LOG"); logFile != "" {
		if f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			logger := log.New(f, "", log.LstdFlags)
			logger.Printf("Scaling service: %s to %d\n", serviceName, desiredCount)
			f.Close()
		}
	}

	// Close dialog and show progress
	m.serviceScaleDialog = nil
	m.scalingInProgress = true
	m.scalingServiceName = serviceName
	m.scalingTargetCount = desiredCount

	// Verify service exists
	serviceFound := false
	for _, svc := range m.services {
		if svc.Name == serviceName {
			serviceFound = true
			break
		}
	}

	if !serviceFound {
		m.scalingInProgress = false
		return m, nil
	}

	// Create the scale command (pass service name, not ARN)
	return m, m.scaleService(serviceName, serviceName, desiredCount)
}

// scaleService creates a command to scale the service
func (m Model) scaleService(serviceNameOrArn string, serviceName string, desiredCount int) tea.Cmd {
	return func() tea.Msg {
		// Log to file for debugging (optional)
		if logFile := os.Getenv("KECS_TUI_DEBUG_LOG"); logFile != "" {
			if f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				logger := log.New(f, "", log.LstdFlags)
				logger.Printf("API Call: UpdateServiceDesiredCount - instance: %s, cluster: %s, service: %s, desired: %d\n",
					m.selectedInstance, m.selectedCluster, serviceName, desiredCount)
				f.Close()
			}
		}

		// Call UpdateService API
		err := m.apiClient.UpdateServiceDesiredCount(
			m.selectedInstance,
			m.selectedCluster,
			serviceNameOrArn, // Pass service name, not full ARN
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
			ServiceName:  serviceName,
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
