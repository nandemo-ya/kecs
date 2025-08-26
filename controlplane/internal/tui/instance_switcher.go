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

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// InstanceSwitcher handles quick instance switching
type InstanceSwitcher struct {
	instances     []Instance
	selectedIndex int
	query         string
	filtered      []int // indices of filtered instances
}

// NewInstanceSwitcher creates a new instance switcher
func NewInstanceSwitcher(instances []Instance) *InstanceSwitcher {
	switcher := &InstanceSwitcher{
		instances:     instances,
		selectedIndex: 0,
	}
	switcher.updateFiltered()
	return switcher
}

// MoveUp moves selection up
func (s *InstanceSwitcher) MoveUp() {
	if s.selectedIndex > 0 {
		s.selectedIndex--
	}
}

// MoveDown moves selection down
func (s *InstanceSwitcher) MoveDown() {
	if s.selectedIndex < len(s.filtered)-1 {
		s.selectedIndex++
	}
}

// SetQuery sets the search query and updates filtering
func (s *InstanceSwitcher) SetQuery(query string) {
	s.query = query
	s.updateFiltered()
	s.selectedIndex = 0
}

// GetSelected returns the selected instance name
func (s *InstanceSwitcher) GetSelected() string {
	if len(s.filtered) == 0 {
		return ""
	}
	if s.selectedIndex >= len(s.filtered) {
		return ""
	}
	return s.instances[s.filtered[s.selectedIndex]].Name
}

// updateFiltered updates the list of filtered instances based on query
func (s *InstanceSwitcher) updateFiltered() {
	s.filtered = make([]int, 0)
	query := strings.ToLower(s.query)

	for i, inst := range s.instances {
		if query == "" || strings.Contains(strings.ToLower(inst.Name), query) {
			s.filtered = append(s.filtered, i)
		}
	}
}

// Render renders the instance switcher
func (s *InstanceSwitcher) Render(width, height int) string {
	// Styles
	switcherStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(1, 2).
		Width(50)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff00")).
		Bold(true)

	inputStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2a2a2a")).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 1).
		Width(46)

	itemStyle := lipgloss.NewStyle().
		PaddingLeft(2)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#005577")).
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true).
		Width(46)

	statusStyle := map[string]lipgloss.Style{
		"running":   lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")),
		"stopped":   lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
		"pending":   lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")),
		"unhealthy": lipgloss.NewStyle().Foreground(lipgloss.Color("#ff8800")),
	}

	// Build content
	var content []string

	// Title
	content = append(content, titleStyle.Render("Switch Instance"))
	content = append(content, "")

	// Input
	inputContent := fmt.Sprintf("> %s_", s.query)
	content = append(content, inputStyle.Render(inputContent))
	content = append(content, "")

	// Instance list
	maxItems := 8
	for i, idx := range s.filtered {
		if i >= maxItems {
			content = append(content, lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render("  ... and more"))
			break
		}

		inst := s.instances[idx]
		status := formatInstanceStatus(inst.Status)
		line := fmt.Sprintf("%-20s %s", inst.Name, status)

		if i == s.selectedIndex {
			line = "▸ " + line
			content = append(content, selectedStyle.Render(line))
		} else {
			style := itemStyle
			if st, ok := statusStyle[inst.Status]; ok {
				style = style.Inherit(st)
			}
			content = append(content, style.Render("  "+line))
		}
	}

	// Help
	content = append(content, "")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	content = append(content, helpStyle.Render("[↑/↓] Navigate  [Enter] Switch  [Esc] Cancel"))

	// Join and render
	dialog := switcherStyle.Render(strings.Join(content, "\n"))

	// Center on screen
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}
