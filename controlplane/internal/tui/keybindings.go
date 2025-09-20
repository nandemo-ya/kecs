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

package tui

// KeyAction represents an action triggered by a key binding
type KeyAction string

const (
	// Global actions
	ActionQuit     KeyAction = "quit"
	ActionBack     KeyAction = "back"
	ActionHelp     KeyAction = "help"
	ActionSearch   KeyAction = "search"
	ActionCommand  KeyAction = "command"
	ActionMoveUp   KeyAction = "move_up"
	ActionMoveDown KeyAction = "move_down"
	ActionRefresh  KeyAction = "refresh"
	ActionGoHome   KeyAction = "go_home"

	// Navigation actions
	ActionSelect                KeyAction = "select"
	ActionNavigateClusters      KeyAction = "nav_clusters"
	ActionNavigateServices      KeyAction = "nav_services"
	ActionNavigateTasks         KeyAction = "nav_tasks"
	ActionNavigateTaskDefs      KeyAction = "nav_task_defs"
	ActionNavigateAllTasks      KeyAction = "nav_all_tasks"
	ActionNavigateLoadBalancers KeyAction = "nav_load_balancers"
	ActionNavigateTargetGroups  KeyAction = "nav_target_groups"
	ActionNavigateListeners     KeyAction = "nav_listeners"

	// Instance actions
	ActionNewInstance    KeyAction = "new_instance"
	ActionToggleInstance KeyAction = "toggle_instance"
	ActionDeleteInstance KeyAction = "delete_instance"
	ActionSwitchInstance KeyAction = "switch_instance"

	// Cluster actions
	ActionCreateCluster KeyAction = "create_cluster"
	ActionDeleteCluster KeyAction = "delete_cluster"

	// Service actions
	ActionScaleService  KeyAction = "scale_service"
	ActionUpdateService KeyAction = "update_service"

	// Task actions
	ActionDescribeTask KeyAction = "describe_task"
	ActionStopTask     KeyAction = "stop_task"
	ActionRestartTask  KeyAction = "restart_task"

	// Task definition actions
	ActionNewTaskDef        KeyAction = "new_task_def"
	ActionCopyTaskDef       KeyAction = "copy_task_def"
	ActionEditTaskDef       KeyAction = "edit_task_def"
	ActionDeregisterTaskDef KeyAction = "deregister_task_def"
	ActionToggleJSON        KeyAction = "toggle_json"

	// Log actions
	ActionViewLogs        KeyAction = "view_logs"
	ActionToggleSplitView KeyAction = "toggle_split_view"
	ActionSaveLogs        KeyAction = "save_logs"

	// Utility actions
	ActionYank       KeyAction = "yank"
	ActionCopyJSON   KeyAction = "copy_json"
	ActionScrollUp   KeyAction = "scroll_up"
	ActionScrollDown KeyAction = "scroll_down"
	ActionPageUp     KeyAction = "page_up"
	ActionPageDown   KeyAction = "page_down"
	ActionHome       KeyAction = "home"
	ActionEnd        KeyAction = "end"
)

// KeyBinding represents a single key binding configuration
type KeyBinding struct {
	Keys        []string           // Multiple keys can trigger the same action (e.g., ["up", "k"])
	Description string             // Human-readable description for help menu
	Action      KeyAction          // The action to perform
	Global      bool               // Whether this is a global key binding
	Condition   func(m Model) bool // Optional condition for showing/enabling the binding
}

// KeyBindingsRegistry manages all key bindings for the application
type KeyBindingsRegistry struct {
	globalBindings map[string]KeyBinding              // key -> binding
	viewBindings   map[ViewType]map[string]KeyBinding // view -> key -> binding
}

// NewKeyBindingsRegistry creates and initializes a new key bindings registry
func NewKeyBindingsRegistry() *KeyBindingsRegistry {
	registry := &KeyBindingsRegistry{
		globalBindings: make(map[string]KeyBinding),
		viewBindings:   make(map[ViewType]map[string]KeyBinding),
	}

	// Register global key bindings
	registry.registerGlobalBindings()

	// Register view-specific key bindings
	registry.registerViewBindings()

	return registry
}

