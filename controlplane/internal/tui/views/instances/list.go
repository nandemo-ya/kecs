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
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/keys"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// Model represents the instances list view
type Model struct {
	endpoint        string
	currentInstance string
	list            list.Model
	instances       []api.Instance
	instanceManager *api.InstanceManager
	keys            keys.KeyMap
	width           int
	height          int
	loading         bool
	err             error
	lastUpdate      time.Time
	selectedIndex   int
	showCreateForm  bool
	createForm      *CreateFormModel
	showConfirmStop bool
	showConfirmDestroy bool
	confirmDeleteData bool
}

// instanceItem implements list.Item interface
type instanceItem struct {
	instance api.Instance
	current  bool
}

func (i instanceItem) Title() string {
	title := i.instance.Name
	if i.current {
		title = "▶ " + title
	} else {
		title = "  " + title
	}
	return title
}

func (i instanceItem) Description() string {
	var parts []string
	
	// Status with color
	var statusStr string
	switch i.instance.Status {
	case api.InstanceRunning:
		statusStr = styles.StatusRunning.Render("● Running")
	case api.InstanceStopped:
		statusStr = styles.StatusPending.Render("○ Stopped")
	default:
		statusStr = styles.StatusFailed.Render("✗ Not Found")
	}
	parts = append(parts, statusStr)
	
	// Data presence
	if i.instance.DataExists {
		parts = append(parts, "✓ Has data")
	} else {
		parts = append(parts, "✗ No data")
	}
	
	// Ports if running
	if i.instance.Status == api.InstanceRunning {
		parts = append(parts, fmt.Sprintf("API:%d Admin:%d", i.instance.APIPort, i.instance.AdminPort))
	}
	
	return strings.Join(parts, " | ")
}

func (i instanceItem) FilterValue() string {
	return i.instance.Name
}

