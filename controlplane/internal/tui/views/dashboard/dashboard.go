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

package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// Stats represents the dashboard statistics
type Stats struct {
	Clusters       int
	Services       int
	RunningTasks   int
	PendingTasks   int
	TaskDefs       int
	LastUpdateTime time.Time
}

// Model represents the dashboard view model
type Model struct {
	endpoint string
	client   *api.Client
	stats    Stats
	width    int
	height   int
	loading  bool
	err      error
}

// tickMsg is sent when the refresh timer ticks
type tickMsg time.Time

// statsMsg is sent when stats are updated
type statsMsg struct {
	stats Stats
	err   error
}

// New creates a new dashboard model
func New(endpoint string) (*Model, error) {
	return &Model{
		endpoint: endpoint,
		client:   api.NewClient(endpoint),
		loading:  true,
	}, nil
}

// Init implements tea.Model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchStats,
		tick(),
	)
}

// Update implements tea.Model
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(
			m.fetchStats,
			tick(),
		)

	case statsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.stats = msg.stats
			m.err = nil
		}
		return m, nil
	}

	return m, nil
}

// View implements tea.Model
func (m *Model) View() string {
	if m.loading && m.stats.LastUpdateTime.IsZero() {
		return m.renderFullWidth("Loading dashboard...")
	}

	if m.err != nil {
		return m.renderFullWidth(styles.Error.Render("Error: " + m.err.Error()))
	}

	// Calculate box dimensions based on available space
	// Leave some padding and account for borders
	availableWidth := m.width - 4
	availableHeight := m.height - 8 // Space for rows, padding, and update info
	
	// Calculate box width: divide by number of columns with some spacing
	boxWidth := (availableWidth / 3) - 2
	if boxWidth < 20 {
		boxWidth = 20 // Minimum width
	}
	
	// Calculate box height
	boxHeight := (availableHeight / 2) - 1
	if boxHeight < 6 {
		boxHeight = 6 // Minimum height
	}
	if boxHeight > 10 {
		boxHeight = 10 // Maximum height to keep it reasonable
	}

	// Create stat boxes with dynamic sizing
	clusterBox := m.createStatBoxDynamic("Clusters", m.stats.Clusters, styles.StatusRunning, boxWidth, boxHeight)
	serviceBox := m.createStatBoxDynamic("Services", m.stats.Services, styles.StatusRunning, boxWidth, boxHeight)
	runningTaskBox := m.createStatBoxDynamic("Running Tasks", m.stats.RunningTasks, styles.StatusRunning, boxWidth, boxHeight)
	pendingTaskBox := m.createStatBoxDynamic("Pending Tasks", m.stats.PendingTasks, styles.StatusPending, boxWidth, boxHeight)
	taskDefBox := m.createStatBoxDynamic("Task Definitions", m.stats.TaskDefs, styles.Info, boxWidth, boxHeight)

	// Arrange boxes in a grid
	row1 := lipgloss.JoinHorizontal(
		lipgloss.Top,
		clusterBox,
		serviceBox,
		runningTaskBox,
	)

	row2 := lipgloss.JoinHorizontal(
		lipgloss.Top,
		pendingTaskBox,
		taskDefBox,
		strings.Repeat(" ", boxWidth+2), // Add empty space to balance the row
	)

	// Add last update time
	lastUpdate := fmt.Sprintf("Last updated: %s", m.stats.LastUpdateTime.Format("15:04:05"))
	updateInfo := styles.Info.Render(lastUpdate)

	// Combine all elements
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		row1,
		row2,
		"",
		updateInfo,
	)

	// Center the content in the available space
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// SetSize sets the size of the dashboard
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// createStatBox creates a styled box for displaying a statistic (legacy method for compatibility)
func (m *Model) createStatBox(title string, value int, valueStyle lipgloss.Style) string {
	return m.createStatBoxDynamic(title, value, valueStyle, 25, 6)
}

// createStatBoxDynamic creates a styled box with dynamic dimensions
func (m *Model) createStatBoxDynamic(title string, value int, valueStyle lipgloss.Style, width, height int) string {
	box := styles.InactivePanel.
		Width(width).
		Height(height).
		Padding(1).
		Margin(0, 1, 1, 0)

	titleStr := styles.ListTitle.Render(title)
	valueStr := valueStyle.Render(fmt.Sprintf("%d", value))

	// Calculate inner dimensions
	innerWidth := width - 2  // Account for borders
	innerHeight := height - 2 // Account for borders

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStr,
		"",
		valueStr,
	)

	return box.Render(lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Center, content))
}

// renderFullWidth renders content centered in the full available space
func (m *Model) renderFullWidth(content string) string {
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		styles.Content.Render(content),
	)
}

// fetchStats fetches the latest statistics
func (m *Model) fetchStats() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats := Stats{
		LastUpdateTime: time.Now(),
	}

	// Fetch clusters
	listResp, err := m.client.ListClusters(ctx)
	if err != nil {
		// Return partial stats even on error
		return statsMsg{stats: stats, err: err}
	}

	stats.Clusters = len(listResp.ClusterArns)

	// If we have clusters, get detailed info
	if len(listResp.ClusterArns) > 0 {
		descResp, err := m.client.DescribeClusters(ctx, listResp.ClusterArns)
		if err == nil {
			for _, cluster := range descResp.Clusters {
				stats.Services += cluster.ActiveServicesCount
				stats.RunningTasks += cluster.RunningTasksCount
				stats.PendingTasks += cluster.PendingTasksCount
			}
		}
	}

	// TODO: Fetch task definitions count when API is available
	// For now, use a placeholder
	stats.TaskDefs = 0

	return statsMsg{
		stats: stats,
		err:   nil,
	}
}

// tick returns a command that sends a tick message after a delay
func tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}