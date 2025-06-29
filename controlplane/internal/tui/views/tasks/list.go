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

package tasks

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
)

// Model represents the task list view model
type Model struct {
	client          *api.Client
	table           table.Model
	tasks           []api.Task
	filtered        []api.Task
	clusters        []api.Cluster
	selectedCluster string
	selectedService string
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

// tasksMsg is sent when tasks are fetched
type tasksMsg struct {
	tasks []api.Task
	err   error
}

// clustersMsg is sent when clusters are fetched
type clustersMsg struct {
	clusters []api.Cluster
	err      error
}

// New creates a new task list model
func New(endpoint string) (*Model, error) {
	client := api.NewClient(endpoint)
	
	// Create table
	columns := []table.Column{
		{Title: "Task ID", Width: 20},
		{Title: "Status", Width: 12},
		{Title: "Task Definition", Width: 25},
		{Title: "Started By", Width: 20},
		{Title: "Created", Width: 15},
		{Title: "CPU", Width: 8},
		{Title: "Memory", Width: 8},
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
		{Label: "RUNNING", Value: "RUNNING"},
		{Label: "PENDING", Value: "PENDING"},
		{Label: "STOPPED", Value: "STOPPED"},
	}

	return &Model{
		client:      client,
		table:       t,
		loading:     true,
		keyMap:      keys.DefaultKeyMap(),
		searchModel: search.New("Search tasks by ID or task definition..."),
		filterModel: filter.New("Filter by Status", filterOptions),
		filtered:    []api.Task{},
	}, nil
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchClusters,
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
					m.selectedARN = m.filtered[m.table.Cursor()].TaskArn
					m.showDetails = true
				}
				return m, nil
				
			case keys.Matches(msg, m.keyMap.Refresh):
				m.loading = true
				return m, m.fetchTasks
				
			case keys.Matches(msg, m.keyMap.Delete):
				// TODO: Implement task stop
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
			m.fetchTasks,
			tick(),
		)

	case clustersMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.clusters = msg.clusters
			// If no cluster selected and we have clusters, select the first one
			if m.selectedCluster == "" && len(m.clusters) > 0 {
				m.selectedCluster = m.clusters[0].ClusterArn
			}
		}
		return m, m.fetchTasks

	case tasksMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.tasks = msg.tasks
			m.applyFilters()
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
	if m.showSearch {
		return m.renderWithSearch()
	}

	if m.showFilter {
		return m.renderWithFilter()
	}

	if m.loading && len(m.tasks) == 0 {
		return styles.Content.Render("Loading tasks...")
	}

	if m.err != nil {
		return styles.Content.Render(
			styles.Error.Render("Error: " + m.err.Error()),
		)
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
	m.table.SetHeight(height - 10)
}

// updateTable updates the table with current tasks
func (m *Model) updateTable() {
	rows := []table.Row{}
	for _, task := range m.filtered {
		// Extract task ID from ARN
		taskID := m.extractTaskID(task.TaskArn)
		
		// Get task status with color
		status := styles.GetStatusStyle(task.LastStatus).Render(task.LastStatus)
		
		// Extract task definition name
		taskDefName := m.extractTaskDefName(task.TaskDefinitionArn)
		
		// Format started by
		startedBy := task.StartedBy
		if strings.HasPrefix(startedBy, "ecs-svc/") {
			startedBy = "service:" + strings.TrimPrefix(startedBy, "ecs-svc/")
		}
		
		// Format created time
		created := ""
		if task.CreatedAt != nil {
			created = task.CreatedAt.Format("15:04:05")
		}
		
		rows = append(rows, table.Row{
			taskID,
			status,
			taskDefName,
			startedBy,
			created,
			task.Cpu,
			task.Memory,
		})
	}
	m.table.SetRows(rows)
}

