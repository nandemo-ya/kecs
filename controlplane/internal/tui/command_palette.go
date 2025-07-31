package tui

import (
	"fmt"
	"strings"
	"time"
)

// CommandCategory represents a category of commands
type CommandCategory string

const (
	CommandCategoryGeneral   CommandCategory = "General"
	CommandCategoryCreate    CommandCategory = "Create"
	CommandCategoryManage    CommandCategory = "Manage"
	CommandCategoryScale     CommandCategory = "Scale"
	CommandCategoryDebug     CommandCategory = "Debug"
	CommandCategoryExport    CommandCategory = "Export"
	CommandCategoryNavigation CommandCategory = "Navigation"
)

// Command represents a command in the palette
type Command struct {
	Name        string          // Command name
	Description string          // Command description
	Category    CommandCategory // Command category
	Shortcut    string          // Optional keyboard shortcut
	Aliases     []string        // Alternative names for the command
	Handler     func(*Model) (string, error) // Command handler
	Available   func(*Model) bool            // Check if command is available in current context
}

// CommandPalette manages the command palette state
type CommandPalette struct {
	commands       []Command
	filteredCmds   []Command
	selectedIndex  int
	query          string
	history        []string
	historyIndex   int
	maxHistory     int
	lastResult     string
	showResult     bool
	resultTimeout  time.Duration
	resultShownAt  time.Time
}

// GetFilteredCommands returns the currently filtered commands (for testing)
func (cp *CommandPalette) GetFilteredCommands() []Command {
	return cp.filteredCmds
}

// GetLastResult returns the last command result (for testing)
func (cp *CommandPalette) GetLastResult() string {
	return cp.lastResult
}

// IsShowingResult returns whether a result is being shown (for testing)
func (cp *CommandPalette) IsShowingResult() bool {
	return cp.showResult
}

// NewCommandPalette creates a new command palette
func NewCommandPalette() *CommandPalette {
	cp := &CommandPalette{
		maxHistory:    50,
		resultTimeout: 3 * time.Second,
	}
	cp.initCommands()
	
	// Set default Available function for commands that don't have one
	for i := range cp.commands {
		if cp.commands[i].Available == nil {
			cp.commands[i].Available = func(m *Model) bool { return true }
		}
	}
	
	return cp
}

