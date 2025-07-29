package tui2

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui2/mock"
)

// loadMockDataCmd loads mock data based on current selections
func (m Model) loadMockDataCmd() tea.Cmd {
	return mock.LoadAllData(m.selectedInstance, m.selectedCluster, m.selectedService, m.selectedTask)
}