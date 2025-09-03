package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// LogViewerModel represents the log viewer component
type LogViewerModel struct {
	viewport     viewport.Model
	searchBar    textinput.Model
	logs         []storage.TaskLog
	filteredLogs []storage.TaskLog
	taskArn      string
	container    string
	apiClient    LogAPIClient
	streaming    bool
	follow       bool
	searchTerm   string
	width        int
	height       int
	loading      bool
	error        error
}

// LogAPIClient interface for log operations
type LogAPIClient interface {
	GetLogs(ctx context.Context, taskArn, container string, follow bool) ([]storage.TaskLog, error)
	StreamLogs(ctx context.Context, taskArn, container string) (<-chan storage.TaskLog, error)
}

// LogStreamMsg represents a log message from streaming
type LogStreamMsg struct {
	Log storage.TaskLog
}

// Note: logsLoadedMsg is defined in commands.go and used by main TUI

// LogErrorMsg represents an error loading logs
type LogErrorMsg struct {
	Error error
}

// NewLogViewer creates a new log viewer model
func NewLogViewer(taskArn, container string, apiClient LogAPIClient) LogViewerModel {
	searchBar := textinput.New()
	searchBar.Placeholder = "Search logs..."
	searchBar.CharLimit = 100
	searchBar.Width = 50

	vp := viewport.New(80, 20)
	vp.SetContent("Loading logs...")

	return LogViewerModel{
		viewport:     vp,
		searchBar:    searchBar,
		taskArn:      taskArn,
		container:    container,
		apiClient:    apiClient,
		logs:         []storage.TaskLog{},
		filteredLogs: []storage.TaskLog{},
		loading:      true,
	}
}

// Init initializes the log viewer
func (m LogViewerModel) Init() tea.Cmd {
	return m.loadLogs()
}

// loadLogs loads logs from the API
func (m LogViewerModel) loadLogs() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logs, err := m.apiClient.GetLogs(ctx, m.taskArn, m.container, false)
		if err != nil {
			return LogErrorMsg{Error: err}
		}

		// Convert storage.TaskLog to LogEntry for consistency with main TUI
		logEntries := make([]LogEntry, len(logs))
		for i, taskLog := range logs {
			logEntries[i] = LogEntry{
				Timestamp: taskLog.Timestamp,
				Level:     taskLog.LogLevel,
				Message:   taskLog.LogLine,
			}
		}
		return logsLoadedMsg{logs: logEntries}
	}
}

// startStreaming starts streaming logs
func (m LogViewerModel) startStreaming() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		stream, err := m.apiClient.StreamLogs(ctx, m.taskArn, m.container)
		if err != nil {
			return LogErrorMsg{Error: err}
		}

		// Start listening to stream in a goroutine
		go func() {
			for log := range stream {
				// Send log message through tea.Program
				// This would need to be handled properly with the program instance
				_ = log
			}
		}()

		return nil
	}
}

// Update handles messages
func (m LogViewerModel) Update(msg tea.Msg) (LogViewerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Adjust viewport size
		headerHeight := 3
		footerHeight := 3
		searchHeight := 3

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight - footerHeight - searchHeight

		// Update search bar width
		m.searchBar.Width = msg.Width - 4

	case logsLoadedMsg:
		// Convert LogEntry to storage.TaskLog
		m.logs = make([]storage.TaskLog, len(msg.logs))
		for i, logEntry := range msg.logs {
			m.logs[i] = storage.TaskLog{
				Timestamp: logEntry.Timestamp,
				LogLevel:  logEntry.Level,
				LogLine:   logEntry.Message,
			}
		}
		m.loading = false
		m.filterLogs()
		m.updateViewport()

	case LogStreamMsg:
		m.logs = append(m.logs, msg.Log)
		m.filterLogs()
		m.updateViewport()

		// Auto-scroll if following
		if m.follow {
			m.viewport.GotoBottom()
		}

	case LogErrorMsg:
		m.error = msg.Error
		m.loading = false

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "/":
			// Focus search bar
			m.searchBar.Focus()
			cmds = append(cmds, textinput.Blink)

		case "esc":
			// Unfocus search bar
			m.searchBar.Blur()
			m.searchTerm = ""
			m.searchBar.SetValue("")
			m.filterLogs()
			m.updateViewport()

		case "F":
			// Toggle follow mode (uppercase F to avoid conflict with split-view toggle)
			m.follow = !m.follow
			if m.follow {
				m.viewport.GotoBottom()
				if !m.streaming {
					m.streaming = true
					cmds = append(cmds, m.startStreaming())
				}
			}

		case "r":
			// Reload logs
			m.loading = true
			cmds = append(cmds, m.loadLogs())

		case "g":
			// Go to top
			m.viewport.GotoTop()

		case "G":
			// Go to bottom
			m.viewport.GotoBottom()

		case "enter":
			if m.searchBar.Focused() {
				m.searchTerm = m.searchBar.Value()
				m.searchBar.Blur()
				m.filterLogs()
				m.updateViewport()
			}
		}

		// Handle search bar input
		if m.searchBar.Focused() {
			var cmd tea.Cmd
			m.searchBar, cmd = m.searchBar.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			// Handle viewport navigation
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// filterLogs filters logs based on search term
func (m *LogViewerModel) filterLogs() {
	if m.searchTerm == "" {
		m.filteredLogs = m.logs
		return
	}

	filtered := []storage.TaskLog{}
	searchLower := strings.ToLower(m.searchTerm)

	for _, log := range m.logs {
		if strings.Contains(strings.ToLower(log.LogLine), searchLower) {
			filtered = append(filtered, log)
		}
	}

	m.filteredLogs = filtered
}

// updateViewport updates the viewport content
func (m *LogViewerModel) updateViewport() {
	var content strings.Builder

	for _, log := range m.filteredLogs {
		timestamp := log.Timestamp.Format("15:04:05.000")

		// Color based on log level
		var levelStyle lipgloss.Style
		switch log.LogLevel {
		case "ERROR":
			levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
		case "WARN":
			levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Orange
		case "INFO":
			levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86")) // Cyan
		case "DEBUG":
			levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // Gray
		default:
			levelStyle = lipgloss.NewStyle()
		}

		level := levelStyle.Render(fmt.Sprintf("[%-5s]", log.LogLevel))

		content.WriteString(fmt.Sprintf("%s %s %s\n",
			timestamp,
			level,
			log.LogLine,
		))
	}

	m.viewport.SetContent(content.String())
}

// View renders the log viewer
func (m LogViewerModel) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Loading logs...")
	}

	if m.error != nil {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("Error: %v", m.error))
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	header := headerStyle.Render(fmt.Sprintf("Logs: %s/%s", m.taskArn, m.container))

	// Status line
	statusItems := []string{
		fmt.Sprintf("Lines: %d", len(m.filteredLogs)),
	}

	if m.follow {
		statusItems = append(statusItems, "Following")
	}

	if m.searchTerm != "" {
		statusItems = append(statusItems, fmt.Sprintf("Filter: %s", m.searchTerm))
	}

	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(strings.Join(statusItems, " | "))

	// Search bar
	searchBar := m.searchBar.View()

	// Footer with shortcuts
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	shortcuts := []string{
		"[/] Search",
		"[F] Follow",
		"[r] Reload",
		"[g/G] Top/Bottom",
		"[f] Toggle View",
		"[Esc] Back",
	}

	footer := footerStyle.Render(strings.Join(shortcuts, "  "))

	// Combine all elements
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		status,
		searchBar,
		m.viewport.View(),
		footer,
	)
}
