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

package help

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/keys"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/styles"
)

// Context represents the current context for help
type Context string

const (
	ContextDashboard     Context = "dashboard"
	ContextClusterList   Context = "cluster-list"
	ContextClusterDetail Context = "cluster-detail"
	ContextClusterCreate Context = "cluster-create"
	ContextServiceList   Context = "service-list"
	ContextServiceDetail Context = "service-detail"
	ContextServiceCreate Context = "service-create"
	ContextTaskList      Context = "task-list"
	ContextTaskDetail    Context = "task-detail"
	ContextTaskDefList   Context = "taskdef-list"
	ContextTaskDefDetail Context = "taskdef-detail"
	ContextSearch        Context = "search"
	ContextFilter        Context = "filter"
)

// ContextualHelp provides context-sensitive help
type ContextualHelp struct {
	context  Context
	showFull bool
	width    int
	height   int
}

// NewContextualHelp creates a new contextual help instance
func NewContextualHelp() *ContextualHelp {
	return &ContextualHelp{
		context: ContextDashboard,
	}
}

// SetContext sets the current context
func (h *ContextualHelp) SetContext(ctx Context) {
	h.context = ctx
}

// Toggle toggles between short and full help
func (h *ContextualHelp) Toggle() {
	h.showFull = !h.showFull
}

// SetSize sets the viewport size
func (h *ContextualHelp) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// Update handles tea messages
func (h *ContextualHelp) Update(msg tea.Msg) (*ContextualHelp, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height
	case tea.KeyMsg:
		if key.Matches(msg, keys.DefaultKeyMap().Help) {
			h.Toggle()
		}
	}
	return h, nil
}

// View renders the help based on current context
func (h *ContextualHelp) View() string {
	if h.showFull {
		return h.renderFullHelp()
	}
	return h.renderShortHelp()
}

// renderShortHelp renders the short help line
func (h *ContextualHelp) renderShortHelp() string {
	shortcuts := h.getShortcuts()
	
	// Build help string
	var parts []string
	for _, s := range shortcuts {
		part := fmt.Sprintf("%s %s", 
			styles.HelpKey.Render(s.Keys),
			styles.HelpDesc.Render(s.Desc),
		)
		parts = append(parts, part)
	}
	
	// Add help toggle hint
	parts = append(parts, fmt.Sprintf("%s %s",
		styles.HelpKey.Render("?"),
		styles.HelpDesc.Render("more help"),
	))
	
	return strings.Join(parts, " • ")
}

// renderFullHelp renders the full help overlay
func (h *ContextualHelp) renderFullHelp() string {
	var b strings.Builder
	
	// Title
	title := fmt.Sprintf("Help - %s", h.getContextTitle())
	b.WriteString(styles.Header.Render(title))
	b.WriteString("\n\n")
	
	// Context description
	b.WriteString(styles.Info.Render(h.getContextDescription()))
	b.WriteString("\n\n")
	
	// Key bindings by category
	categories := h.getHelpCategories()
	for _, cat := range categories {
		if len(cat.Bindings) == 0 {
			continue
		}
		
		// Category title
		b.WriteString(styles.ListTitle.Render(cat.Name))
		b.WriteString("\n")
		
		// Bindings
		for _, binding := range cat.Bindings {
			b.WriteString(fmt.Sprintf("  %s  %s\n",
				styles.HelpKey.Width(12).Render(binding.Keys),
				styles.HelpDesc.Render(binding.Desc),
			))
		}
		b.WriteString("\n")
	}
	
	// Close help hint
	b.WriteString("\n")
	b.WriteString(styles.Info.Render("Press ? or ESC to close help"))
	
	// Style the whole help panel
	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.GetStatusStyle("ACTIVE").GetForeground()).
		Padding(1, 2).
		MaxWidth(80).
		MaxHeight(h.height - 4)
	
	return helpStyle.Render(b.String())
}

