package tui2

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui2/mock"
)

// Run starts the TUI application
func Run() error {
	p := tea.NewProgram(
		NewModel(),
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

	case DataLoadedMsg:
		m.instances = msg.Instances
		m.clusters = msg.Clusters
		m.services = msg.Services
		m.tasks = msg.Tasks
		m.logs = msg.Logs

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

	case errMsg:
		m.err = msg.err
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
		// Create new instance (mock)
		m.commandMode = true
		m.commandInput = "create instance "
	case "S":
		// Start/Stop instance (mock)
		if len(m.instances) > 0 {
			// Toggle status in mock
		}
	case "D":
		// Delete instance (mock)
		if len(m.instances) > 0 {
			// Show confirmation dialog in real implementation
		}
	case "ctrl+i":
		// Quick switch instance
		// In real implementation, show instance switcher
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
		// Describe task (mock)
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