# ADR-0016: TUI Interface for KECS Resource Management

**Date:** 2025-07-29

## Status

Proposed

## Context

KECS users need an efficient way to manage and monitor their ECS resources across multiple instances. While the CLI provides comprehensive functionality, a Terminal User Interface (TUI) would significantly improve the user experience for:

- Navigating between multiple KECS instances (environments)
- Monitoring ECS clusters, services, and tasks in real-time
- Viewing logs and debugging applications
- Performing common operations without remembering complex CLI commands

The TUI should provide a k9s-like experience that is familiar to Kubernetes users while abstracting away the underlying k3d/Kubernetes implementation details.

## Decision

We will implement a TUI interface for KECS that provides:

1. **Hierarchical navigation** through Instances → Clusters → Services → Tasks
2. **Real-time monitoring** of resource states and metrics
3. **Keyboard-driven interface** with vim-style keybindings
4. **Multi-instance management** for development, staging, and production environments
5. **Integrated log viewing** with filtering and search capabilities

### Architecture

The TUI will be implemented as a separate command (`kecs tui`) that connects to KECS instances via their API endpoints.

```
┌─────────────────────────────────────────────────────────────────────────┐
│ KECS v1.0.0 | Environment: development | Status: ● Active              │
├─────────────────────────────────────────────────────────────────────────┤
│ [Instances] > development > [Clusters] > default > [Services]          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│                         Resource List View                              │
│                                                                         │
│                                                                         │
│                                                                         │
│                                                                         │
│                                                                         │
├─────────────────────────────────────────────────────────────────────────┤
│ [i] Instances [c] Clusters [s] Services [t] Tasks [/] Search [?] Help  │
└─────────────────────────────────────────────────────────────────────────┘
```

### Resource Hierarchy

```
Instance (KECS Environment)
└── Clusters (ECS Clusters)
    └── Services
        └── Tasks
            └── Containers
```

### Key Views

#### 1. Instances View (Top Level)
Displays all KECS instances with their status and resource utilization:

```
NAME            STATUS    CLUSTERS    SERVICES    TASKS      API PORT    AGE
development     ACTIVE    3           12          28         6443        5d
staging         ACTIVE    2           8           18         6444        3d
testing         STOPPED   1           0           0          6445        7d
production      ACTIVE    5           25          62         6446        1d
local           ACTIVE    1           3           5          6447        2h
```

#### 2. Clusters View
Shows ECS clusters within a selected instance:

```
Instance: development > Clusters

NAME            STATUS    SERVICES    TASKS    CPU        MEMORY      NAMESPACE              AGE
default         ACTIVE    5           12       2.4/8.0    3.2G/16G    default.us-east-1      2d
staging         ACTIVE    4           10       1.8/6.0    2.8G/12G    staging.us-east-1      1d
development     ACTIVE    3           6        0.6/4.0    1.5G/8G     development.us-east-1  5h
```

#### 3. Services View
Lists services within a selected cluster:

```
NAME            DESIRED    RUNNING    PENDING    STATUS     TASK DEF           AGE
web-service     3          3          0          ACTIVE     web-app:5          12h
api-service     2          2          0          ACTIVE     api:12             3d
worker          1          1          0          ACTIVE     worker:3           1h
db-migrate      0          0          0          INACTIVE   db-migrate:1       5m
```

#### 4. Tasks View
Displays tasks with their current status:

```
ID                      SERVICE         STATUS     HEALTH     CPU      MEMORY     IP              AGE
a1b2c3d4-1234-5678     web-service     RUNNING    HEALTHY    0.5      512M       10.0.1.5        2h
e5f6g7h8-9012-3456     web-service     RUNNING    HEALTHY    0.6      520M       10.0.1.6        2h
i9j0k1l2-3456-7890     api-service     RUNNING    UNKNOWN    0.3      256M       10.0.1.7        5m
m3n4o5p6-7890-1234     worker          PENDING    -          -        -          -               30s
```

### Keyboard Shortcuts

#### Global Navigation
| Key | Action |
|-----|---------|
| `?` | Show help |
| `q`, `Ctrl-C` | Quit |
| `/` | Search |
| `Esc` | Cancel/Back |
| `↑`, `k` | Move up |
| `↓`, `j` | Move down |
| `Enter` | Select/Drill down |
| `Backspace` | Go back to parent |

#### Resource Navigation
| Key | Action |
|-----|---------|
| `i` | Go to instances |
| `c` | Go to clusters |
| `s` | Go to services |
| `t` | Go to tasks |
| `d` | Go to task definitions |

