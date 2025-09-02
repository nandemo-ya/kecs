package tui

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/mock"
)

// refreshInstancesMsg is sent to trigger instance list refresh
type refreshInstancesMsg struct{}

// Run starts the TUI application
func Run() error {
	// Suppress logging output while in TUI mode
	// This prevents k3d and other components from writing logs to the terminal
	logging.SetOutput(io.Discard)

	// Load configuration
	cfg := LoadConfig()

	// Create API client
	client := CreateAPIClient(cfg)

	// Create model with client
	var model Model
	if cfg.UseMockData {
		model = NewModel() // Uses mock client by default
	} else {
		model = NewModelWithClient(client)
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	// Clean up resources
	if client != nil {
		client.Close()
	}

	return nil
}

// Update handles all messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle global keys first
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global navigation
		switch msg.String() {
		case "ctrl+c":
			// Exit application (k9s style - only Ctrl+C exits)
			if !m.searchMode && !m.commandMode && m.currentView != ViewTaskDefinitionEditor {
				return m, tea.Quit
			}
		case "esc":
			// Go back to previous view or cancel current mode (k9s style)
			if m.searchMode {
				m.searchMode = false
				m.searchQuery = ""
			} else if m.commandMode {
				m.commandMode = false
				m.commandInput = ""
			} else if m.showHelp {
				m.showHelp = false
			} else if m.currentView == ViewCommandPalette {
				m.currentView = m.previousView
				m.commandPalette.Reset()
			} else if m.currentView == ViewTaskDescribe {
				m.currentView = m.previousView
			} else if m.currentView == ViewTaskDefinitionRevisions && m.showTaskDefJSON {
				// Special case: just hide JSON view without navigating
				m.showTaskDefJSON = false
			} else if m.currentView != ViewInstances {
				// Go back to previous view if not at root
				m.goBack()
				return m, m.loadDataFromAPI()
			}
			return m, nil
		case "?":
			if !m.searchMode && !m.commandMode {
				m.showHelp = !m.showHelp
				if m.showHelp {
					m.previousView = m.currentView
					m.currentView = ViewHelp
				} else {
					m.currentView = m.previousView
				}
			}
			return m, nil
		}

		// Handle input modes
		if m.searchMode {
			return m.handleSearchInput(msg)
		}
		if m.commandMode {
			return m.handleCommandInput(msg)
		}
		if m.currentView == ViewCommandPalette {
			return m.handleCommandPaletteInput(msg)
		}
		if m.currentView == ViewInstanceCreate {
			return m.handleInstanceCreateInput(msg)
		}
		if m.currentView == ViewInstanceSwitcher {
			return m.handleInstanceSwitcherInput(msg)
		}

		// Handle service scale dialog
		if m.serviceScaleDialog != nil {
			return m.handleServiceScaleDialogKeys(msg)
		}

		// Handle service update dialog
		if m.serviceUpdateDialog != nil {
			return m.handleServiceUpdateDialogKeys(msg)
		}

		// View-specific key handling
		switch m.currentView {
		case ViewInstances:
			m, cmd = m.handleInstancesKeys(msg)
		case ViewClusters:
			m, cmd = m.handleClustersKeys(msg)
		case ViewServices:
			m, cmd = m.handleServicesKeys(msg)
		case ViewTasks:
			m, cmd = m.handleTasksKeys(msg)
		case ViewTaskDescribe:
			m, cmd = m.handleTaskDescribeKeys(msg)
		case ViewLogs:
			m, cmd = m.handleLogsKeys(msg)
		case ViewHelp:
			m, cmd = m.handleHelpKeys(msg)
		case ViewConfirmDialog:
			m, cmd = m.handleConfirmDialogKeys(msg)
		case ViewTaskDefinitionFamilies:
			m, cmd = m.handleTaskDefinitionFamiliesKeys(msg)
		case ViewTaskDefinitionRevisions:
			m, cmd = m.handleTaskDefinitionRevisionsKeys(msg)
		case ViewTaskDefinitionEditor:
			m, cmd = m.handleTaskDefinitionEditorKeys(msg)
		case ViewClusterCreate:
			m, cmd = m.handleClusterCreateKeys(msg)
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case spinner.TickMsg:
		// Update spinner if deleting
		if m.isDeleting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tickMsg:
		// Update data periodically
		cmds = append(cmds, tickCmd())
		if m.lastUpdate.Add(m.refreshInterval).Before(time.Time(msg)) {
			cmds = append(cmds, m.loadMockDataCmd())
			m.lastUpdate = time.Time(msg)
		}
		// Check if command result should be cleared
		if m.commandPalette != nil {
			m.commandPalette.ShouldShowResult() // This will clear expired results
		}

	case TaskDefinitionsFetchedMsg:
		// Handle fetched task definitions and open dialog
		if msg.Error != nil {
			// Still show dialog with fallback data if there was an error
			if logFile := os.Getenv("KECS_TUI_DEBUG_LOG"); logFile != "" {
				if f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					logger := log.New(f, "", log.LstdFlags)
					logger.Printf("Error fetching task definitions: %v\n", msg.Error)
					f.Close()
				}
			}
		}
		m.serviceUpdateDialog = NewServiceUpdateDialog(msg.ServiceName, msg.CurrentTaskDef, msg.TaskDefs)
		return m, nil
	case ServiceUpdatedMsg:
		// Handle service update completion
		m.updatingInProgress = false

		// Log the result for debugging
		if logFile := os.Getenv("KECS_TUI_DEBUG_LOG"); logFile != "" {
			if f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				logger := log.New(f, "", log.LstdFlags)
				if msg.Success {
					logger.Printf("Service updated successfully: %s to %s\n", msg.ServiceName, msg.TaskDef)
				} else {
					logger.Printf("Service update failed: %v\n", msg.Error)
				}
				f.Close()
			}
		}

		if msg.Success {
			// Refresh services to show updated task definition
			cmds = append(cmds, m.loadDataFromAPI())
		} else {
			// Show error (could add error dialog here)
			m.err = msg.Error
		}

	case ServiceScaledMsg:
		// Handle service scaling completion
		m.scalingInProgress = false

		// Log the result for debugging
		if logFile := os.Getenv("KECS_TUI_DEBUG_LOG"); logFile != "" {
			if f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				logger := log.New(f, "", log.LstdFlags)
				if msg.Success {
					logger.Printf("Service scaled successfully: %s to %d\n", msg.ServiceName, msg.DesiredCount)
				} else {
					logger.Printf("Service scaling failed: %v\n", msg.Error)
				}
				f.Close()
			}
		}

		if msg.Success {
			// Refresh services to show updated count
			cmds = append(cmds, m.loadDataFromAPI())
		} else {
			// Show error (could add error dialog here)
			m.err = msg.Error
		}

	case statusTickMsg:
		// Update instance statuses periodically
		cmds = append(cmds, statusTickCmd())
		if m.currentView == ViewInstances {
			cmds = append(cmds, m.updateInstanceStatusCmd())
		}

	case DataLoadedMsg:
		m.instances = msg.Instances
		m.clusters = msg.Clusters
		m.services = msg.Services
		m.tasks = msg.Tasks
		m.logs = msg.Logs

	case dataLoadedMsg:
		// Handle API data
		m.instances = msg.instances
		m.clusters = msg.clusters
		m.services = msg.services
		m.tasks = msg.tasks

	case logsLoadedMsg:
		// Handle loaded logs
		m.logs = msg.logs
		// If we have an active log viewer, pass the message to it
		if m.logViewer != nil {
			updatedViewer, cmd := m.logViewer.Update(msg)
			m.logViewer = &updatedViewer
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case taskDetailLoadedMsg:
		// Handle loaded task details
		m.selectedTaskDetail = msg.detail
		m.taskDescribeScroll = 0 // Reset scroll position when new details are loaded

	case instanceCreatedMsg:
		// Add new instance to list
		m.instances = append(m.instances, msg.instance)
		// Select the new instance
		m.selectedInstance = msg.instance.Name

		// Reset instance form and close it
		if m.instanceForm != nil {
			m.instanceForm.successMsg = fmt.Sprintf("Instance '%s' created successfully!", msg.instance.Name)
			m.instanceForm.isCreating = false
			m.instanceForm.errorMsg = ""
			m.instanceForm.creationSteps = nil

			// Show success message briefly then close
			cmds = append(cmds, tea.Sequence(
				tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return closeFormMsg{}
				}),
			))
		} else {
			// Navigate directly if no form
			m.currentView = ViewInstances
			cmds = append(cmds, m.loadMockDataCmd())
		}

	case instanceStatusUpdateMsg:
		// Update instance statuses
		m.instances = msg.instances

	case taskDefFamiliesLoadedMsg:
		// Update task definition families
		m.taskDefFamilies = msg.families

	case taskDefRevisionsLoadedMsg:
		// Update task definition revisions
		m.taskDefRevisions = msg.revisions

	case logViewerCreatedMsg:
		// Set log viewer
		m.logViewer = &msg.viewer
		m.logViewerTaskArn = msg.taskArn
		m.logViewerContainer = msg.container
		m.currentView = ViewLogs

		// Initialize the log viewer to start loading logs
		cmd := m.logViewer.Init()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case taskDefJSONLoadedMsg:
		// Cache loaded JSON
		m.taskDefJSONCache[msg.revision] = msg.json

		// If we're in the editor view, pass the JSON to the editor
		if m.currentView == ViewTaskDefinitionEditor && m.taskDefEditor != nil {
			m.taskDefEditor, _ = m.taskDefEditor.Update(msg)
		}

	case editorSaveMsg:
		// Handle editor save
		// In a real implementation, this would create a new revision
		m.commandPalette.lastResult = fmt.Sprintf("Task definition %s saved as revision %d", msg.family, msg.revision)
		m.commandPalette.showResult = true
		// Go back to revisions view
		m.currentView = ViewTaskDefinitionRevisions
		m.taskDefEditor = nil
		// Reload revisions
		cmds = append(cmds, m.loadTaskDefinitionRevisionsCmd())

	case editorQuitMsg:
		// Handle editor quit without saving
		m.currentView = m.previousView
		m.taskDefEditor = nil

	case closeFormMsg:
		// Close the instance creation form and go back to instances view
		if m.currentView == ViewInstanceCreate {
			m.currentView = ViewInstances
			if m.instanceForm != nil {
				m.instanceForm.Reset()
			}
			// Reload instances list
			cmds = append(cmds, m.loadMockDataCmd())
		}

	case instanceCreationStatusMsg:
		// Update creation steps status
		if m.instanceForm != nil && m.instanceForm.isCreating {
			m.instanceForm.creationSteps = msg.steps
			m.instanceForm.elapsedTime = msg.elapsed
			// Start ticker for status updates
			cmds = append(cmds, m.updateCreationStatusCmd())
		}

	case actualCreationStatusMsg:
		// Update creation steps based on actual status from API
		if m.instanceForm != nil && m.instanceForm.isCreating && msg.status != nil {
			// Find the step that matches the current status
			for i := range m.instanceForm.creationSteps {
				step := &m.instanceForm.creationSteps[i]

				// Update matching step
				if step.Name == msg.status.Step {
					step.Status = msg.status.Status
					if msg.status.Status == "failed" && msg.status.Message != "" {
						// Show error message
						m.instanceForm.errorMsg = msg.status.Message
					}
				} else if step.Status == "running" && msg.status.Step != step.Name {
					// Previous running step is now done
					step.Status = "done"
				}
			}
		}

	case creationStatusTickMsg:
		// Update creation progress every second
		if m.instanceForm != nil && m.instanceForm.isCreating {
			elapsed := time.Since(m.instanceForm.startTime)
			m.instanceForm.elapsedTime = fmt.Sprintf("%.0fs", elapsed.Seconds())

			// Get the instance name from form
			instanceName := m.instanceForm.instanceName
			if instanceName != "" {
				// Start async status check
				cmds = append(cmds, m.checkCreationStatusCmd(instanceName))
			}

			// Check all steps status
			allDone := true
			hasFailed := false
			for _, step := range m.instanceForm.creationSteps {
				if step.Status == "failed" {
					hasFailed = true
					break
				}
				if step.Status != "done" {
					allDone = false
				}
			}

			// If all done, failed, or timeout, stop ticking
			if allDone || hasFailed || elapsed > 3*time.Minute {
				if allDone && !hasFailed {
					// All steps completed successfully
					m.instanceForm.isCreating = false
					m.instanceForm.successMsg = fmt.Sprintf("Instance '%s' created successfully!", instanceName)
					// Auto-close form after showing success message
					cmds = append(cmds, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
						return closeFormMsg{}
					}))
				} else if hasFailed {
					// Creation failed, keep form open to show error
					m.instanceForm.isCreating = false
				} else if elapsed > 3*time.Minute && !m.instanceForm.showTimeoutPrompt {
					// Timeout
					m.instanceForm.showTimeoutPrompt = true
				}
			} else {
				// Continue ticking
				cmds = append(cmds, m.updateCreationStatusCmd())
			}
		}

	case instanceCreationTimeoutMsg:
		// Show timeout prompt
		if m.instanceForm != nil && m.instanceForm.isCreating {
			m.instanceForm.showTimeoutPrompt = true
			m.instanceForm.elapsedTime = msg.elapsed
		}

	case instanceCreationContinueMsg:
		// Handle timeout response
		if m.instanceForm != nil {
			m.instanceForm.showTimeoutPrompt = false
			if !msg.continueWaiting {
				// Abort creation
				m.instanceForm.isCreating = false
				m.instanceForm.errorMsg = "Instance creation aborted"
				m.instanceForm.successMsg = ""
			}
		}

	case instanceDeletingMsg:
		// Start deletion process with spinner
		m.isDeleting = true
		m.deletingMessage = fmt.Sprintf("Deleting instance '%s'...", msg.name)
		// Start spinner
		cmds = append(cmds, m.spinner.Tick)
		// Perform the actual deletion
		cmds = append(cmds, m.performInstanceDeletionCmd(msg.name))

	case instanceDeletedMsg:
		// Deletion completed
		m.isDeleting = false
		m.deletingMessage = ""
		if msg.err != nil {
			m.err = msg.err
		} else {
			// If the deleted instance was selected, clear selection
			if m.selectedInstance == msg.name {
				m.selectedInstance = ""
			}
			// Reload instances
			cmds = append(cmds, m.loadDataFromAPI())
		}
		// Go back to previous view if we were in confirm dialog
		if m.currentView == ViewConfirmDialog {
			m.confirmDialog = nil
			m.currentView = m.previousView
		}

	case clusterCreatingMsg:
		// Handle cluster creation in progress
		if m.clusterForm != nil {
			// Update the form with the message
			m.clusterForm, cmd = m.clusterForm.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// Perform the actual API call
			cmds = append(cmds, m.createClusterCmd(msg.clusterName, msg.region))
		}

	case clusterCreatedMsg:
		// Handle cluster creation completed
		if m.clusterForm != nil {
			m.clusterForm, cmd = m.clusterForm.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case clusterFormCloseMsg:
		// Close the cluster form
		m.clusterForm = nil
		m.currentView = m.previousView
		// Reload clusters
		cmds = append(cmds, m.loadDataFromAPI())

	case errMsg:
		// Handle API errors
		m.err = msg.err
		// If we're in instance creation, show error in form
		if m.currentView == ViewInstanceCreate && m.instanceForm != nil {
			m.instanceForm.errorMsg = msg.err.Error()
			m.instanceForm.successMsg = ""
			m.instanceForm.isCreating = false
		}

	case mock.DataMsg:
		// Convert mock data to model data
		m.instances = make([]Instance, len(msg.Instances))
		for i, inst := range msg.Instances {
			m.instances[i] = Instance{
				Name:     inst.Name,
				Status:   inst.Status,
				Clusters: inst.Clusters,
				Services: inst.Services,
				Tasks:    inst.Tasks,
				APIPort:  inst.APIPort,
				Age:      inst.Age,
			}
		}

		m.clusters = make([]Cluster, len(msg.Clusters))
		for i, cl := range msg.Clusters {
			m.clusters[i] = Cluster{
				Name:     cl.Name,
				Status:   cl.Status,
				Region:   cl.Region,
				Services: cl.Services,
				Tasks:    cl.Tasks,
				Age:      cl.Age,
			}
		}

		m.services = make([]Service, len(msg.Services))
		for i, svc := range msg.Services {
			m.services[i] = Service{
				Name:    svc.Name,
				Desired: svc.Desired,
				Running: svc.Running,
				Pending: svc.Pending,
				Status:  svc.Status,
				TaskDef: svc.TaskDef,
				Age:     svc.Age,
			}
		}

		m.tasks = make([]Task, len(msg.Tasks))
		for i, task := range msg.Tasks {
			m.tasks[i] = Task{
				ID:      task.ID,
				Service: task.Service,
				Status:  task.Status,
				Health:  task.Health,
				CPU:     task.CPU,
				Memory:  task.Memory,
				IP:      task.IP,
				Age:     task.Age,
			}
		}

		m.logs = make([]LogEntry, len(msg.Logs))
		for i, log := range msg.Logs {
			m.logs[i] = LogEntry{
				Timestamp: log.Timestamp,
				Level:     log.Level,
				Message:   log.Message,
			}
		}

	}

	return m, tea.Batch(cmds...)
}

