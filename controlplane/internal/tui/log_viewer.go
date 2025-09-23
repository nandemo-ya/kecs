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
		follow:       true, // Auto-enable follow mode
		streaming:    true, // Auto-enable streaming
	}
}

// Init initializes the log viewer
func (m LogViewerModel) Init() tea.Cmd {
	// Start with loading logs and polling
	return tea.Batch(
		m.loadLogs(),
		m.pollLogsCmd(),
	)
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

// pollLogsCmd polls for new logs periodically
func (m LogViewerModel) pollLogsCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return pollLogsTick{}
	})
}

// pollLogsTick is sent periodically to fetch new logs when in follow mode
type pollLogsTick struct{}

// Update handles messages
func (m LogViewerModel) Update(msg tea.Msg) (LogViewerModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Adjust viewport size - count actual UI elements:
		// header (1 line) + status (1 line) + search bar (1 line) + footer (1 line) = 4 lines total
		headerHeight := 1
		statusHeight := 1
		searchHeight := 1
		footerHeight := 1
		totalUIHeight := headerHeight + statusHeight + searchHeight + footerHeight

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - totalUIHeight

		// Ensure minimum height
		if m.viewport.Height < 5 {
			m.viewport.Height = 5
		}

		// Update search bar width
		m.searchBar.Width = msg.Width - 4

	case logsLoadedMsg:
		// Convert LogEntry to storage.TaskLog
		newLogs := make([]storage.TaskLog, len(msg.logs))
		for i, logEntry := range msg.logs {
			newLogs[i] = storage.TaskLog{
				Timestamp: logEntry.Timestamp,
				LogLevel:  logEntry.Level,
				LogLine:   logEntry.Message,
			}
		}

		// If we're in follow mode and streaming, merge new logs
		if m.follow && m.streaming && len(m.logs) > 0 {
			// Create a map of existing logs by timestamp+message to avoid duplicates
			existingLogs := make(map[string]bool)
			for _, log := range m.logs {
				key := log.Timestamp.Format(time.RFC3339Nano) + "|" + log.LogLine
				existingLogs[key] = true
			}

			// Append only new logs that don't exist
			for _, newLog := range newLogs {
				key := newLog.Timestamp.Format(time.RFC3339Nano) + "|" + newLog.LogLine
				if !existingLogs[key] {
					m.logs = append(m.logs, newLog)
				}
			}

			// Auto-scroll to bottom if new logs were added
			if m.follow {
				m.viewport.GotoBottom()
			}
		} else {
			// Initial load or not in follow mode - replace all logs
			m.logs = newLogs
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

	case pollLogsTick:
		// If we're in follow mode, fetch new logs
		if m.follow && m.streaming {
			// Load logs without setting loading flag to avoid UI flicker
			cmds = append(cmds, m.loadLogs())
			// Continue polling
			cmds = append(cmds, m.pollLogsCmd())
		}

	case tea.KeyMsg:
		// Handle search bar input first if focused
		if m.searchBar.Focused() {
			switch msg.String() {
			case "esc":
				// Unfocus search bar and clear search
				m.searchBar.Blur()
				m.searchTerm = ""
				m.searchBar.SetValue("")
				m.filterLogs()
				m.updateViewport()
			case "enter":
				// Just unfocus the search bar (filtering already applied in real-time)
				m.searchBar.Blur()
			default:
				// Let the search bar handle all other input
				var cmd tea.Cmd
				m.searchBar, cmd = m.searchBar.Update(msg)
				cmds = append(cmds, cmd)

				// Apply filter in real-time as user types
				if m.searchBar.Value() != m.searchTerm {
					m.searchTerm = m.searchBar.Value()
					m.filterLogs()
					m.updateViewport()
				}
			}
			return m, tea.Batch(cmds...)
		}

		// Handle normal key bindings when search bar is not focused
		switch msg.String() {
		case "ctrl+c":
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
			// Toggle auto-refresh mode (uppercase F to avoid conflict with split-view toggle)
			m.follow = !m.follow
			m.streaming = m.follow
			if m.follow {
				m.viewport.GotoBottom()
				// Resume polling if re-enabled
				cmds = append(cmds, m.pollLogsCmd())
			}

		case "g":
			// Go to top
			m.viewport.GotoTop()

		case "G":
			// Go to bottom
			m.viewport.GotoBottom()

		case "enter":
			if m.searchBar.Focused() {
				// Just unfocus the search bar (filtering already applied in real-time)
				m.searchBar.Blur()
			}
		}

		// Handle search bar input
		if m.searchBar.Focused() {
			var cmd tea.Cmd
			m.searchBar, cmd = m.searchBar.Update(msg)
			cmds = append(cmds, cmd)

			// Apply filter in real-time as user types
			if m.searchBar.Value() != m.searchTerm {
				m.searchTerm = m.searchBar.Value()
				m.filterLogs()
				m.updateViewport()
			}
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

	// Highlight style for search matches
	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("226")). // Yellow background
		Foreground(lipgloss.Color("16"))   // Black text

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

		// Highlight search term in log line if present
		logLine := log.LogLine
		if m.searchTerm != "" {
			// Case-insensitive highlighting
			searchLower := strings.ToLower(m.searchTerm)
			logLower := strings.ToLower(logLine)

			// Find all occurrences and highlight them
			var result strings.Builder
			lastEnd := 0

			for {
				idx := strings.Index(logLower[lastEnd:], searchLower)
				if idx == -1 {
					// No more matches, append the rest
					result.WriteString(logLine[lastEnd:])
					break
				}

				// Calculate actual position in original string
				actualIdx := lastEnd + idx

				// Append text before match
				result.WriteString(logLine[lastEnd:actualIdx])

				// Append highlighted match (preserve original case)
				highlighted := highlightStyle.Render(logLine[actualIdx : actualIdx+len(m.searchTerm)])
				result.WriteString(highlighted)

				lastEnd = actualIdx + len(m.searchTerm)
			}

			logLine = result.String()
		}

		content.WriteString(fmt.Sprintf("%s %s %s\n",
			timestamp,
			level,
			logLine,
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

	// Status line with scroll position
	statusItems := []string{
		fmt.Sprintf("Lines: %d", len(m.filteredLogs)),
	}

	// Add scroll position indicator
	scrollPercent := 0
	if m.viewport.TotalLineCount() > 0 {
		scrollPercent = int(float64(m.viewport.YOffset) / float64(max(1, m.viewport.TotalLineCount()-m.viewport.Height)) * 100)
		if scrollPercent > 100 {
			scrollPercent = 100
		}
		if scrollPercent < 0 {
			scrollPercent = 0
		}
	}
	statusItems = append(statusItems, fmt.Sprintf("Scroll: %d%%", scrollPercent))

	if m.follow {
		statusItems = append(statusItems, "Auto-refresh ON")
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
		"[F] Toggle Auto-refresh",
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

// IsSearchFocused returns whether the search bar is currently focused
func (m LogViewerModel) IsSearchFocused() bool {
	return m.searchBar.Focused()
}
