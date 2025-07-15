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
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/components/filter"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/components/search"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/keys"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/views/common"
)

// Model represents the task definition list view model
type Model struct {
	client          *api.Client
	table           table.Model
	taskDefs        []api.TaskDefinition
	filtered        []api.TaskDefinition
	familyPrefix    string
	width           int
	height          int
	loading         bool
	err             error
	keyMap          keys.KeyMap
	selectedARN     string
	showDetails     bool
	searchModel     search.Model
	filterModel     filter.Model
	showSearch      bool
	showFilter      bool
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

	// Create filter options
	filterOptions := []filter.Option{
		{Label: "ACTIVE", Value: "ACTIVE"},
		{Label: "INACTIVE", Value: "INACTIVE"},
	}

	return &Model{
		client:      client,
		table:       t,
		loading:     true,
		keyMap:      keys.DefaultKeyMap(),
		searchModel: search.New("Search task definitions by family or revision..."),
		filterModel: filter.New("Filter by Status", filterOptions),
		filtered:    []api.TaskDefinition{},
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

	// Handle search updates
	if m.showSearch {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if keys.Matches(msg, m.keyMap.Back) {
				m.showSearch = false
				m.searchModel.SetActive(false)
				m.applyFilters()
				return m, nil
			}
		}
		m.searchModel, cmd = m.searchModel.Update(msg)
		m.applyFilters()
		return m, cmd
	}

	// Handle filter updates
	if m.showFilter {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if keys.Matches(msg, m.keyMap.Back) {
				m.showFilter = false
				m.filterModel.SetActive(false)
				m.applyFilters()
				return m, nil
			}
		}
		m.filterModel, cmd = m.filterModel.Update(msg)
		m.applyFilters()
		return m, cmd
	}

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
				if len(m.filtered) > 0 && m.table.SelectedRow() != nil {
					m.selectedARN = m.filtered[m.table.Cursor()].TaskDefinitionArn
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
				
			case keys.Matches(msg, m.keyMap.Search):
				m.showSearch = true
				m.searchModel.SetActive(true)
				return m, nil
				
			case keys.Matches(msg, m.keyMap.Filter):
				m.showFilter = true
				m.filterModel.SetActive(true)
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
			m.applyFilters()
			m.err = nil
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Use SetSize method to handle size changes properly
		m.SetSize(msg.Width, msg.Height)
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
	if m.showSearch {
		return m.renderWithSearch()
	}

	if m.showFilter {
		return m.renderWithFilter()
	}

	if m.loading && len(m.taskDefs) == 0 {
		return m.renderFullScreen("Loading task definitions...")
	}

	if m.err != nil {
		return m.renderFullScreen(styles.Error.Render("Error: " + m.err.Error()))
	}

	if m.showDetails {
		return m.renderDetails()
	}

	return m.renderList()
}

// SetSize sets the size of the view
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Use common layout helper for consistent table height calculation
	tableHeight := common.CalculateTableHeight(height, true, true)
	m.table.SetHeight(tableHeight)
	
	// Update table columns based on available width
	m.updateTableColumns()
}

// updateTableColumns updates table column widths based on available space
func (m *Model) updateTableColumns() {
	if m.width == 0 {
		return // No width set yet
	}
	
	// Calculate available width (account for borders and padding)
	availableWidth := m.width - 4
	
	// Define minimum column widths and distribution weights
	minWidths := []int{20, 5, 10, 12, 6, 6, 20} // Family, Rev, Status, Compatibility, CPU, Memory, Containers
	distribution := []int{25, 5, 10, 15, 5, 5, 35} // Distribution weights for extra space
	
	// Use common layout helper for consistent column width distribution
	widths := common.DistributeColumnWidths(availableWidth, minWidths, distribution)
	
	// Create table columns with calculated widths
	titles := []string{"Family", "Rev", "Status", "Compatibility", "CPU", "Memory", "Containers"}
	columns := common.CreateTableColumns(titles, widths)
	m.table.SetColumns(columns)
}

