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

package services

import (
	"context"
	"fmt"
	"strconv"
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
	Service *api.Service
}
type CreateErrorMsg struct {
	Error error
}

type CreateModel struct {
	client   api.APIClient
	inputs   []textinput.Model
	clusters []api.Cluster
	focus    int
	help     help.Model
	error    error
}

const (
	nameField = iota
	clusterField
	taskDefField
	desiredCountField
)

func NewCreateModel(client api.APIClient, clusters []api.Cluster) CreateModel {
	// Initialize input fields
	nameInput := textinput.New()
	nameInput.Placeholder = "my-service"
	nameInput.Focus()
	nameInput.CharLimit = 255
	nameInput.Prompt = "Service Name: "
	nameInput.Width = 50

	clusterInput := textinput.New()
	if len(clusters) > 0 {
		clusterInput.Placeholder = clusters[0].ClusterName
	} else {
		clusterInput.Placeholder = "default"
	}
	clusterInput.CharLimit = 255
	clusterInput.Prompt = "Cluster: "
	clusterInput.Width = 50

	taskDefInput := textinput.New()
	taskDefInput.Placeholder = "my-task-def:1"
	taskDefInput.CharLimit = 255
	taskDefInput.Prompt = "Task Definition: "
	taskDefInput.Width = 50

	desiredCountInput := textinput.New()
	desiredCountInput.Placeholder = "1"
	desiredCountInput.CharLimit = 10
	desiredCountInput.Prompt = "Desired Count: "
	desiredCountInput.Width = 20

	inputs := []textinput.Model{nameInput, clusterInput, taskDefInput, desiredCountInput}

	return CreateModel{
		client:   client,
		inputs:   inputs,
		clusters: clusters,
		focus:    0,
		help:     help.New(),
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
			return m, func() tea.Msg { return ServiceListMsg{} }
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
	title := styles.Header.Render("Create Service")
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

	// Available clusters hint
	if len(m.clusters) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.Info.Render("Available clusters: "))
		clusterNames := make([]string, len(m.clusters))
		for i, c := range m.clusters {
			clusterNames[i] = c.ClusterName
		}
		b.WriteString(strings.Join(clusterNames, ", "))
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
	// Validate service name
	name := strings.TrimSpace(m.inputs[nameField].Value())
	if name == "" {
		return fmt.Errorf("service name is required")
	}

	// Validate cluster
	cluster := strings.TrimSpace(m.inputs[clusterField].Value())
	if cluster == "" && len(m.clusters) == 0 {
		return fmt.Errorf("cluster is required")
	}

	// Validate task definition
	taskDef := strings.TrimSpace(m.inputs[taskDefField].Value())
	if taskDef == "" {
		return fmt.Errorf("task definition is required")
	}

	// Validate desired count
	desiredCountStr := strings.TrimSpace(m.inputs[desiredCountField].Value())
	if desiredCountStr != "" {
		if _, err := strconv.ParseInt(desiredCountStr, 10, 64); err != nil {
			return fmt.Errorf("desired count must be a valid number")
		}
	}

	return nil
}

func (m CreateModel) submit() tea.Msg {
	name := strings.TrimSpace(m.inputs[nameField].Value())
	cluster := strings.TrimSpace(m.inputs[clusterField].Value())
	if cluster == "" && len(m.clusters) > 0 {
		cluster = m.clusters[0].ClusterName
	}
	taskDef := strings.TrimSpace(m.inputs[taskDefField].Value())
	
	desiredCountStr := strings.TrimSpace(m.inputs[desiredCountField].Value())
	var desiredCount int = 1
	if desiredCountStr != "" {
		parsed, _ := strconv.ParseInt(desiredCountStr, 10, 32)
		desiredCount = int(parsed)
	}
	
	req := &api.CreateServiceRequest{
		ServiceName:    name,
		Cluster:        cluster,
		TaskDefinition: taskDef,
		DesiredCount:   desiredCount,
	}

	resp, err := m.client.CreateService(context.Background(), req)
	if err != nil {
		return CreateErrorMsg{Error: err}
	}

	service := &resp.Service

	return CreatedMsg{Service: service}
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