// View renders the current view
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// For help view, use full screen
	if m.currentView == ViewHelp {
		return m.renderHelpView()
	}

	// For command palette, use overlay
	if m.currentView == ViewCommandPalette {
		return m.renderCommandPaletteOverlay()
	}

	// For instance create, use overlay
	if m.currentView == ViewInstanceCreate {
		return m.renderInstanceCreateOverlay()
	}

	// For cluster create, use overlay
	if m.currentView == ViewClusterCreate {
		return m.renderClusterCreateOverlay()
	}

	// For task describe, use full screen
	if m.currentView == ViewTaskDescribe {
		return m.renderTaskDescribe()
	}

	// For confirm dialog, use overlay
	if m.currentView == ViewConfirmDialog {
		return m.renderConfirmDialogOverlay()
	}

	// For instance switcher, use overlay
	if m.currentView == ViewInstanceSwitcher {
		return m.renderInstanceSwitcherOverlay()
	}

	// For task definition editor, use full screen
	if m.currentView == ViewTaskDefinitionEditor {
		if m.taskDefEditor != nil {
			return m.taskDefEditor.Render(m.width, m.height)
		}
		// Fallback if editor is nil
		return m.View()
	}

	// For logs view with active log viewer, use full screen
	if m.currentView == ViewLogs && m.logViewer != nil {
		return m.logViewer.View()
	}

	// If deleting, show spinner overlay
	if m.isDeleting {
		return m.renderDeletingOverlay()
	}

	// Service scale dialog has priority
	if m.serviceScaleDialog != nil {
		return m.renderServiceScaleDialog()
	}

	// Service update dialog
	if m.serviceUpdateDialog != nil {
		return m.renderServiceUpdateDialog()
	}

	// Scaling progress overlay
	if m.scalingInProgress {
		return m.renderScalingProgress()
	}

	// Updating progress overlay
	if m.updatingInProgress {
		return m.renderUpdatingProgress()
	}

	// Calculate exact heights for panels
	footerHeight := 1
	availableHeight := m.height - footerHeight
	navPanelHeight := int(float64(availableHeight) * 0.3)
	resourcePanelHeight := availableHeight - navPanelHeight

	// Ensure minimum heights
	if navPanelHeight < 10 {
		navPanelHeight = 10
	}
	if resourcePanelHeight < 10 {
		resourcePanelHeight = 10
	}

	// Render navigation panel (30% height)
	navigationPanel := m.renderNavigationPanel()

	// Render resource panel (70% height)
	resourcePanel := m.renderResourcePanel()

	// Render footer
	footer := m.renderFooter()

	// Combine all components without extra spacing
	// The panels already have their own borders and padding
	return lipgloss.JoinVertical(
		lipgloss.Top,
		navigationPanel,
		resourcePanel,
		footer,
	)
}

