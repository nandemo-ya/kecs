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

package help

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the help component
type Model struct {
	help     help.Model
	ShowAll  bool
}

// New creates a new help model
func New() Model {
	h := help.New()
	h.ShowAll = false
	return Model{
		help:    h,
		ShowAll: false,
	}
}

// Update handles tea messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// The help bubble doesn't produce commands
	return m, nil
}

// View renders the help
func (m Model) View(keyMap interface{}) string {
	// Check if keyMap implements help.KeyMap interface
	if km, ok := keyMap.(help.KeyMap); ok {
		m.help.ShowAll = m.ShowAll
		return m.help.View(km)
	}

	// If not, we need to build a manual help string
	var b strings.Builder
	b.WriteString("Help not available")
	return b.String()
}

// ShortHelp returns the short help
func (m Model) ShortHelp(keyBindings []key.Binding) string {
	return m.help.ShortHelpView(keyBindings)
}

// FullHelp returns the full help
func (m Model) FullHelp(keyGroups [][]key.Binding) string {
	return m.help.FullHelpView(keyGroups)
}