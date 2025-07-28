// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/components/help"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/keys"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/clusters"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/dashboard"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/instances"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/services"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/taskdefs"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/tasks"
)

// ViewType represents the different views available in the TUI
type ViewType int

const (
	ViewDashboard ViewType = iota
	ViewClusters
	ViewServices
	ViewTasks
	ViewTaskDefs
	ViewInstances
	ViewHelp
)

// App represents the main TUI application
type App struct {
	endpoint        string
	currentInstance string
	currentView     ViewType
	apiClient       *api.Client
	dashboard       *dashboard.Model
	clusterList     *clusters.Model
	serviceList     *services.Model
	taskList        *tasks.Model
	taskDefList     *taskdefs.Model
	instanceList    *instances.Model
	quickSwitch     *instances.QuickSwitchModel
	width           int
	height          int
	ready           bool
	quitting        bool
	keyMap          keys.KeyMap
	help            *help.ContextualHelp
}

// New creates a new TUI application
func New(endpoint string) (*App, error) {
	dashboardModel, err := dashboard.New(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create dashboard: %w", err)
	}

	clusterListModel, err := clusters.New(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster list: %w", err)
	}

	serviceListModel, err := services.New(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create service list: %w", err)
	}

	taskListModel, err := tasks.New(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create task list: %w", err)
	}

	taskDefListModel, err := taskdefs.New(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create task definition list: %w", err)
	}

	instanceListModel, err := instances.New(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance list: %w", err)
	}

	// Try to detect current instance from endpoint
	currentInstance := detectCurrentInstance(endpoint)
	
	quickSwitchModel, err := instances.NewQuickSwitch(currentInstance)
	if err != nil {
		return nil, fmt.Errorf("failed to create quick switch: %w", err)
	}
	
	// Create API client
	apiClient := api.NewClient(endpoint)

	app := &App{
		endpoint:        endpoint,
		currentInstance: currentInstance,
		currentView:     ViewDashboard,
		apiClient:      apiClient,
		dashboard:       dashboardModel,
		clusterList:     clusterListModel,
		serviceList:     serviceListModel,
		taskList:        taskListModel,
		taskDefList:     taskDefListModel,
		instanceList:    instanceListModel,
		quickSwitch:     quickSwitchModel,
		keyMap:          keys.DefaultKeyMap(),
		help:            help.NewContextualHelp(),
	}
	
	// Set current instance in dashboard
	dashboardModel.SetCurrentInstance(currentInstance)
	
	return app, nil
}

