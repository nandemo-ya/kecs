# KECS TUI Design Document

## Overview

The KECS Terminal User Interface (TUI) provides an interactive terminal-based interface for managing ECS resources. Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), it offers a keyboard-driven experience optimized for developers.

## Design Principles

1. **Keyboard-First**: All actions accessible via keyboard shortcuts
2. **Familiar Navigation**: Vim-like keybindings where appropriate
3. **Information Density**: Show relevant information without clutter
4. **Real-time Updates**: Live refresh of resource states
5. **Context Awareness**: Show relevant actions based on current view
6. **Responsive**: Adapt to terminal size changes

## Architecture

### Component Structure

```
┌─────────────────────────────────────────────────────┐
│ Header Bar                                          │
├─────────────────────────────────────────────────────┤
│ Navigation │ Resource List │ Details/Actions        │
│            │               │                         │
│ > Clusters │ cluster-1     │ Name: cluster-1         │
│   Services │ cluster-2     │ Status: ACTIVE          │
│   Tasks    │ cluster-3     │ Services: 5             │
│   TaskDefs │               │ Tasks: 12               │
│            │               │                         │
│            │               │ [Actions]               │
│            │               │ (c) Create Service      │
│            │               │ (d) Delete Cluster      │
│            │               │ (r) Refresh             │
├─────────────────────────────────────────────────────┤
│ Status Bar / Help                                   │
└─────────────────────────────────────────────────────┘
```

### Model-View-Update Pattern

Following Bubbletea's architecture:

1. **Model**: Holds application state
2. **View**: Renders the UI based on model
3. **Update**: Handles events and updates model

## Navigation Flow

```
Dashboard (Home)
├── Clusters View
│   ├── Cluster Details
│   │   ├── Services List
│   │   └── Container Instances
│   └── Create Cluster
├── Services View
│   ├── Service Details
│   │   ├── Tasks List
│   │   ├── Deployments
│   │   └── Events
│   └── Create Service
├── Tasks View
│   ├── Task Details
│   │   ├── Container Details
│   │   └── Logs Viewer
│   └── Run Task
└── Task Definitions View
    ├── Task Definition Details
    │   └── Revision History
    └── Register Task Definition
```

## Key Bindings

### Global
- `?` - Show help
- `q` - Quit/Back
- `ctrl+c` - Force quit
- `/` - Search
- `n/N` - Next/Previous search result
- `r` - Refresh current view
- `ctrl+l` - Clear and redraw

### Navigation
- `h/←` - Navigate left/back
- `j/↓` - Navigate down
- `k/↑` - Navigate up
- `l/→` - Navigate right/enter
- `g/G` - Go to top/bottom
- `Tab` - Switch panes

### Resource Actions
- `c` - Create new resource
- `d` - Delete selected resource
- `e` - Edit resource
- `v` - View details
- `Enter` - Select/Drill down

### View-Specific
- **Services**: `s` - Scale service, `u` - Update service
- **Tasks**: `l` - View logs, `x` - Stop task
- **Task Definitions**: `r` - Register new revision

## Color Scheme

Using Lipgloss for consistent styling:

- **Primary**: Blue (#3B82F6)
- **Success**: Green (#10B981)
- **Warning**: Yellow (#F59E0B)
- **Error**: Red (#EF4444)
- **Muted**: Gray (#6B7280)

Status indicators:
- `ACTIVE` / `RUNNING` - Green
- `PENDING` / `PROVISIONING` - Yellow
- `STOPPED` / `INACTIVE` - Gray
- `FAILED` / `ERROR` - Red

## Components

### 1. Header Bar
- Current context (cluster/region)
- Connection status
- Time/Last refresh

### 2. Navigation Pane
- Resource type selector
- Hierarchical view of resources
- Quick stats (counts)

### 3. List View
- Sortable columns
- Filterable
- Multi-select for batch operations
- Status indicators

### 4. Details View
- Resource metadata
- Current status
- Related resources
- Available actions
- Events/History

### 5. Status Bar
- Current selection
- Available shortcuts
- Error messages
- Progress indicators

## Real-time Updates

Two update mechanisms:

1. **Polling** (default): Refresh every 5 seconds
2. **WebSocket** (if available): Live updates

Visual indicators:
- Spinner during refresh
- "Last updated" timestamp
- Change highlighting

## Error Handling

- Non-blocking error messages
- Retry mechanisms
- Graceful degradation
- Clear error descriptions

## Accessibility

- High contrast mode
- Screen reader friendly output mode
- Customizable key bindings
- Terminal bell for notifications

## Configuration

```yaml
# ~/.kecs/tui.yaml
theme: default
refresh_interval: 5s
vim_mode: true
show_help_bar: true
confirm_destructive: true
log_level: info
```

## Implementation Phases

### Phase 1: Core Navigation (MVP)
- Basic layout and navigation
- Cluster and service views
- Read-only operations

### Phase 2: CRUD Operations
- Create/Delete resources
- Update configurations
- Basic forms

### Phase 3: Advanced Features
- Log viewing
- Task execution
- Batch operations
- Search and filtering

### Phase 4: Polish
- Themes and customization
- Export functionality
- Performance optimizations

## Testing Strategy

1. **Unit Tests**: Component logic
2. **Integration Tests**: API interactions
3. **Visual Tests**: Screenshot regression
4. **Manual Testing**: Terminal compatibility

## References

- [Bubbletea Examples](https://github.com/charmbracelet/bubbletea/tree/master/examples)
- [k9s UI](https://k9scli.io/) - Navigation inspiration
- [lazydocker](https://github.com/jesseduffield/lazydocker) - Layout inspiration