// Shortcut represents a keyboard shortcut
type Shortcut struct {
	Keys string
	Desc string
}

// Category represents a help category
type Category struct {
	Name     string
	Bindings []Shortcut
}

// getShortcuts returns context-specific shortcuts for the help line
func (h *ContextualHelp) getShortcuts() []Shortcut {
	switch h.context {
	case ContextClusterList, ContextServiceList, ContextTaskList, ContextTaskDefList:
		return []Shortcut{
			{Keys: "↑↓", Desc: "navigate"},
			{Keys: "enter", Desc: "select"},
			{Keys: "n", Desc: "create"},
			{Keys: "/", Desc: "search"},
			{Keys: "f", Desc: "filter"},
			{Keys: "r", Desc: "refresh"},
		}
	case ContextClusterDetail, ContextServiceDetail, ContextTaskDetail, ContextTaskDefDetail:
		return []Shortcut{
			{Keys: "↑↓", Desc: "scroll"},
			{Keys: "esc", Desc: "back"},
			{Keys: "r", Desc: "refresh"},
		}
	case ContextClusterCreate, ContextServiceCreate:
		return []Shortcut{
			{Keys: "tab", Desc: "next field"},
			{Keys: "shift+tab", Desc: "prev field"},
			{Keys: "ctrl+s", Desc: "submit"},
			{Keys: "esc", Desc: "cancel"},
		}
	case ContextSearch:
		return []Shortcut{
			{Keys: "type", Desc: "search"},
			{Keys: "esc", Desc: "close"},
		}
	case ContextFilter:
		return []Shortcut{
			{Keys: "↑↓", Desc: "navigate"},
			{Keys: "space", Desc: "toggle"},
			{Keys: "a", Desc: "select all"},
			{Keys: "c", Desc: "clear"},
			{Keys: "esc", Desc: "apply"},
		}
	default:
		return []Shortcut{
			{Keys: "1-5", Desc: "switch view"},
			{Keys: "q", Desc: "quit"},
		}
	}
}

// getContextTitle returns the title for the current context
func (h *ContextualHelp) getContextTitle() string {
	switch h.context {
	case ContextDashboard:
		return "Dashboard"
	case ContextClusterList:
		return "Cluster List"
	case ContextClusterDetail:
		return "Cluster Details"
	case ContextClusterCreate:
		return "Create Cluster"
	case ContextServiceList:
		return "Service List"
	case ContextServiceDetail:
		return "Service Details"
	case ContextServiceCreate:
		return "Create Service"
	case ContextTaskList:
		return "Task List"
	case ContextTaskDetail:
		return "Task Details"
	case ContextTaskDefList:
		return "Task Definition List"
	case ContextTaskDefDetail:
		return "Task Definition Details"
	case ContextSearch:
		return "Search"
	case ContextFilter:
		return "Filter"
	default:
		return "Help"
	}
}

// getContextDescription returns a description for the current context
func (h *ContextualHelp) getContextDescription() string {
	switch h.context {
	case ContextDashboard:
		return "Overview of your ECS resources. Navigate between different resource types using number keys."
	case ContextClusterList:
		return "View and manage ECS clusters. Create new clusters, search by name, or filter by status."
	case ContextClusterDetail:
		return "Detailed information about a specific cluster including services, tasks, and container instances."
	case ContextClusterCreate:
		return "Create a new ECS cluster. Fill in the required fields and submit to create."
	case ContextServiceList:
		return "View and manage ECS services. Services maintain the desired number of tasks running."
	case ContextServiceDetail:
		return "Detailed information about a service including task status, deployments, and configuration."
	case ContextServiceCreate:
		return "Create a new ECS service. Specify the task definition and desired task count."
	case ContextTaskList:
		return "View running and stopped tasks. Tasks are instances of your containerized applications."
	case ContextTaskDetail:
		return "Detailed information about a task including container status, resource usage, and logs."
	case ContextTaskDefList:
		return "View task definitions. Task definitions specify how containers should run."
	case ContextTaskDefDetail:
		return "Detailed information about a task definition including container definitions and settings."
	case ContextSearch:
		return "Search for resources by name or ARN. Results update as you type."
	case ContextFilter:
		return "Filter resources by status or other attributes. Use space to toggle selections."
	default:
		return "Terminal User Interface for managing ECS resources."
	}
}

