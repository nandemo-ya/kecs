# Dashboard Screen

## Overview

The dashboard provides a high-level overview of all ECS resources across clusters.

## Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ KECS Dashboard                              Connected │ 15:23:45 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Clusters: 3        Services: 12      Tasks: 45                │
│  ██████████         ████████████      ██████████████           │
│  ACTIVE: 3          ACTIVE: 10        RUNNING: 40              │
│                     DRAINING: 2       PENDING: 5               │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Recent Activity                                                 │
│                                                                 │
│ 15:22:31  [CLUSTER]  prod-cluster      Created                │
│ 15:21:45  [SERVICE]  web-service       Scaled to 5            │
│ 15:20:12  [TASK]     task-abc123       Started                │
│ 15:19:55  [TASK]     task-def456       Stopped                │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Quick Actions                                                   │
│                                                                 │
│ [c] Create Cluster  [s] Create Service  [t] Run Task          │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [Tab] Navigate  [Enter] Select  [?] Help  [q] Quit            │
└─────────────────────────────────────────────────────────────────┘
```

## Features

### Resource Summary
- Total count for each resource type
- Status breakdown with visual indicators
- Progress bars showing resource utilization

### Activity Feed
- Real-time event stream
- Filterable by resource type
- Color-coded by event type

### Quick Actions
- Shortcuts to common operations
- Context-aware based on permissions

## Navigation

From the dashboard, users can:
- Press number keys (1-4) to jump to resource views
- Use arrow keys to navigate activity feed
- Press action keys for quick operations

## Data Updates

- Summary stats refresh every 5 seconds
- Activity feed updates in real-time
- Visual spinner during refresh

## Implementation Notes

```go
type DashboardModel struct {
    clusters  []ClusterSummary
    services  []ServiceSummary
    tasks     []TaskSummary
    events    []Event
    selected  int
    loading   bool
}
```