package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
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
	ViewLoadBalancers
	ViewTargetGroups
	ViewListeners
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
	case ViewLoadBalancers:
		return "Load Balancers"
	case ViewTargetGroups:
		return "Target Groups"
	case ViewListeners:
		return "Listeners"
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

// LoadBalancer represents an ELBv2 load balancer
type LoadBalancer struct {
	ARN       string
	Name      string
	DNSName   string
	Type      string // application, network, gateway
	Scheme    string // internet-facing, internal
	State     string // active, provisioning, failed, etc.
	VpcID     string
	Subnets   []string
	CreatedAt time.Time
}

// TargetGroup represents an ELBv2 target group
type TargetGroup struct {
	ARN                    string
	Name                   string
	Port                   int
	Protocol               string
	TargetType             string // instance, ip, lambda
	VpcID                  string
	HealthCheckEnabled     bool
	HealthCheckPath        string
	HealthyTargetCount     int
	UnhealthyTargetCount   int
	RegisteredTargetsCount int
	CreatedAt              time.Time
}

// Listener represents an ELBv2 listener
type Listener struct {
	ARN             string
	LoadBalancerARN string
	Port            int
	Protocol        string           // HTTP, HTTPS, TCP, TLS, UDP, TCP_UDP
	DefaultActions  []ListenerAction // List of default actions
	RuleCount       int
}

// ListenerAction represents an action for a listener
type ListenerAction struct {
	Type           string // forward, redirect, fixed-response, etc.
	TargetGroupArn string // If action is forward
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
	selectedContainer  int         // Selected container index in task describe view

	// List cursors
	instanceCursor int
	clusterCursor  int
	serviceCursor  int
	taskCursor     int
	logCursor      int

	// Instance carousel state
	instanceCarouselOffset int  // First visible instance in carousel
	maxVisibleInstances    int  // Maximum instances visible based on width
	autoSelectedInstance   bool // Flag for auto-selected single instance

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
	apiClient api.Client

	// Clipboard notification
	clipboardMsg     string
	clipboardMsgTime time.Time

	// Log viewer
	logViewer          *LogViewerModel
	logViewerTaskArn   string
	logViewerContainer string

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

	// ELBv2 state
	loadBalancers  []LoadBalancer
	targetGroups   []TargetGroup
	listeners      []Listener
	selectedLB     string // Selected load balancer ARN
	selectedTG     string // Selected target group ARN
	lbCursor       int
	tgCursor       int
	listenerCursor int
	elbv2SubView   int // 0=LoadBalancers, 1=TargetGroups, 2=Listeners
}

// NewModel creates a new application model
func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		currentView:      ViewInstances,
		refreshInterval:  2 * time.Second,
		ready:            false,
		commandPalette:   NewCommandPalette(),
		apiClient:        api.NewHTTPClient("http://localhost:5373"),
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
		refreshInterval:  2 * time.Second,
		ready:            false,
		commandPalette:   NewCommandPalette(),
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

	// Load real data from API
	cmds = append(cmds, m.loadDataFromAPI())

	// Also immediately update instance status for health checks
	cmds = append(cmds, m.updateInstanceStatusCmd())

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

// isInDialogView checks if the current view is a dialog that shouldn't be interrupted
func (m *Model) isInDialogView() bool {
	return m.currentView == ViewInstanceCreate ||
		m.currentView == ViewClusterCreate ||
		m.currentView == ViewTaskDefinitionEditor ||
		m.currentView == ViewConfirmDialog
}

func (m *Model) goBack() {
	switch m.currentView {
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

// Carousel helper methods

// getSelectedInstanceIndex returns the index of the currently selected instance
func (m *Model) getSelectedInstanceIndex() int {
	for i, inst := range m.instances {
		if inst.Name == m.selectedInstance {
			return i
		}
	}
	return 0
}

// calculateMaxVisibleInstances calculates how many instances can fit in the carousel
func (m *Model) calculateMaxVisibleInstances() {
	avgInstanceWidth := 20 // Average width per instance name + spacing
	indicatorWidth := 4    // Space for "◀ " and " ▶"
	padding := 4           // General padding

	availableWidth := m.width - indicatorWidth - padding
	m.maxVisibleInstances = availableWidth / avgInstanceWidth

	if m.maxVisibleInstances < 1 {
		m.maxVisibleInstances = 1
	}
}

// updateCarouselOffset adjusts the carousel offset to ensure selected instance is visible
func (m *Model) updateCarouselOffset() {
	selectedIdx := m.getSelectedInstanceIndex()

	// If selected instance is before the visible window, scroll left
	if selectedIdx < m.instanceCarouselOffset {
		m.instanceCarouselOffset = selectedIdx
	}
	// If selected instance is after the visible window, scroll right
	if selectedIdx >= m.instanceCarouselOffset+m.maxVisibleInstances {
		m.instanceCarouselOffset = selectedIdx - m.maxVisibleInstances + 1
	}
}

// switchToNextInstance switches to the next instance in the carousel
func (m *Model) switchToNextInstance() tea.Cmd {
	if len(m.instances) <= 1 {
		return nil
	}

	currentIdx := m.getSelectedInstanceIndex()
	nextIdx := (currentIdx + 1) % len(m.instances)
	m.selectedInstance = m.instances[nextIdx].Name
	m.updateCarouselOffset()

	// Reset view to clusters when switching instances
	m.currentView = ViewClusters
	m.clusterCursor = 0
	m.selectedCluster = ""
	m.serviceCursor = 0
	m.selectedService = ""
	m.taskCursor = 0
	m.selectedTask = ""

	// Load data for the new instance
	return m.loadDataFromAPI()
}

// switchToPreviousInstance switches to the previous instance in the carousel
func (m *Model) switchToPreviousInstance() tea.Cmd {
	if len(m.instances) <= 1 {
		return nil
	}

	currentIdx := m.getSelectedInstanceIndex()
	prevIdx := currentIdx - 1
	if prevIdx < 0 {
		prevIdx = len(m.instances) - 1
	}
	m.selectedInstance = m.instances[prevIdx].Name
	m.updateCarouselOffset()

	// Reset view to clusters when switching instances
	m.currentView = ViewClusters
	m.clusterCursor = 0
	m.selectedCluster = ""
	m.serviceCursor = 0
	m.selectedService = ""
	m.taskCursor = 0
	m.selectedTask = ""

	// Load data for the new instance
	return m.loadDataFromAPI()
}