// initCommands initializes all available commands
func (cp *CommandPalette) initCommands() {
	cp.commands = []Command{
		// General commands
		{
			Name:        "help",
			Description: "Show help documentation",
			Category:    CommandCategoryGeneral,
			Shortcut:    "?",
			Aliases:     []string{"h", "?"},
			Handler: func(m *Model) (string, error) {
				m.showHelp = true
				m.previousView = m.currentView
				m.currentView = ViewHelp
				return "Showing help", nil
			},
			Available: func(m *Model) bool { return true },
		},
		{
			Name:        "refresh",
			Description: "Refresh current data",
			Category:    CommandCategoryGeneral,
			Shortcut:    "R",
			Aliases:     []string{"r", "reload"},
			Handler: func(m *Model) (string, error) {
				return "Data refreshed", nil
			},
			Available: func(m *Model) bool { return true },
		},
		{
			Name:        "search",
			Description: "Search in current view",
			Category:    CommandCategoryGeneral,
			Shortcut:    "/",
			Aliases:     []string{"find", "f"},
			Handler: func(m *Model) (string, error) {
				m.searchMode = true
				m.searchQuery = ""
				return "Search mode activated", nil
			},
			Available: func(m *Model) bool { return true },
		},

		// Navigation commands
		{
			Name:        "goto instances",
			Description: "Navigate to instances view",
			Category:    CommandCategoryNavigation,
			Shortcut:    "i",
			Aliases:     []string{"instances", "gi"},
			Handler: func(m *Model) (string, error) {
				m.currentView = ViewInstances
				m.selectedInstance = ""
				return "Navigated to instances", nil
			},
			Available: func(m *Model) bool { return true },
		},
		{
			Name:        "goto clusters",
			Description: "Navigate to clusters view",
			Category:    CommandCategoryNavigation,
			Shortcut:    "c",
			Aliases:     []string{"clusters", "gc"},
			Handler: func(m *Model) (string, error) {
				if m.selectedInstance == "" {
					return "", fmt.Errorf("no instance selected")
				}
				m.currentView = ViewClusters
				m.selectedCluster = ""
				return fmt.Sprintf("Navigated to clusters in %s", m.selectedInstance), nil
			},
			Available: func(m *Model) bool { 
				if m == nil {
					return false
				}
				return m.selectedInstance != "" 
			},
		},
		{
			Name:        "goto services",
			Description: "Navigate to services view",
			Category:    CommandCategoryNavigation,
			Shortcut:    "s",
			Aliases:     []string{"services", "gs"},
			Handler: func(m *Model) (string, error) {
				if m.selectedCluster == "" {
					return "", fmt.Errorf("no cluster selected")
				}
				m.currentView = ViewServices
				m.selectedService = ""
				return fmt.Sprintf("Navigated to services in %s", m.selectedCluster), nil
			},
			Available: func(m *Model) bool { return m.selectedCluster != "" },
		},
		{
			Name:        "goto tasks",
			Description: "Navigate to tasks view",
			Category:    CommandCategoryNavigation,
			Shortcut:    "t",
			Aliases:     []string{"tasks", "gt"},
			Handler: func(m *Model) (string, error) {
				if m.selectedService == "" {
					return "", fmt.Errorf("no service selected")
				}
				m.currentView = ViewTasks
				return fmt.Sprintf("Navigated to tasks in %s", m.selectedService), nil
			},
			Available: func(m *Model) bool { return m.selectedService != "" },
		},
		{
			Name:        "logs",
			Description: "View logs for selected resource",
			Category:    CommandCategoryNavigation,
			Shortcut:    "l",
			Aliases:     []string{"log", "gl"},
			Handler: func(m *Model) (string, error) {
				switch m.currentView {
				case ViewServices:
					if len(m.services) > 0 && m.serviceCursor < len(m.services) {
						m.previousView = m.currentView
						m.currentView = ViewLogs
						return fmt.Sprintf("Viewing logs for service %s", m.services[m.serviceCursor].Name), nil
					}
				case ViewTasks:
					if len(m.tasks) > 0 && m.taskCursor < len(m.tasks) {
						m.selectedTask = m.tasks[m.taskCursor].ID
						m.previousView = m.currentView
						m.currentView = ViewLogs
						return fmt.Sprintf("Viewing logs for task %s", m.selectedTask), nil
					}
				}
				return "", fmt.Errorf("no loggable resource selected")
			},
			Available: func(m *Model) bool {
				return m.currentView == ViewServices || m.currentView == ViewTasks
			},
		},

		// Create commands
		{
			Name:        "create instance",
			Description: "Create a new KECS instance",
			Category:    CommandCategoryCreate,
			Shortcut:    "N",
			Aliases:     []string{"new instance", "ci"},
			Handler: func(m *Model) (string, error) {
				// Mock implementation
				newInstance := Instance{
					Name:     fmt.Sprintf("instance-%d", time.Now().Unix()),
					Status:   "ACTIVE",
					Clusters: 0,
					Services: 0,
					Tasks:    0,
					APIPort:  8080 + len(m.instances),
					Age:      0,
				}
				m.instances = append(m.instances, newInstance)
				return fmt.Sprintf("Created instance %s", newInstance.Name), nil
			},
			Available: func(m *Model) bool { return m.currentView == ViewInstances },
		},
		{
			Name:        "create cluster",
			Description: "Create a new ECS cluster",
			Category:    CommandCategoryCreate,
			Aliases:     []string{"new cluster", "cc"},
			Handler: func(m *Model) (string, error) {
				if m.selectedInstance == "" {
					return "", fmt.Errorf("no instance selected")
				}
				// Mock implementation
				newCluster := Cluster{
					Name:      fmt.Sprintf("cluster-%d", time.Now().Unix()),
					Status:    "ACTIVE",
					Services:  0,
					Tasks:     0,
					CPUUsed:   0,
					CPUTotal:  100,
					MemoryUsed: "0 GB",
					MemoryTotal: "16 GB",
					Namespace: m.selectedInstance,
					Age:       0,
				}
				m.clusters = append(m.clusters, newCluster)
				return fmt.Sprintf("Created cluster %s", newCluster.Name), nil
			},
			Available: func(m *Model) bool { return m.currentView == ViewClusters },
		},
		{
			Name:        "create service",
			Description: "Create a new ECS service",
			Category:    CommandCategoryCreate,
			Aliases:     []string{"new service", "cs"},
			Handler: func(m *Model) (string, error) {
				if m.selectedCluster == "" {
					return "", fmt.Errorf("no cluster selected")
				}
				// Mock implementation
				newService := Service{
					Name:    fmt.Sprintf("service-%d", time.Now().Unix()),
					Desired: 1,
					Running: 0,
					Pending: 1,
					Status:  "DEPLOYING",
					TaskDef: "task-def:1",
					Age:     0,
				}
				m.services = append(m.services, newService)
				return fmt.Sprintf("Created service %s", newService.Name), nil
			},
			Available: func(m *Model) bool { return m.currentView == ViewServices },
		},

		// Manage commands
		{
			Name:        "start",
			Description: "Start selected resource",
			Category:    CommandCategoryManage,
			Shortcut:    "S",
			Aliases:     []string{"run"},
			Handler: func(m *Model) (string, error) {
				switch m.currentView {
				case ViewInstances:
					if len(m.instances) > 0 && m.instanceCursor < len(m.instances) {
						m.instances[m.instanceCursor].Status = "ACTIVE"
						return fmt.Sprintf("Started instance %s", m.instances[m.instanceCursor].Name), nil
					}
				}
				return "", fmt.Errorf("no startable resource selected")
			},
			Available: func(m *Model) bool { return m.currentView == ViewInstances },
		},
		{
			Name:        "stop",
			Description: "Stop selected resource",
			Category:    CommandCategoryManage,
			Shortcut:    "x",
			Aliases:     []string{"halt"},
			Handler: func(m *Model) (string, error) {
				switch m.currentView {
				case ViewInstances:
					if len(m.instances) > 0 && m.instanceCursor < len(m.instances) {
						m.instances[m.instanceCursor].Status = "STOPPED"
						return fmt.Sprintf("Stopped instance %s", m.instances[m.instanceCursor].Name), nil
					}
				case ViewServices:
					if len(m.services) > 0 && m.serviceCursor < len(m.services) {
						m.services[m.serviceCursor].Status = "STOPPED"
						m.services[m.serviceCursor].Running = 0
						return fmt.Sprintf("Stopped service %s", m.services[m.serviceCursor].Name), nil
					}
				}
				return "", fmt.Errorf("no stoppable resource selected")
			},
			Available: func(m *Model) bool {
				return m.currentView == ViewInstances || m.currentView == ViewServices
			},
		},
		{
			Name:        "restart",
			Description: "Restart selected service",
			Category:    CommandCategoryManage,
			Shortcut:    "r",
			Aliases:     []string{"reload service"},
			Handler: func(m *Model) (string, error) {
				if m.currentView != ViewServices {
					return "", fmt.Errorf("not in services view")
				}
				if len(m.services) > 0 && m.serviceCursor < len(m.services) {
					service := &m.services[m.serviceCursor]
					service.Status = "RESTARTING"
					// Simulate restart
					go func() {
						time.Sleep(2 * time.Second)
						service.Status = "ACTIVE"
					}()
					return fmt.Sprintf("Restarting service %s", service.Name), nil
				}
				return "", fmt.Errorf("no service selected")
			},
			Available: func(m *Model) bool { return m.currentView == ViewServices },
		},
		{
			Name:        "delete",
			Description: "Delete selected resource",
			Category:    CommandCategoryManage,
			Shortcut:    "D",
			Aliases:     []string{"remove", "rm"},
			Handler: func(m *Model) (string, error) {
				switch m.currentView {
				case ViewInstances:
					if len(m.instances) > 0 && m.instanceCursor < len(m.instances) {
						name := m.instances[m.instanceCursor].Name
						m.instances = append(m.instances[:m.instanceCursor], m.instances[m.instanceCursor+1:]...)
						if m.instanceCursor >= len(m.instances) && m.instanceCursor > 0 {
							m.instanceCursor--
						}
						return fmt.Sprintf("Deleted instance %s", name), nil
					}
				case ViewClusters:
					if len(m.clusters) > 0 && m.clusterCursor < len(m.clusters) {
						name := m.clusters[m.clusterCursor].Name
						m.clusters = append(m.clusters[:m.clusterCursor], m.clusters[m.clusterCursor+1:]...)
						if m.clusterCursor >= len(m.clusters) && m.clusterCursor > 0 {
							m.clusterCursor--
						}
						return fmt.Sprintf("Deleted cluster %s", name), nil
					}
				case ViewServices:
					if len(m.services) > 0 && m.serviceCursor < len(m.services) {
						name := m.services[m.serviceCursor].Name
						m.services = append(m.services[:m.serviceCursor], m.services[m.serviceCursor+1:]...)
						if m.serviceCursor >= len(m.services) && m.serviceCursor > 0 {
							m.serviceCursor--
						}
						return fmt.Sprintf("Deleted service %s", name), nil
					}
				}
				return "", fmt.Errorf("no deletable resource selected")
			},
			Available: func(m *Model) bool {
				return m.currentView == ViewInstances || m.currentView == ViewClusters || m.currentView == ViewServices
			},
		},

		// Scale commands
		{
			Name:        "scale up",
			Description: "Scale up selected service",
			Category:    CommandCategoryScale,
			Aliases:     []string{"scale+", "su"},
			Handler: func(m *Model) (string, error) {
				if m.currentView != ViewServices {
					return "", fmt.Errorf("not in services view")
				}
				if len(m.services) > 0 && m.serviceCursor < len(m.services) {
					service := &m.services[m.serviceCursor]
					service.Desired++
					return fmt.Sprintf("Scaled %s to %d instances", service.Name, service.Desired), nil
				}
				return "", fmt.Errorf("no service selected")
			},
			Available: func(m *Model) bool { return m.currentView == ViewServices },
		},
		{
			Name:        "scale down",
			Description: "Scale down selected service",
			Category:    CommandCategoryScale,
			Aliases:     []string{"scale-", "sd"},
			Handler: func(m *Model) (string, error) {
				if m.currentView != ViewServices {
					return "", fmt.Errorf("not in services view")
				}
				if len(m.services) > 0 && m.serviceCursor < len(m.services) {
					service := &m.services[m.serviceCursor]
					if service.Desired > 0 {
						service.Desired--
						return fmt.Sprintf("Scaled %s to %d instances", service.Name, service.Desired), nil
					}
					return "", fmt.Errorf("service already at 0 instances")
				}
				return "", fmt.Errorf("no service selected")
			},
			Available: func(m *Model) bool { return m.currentView == ViewServices },
		},
		{
			Name:        "scale",
			Description: "Scale service to specific count",
			Category:    CommandCategoryScale,
			Aliases:     []string{"scale to"},
			Handler: func(m *Model) (string, error) {
				// This would typically parse a number from the command
				return "", fmt.Errorf("use 'scale <number>' to set specific count")
			},
			Available: func(m *Model) bool { return m.currentView == ViewServices },
		},

		// Debug commands
		{
			Name:        "describe",
			Description: "Show detailed information about selected resource",
			Category:    CommandCategoryDebug,
			Shortcut:    "d",
			Aliases:     []string{"desc", "info"},
			Handler: func(m *Model) (string, error) {
				switch m.currentView {
				case ViewTasks:
					if len(m.tasks) > 0 && m.taskCursor < len(m.tasks) {
						return fmt.Sprintf("Describing task %s", m.tasks[m.taskCursor].ID), nil
					}
				case ViewServices:
					if len(m.services) > 0 && m.serviceCursor < len(m.services) {
						return fmt.Sprintf("Describing service %s", m.services[m.serviceCursor].Name), nil
					}
				}
				return "", fmt.Errorf("no describable resource selected")
			},
			Available: func(m *Model) bool {
				return m.currentView == ViewServices || m.currentView == ViewTasks
			},
		},
		{
			Name:        "exec",
			Description: "Execute command in task container",
			Category:    CommandCategoryDebug,
			Aliases:     []string{"shell"},
			Handler: func(m *Model) (string, error) {
				if m.currentView != ViewTasks {
					return "", fmt.Errorf("not in tasks view")
				}
				if len(m.tasks) > 0 && m.taskCursor < len(m.tasks) {
					return fmt.Sprintf("Opening shell in task %s", m.tasks[m.taskCursor].ID), nil
				}
				return "", fmt.Errorf("no task selected")
			},
			Available: func(m *Model) bool { return m.currentView == ViewTasks },
		},

		// Export commands
		{
			Name:        "export json",
			Description: "Export current view data as JSON",
			Category:    CommandCategoryExport,
			Aliases:     []string{"ej"},
			Handler: func(m *Model) (string, error) {
				viewName := getViewName(m.currentView)
				return fmt.Sprintf("Exported %s data to %s.json", viewName, viewName), nil
			},
			Available: func(m *Model) bool { return true },
		},
		{
			Name:        "export yaml",
			Description: "Export current view data as YAML",
			Category:    CommandCategoryExport,
			Aliases:     []string{"ey"},
			Handler: func(m *Model) (string, error) {
				viewName := getViewName(m.currentView)
				return fmt.Sprintf("Exported %s data to %s.yaml", viewName, viewName), nil
			},
			Available: func(m *Model) bool { return true },
		},
		{
			Name:        "export csv",
			Description: "Export current view data as CSV",
			Category:    CommandCategoryExport,
			Aliases:     []string{"ec"},
			Handler: func(m *Model) (string, error) {
				viewName := getViewName(m.currentView)
				return fmt.Sprintf("Exported %s data to %s.csv", viewName, viewName), nil
			},
			Available: func(m *Model) bool { return true },
		},
	}
}