// Key handlers for each view

func (m Model) handleInstancesKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()
	case "enter":
		if len(m.instances) > 0 {
			m.selectedInstance = m.instances[m.instanceCursor].Name
			m.currentView = ViewClusters
			m.clusterCursor = 0
			return m, m.loadDataFromAPI()
		}
	case "y":
		// Copy instance name to clipboard
		if len(m.instances) > 0 && m.instanceCursor < len(m.instances) {
			inst := m.instances[m.instanceCursor]
			err := copyToClipboard(inst.Name)
			if err == nil {
				m.clipboardMsg = fmt.Sprintf("Copied: %s", inst.Name)
				m.clipboardMsgTime = time.Now()
			} else {
				m.clipboardMsg = fmt.Sprintf("Copy failed: %v", err)
				m.clipboardMsgTime = time.Now()
			}
		}
	case "n":
		// Open instance creation form
		if m.instanceForm == nil {
			m.instanceForm = NewInstanceFormWithSuggestions(m.instances)
		} else {
			// Reset with new suggestions
			m.instanceForm = NewInstanceFormWithSuggestions(m.instances)
		}
		m.previousView = m.currentView
		m.currentView = ViewInstanceCreate
		return m, nil
	case "s":
		// Start/Stop instance
		if len(m.instances) > 0 && m.instanceCursor < len(m.instances) {
			instanceName := m.instances[m.instanceCursor].Name
			instanceStatus := strings.ToLower(m.instances[m.instanceCursor].Status)

			// Toggle based on current status
			if instanceStatus == "stopped" {
				// Create start confirmation dialog
				m.confirmDialog = StartInstanceDialog(
					instanceName,
					func() error {
						// Update local status immediately for UI feedback
						m.instances[m.instanceCursor].Status = "Starting"
						// Don't return error here, handle asynchronously
						return nil
					},
					func() {
						// Cancel - just close dialog
					},
				)
				// Store the command to execute after confirmation
				m.pendingCommand = m.startInstanceCmd(instanceName)
				m.previousView = m.currentView
				m.currentView = ViewConfirmDialog
				return m, nil
			} else if instanceStatus == "running" {
				// Create stop confirmation dialog
				m.confirmDialog = StopInstanceDialog(
					instanceName,
					func() error {
						// Update local status immediately for UI feedback
						m.instances[m.instanceCursor].Status = "Stopping"
						// Don't return error here, handle asynchronously
						return nil
					},
					func() {
						// Cancel - just close dialog
					},
				)
				// Store the command to execute after confirmation
				m.pendingCommand = m.stopInstanceCmd(instanceName)
				m.previousView = m.currentView
				m.currentView = ViewConfirmDialog
				return m, nil
			} else {
				// Instance is in transition state, don't allow toggle
				m.err = fmt.Errorf("Cannot start/stop instance in %s state", m.instances[m.instanceCursor].Status)
				return m, nil
			}
		}
	case "d":
		// Delete instance
		if len(m.instances) > 0 && m.instanceCursor < len(m.instances) {
			instanceName := m.instances[m.instanceCursor].Name

			// Don't allow deleting "default" instance
			if instanceName == "default" {
				m.err = fmt.Errorf("Cannot delete default instance")
				return m, nil
			}

			// Don't allow delete if already deleting
			if m.isDeleting {
				return m, nil
			}

			// Create confirmation dialog
			m.confirmDialog = DeleteInstanceDialog(
				instanceName,
				func() error {
					// Just return nil here, actual deletion will be handled via message
					return nil
				},
				func() {
					// Cancel - just close dialog
				},
			)

			// Store the pending deletion command
			m.pendingCommand = m.deleteInstanceCmd(instanceName)
			m.previousView = m.currentView
			m.currentView = ViewConfirmDialog
			return m, nil
		}
	case "ctrl+i":
		// Quick switch instance
		if len(m.instances) > 1 {
			m.instanceSwitcher = NewInstanceSwitcher(m.instances)
			m.previousView = m.currentView
			m.currentView = ViewInstanceSwitcher
		}
	case "t":
		// Navigate to task definitions
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
		}
	case "T":
		// Navigate to all tasks in cluster (uppercase T)
		// This key is not valid from instances view since no cluster is selected
		// User must first navigate to a cluster
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "r":
		return m, m.loadDataFromAPI()
	}
	return m, nil
}