// New creates a new instances list model
func New(endpoint string) (*Model, error) {
	manager, err := api.NewInstanceManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create instance manager: %w", err)
	}
	
	// Create list with custom styles
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "KECS Instances"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = styles.TitleStyle
	l.KeyMap.Quit = key.NewBinding() // Disable default quit key
	
	// Customize item styles
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(styles.PrimaryColor)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(styles.SecondaryColor)
	l.SetDelegate(delegate)
	
	return &Model{
		endpoint:        endpoint,
		list:           l,
		instanceManager: manager,
		keys:           keys.DefaultKeyMap(),
		loading:        true,
		createForm:     NewCreateForm(),
	}, nil
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return m.loadInstances()
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-2)
		
	case tea.KeyMsg:
		// Handle confirmation dialogs first
		if m.showConfirmStop || m.showConfirmDestroy {
			switch msg.String() {
			case "y", "Y":
				if m.showConfirmStop {
					m.showConfirmStop = false
					selected := m.getSelectedInstance()
					if selected != nil {
						return m, m.stopInstance(selected.Name)
					}
				} else if m.showConfirmDestroy {
					m.showConfirmDestroy = false
					selected := m.getSelectedInstance()
					if selected != nil {
						return m, m.destroyInstance(selected.Name, m.confirmDeleteData)
					}
				}
			case "n", "N", "esc":
				m.showConfirmStop = false
				m.showConfirmDestroy = false
				m.confirmDeleteData = false
			case "d", "D":
				if m.showConfirmDestroy {
					m.confirmDeleteData = !m.confirmDeleteData
				}
			}
			return m, nil
		}
		
		// Handle create form
		if m.showCreateForm {
			if msg.String() == "esc" {
				m.showCreateForm = false
				m.createForm.Reset()
				return m, nil
			}
		}
		
		// Normal key handling
		switch {
		case key.Matches(msg, m.keys.Refresh):
			return m, m.loadInstances()
			
		case msg.String() == "n":
			m.showCreateForm = true
			m.createForm.Reset()
			
		case msg.String() == "enter":
			selected := m.getSelectedInstance()
			if selected != nil {
				if selected.Status == api.InstanceStopped {
					return m, m.startInstance(selected.Name)
				} else if selected.Status == api.InstanceRunning && selected.Name != m.currentInstance {
					// Switch to this instance
					return m, m.switchToInstance(selected.Name)
				}
			}
			
		case msg.String() == "s":
			selected := m.getSelectedInstance()
			if selected != nil && selected.Status == api.InstanceRunning {
				if selected.Name == m.currentInstance {
					m.showConfirmStop = true
				} else {
					return m, m.stopInstance(selected.Name)
				}
			}
			
		case msg.String() == "d":
			selected := m.getSelectedInstance()
			if selected != nil {
				if selected.Status == api.InstanceRunning && selected.Name == m.currentInstance {
					// Cannot destroy current running instance
					m.err = fmt.Errorf("cannot destroy current running instance")
				} else {
					m.showConfirmDestroy = true
				}
			}
		}
		
	case instancesLoadedMsg:
		m.instances = msg.instances
		m.loading = false
		m.err = msg.err
		m.lastUpdate = time.Now()
		m.updateList()
		
		// Set up auto-refresh
		return m, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return refreshMsg{}
		})
		
	case refreshMsg:
		return m, m.loadInstances()
		
	case instanceOperationMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Reload instances after successful operation
			return m, m.loadInstances()
		}
		
	case switchInstanceMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Update current instance and reload
			m.currentInstance = msg.instanceName
			return m, m.loadInstances()
		}
		
	case instanceCreatedMsg:
		m.showCreateForm = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Reload instances after successful creation
			return m, m.loadInstances()
		}
	}

	// Update create form if shown
	if m.showCreateForm {
		var formCmd tea.Cmd
		m.createForm, formCmd = m.createForm.Update(msg)
		cmds = append(cmds, formCmd)
	} else if !m.showConfirmStop && !m.showConfirmDestroy {
		// Update list
		newList, cmd := m.list.Update(msg)
		m.list = newList
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var content string
	
	// Show confirmation dialogs
	if m.showConfirmStop {
		selected := m.getSelectedInstance()
		content = m.renderConfirmDialog(
			fmt.Sprintf("Stop instance '%s'?", selected.Name),
			"This will stop the running instance. Data will be preserved.",
			[]string{"[y] Yes", "[n] No"},
		)
	} else if m.showConfirmDestroy {
		selected := m.getSelectedInstance()
		options := []string{"[y] Yes", "[n] No"}
		if m.confirmDeleteData {
			options = append(options, "[d] ✓ Delete data")
		} else {
			options = append(options, "[d] ✗ Delete data")
		}
		content = m.renderConfirmDialog(
			fmt.Sprintf("Destroy instance '%s'?", selected.Name),
			"This action cannot be undone.",
			options,
		)
	} else if m.showCreateForm {
		content = m.renderCreateForm()
	} else {
		// Normal list view
		content = m.list.View()
		
		// Add instance details panel
		if selected := m.getSelectedInstance(); selected != nil {
			detailsHeight := 10
			listHeight := m.height - detailsHeight - 3
			m.list.SetSize(m.width, listHeight)
			
			details := m.renderInstanceDetails(selected)
			content = lipgloss.JoinVertical(
				lipgloss.Top,
				m.list.View(),
				styles.BoxStyle.Width(m.width-2).Height(detailsHeight).Render(details),
			)
		}
	}

	// Add error message if any
	if m.err != nil {
		errorView := styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err))
		content = lipgloss.JoinVertical(lipgloss.Top, errorView, content)
	}

	// Add footer
	footer := m.renderFooter()
	
	return lipgloss.JoinVertical(
		lipgloss.Top,
		content,
		footer,
	)
}

// SetSize sets the size of the view
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height-2)
}

// Helper methods

func (m *Model) loadInstances() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		instances, err := m.instanceManager.ListInstances(ctx)
		return instancesLoadedMsg{
			instances: instances,
			err:      err,
		}
	}
}

