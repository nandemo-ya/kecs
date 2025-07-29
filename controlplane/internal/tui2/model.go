package tui2

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui2/mock"
)

// ViewType represents the current view in the TUI
type ViewType int

const (
	ViewInstances ViewType = iota
	ViewClusters
	ViewServices
	ViewTasks
	ViewLogs
	ViewHelp
	ViewCommandPalette
)

// Instance represents a KECS instance
type Instance struct {
	Name      string
	Status    string
	Clusters  int
	Services  int
	Tasks     int
	APIPort   int
	Age       time.Duration
	Selected  bool
}

// Cluster represents an ECS cluster
type Cluster struct {
	Name       string
	Status     string
	Services   int
	Tasks      int
	CPUUsed    float64
	CPUTotal   float64
	MemoryUsed string
	MemoryTotal string
	Namespace  string
	Age        time.Duration
}

// Service represents an ECS service
type Service struct {
	Name       string
	Desired    int
	Running    int
	Pending    int
	Status     string
	TaskDef    string
	Age        time.Duration
}

// Task represents an ECS task
type Task struct {
	ID         string
	Service    string
	Status     string
	Health     string
	CPU        float64
	Memory     string
	IP         string
	Age        time.Duration
}

// LogEntry represents a log line
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// Model holds the application state
type Model struct {
	// View state
	currentView     ViewType
	previousView    ViewType
	width           int
	height          int
	
	// Navigation state
	selectedInstance string
	selectedCluster  string
	selectedService  string
	selectedTask     string
	
	// List cursors
	instanceCursor  int
	clusterCursor   int
	serviceCursor   int
	taskCursor      int
	logCursor       int
	
	// Data
	instances       []Instance
	clusters        []Cluster
	services        []Service
	tasks           []Task
	logs            []LogEntry
	
	// UI state
	searchMode      bool
	searchQuery     string
	commandMode     bool
	commandInput    string
	showHelp        bool
	err             error
	
	// Update control
	lastUpdate      time.Time
	refreshInterval time.Duration
	
	// Terminal
	ready           bool
}

// NewModel creates a new application model
func NewModel() Model {
	return Model{
		currentView:     ViewInstances,
		refreshInterval: 5 * time.Second,
		ready:           false,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		mock.LoadAllData("", "", "", ""),
	)
}

// Messages for tea.Model updates

type tickMsg time.Time

// DataLoadedMsg contains loaded data
type DataLoadedMsg struct {
	Instances []Instance
	Clusters  []Cluster
	Services  []Service
	Tasks     []Task
	Logs      []LogEntry
}

type errMsg struct {
	err error
}

// Commands

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}


// Navigation helpers

func (m *Model) canGoBack() bool {
	return m.currentView != ViewInstances
}

func (m *Model) goBack() {
	switch m.currentView {
	case ViewClusters:
		m.currentView = ViewInstances
		m.selectedInstance = ""
	case ViewServices:
		m.currentView = ViewClusters
		m.selectedCluster = ""
	case ViewTasks:
		m.currentView = ViewServices
		m.selectedService = ""
	case ViewLogs:
		m.currentView = m.previousView
	}
}

func (m *Model) getCurrentListLength() int {
	switch m.currentView {
	case ViewInstances:
		return len(m.instances)
	case ViewClusters:
		return len(m.clusters)
	case ViewServices:
		return len(m.services)
	case ViewTasks:
		return len(m.tasks)
	case ViewLogs:
		return len(m.logs)
	default:
		return 0
	}
}

func (m *Model) getCurrentCursor() int {
	switch m.currentView {
	case ViewInstances:
		return m.instanceCursor
	case ViewClusters:
		return m.clusterCursor
	case ViewServices:
		return m.serviceCursor
	case ViewTasks:
		return m.taskCursor
	case ViewLogs:
		return m.logCursor
	default:
		return 0
	}
}

func (m *Model) moveCursorUp() {
	switch m.currentView {
	case ViewInstances:
		if m.instanceCursor > 0 {
			m.instanceCursor--
		}
	case ViewClusters:
		if m.clusterCursor > 0 {
			m.clusterCursor--
		}
	case ViewServices:
		if m.serviceCursor > 0 {
			m.serviceCursor--
		}
	case ViewTasks:
		if m.taskCursor > 0 {
			m.taskCursor--
		}
	case ViewLogs:
		if m.logCursor > 0 {
			m.logCursor--
		}
	}
}

func (m *Model) moveCursorDown() {
	maxIndex := m.getCurrentListLength() - 1
	switch m.currentView {
	case ViewInstances:
		if m.instanceCursor < maxIndex {
			m.instanceCursor++
		}
	case ViewClusters:
		if m.clusterCursor < maxIndex {
			m.clusterCursor++
		}
	case ViewServices:
		if m.serviceCursor < maxIndex {
			m.serviceCursor++
		}
	case ViewTasks:
		if m.taskCursor < maxIndex {
			m.taskCursor++
		}
	case ViewLogs:
		if m.logCursor < maxIndex {
			m.logCursor++
		}
	}
}