// FilterCommands filters commands based on query and context
func (cp *CommandPalette) FilterCommands(query string, model *Model) {
	// Ensure model is not nil
	if model == nil {
		cp.query = query
		cp.filteredCmds = []Command{}
		return
	}
	
	cp.query = query
	cp.filteredCmds = []Command{}
	
	queryLower := strings.ToLower(query)
	
	for _, cmd := range cp.commands {
		// Check if command is available in current context
		if cmd.Available != nil {
			// Safe call with nil check
			available := false
			if model != nil {
				available = cmd.Available(model)
			}
			if !available {
				continue
			}
		}
		
		// If no query, show all available commands
		if query == "" {
			cp.filteredCmds = append(cp.filteredCmds, cmd)
			continue
		}
		
		// Check if query matches command name or aliases
		if strings.Contains(strings.ToLower(cmd.Name), queryLower) {
			cp.filteredCmds = append(cp.filteredCmds, cmd)
			continue
		}
		
		// Check aliases
		for _, alias := range cmd.Aliases {
			if strings.Contains(strings.ToLower(alias), queryLower) {
				cp.filteredCmds = append(cp.filteredCmds, cmd)
				break
			}
		}
	}
	
	// Reset selection if needed
	if cp.selectedIndex >= len(cp.filteredCmds) {
		cp.selectedIndex = 0
	}
}

