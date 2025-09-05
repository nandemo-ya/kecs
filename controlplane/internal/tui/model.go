package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/mock"
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
	ViewInstanceCreate
	ViewTaskDescribe
	ViewConfirmDialog
	ViewInstanceSwitcher
	ViewTaskDefinitionFamilies
	ViewTaskDefinitionRevisions
	ViewTaskDefinitionEditor
	ViewTaskDefinitionDiff
	ViewClusterCreate
)

// String returns the string representation of ViewType
func (v ViewType) String() string {
	switch v {
	case ViewInstances:
		return "Instances"
	case ViewClusters:
		return "Clusters"
	case ViewServices:
		return "Services"
	case ViewTasks:
		return "Tasks"
	case ViewLogs:
		return "Logs"
	case ViewHelp:
		return "Help"
	case ViewCommandPalette:
		return "Command Palette"
	case ViewInstanceCreate:
		return "Instance Create"
	case ViewTaskDescribe:
		return "Task Describe"
	case ViewConfirmDialog:
		return "Confirm Dialog"
	case ViewInstanceSwitcher:
		return "Instance Switcher"
	case ViewTaskDefinitionFamilies:
		return "Task Definition Families"
	case ViewTaskDefinitionRevisions:
		return "Task Definition Revisions"
	case ViewTaskDefinitionEditor:
		return "Task Definition Editor"
	case ViewTaskDefinitionDiff:
		return "Task Definition Diff"
	case ViewClusterCreate:
		return "Cluster Create"
	default:
		return "Unknown"
	}
}

// Instance represents a KECS instance
type Instance struct {
	Name       string
	Status     string
	Clusters   int
	Services   int
	Tasks      int
	APIPort    int
	AdminPort  int
	LocalStack bool
	Traefik    bool
	Age        time.Duration
	Selected   bool
}

// Cluster represents an ECS cluster
type Cluster struct {
	Name     string
	Status   string
	Region   string
	Services int
	Tasks    int
	Age      time.Duration
}

// Service represents an ECS service
type Service struct {
	Name    string
	Desired int
	Running int
	Pending int
	Status  string
	TaskDef string
	Age     time.Duration
}

// Task represents an ECS task
type Task struct {
	ID         string
	ARN        string
	Service    string
	Status     string
	Health     string
	CPU        float64
	Memory     string
	IP         string
	Age        time.Duration
	Containers []string
}

// LogEntry represents a log line
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Message   string
}

// TaskDefinitionFamily represents a task definition family
type TaskDefinitionFamily struct {
	Family         string
	LatestRevision int
	ActiveCount    int
	TotalCount     int
	LastUpdated    time.Time
}

// TaskDefinitionRevision represents a specific revision of a task definition
type TaskDefinitionRevision struct {
	Family    string
	Revision  int
	Status    string
	CPU       string
	Memory    string
	CreatedAt time.Time
	JSON      string // Complete task definition JSON
}

// TaskDefinitionEditor manages task definition JSON editing
type TaskDefinitionEditor struct {
	family        string
	baseRevision  *int   // Source revision for copy
	content       string // JSON being edited
	cursorLine    int
	cursorCol     int
	errors        []ValidationError
	mode          TaskDefEditorMode
	commandBuffer string // Buffer for command mode input
}

// ValidationError represents a JSON validation error
type ValidationError struct {
	Line    int
	Column  int
	Message string
}

// Model holds the application state
type Model struct {
	// View state
	currentView  ViewType
	previousView ViewType
	width        int
	height       int

	// Navigation state
	selectedInstance   string
	selectedCluster    string
	selectedService    string
	selectedTask       string
	selectedTaskDetail *TaskDetail // Detailed task information
	taskDescribeScroll int         // Scroll position for task describe view

	// List cursors
	instanceCursor int
	clusterCursor  int
	serviceCursor  int
	taskCursor     int
	logCursor      int

	// Data
	instances []Instance
	clusters  []Cluster
	services  []Service
	tasks     []Task
	logs      []LogEntry

	// UI state
	searchMode   bool
	searchQuery  string
	commandMode  bool
	commandInput string
	showHelp     bool
	err          error

	// Command palette
	commandPalette *CommandPalette

	// Instance form
	instanceForm *InstanceForm

	// Confirm dialog
	confirmDialog  *ConfirmDialog
	pendingCommand tea.Cmd // Command to execute after dialog confirmation

	// Instance switcher
	instanceSwitcher *InstanceSwitcher

	// Task Definition state
	taskDefFamilies       []TaskDefinitionFamily
	taskDefRevisions      []TaskDefinitionRevision
	selectedFamily        string
	selectedRevision      int
	taskDefFamilyCursor   int
	taskDefRevisionCursor int
	taskDefEditor         *TaskDefinitionEditor
	showTaskDefJSON       bool           // 2-column display flag
	taskDefJSONScroll     int            // JSON display scroll position
	taskDefDiffMode       bool           // Diff display mode
	diffRevision1         int            // Diff comparison target 1
	diffRevision2         int            // Diff comparison target 2
	taskDefJSONCache      map[int]string // Cache of loaded task definition JSONs by revision

	// Update control
	lastUpdate      time.Time
	refreshInterval time.Duration

	// Terminal
	ready bool

	// API client
	apiClient   api.Client
	useMockData bool

	// Clipboard notification
	clipboardMsg     string
	clipboardMsgTime time.Time

	// Log viewer
	logViewer          *LogViewerModel
	logViewerTaskArn   string
	logViewerContainer string
	logSplitView       bool // true for split-view, false for fullscreen

	// Spinner for long-running operations
	spinner         spinner.Model
	isDeleting      bool
	deletingMessage string

	// Cluster creation form
	clusterForm *ClusterForm

	// Service scaling
	serviceScaleDialog *ServiceScaleDialog
	scalingInProgress  bool
	scalingServiceName string
	scalingTargetCount int

	// Service updating
	serviceUpdateDialog *ServiceUpdateDialog
	updatingInProgress  bool
	updatingServiceName string
	updatingTaskDef     string

	// Key bindings registry
	keyBindings *KeyBindingsRegistry
}

