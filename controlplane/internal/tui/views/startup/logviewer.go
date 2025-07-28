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
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// LogViewerModel represents the startup log viewer
type LogViewerModel struct {
	viewport     viewport.Model
	spinner      spinner.Model
	logs         []string
	width        int
	height       int
	ready        bool
	starting     bool
	completed    bool
	failed       bool
	errorMsg     string
	instanceName string
	Streamer     *LogStreamer
}

// NewLogViewer creates a new log viewer
func NewLogViewer(instanceName string) *LogViewerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.PrimaryColor)

	// Create with default size
	m := &LogViewerModel{
		spinner:      s,
		logs:         []string{},
		instanceName: instanceName,
		starting:     true,
		ready:        false,
		width:        80,  // Default width
		height:       24,  // Default height
	}
	
	// Initialize viewport with default size
	m.viewport = viewport.New(76, 14) // width-4, height-10
	m.viewport.YPosition = 0
	m.ready = true
	
	return m
}

// Init initializes the log viewer
func (m *LogViewerModel) Init() tea.Cmd {
	// Add initial log message
	m.logs = append(m.logs, fmt.Sprintf("[%s] Initializing KECS startup...", time.Now().Format("15:04:05")))
	
	return tea.Batch(
		m.spinner.Tick,
		m.startKECS(),
	)
}

// Update handles messages
func (m *LogViewerModel) Update(msg tea.Msg) (*LogViewerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		if !m.ready {
			// Initialize viewport
			m.viewport = viewport.New(msg.Width-4, msg.Height-10)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 10
		}
		m.updateViewport()

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			if m.completed || m.failed {
				return m, tea.Quit
			}
		case "enter":
			if m.completed {
				// Continue to TUI
				return m, nil
			}
		}

	case startupLogMsg:
		m.logs = append(m.logs, msg.line)
		m.updateViewport()
		return m, nil
		
	case startupProgressMsg:
		m.logs = append(m.logs, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg.message))
		m.updateViewport()
		return m, nil

	case startupCompleteMsg:
		m.starting = false
		m.completed = true
		m.logs = append(m.logs, "\n✅ KECS started successfully!")
		m.logs = append(m.logs, "Press ENTER to continue to the TUI")
		m.updateViewport()

	case startupErrorMsg:
		m.starting = false
		m.failed = true
		m.errorMsg = msg.err.Error()
		m.logs = append(m.logs, fmt.Sprintf("\n❌ Failed to start KECS: %s", msg.err))
		m.logs = append(m.logs, "Press ESC to exit")
		m.updateViewport()

	case StartupStreamerMsg:
		// Store the streamer reference
		m.Streamer = msg.Streamer
		// The streamer will send messages to the program

	case spinner.TickMsg:
		if m.starting {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the log viewer
func (m *LogViewerModel) View() string {
	if !m.ready {
		// Show more information when not ready
		return fmt.Sprintf("\n  Initializing... (width: %d, height: %d, logs: %d)\n", 
			m.width, m.height, len(m.logs))
	}

	var b strings.Builder

	// Title
	title := "Starting KECS"
	if m.instanceName != "" {
		title = fmt.Sprintf("Starting KECS Instance: %s", m.instanceName)
	}
	b.WriteString(styles.TitleStyle.Render(title) + "\n\n")

	// Status
	if m.starting {
		b.WriteString(fmt.Sprintf("%s Starting KECS...\n\n", m.spinner.View()))
	} else if m.completed {
		b.WriteString(styles.Success.Render("✅ Started successfully") + "\n\n")
	} else if m.failed {
		b.WriteString(styles.Error.Render("❌ Startup failed") + "\n\n")
	}

	// Log viewport
	viewportStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1)

	b.WriteString(viewportStyle.Render(m.viewport.View()))

	// Footer
	b.WriteString("\n")
	if m.starting {
		b.WriteString(styles.SubtleStyle.Render("Starting KECS... Please wait"))
	} else if m.completed {
		b.WriteString(styles.Info.Render("Press ENTER to continue"))
	} else if m.failed {
		b.WriteString(styles.Error.Render("Press ESC to exit"))
	}

	return b.String()
}

// SetSize updates the viewer size
func (m *LogViewerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	
	// Initialize viewport if not ready
	if !m.ready && width > 0 && height > 0 {
		m.viewport = viewport.New(width-4, height-10)
		m.viewport.YPosition = 0
		m.ready = true
		m.updateViewport()
	} else if m.ready {
		m.viewport.Width = width - 4
		m.viewport.Height = height - 10
		m.updateViewport()
	}
}

// IsCompleted returns whether startup completed successfully
func (m *LogViewerModel) IsCompleted() bool {
	return m.completed
}

// IsFailed returns whether startup failed
func (m *LogViewerModel) IsFailed() bool {
	return m.failed
}

// updateViewport updates the viewport content
func (m *LogViewerModel) updateViewport() {
	if m.ready {
		content := strings.Join(m.logs, "\n")
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()
	}
}

// startKECS starts the KECS instance
func (m *LogViewerModel) startKECS() tea.Cmd {
	// Ensure we have an instance name
	instanceName := m.instanceName
	if instanceName == "" {
		instanceName = "default"
	}
	
	// Extract port from instance name or use default
	apiPort := 8080
	if instanceName != "default" {
		// Try to determine port from instance name
		// This matches the port allocation in instances.go
		switch instanceName {
		case "dev":
			apiPort = 8080
		case "staging":
			apiPort = 8090
		case "test":
			apiPort = 8100
		case "local":
			apiPort = 8110
		case "prod":
			apiPort = 8200
		default:
			// Use hash-based allocation for custom instances
			hash := 0
			for _, c := range instanceName {
				hash = hash*31 + int(c)
			}
			apiPort = 8300 + (hash % 700)
		}
	}
	
	// Add log about starting
	m.logs = append(m.logs, fmt.Sprintf("[%s] Starting KECS instance '%s' on port %d...", 
		time.Now().Format("15:04:05"), instanceName, apiPort))
	
	return StartKECSWithStreamer(instanceName, apiPort)
}

// Message types

type startupLogMsg struct {
	line string
}

type startupProgressMsg struct {
	message string
}

type startupCompleteMsg struct{}

type startupErrorMsg struct {
	err error
}