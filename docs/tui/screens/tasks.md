# Tasks Screen

## Overview

The tasks screen provides comprehensive task management and monitoring capabilities.

## List View

```
┌─────────────────────────────────────────────────────────────────┐
│ Tasks (45)                      Cluster: [all-clusters     ▼]  │
├─────────────────────────────────────────────────────────────────┤
│ Task ID         Service      Status    Started      CPU   Mem  │
│ ────────────────────────────────────────────────────────────   │
│ > task-abc123   web-app      RUNNING   5m ago      25%   512MB│
│   task-def456   api-service  RUNNING   10m ago     10%   256MB│
│   task-ghi789   worker       PENDING   -           -     -    │
│   task-jkl012   batch-job    STOPPED   2h ago      0%    0MB  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [↑↓] Navigate  [Enter] Details  [x] Stop  [l] Logs  [/] Filter│
└─────────────────────────────────────────────────────────────────┘
```

## Detail View

```
┌─────────────────────────────────────────────────────────────────┐
│ Task: task-abc123                                              │
├─────────────────────────────────────────────────────────────────┤
│ Overview                       │ Containers (2)                │
│                               │                                │
│ Task Definition: web-app:15   │ > web        RUNNING          │
│ Service:         web-app      │   Status:    Healthy          │
│ Cluster:         dev-cluster  │   CPU:       25% (256 cpu)    │
│ Launch Type:     FARGATE      │   Memory:    45% (512 MB)     │
│ Platform:        1.4.0        │   Port:      80 → 32768       │
│                               │                                │
│ Status:          RUNNING      │   sidecar    RUNNING          │
│ Desired Status:  RUNNING      │   Status:    Healthy          │
│ Started:         5m ago       │   CPU:       5% (128 cpu)     │
│ Group:           service:web  │   Memory:    20% (256 MB)     │
│                               │                                │
│ Network                       ├────────────────────────────────┤
│ VPC:        vpc-12345         │ Events                         │
│ Subnet:     subnet-1a         │                                │
│ Public IP:  54.123.45.67      │ 15:23:45 Container started    │
│ Private IP: 10.0.1.23         │ 15:23:30 Task provisioning    │
│                               │ 15:23:00 Task pending         │
├─────────────────────────────────────────────────────────────────┤
│ [Tab] Pane  [l] Logs  [x] Stop  [s] Shell  [r] Refresh       │
└─────────────────────────────────────────────────────────────────┘
```

## Container Logs View

```
┌─────────────────────────────────────────────────────────────────┐
│ Logs: task-abc123 / web                    [Following]         │
├─────────────────────────────────────────────────────────────────┤
│ 2024-06-27 15:23:45 INFO  Starting application...              │
│ 2024-06-27 15:23:46 INFO  Loaded configuration                 │
│ 2024-06-27 15:23:47 INFO  Database connection established      │
│ 2024-06-27 15:23:48 INFO  Server listening on :8080           │
│ 2024-06-27 15:24:12 INFO  GET /health 200 2ms                 │
│ 2024-06-27 15:24:42 INFO  GET /api/users 200 45ms             │
│ 2024-06-27 15:25:01 WARN  Slow query detected (523ms)         │
│ 2024-06-27 15:25:15 INFO  GET /api/products 200 67ms          │
│ 2024-06-27 15:25:45 ERROR Failed to connect to cache          │
│ 2024-06-27 15:25:46 INFO  Retrying cache connection...        │
│ 2024-06-27 15:25:47 INFO  Cache connection restored           │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [f] Follow  [w] Wrap  [/] Search  [n/N] Next/Prev  [q] Back  │
└─────────────────────────────────────────────────────────────────┘
```

## Run Task Form

```
┌─────────────────────────────────────────────────────────────────┐
│ Run Task                                                        │
├─────────────────────────────────────────────────────────────────┤
│ Task Definition                                                 │
│                                                                 │
│ Family:          [web-app               ▼]                     │
│ Revision:        [15 (latest)           ▼]                     │
│                                                                 │
│ Cluster:         [dev-cluster           ▼]                     │
│ Launch Type:     ( ) EC2  (•) FARGATE                         │
│                                                                 │
│ Count:           [1  ]                                         │
│                                                                 │
│ Network Configuration                                           │
│                                                                 │
│ VPC:            [vpc-12345              ▼]                     │
│ Subnets:        [x] subnet-1a  [x] subnet-1b                  │
│ Security Group: [sg-webapp              ▼]                     │
│ Public IP:      [x] Enabled                                    │
│                                                                 │
│ Overrides (Optional)                                           │
│                                                                 │
│ Container:      [web                    ▼]                     │
│ Command:        [                             ]                │
│ Environment:    [+] Add Variable                               │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [Tab] Next  [Shift+Tab] Previous  [Enter] Run  [Esc] Cancel   │
└─────────────────────────────────────────────────────────────────┘
```

## Features

### Task Management
- Stop individual tasks
- Run new tasks (standalone or service)
- View task resource utilization
- Access container logs

### Monitoring
- Real-time CPU/Memory metrics
- Container health status
- Network configuration details
- Event history

### Log Viewer
- Follow mode for real-time logs
- Search functionality
- Line wrapping toggle
- Multi-container support

### Filtering
- By cluster
- By service
- By status (RUNNING, PENDING, STOPPED)
- By launch type
- By task definition family

## Keyboard Shortcuts

- `x` - Stop selected task
- `l` - View logs
- `s` - Shell into container (if exec enabled)
- `r` - Run new task
- `f` - Filter tasks
- `c` - Copy task ARN
- `/` - Search tasks

## Status Indicators

- 🟢 `RUNNING` - Task is running normally
- 🟡 `PENDING` - Task is being provisioned
- 🔵 `PROVISIONING` - Resources being allocated
- ⚪ `STOPPED` - Task has stopped
- 🔴 `STOPPING` - Task is stopping
- 🔴 `FAILED` - Task failed to start

## Implementation Notes

```go
type TasksModel struct {
    tasks         []Task
    selectedIndex int
    view          ViewType
    detailTask    *Task
    containers    []Container
    events        []Event
    logs          *LogViewer
    runForm       *RunTaskForm
    filter        TaskFilter
}
```

## Resource Metrics

- CPU: Show percentage and allocated CPU units
- Memory: Show percentage and MB used/allocated
- Network: Show inbound/outbound traffic if available
- Storage: Show ephemeral storage usage