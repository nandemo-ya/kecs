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

package keys

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// KeyMap defines all keyboard shortcuts for the TUI
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Actions
	Select  key.Binding
	Back    key.Binding
	Refresh key.Binding
	Delete  key.Binding
	Create  key.Binding
	Edit    key.Binding

	// Views
	Dashboard key.Binding
	Clusters  key.Binding
	Services  key.Binding
	Tasks     key.Binding
	TaskDefs  key.Binding
	Help      key.Binding

	// Application
	Quit     key.Binding
	ForceQuit key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+b"),
			key.WithHelp("pgup/ctrl+b", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+f"),
			key.WithHelp("pgdn/ctrl+f", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "go to start"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "go to end"),
		),

		// Actions
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r", "ctrl+r"),
			key.WithHelp("r/ctrl+r", "refresh"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d/del", "delete"),
		),
		Create: key.NewBinding(
			key.WithKeys("n", "ctrl+n"),
			key.WithHelp("n/ctrl+n", "create new"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),

		// Views
		Dashboard: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "dashboard"),
		),
		Clusters: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "clusters"),
		),
		Services: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "services"),
		),
		Tasks: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "tasks"),
		),
		TaskDefs: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "task definitions"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),

		// Application
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+\\"),
			key.WithHelp("ctrl+\\", "force quit"),
		),
	}
}

// Matches checks if a key message matches a binding
func Matches(msg tea.KeyMsg, binding key.Binding) bool {
	return key.Matches(msg, binding)
}