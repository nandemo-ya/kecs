package tui

import (
	"fmt"
	"strings"
)

// ServiceUpdateDialog represents a dialog for updating a service
type ServiceUpdateDialog struct {
	serviceName        string
	currentTaskDef     string
	availableTaskDefs  []string
	selectedTaskDefIdx int
	confirmMode        bool
	focusedButton      int // -1: Task def list, 0: OK, 1: Cancel, 2: Close
	errorMsg           string
}

// NewServiceUpdateDialog creates a new service update dialog
func NewServiceUpdateDialog(serviceName, currentTaskDef string, availableTaskDefs []string) *ServiceUpdateDialog {
	// Find current task def index
	currentIdx := 0
	for i, td := range availableTaskDefs {
		if td == currentTaskDef {
			currentIdx = i
			break
		}
	}

	return &ServiceUpdateDialog{
		serviceName:        serviceName,
		currentTaskDef:     currentTaskDef,
		availableTaskDefs:  availableTaskDefs,
		selectedTaskDefIdx: currentIdx,
		confirmMode:        false,
		focusedButton:      -1, // Start with task def list focused
	}
}

// MoveSelectionUp moves the task definition selection up
func (d *ServiceUpdateDialog) MoveSelectionUp() {
	if d.selectedTaskDefIdx > 0 && d.focusedButton == -1 {
		d.selectedTaskDefIdx--
	}
}

// MoveSelectionDown moves the task definition selection down
func (d *ServiceUpdateDialog) MoveSelectionDown() {
	if d.selectedTaskDefIdx < len(d.availableTaskDefs)-1 && d.focusedButton == -1 {
		d.selectedTaskDefIdx++
	}
}

// MoveFocus moves focus between task def list and buttons
func (d *ServiceUpdateDialog) MoveFocus() {
	if d.confirmMode {
		// In confirm mode, cycle through OK, Cancel, Close
		if d.focusedButton < 0 {
			d.focusedButton = 0
		} else {
			d.focusedButton = (d.focusedButton + 1) % 3
		}
	} else {
		// In selection mode, cycle through list, OK, Cancel, Close
		d.focusedButton = (d.focusedButton+2)%4 - 1 // -1, 0, 1, 2
	}
}

// Validate validates the selection
func (d *ServiceUpdateDialog) Validate() bool {
	d.errorMsg = ""

	// Check if selection has changed
	if d.selectedTaskDefIdx < 0 || d.selectedTaskDefIdx >= len(d.availableTaskDefs) {
		d.errorMsg = "Invalid selection"
		return false
	}

	selectedTaskDef := d.availableTaskDefs[d.selectedTaskDefIdx]
	if selectedTaskDef == d.currentTaskDef {
		d.errorMsg = "Same task definition selected. No update needed."
		return false
	}

	return true
}

// GetSelectedTaskDef returns the selected task definition
func (d *ServiceUpdateDialog) GetSelectedTaskDef() string {
	if d.selectedTaskDefIdx >= 0 && d.selectedTaskDefIdx < len(d.availableTaskDefs) {
		return d.availableTaskDefs[d.selectedTaskDefIdx]
	}
	return ""
}

// IsListFocused returns true if task def list is focused
func (d *ServiceUpdateDialog) IsListFocused() bool {
	return d.focusedButton == -1
}

// IsOKFocused returns true if OK button is focused
func (d *ServiceUpdateDialog) IsOKFocused() bool {
	return d.focusedButton == 0
}

// IsCancelFocused returns true if Cancel button is focused
func (d *ServiceUpdateDialog) IsCancelFocused() bool {
	return d.focusedButton == 1
}

// IsCloseFocused returns true if Close button is focused
func (d *ServiceUpdateDialog) IsCloseFocused() bool {
	return d.focusedButton == 2
}

// GetMessage returns the dialog message
func (d *ServiceUpdateDialog) GetMessage() string {
	if d.confirmMode {
		newTaskDef := d.GetSelectedTaskDef()
		// Extract family and revision for cleaner display
		currentParts := strings.Split(d.currentTaskDef, ":")
		newParts := strings.Split(newTaskDef, ":")

		currentDisplay := d.currentTaskDef
		newDisplay := newTaskDef
		if len(currentParts) == 2 {
			currentDisplay = fmt.Sprintf("%s (rev %s)", currentParts[0], currentParts[1])
		}
		if len(newParts) == 2 {
			newDisplay = fmt.Sprintf("%s (rev %s)", newParts[0], newParts[1])
		}

		return fmt.Sprintf("Update service '%s' from:\n  %s\nto:\n  %s?",
			d.serviceName, currentDisplay, newDisplay)
	}
	return fmt.Sprintf("Update service '%s' (current: %s)", d.serviceName, d.currentTaskDef)
}

// SetConfirmMode sets the confirm mode
func (d *ServiceUpdateDialog) SetConfirmMode(confirm bool) {
	d.confirmMode = confirm
	if confirm {
		d.focusedButton = 0 // Focus OK button in confirm mode
	}
}