func (m Model) handleClusterCreateKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.clusterForm == nil {
		return m, nil
	}

	// Let the form handle the key event (convert tea.KeyMsg to tea.Msg)
	updatedForm, cmd := m.clusterForm.Update(tea.Msg(msg))

	if updatedForm == nil {
		// Form closed
		m.clusterForm = nil
		m.currentView = m.previousView
		return m, m.loadDataFromAPI() // Reload clusters after creation
	}

	m.clusterForm = updatedForm
	return m, cmd
}

func (m Model) handleClustersKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()
	case "enter":
		if len(m.clusters) > 0 {
			m.selectedCluster = m.clusters[m.clusterCursor].Name
			m.currentView = ViewServices
			m.serviceCursor = 0
			return m, m.loadDataFromAPI()
		}
	case "i":
		m.currentView = ViewInstances
		m.selectedInstance = ""
	case "s":
		if m.selectedCluster != "" {
			m.currentView = ViewServices
		}
	case "n":
		// Create new cluster
		if m.selectedInstance != "" {
			m.clusterForm = NewClusterForm()
			m.previousView = m.currentView
			m.currentView = ViewClusterCreate
		}
	// Removed 'd' binding as it conflicts with 'D' for delete cluster
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "r":
		return m, m.loadDataFromAPI()
	case "ctrl+i":
		// Quick switch instance
		if len(m.instances) > 1 {
			m.instanceSwitcher = NewInstanceSwitcher(m.instances)
			m.previousView = m.currentView
			m.currentView = ViewInstanceSwitcher
		}
	case "t":
		// Navigate to task definitions
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
		}
	case "T":
		// Navigate to all tasks in selected cluster (uppercase T)
		if m.selectedInstance != "" && len(m.clusters) > 0 {
			// Select current cluster if not already selected
			if m.selectedCluster == "" && m.clusterCursor < len(m.clusters) {
				m.selectedCluster = m.clusters[m.clusterCursor].Name
			}
			if m.selectedCluster != "" {
				m.currentView = ViewTasks
				m.taskCursor = 0
				// Clear service selection to show all cluster tasks
				m.selectedService = ""
				return m, m.loadDataFromAPI()
			}
		}
	}
	return m, nil
}

