# Clusters Screen

## Overview

The clusters screen provides management capabilities for ECS clusters.

## List View

```
┌─────────────────────────────────────────────────────────────────┐
│ Clusters (3)                                [/] Search          │
├─────────────────────────────────────────────────────────────────┤
│ Name              Status    Services  Tasks   Created          │
│ ─────────────────────────────────────────────────────────────  │
│ > dev-cluster     ACTIVE    4         12      2d ago           │
│   staging-cluster ACTIVE    8         24      5d ago           │
│   prod-cluster    ACTIVE    15        45      30d ago          │
│                                                                 │
│                                                                 │
│                                                                 │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [↑↓] Navigate  [Enter] Details  [c] Create  [d] Delete        │
└─────────────────────────────────────────────────────────────────┘
```

## Detail View

```
┌─────────────────────────────────────────────────────────────────┐
│ Cluster: dev-cluster                                           │
├─────────────────────────────────────────────────────────────────┤
│ Details                        │ Services (4)                  │
│                               │                                │
│ Name:     dev-cluster         │ > web-app      ACTIVE  3/3    │
│ ARN:      arn:aws:ecs:...     │   api-service  ACTIVE  2/2    │
│ Status:   ACTIVE              │   worker       ACTIVE  1/1    │
│ Created:  2024-06-24 10:30    │   database     ACTIVE  1/1    │
│                               │                                │
│ Statistics:                   ├────────────────────────────────┤
│ - Services:        4          │ Container Instances (2)        │
│ - Running Tasks:   12         │                                │
│ - Pending Tasks:   0          │ > i-1234567    ACTIVE  6/10    │
│ - Instances:       2          │   i-8901234    ACTIVE  6/10    │
│                               │                                │
│ Capacity Providers:           │                                │
│ - FARGATE                     │                                │
│ - FARGATE_SPOT               │                                │
│                               │                                │
├─────────────────────────────────────────────────────────────────┤
│ [Tab] Switch Pane  [s] Create Service  [r] Refresh  [b] Back  │
└─────────────────────────────────────────────────────────────────┘
```

## Create Cluster Form

```
┌─────────────────────────────────────────────────────────────────┐
│ Create Cluster                                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Cluster Name: [test-cluster                    ]               │
│                                                                 │
│ Capacity Providers:                                             │
│ [x] FARGATE                                                     │
│ [x] FARGATE_SPOT                                               │
│ [ ] EC2                                                         │
│                                                                 │
│ Default Capacity Provider Strategy:                             │
│ Provider: [FARGATE     ▼]  Base: [0  ]  Weight: [100]         │
│                                                                 │
│ Container Insights: [x] Enabled                                 │
│                                                                 │
│ Tags:                                                           │
│ Key: [Environment    ]  Value: [development         ]          │
│ [+] Add Tag                                                     │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [Tab] Next Field  [Enter] Submit  [Esc] Cancel                 │
└─────────────────────────────────────────────────────────────────┘
```

## Features

### List View
- Sortable columns (click header or use `s` key)
- Real-time status updates
- Quick stats for each cluster
- Search/filter functionality

### Detail View
- Split pane showing cluster info and resources
- Navigate between panes with Tab
- Drill down into services/instances
- Real-time capacity monitoring

### Actions
- Create new cluster with validation
- Delete cluster (with confirmation)
- Update cluster settings
- View cluster events

## Keyboard Shortcuts

- `c` - Create new cluster
- `d` - Delete selected cluster
- `e` - Edit cluster settings
- `s` - View services in cluster
- `i` - View container instances
- `/` - Search clusters
- `r` - Refresh list

## State Management

```go
type ClustersModel struct {
    clusters      []Cluster
    selected      int
    view          ViewType // LIST, DETAIL, CREATE
    detailCluster *Cluster
    form          *ClusterForm
    filter        string
}
```