// registerGlobalBindings registers all global key bindings
func (r *KeyBindingsRegistry) registerGlobalBindings() {
	globalBindings := []KeyBinding{
		{Keys: []string{"ctrl+c"}, Description: "Quit", Action: ActionQuit, Global: true},
		{Keys: []string{"esc"}, Description: "Back", Action: ActionBack, Global: true},
		{Keys: []string{"?"}, Description: "Help", Action: ActionHelp, Global: true},
		{Keys: []string{"/"}, Description: "Search", Action: ActionSearch, Global: true},
		{Keys: []string{":"}, Description: "Command", Action: ActionCommand, Global: true},
		{Keys: []string{"up", "k"}, Description: "Move up", Action: ActionMoveUp, Global: true},
		{Keys: []string{"down", "j"}, Description: "Move down", Action: ActionMoveDown, Global: true},
		{Keys: []string{"r"}, Description: "Refresh", Action: ActionRefresh, Global: true},
		{Keys: []string{"h"}, Description: "Home", Action: ActionGoHome, Global: true},
		{Keys: []string{"ctrl+i"}, Description: "Switch instance", Action: ActionSwitchInstance, Global: true},
	}

	for _, binding := range globalBindings {
		for _, key := range binding.Keys {
			r.globalBindings[key] = binding
		}
	}
}

// registerViewBindings registers all view-specific key bindings
func (r *KeyBindingsRegistry) registerViewBindings() {
	// Instances view
	r.registerViewKeys(ViewInstances, []KeyBinding{
		{Keys: []string{"enter"}, Description: "Select", Action: ActionSelect},
		{Keys: []string{"n"}, Description: "New instance", Action: ActionNewInstance},
		{Keys: []string{"s"}, Description: "Start/Stop", Action: ActionToggleInstance},
		{Keys: []string{"d"}, Description: "Delete", Action: ActionDeleteInstance},
		{Keys: []string{"t"}, Description: "Task defs", Action: ActionNavigateTaskDefs,
			Condition: func(m Model) bool { return m.selectedInstance != "" }},
		{Keys: []string{"y"}, Description: "Yank name", Action: ActionYank},
	})

	// Clusters view
	r.registerViewKeys(ViewClusters, []KeyBinding{
		{Keys: []string{"enter"}, Description: "Select", Action: ActionSelect},
		{Keys: []string{"n"}, Description: "Create cluster", Action: ActionCreateCluster},
		{Keys: []string{"s"}, Description: "Services", Action: ActionNavigateServices},
		{Keys: []string{"t"}, Description: "Task defs", Action: ActionNavigateTaskDefs},
		{Keys: []string{"T"}, Description: "All tasks", Action: ActionNavigateAllTasks},
		{Keys: []string{"b"}, Description: "Load Balancers", Action: ActionNavigateLoadBalancers},
		{Keys: []string{"g"}, Description: "Target Groups", Action: ActionNavigateTargetGroups},
	})

	// Services view
	r.registerViewKeys(ViewServices, []KeyBinding{
		{Keys: []string{"enter"}, Description: "Select", Action: ActionSelect},
		{Keys: []string{"s"}, Description: "Scale", Action: ActionScaleService},
		{Keys: []string{"u"}, Description: "Update", Action: ActionUpdateService},
		{Keys: []string{"l"}, Description: "Logs", Action: ActionViewLogs},
		{Keys: []string{"t"}, Description: "Task defs", Action: ActionNavigateTaskDefs},
		{Keys: []string{"c"}, Description: "Clusters", Action: ActionNavigateClusters},
	})

	// Tasks view
	r.registerViewKeys(ViewTasks, []KeyBinding{
		{Keys: []string{"enter"}, Description: "Describe", Action: ActionDescribeTask},
		{Keys: []string{"s"}, Description: "Stop task", Action: ActionStopTask},
		{Keys: []string{"l"}, Description: "Logs", Action: ActionViewLogs},
		{Keys: []string{"t"}, Description: "Task defs", Action: ActionNavigateTaskDefs},
		{Keys: []string{"c"}, Description: "Clusters", Action: ActionNavigateClusters},
	})

	// Task Describe view
	r.registerViewKeys(ViewTaskDescribe, []KeyBinding{
		{Keys: []string{"l"}, Description: "View Logs", Action: ActionViewLogs},
		{Keys: []string{"r"}, Description: "Restart", Action: ActionRestartTask},
		{Keys: []string{"s"}, Description: "Stop", Action: ActionStopTask},
		{Keys: []string{"g"}, Description: "Go to top", Action: ActionHome},
		{Keys: []string{"G"}, Description: "Go to bottom", Action: ActionEnd},
		{Keys: []string{"ctrl+u", "pgup"}, Description: "Page up", Action: ActionPageUp},
		{Keys: []string{"ctrl+d", "pgdown"}, Description: "Page down", Action: ActionPageDown},
	})

	// Logs view
	r.registerViewKeys(ViewLogs, []KeyBinding{
		{Keys: []string{"f"}, Description: "Toggle split-view", Action: ActionToggleSplitView},
		{Keys: []string{"s"}, Description: "Save", Action: ActionSaveLogs},
	})

	// Task Definition Families view
	r.registerViewKeys(ViewTaskDefinitionFamilies, []KeyBinding{
		{Keys: []string{"enter"}, Description: "Select", Action: ActionSelect},
		{Keys: []string{"N"}, Description: "New", Action: ActionNewTaskDef},
		{Keys: []string{"C"}, Description: "Copy latest", Action: ActionCopyTaskDef},
	})

	// Task Definition Revisions view
	r.registerViewKeys(ViewTaskDefinitionRevisions, []KeyBinding{
		{Keys: []string{"enter"}, Description: "Toggle JSON", Action: ActionToggleJSON},
		{Keys: []string{"y"}, Description: "Yank name", Action: ActionYank},
		{Keys: []string{"e"}, Description: "Edit", Action: ActionEditTaskDef},
		{Keys: []string{"c"}, Description: "Copy JSON", Action: ActionCopyJSON},
		{Keys: []string{"d"}, Description: "Deregister", Action: ActionDeregisterTaskDef},
	})

	// Load Balancers view
	r.registerViewKeys(ViewLoadBalancers, []KeyBinding{
		{Keys: []string{"enter"}, Description: "View Listeners", Action: ActionNavigateListeners},
		{Keys: []string{"g"}, Description: "Target Groups", Action: ActionNavigateTargetGroups},
		{Keys: []string{"c"}, Description: "Clusters", Action: ActionNavigateClusters},
		{Keys: []string{"y"}, Description: "Yank ARN", Action: ActionYank},
	})

	// Target Groups view
	r.registerViewKeys(ViewTargetGroups, []KeyBinding{
		{Keys: []string{"enter"}, Description: "View Targets", Action: ActionSelect},
		{Keys: []string{"b"}, Description: "Load Balancers", Action: ActionNavigateLoadBalancers},
		{Keys: []string{"c"}, Description: "Clusters", Action: ActionNavigateClusters},
		{Keys: []string{"y"}, Description: "Yank ARN", Action: ActionYank},
	})

	// Listeners view
	r.registerViewKeys(ViewListeners, []KeyBinding{
		{Keys: []string{"enter"}, Description: "View Rules", Action: ActionSelect},
		{Keys: []string{"b"}, Description: "Load Balancers", Action: ActionNavigateLoadBalancers},
		{Keys: []string{"g"}, Description: "Target Groups", Action: ActionNavigateTargetGroups},
		{Keys: []string{"c"}, Description: "Clusters", Action: ActionNavigateClusters},
		{Keys: []string{"y"}, Description: "Yank ARN", Action: ActionYank},
	})
}