func (m Model) handleServicesKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()
	case "enter":
		if len(m.services) > 0 {
			m.selectedService = m.services[m.serviceCursor].Name
			m.currentView = ViewTasks
			m.taskCursor = 0
			return m, m.loadDataFromAPI()
		}
	case "i":
		m.currentView = ViewInstances
		m.selectedInstance = ""
	case "c":
		m.currentView = ViewClusters
		m.selectedCluster = ""
	// Removed 't' binding as it conflicts with 'T' for task definitions
	case "r":
		// Restart service or refresh (context-dependent)
		return m, m.loadDataFromAPI()
	case "s":
		// Scale service
		if len(m.services) > 0 && m.serviceCursor < len(m.services) {
			service := m.services[m.serviceCursor]
			m.serviceScaleDialog = NewServiceScaleDialog(service.Name, service.Desired)
		}
	case "u":
		// Update service - fetch available task definitions
		if len(m.services) > 0 && m.serviceCursor < len(m.services) {
			service := m.services[m.serviceCursor]
			// Start fetching task definitions
			return m, m.fetchTaskDefinitionsForUpdate(service.Name, service.TaskDef)
		}
	case "l":
		// View logs
		if len(m.services) > 0 {
			m.previousView = m.currentView
			m.currentView = ViewLogs
			return m, m.loadDataFromAPI()
		}
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "ctrl+i":
		// Quick switch instance
		if len(m.instances) > 1 {
			m.instanceSwitcher = NewInstanceSwitcher(m.instances)
			m.previousView = m.currentView
			m.currentView = ViewInstanceSwitcher
		}
	case "t":
		// Navigate to task definitions
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
		}
	}
	return m, nil
}

func (m Model) handleTaskDescribeKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Calculate content size for proper scroll limits
	// This is a simplified check - the actual calculation happens in render
	maxScroll := 100 // Default reasonable max
	if m.selectedTaskDetail != nil {
		// Rough estimate: each section has about 10-20 lines
		estimatedLines := 30                                        // Overview + Network + Timestamps
		estimatedLines += len(m.selectedTaskDetail.Containers) * 10 // Each container ~10 lines
		maxScroll = estimatedLines
	}

	switch msg.String() {
	case "up", "k":
		if m.taskDescribeScroll > 0 {
			m.taskDescribeScroll--
		}
	case "down", "j":
		// Check against estimated max
		if m.taskDescribeScroll < maxScroll {
			m.taskDescribeScroll++
		}
	case "pgup", "ctrl+u":
		// Page up
		m.taskDescribeScroll -= 10
		if m.taskDescribeScroll < 0 {
			m.taskDescribeScroll = 0
		}
	case "pgdown", "ctrl+d":
		// Page down
		m.taskDescribeScroll += 10
		if m.taskDescribeScroll > maxScroll {
			m.taskDescribeScroll = maxScroll
		}
	case "home", "g":
		// Go to top
		m.taskDescribeScroll = 0
	case "end", "G":
		// Go to bottom (will be adjusted in render)
		m.taskDescribeScroll = maxScroll
	case "l":
		// View logs for the detailed task
		if m.selectedTaskDetail != nil {
			m.previousView = m.currentView
			// Use first container name if available
			containerName := ""
			if len(m.selectedTaskDetail.Containers) > 0 {
				containerName = m.selectedTaskDetail.Containers[0].Name
			}
			// Open log viewer with the task ARN
			return m, m.viewTaskLogsCmd(m.selectedTaskDetail.TaskARN, containerName)
		}
	case "esc", "backspace":
		// Go back to tasks list
		m.currentView = m.previousView
		m.selectedTaskDetail = nil
		m.taskDescribeScroll = 0
	case "r":
		// TODO: Implement restart task
		// This would require calling StopTask and then RunTask
	case "s":
		// TODO: Implement stop task
		// This would require calling StopTask API
	}
	return m, nil
}

func (m Model) handleTasksKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()
	case "enter":
		// View task details
		if len(m.tasks) > 0 && m.taskCursor < len(m.tasks) {
			m.selectedTask = m.tasks[m.taskCursor].ID
			m.previousView = m.currentView
			m.currentView = ViewTaskDescribe
			m.selectedTaskDetail = nil // Clear previous details
			m.taskDescribeScroll = 0
			return m, m.loadTaskDetailsCmd()
		}
	case "l":
		// View logs for selected task
		if len(m.tasks) > 0 && m.taskCursor < len(m.tasks) {
			task := m.tasks[m.taskCursor]
			m.selectedTask = task.ID
			m.previousView = m.currentView

			// Use first container name if available
			containerName := ""
			if len(task.Containers) > 0 {
				containerName = task.Containers[0]
			}

			// Open log viewer
			return m, m.viewTaskLogsCmd(task.ARN, containerName)
		}
	case "i":
		m.currentView = ViewInstances
		m.selectedInstance = ""
	case "c":
		m.currentView = ViewClusters
		m.selectedCluster = ""
	case "esc", "b":
		// Go back to previous view
		if m.selectedService != "" {
			// If we came from services view, go back to services
			m.currentView = ViewServices
		} else {
			// If we came from clusters view (showing all tasks), go back to clusters
			m.currentView = ViewClusters
		}
		m.selectedTask = ""
	case "s":
		m.currentView = ViewServices
		m.selectedService = ""
	case "d":
		// Describe task
		if len(m.tasks) > 0 {
			m.selectedTask = m.tasks[m.taskCursor].ID
			m.previousView = m.currentView
			m.currentView = ViewTaskDescribe
		}
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "r":
		return m, m.loadDataFromAPI()
	case "ctrl+i":
		// Quick switch instance
		if len(m.instances) > 1 {
			m.instanceSwitcher = NewInstanceSwitcher(m.instances)
			m.previousView = m.currentView
			m.currentView = ViewInstanceSwitcher
		}
	case "t":
		// Navigate to task definitions
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
		}
	}
	return m, nil
}

