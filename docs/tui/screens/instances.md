# Instance Management Screen

## Overview

The Instance Management screen provides comprehensive control over KECS instances, allowing users to create, start, stop, destroy, and switch between instances directly from the TUI.

## Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ KECS Instances                    Current: dev-cluster │ 15:23:45│
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Instance Name         Status      Data       Ports              │
│ ─────────────────────────────────────────────────────────────  │
│ ▶ dev-cluster         ● Running   ✓ Present  API:8080 Admin:8081│
│   staging-cluster     ○ Stopped   ✓ Present  API:8090 Admin:8091│
│   test-env-1          ○ Stopped   ✗ No data  -                 │
│   prod-mirror         ● Running   ✓ Present  API:8100 Admin:8101│
│                                                                 │
│                                                                 │
│ Instance Details                                                │
│ ─────────────────────────────────────────────────────────────  │
│ Name: dev-cluster                                               │
│ Status: Running (2h 45m)                                        │
│ Data Directory: ~/.kecs/instances/dev-cluster/data             │
│ API Endpoint: http://localhost:8080                            │
│ Admin Endpoint: http://localhost:8081                          │
│ Created: 2025-01-28 13:00:00                                   │
│ Resources: 3 clusters, 12 services, 45 tasks                   │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [n] New  [Enter] Start/Resume  [s] Stop  [d] Destroy  [↵] Switch│
│ [r] Refresh  [?] Help  [q] Back to Dashboard                   │
└─────────────────────────────────────────────────────────────────┘
```

## Features

### Instance List
- **Visual Status Indicators**:
  - `●` Running (green)
  - `○` Stopped (yellow)
  - `✗` Destroyed/Not found (red)
- **Data Presence**: Shows if persistent data exists
- **Port Mappings**: Displays assigned API and Admin ports
- **Current Instance**: Highlighted with `▶` marker

### Instance Details
- Real-time status information
- Runtime duration for running instances
- Full path to data directory
- Endpoint URLs for easy access
- Creation timestamp
- Resource count summary

### Operations

#### Create New Instance (`n`)
- Opens instance creation form
- Options:
  - Name (auto-generated if empty)
  - API Port (default: auto-assign)
  - Admin Port (default: auto-assign)
  - Enable LocalStack (default: true)
  - Enable Traefik (default: true)
  - Dev Mode (default: false)

#### Start/Resume Instance (`Enter`)
- Starts stopped instance
- Resumes with existing data
- Shows progress indicator during startup
- Auto-switches to instance when ready

#### Stop Instance (`s`)
- Gracefully stops running instance
- Preserves all data
- Confirms action for current instance

#### Destroy Instance (`d`)
- Shows confirmation dialog
- Options:
  - Delete data (checkbox)
  - Force destroy (bypass confirmation)
- Cannot destroy current running instance

#### Switch Instance (`↵` or `Enter` on running instance)
- Switches TUI connection to selected instance
- Updates all views to show new instance data
- Shows connection status during switch

## Navigation

### From Other Screens
- Press `6` from any screen to access instances
- Press `i` for quick instance switch dialog

### Within Instance Screen
- `↑/↓` or `j/k`: Navigate instance list
- `Tab`: Switch between list and details
- `Enter`: Start stopped instance or switch to running instance
- `Esc`: Return to previous view

## Real-time Updates

- Instance status refreshes every 5 seconds
- Running time updates every second
- Resource counts update when switching instances
- Visual indicators for state changes

## Implementation Details

### Data Structure
```go
type Instance struct {
    Name         string
    Status       InstanceStatus  // Running, Stopped, NotFound
    Running      bool
    DataExists   bool
    APIPort      int
    AdminPort    int
    CreatedAt    time.Time
    StartedAt    *time.Time
    Resources    ResourceSummary
    DataDir      string
}

type InstanceStatus string

const (
    InstanceRunning   InstanceStatus = "running"
    InstanceStopped   InstanceStatus = "stopped"
    InstanceNotFound  InstanceStatus = "notfound"
)
```

### Integration Points

1. **K3dClusterManager**: Direct integration for instance operations
2. **Config Management**: Reads/writes instance configurations
3. **API Client**: Switches connection when changing instances
4. **Navigation**: Updates app state with current instance

### State Management

The instance view maintains:
- List of all instances
- Current selected instance
- Active instance (connected)
- Operation in progress flags
- Error messages
- Form states for creation

## Error Handling

- Port conflicts during creation
- Failed instance starts
- Connection errors during switch
- Data directory access issues
- Concurrent operation prevention

## Future Enhancements

1. **Instance Templates**: Save and reuse configurations
2. **Bulk Operations**: Start/stop multiple instances
3. **Import/Export**: Instance configuration sharing
4. **Resource Limits**: Set CPU/memory constraints
5. **Instance Cloning**: Duplicate existing instance
6. **Health Monitoring**: Detailed health metrics
7. **Log Viewer**: Integrated instance logs