// Run starts the TUI application
func (a *App) Run() error {
	p := tea.NewProgram(a, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		a.dashboard.Init(),
		a.clusterList.Init(),
		a.serviceList.Init(),
		a.taskList.Init(),
		a.taskDefList.Init(),
		a.instanceList.Init(),
	)
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		
		// Update view sizes
		a.dashboard.SetSize(msg.Width, msg.Height-3) // Leave room for header and footer
		a.clusterList.SetSize(msg.Width, msg.Height-3)
		a.serviceList.SetSize(msg.Width, msg.Height-3)
		a.taskList.SetSize(msg.Width, msg.Height-3)
		a.taskDefList.SetSize(msg.Width, msg.Height-3)
		a.instanceList.SetSize(msg.Width, msg.Height-3)
		a.quickSwitch.SetSize(msg.Width, msg.Height)
		a.help.SetSize(msg.Width, msg.Height-3)
		
	case instanceSwitchedMsg:
		if msg.err != nil {
			// Handle error - maybe show a notification
			// For now, just log the error
			return a, nil
		}
		// Re-initialize all views after instance switch
		cmds = append(cmds, 
			a.dashboard.Init(),
			a.clusterList.Init(),
			a.serviceList.Init(),
			a.taskList.Init(),
			a.taskDefList.Init(),
		)
		
	case tea.KeyMsg:
		switch {
		case keys.Matches(msg, a.keyMap.Quit):
			a.quitting = true
			return a, tea.Quit
			
		case keys.Matches(msg, a.keyMap.Help):
			// Toggle help in the contextual help system
			a.help.Toggle()
			
		case keys.Matches(msg, a.keyMap.Dashboard):
			a.currentView = ViewDashboard
			a.help.SetContext(help.ContextDashboard)
			
		case keys.Matches(msg, a.keyMap.Clusters):
			a.currentView = ViewClusters
			a.help.SetContext(help.ContextClusterList)
			
		case keys.Matches(msg, a.keyMap.Services):
			a.currentView = ViewServices
			a.help.SetContext(help.ContextServiceList)
			
		case keys.Matches(msg, a.keyMap.Tasks):
			a.currentView = ViewTasks
			a.help.SetContext(help.ContextTaskList)
			
		case keys.Matches(msg, a.keyMap.TaskDefs):
			a.currentView = ViewTaskDefs
			a.help.SetContext(help.ContextTaskDefList)
			
		case msg.String() == "6":
			a.currentView = ViewInstances
			// TODO: Add ContextInstanceList when available
			// For now, use dashboard context as placeholder
			a.help.SetContext(help.ContextDashboard)
			
		case msg.String() == "i":
			// Show quick instance switch
			if !a.quickSwitch.IsVisible() {
				return a, a.quickSwitch.Show()
			}
		}
	}

	// Update quick switch first if visible
	if a.quickSwitch.IsVisible() {
		var quickSwitchCmd tea.Cmd
		a.quickSwitch, quickSwitchCmd = a.quickSwitch.Update(msg)
		cmds = append(cmds, quickSwitchCmd)
		
		// Check if an instance was selected
		if instanceName, selected := a.quickSwitch.GetSelectedInstance(); selected {
			// Switch to the selected instance
			cmds = append(cmds, a.switchToInstance(instanceName))
		}
		
		// Don't process other updates if quick switch is visible
		if a.quickSwitch.IsVisible() {
			return a, tea.Batch(cmds...)
		}
	}
	
	// Update help system
	var helpCmd tea.Cmd
	a.help, helpCmd = a.help.Update(msg)
	cmds = append(cmds, helpCmd)

	// Update the current view
	switch a.currentView {
	case ViewDashboard:
		var dashboardCmd tea.Cmd
		a.dashboard, dashboardCmd = a.dashboard.Update(msg)
		cmds = append(cmds, dashboardCmd)
		
	case ViewClusters:
		var clusterCmd tea.Cmd
		a.clusterList, clusterCmd = a.clusterList.Update(msg)
		cmds = append(cmds, clusterCmd)
		
	case ViewServices:
		var serviceCmd tea.Cmd
		a.serviceList, serviceCmd = a.serviceList.Update(msg)
		cmds = append(cmds, serviceCmd)
		
	case ViewTasks:
		var taskCmd tea.Cmd
		a.taskList, taskCmd = a.taskList.Update(msg)
		cmds = append(cmds, taskCmd)
		
	case ViewTaskDefs:
		var taskDefCmd tea.Cmd
		a.taskDefList, taskDefCmd = a.taskDefList.Update(msg)
		cmds = append(cmds, taskDefCmd)
		
	case ViewInstances:
		var instanceCmd tea.Cmd
		newModel, instanceCmd := a.instanceList.Update(msg)
		if model, ok := newModel.(*instances.Model); ok {
			a.instanceList = model
		}
		cmds = append(cmds, instanceCmd)
	}

	return a, tea.Batch(cmds...)
}

// View implements tea.Model
func (a *App) View() string {
	if !a.ready {
		return "\n  Initializing..."
	}

	if a.quitting {
		return ""
	}

	var content string
	
	// Render header
	header := a.renderHeader()
	
	// Render current view
	switch a.currentView {
	case ViewDashboard:
		content = a.dashboard.View()
		
	case ViewClusters:
		content = a.clusterList.View()
		
	case ViewServices:
		content = a.serviceList.View()
		
	case ViewTasks:
		content = a.taskList.View()
		
	case ViewTaskDefs:
		content = a.taskDefList.View()
		
	case ViewInstances:
		content = a.instanceList.View()
		
	case ViewHelp:
		content = a.renderHelp()
		
	default:
		content = "\n  View not implemented yet"
	}
	
	// Render footer with help
	footer := a.renderFooter()
	
	// Compose the view
	mainView := header + "\n" + content + "\n" + footer
	
	// If quick switch is shown, overlay it
	if a.quickSwitch.IsVisible() {
		return a.quickSwitch.View()
	}
	
	// If help is shown, overlay it
	helpView := a.help.View()
	if strings.Contains(helpView, "Help - ") { // Check if full help is shown
		// Split main view into lines
		lines := strings.Split(mainView, "\n")
		dimmed := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		var dimmedView strings.Builder
		for i, line := range lines {
			if i > 0 {
				dimmedView.WriteString("\n")
			}
			dimmedView.WriteString(dimmed.Render(line))
		}
		
		// Simple overlay approach: just render the help below the dimmed view
		return dimmedView.String() + "\n\n" + helpView
	}
	
	return mainView
}

