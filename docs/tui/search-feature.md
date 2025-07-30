# TUI v2 Search Feature

## Overview

The TUI v2 includes a powerful search functionality that allows users to filter resources across all views.

## Usage

### Activating Search Mode

Press `/` in any view to activate search mode. The footer will change to show:
```
Search: [your query]_ [Enter] Apply [Esc] Cancel
```

### Search Behavior

- **Real-time filtering**: Results are filtered as you type
- **Case-insensitive**: All searches are case-insensitive
- **Partial matching**: Searches match partial strings (e.g., "prod" matches "production")

### What is Searched

#### Instances View
- Instance name
- Status (ACTIVE, STOPPED)
- API port number

#### Clusters View
- Cluster name
- Status
- Namespace

#### Services View
- Service name
- Status (ACTIVE, INACTIVE, UPDATING, etc.)
- Task definition name

#### Tasks View
- Task ID
- Service name
- Status (RUNNING, PENDING, STOPPING, etc.)
- Health status (HEALTHY, UNHEALTHY)
- IP address

#### Logs View
- Log level (INFO, WARN, ERROR, DEBUG)
- Log message content

### Search Indicators

When a search is active:
1. A yellow search indicator appears at the bottom of the resource list: `[Search: your query]`
2. The scroll indicator shows the filtered count: `[Showing 1-10 of 15 instances]`
3. The cursor automatically resets to the first item when search results change

### Exiting Search Mode

- **Enter**: Apply the search and exit search mode (filter remains active)
- **Esc**: Clear the search and exit search mode

### Clearing Search

To clear an active search filter:
1. Press `/` to enter search mode
2. Press `Esc` to clear and exit

## Implementation Details

The search functionality is implemented in:
- `search.go`: Core filtering logic
- `app.go`: Search input handling
- `render.go`: UI updates for filtered results

Each view maintains its own filtered results, and the cursor position is automatically adjusted when the filtered list changes.