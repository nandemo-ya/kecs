# Services Screen

## Overview

The services screen provides comprehensive service management capabilities.

## List View

```
┌─────────────────────────────────────────────────────────────────┐
│ Services (12)                   Cluster: [dev-cluster    ▼]    │
├─────────────────────────────────────────────────────────────────┤
│ Name            Status   Desired  Running  Pending  Health     │
│ ───────────────────────────────────────────────────────────     │
│ > web-app       ACTIVE   5        5        0        HEALTHY    │
│   api-service   ACTIVE   3        2        1        DEGRADED   │
│   worker        ACTIVE   2        2        0        HEALTHY    │
│   batch-job     DRAINING 0        1        0        -          │
│                                                                 │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [↑↓] Navigate  [Enter] Details  [s] Scale  [u] Update         │
└─────────────────────────────────────────────────────────────────┘
```

## Detail View

```
┌─────────────────────────────────────────────────────────────────┐
│ Service: web-app                                               │
├─────────────────────────────────────────────────────────────────┤
│ Overview                       │ Tasks (5)                     │
│                               │                                │
│ Status:         ACTIVE        │ > task-123  RUNNING  5m       │
│ Launch Type:    FARGATE       │   task-124  RUNNING  5m       │
│ Task Def:       web-app:15    │   task-125  RUNNING  3m       │
│ Desired Count:  5             │   task-126  RUNNING  3m       │
│ Running Count:  5             │   task-127  RUNNING  1m       │
│ Pending Count:  0             │                                │
│                               ├────────────────────────────────┤
│ Deployment                    │ Events                         │
│ Strategy:   ROLLING_UPDATE    │                                │
│ Min Health: 100%              │ 15:23:45 Task started         │
│ Max Health: 200%              │ 15:22:30 Scaling to 5         │
│                               │ 15:21:00 Deployment complete  │
│ Load Balancer                 │ 15:20:00 Starting deployment  │
│ Target Group: web-app-tg      │                                │
│ Container:    web             │                                │
│ Port:         80              │                                │
├─────────────────────────────────────────────────────────────────┤
│ [Tab] Pane  [s] Scale  [u] Update  [l] Logs  [r] Refresh     │
└─────────────────────────────────────────────────────────────────┘
```

## Scale Service Dialog

```
┌─────────────────────────────────────────────────────────────────┐
│ Scale Service: web-app                                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Current Desired Count: 5                                        │
│                                                                 │
│ New Desired Count: [8        ]                                 │
│                                                                 │
│ ┌─────────────────────────────────────────────┐               │
│ │ This will trigger a new deployment.         │               │
│ │ Tasks will be gradually replaced.           │               │
│ └─────────────────────────────────────────────┘               │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [Enter] Confirm  [Esc] Cancel                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Create Service Form

```
┌─────────────────────────────────────────────────────────────────┐
│ Create Service                                                  │
├─────────────────────────────────────────────────────────────────┤
│ Basic Configuration                                             │
│                                                                 │
│ Service Name:    [my-service               ]                   │
│ Cluster:         [dev-cluster           ▼]                     │
│ Launch Type:     ( ) EC2  (•) FARGATE                         │
│                                                                 │
│ Task Definition                                                 │
│                                                                 │
│ Family:          [web-app               ▼]                     │
│ Revision:        [15 (latest)           ▼]                     │
│                                                                 │
│ Deployment Configuration                                        │
│                                                                 │
│ Desired Count:   [3  ]                                         │
│ Min Healthy %:   [100]     Max %: [200]                       │
│                                                                 │
│ Network Configuration                                           │
│                                                                 │
│ VPC:            [vpc-12345              ▼]                     │
│ Subnets:        [x] subnet-1a  [x] subnet-1b                  │
│ Security Group: [sg-webapp              ▼]                     │
│ Public IP:      [x] Enabled                                    │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [Tab] Next  [Shift+Tab] Previous  [Enter] Create  [Esc] Cancel│
└─────────────────────────────────────────────────────────────────┘
```

## Features

### Service Management
- Scale services up/down
- Update service configuration
- Force new deployment
- Stop service (set desired count to 0)

### Monitoring
- Real-time task status
- Deployment progress
- Health check status
- Event history

### Task Operations
- View task details
- Stop individual tasks
- View container logs
- SSH into containers (if enabled)

## Keyboard Shortcuts

- `s` - Scale service
- `u` - Update service
- `d` - Delete service
- `f` - Force new deployment
- `l` - View logs
- `t` - View tasks
- `e` - View events
- `/` - Filter services

## Status Indicators

- 🟢 `HEALTHY` - All tasks running and passing health checks
- 🟡 `DEGRADED` - Some tasks unhealthy or pending
- 🔴 `UNHEALTHY` - No healthy tasks
- ⚪ `DRAINING` - Service being removed

## Implementation Notes

```go
type ServicesModel struct {
    services       []Service
    selectedIndex  int
    view          ViewType
    detailService *Service
    tasks         []Task
    events        []Event
    scaleDialog   *ScaleDialog
    createForm    *ServiceForm
}
```