// registerViewKeys registers key bindings for a specific view
func (r *KeyBindingsRegistry) registerViewKeys(view ViewType, bindings []KeyBinding) {
	if r.viewBindings[view] == nil {
		r.viewBindings[view] = make(map[string]KeyBinding)
	}

	for _, binding := range bindings {
		for _, key := range binding.Keys {
			r.viewBindings[view][key] = binding
		}
	}
}

// GetGlobalAction returns the action for a global key binding
func (r *KeyBindingsRegistry) GetGlobalAction(key string) (KeyAction, bool) {
	binding, found := r.globalBindings[key]
	if found {
		return binding.Action, true
	}
	return "", false
}

// GetViewAction returns the action for a view-specific key binding
func (r *KeyBindingsRegistry) GetViewAction(view ViewType, key string) (KeyAction, bool) {
	if viewBindings, hasView := r.viewBindings[view]; hasView {
		if binding, found := viewBindings[key]; found {
			return binding.Action, true
		}
	}
	return "", false
}

// mergeUniqueKeys merges two key slices, removing duplicates while preserving order
func mergeUniqueKeys(existing, new []string) []string {
	keySet := make(map[string]bool)
	var result []string

	// Add existing keys first (preserving order)
	for _, key := range existing {
		if !keySet[key] {
			keySet[key] = true
			result = append(result, key)
		}
	}

	// Add new keys that aren't already present
	for _, key := range new {
		if !keySet[key] {
			keySet[key] = true
			result = append(result, key)
		}
	}

	return result
}

// GetGlobalBindings returns all global key bindings in a stable order
func (r *KeyBindingsRegistry) GetGlobalBindings() []KeyBinding {
	// Define the order of global actions for consistent display
	actionOrder := []KeyAction{
		ActionMoveUp,
		ActionMoveDown,
		ActionBack,
		ActionGoHome,
		ActionCommand,
		ActionSearch,
		ActionHelp,
		ActionRefresh,
		ActionSwitchInstance,
		ActionQuit,
	}

	// Create a unique set of bindings (deduplicate by action)
	uniqueBindings := make(map[KeyAction]KeyBinding)
	for _, binding := range r.globalBindings {
		if existing, found := uniqueBindings[binding.Action]; found {
			// Merge keys from duplicate bindings (removing duplicates)
			existing.Keys = mergeUniqueKeys(existing.Keys, binding.Keys)
			uniqueBindings[binding.Action] = existing
		} else {
			uniqueBindings[binding.Action] = binding
		}
	}

	// Convert to slice in stable order
	var result []KeyBinding
	for _, action := range actionOrder {
		if binding, found := uniqueBindings[action]; found {
			result = append(result, binding)
		}
	}

	// Add any remaining actions not in the order list
	for action, binding := range uniqueBindings {
		found := false
		for _, orderedAction := range actionOrder {
			if action == orderedAction {
				found = true
				break
			}
		}
		if !found {
			result = append(result, binding)
		}
	}

	return result
}

