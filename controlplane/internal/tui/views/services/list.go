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

package services

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

// Model represents the service list view model
type Model struct {
	client         *api.Client
	table          table.Model
	services       []api.Service
	filtered       []api.Service
	clusters       []api.Cluster
	selectedCluster string
	width          int
	height         int
	loading        bool
	err            error
	keyMap         keys.KeyMap
	selectedARN    string
	showDetails    bool
	showCreate     bool
	createModel    CreateModel
	searchModel    search.Model
	filterModel    filter.Model
	showSearch     bool
	showFilter     bool
}

// tickMsg is sent when the refresh timer ticks
type tickMsg time.Time

// servicesMsg is sent when services are fetched
type servicesMsg struct {
	services []api.Service
	err      error
}

// clustersMsg is sent when clusters are fetched
type clustersMsg struct {
	clusters []api.Cluster
	err      error
}

// ServiceListMsg is sent to return to the service list view
type ServiceListMsg struct{}

// New creates a new service list model
func New(endpoint string) (*Model, error) {
	client := api.NewClient(endpoint)
	
	// Create table
	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Status", Width: 10},
		{Title: "Desired", Width: 8},
		{Title: "Running", Width: 8},
		{Title: "Pending", Width: 8},
		{Title: "Task Definition", Width: 30},
		{Title: "Cluster", Width: 20},
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
		{Label: "DRAINING", Value: "DRAINING"},
		{Label: "RUNNING", Value: "RUNNING"},
		{Label: "PENDING", Value: "PENDING"},
		{Label: "INACTIVE", Value: "INACTIVE"},
	}

	return &Model{
		client:      client,
		table:       t,
		loading:     true,
		keyMap:      keys.DefaultKeyMap(),
		createModel: NewCreateModel(client, []api.Cluster{}),
		searchModel: search.New("Search services by name or ARN..."),
		filterModel: filter.New("Filter by Status", filterOptions),
		filtered:    []api.Service{},
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
		case ServiceListMsg:
			m.showCreate = false
			m.loading = true
			return m, m.fetchServices
		case CreatedMsg:
			m.showCreate = false
			m.loading = true
			return m, m.fetchServices
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
				if len(m.filtered) > 0 && m.table.SelectedRow() != nil {
					m.selectedARN = m.filtered[m.table.Cursor()].ServiceArn
					m.showDetails = true
				}
				return m, nil
				
			case keys.Matches(msg, m.keyMap.Refresh):
				m.loading = true
				return m, m.fetchServices
				
			case keys.Matches(msg, m.keyMap.Create):
				m.showCreate = true
				m.createModel = NewCreateModel(m.client, m.clusters)
				return m, m.createModel.Init()
				
			case keys.Matches(msg, m.keyMap.Delete):
				// TODO: Implement service deletion
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
			m.fetchServices,
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
		return m, m.fetchServices

	case servicesMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.services = msg.services
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
	if m.showCreate {
		return m.createModel.View()
	}

	if m.showSearch {
		return m.renderWithSearch()
	}

	if m.showFilter {
		return m.renderWithFilter()
	}

	if m.loading && len(m.services) == 0 {
		return styles.Content.Render("Loading services...")
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

// updateTable updates the table with current services
func (m *Model) updateTable() {
	rows := []table.Row{}
	for _, service := range m.filtered {
		status := styles.GetStatusStyle(service.Status).Render(service.Status)
		taskDef := m.extractTaskDefName(service.TaskDefinition)
		clusterName := m.getClusterName(service.ClusterArn)
		
		rows = append(rows, table.Row{
			service.ServiceName,
			status,
			fmt.Sprintf("%d", service.DesiredCount),
			fmt.Sprintf("%d", service.RunningCount),
			fmt.Sprintf("%d", service.PendingCount),
			taskDef,
			clusterName,
		})
	}
	m.table.SetRows(rows)
}

// renderDetails renders the service detail view
func (m *Model) renderDetails() string {
	if m.selectedARN == "" {
		return styles.Content.Render("No service selected")
	}

	// Find the selected service in the full services list
	var service *api.Service
	for i := range m.services {
		if m.services[i].ServiceArn == m.selectedARN {
			service = &m.services[i]
			break
		}
	}

	if service == nil {
		return styles.Content.Render("Service not found")
	}

	var content strings.Builder
	content.WriteString(styles.ListTitle.Render("Service Details"))
	content.WriteString("\n\n")

	// Basic info
	content.WriteString(fmt.Sprintf("Name: %s\n", styles.Info.Render(service.ServiceName)))
	content.WriteString(fmt.Sprintf("ARN: %s\n", styles.Info.Render(service.ServiceArn)))
	content.WriteString(fmt.Sprintf("Status: %s\n", styles.GetStatusStyle(service.Status).Render(service.Status)))
	content.WriteString(fmt.Sprintf("Cluster: %s\n", styles.Info.Render(m.getClusterName(service.ClusterArn))))
	content.WriteString("\n")

	// Task counts
	content.WriteString(styles.ListTitle.Render("Task Status"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("Desired Count: %d\n", service.DesiredCount))
	content.WriteString(fmt.Sprintf("Running Count: %d\n", service.RunningCount))
	content.WriteString(fmt.Sprintf("Pending Count: %d\n", service.PendingCount))
	content.WriteString("\n")

	// Task definition
	content.WriteString(styles.ListTitle.Render("Configuration"))
	content.WriteString("\n")
	content.WriteString(fmt.Sprintf("Task Definition: %s\n", service.TaskDefinition))
	if service.LaunchType != "" {
		content.WriteString(fmt.Sprintf("Launch Type: %s\n", service.LaunchType))
	}
	if service.PlatformVersion != "" {
		content.WriteString(fmt.Sprintf("Platform Version: %s\n", service.PlatformVersion))
	}
	if service.SchedulingStrategy != "" {
		content.WriteString(fmt.Sprintf("Scheduling Strategy: %s\n", service.SchedulingStrategy))
	}
	content.WriteString("\n")

	// Metadata
	if !service.CreatedAt.IsZero() {
		content.WriteString(styles.ListTitle.Render("Metadata"))
		content.WriteString("\n")
		content.WriteString(fmt.Sprintf("Created At: %s\n", service.CreatedAt.Format(time.RFC3339)))
		if service.CreatedBy != "" {
			content.WriteString(fmt.Sprintf("Created By: %s\n", service.CreatedBy))
		}
		content.WriteString("\n")
	}

	// Tags
	if len(service.Tags) > 0 {
		content.WriteString(styles.ListTitle.Render("Tags"))
		content.WriteString("\n")
		for _, tag := range service.Tags {
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

// fetchServices fetches the list of services
func (m *Model) fetchServices() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	allServices := []api.Service{}

	// If we have clusters, fetch services for each cluster
	if len(m.clusters) > 0 {
		for _, cluster := range m.clusters {
			// Skip if we have a selected cluster and this isn't it
			if m.selectedCluster != "" && cluster.ClusterArn != m.selectedCluster {
				continue
			}

			listResp, err := m.client.ListServices(ctx, cluster.ClusterArn)
			if err != nil {
				continue // Skip this cluster on error
			}

			if len(listResp.ServiceArns) > 0 {
				descResp, err := m.client.DescribeServices(ctx, cluster.ClusterArn, listResp.ServiceArns)
				if err == nil {
					allServices = append(allServices, descResp.Services...)
				}
			}
		}
	}

	return servicesMsg{services: allServices}
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

// applyFilters applies search and filter to the services
func (m *Model) applyFilters() {
	// Start with all services
	filtered := m.services

	// Apply search filter
	if m.searchModel.Value() != "" {
		filtered = search.Filter(filtered, m.searchModel.Value(), func(s api.Service) []string {
			return []string{s.ServiceName, s.ServiceArn}
		})
	}

	// Apply status filter
	selectedStatuses := m.filterModel.SelectedValues()
	if len(selectedStatuses) > 0 {
		filtered = filter.Apply(filtered, selectedStatuses, func(s api.Service, values []string) bool {
			for _, status := range values {
				if s.Status == status {
					return true
				}
			}
			return false
		})
	}

	// Apply cluster filter
	if m.selectedCluster != "" {
		var clusterFiltered []api.Service
		for _, service := range filtered {
			if service.ClusterArn == m.selectedCluster {
				clusterFiltered = append(clusterFiltered, service)
			}
		}
		filtered = clusterFiltered
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

// renderList renders the main service list
func (m *Model) renderList() string {
	var content strings.Builder
	content.WriteString(styles.ListTitle.Render("Services"))
	
	// Show active filters
	var filters []string
	if m.selectedCluster != "" {
		clusterName := m.getClusterName(m.selectedCluster)
		filters = append(filters, fmt.Sprintf("Cluster: %s", clusterName))
	}
	if m.searchModel.Value() != "" {
		filters = append(filters, fmt.Sprintf("Search: %s", m.searchModel.Value()))
	}
	if len(m.filterModel.SelectedValues()) > 0 {
		filters = append(filters, fmt.Sprintf("Status: %s", strings.Join(m.filterModel.SelectedValues(), ", ")))
	}
	
	if len(filters) > 0 {
		content.WriteString(" ")
		content.WriteString(styles.Info.Render("[" + strings.Join(filters, ", ") + "]"))
	}
	
	content.WriteString("\n\n")

	if len(m.filtered) == 0 {
		if len(m.services) == 0 {
			content.WriteString(styles.Info.Render("No services found. Press 'n' to create one."))
		} else {
			content.WriteString(styles.Info.Render("No services match the current filters."))
		}
	} else {
		content.WriteString(m.table.View())
		content.WriteString("\n\n")
		content.WriteString(styles.Info.Render(fmt.Sprintf("Showing %d of %d services", len(m.filtered), len(m.services))))
	}

	return styles.Content.Render(content.String())
}