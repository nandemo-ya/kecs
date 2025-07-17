package bubbletea

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TaskStatus represents the status of a task
type TaskStatus int

const (
	TaskStatusPending TaskStatus = iota
	TaskStatusRunning
	TaskStatusCompleted
	TaskStatusFailed
)

// Task represents a single task being tracked
type Task struct {
	ID          string
	Name        string
	Status      TaskStatus
	Progress    float64
	Message     string
	StartTime   time.Time
	EndTime     time.Time
	Error       error
}

// LogEntry represents a log message
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// Model represents the Bubble Tea model for progress tracking
type Model struct {
	title      string
	tasks      map[string]*Task
	taskOrder  []string // Maintain task order
	logs       []LogEntry
	
	// UI components
	progressBars map[string]progress.Model
	logViewport  viewport.Model
	
	// Layout
	width       int
	height      int
	splitRatio  float64 // How much of the screen is for progress (0.5 = 50%)
	
	// Styling
	titleStyle    lipgloss.Style
	taskStyle     lipgloss.Style
	logStyle      lipgloss.Style
	borderStyle   lipgloss.Style
	
	// State
	startTime    time.Time
	completed    bool
	showLogs     bool
}

// New creates a new progress model
func New(title string) Model {
	return Model{
		title:        title,
		tasks:        make(map[string]*Task),
		taskOrder:    make([]string, 0),
		logs:         make([]LogEntry, 0),
		progressBars: make(map[string]progress.Model),
		splitRatio:   0.5,
		startTime:    time.Now(),
		showLogs:     true,
		
		// Styling
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			MarginBottom(1),
		
		taskStyle: lipgloss.NewStyle().
			PaddingLeft(2),
		
		logStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		
		borderStyle: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
	)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Update viewport size
		headerHeight := 4 // Title + borders
		progressHeight := int(float64(m.height-headerHeight) * m.splitRatio)
		logHeight := m.height - headerHeight - progressHeight - 1 // -1 for separator
		
		m.logViewport = viewport.New(m.width-4, logHeight) // -4 for borders
		m.logViewport.SetContent(m.renderLogs())
		
	case tickMsg:
		// Update progress bars animation
		for id, bar := range m.progressBars {
			if task, ok := m.tasks[id]; ok && task.Status == TaskStatusRunning {
				newBar, cmd := bar.Update(msg)
				if newProgressBar, ok := newBar.(progress.Model); ok {
					m.progressBars[id] = newProgressBar
				}
				cmds = append(cmds, cmd)
			}
		}
		cmds = append(cmds, tickCmd())
		
	case AddTaskMsg:
		m.addTask(msg.ID, msg.Name, msg.Total)
		
	case UpdateTaskMsg:
		m.updateTask(msg.ID, msg.Progress, msg.Message, msg.Status)
		
	case AddLogMsg:
		m.addLog(msg.Level, msg.Message)
		m.logViewport.SetContent(m.renderLogs())
		m.logViewport.GotoBottom()
		
	case CompleteMsg:
		m.completed = true
		
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "l":
			m.showLogs = !m.showLogs
		}
		
		// Handle viewport scrolling
		var cmd tea.Cmd
		m.logViewport, cmd = m.logViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the model
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	var sections []string

	// Title with elapsed time
	elapsed := time.Since(m.startTime).Round(time.Second)
	title := m.titleStyle.Render(fmt.Sprintf("🚀 %s (%s)", m.title, elapsed))
	sections = append(sections, title)

	// Progress section
	progressSection := m.renderProgress()
	sections = append(sections, progressSection)

	// Logs section (if enabled)
	if m.showLogs {
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(strings.Repeat("─", m.width))
		sections = append(sections, separator)
		
		logSection := m.renderLogSection()
		sections = append(sections, logSection)
	}

	// Footer
	footer := m.renderFooter()
	sections = append(sections, footer)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// Helper methods

func (m *Model) addTask(id, name string, total float64) {
	task := &Task{
		ID:        id,
		Name:      name,
		Status:    TaskStatusPending,
		Progress:  0,
		StartTime: time.Now(),
	}
	
	m.tasks[id] = task
	m.taskOrder = append(m.taskOrder, id)
	
	// Create progress bar
	bar := progress.New(progress.WithDefaultGradient())
	bar.Width = m.width - 40 // Leave room for labels
	m.progressBars[id] = bar
}

