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
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/keys"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// Model represents the cluster list view model
type Model struct {
	client       *api.Client
	table        table.Model
	clusters     []api.Cluster
	width        int
	height       int
	loading      bool
	err          error
	keyMap       keys.KeyMap
	selectedARN  string
	showDetails  bool
	showCreate   bool
	createModel  CreateModel
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

	return &Model{
		client:      client,
		table:       t,
		loading:     true,
		keyMap:      keys.DefaultKeyMap(),
		createModel: NewCreateModel(client),
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
	if m.showCreate {
		return m.createModel.View()
	}

	if m.loading && len(m.clusters) == 0 {
		return styles.Content.Render("Loading clusters...")
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
	content.WriteString(styles.ListTitle.Render("Clusters"))
	content.WriteString("\n\n")

	if len(m.clusters) == 0 {
		content.WriteString(styles.Info.Render("No clusters found. Press 'n' to create one."))
	} else {
		content.WriteString(m.table.View())
		content.WriteString("\n\n")
		content.WriteString(styles.Info.Render(fmt.Sprintf("Showing %d clusters", len(m.clusters))))
	}

	return styles.Content.Render(content.String())
}

// SetSize sets the size of the view
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetHeight(height - 10)
}

// updateTable updates the table with current clusters
func (m *Model) updateTable() {
	rows := []table.Row{}
	for _, cluster := range m.clusters {
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
		return styles.Content.Render("No cluster selected")
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
		return styles.Content.Render("Cluster not found")
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

	return styles.Content.Render(content.String())
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