// renderDetails renders the task detail view
func (m *Model) renderDetails() string {
	if m.selectedARN == "" {
		return styles.Content.Render("No task selected")
	}

	// Find the selected task
	var task *api.Task
	for i := range m.filtered {
		if m.filtered[i].TaskArn == m.selectedARN {
			task = &m.filtered[i]
			break
		}
	}

	if task == nil {
		return styles.Content.Render("Task not found")
	}

	var content strings.Builder
	content.WriteString(styles.ListTitle.Render("Task Details"))
	content.WriteString("\n\n")

	// Basic info
	content.WriteString(fmt.Sprintf("Task ID: %s\n", styles.Info.Render(m.extractTaskID(task.TaskArn))))
	content.WriteString(fmt.Sprintf("ARN: %s\n", styles.Info.Render(task.TaskArn)))
	content.WriteString(fmt.Sprintf("Status: %s\n", styles.GetStatusStyle(task.LastStatus).Render(task.LastStatus)))
	content.WriteString(fmt.Sprintf("Desired Status: %s\n", styles.Info.Render(task.DesiredStatus)))
	content.WriteString("\n")

	// Task Definition
	content.WriteString(styles.ListTitle.Render("Task Definition"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("Task Definition: %s\n", task.TaskDefinitionArn))
	if task.LaunchType != "" {
		content.WriteString(fmt.Sprintf("Launch Type: %s\n", task.LaunchType))
	}
	if task.PlatformVersion != "" {
		content.WriteString(fmt.Sprintf("Platform Version: %s\n", task.PlatformVersion))
	}
	content.WriteString("\n")

	// Resources
	content.WriteString(styles.ListTitle.Render("Resources"))
	content.WriteString("\n")
	if task.Cpu != "" {
		content.WriteString(fmt.Sprintf("CPU: %s units\n", task.Cpu))
	}
	if task.Memory != "" {
		content.WriteString(fmt.Sprintf("Memory: %s MB\n", task.Memory))
	}
	content.WriteString("\n")

	// Containers
	if len(task.Containers) > 0 {
		content.WriteString(styles.ListTitle.Render("Containers"))
		content.WriteString("\n")
		for _, container := range task.Containers {
			content.WriteString(fmt.Sprintf("- %s: %s", container.Name, 
				styles.GetStatusStyle(container.LastStatus).Render(container.LastStatus)))
			if container.ExitCode != nil {
				content.WriteString(fmt.Sprintf(" (Exit Code: %d)", *container.ExitCode))
			}
			if container.Reason != "" {
				content.WriteString(fmt.Sprintf(" - %s", container.Reason))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Timing
	content.WriteString(styles.ListTitle.Render("Timing"))
	content.WriteString("\n")
	if task.CreatedAt != nil {
		content.WriteString(fmt.Sprintf("Created: %s\n", task.CreatedAt.Format(time.RFC3339)))
	}
	if task.StartedAt != nil {
		content.WriteString(fmt.Sprintf("Started: %s\n", task.StartedAt.Format(time.RFC3339)))
	}
	if task.StoppedAt != nil {
		content.WriteString(fmt.Sprintf("Stopped: %s\n", task.StoppedAt.Format(time.RFC3339)))
		if task.StoppedReason != "" {
			content.WriteString(fmt.Sprintf("Stop Reason: %s\n", task.StoppedReason))
		}
	}
	content.WriteString("\n")

	// Metadata
	if task.StartedBy != "" || task.Group != "" {
		content.WriteString(styles.ListTitle.Render("Metadata"))
		content.WriteString("\n")
		if task.StartedBy != "" {
			content.WriteString(fmt.Sprintf("Started By: %s\n", task.StartedBy))
		}
		if task.Group != "" {
			content.WriteString(fmt.Sprintf("Group: %s\n", task.Group))
		}
		content.WriteString("\n")
	}

	// Tags
	if len(task.Tags) > 0 {
		content.WriteString(styles.ListTitle.Render("Tags"))
		content.WriteString("\n")
		for _, tag := range task.Tags {
			content.WriteString(fmt.Sprintf("%s: %s\n", tag.Key, tag.Value))
		}
		content.WriteString("\n")
	}

	content.WriteString(styles.Info.Render("Press ESC to go back"))

	return styles.Content.Render(content.String())
}

// fetchClusters fetches the list of clusters
func (m *Model) fetchClusters() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listResp, err := m.client.ListClusters(ctx)
	if err != nil {
		return clustersMsg{err: err}
	}

	if len(listResp.ClusterArns) == 0 {
		return clustersMsg{clusters: []api.Cluster{}}
	}

	descResp, err := m.client.DescribeClusters(ctx, listResp.ClusterArns)
	if err != nil {
		return clustersMsg{err: err}
	}

	return clustersMsg{clusters: descResp.Clusters}
}

// fetchTasks fetches the list of tasks
func (m *Model) fetchTasks() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	allTasks := []api.Task{}

	// If we have clusters, fetch tasks for each cluster
	if len(m.clusters) > 0 {
		for _, cluster := range m.clusters {
			// Skip if we have a selected cluster and this isn't it
			if m.selectedCluster != "" && cluster.ClusterArn != m.selectedCluster {
				continue
			}

			listResp, err := m.client.ListTasks(ctx, cluster.ClusterArn, m.selectedService)
			if err != nil {
				continue // Skip this cluster on error
			}

			if len(listResp.TaskArns) > 0 {
				descResp, err := m.client.DescribeTasks(ctx, cluster.ClusterArn, listResp.TaskArns)
				if err == nil {
					allTasks = append(allTasks, descResp.Tasks...)
				}
			}
		}
	}

	return tasksMsg{tasks: allTasks}
}

// getClusterName extracts the cluster name from ARN
func (m *Model) getClusterName(arn string) string {
	// First check if we have the cluster in our list
	for _, cluster := range m.clusters {
		if cluster.ClusterArn == arn {
			return cluster.ClusterName
		}
	}
	
	// Fallback to extracting from ARN
	parts := strings.Split(arn, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return arn
}

// extractTaskID extracts the task ID from ARN
func (m *Model) extractTaskID(arn string) string {
	// Task ARN format: arn:aws:ecs:region:account:task/cluster-name/task-id
	parts := strings.Split(arn, "/")
	if len(parts) > 0 {
		id := parts[len(parts)-1]
		// Truncate long UUIDs for display
		if len(id) > 20 {
			return id[:8] + "..." + id[len(id)-8:]
		}
		return id
	}
	return arn
}

// extractTaskDefName extracts the task definition name from ARN
func (m *Model) extractTaskDefName(arn string) string {
	// Task definition ARN format: arn:aws:ecs:region:account:task-definition/name:revision
	parts := strings.Split(arn, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return arn
}

// tick returns a command that sends a tick message after a delay
func tick() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// applyFilters applies search and filter to the tasks
func (m *Model) applyFilters() {
	// Start with all tasks
	filtered := m.tasks

	// Apply search filter
	if m.searchModel.Value() != "" {
		filtered = search.Filter(filtered, m.searchModel.Value(), func(t api.Task) []string {
			return []string{t.TaskArn, t.TaskDefinitionArn, m.extractTaskID(t.TaskArn), m.extractTaskDefName(t.TaskDefinitionArn)}
		})
	}

	// Apply status filter
	selectedStatuses := m.filterModel.SelectedValues()
	if len(selectedStatuses) > 0 {
		filtered = filter.Apply(filtered, selectedStatuses, func(t api.Task, values []string) bool {
			for _, status := range values {
				if t.LastStatus == status {
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

// renderList renders the main task list
func (m *Model) renderList() string {
	var content strings.Builder
	content.WriteString(styles.ListTitle.Render("Tasks"))
	
	// Show current filters if any
	if m.selectedCluster != "" {
		clusterName := m.getClusterName(m.selectedCluster)
		content.WriteString(fmt.Sprintf(" - Cluster: %s", styles.Info.Render(clusterName)))
	}
	if m.selectedService != "" {
		content.WriteString(fmt.Sprintf(", Service: %s", styles.Info.Render(m.selectedService)))
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
		if len(m.tasks) == 0 {
			content.WriteString(styles.Info.Render("No tasks found."))
		} else {
			content.WriteString(styles.Info.Render("No tasks match the current filters."))
		}
	} else {
		content.WriteString(m.table.View())
		content.WriteString("\n\n")
		content.WriteString(styles.Info.Render(fmt.Sprintf("Showing %d of %d tasks", len(m.filtered), len(m.tasks))))
	}

	return styles.Content.Render(content.String())
}