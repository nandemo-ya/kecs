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

package instances

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// QuickSwitchModel represents the quick instance switch dialog
type QuickSwitchModel struct {
	instances       []api.Instance
	instanceManager *api.InstanceManager
	list            list.Model
	currentInstance string
	width           int
	height          int
	loading         bool
	err             error
	visible         bool
	selected        bool
}

// NewQuickSwitch creates a new quick switch dialog
func NewQuickSwitch(currentInstance string) (*QuickSwitchModel, error) {
	manager, err := api.NewInstanceManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create instance manager: %w", err)
	}
	
	// Create list with custom styles
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Switch Instance"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = styles.TitleStyle
	
	return &QuickSwitchModel{
		instanceManager: manager,
		list:            l,
		currentInstance: currentInstance,
		loading:         true,
	}, nil
}

// Init initializes the model
func (m *QuickSwitchModel) Init() tea.Cmd {
	return m.loadInstances()
}

// Update handles messages
func (m *QuickSwitchModel) Update(msg tea.Msg) (*QuickSwitchModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.visible = false
			m.selected = false
			return m, nil
			
		case "enter":
			if selected := m.list.SelectedItem(); selected != nil {
				if item, ok := selected.(instanceItem); ok {
					m.selected = true
					m.visible = false
					return m, m.switchToInstance(item.instance.Name)
				}
			}
		}
		
	case instancesLoadedMsg:
		m.instances = msg.instances
		m.loading = false
		m.err = msg.err
		m.updateList()
		
	case switchInstanceMsg:
		// This will be handled by the parent
		return m, nil
	}
	
	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	
	return m, cmd
}

// View renders the dialog
func (m *QuickSwitchModel) View() string {
	if !m.visible {
		return ""
	}
	
	if m.loading {
		return m.renderDialog("Loading instances...")
	}
	
	if m.err != nil {
		return m.renderDialog(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}
	
	// Set dialog size
	dialogWidth := 60
	dialogHeight := 15
	if dialogHeight > m.height-4 {
		dialogHeight = m.height - 4
	}
	
	m.list.SetSize(dialogWidth-4, dialogHeight-4)
	
	content := m.list.View()
	footer := styles.SubtleStyle.Render("[Enter] Select  [Esc] Cancel")
	
	dialog := lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		"",
		footer,
	)
	
	return m.renderDialog(dialog)
}

// Show makes the dialog visible
func (m *QuickSwitchModel) Show() tea.Cmd {
	m.visible = true
	m.selected = false
	return m.loadInstances()
}

// Hide hides the dialog
func (m *QuickSwitchModel) Hide() {
	m.visible = false
}

// IsVisible returns whether the dialog is visible
func (m *QuickSwitchModel) IsVisible() bool {
	return m.visible
}

// SetSize updates the dialog size
func (m *QuickSwitchModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// GetSelectedInstance returns the selected instance name if any
func (m *QuickSwitchModel) GetSelectedInstance() (string, bool) {
	if !m.selected {
		return "", false
	}
	
	if selected := m.list.SelectedItem(); selected != nil {
		if item, ok := selected.(instanceItem); ok {
			return item.instance.Name, true
		}
	}
	
	return "", false
}

// Helper methods

func (m *QuickSwitchModel) loadInstances() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		instances, err := m.instanceManager.ListInstances(ctx)
		
		// Filter out only running instances
		runningInstances := make([]api.Instance, 0, len(instances))
		for _, instance := range instances {
			if instance.Status == api.InstanceRunning {
				runningInstances = append(runningInstances, instance)
			}
		}
		
		return instancesLoadedMsg{
			instances: runningInstances,
			err:       err,
		}
	}
}

func (m *QuickSwitchModel) updateList() {
	items := make([]list.Item, len(m.instances))
	for i, instance := range m.instances {
		items[i] = instanceItem{
			instance: instance,
			current:  instance.Name == m.currentInstance,
		}
	}
	m.list.SetItems(items)
}

func (m *QuickSwitchModel) switchToInstance(name string) tea.Cmd {
	return func() tea.Msg {
		// Return a message that the parent can handle
		return switchInstanceMsg{
			instanceName: name,
			err:         nil,
		}
	}
}

func (m *QuickSwitchModel) renderDialog(content string) string {
	// Create overlay effect
	overlay := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("240"))
	
	// Create dialog box
	dialog := styles.BoxStyle.
		Width(60).
		Padding(1).
		Align(lipgloss.Center).
		Render(content)
	
	// Center the dialog
	centered := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
	
	// Layer dialog over dimmed background
	dimBg := strings.Repeat("\n", m.height)
	return overlay.Render(dimBg) + "\n" + centered
}