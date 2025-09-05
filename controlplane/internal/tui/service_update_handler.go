package tui

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// handleServiceUpdateDialogKeys handles key events for the service update dialog
func (m Model) handleServiceUpdateDialogKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.serviceUpdateDialog == nil {
		return m, nil
	}

	d := m.serviceUpdateDialog

	switch msg.String() {
	case "esc":
		// Cancel dialog
		m.serviceUpdateDialog = nil
		return m, nil

	case "up", "k":
		// Move selection up in list
		if d.IsListFocused() {
			d.MoveSelectionUp()
		}

	case "down", "j":
		// Move selection down in list
		if d.IsListFocused() {
			d.MoveSelectionDown()
		}

	case "tab":
		// Move focus between list and buttons
		d.MoveFocus()

	case "enter":
		if d.IsListFocused() {
			// In list, move to OK button
			d.focusedButton = 0
		} else if d.IsOKFocused() {
			if !d.confirmMode {
				// Validate and show confirmation
				if d.Validate() {
					d.SetConfirmMode(true)
				}
			} else {
				// Execute update
				return m.executeServiceUpdate()
			}
		} else if d.IsCancelFocused() || d.IsCloseFocused() {
			// Cancel dialog
			m.serviceUpdateDialog = nil
			return m, nil
		}
	}

	return m, nil
}

// executeServiceUpdate executes the service update operation
func (m Model) executeServiceUpdate() (Model, tea.Cmd) {
	if m.serviceUpdateDialog == nil {
		return m, nil
	}

	d := m.serviceUpdateDialog
	newTaskDef := d.GetSelectedTaskDef()
	serviceName := d.serviceName

	// Close dialog and show progress
	m.serviceUpdateDialog = nil
	m.updatingInProgress = true
	m.updatingServiceName = serviceName
	m.updatingTaskDef = newTaskDef

	// Verify service exists
	serviceFound := false
	for _, svc := range m.services {
		if svc.Name == serviceName {
			serviceFound = true
			break
		}
	}

	if !serviceFound {
		m.updatingInProgress = false
		return m, nil
	}

	// Create the update command
	return m, m.updateService(serviceName, newTaskDef)
}

// fetchTaskDefinitionsForUpdate fetches available task definitions and opens the update dialog
func (m Model) fetchTaskDefinitionsForUpdate(serviceName string, currentTaskDef string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Extract current task def family:revision format if it's an ARN
		currentTaskDefFormatted := currentTaskDef
		if strings.Contains(currentTaskDef, "task-definition/") {
			parts := strings.Split(currentTaskDef, "/")
			if len(parts) > 1 {
				currentTaskDefFormatted = parts[len(parts)-1]
			}
		}

		// Extract the family name from current task definition
		currentFamily := ""
		if parts := strings.Split(currentTaskDefFormatted, ":"); len(parts) > 0 {
			currentFamily = parts[0]
		}

		// Fetch all task definitions from the instance
		taskDefs, err := m.apiClient.ListTaskDefinitions(ctx, m.selectedInstance)
		if err != nil {
			// If error, create dialog with fallback data
			return TaskDefinitionsFetchedMsg{
				ServiceName:    serviceName,
				CurrentTaskDef: currentTaskDefFormatted,
				TaskDefs: []string{
					currentTaskDefFormatted, // Always include current
				},
				Error: err,
			}
		}

		// Extract and filter task definitions by the same family
		// This matches AWS ECS console behavior which shows revisions from the same family
		// TODO: Consider adding an option to switch families (like "Change task definition family" in AWS console)
		var familyTaskDefs []string
		for _, arn := range taskDefs {
			// ARN format: arn:aws:ecs:region:account:task-definition/family:revision
			parts := strings.Split(arn, "/")
			if len(parts) > 1 {
				familyRevision := parts[len(parts)-1]
				// Check if this task def belongs to the same family
				if currentFamily != "" && strings.HasPrefix(familyRevision, currentFamily+":") {
					familyTaskDefs = append(familyTaskDefs, familyRevision)
				}
			}
		}

		// If no task definitions from the same family found, include at least the current one
		if len(familyTaskDefs) == 0 {
			familyTaskDefs = []string{currentTaskDefFormatted}
		}

		// Sort by revision number (descending - newest first)
		sort.Slice(familyTaskDefs, func(i, j int) bool {
			// Extract revision numbers
			iRev := extractRevision(familyTaskDefs[i])
			jRev := extractRevision(familyTaskDefs[j])
			return iRev > jRev // Descending order
		})

		return TaskDefinitionsFetchedMsg{
			ServiceName:    serviceName,
			CurrentTaskDef: currentTaskDefFormatted,
			TaskDefs:       familyTaskDefs,
		}
	}
}

// extractRevision extracts the revision number from a task definition string
func extractRevision(taskDef string) int {
	parts := strings.Split(taskDef, ":")
	if len(parts) > 1 {
		rev, _ := strconv.Atoi(parts[len(parts)-1])
		return rev
	}
	return 0
}

// TaskDefinitionsFetchedMsg is sent when task definitions are fetched
type TaskDefinitionsFetchedMsg struct {
	ServiceName    string
	CurrentTaskDef string
	TaskDefs       []string
	Error          error
}

// updateService creates a command to update the service
func (m Model) updateService(serviceName string, taskDef string) tea.Cmd {
	return func() tea.Msg {
		// Call UpdateService API
		err := m.apiClient.UpdateServiceTaskDefinition(
			m.selectedInstance,
			m.selectedCluster,
			serviceName,
			taskDef,
		)

		if err != nil {
			return ServiceUpdatedMsg{
				Success: false,
				Error:   err,
			}
		}

		// Wait a bit to show progress
		time.Sleep(1 * time.Second)

		return ServiceUpdatedMsg{
			Success:     true,
			ServiceName: serviceName,
			TaskDef:     taskDef,
		}
	}
}

// ServiceUpdatedMsg is sent when a service update operation completes
type ServiceUpdatedMsg struct {
	Success     bool
	ServiceName string
	TaskDef     string
	Error       error
}
