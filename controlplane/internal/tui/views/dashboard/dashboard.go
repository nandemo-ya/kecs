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
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		return styles.Content.Render("Loading dashboard...")
	}

	if m.err != nil {
		return styles.Content.Render(
			styles.Error.Render("Error: " + m.err.Error()),
		)
	}

	// Create stat boxes
	clusterBox := m.createStatBox("Clusters", m.stats.Clusters, styles.StatusRunning)
	serviceBox := m.createStatBox("Services", m.stats.Services, styles.StatusRunning)
	runningTaskBox := m.createStatBox("Running Tasks", m.stats.RunningTasks, styles.StatusRunning)
	pendingTaskBox := m.createStatBox("Pending Tasks", m.stats.PendingTasks, styles.StatusPending)
	taskDefBox := m.createStatBox("Task Definitions", m.stats.TaskDefs, styles.Info)

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

	return styles.Content.Render(content)
}

// SetSize sets the size of the dashboard
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// createStatBox creates a styled box for displaying a statistic
func (m *Model) createStatBox(title string, value int, valueStyle lipgloss.Style) string {
	box := styles.InactivePanel.
		Width(25).
		Height(6).
		Padding(1).
		Margin(0, 1, 1, 0)

	titleStr := styles.ListTitle.Render(title)
	valueStr := valueStyle.Render(fmt.Sprintf("%d", value))

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStr,
		"",
		valueStr,
	)

	return box.Render(lipgloss.Place(23, 4, lipgloss.Center, lipgloss.Center, content))
}

// fetchStats fetches the latest statistics
func (m *Model) fetchStats() tea.Msg {
	// TODO: Implement actual API calls to fetch stats
	// For now, return mock data
	return statsMsg{
		stats: Stats{
			Clusters:       3,
			Services:       12,
			RunningTasks:   45,
			PendingTasks:   5,
			TaskDefs:       18,
			LastUpdateTime: time.Now(),
		},
		err: nil,
	}
}

// tick returns a command that sends a tick message after a delay
func tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}