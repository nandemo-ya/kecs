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

package instances

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/keys"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// CreateFormModel represents the instance creation form
type CreateFormModel struct {
	inputs          []textinput.Model
	focusIndex      int
	localstackEnabled bool
	traefikEnabled   bool
	devModeEnabled   bool
	width           int
	height          int
	err             error
	creating        bool
	keyMap          keys.KeyMap
}

const (
	nameInput = iota
	apiPortInput
	adminPortInput
)

// NewCreateForm creates a new instance creation form
func NewCreateForm() *CreateFormModel {
	inputs := make([]textinput.Model, 3)
	
	// Instance name input
	inputs[nameInput] = textinput.New()
	inputs[nameInput].Placeholder = "Auto-generated if empty"
	inputs[nameInput].Focus()
	inputs[nameInput].CharLimit = 50
	inputs[nameInput].Width = 40
	inputs[nameInput].Prompt = "Name: "
	
	// API Port input
	inputs[apiPortInput] = textinput.New()
	inputs[apiPortInput].Placeholder = "8080 or auto"
	inputs[apiPortInput].CharLimit = 5
	inputs[apiPortInput].Width = 20
	inputs[apiPortInput].Prompt = "API Port: "
	
	// Admin Port input
	inputs[adminPortInput] = textinput.New()
	inputs[adminPortInput].Placeholder = "8081 or auto"
	inputs[adminPortInput].CharLimit = 5
	inputs[adminPortInput].Width = 20
	inputs[adminPortInput].Prompt = "Admin Port: "
	
	return &CreateFormModel{
		inputs:            inputs,
		localstackEnabled: true,
		traefikEnabled:    true,
		devModeEnabled:    false,
		keyMap:            keys.DefaultKeyMap(),
	}
}

// Update handles messages
func (m *CreateFormModel) Update(msg tea.Msg) (*CreateFormModel, tea.Cmd) {
	if m.creating {
		return m, nil
	}
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "up", "down":
			// Navigate between inputs
			if msg.String() == "tab" || msg.String() == "down" {
				m.focusIndex++
				if m.focusIndex > len(m.inputs)+2 { // +3 for checkboxes
					m.focusIndex = 0
				}
			} else {
				m.focusIndex--
				if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs) + 2
				}
			}
			
			// Update focus
			for i := range m.inputs {
				if i == m.focusIndex {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			
		case " ":
			// Toggle checkboxes
			switch m.focusIndex {
			case len(m.inputs):
				m.localstackEnabled = !m.localstackEnabled
			case len(m.inputs) + 1:
				m.traefikEnabled = !m.traefikEnabled
			case len(m.inputs) + 2:
				m.devModeEnabled = !m.devModeEnabled
			}
			
		case "ctrl+s":
			// Submit form
			return m, m.createInstance()
		}
	}
	
	// Update text inputs
	var cmds []tea.Cmd
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}
	
	return m, tea.Batch(cmds...)
}

// View renders the form
func (m *CreateFormModel) View() string {
	if m.creating {
		return styles.Info.Render("Creating instance...")
	}
	
	var b strings.Builder
	
	b.WriteString(styles.TitleStyle.Render("Create New Instance") + "\n\n")
	
	// Show error if any
	if m.err != nil {
		b.WriteString(styles.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\n")
	}
	
	// Text inputs
	for i, input := range m.inputs {
		if i == m.focusIndex {
			b.WriteString(styles.ActiveStyle.Render(input.View()) + "\n")
		} else {
			b.WriteString(input.View() + "\n")
		}
	}
	
	b.WriteString("\n")
	
	// Checkboxes
	checkboxes := []struct {
		label   string
		enabled *bool
		index   int
	}{
		{"Enable LocalStack", &m.localstackEnabled, len(m.inputs)},
		{"Enable Traefik", &m.traefikEnabled, len(m.inputs) + 1},
		{"Dev Mode", &m.devModeEnabled, len(m.inputs) + 2},
	}
	
	for _, cb := range checkboxes {
		checkbox := "[ ]"
		if *cb.enabled {
			checkbox = "[✓]"
		}
		
		label := fmt.Sprintf("%s %s", checkbox, cb.label)
		if m.focusIndex == cb.index {
			b.WriteString(styles.ActiveStyle.Render(label) + "\n")
		} else {
			b.WriteString(label + "\n")
		}
	}
	
	b.WriteString("\n")
	b.WriteString(styles.SubtleStyle.Render("[Tab] Navigate  [Space] Toggle  [Ctrl+S] Create  [Esc] Cancel"))
	
	return b.String()
}

// SetSize updates the form size
func (m *CreateFormModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Reset clears the form
func (m *CreateFormModel) Reset() {
	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}
	m.focusIndex = 0
	m.inputs[0].Focus()
	m.err = nil
	m.creating = false
	m.localstackEnabled = true
	m.traefikEnabled = true
	m.devModeEnabled = false
}

// Helper methods

func (m *CreateFormModel) createInstance() tea.Cmd {
	return func() tea.Msg {
		// Validate inputs
		name := strings.TrimSpace(m.inputs[nameInput].Value())
		
		apiPort := 0
		if portStr := strings.TrimSpace(m.inputs[apiPortInput].Value()); portStr != "" && portStr != "auto" {
			port, err := strconv.Atoi(portStr)
			if err != nil || port < 1 || port > 65535 {
				return instanceCreatedMsg{err: fmt.Errorf("invalid API port: %s", portStr)}
			}
			apiPort = port
		}
		
		adminPort := 0
		if portStr := strings.TrimSpace(m.inputs[adminPortInput].Value()); portStr != "" && portStr != "auto" {
			port, err := strconv.Atoi(portStr)
			if err != nil || port < 1 || port > 65535 {
				return instanceCreatedMsg{err: fmt.Errorf("invalid admin port: %s", portStr)}
			}
			adminPort = port
		}
		
		// Create request
		_ = api.CreateInstanceRequest{
			Name:         name,
			APIPort:      apiPort,
			AdminPort:    adminPort,
			NoLocalStack: !m.localstackEnabled,
			NoTraefik:    !m.traefikEnabled,
			DevMode:      m.devModeEnabled,
		}
		
		// TODO: Actually create the instance
		// For now, return an error
		return instanceCreatedMsg{
			err: fmt.Errorf("instance creation not implemented yet - use CLI: kecs start"),
		}
	}
}

// Message types

type instanceCreatedMsg struct {
	err error
}