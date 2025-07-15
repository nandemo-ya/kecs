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

package clusters

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

// Model represents the cluster list view model
type Model struct {
	client       *api.Client
	table        table.Model
	clusters     []api.Cluster
	filtered     []api.Cluster
	width        int
	height       int
	loading      bool
	err          error
	keyMap       keys.KeyMap
	selectedARN  string
	showDetails  bool
	showCreate   bool
	createModel  CreateModel
	searchModel  search.Model
	filterModel  filter.Model
	showSearch   bool
	showFilter   bool
}

// tickMsg is sent when the refresh timer ticks
type tickMsg time.Time

// clustersMsg is sent when clusters are fetched
type clustersMsg struct {
	clusters []api.Cluster
	err      error
}

// ClusterListMsg is sent to return to the cluster list view
type ClusterListMsg struct{}

// New creates a new cluster list model
func New(endpoint string) (*Model, error) {
	client := api.NewClient(endpoint)
	
	// Create table
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Status", Width: 10},
		{Title: "Services", Width: 10},
		{Title: "Running Tasks", Width: 15},
		{Title: "Pending Tasks", Width: 15},
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
		{Label: "PROVISIONING", Value: "PROVISIONING"},
		{Label: "DEPROVISIONING", Value: "DEPROVISIONING"},
		{Label: "FAILED", Value: "FAILED"},
		{Label: "INACTIVE", Value: "INACTIVE"},
	}

	return &Model{
		client:      client,
		table:       t,
		loading:     true,
		keyMap:      keys.DefaultKeyMap(),
		createModel: NewCreateModel(client),
		searchModel: search.New("Search clusters by name..."),
		filterModel: filter.New("Filter by Status", filterOptions),
		filtered:    []api.Cluster{},
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

	// Handle create model updates
	if m.showCreate {
		switch msg := msg.(type) {
		case ClusterListMsg:
			m.showCreate = false
			m.loading = true
			return m, m.fetchClusters
		case CreatedMsg:
			m.showCreate = false
			m.loading = true
			return m, m.fetchClusters
		default:
			m.createModel, cmd = m.createModel.Update(msg)
			return m, cmd
		}
	}

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
				if len(m.clusters) > 0 && m.table.SelectedRow() != nil {
					m.selectedARN = m.clusters[m.table.Cursor()].ClusterArn
					m.showDetails = true
				}
				return m, nil
				
			case keys.Matches(msg, m.keyMap.Refresh):
				m.loading = true
				return m, m.fetchClusters
				
			case keys.Matches(msg, m.keyMap.Create):
				m.showCreate = true
				m.createModel = NewCreateModel(m.client)
				return m, m.createModel.Init()
				
			case keys.Matches(msg, m.keyMap.Delete):
				// TODO: Implement cluster deletion
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
			m.fetchClusters,
			tick(),
		)

	case clustersMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.clusters = msg.clusters
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
	if m.showCreate {
		return m.createModel.View()
	}

	if m.showSearch {
		return m.renderWithSearch()
	}

	if m.showFilter {
		return m.renderWithFilter()
	}

	if m.loading && len(m.clusters) == 0 {
		return m.renderFullScreen("Loading clusters...")
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
	
	// Update table width to use available space
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
	minWidths := []int{20, 10, 10, 12, 12} // Name, Status, Services, Running, Pending
	distribution := []int{60, 10, 10, 10, 10} // Distribution weights for extra space
	
	// Use common layout helper for consistent column width distribution
	widths := common.DistributeColumnWidths(availableWidth, minWidths, distribution)
	
	// Create table columns with calculated widths
	titles := []string{"Name", "Status", "Services", "Running Tasks", "Pending Tasks"}
	columns := common.CreateTableColumns(titles, widths)
	m.table.SetColumns(columns)
}

// updateTable updates the table with current clusters
func (m *Model) updateTable() {
	rows := []table.Row{}
	for _, cluster := range m.filtered {
		status := styles.GetStatusStyle(cluster.Status).Render(cluster.Status)
		rows = append(rows, table.Row{
			cluster.ClusterName,
			status,
			fmt.Sprintf("%d", cluster.ActiveServicesCount),
			fmt.Sprintf("%d", cluster.RunningTasksCount),
			fmt.Sprintf("%d", cluster.PendingTasksCount),
		})
	}
	m.table.SetRows(rows)
}