func (m Model) handleLogsKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	// If we have an active log viewer, delegate to it
	if m.logViewer != nil {
		updatedViewer, cmd := m.logViewer.Update(msg)
		m.logViewer = &updatedViewer

		// Check if user wants to exit log viewer (k9s style - only ESC)
		if msg.String() == "esc" {
			m.logViewer = nil
			m.currentView = m.previousView
			return m, m.loadDataFromAPI()
		}

		return m, cmd
	}

	// Otherwise handle log list view
	switch msg.String() {
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()
	case "esc":
		m.currentView = m.previousView
		return m, m.loadDataFromAPI()
	case "enter":
		// Open log viewer for selected task
		if m.selectedTask != "" && len(m.tasks) > m.taskCursor {
			task := m.tasks[m.taskCursor]
			// Use first container name if available
			containerName := ""
			if len(task.Containers) > 0 {
				containerName = task.Containers[0]
			}
			return m, m.viewTaskLogsCmd(task.ARN, containerName)
		}
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	}
	return m, nil
}

func (m Model) handleHelpKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?":
		// Close help (k9s style - ESC or ?)
		m.showHelp = false
		m.currentView = m.previousView
	}
	return m, nil
}

func (m Model) handleSearchInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	previousQuery := m.searchQuery

	switch msg.String() {
	case "enter":
		m.searchMode = false
	case "esc":
		m.searchMode = false
		m.searchQuery = ""
		m.resetCursorAfterSearch()
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			m.searchQuery += msg.String()
		}
	}

	// Reset cursor if search query changed
	if previousQuery != m.searchQuery {
		m.resetCursorAfterSearch()
	}

	return m, nil
}

func (m Model) handleCommandInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.commandMode = false
		// If no input, show command palette
		if m.commandInput == "" {
			// Ensure command palette is initialized
			if m.commandPalette == nil {
				m.commandPalette = NewCommandPalette()
			}
			m.previousView = m.currentView
			m.currentView = ViewCommandPalette
			m.commandPalette.Reset()
			m.commandPalette.FilterCommands("", &m)
			return m, nil
		}
		// Execute direct command
		cmd := m.commandInput
		m.commandInput = ""
		// Ensure command palette is initialized
		if m.commandPalette == nil {
			m.commandPalette = NewCommandPalette()
		}
		result, err := m.commandPalette.ExecuteByName(cmd, &m)
		if err != nil {
			m.err = err
		} else {
			m.commandPalette.lastResult = result
			m.commandPalette.showResult = true
		}
		return m, m.loadDataFromAPI()
	case "tab":
		// Switch to command palette for autocomplete
		if m.commandInput != "" {
			m.previousView = m.currentView
			m.currentView = ViewCommandPalette
			m.commandPalette.FilterCommands(m.commandInput, &m)
		}
		return m, nil
	case "up":
		// Navigate command history
		if cmd := m.commandPalette.PreviousFromHistory(); cmd != "" {
			m.commandInput = cmd
		}
		return m, nil
	case "down":
		// Navigate command history
		if cmd := m.commandPalette.NextFromHistory(); cmd != "" {
			m.commandInput = cmd
		}
		return m, nil
	case "backspace":
		if len(m.commandInput) > 0 {
			m.commandInput = m.commandInput[:len(m.commandInput)-1]
		}
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			m.commandInput += msg.String()
		}
	}
	return m, nil
}

func (m Model) handleConfirmDialogKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.confirmDialog == nil {
		// Safety check - go back to previous view
		m.currentView = m.previousView
		return m, nil
	}

	switch msg.String() {
	case "left", "h":
		m.confirmDialog.FocusYes()
	case "right", "l":
		m.confirmDialog.FocusNo()
	case "tab":
		// Toggle between Yes and No
		if m.confirmDialog.focused {
			m.confirmDialog.FocusNo()
		} else {
			m.confirmDialog.FocusYes()
		}
	case "enter", " ":
		// Execute the selected action
		err := m.confirmDialog.Execute()
		if err != nil {
			m.err = err
		}
		// Store pending command if any
		cmd := m.pendingCommand
		// Clear dialog and pending command
		m.confirmDialog = nil
		m.pendingCommand = nil
		m.currentView = m.previousView
		// Execute pending command if it exists, otherwise reload data
		if cmd != nil {
			return m, cmd
		}
		// Reload data after potential deletion
		return m, m.loadDataFromAPI()
	case "esc":
		// Cancel and go back (k9s style - only ESC cancels)
		if m.confirmDialog.onNo != nil {
			m.confirmDialog.onNo()
		}
		m.confirmDialog = nil
		m.pendingCommand = nil // Clear pending command on cancel
		m.currentView = m.previousView
	}
	return m, nil
}

func (m Model) handleCommandPaletteInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.currentView = m.previousView
		m.commandPalette.Reset()
		return m, nil
	case "enter":
		_, err := m.commandPalette.ExecuteCommand(&m)
		if err != nil {
			m.err = err
			// Stay in command palette to show error
			return m, nil
		}
		// Command executed successfully
		m.currentView = m.previousView
		m.commandPalette.Reset()
		return m, m.loadDataFromAPI()
	case "up", "ctrl+p":
		m.commandPalette.MoveUp()
		return m, nil
	case "down", "ctrl+n":
		m.commandPalette.MoveDown()
		return m, nil
	case "backspace":
		if len(m.commandPalette.query) > 0 {
			m.commandPalette.query = m.commandPalette.query[:len(m.commandPalette.query)-1]
			m.commandPalette.FilterCommands(m.commandPalette.query, &m)
		}
		return m, nil
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			m.commandPalette.query += msg.String()
			m.commandPalette.FilterCommands(m.commandPalette.query, &m)
		}
	}
	return m, nil
}

