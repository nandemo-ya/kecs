package tui

import (
	"fmt"
	"strconv"
	"strings"
)

// ServiceScaleDialog represents a dialog for scaling a service
type ServiceScaleDialog struct {
	serviceName   string
	currentCount  int
	desiredCount  string
	errorMsg      string
	confirmMode   bool
	focusedButton int // -1: Input field, 0: OK, 1: Cancel, 2: Close
}

// NewServiceScaleDialog creates a new service scale dialog
func NewServiceScaleDialog(serviceName string, currentCount int) *ServiceScaleDialog {
	return &ServiceScaleDialog{
		serviceName:   serviceName,
		currentCount:  currentCount,
		desiredCount:  strconv.Itoa(currentCount),
		confirmMode:   false,
		focusedButton: -1, // Start with input field focused
	}
}

// UpdateInput updates the desired count input
func (d *ServiceScaleDialog) UpdateInput(input string) {
	// Only allow digits
	if input == "" || isValidNumber(input) {
		d.desiredCount = input
		d.errorMsg = ""
	}
}

// RemoveLastChar removes the last character from input
func (d *ServiceScaleDialog) RemoveLastChar() {
	if len(d.desiredCount) > 0 {
		d.desiredCount = d.desiredCount[:len(d.desiredCount)-1]
		d.errorMsg = ""
	}
}

// MoveFocus moves focus between input field and buttons
func (d *ServiceScaleDialog) MoveFocus() {
	if d.confirmMode {
		// In confirm mode, cycle through OK, Cancel, Close
		if d.focusedButton < 0 {
			d.focusedButton = 0
		} else {
			d.focusedButton = (d.focusedButton + 1) % 3
		}
	} else {
		// In input mode, cycle through input field, OK, Cancel, Close
		d.focusedButton = (d.focusedButton+2)%4 - 1 // -1, 0, 1, 2
	}
}

// Validate validates the input
func (d *ServiceScaleDialog) Validate() bool {
	d.errorMsg = ""

	// Check if empty
	if strings.TrimSpace(d.desiredCount) == "" {
		d.errorMsg = "Desired count is required"
		return false
	}

	// Parse number
	count, err := strconv.Atoi(d.desiredCount)
	if err != nil {
		d.errorMsg = "Invalid number"
		return false
	}

	// Check range
	if count < 0 {
		d.errorMsg = "Count cannot be negative"
		return false
	}

	if count > 100 {
		d.errorMsg = "Count cannot exceed 100"
		return false
	}

	return true
}

// GetDesiredCount returns the desired count as integer
func (d *ServiceScaleDialog) GetDesiredCount() int {
	count, _ := strconv.Atoi(d.desiredCount)
	return count
}

// IsOKFocused returns true if OK button is focused
func (d *ServiceScaleDialog) IsOKFocused() bool {
	return d.focusedButton == 0
}

// IsInputFocused returns true if input field is focused
func (d *ServiceScaleDialog) IsInputFocused() bool {
	return d.focusedButton == -1
}

// IsCancelFocused returns true if Cancel button is focused
func (d *ServiceScaleDialog) IsCancelFocused() bool {
	return d.focusedButton == 1
}

// IsCloseFocused returns true if Close button is focused
func (d *ServiceScaleDialog) IsCloseFocused() bool {
	return d.focusedButton == 2
}

// GetMessage returns the dialog message
func (d *ServiceScaleDialog) GetMessage() string {
	if d.confirmMode {
		newCount := d.GetDesiredCount()
		if newCount == d.currentCount {
			return fmt.Sprintf("Service '%s' already has %d task(s)", d.serviceName, d.currentCount)
		} else if newCount > d.currentCount {
			return fmt.Sprintf("Scale UP service '%s' from %d to %d task(s)?",
				d.serviceName, d.currentCount, newCount)
		} else {
			return fmt.Sprintf("Scale DOWN service '%s' from %d to %d task(s)?",
				d.serviceName, d.currentCount, newCount)
		}
	}
	return fmt.Sprintf("Scale service '%s' (current: %d)", d.serviceName, d.currentCount)
}

// SetConfirmMode sets the confirm mode
func (d *ServiceScaleDialog) SetConfirmMode(confirm bool) {
	d.confirmMode = confirm
}

// Helper function to check if string is a valid number
func isValidNumber(s string) bool {
	if s == "" {
		return true
	}
	_, err := strconv.Atoi(s)
	return err == nil
}