// ExecuteCommand executes the selected command
func (cp *CommandPalette) ExecuteCommand(model *Model) (string, error) {
	if len(cp.filteredCmds) == 0 {
		return "", fmt.Errorf("no matching commands")
	}
	
	if cp.selectedIndex >= len(cp.filteredCmds) {
		return "", fmt.Errorf("invalid command selection")
	}
	
	cmd := cp.filteredCmds[cp.selectedIndex]
	
	// Add to history
	cp.addToHistory(cmd.Name)
	
	// Execute the command
	result, err := cmd.Handler(model)
	if err != nil {
		return "", err
	}
	
	cp.lastResult = result
	cp.showResult = true
	cp.resultShownAt = time.Now()
	
	return result, nil
}

// ExecuteByName executes a command by name or alias
func (cp *CommandPalette) ExecuteByName(name string, model *Model) (string, error) {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	
	for _, cmd := range cp.commands {
		if !cmd.Available(model) {
			continue
		}
		
		if strings.ToLower(cmd.Name) == nameLower {
			cp.addToHistory(cmd.Name)
			result, err := cmd.Handler(model)
			if err == nil {
				cp.lastResult = result
				cp.showResult = true
				cp.resultShownAt = time.Now()
			}
			return result, err
		}
		
		for _, alias := range cmd.Aliases {
			if strings.ToLower(alias) == nameLower {
				cp.addToHistory(cmd.Name)
				result, err := cmd.Handler(model)
				if err == nil {
					cp.lastResult = result
					cp.showResult = true
					cp.resultShownAt = time.Now()
				}
				return result, err
			}
		}
	}
	
	return "", fmt.Errorf("unknown command: %s", name)
}

