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

package startup

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// DialogModel represents the startup confirmation dialog
type DialogModel struct {
	width     int
	height    int
	confirmed bool
	cancelled bool
	visible   bool
	endpoint  string
}

// NewDialog creates a new startup dialog
func NewDialog(endpoint string) *DialogModel {
	return &DialogModel{
		endpoint: endpoint,
		visible:  true,
	}
}

// Init initializes the dialog
func (m *DialogModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *DialogModel) Update(msg tea.Msg) (*DialogModel, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			m.confirmed = true
			m.visible = false
			return m, nil
		case "n", "N", "esc", "q":
			m.cancelled = true
			m.visible = false
			return m, tea.Quit
		case "ctrl+c":
			// Always allow quitting with Ctrl+C
			m.cancelled = true
			m.visible = false
			return m, tea.Quit
		}
	}

	return m, nil
}

// View renders the dialog
func (m *DialogModel) View() string {
	if !m.visible {
		return ""
	}

	var content strings.Builder

	// Title
	title := styles.TitleStyle.Render("KECS Not Running")
	content.WriteString(title + "\n\n")

	// Message
	message := fmt.Sprintf(
		"Could not connect to KECS at %s.\n\n"+
			"Would you like to start KECS now?\n\n"+
			"This will:\n"+
			"• Start a new KECS instance\n"+
			"• Create necessary Kubernetes resources\n"+
			"• Display startup logs",
		m.endpoint,
	)
	content.WriteString(message + "\n\n")

	// Options
	options := styles.SubtleStyle.Render("[Y] Yes, start KECS  [N] No, exit")
	content.WriteString(options)

	// Center the dialog
	dialog := styles.BoxStyle.
		Width(60).
		Height(15).
		Align(lipgloss.Center).
		Render(content.String())

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// SetSize updates the dialog size
func (m *DialogModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// IsVisible returns whether the dialog is visible
func (m *DialogModel) IsVisible() bool {
	return m.visible
}

// IsConfirmed returns whether the user confirmed
func (m *DialogModel) IsConfirmed() bool {
	return m.confirmed
}

// IsCancelled returns whether the user cancelled
func (m *DialogModel) IsCancelled() bool {
	return m.cancelled
}