// renderDetails renders the cluster detail view
func (m *Model) renderDetails() string {
	if m.selectedARN == "" {
		return m.renderFullScreen("No cluster selected")
	}

	// Find the selected cluster
	var cluster *api.Cluster
	for i := range m.clusters {
		if m.clusters[i].ClusterArn == m.selectedARN {
			cluster = &m.clusters[i]
			break
		}
	}

	if cluster == nil {
		return m.renderFullScreen("Cluster not found")
	}

	var content strings.Builder
	content.WriteString(styles.ListTitle.Render("Cluster Details"))
	content.WriteString("\n\n")

	// Basic info
	content.WriteString(fmt.Sprintf("Name: %s\n", styles.Info.Render(cluster.ClusterName)))
	content.WriteString(fmt.Sprintf("ARN: %s\n", styles.Info.Render(cluster.ClusterArn)))
	content.WriteString(fmt.Sprintf("Status: %s\n", styles.GetStatusStyle(cluster.Status).Render(cluster.Status)))
	content.WriteString("\n")

	// Resource counts
	content.WriteString(styles.ListTitle.Render("Resources"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("Active Services: %d\n", cluster.ActiveServicesCount))
	content.WriteString(fmt.Sprintf("Running Tasks: %d\n", cluster.RunningTasksCount))
	content.WriteString(fmt.Sprintf("Pending Tasks: %d\n", cluster.PendingTasksCount))
	content.WriteString(fmt.Sprintf("Container Instances: %d\n", cluster.RegisteredContainerInstancesCount))
	content.WriteString("\n")

	// Tags
	if len(cluster.Tags) > 0 {
		content.WriteString(styles.ListTitle.Render("Tags"))
		content.WriteString("\n")
		for _, tag := range cluster.Tags {
			content.WriteString(fmt.Sprintf("%s: %s\n", tag.Key, tag.Value))
		}
		content.WriteString("\n")
	}

	content.WriteString(styles.Info.Render("Press ESC to go back"))

	// Use common layout helper for consistent detail view rendering
	return common.RenderListView(m.width, m.height, content.String())
}

// fetchClusters fetches the list of clusters
func (m *Model) fetchClusters() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First, get the list of cluster ARNs
	listResp, err := m.client.ListClusters(ctx)
	if err != nil {
		return clustersMsg{err: err}
	}

	if len(listResp.ClusterArns) == 0 {
		return clustersMsg{clusters: []api.Cluster{}}
	}

	// Then, describe the clusters to get full details
	descResp, err := m.client.DescribeClusters(ctx, listResp.ClusterArns)
	if err != nil {
		return clustersMsg{err: err}
	}

	return clustersMsg{clusters: descResp.Clusters}
}

// tick returns a command that sends a tick message after a delay
func tick() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// applyFilters applies search and filter to the clusters
func (m *Model) applyFilters() {
	// Start with all clusters
	filtered := m.clusters

	// Apply search filter
	if m.searchModel.Value() != "" {
		filtered = search.Filter(filtered, m.searchModel.Value(), func(c api.Cluster) []string {
			return []string{c.ClusterName, c.ClusterArn}
		})
	}

	// Apply status filter
	selectedStatuses := m.filterModel.SelectedValues()
	if len(selectedStatuses) > 0 {
		filtered = filter.Apply(filtered, selectedStatuses, func(c api.Cluster, values []string) bool {
			for _, status := range values {
				if c.Status == status {
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

// renderFullScreen renders content centered in the full available space
func (m *Model) renderFullScreen(content string) string {
	return common.RenderFullScreen(m.width, m.height, content)
}

// renderList renders the main cluster list
func (m *Model) renderList() string {
	var content strings.Builder
	content.WriteString(styles.ListTitle.Render("Clusters"))
	
	// Show active filters
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
		if len(m.clusters) == 0 {
			content.WriteString(styles.Info.Render("No clusters found. Press 'n' to create one."))
		} else {
			content.WriteString(styles.Info.Render("No clusters match the current filters."))
		}
	} else {
		content.WriteString(m.table.View())
		content.WriteString("\n\n")
		content.WriteString(styles.Info.Render(fmt.Sprintf("Showing %d of %d clusters", len(m.filtered), len(m.clusters))))
	}

	// Use common layout helper for consistent list view rendering
	return common.RenderListView(m.width, m.height, content.String())
}