package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/mock"
)

// Run starts the TUI application
func Run() error {
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
		case "ctrl+c", "q":
			if !m.searchMode && !m.commandMode {
				return m, tea.Quit
			}
		case "esc":
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
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

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

	case instanceCreatedMsg:
		// Add new instance to list
		m.instances = append(m.instances, msg.instance)
		// Select the new instance
		m.selectedInstance = msg.instance.Name
		// Reset instance form if it exists
		if m.instanceForm != nil {
			m.instanceForm.successMsg = fmt.Sprintf("Instance '%s' created successfully", msg.instance.Name)
			m.instanceForm.isCreating = false
			m.instanceForm.errorMsg = ""
			// Close form after short delay
			cmds = append(cmds, tea.Sequence(
				tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
					return nil
				}),
				func() tea.Msg {
					// Navigate to clusters view
					m.currentView = ViewClusters
					m.clusterCursor = 0
					m.instanceForm.Reset()
					return m.loadMockDataCmd()()
				},
			))
		} else {
			// Navigate directly if no form
			m.currentView = ViewClusters
			m.clusterCursor = 0
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
	
	case taskDefJSONLoadedMsg:
		// Cache loaded JSON
		m.taskDefJSONCache[msg.revision] = msg.json

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
				Name:        cl.Name,
				Status:      cl.Status,
				Services:    cl.Services,
				Tasks:       cl.Tasks,
				CPUUsed:     cl.CPUUsed,
				CPUTotal:    cl.CPUTotal,
				MemoryUsed:  cl.MemoryUsed,
				MemoryTotal: cl.MemoryTotal,
				Namespace:   cl.Namespace,
				Age:         cl.Age,
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
	
	// For task describe, use full screen
	if m.currentView == ViewTaskDescribe {
		return m.renderTaskDescribeView()
	}
	
	// For confirm dialog, use overlay
	if m.currentView == ViewConfirmDialog {
		return m.renderConfirmDialogOverlay()
	}
	
	// For instance switcher, use overlay
	if m.currentView == ViewInstanceSwitcher {
		return m.renderInstanceSwitcherOverlay()
	}
	
	// For task definition families, use regular view
	if m.currentView == ViewTaskDefinitionFamilies {
		return m.renderTaskDefinitionFamiliesView()
	}
	
	// For task definition revisions, use regular view (possibly 2-column)
	if m.currentView == ViewTaskDefinitionRevisions {
		return m.renderTaskDefinitionRevisionsView()
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
			return m, m.loadMockDataCmd()
		}
	case "N":
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
	case "S":
		// Start/Stop instance (mock)
		if len(m.instances) > 0 {
			// Toggle status in mock
		}
	case "D":
		// Delete instance
		if len(m.instances) > 0 && m.instanceCursor < len(m.instances) {
			instanceName := m.instances[m.instanceCursor].Name
			
			// Don't allow deleting "default" instance  
			if instanceName == "default" {
				m.err = fmt.Errorf("Cannot delete default instance")
				return m, nil
			}
			
			// Create confirmation dialog
			m.confirmDialog = DeleteInstanceDialog(
				instanceName,
				func() error {
					// Delete instance via API
					ctx := context.Background()
					err := m.apiClient.DeleteInstance(ctx, instanceName)
					if err != nil {
						return err
					}
					
					// If the deleted instance was selected, clear selection
					if m.selectedInstance == instanceName {
						m.selectedInstance = ""
					}
					
					// Reload instances
					return nil
				},
				func() {
					// Cancel - just close dialog
				},
			)
			
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
	case "T":
		// Navigate to task definitions
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
		}
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "R":
		return m, m.loadMockDataCmd()
	}
	return m, nil
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
			return m, m.loadMockDataCmd()
		}
	case "backspace":
		m.goBack()
		return m, m.loadMockDataCmd()
	case "i":
		m.currentView = ViewInstances
		m.selectedInstance = ""
	case "s":
		if m.selectedCluster != "" {
			m.currentView = ViewServices
		}
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "R":
		return m, m.loadMockDataCmd()
	case "ctrl+i":
		// Quick switch instance
		if len(m.instances) > 1 {
			m.instanceSwitcher = NewInstanceSwitcher(m.instances)
			m.previousView = m.currentView
			m.currentView = ViewInstanceSwitcher
		}
	case "T":
		// Navigate to task definitions
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
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
			return m, m.loadMockDataCmd()
		}
	case "backspace":
		m.goBack()
		return m, m.loadMockDataCmd()
	case "i":
		m.currentView = ViewInstances
		m.selectedInstance = ""
	case "c":
		m.currentView = ViewClusters
		m.selectedCluster = ""
	case "t":
		if m.selectedService != "" {
			m.currentView = ViewTasks
		}
	case "r":
		// Restart service (mock)
	case "S":
		// Scale service (mock)
		m.commandMode = true
		m.commandInput = fmt.Sprintf("scale service %s ", m.services[m.serviceCursor].Name)
	case "u":
		// Update service (mock)
	case "x":
		// Stop service (mock)
	case "l":
		// View logs
		if len(m.services) > 0 {
			m.previousView = m.currentView
			m.currentView = ViewLogs
			return m, m.loadMockDataCmd()
		}
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "R":
		return m, m.loadMockDataCmd()
	case "ctrl+i":
		// Quick switch instance
		if len(m.instances) > 1 {
			m.instanceSwitcher = NewInstanceSwitcher(m.instances)
			m.previousView = m.currentView
			m.currentView = ViewInstanceSwitcher
		}
	case "T":
		// Navigate to task definitions
		if m.selectedInstance != "" {
			m.currentView = ViewTaskDefinitionFamilies
			m.taskDefFamilyCursor = 0
			return m, m.loadTaskDefinitionFamiliesCmd()
		}
	}
	return m, nil
}