func (m Model) handleInstanceCreateInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.instanceForm == nil {
		return m, nil
	}

	// If creating, only allow ESC to cancel
	if m.instanceForm.isCreating {
		if msg.String() == "esc" {
			m.instanceForm.isCreating = false
			m.instanceForm.successMsg = ""
			m.currentView = m.previousView
			m.instanceForm.Reset()
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		// Close form
		m.currentView = m.previousView
		m.instanceForm.Reset()
		return m, nil

	case "tab":
		// Move to next field
		m.instanceForm.MoveFocusDown()
		return m, nil

	case "shift+tab":
		// Move to previous field
		m.instanceForm.MoveFocusUp()
		return m, nil

	case "enter":
		// Handle action based on focused field
		switch m.instanceForm.focusedField {
		case FieldInstanceCloseButton:
			// Close form
			m.currentView = m.previousView
			m.instanceForm.Reset()
			return m, nil
		case FieldInstanceName:
			// If on name field and pressed enter, generate new name
			m.instanceForm.GenerateNewName()
			return m, nil
		case FieldSubmit:
			// Validate form
			if !m.instanceForm.Validate() {
				// Validation errors are already set in form
				return m, nil
			}

			// Get form data
			formData := m.instanceForm.GetFormData()

			// Create API options
			opts := api.CreateInstanceOptions{
				Name:       formData["instanceName"].(string),
				APIPort:    formData["apiPort"].(int),
				AdminPort:  formData["adminPort"].(int),
				LocalStack: formData["localStack"].(bool),
			}

			// Initialize creation steps
			steps := []CreationStep{
				{Name: "Creating k3d cluster", Status: "pending"},
				{Name: "Deploying control plane", Status: "pending"},
			}
			if opts.LocalStack {
				steps = append(steps, CreationStep{Name: "Starting LocalStack", Status: "pending"})
			}
			steps = append(steps, CreationStep{Name: "Finalizing", Status: "pending"})

			// Set initial state
			m.instanceForm.successMsg = "Creating instance..."
			m.instanceForm.isCreating = true
			m.instanceForm.errorMsg = ""
			m.instanceForm.creationSteps = steps
			m.instanceForm.startTime = time.Now()

			// Start creation and monitoring
			return m, tea.Batch(
				m.createInstanceCmd(opts),
				m.monitorInstanceCreation(opts.Name, opts.LocalStack),
			)
		case FieldCancel:
			// Cancel and close
			m.currentView = m.previousView
			m.instanceForm.Reset()
			return m, nil
		}

	case " ", "space":
		// Toggle checkbox or press button
		switch m.instanceForm.focusedField {
		// No checkboxes anymore, LocalStack is always enabled
		case FieldSubmit:
			// Same as enter on submit
			if !m.instanceForm.Validate() {
				// Validation errors are already set in form
				return m, nil
			}

			// Get form data
			formData := m.instanceForm.GetFormData()

			// Create API options
			opts := api.CreateInstanceOptions{
				Name:       formData["instanceName"].(string),
				APIPort:    formData["apiPort"].(int),
				AdminPort:  formData["adminPort"].(int),
				LocalStack: formData["localStack"].(bool),
			}

			// Initialize creation steps
			steps := []CreationStep{
				{Name: "Creating k3d cluster", Status: "pending"},
				{Name: "Deploying control plane", Status: "pending"},
			}
			if opts.LocalStack {
				steps = append(steps, CreationStep{Name: "Starting LocalStack", Status: "pending"})
			}
			steps = append(steps, CreationStep{Name: "Finalizing", Status: "pending"})

			// Set initial state
			m.instanceForm.successMsg = "Creating instance..."
			m.instanceForm.isCreating = true
			m.instanceForm.errorMsg = ""
			m.instanceForm.creationSteps = steps
			m.instanceForm.startTime = time.Now()

			// Start creation and monitoring
			return m, tea.Batch(
				m.createInstanceCmd(opts),
				m.monitorInstanceCreation(opts.Name, opts.LocalStack),
			)
		case FieldCancel:
			m.currentView = m.previousView
			m.instanceForm.Reset()
			return m, nil
		}

	case "backspace":
		// Remove character from text field
		m.instanceForm.RemoveLastChar()
		return m, nil

	default:
		// Handle text input
		if len(msg.String()) == 1 {
			m.instanceForm.UpdateField(m.instanceForm.GetCurrentFieldValue() + msg.String())
		}
	}

	return m, nil
}

// handleInstanceSwitcherInput handles input for instance switcher
func (m Model) handleInstanceSwitcherInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.instanceSwitcher == nil {
		// Safety check
		m.currentView = m.previousView
		return m, nil
	}

	switch msg.String() {
	case "esc":
		// Cancel switching
		m.currentView = m.previousView
		m.instanceSwitcher = nil
		return m, nil

	case "enter":
		// Switch to selected instance
		selected := m.instanceSwitcher.GetSelected()
		if selected != "" && selected != m.selectedInstance {
			m.selectedInstance = selected
			// Find the instance cursor position
			for i, inst := range m.instances {
				if inst.Name == selected {
					m.instanceCursor = i
					break
				}
			}
			// Navigate to clusters view
			m.currentView = ViewClusters
			m.clusterCursor = 0
			m.instanceSwitcher = nil
			return m, m.loadDataFromAPI()
		}
		// If same instance selected, just close
		m.currentView = m.previousView
		m.instanceSwitcher = nil
		return m, nil

	case "up", "ctrl+p":
		m.instanceSwitcher.MoveUp()
		return m, nil

	case "down", "ctrl+n":
		m.instanceSwitcher.MoveDown()
		return m, nil

	case "backspace":
		query := m.instanceSwitcher.query
		if len(query) > 0 {
			m.instanceSwitcher.SetQuery(query[:len(query)-1])
		}
		return m, nil

	default:
		// Handle text input
		if len(msg.String()) == 1 || msg.String() == " " {
			m.instanceSwitcher.SetQuery(m.instanceSwitcher.query + msg.String())
		}
	}

	return m, nil
}