func (m *Model) updateList() {
	items := make([]list.Item, len(m.instances))
	for i, instance := range m.instances {
		items[i] = instanceItem{
			instance: instance,
			current:  instance.Name == m.currentInstance,
		}
	}
	m.list.SetItems(items)
}

func (m *Model) getSelectedInstance() *api.Instance {
	selected := m.list.SelectedItem()
	if selected == nil {
		return nil
	}
	
	item, ok := selected.(instanceItem)
	if !ok {
		return nil
	}
	
	return &item.instance
}

func (m *Model) startInstance(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.instanceManager.StartInstance(ctx, name)
		return instanceOperationMsg{err: err}
	}
}

func (m *Model) stopInstance(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.instanceManager.StopInstance(ctx, name)
		return instanceOperationMsg{err: err}
	}
}

func (m *Model) destroyInstance(name string, deleteData bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.instanceManager.DestroyInstance(ctx, name, deleteData)
		return instanceOperationMsg{err: err}
	}
}

func (m *Model) switchToInstance(name string) tea.Cmd {
	return func() tea.Msg {
		// Return a message for the parent to handle the actual switch
		return switchInstanceMsg{
			instanceName: name,
			err:         nil,
		}
	}
}

func (m *Model) renderInstanceDetails(instance *api.Instance) string {
	var details strings.Builder
	
	details.WriteString(styles.SubtitleStyle.Render("Instance Details") + "\n")
	details.WriteString(strings.Repeat("─", 50) + "\n")
	
	details.WriteString(fmt.Sprintf("Name: %s\n", instance.Name))
	
	// Status with duration
	statusStr := string(instance.Status)
	if instance.Status == api.InstanceRunning && instance.StartedAt != nil {
		duration := time.Since(*instance.StartedAt)
		statusStr = fmt.Sprintf("Running (%s)", formatDuration(duration))
	}
	details.WriteString(fmt.Sprintf("Status: %s\n", statusStr))
	
	details.WriteString(fmt.Sprintf("Data Directory: %s\n", instance.DataDir))
	
	if instance.Status == api.InstanceRunning {
		details.WriteString(fmt.Sprintf("API Endpoint: http://localhost:%d\n", instance.APIPort))
		details.WriteString(fmt.Sprintf("Admin Endpoint: http://localhost:%d\n", instance.AdminPort))
	}
	
	// Resource counts if available
	if instance.Resources.Clusters > 0 || instance.Resources.Services > 0 || instance.Resources.Tasks > 0 {
		details.WriteString(fmt.Sprintf("Resources: %d clusters, %d services, %d tasks\n",
			instance.Resources.Clusters,
			instance.Resources.Services,
			instance.Resources.Tasks))
	}
	
	return details.String()
}

func (m *Model) renderCreateForm() string {
	m.createForm.SetSize(m.width, m.height)
	return styles.BoxStyle.Width(80).Align(lipgloss.Center).Render(m.createForm.View())
}

func (m *Model) renderConfirmDialog(title, message string, options []string) string {
	var content strings.Builder
	
	content.WriteString(styles.TitleStyle.Render(title) + "\n\n")
	content.WriteString(message + "\n\n")
	content.WriteString(strings.Join(options, "  "))
	
	return styles.BoxStyle.Width(60).Align(lipgloss.Center).Render(content.String())
}

func (m *Model) renderFooter() string {
	actions := []string{
		"[n] New",
		"[Enter] Start/Switch",
		"[s] Stop",
		"[d] Destroy",
		"[r] Refresh",
		"[?] Help",
		"[q] Back",
	}
	
	return styles.FooterStyle.Render(strings.Join(actions, "  "))
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// Message types

type instancesLoadedMsg struct {
	instances []api.Instance
	err       error
}

type refreshMsg struct{}

type instanceOperationMsg struct {
	err error
}

type switchInstanceMsg struct {
	instanceName string
	err          error
}