func (a *App) renderHeader() string {
	title := "KECS TUI"
	if a.currentInstance != "" {
		title = fmt.Sprintf("KECS TUI [%s]", a.currentInstance)
	}
	title = fmt.Sprintf("%s - %s", title, a.endpoint)
	return styles.Header.Render(title)
}

func (a *App) renderFooter() string {
	// Get context-specific help from the help system
	helpLine := a.help.View()
	
	// If it's not the full help, use it as the footer
	if !strings.Contains(helpLine, "Help - ") {
		return styles.Footer.Render(helpLine)
	}
	
	// Otherwise, show the default navigation help
	var items []string
	
	items = append(items, "1:Dashboard")
	items = append(items, "2:Clusters")
	items = append(items, "3:Services")
	items = append(items, "4:Tasks")
	items = append(items, "5:TaskDefs")
	items = append(items, "6:Instances")
	items = append(items, "i:Switch")
	items = append(items, "?:Help")
	items = append(items, "q:Quit")
	
	footer := ""
	for i, item := range items {
		if i > 0 {
			footer += " • "
		}
		footer += item
	}
	
	return styles.Footer.Render(footer)
}

func (a *App) renderHelp() string {
	help := `
# KECS TUI Help

## Navigation
- 1: Switch to Dashboard view
- 2: Switch to Clusters view
- 3: Switch to Services view
- 4: Switch to Tasks view
- 5: Switch to Task Definitions view
- 6: Switch to Instances view
- i: Quick instance switch
- ?: Toggle this help screen
- q: Quit the application

## General Keys
- ↑/k: Move up
- ↓/j: Move down
- ←/h: Move left
- →/l: Move right
- Enter: Select/Confirm
- Esc: Cancel/Back
- Tab: Next field
- Shift+Tab: Previous field

## View-Specific Keys
Each view has additional keyboard shortcuts. Press '?' within a view to see its specific help.
`
	return styles.Content.Render(help)
}

// detectCurrentInstance tries to detect the current instance from the endpoint
func detectCurrentInstance(endpoint string) string {
	// Check if it's a local endpoint
	if strings.Contains(endpoint, "localhost") || strings.Contains(endpoint, "127.0.0.1") {
		// Extract port from endpoint
		parts := strings.Split(endpoint, ":")
		if len(parts) >= 3 {
			// TODO: Map port to instance name
			// For now, return "local"
			return "local"
		}
	}
	
	// For remote endpoints, we can't determine the instance name
	return ""
}

// switchToInstance switches the TUI to a different instance
func (a *App) switchToInstance(instanceName string) tea.Cmd {
	return func() tea.Msg {
		// Get instance details to find the port
		ctx := context.Background()
		instanceManager, err := api.NewInstanceManager()
		if err != nil {
			return instanceSwitchedMsg{
				instanceName: instanceName,
				err:          fmt.Errorf("failed to create instance manager: %w", err),
			}
		}
		
		instance, err := instanceManager.GetInstance(ctx, instanceName)
		if err != nil {
			return instanceSwitchedMsg{
				instanceName: instanceName,
				err:          fmt.Errorf("failed to get instance details: %w", err),
			}
		}
		
		if instance.Status != api.InstanceRunning {
			return instanceSwitchedMsg{
				instanceName: instanceName,
				err:          fmt.Errorf("instance %s is not running", instanceName),
			}
		}
		
		// Build the new endpoint URL
		newEndpoint := fmt.Sprintf("http://localhost:%d", instance.APIPort)
		
		// Update the API client endpoint
		a.apiClient.SetEndpoint(newEndpoint)
		
		// Update current instance
		a.currentInstance = instanceName
		a.endpoint = newEndpoint
		
		// Update all views with new endpoint
		a.dashboard.SetCurrentInstance(instanceName)
		// Update other views to use the new endpoint
		a.clusterList.SetEndpoint(newEndpoint)
		a.serviceList.SetEndpoint(newEndpoint)
		a.taskList.SetEndpoint(newEndpoint)
		a.taskDefList.SetEndpoint(newEndpoint)
		
		return instanceSwitchedMsg{instanceName: instanceName}
	}
}

// instanceSwitchedMsg is sent when instance is switched
type instanceSwitchedMsg struct {
	instanceName string
	err          error
}