// handleTaskDefinitionFamiliesKeys handles input for task definition families view
func (m Model) handleTaskDefinitionFamiliesKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()
	case "enter":
		if len(m.taskDefFamilies) > 0 && m.taskDefFamilyCursor < len(m.taskDefFamilies) {
			m.selectedFamily = m.taskDefFamilies[m.taskDefFamilyCursor].Family
			m.currentView = ViewTaskDefinitionRevisions
			m.taskDefRevisionCursor = 0
			m.showTaskDefJSON = false
			return m, m.loadTaskDefinitionRevisionsCmd()
		}
	case "c":
		// Switch to clusters view
		m.currentView = ViewClusters
		m.clusterCursor = 0
		return m, m.loadDataFromAPI()
	case "n":
		// Create new task definition
		m.taskDefEditor = NewTaskDefinitionEditor("new-task-definition", nil)
		m.previousView = m.currentView
		m.currentView = ViewTaskDefinitionEditor
		// Load a template
		return m, func() tea.Msg {
			template := `{
  "family": "new-task-definition",
  "containerDefinitions": [
    {
      "name": "main",
      "image": "nginx:latest",
      "memory": 512,
      "cpu": 256,
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "protocol": "tcp"
        }
      ]
    }
  ],
  "requiresCompatibilities": ["EC2"],
  "networkMode": "bridge",
  "memory": "512",
  "cpu": "256"
}`
			return taskDefJSONLoadedMsg{
				revision: 0,
				json:     template,
			}
		}
	case "C":
		// Copy selected family's latest revision
		// TODO: Implement copy functionality
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "r":
		return m, m.loadTaskDefinitionFamiliesCmd()
	case "ctrl+i":
		// Quick switch instance
		if len(m.instances) > 1 {
			m.instanceSwitcher = NewInstanceSwitcher(m.instances)
			m.previousView = m.currentView
			m.currentView = ViewInstanceSwitcher
		}
	}
	return m, nil
}

// handleTaskDefinitionRevisionsKeys handles input for task definition revisions view
func (m Model) handleTaskDefinitionRevisionsKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.taskDefRevisionCursor > 0 {
			m.taskDefRevisionCursor--
		}
	case "down", "j":
		if m.taskDefRevisionCursor < len(m.taskDefRevisions)-1 {
			m.taskDefRevisionCursor++
		}
	case "enter":
		// Toggle JSON display
		m.showTaskDefJSON = !m.showTaskDefJSON
		if m.showTaskDefJSON && len(m.taskDefRevisions) > 0 && m.taskDefRevisionCursor < len(m.taskDefRevisions) {
			// Load full task definition for selected revision
			selectedRev := m.taskDefRevisions[m.taskDefRevisionCursor]
			// Check if already cached
			if _, cached := m.taskDefJSONCache[selectedRev.Revision]; !cached {
				// Create task definition ARN
				taskDefArn := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:task-definition/%s:%d",
					selectedRev.Family, selectedRev.Revision)
				return m, m.loadTaskDefinitionJSONCmd(taskDefArn)
			}
		}
		// Clear JSON cache to save memory
		m.taskDefJSONCache = make(map[int]string)
	case "e":
		// Edit as new revision
		if len(m.taskDefRevisions) > 0 && m.taskDefRevisionCursor < len(m.taskDefRevisions) {
			selectedRev := m.taskDefRevisions[m.taskDefRevisionCursor]
			// Create editor with the selected revision as base
			m.taskDefEditor = NewTaskDefinitionEditor(selectedRev.Family, &selectedRev.Revision)
			m.previousView = m.currentView
			m.currentView = ViewTaskDefinitionEditor

			// Load the JSON content for editing
			if jsonContent, cached := m.taskDefJSONCache[selectedRev.Revision]; cached {
				// Use cached content
				return m, func() tea.Msg {
					return taskDefJSONLoadedMsg{
						revision: selectedRev.Revision,
						json:     jsonContent,
					}
				}
			} else {
				// Load from API
				taskDefArn := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:task-definition/%s:%d",
					selectedRev.Family, selectedRev.Revision)
				return m, m.loadTaskDefinitionJSONCmd(taskDefArn)
			}
		}
	case "y":
		// Yank (copy) family:revision to clipboard
		if len(m.taskDefRevisions) > 0 && m.taskDefRevisionCursor < len(m.taskDefRevisions) {
			rev := m.taskDefRevisions[m.taskDefRevisionCursor]
			taskDefName := fmt.Sprintf("%s:%d", rev.Family, rev.Revision)
			err := copyToClipboard(taskDefName)
			if err == nil {
				m.clipboardMsg = fmt.Sprintf("Copied: %s", taskDefName)
				m.clipboardMsgTime = time.Now()
			} else {
				m.clipboardMsg = fmt.Sprintf("Copy failed: %v", err)
				m.clipboardMsgTime = time.Now()
			}
		}
	case "c":
		// Copy full task definition JSON to clipboard
		// TODO: Implement full JSON copy
	// Removed lowercase 'd' - use uppercase 'D' for deregister to avoid conflicts
	case "a":
		// Activate revision
		// TODO: Implement activate
	case "d":
		// Enter diff mode
		// TODO: Implement diff mode
	case "ctrl+u", "pgup":
		// Scroll JSON up half page
		if m.showTaskDefJSON {
			m.taskDefJSONScroll -= 10
			if m.taskDefJSONScroll < 0 {
				m.taskDefJSONScroll = 0
			}
		}
	case "ctrl+d", "pgdown":
		// Scroll JSON down half page
		if m.showTaskDefJSON {
			m.taskDefJSONScroll += 10
		}
	case "g":
		// Go to top of JSON
		if m.showTaskDefJSON {
			m.taskDefJSONScroll = 0
		}
	case "G":
		// Go to bottom of JSON
		if m.showTaskDefJSON {
			// Will be adjusted in render to max value
			m.taskDefJSONScroll = 99999
		}
	case "J":
		// Scroll JSON down one line
		if m.showTaskDefJSON {
			m.taskDefJSONScroll++
		}
	case "K":
		// Scroll JSON up one line
		if m.showTaskDefJSON {
			m.taskDefJSONScroll--
			if m.taskDefJSONScroll < 0 {
				m.taskDefJSONScroll = 0
			}
		}
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "r":
		return m, m.loadTaskDefinitionRevisionsCmd()
	case "ctrl+i":
		// Quick switch instance
		if len(m.instances) > 1 {
			m.instanceSwitcher = NewInstanceSwitcher(m.instances)
			m.previousView = m.currentView
			m.currentView = ViewInstanceSwitcher
		}
	}
	return m, nil
}

// handleTaskDefinitionEditorKeys handles input for task definition editor view
func (m Model) handleTaskDefinitionEditorKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.taskDefEditor == nil {
		// Safety check - go back if no editor
		m.currentView = m.previousView
		return m, nil
	}

	// Pass the key to the editor
	var cmd tea.Cmd
	m.taskDefEditor, cmd = m.taskDefEditor.Update(msg)

	// Check for editor messages that need handling
	if cmd != nil {
		// We'll handle this through the message system
		return m, cmd
	}

	// Handle ESC to exit editor (if not handled by editor)
	if msg.String() == "esc" && m.taskDefEditor.mode == EditorModeNormal {
		m.currentView = m.previousView
		m.taskDefEditor = nil
	}

	return m, nil
}
