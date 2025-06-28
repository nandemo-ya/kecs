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

package taskdefs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/keys"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// Model represents the task definition list view model
type Model struct {
	client          *api.Client
	table           table.Model
	taskDefs        []api.TaskDefinition
	familyPrefix    string
	width           int
	height          int
	loading         bool
	err             error
	keyMap          keys.KeyMap
	selectedARN     string
	showDetails     bool
}

// tickMsg is sent when the refresh timer ticks
type tickMsg time.Time

// taskDefsMsg is sent when task definitions are fetched
type taskDefsMsg struct {
	taskDefs []api.TaskDefinition
	err      error
}

// New creates a new task definition list model
func New(endpoint string) (*Model, error) {
	client := api.NewClient(endpoint)
	
	// Create table
	columns := []table.Column{
		{Title: "Family", Width: 25},
		{Title: "Rev", Width: 5},
		{Title: "Status", Width: 10},
		{Title: "Compatibility", Width: 15},
		{Title: "CPU", Width: 8},
		{Title: "Memory", Width: 8},
		{Title: "Containers", Width: 30},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return &Model{
		client:  client,
		table:   t,
		loading: true,
		keyMap:  keys.DefaultKeyMap(),
	}, nil
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchTaskDefs,
		tick(),
	)
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showDetails {
			switch {
			case keys.Matches(msg, m.keyMap.Back):
				m.showDetails = false
				return m, nil
			}
		} else {
			switch {
			case keys.Matches(msg, m.keyMap.Select):
				if len(m.taskDefs) > 0 && m.table.SelectedRow() != nil {
					m.selectedARN = m.taskDefs[m.table.Cursor()].TaskDefinitionArn
					m.showDetails = true
				}
				return m, nil
				
			case keys.Matches(msg, m.keyMap.Refresh):
				m.loading = true
				return m, m.fetchTaskDefs
				
			case keys.Matches(msg, m.keyMap.Create):
				// TODO: Implement task definition creation
				return m, nil
				
			case keys.Matches(msg, m.keyMap.Delete):
				// TODO: Implement task definition deregistration
				return m, nil
			}
		}

	case tickMsg:
		return m, tea.Batch(
			m.fetchTaskDefs,
			tick(),
		)

	case taskDefsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.taskDefs = msg.taskDefs
			m.updateTable()
			m.err = nil
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(msg.Height - 10)
		return m, nil
	}

	// Update table
	if !m.showDetails {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *Model) View() string {
	if m.loading && len(m.taskDefs) == 0 {
		return styles.Content.Render("Loading task definitions...")
	}

	if m.err != nil {
		return styles.Content.Render(
			styles.Error.Render("Error: " + m.err.Error()),
		)
	}

	if m.showDetails {
		return m.renderDetails()
	}

	var content strings.Builder
	content.WriteString(styles.ListTitle.Render("Task Definitions"))
	
	// Show family filter if any
	if m.familyPrefix != "" {
		content.WriteString(fmt.Sprintf(" - Family: %s", styles.Info.Render(m.familyPrefix)))
	}
	
	content.WriteString("\n\n")

	if len(m.taskDefs) == 0 {
		content.WriteString(styles.Info.Render("No task definitions found. Press 'n' to create one."))
	} else {
		content.WriteString(m.table.View())
		content.WriteString("\n\n")
		content.WriteString(styles.Info.Render(fmt.Sprintf("Showing %d task definitions", len(m.taskDefs))))
	}

	return styles.Content.Render(content.String())
}

// SetSize sets the size of the view
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetHeight(height - 10)
}

// updateTable updates the table with current task definitions
func (m *Model) updateTable() {
	rows := []table.Row{}
	for _, td := range m.taskDefs {
		// Get status with color
		status := styles.GetStatusStyle(td.Status).Render(td.Status)
		
		// Get compatibility
		compatibility := strings.Join(td.RequiresCompatibilities, ",")
		if compatibility == "" {
			compatibility = "EC2"
		}
		
		// Get container names
		var containerNames []string
		for _, container := range td.ContainerDefinitions {
			containerNames = append(containerNames, container.Name)
		}
		containers := strings.Join(containerNames, ", ")
		if len(containers) > 30 {
			containers = containers[:27] + "..."
		}
		
		rows = append(rows, table.Row{
			td.Family,
			fmt.Sprintf("%d", td.Revision),
			status,
			compatibility,
			td.Cpu,
			td.Memory,
			containers,
		})
	}
	m.table.SetRows(rows)
}

