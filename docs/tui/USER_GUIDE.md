# KECS TUI User Guide

The KECS Terminal User Interface (TUI) provides an interactive, keyboard-driven interface for managing ECS resources. This guide covers all features and keyboard shortcuts.

## Starting the TUI

To launch the TUI, run:

```bash
kecs
```

You can specify a custom endpoint:

```bash
kecs --endpoint http://remote-kecs:8080
```

## Navigation

The TUI consists of multiple views that you can switch between:

- **Dashboard** (Press `1`): Overview of all resources
- **Clusters** (Press `2`): Manage ECS clusters  
- **Services** (Press `3`): Manage ECS services
- **Tasks** (Press `4`): View running and stopped tasks
- **Task Definitions** (Press `5`): Manage task definitions

## Common Keyboard Shortcuts

### Navigation Keys
- `↑`/`k`: Move up
- `↓`/`j`: Move down
- `←`/`h`: Move left
- `→`/`l`: Move right
- `PgUp`/`Ctrl+b`: Page up
- `PgDn`/`Ctrl+d`: Page down
- `Home`/`g`: Go to start
- `End`/`G`: Go to end

### Action Keys
- `Enter`: Select/View details
- `Esc`: Go back/Cancel
- `r`/`Ctrl+r`: Refresh data
- `q`/`Ctrl+c`: Quit application
- `?`: Toggle help (context-sensitive)

## Features by View

### Clusters View

#### List View
- `n`: Create new cluster
- `d`: Delete selected cluster (TODO)
- `/`: Search clusters by name or ARN
- `f`: Filter clusters by status
- `Enter`: View cluster details

#### Create Cluster Form
- `Tab`: Next field
- `Shift+Tab`: Previous field
- `Ctrl+s`: Submit form
- `Esc`: Cancel

#### Search Mode
- Type to search in real-time
- `Esc`: Clear search and exit search mode

#### Filter Mode
- `↑`/`↓`: Navigate filter options
- `Space`: Toggle selection
- `a`: Select/deselect all
- `c`: Clear all selections
- `Esc`: Apply filters and close

### Services View

#### List View
- `n`: Create new service
- `d`: Delete selected service (TODO)
- `/`: Search services by name or ARN
- `f`: Filter services by status
- `Enter`: View service details

#### Create Service Form
- Fill in required fields:
  - Service name
  - Cluster (defaults to first available)
  - Task definition
  - Desired count
- `Tab`/`Shift+Tab`: Navigate fields
- `Ctrl+s`: Submit
- `Esc`: Cancel

### Tasks View

#### List View
- `/`: Search tasks by ID or task definition
- `f`: Filter tasks by status (RUNNING, PENDING, STOPPED)
- `Enter`: View task details
- Task list shows:
  - Task ID
  - Status with color coding
  - Task definition
  - Started by
  - Created time
  - CPU/Memory allocation

### Task Definitions View

#### List View
- `/`: Search by family name or revision
- `f`: Filter by status (ACTIVE, INACTIVE)
- `Enter`: View task definition details
- List shows:
  - Family name
  - Revision
  - Status
  - Compatibility
  - CPU/Memory requirements
  - Container count

## Search and Filter

### Search Feature (`/`)
- Real-time search as you type
- Case-insensitive
- Searches multiple fields depending on context:
  - Clusters: name, ARN
  - Services: name, ARN
  - Tasks: ID, task definition name
  - Task Definitions: family, revision

### Filter Feature (`f`)
- Multi-select status filters
- Filters are additive (OR logic within same category)
- Combined with search for refined results
- Shows count of filtered items (e.g., "Showing 5 of 10")

## Status Color Coding

The TUI uses colors to indicate resource status:

- **Green**: RUNNING, ACTIVE, HEALTHY
- **Yellow**: PENDING, PROVISIONING, ACTIVATING
- **Red**: STOPPED, INACTIVE, FAILED, DRAINING, DEREGISTERING
- **Gray**: UNKNOWN or other statuses

## Detail Views

When viewing details of a resource (by pressing Enter):

- Comprehensive information about the selected resource
- Formatted for easy reading
- Press `Esc` to return to the list view
- Press `r` to refresh the details

## Context-Sensitive Help

Press `?` at any time to view help specific to your current context:

- Shows available keyboard shortcuts
- Provides context-specific guidance
- Displays full help overlay
- Press `?` or `Esc` to close help

## Tips and Best Practices

1. **Auto-refresh**: Lists automatically refresh every 30 seconds
2. **Manual refresh**: Press `r` anytime to refresh immediately
3. **Efficient navigation**: Use `g` and `G` to quickly jump to start/end of lists
4. **Quick filtering**: Combine search (`/`) and filter (`f`) for precise results
5. **Keyboard-first**: All features are accessible via keyboard shortcuts

## Troubleshooting

### TUI doesn't start
- Ensure KECS server is running
- Check the endpoint is correct
- Verify network connectivity

### Data not updating
- Press `r` to manually refresh
- Check server logs for any errors
- Ensure you have proper permissions

### Display issues
- Ensure terminal supports UTF-8
- Try resizing terminal window
- Check terminal color support

## Future Features

The following features are planned for future releases:

- Edit existing resources
- Delete resources with confirmation
- Task logs viewing
- Service scaling operations
- Batch operations
- Export functionality
- Custom key bindings
- Theme customization