// GetViewBindings returns all key bindings for a specific view in a stable order
func (r *KeyBindingsRegistry) GetViewBindings(view ViewType) []KeyBinding {
	viewMap := r.viewBindings[view]
	if viewMap == nil {
		return nil
	}

	// Define the typical order of view actions for consistent display
	commonActionOrder := []KeyAction{
		ActionSelect,
		ActionDescribeTask,
		ActionNewInstance,
		ActionCreateCluster,
		ActionNewTaskDef,
		ActionToggleInstance,
		ActionStopTask,
		ActionScaleService,
		ActionUpdateService,
		ActionViewLogs,
		ActionToggleSplitView,
		ActionToggleJSON,
		ActionYank,
		ActionCopyJSON,
		ActionCopyTaskDef,
		ActionEditTaskDef,
		ActionDeleteInstance,
		ActionDeregisterTaskDef,
		ActionNavigateClusters,
		ActionNavigateServices,
		ActionNavigateTasks,
		ActionNavigateTaskDefs,
		ActionNavigateAllTasks,
		ActionNavigateLoadBalancers,
		ActionNavigateTargetGroups,
		ActionNavigateListeners,
		ActionRestartTask,
		ActionSaveLogs,
		ActionHome,
		ActionEnd,
		ActionPageUp,
		ActionPageDown,
		ActionScrollUp,
		ActionScrollDown,
	}

	// Create a unique set of bindings (deduplicate by action)
	uniqueBindings := make(map[KeyAction]KeyBinding)
	for _, binding := range viewMap {
		if existing, found := uniqueBindings[binding.Action]; found {
			// Merge keys from duplicate bindings (removing duplicates)
			existing.Keys = mergeUniqueKeys(existing.Keys, binding.Keys)
			uniqueBindings[binding.Action] = existing
		} else {
			uniqueBindings[binding.Action] = binding
		}
	}

	// Convert to slice in stable order
	var result []KeyBinding
	for _, action := range commonActionOrder {
		if binding, found := uniqueBindings[action]; found {
			result = append(result, binding)
		}
	}

	// Add any remaining actions not in the order list
	for action, binding := range uniqueBindings {
		found := false
		for _, orderedAction := range commonActionOrder {
			if action == orderedAction {
				found = true
				break
			}
		}
		if !found {
			result = append(result, binding)
		}
	}

	return result
}

// GetAllBindingsForView returns both global and view-specific bindings for a view
func (r *KeyBindingsRegistry) GetAllBindingsForView(view ViewType, model Model) ([]KeyBinding, []KeyBinding) {
	var globalBindings, viewBindings []KeyBinding

	// Get unique global bindings
	for _, binding := range r.GetGlobalBindings() {
		if binding.Condition == nil || binding.Condition(model) {
			globalBindings = append(globalBindings, binding)
		}
	}

	// Get unique view bindings
	for _, binding := range r.GetViewBindings(view) {
		if binding.Condition == nil || binding.Condition(model) {
			viewBindings = append(viewBindings, binding)
		}
	}

	return viewBindings, globalBindings
}

// FormatKeyString formats multiple keys into a display string
func FormatKeyString(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	if len(keys) == 1 {
		return formatSingleKey(keys[0])
	}
	// Format as "key1/key2" for multiple keys (more compact)
	result := formatSingleKey(keys[0])
	for i := 1; i < len(keys); i++ {
		result += "/" + formatSingleKey(keys[i])
	}
	return result
}

// formatSingleKey formats a single key for display
func formatSingleKey(key string) string {
	// Special formatting for common keys
	switch key {
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "left":
		return "←"
	case "right":
		return "→"
	case "enter":
		return "⏎"
	case "esc":
		return "ESC"
	case "ctrl+c":
		return "^C"
	case "ctrl+i":
		return "^I"
	case "ctrl+u":
		return "^U"
	case "ctrl+d":
		return "^D"
	case "pgup":
		return "PgUp"
	case "pgdown":
		return "PgDn"
	default:
		// For single letters and other keys, return as-is
		if len(key) == 1 {
			return key
		}
		// For longer keys, wrap in brackets
		return "<" + key + ">"
	}
}