func (m *Model) updateTask(id string, prog float64, message string, status TaskStatus) {
	if task, ok := m.tasks[id]; ok {
		task.Progress = prog
		task.Message = message
		task.Status = status
		
		if status == TaskStatusCompleted || status == TaskStatusFailed {
			task.EndTime = time.Now()
		}
		
		// Update progress bar percentage
		// The progress bar will animate to the new percentage in its Update method
		if _, ok := m.progressBars[id]; ok {
			// We don't update the bar directly here.
			// Instead, we'll update it when rendering
		}
	}
}

func (m *Model) addLog(level, message string) {
	m.logs = append(m.logs, LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	})
}

func (m Model) renderProgress() string {
	var lines []string
	
	for _, id := range m.taskOrder {
		task := m.tasks[id]
		line := m.renderTask(task)
		lines = append(lines, line)
	}
	
	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	
	// Wrap in a box
	box := m.borderStyle.
		Width(m.width - 2).
		Render(content)
	
	return box
}

func (m Model) renderTask(task *Task) string {
	var icon string
	var color lipgloss.Color
	
	switch task.Status {
	case TaskStatusPending:
		icon = "⏳"
		color = lipgloss.Color("240")
	case TaskStatusRunning:
		icon = "🔄"
		color = lipgloss.Color("214")
	case TaskStatusCompleted:
		icon = "✅"
		color = lipgloss.Color("42")
	case TaskStatusFailed:
		icon = "❌"
		color = lipgloss.Color("196")
	}
	
	nameStyle := lipgloss.NewStyle().
		Foreground(color).
		Width(20)
	
	name := nameStyle.Render(fmt.Sprintf("%s %s", icon, task.Name))
	
	// Progress bar or status message
	var progressPart string
	if task.Status == TaskStatusRunning || task.Status == TaskStatusCompleted {
		if bar, ok := m.progressBars[task.ID]; ok {
			// Use ViewAs to render the bar with the current task progress
			progressPart = bar.ViewAs(task.Progress / 100.0)
		}
	} else if task.Status == TaskStatusFailed && task.Error != nil {
		progressPart = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(task.Error.Error())
	} else {
		progressPart = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(task.Message)
	}
	
	// Progress percentage
	percentage := lipgloss.NewStyle().
		Width(5).
		Align(lipgloss.Right).
		Render(fmt.Sprintf("%3.0f%%", task.Progress))
	
	return m.taskStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Left, name, " ", progressPart, " ", percentage),
	)
}

func (m Model) renderLogs() string {
	var lines []string
	
	for _, log := range m.logs {
		timestamp := log.Timestamp.Format("15:04:05")
		
		var levelColor lipgloss.Color
		switch log.Level {
		case "ERROR":
			levelColor = lipgloss.Color("196")
		case "WARN":
			levelColor = lipgloss.Color("214")
		case "INFO":
			levelColor = lipgloss.Color("86")
		default:
			levelColor = lipgloss.Color("240")
		}
		
		line := fmt.Sprintf(
			"%s %s %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(timestamp),
			lipgloss.NewStyle().Foreground(levelColor).Width(5).Render(log.Level),
			log.Message,
		)
		
		lines = append(lines, line)
	}
	
	return strings.Join(lines, "\n")
}

func (m Model) renderLogSection() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("📋 Logs")
	
	logs := m.borderStyle.
		Width(m.width - 2).
		Height(m.logViewport.Height + 2).
		Render(m.logViewport.View())
	
	return lipgloss.JoinVertical(lipgloss.Left, title, logs)
}

func (m Model) renderFooter() string {
	help := "l: toggle logs • q: quit"
	if m.showLogs {
		help = "↑/↓: scroll logs • " + help
	}
	
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(1).
		Render(help)
}

// Messages

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// AddTaskMsg adds a new task
type AddTaskMsg struct {
	ID    string
	Name  string
	Total float64
}

// UpdateTaskMsg updates a task's progress
type UpdateTaskMsg struct {
	ID       string
	Progress float64
	Message  string
	Status   TaskStatus
}

// AddLogMsg adds a log entry
type AddLogMsg struct {
	Level   string
	Message string
}

// CompleteMsg marks the entire operation as complete
type CompleteMsg struct{}