// getHelpCategories returns categorized help for the current context
func (h *ContextualHelp) getHelpCategories() []Category {
	common := Category{
		Name: "Common",
		Bindings: []Shortcut{
			{Keys: "?", Desc: "Toggle this help"},
			{Keys: "q/ctrl+c", Desc: "Quit application"},
			{Keys: "1-5", Desc: "Switch between views"},
		},
	}
	
	navigation := Category{
		Name: "Navigation",
		Bindings: []Shortcut{
			{Keys: "↑/k", Desc: "Move up"},
			{Keys: "↓/j", Desc: "Move down"},
			{Keys: "←/h", Desc: "Move left"},
			{Keys: "→/l", Desc: "Move right"},
			{Keys: "pgup/ctrl+b", Desc: "Page up"},
			{Keys: "pgdn/ctrl+d", Desc: "Page down"},
			{Keys: "home/g", Desc: "Go to start"},
			{Keys: "end/G", Desc: "Go to end"},
		},
	}
	
	switch h.context {
	case ContextClusterList, ContextServiceList, ContextTaskList, ContextTaskDefList:
		return []Category{
			common,
			navigation,
			{
				Name: "Actions",
				Bindings: []Shortcut{
					{Keys: "enter", Desc: "View details"},
					{Keys: "n/ctrl+n", Desc: "Create new"},
					{Keys: "d/del", Desc: "Delete selected"},
					{Keys: "r/ctrl+r", Desc: "Refresh list"},
					{Keys: "/", Desc: "Search"},
					{Keys: "f", Desc: "Filter"},
					{Keys: "esc", Desc: "Clear search/filter"},
				},
			},
		}
	case ContextClusterDetail, ContextServiceDetail, ContextTaskDetail, ContextTaskDefDetail:
		return []Category{
			common,
			navigation,
			{
				Name: "Actions",
				Bindings: []Shortcut{
					{Keys: "esc", Desc: "Back to list"},
					{Keys: "r/ctrl+r", Desc: "Refresh"},
					{Keys: "e", Desc: "Edit"},
					{Keys: "d", Desc: "Delete"},
				},
			},
		}
	case ContextClusterCreate, ContextServiceCreate:
		return []Category{
			common,
			{
				Name: "Form Navigation",
				Bindings: []Shortcut{
					{Keys: "tab", Desc: "Next field"},
					{Keys: "shift+tab", Desc: "Previous field"},
				},
			},
			{
				Name: "Actions",
				Bindings: []Shortcut{
					{Keys: "ctrl+s", Desc: "Submit form"},
					{Keys: "esc", Desc: "Cancel and go back"},
				},
			},
		}
	case ContextSearch:
		return []Category{
			{
				Name: "Search",
				Bindings: []Shortcut{
					{Keys: "type", Desc: "Enter search query"},
					{Keys: "esc", Desc: "Clear search and close"},
					{Keys: "enter", Desc: "Apply search"},
				},
			},
		}
	case ContextFilter:
		return []Category{
			{
				Name: "Filter",
				Bindings: []Shortcut{
					{Keys: "↑/↓", Desc: "Navigate options"},
					{Keys: "space", Desc: "Toggle selection"},
					{Keys: "a", Desc: "Select/deselect all"},
					{Keys: "c", Desc: "Clear all selections"},
					{Keys: "esc", Desc: "Apply and close"},
				},
			},
		}
	default:
		return []Category{common, navigation}
	}
}