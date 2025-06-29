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

package filter

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// Option represents a filter option
type Option struct {
	Label string
	Value string
}

// Model represents the filter component
type Model struct {
	title    string
	options  []Option
	selected map[string]bool
	active   bool
	cursor   int
}

// New creates a new filter model
func New(title string, options []Option) Model {
	return Model{
		title:    title,
		options:  options,
		selected: make(map[string]bool),
	}
}

// Active returns whether the filter is active
func (m Model) Active() bool {
	return m.active
}

// SetActive sets the filter active state
func (m *Model) SetActive(active bool) {
	m.active = active
	if !active {
		m.cursor = 0
	}
}

// SelectedValues returns the selected filter values
func (m Model) SelectedValues() []string {
	var values []string
	for _, opt := range m.options {
		if m.selected[opt.Value] {
			values = append(values, opt.Value)
		}
	}
	return values
}

// Update handles tea messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case " ", "enter":
			if m.cursor < len(m.options) {
				opt := m.options[m.cursor]
				m.selected[opt.Value] = !m.selected[opt.Value]
			}
		case "a":
			// Toggle all
			allSelected := len(m.selected) == len(m.options)
			for _, opt := range m.options {
				m.selected[opt.Value] = !allSelected
			}
		case "c":
			// Clear all
			m.selected = make(map[string]bool)
		}
	}

	return m, nil
}

// View renders the filter component
func (m Model) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.GetStatusStyle("ACTIVE").GetForeground())
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")

	// Options
	for i, opt := range m.options {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		checkbox := "[ ]"
		if m.selected[opt.Value] {
			checkbox = "[✓]"
		}

		line := fmt.Sprintf("%s%s %s", cursor, checkbox, opt.Label)
		
		if i == m.cursor {
			b.WriteString(styles.SelectedListItem.Render(line))
		} else {
			b.WriteString(styles.ListItem.Render(line))
		}
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(styles.GetStatusStyle("UNKNOWN").GetForeground())
	b.WriteString(helpStyle.Render("space: toggle, a: all, c: clear"))

	// Wrap in a border
	filterStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.GetStatusStyle("ACTIVE").GetForeground()).
		Padding(1, 2)

	return filterStyle.Render(b.String())
}

// FilterFunc is a function that filters items based on selected values
type FilterFunc[T any] func(item T, selectedValues []string) bool

// Apply applies the filter to a list of items
func Apply[T any](items []T, selectedValues []string, filterFunc FilterFunc[T]) []T {
	if len(selectedValues) == 0 {
		return items
	}

	var filtered []T
	for _, item := range items {
		if filterFunc(item, selectedValues) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}