// renderDetails renders the task definition detail view
func (m *Model) renderDetails() string {
	if m.selectedARN == "" {
		return styles.Content.Render("No task definition selected")
	}

	// Find the selected task definition
	var taskDef *api.TaskDefinition
	for i := range m.taskDefs {
		if m.taskDefs[i].TaskDefinitionArn == m.selectedARN {
			taskDef = &m.taskDefs[i]
			break
		}
	}

	if taskDef == nil {
		return styles.Content.Render("Task definition not found")
	}

	var content strings.Builder
	content.WriteString(styles.ListTitle.Render("Task Definition Details"))
	content.WriteString("\n\n")

	// Basic info
	content.WriteString(fmt.Sprintf("Family: %s\n", styles.Info.Render(taskDef.Family)))
	content.WriteString(fmt.Sprintf("Revision: %s\n", styles.Info.Render(fmt.Sprintf("%d", taskDef.Revision))))
	content.WriteString(fmt.Sprintf("ARN: %s\n", styles.Info.Render(taskDef.TaskDefinitionArn)))
	content.WriteString(fmt.Sprintf("Status: %s\n", styles.GetStatusStyle(taskDef.Status).Render(taskDef.Status)))
	content.WriteString("\n")

	// Resources
	content.WriteString(styles.ListTitle.Render("Resources"))
	content.WriteString("\n")
	if taskDef.Cpu != "" {
		content.WriteString(fmt.Sprintf("CPU: %s units\n", taskDef.Cpu))
	}
	if taskDef.Memory != "" {
		content.WriteString(fmt.Sprintf("Memory: %s MB\n", taskDef.Memory))
	}
	if taskDef.NetworkMode != "" {
		content.WriteString(fmt.Sprintf("Network Mode: %s\n", taskDef.NetworkMode))
	}
	if len(taskDef.RequiresCompatibilities) > 0 {
		content.WriteString(fmt.Sprintf("Compatibility: %s\n", strings.Join(taskDef.RequiresCompatibilities, ", ")))
	}
	content.WriteString("\n")

	// Roles
	if taskDef.TaskRoleArn != "" || taskDef.ExecutionRoleArn != "" {
		content.WriteString(styles.ListTitle.Render("IAM Roles"))
		content.WriteString("\n")
		if taskDef.TaskRoleArn != "" {
			content.WriteString(fmt.Sprintf("Task Role: %s\n", taskDef.TaskRoleArn))
		}
		if taskDef.ExecutionRoleArn != "" {
			content.WriteString(fmt.Sprintf("Execution Role: %s\n", taskDef.ExecutionRoleArn))
		}
		content.WriteString("\n")
	}

	// Containers
	content.WriteString(styles.ListTitle.Render("Containers"))
	content.WriteString("\n")
	for i, container := range taskDef.ContainerDefinitions {
		if i > 0 {
			content.WriteString("\n")
		}
		content.WriteString(fmt.Sprintf("- %s:\n", styles.Info.Render(container.Name)))
		content.WriteString(fmt.Sprintf("  Image: %s\n", container.Image))
		if container.Cpu > 0 {
			content.WriteString(fmt.Sprintf("  CPU: %d\n", container.Cpu))
		}
		if container.Memory > 0 {
			content.WriteString(fmt.Sprintf("  Memory: %d MB\n", container.Memory))
		}
		if container.MemoryReservation > 0 {
			content.WriteString(fmt.Sprintf("  Memory Reservation: %d MB\n", container.MemoryReservation))
		}
		content.WriteString(fmt.Sprintf("  Essential: %v\n", container.Essential))
		
		// Port mappings
		if len(container.PortMappings) > 0 {
			content.WriteString("  Port Mappings:\n")
			for _, pm := range container.PortMappings {
				protocol := pm.Protocol
				if protocol == "" {
					protocol = "tcp"
				}
				content.WriteString(fmt.Sprintf("    - %d:%d/%s\n", pm.HostPort, pm.ContainerPort, protocol))
			}
		}
		
		// Environment variables
		if len(container.Environment) > 0 {
			content.WriteString("  Environment:\n")
			for _, env := range container.Environment {
				content.WriteString(fmt.Sprintf("    - %s=%s\n", env.Name, env.Value))
			}
		}
	}
	content.WriteString("\n")

	// Metadata
	if taskDef.RegisteredAt != nil || taskDef.RegisteredBy != "" {
		content.WriteString(styles.ListTitle.Render("Metadata"))
		content.WriteString("\n")
		if taskDef.RegisteredAt != nil {
			content.WriteString(fmt.Sprintf("Registered At: %s\n", taskDef.RegisteredAt.Format(time.RFC3339)))
		}
		if taskDef.RegisteredBy != "" {
			content.WriteString(fmt.Sprintf("Registered By: %s\n", taskDef.RegisteredBy))
		}
		if taskDef.DeregisteredAt != nil {
			content.WriteString(fmt.Sprintf("Deregistered At: %s\n", taskDef.DeregisteredAt.Format(time.RFC3339)))
		}
		content.WriteString("\n")
	}

	// Tags
	if len(taskDef.Tags) > 0 {
		content.WriteString(styles.ListTitle.Render("Tags"))
		content.WriteString("\n")
		for _, tag := range taskDef.Tags {
			content.WriteString(fmt.Sprintf("%s: %s\n", tag.Key, tag.Value))
		}
		content.WriteString("\n")
	}

	content.WriteString(styles.Info.Render("Press ESC to go back"))

	return styles.Content.Render(content.String())
}

// fetchTaskDefs fetches the list of task definitions
func (m *Model) fetchTaskDefs() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get list of task definition ARNs
	listResp, err := m.client.ListTaskDefinitions(ctx, m.familyPrefix)
	if err != nil {
		return taskDefsMsg{err: err}
	}

	if len(listResp.TaskDefinitionArns) == 0 {
		return taskDefsMsg{taskDefs: []api.TaskDefinition{}}
	}

	// Describe each task definition
	var taskDefs []api.TaskDefinition
	for _, arn := range listResp.TaskDefinitionArns {
		descResp, err := m.client.DescribeTaskDefinition(ctx, arn)
		if err == nil {
			taskDefs = append(taskDefs, descResp.TaskDefinition)
		}
	}

	return taskDefsMsg{taskDefs: taskDefs}
}

// tick returns a command that sends a tick message after a delay
func tick() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}