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

package search

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// Model represents the search component
type Model struct {
	input       textinput.Model
	active      bool
	placeholder string
}

// New creates a new search model
func New(placeholder string) Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 100
	ti.Width = 40
	ti.Prompt = "🔍 "

	return Model{
		input:       ti,
		placeholder: placeholder,
	}
}

// Active returns whether the search is active
func (m Model) Active() bool {
	return m.active
}

// Value returns the current search value
func (m Model) Value() string {
	return strings.TrimSpace(m.input.Value())
}

// SetActive sets the search active state
func (m *Model) SetActive(active bool) {
	m.active = active
	if active {
		m.input.Focus()
	} else {
		m.input.Blur()
		m.input.SetValue("")
	}
}

// Update handles tea messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the search component
func (m Model) View() string {
	if !m.active {
		return ""
	}

	searchStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.GetStatusStyle("ACTIVE").GetForeground()).
		Padding(0, 1)

	return searchStyle.Render(m.input.View())
}

// Filter applies the search filter to a list of items
func Filter[T any](items []T, query string, getFields func(T) []string) []T {
	if query == "" {
		return items
	}

	query = strings.ToLower(query)
	var filtered []T

	for _, item := range items {
		fields := getFields(item)
		for _, field := range fields {
			if strings.Contains(strings.ToLower(field), query) {
				filtered = append(filtered, item)
				break
			}
		}
	}

	return filtered
}