#### Instance Operations
| Key | Action | Context |
|-----|---------|---------|
| `N` | Create new instance | Instances view |
| `S` | Stop/Start instance | Instance selected |
| `D` | Delete instance | Instance selected |
| `Ctrl+I` | Quick switch instance | Any view |

#### Service Operations
| Key | Action | Context |
|-----|---------|---------|
| `r` | Restart service | Service selected |
| `S` | Scale service | Service selected |
| `u` | Update service | Service selected |
| `x` | Stop service | Service selected |

#### Common Operations
| Key | Action | Context |
|-----|---------|---------|
| `l` | View logs | Task/Service selected |
| `D` | Describe resource | Any resource |
| `R` | Refresh view | Any view |
| `M` | Multi-instance overview | Any view |

### Special Features

#### Quick Instance Switch (`Ctrl+I`)
```
┌─ Switch Instance ───────────────────────────────────┐
│ > development  ● (current)                          │
│   staging      ●                                    │
│   testing      ○                                    │
│   production   ●                                    │
│   local        ●                                    │
└─────────────────────────────────────────────────────┘
```

#### Log Viewer (`l` key)
```
┌─ Logs: web-service/a1b2c3d4-1234-5678 ─────────────────────────────────┐
│ 2024-01-15 16:30:45 [INFO] Server started on port 8080                 │
│ 2024-01-15 16:30:46 [INFO] Connected to database                       │
│ 2024-01-15 16:31:02 [INFO] GET /health 200 5ms                        │
│ 2024-01-15 16:31:15 [INFO] GET /api/users 200 23ms                    │
│ 2024-01-15 16:31:23 [WARN] Slow query detected: 150ms                 │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
[Esc] Back  [f] Follow  [/] Filter  [s] Save  [↑↓] Scroll
```

#### Command Palette (`:` key)
```
┌─ Command ───────────────────────────────────────────┐
│ > create instance production --port 6449            │
└─────────────────────────────────────────────────────┘

Commands:
- create instance <name> [--port PORT]
- delete instance <name>
- stop instance <name>
- start instance <name>
- list instances
```

### Color Scheme

- **Green**: Healthy/Running states (RUNNING, ACTIVE, HEALTHY)
- **Yellow**: Warning/Transitional states (PENDING, PROVISIONING)
- **Red**: Error/Stopped states (STOPPED, FAILED, UNHEALTHY)
- **Blue**: Informational/Inactive states (INACTIVE)
- **Cyan**: Headers and selected rows
- **White**: Normal text
- **Gray**: Disabled or stale information

### Implementation Details

#### Technology Stack
- **Framework**: [bubbletea](https://github.com/charmbracelet/bubbletea) for modern TUI development
- **Styling**: [lipgloss](https://github.com/charmbracelet/lipgloss) for consistent styling
- **Tables**: [bubbles](https://github.com/charmbracelet/bubbles) table component

#### Data Updates
- Auto-refresh every 5 seconds (configurable)
- Manual refresh with `R` key
- Real-time log streaming via WebSocket connection
- Efficient partial updates to minimize API calls

#### Configuration
```yaml
# ~/.kecs/tui.yaml
refresh_interval: 5s
color_scheme: default
vim_bindings: true
default_view: instances
log_buffer_size: 1000
```

## Consequences

### Positive
- **Improved productivity**: Quick navigation and operation without remembering CLI commands
- **Better monitoring**: Real-time view of all resources across instances
- **Familiar interface**: k9s-like experience for Kubernetes users
- **Multi-environment management**: Easy switching between dev/staging/prod
- **Reduced context switching**: All operations in one interface

### Negative
- **Additional maintenance**: Another interface to maintain alongside CLI
- **Learning curve**: New users need to learn keyboard shortcuts
- **Terminal limitations**: Complex operations may still require CLI
- **Resource usage**: Continuous API polling for updates

### Implementation Plan

1. **Phase 1: Core Framework** (2 weeks)
   - Basic TUI structure with bubbletea
   - Instance and cluster navigation
   - API client integration

2. **Phase 2: Resource Views** (3 weeks)
   - Services and tasks views
   - Real-time updates
   - Search and filtering

3. **Phase 3: Operations** (2 weeks)
   - Create/delete/scale operations
   - Log viewing
   - Command palette

4. **Phase 4: Polish** (1 week)
   - Performance optimization
   - Configuration management
   - Documentation

## References

- [bubbletea TUI framework](https://github.com/charmbracelet/bubbletea)
- [k9s - Kubernetes CLI To Manage Your Clusters](https://k9scli.io/)
- [ADR-0001: Product Concept](./0001-product-concept.md)
- [ADR-0002: Architecture](./0002-architecture.md)