func (m Model) handleTasksKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()
	case "enter", "l":
		// View logs for selected task
		if len(m.tasks) > 0 {
			m.selectedTask = m.tasks[m.taskCursor].ID
			m.previousView = m.currentView
			m.currentView = ViewLogs
			return m, m.loadMockDataCmd()
		}
	case "backspace":
		m.goBack()
		return m, m.loadMockDataCmd()
	case "i":
		m.currentView = ViewInstances
		m.selectedInstance = ""
	case "c":
		m.currentView = ViewClusters
		m.selectedCluster = ""
	case "s":
		m.currentView = ViewServices
		m.selectedService = ""
	case "D":
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
	case "R":
		return m, m.loadMockDataCmd()
	case "ctrl+i":
		// Quick switch instance
		if len(m.instances) > 1 {
			m.instanceSwitcher = NewInstanceSwitcher(m.instances)
			m.previousView = m.currentView
			m.currentView = ViewInstanceSwitcher
		}
	case "T":
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
	switch msg.String() {
	case "up", "k":
		m.moveCursorUp()
	case "down", "j":
		m.moveCursorDown()
	case "esc", "backspace":
		m.currentView = m.previousView
		return m, m.loadMockDataCmd()
	case "f":
		// Toggle follow mode (mock)
	case "s":
		// Save logs (mock)
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	}
	return m, nil
}

func (m Model) handleHelpKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?", "q":
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
		return m, m.loadMockDataCmd()
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
		// Clear dialog and go back
		m.confirmDialog = nil
		m.currentView = m.previousView
		// Reload data after potential deletion
		return m, m.loadMockDataCmd()
	case "esc", "q":
		// Cancel and go back
		if m.confirmDialog.onNo != nil {
			m.confirmDialog.onNo()
		}
		m.confirmDialog = nil
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
		return m, m.loadMockDataCmd()
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
				Traefik:    formData["traefik"].(bool),
				DevMode:    formData["devMode"].(bool),
			}
			
			// Show creating message and set loading state
			m.instanceForm.successMsg = "Creating instance..."
			m.instanceForm.isCreating = true
			m.instanceForm.errorMsg = ""
			
			// Create instance via API
			return m, m.createInstanceCmd(opts)
		case FieldCancel:
			// Cancel and close
			m.currentView = m.previousView
			m.instanceForm.Reset()
			return m, nil
		}
		
	case " ", "space":
		// Toggle checkbox or press button
		switch m.instanceForm.focusedField {
		case FieldLocalStack, FieldTraefik, FieldDevMode:
			m.instanceForm.ToggleCheckbox()
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
				Traefik:    formData["traefik"].(bool),
				DevMode:    formData["devMode"].(bool),
			}
			
			// Show creating message and set loading state
			m.instanceForm.successMsg = "Creating instance..."
			m.instanceForm.isCreating = true
			m.instanceForm.errorMsg = ""
			
			// Create instance via API
			return m, m.createInstanceCmd(opts)
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
			return m, m.loadMockDataCmd()
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
	case "backspace":
		// Go back to instances
		m.currentView = ViewInstances
		m.selectedFamily = ""
		m.taskDefFamilies = []TaskDefinitionFamily{}
	case "c":
		// Switch to clusters view
		m.currentView = ViewClusters
		m.clusterCursor = 0
		return m, m.loadMockDataCmd()
	case "N":
		// Create new task definition
		// TODO: Implement task definition editor
	case "C":
		// Copy selected family's latest revision
		// TODO: Implement copy functionality
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "R":
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
	case "backspace":
		// Go back to families
		m.currentView = ViewTaskDefinitionFamilies
		m.selectedFamily = ""
		m.taskDefRevisions = []TaskDefinitionRevision{}
		m.showTaskDefJSON = false
		// Clear JSON cache to save memory
		m.taskDefJSONCache = make(map[int]string)
	case "e":
		// Edit as new revision
		// TODO: Implement editor
	case "c":
		// Copy to clipboard
		// TODO: Implement clipboard copy
	case "d":
		// Deregister revision
		// TODO: Implement deregister
	case "a":
		// Activate revision
		// TODO: Implement activate
	case "D":
		// Enter diff mode
		// TODO: Implement diff mode
	case "ctrl+u":
		// Scroll JSON up half page
		if m.showTaskDefJSON {
			m.taskDefJSONScroll -= 10
			if m.taskDefJSONScroll < 0 {
				m.taskDefJSONScroll = 0
			}
		}
	case "ctrl+d":
		// Scroll JSON down half page
		if m.showTaskDefJSON {
			m.taskDefJSONScroll += 10
		}
	case "/":
		m.searchMode = true
		m.searchQuery = ""
	case ":":
		m.commandMode = true
		m.commandInput = ""
	case "R":
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