// NewModel creates a new application model
func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		currentView:      ViewInstances,
		refreshInterval:  5 * time.Second,
		ready:            false,
		commandPalette:   NewCommandPalette(),
		useMockData:      true, // Default to mock data for now
		apiClient:        api.NewMockClient(),
		taskDefJSONCache: make(map[int]string),
		spinner:          s,
		keyBindings:      NewKeyBindingsRegistry(),
	}
}

// NewModelWithClient creates a new application model with a specific API client
func NewModelWithClient(client api.Client) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		currentView:      ViewInstances,
		refreshInterval:  5 * time.Second,
		ready:            false,
		commandPalette:   NewCommandPalette(),
		useMockData:      false,
		apiClient:        client,
		taskDefJSONCache: make(map[int]string),
		spinner:          s,
		keyBindings:      NewKeyBindingsRegistry(),
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tickCmd(),
		statusTickCmd(),
	}

	// Only load mock data if we're using mock mode
	if m.useMockData {
		cmds = append(cmds, mock.LoadAllData("", "", "", ""))
	} else {
		// Load real data from API
		cmds = append(cmds, m.loadDataFromAPI())
	}

	return tea.Batch(cmds...)
}

// Messages for tea.Model updates

type tickMsg time.Time

// statusTickMsg is sent periodically to update instance status
type statusTickMsg time.Time

// DataLoadedMsg contains loaded data
type DataLoadedMsg struct {
	Instances []Instance
	Clusters  []Cluster
	Services  []Service
	Tasks     []Task
	Logs      []LogEntry
}

// Commands

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// statusTickCmd creates a ticker for status updates
func statusTickCmd() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return statusTickMsg(t)
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
	case ViewTaskDefinitionFamilies:
		m.currentView = ViewClusters
		m.selectedFamily = ""
	case ViewTaskDefinitionRevisions:
		// Special handling for JSON view
		if m.showTaskDefJSON {
			// Just hide the JSON view
			m.showTaskDefJSON = false
		} else {
			// Go back to families view
			m.currentView = ViewTaskDefinitionFamilies
			m.selectedFamily = ""
			m.taskDefRevisionCursor = 0
		}
	}
}

func (m *Model) getCurrentListLength() int {
	switch m.currentView {
	case ViewInstances:
		return len(m.filterInstances(m.instances))
	case ViewClusters:
		return len(m.filterClusters(m.clusters))
	case ViewServices:
		return len(m.filterServices(m.services))
	case ViewTasks:
		return len(m.filterTasks(m.tasks))
	case ViewLogs:
		return len(m.filterLogs(m.logs))
	case ViewTaskDefinitionFamilies:
		return len(m.filterTaskDefFamilies(m.taskDefFamilies))
	case ViewTaskDefinitionRevisions:
		return len(m.taskDefRevisions)
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
	case ViewTaskDefinitionFamilies:
		return m.taskDefFamilyCursor
	case ViewTaskDefinitionRevisions:
		return m.taskDefRevisionCursor
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
	case ViewTaskDefinitionFamilies:
		if m.taskDefFamilyCursor > 0 {
			m.taskDefFamilyCursor--
		}
	case ViewTaskDefinitionRevisions:
		if m.taskDefRevisionCursor > 0 {
			m.taskDefRevisionCursor--
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
	case ViewTaskDefinitionFamilies:
		if m.taskDefFamilyCursor < maxIndex {
			m.taskDefFamilyCursor++
		}
	case ViewTaskDefinitionRevisions:
		if m.taskDefRevisionCursor < maxIndex {
			m.taskDefRevisionCursor++
		}
	}
}

// Getters for testing

// GetSelectedInstance returns the selected instance name
func (m *Model) GetSelectedInstance() string {
	return m.selectedInstance
}

// SetSelectedInstance sets the selected instance name
func (m *Model) SetSelectedInstance(instance string) {
	m.selectedInstance = instance
}

// IsHelpShown returns whether help is being shown
func (m *Model) IsHelpShown() bool {
	return m.showHelp
}

// GetCommandPalette returns the command palette
func (m *Model) GetCommandPalette() *CommandPalette {
	return m.commandPalette
}
