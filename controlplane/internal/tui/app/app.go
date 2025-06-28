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
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/keys"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/clusters"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/dashboard"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/services"
)

// ViewType represents the different views available in the TUI
type ViewType int

const (
	ViewDashboard ViewType = iota
	ViewClusters
	ViewServices
	ViewTasks
	ViewTaskDefs
	ViewHelp
)

// App represents the main TUI application
type App struct {
	endpoint     string
	currentView  ViewType
	dashboard    *dashboard.Model
	clusterList  *clusters.Model
	serviceList  *services.Model
	width        int
	height       int
	ready        bool
	quitting     bool
	keyMap       keys.KeyMap
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

	return &App{
		endpoint:    endpoint,
		currentView: ViewDashboard,
		dashboard:   dashboardModel,
		clusterList: clusterListModel,
		serviceList: serviceListModel,
		keyMap:      keys.DefaultKeyMap(),
	}, nil
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
		
	case tea.KeyMsg:
		switch {
		case keys.Matches(msg, a.keyMap.Quit):
			a.quitting = true
			return a, tea.Quit
			
		case keys.Matches(msg, a.keyMap.Help):
			// Toggle help view
			if a.currentView == ViewHelp {
				a.currentView = ViewDashboard
			} else {
				a.currentView = ViewHelp
			}
			
		case keys.Matches(msg, a.keyMap.Dashboard):
			a.currentView = ViewDashboard
			
		case keys.Matches(msg, a.keyMap.Clusters):
			a.currentView = ViewClusters
			
		case keys.Matches(msg, a.keyMap.Services):
			a.currentView = ViewServices
			
		case keys.Matches(msg, a.keyMap.Tasks):
			a.currentView = ViewTasks
			
		case keys.Matches(msg, a.keyMap.TaskDefs):
			a.currentView = ViewTaskDefs
		}
	}

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
		
	// TODO: Handle other views
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
		
	case ViewHelp:
		content = a.renderHelp()
		
	default:
		content = "\n  View not implemented yet"
	}
	
	// Render footer
	footer := a.renderFooter()
	
	return header + "\n" + content + "\n" + footer
}

func (a *App) renderHeader() string {
	title := fmt.Sprintf("KECS TUI - %s", a.endpoint)
	return styles.Header.Render(title)
}

func (a *App) renderFooter() string {
	var items []string
	
	items = append(items, "1:Dashboard")
	items = append(items, "2:Clusters")
	items = append(items, "3:Services")
	items = append(items, "4:Tasks")
	items = append(items, "5:TaskDefs")
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