// Navigation methods
func (cp *CommandPalette) MoveUp() {
	if cp.selectedIndex > 0 {
		cp.selectedIndex--
	}
}

func (cp *CommandPalette) MoveDown() {
	if cp.selectedIndex < len(cp.filteredCmds)-1 {
		cp.selectedIndex++
	}
}

func (cp *CommandPalette) Reset() {
	cp.selectedIndex = 0
	cp.query = ""
	cp.FilterCommands("", nil)
}

// History methods
func (cp *CommandPalette) addToHistory(command string) {
	// Remove duplicates
	for i, h := range cp.history {
		if h == command {
			cp.history = append(cp.history[:i], cp.history[i+1:]...)
			break
		}
	}
	
	// Add to front
	cp.history = append([]string{command}, cp.history...)
	
	// Trim to max size
	if len(cp.history) > cp.maxHistory {
		cp.history = cp.history[:cp.maxHistory]
	}
	
	cp.historyIndex = -1
}

func (cp *CommandPalette) PreviousFromHistory() string {
	if len(cp.history) == 0 {
		return ""
	}
	
	if cp.historyIndex < len(cp.history)-1 {
		cp.historyIndex++
		return cp.history[cp.historyIndex]
	}
	
	return cp.history[cp.historyIndex]
}

func (cp *CommandPalette) NextFromHistory() string {
	if cp.historyIndex > 0 {
		cp.historyIndex--
		return cp.history[cp.historyIndex]
	}
	
	cp.historyIndex = -1
	return ""
}

// ShouldShowResult checks if the result should still be displayed
func (cp *CommandPalette) ShouldShowResult() bool {
	if !cp.showResult {
		return false
	}
	
	// Hide result after timeout
	if time.Since(cp.resultShownAt) > cp.resultTimeout {
		cp.showResult = false
		cp.lastResult = ""
		return false
	}
	
	return true
}

// Helper functions
func getViewName(view ViewType) string {
	switch view {
	case ViewInstances:
		return "instances"
	case ViewClusters:
		return "clusters"
	case ViewServices:
		return "services"
	case ViewTasks:
		return "tasks"
	case ViewLogs:
		return "logs"
	default:
		return "unknown"
	}
}