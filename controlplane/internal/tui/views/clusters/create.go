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

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/components/help"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

type createKeyMap struct {
	Submit key.Binding
	Cancel key.Binding
	Next   key.Binding
	Prev   key.Binding
	Help   key.Binding
}

var createKeys = createKeyMap{
	Submit: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "submit"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	Next: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	Prev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous field"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

type CreateMsg struct{}
type CreatedMsg struct {
	Cluster *api.Cluster
}
type CreateErrorMsg struct {
	Error error
}

type CreateModel struct {
	client api.APIClient
	inputs []textinput.Model
	focus  int
	help   help.Model
	error  error
}

func NewCreateModel(client api.APIClient) CreateModel {
	// Initialize input fields
	nameInput := textinput.New()
	nameInput.Placeholder = "my-cluster"
	nameInput.Focus()
	nameInput.CharLimit = 255
	nameInput.Prompt = "Name: "
	nameInput.Width = 50

	// TODO: Add more fields as needed (tags, settings, etc.)

	inputs := []textinput.Model{nameInput}

	return CreateModel{
		client: client,
		inputs: inputs,
		focus:  0,
		help:   help.New(),
	}
}

func (m CreateModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CreateModel) Update(msg tea.Msg) (CreateModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, createKeys.Submit):
			// Validate and submit
			if err := m.validate(); err != nil {
				m.error = err
				return m, nil
			}
			return m, m.submit
		case key.Matches(msg, createKeys.Cancel):
			return m, func() tea.Msg { return ClusterListMsg{} }
		case key.Matches(msg, createKeys.Next):
			m.focus = (m.focus + 1) % len(m.inputs)
			cmds = append(cmds, m.updateFocus())
		case key.Matches(msg, createKeys.Prev):
			m.focus--
			if m.focus < 0 {
				m.focus = len(m.inputs) - 1
			}
			cmds = append(cmds, m.updateFocus())
		case key.Matches(msg, createKeys.Help):
			m.help.ShowAll = !m.help.ShowAll
		}
	}

	// Update the focused input
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		if i == m.focus {
			cmds = append(cmds, cmd)
		}
	}

	// Update help
	m.help, _ = m.help.Update(msg)

	return m, tea.Batch(cmds...)
}

func (m CreateModel) View() string {
	var b strings.Builder

	// Title
	title := styles.Header.Render("Create Cluster")
	b.WriteString(title + "\n\n")

	// Error message
	if m.error != nil {
		errorMsg := styles.Error.Render(fmt.Sprintf("Error: %v", m.error))
		b.WriteString(errorMsg + "\n\n")
	}

	// Form fields
	for i, input := range m.inputs {
		if i == m.focus {
			b.WriteString(styles.SelectedListItem.Render(input.View()))
		} else {
			b.WriteString(styles.ListItem.Render(input.View()))
		}
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n" + m.help.View(createKeys))

	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func (m CreateModel) updateFocus() tea.Cmd {
	// Blur all inputs
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	// Focus the current one
	if m.focus >= 0 && m.focus < len(m.inputs) {
		return m.inputs[m.focus].Focus()
	}
	return nil
}

func (m CreateModel) validate() error {
	// Validate cluster name
	name := strings.TrimSpace(m.inputs[0].Value())
	if name == "" {
		return fmt.Errorf("cluster name is required")
	}
	if len(name) > 255 {
		return fmt.Errorf("cluster name must be 255 characters or less")
	}
	// TODO: Add more validation rules (e.g., allowed characters)
	return nil
}

func (m CreateModel) submit() tea.Msg {
	name := strings.TrimSpace(m.inputs[0].Value())
	
	req := &api.CreateClusterRequest{
		ClusterName: name,
	}

	resp, err := m.client.CreateCluster(context.Background(), req)
	if err != nil {
		return CreateErrorMsg{Error: err}
	}

	cluster := &resp.Cluster

	return CreatedMsg{Cluster: cluster}
}

func (m CreateModel) ShortHelp() []key.Binding {
	return []key.Binding{createKeys.Submit, createKeys.Cancel, createKeys.Help}
}

func (m CreateModel) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{createKeys.Submit, createKeys.Cancel},
		{createKeys.Next, createKeys.Prev},
		{createKeys.Help},
	}
}