// updateTable updates the table with current task definitions
func (m *Model) updateTable() {
	rows := []table.Row{}
	for _, td := range m.filtered {
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
		return m.renderFullScreen("No task definition selected")
	}

	// Find the selected task definition
	var taskDef *api.TaskDefinition
	for i := range m.filtered {
		if m.filtered[i].TaskDefinitionArn == m.selectedARN {
			taskDef = &m.filtered[i]
			break
		}
	}

	if taskDef == nil {
		return m.renderFullScreen("Task definition not found")
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

	// Use common layout helper for consistent detail view rendering
	return common.RenderListView(m.width, m.height, content.String())
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

// applyFilters applies search and filter to the task definitions
func (m *Model) applyFilters() {
	// Start with all task definitions
	filtered := m.taskDefs

	// Apply search filter
	if m.searchModel.Value() != "" {
		filtered = search.Filter(filtered, m.searchModel.Value(), func(td api.TaskDefinition) []string {
			return []string{td.Family, td.TaskDefinitionArn, fmt.Sprintf("%s:%d", td.Family, td.Revision)}
		})
	}

	// Apply status filter
	selectedStatuses := m.filterModel.SelectedValues()
	if len(selectedStatuses) > 0 {
		filtered = filter.Apply(filtered, selectedStatuses, func(td api.TaskDefinition, values []string) bool {
			for _, status := range values {
				if td.Status == status {
					return true
				}
			}
			return false
		})
	}

	m.filtered = filtered
	m.updateTable()
}

// renderWithSearch renders the view with search overlay
func (m *Model) renderWithSearch() string {
	// Render the main list
	mainContent := m.renderList()
	
	// Add search overlay at the bottom
	return lipgloss.JoinVertical(
		lipgloss.Top,
		mainContent,
		"\n\n",
		m.searchModel.View(),
	)
}

// renderWithFilter renders the view with filter overlay
func (m *Model) renderWithFilter() string {
	// Render the main list dimmed
	mainContent := m.renderList()
	dimmed := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(mainContent)
	
	// Overlay filter in center
	filterView := m.filterModel.View()
	
	// Simple overlay by joining vertically with some spacing
	var content strings.Builder
	content.WriteString(dimmed)
	content.WriteString("\n\n")
	content.WriteString(filterView)
	
	return content.String()
}

// renderList renders the main task definition list
func (m *Model) renderList() string {
	var content strings.Builder
	content.WriteString(styles.ListTitle.Render("Task Definitions"))
	
	// Show family filter if any
	if m.familyPrefix != "" {
		content.WriteString(fmt.Sprintf(" - Family: %s", styles.Info.Render(m.familyPrefix)))
	}
	
	// Show active search/filter
	if m.searchModel.Value() != "" || len(m.filterModel.SelectedValues()) > 0 {
		content.WriteString(" ")
		if m.searchModel.Value() != "" {
			content.WriteString(styles.Info.Render(fmt.Sprintf("[Search: %s]", m.searchModel.Value())))
		}
		if len(m.filterModel.SelectedValues()) > 0 {
			content.WriteString(styles.Info.Render(fmt.Sprintf("[Filter: %s]", strings.Join(m.filterModel.SelectedValues(), ", "))))
		}
	}
	
	content.WriteString("\n\n")

	if len(m.filtered) == 0 {
		if len(m.taskDefs) == 0 {
			content.WriteString(styles.Info.Render("No task definitions found. Press 'n' to create one."))
		} else {
			content.WriteString(styles.Info.Render("No task definitions match the current filters."))
		}
	} else {
		content.WriteString(m.table.View())
		content.WriteString("\n\n")
		content.WriteString(styles.Info.Render(fmt.Sprintf("Showing %d of %d task definitions", len(m.filtered), len(m.taskDefs))))
	}

	// Use common layout helper for consistent list view rendering
	return common.RenderListView(m.width, m.height, content.String())
}

// renderFullScreen renders content centered in the full available space
func (m *Model) renderFullScreen(content string) string {
	return common.RenderFullScreen(m.width, m.height, content)
}