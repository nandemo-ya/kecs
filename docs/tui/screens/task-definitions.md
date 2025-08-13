# Task Definitions Screen

## Overview

The task definitions screen manages ECS task definitions and their revisions.

## List View

```
┌─────────────────────────────────────────────────────────────────┐
│ Task Definitions (8)                    [/] Search              │
├─────────────────────────────────────────────────────────────────┤
│ Family            Latest Rev  Active Rev  Status    Created    │
│ ──────────────────────────────────────────────────────────     │
│ > web-app         15          15          ACTIVE    2d ago     │
│   api-service     8           7           ACTIVE    5d ago     │
│   worker          12          12          ACTIVE    1d ago     │
│   batch-processor 3           3           INACTIVE  10d ago    │
│   ml-inference    6           5           ACTIVE    3d ago     │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [↑↓] Navigate  [Enter] Details  [n] New  [r] Register Revision│
└─────────────────────────────────────────────────────────────────┘
```

## Detail View

```
┌─────────────────────────────────────────────────────────────────┐
│ Task Definition: web-app:15                                     │
├─────────────────────────────────────────────────────────────────┤
│ Details                        │ Containers (2)                │
│                               │                                │
│ Family:      web-app          │ > web                         │
│ Revision:    15               │   Image: web-app:v2.1.0       │
│ Status:      ACTIVE           │   CPU: 256  Memory: 512 MB    │
│ Task Role:   ecsTaskRole      │   Port: 80/tcp                │
│ Network:     awsvpc           │   Essential: Yes              │
│ CPU:         512              │                                │
│ Memory:      1024 MB          │   sidecar                     │
│ Created:     2024-06-25       │   Image: fluentbit:latest     │
│                               │   CPU: 128  Memory: 256 MB    │
│ Compatibilities:              │   Essential: No               │
│ - FARGATE                     │                                │
│ - EC2                         ├────────────────────────────────┤
│                               │ Revisions (15)                 │
│ Volumes: None                 │                                │
│                               │ > 15 (ACTIVE)    2d ago       │
│ Tags:                         │   14             3d ago       │
│ - env: production             │   13             5d ago       │
│ - team: platform              │   12             7d ago       │
│                               │   11             10d ago      │
├─────────────────────────────────────────────────────────────────┤
│ [Tab] Pane  [r] New Revision  [d] Deregister  [j] View JSON   │
└─────────────────────────────────────────────────────────────────┘
```

## Container Detail View

```
┌─────────────────────────────────────────────────────────────────┐
│ Container: web                                                  │
├─────────────────────────────────────────────────────────────────┤
│ Configuration                                                   │
│                                                                 │
│ Image:          web-app:v2.1.0                                 │
│ CPU:            256 cpu units                                  │
│ Memory:         512 MB (soft)                                  │
│ Memory Resv:    512 MB (hard)                                  │
│ Essential:      Yes                                            │
│                                                                 │
│ Port Mappings                                                   │
│ Container  Host    Protocol                                     │
│ 80         80      tcp                                         │
│                                                                 │
│ Environment Variables                                           │
│ PORT            = 80                                           │
│ LOG_LEVEL       = info                                         │
│ DB_HOST         = ${DB_HOST}                                   │
│                                                                 │
│ Secrets                                                         │
│ API_KEY         from: arn:aws:secretsmanager:...              │
│                                                                 │
│ Health Check                                                    │
│ Command:        ["CMD-SHELL", "curl -f http://localhost/health"]│
│ Interval:       30s                                            │
│ Timeout:        5s                                             │
│ Retries:        3                                              │
│ Start Period:   60s                                            │
│                                                                 │
│ Logging                                                         │
│ Driver:         awslogs                                        │
│ Options:                                                        │
│   group:        /ecs/web-app                                   │
│   region:       us-east-1                                      │
│   prefix:       web                                            │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [e] Edit  [c] Copy Config  [Esc] Back                         │
└─────────────────────────────────────────────────────────────────┘
```

## Register Task Definition Form

```
┌─────────────────────────────────────────────────────────────────┐
│ Register Task Definition                                        │
├─────────────────────────────────────────────────────────────────┤
│ Basic Information                                               │
│                                                                 │
│ Family:         [my-app                        ]               │
│ Task Role:      [ecsTaskRole                ▼]                 │
│ Execution Role: [ecsTaskExecutionRole       ▼]                 │
│                                                                 │
│ Task Size                                                       │
│                                                                 │
│ CPU:    [0.25 vCPU  ▼]    Memory: [0.5 GB      ▼]            │
│                                                                 │
│ Compatibilities                                                 │
│ [x] FARGATE    [x] EC2                                        │
│                                                                 │
│ Network Mode:   (•) awsvpc  ( ) bridge  ( ) host              │
│                                                                 │
│ Container Definitions                                           │
│ ┌─────────────────────────────────────────────┐               │
│ │ Name:     [web                    ]         │               │
│ │ Image:    [nginx:latest           ]         │               │
│ │ CPU:      [256]    Memory: [512  ] MB       │               │
│ │ Port:     [80 ]    Protocol: [tcp ▼]       │               │
│ │ Essential: [x]                               │               │
│ └─────────────────────────────────────────────┘               │
│                                                                 │
│ [+] Add Container                                              │
│                                                                 │
│ [ ] Import from JSON                                           │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ [Tab] Next  [Enter] Register  [Esc] Cancel                     │
└─────────────────────────────────────────────────────────────────┘
```

## JSON View

```
┌─────────────────────────────────────────────────────────────────┐
│ Task Definition JSON: web-app:15               [e] Edit        │
├─────────────────────────────────────────────────────────────────┤
│ {                                                               │
│   "family": "web-app",                                         │
│   "revision": 15,                                              │
│   "taskRoleArn": "arn:aws:iam::123456789012:role/ecsTaskRole",│
│   "executionRoleArn": "arn:aws:iam::123456789012:role/...",   │
│   "networkMode": "awsvpc",                                     │
│   "containerDefinitions": [                                     │
│     {                                                           │
│       "name": "web",                                           │
│       "image": "web-app:v2.1.0",                              │
│       "cpu": 256,                                              │
│       "memory": 512,                                           │
│       "essential": true,                                       │
│       "portMappings": [                                        │
│         {                                                       │
│           "containerPort": 80,                                 │
│           "protocol": "tcp"                                    │
│         }                                                       │
│       ],                                                        │
│       "environment": [                                          │
│         {                                                       │
│           "name": "PORT",                                      │
│           "value": "80"                                        │
│         }                                                       │
│       ]                                                         │
│     }                                                           │
│   ],                                                            │
│   "requiresCompatibilities": ["FARGATE", "EC2"],              │
│   "cpu": "512",                                                │
│   "memory": "1024"                                             │
│ }                                                               │
├─────────────────────────────────────────────────────────────────┤
│ [c] Copy  [s] Save to File  [r] Register as New  [Esc] Back   │
└─────────────────────────────────────────────────────────────────┘
```

## Features

### Task Definition Management
- View all task definition families
- Browse revision history
- Register new task definitions
- Clone existing definitions
- Import/Export JSON

### Container Configuration
- Add/remove containers
- Configure resource limits
- Set environment variables
- Manage secrets
- Configure health checks
- Set up logging

### Advanced Features
- Volume management
- Task placement constraints
- ProxyConfiguration
- FireLens configuration
- Sidecar dependencies

## Keyboard Shortcuts

- `n` - Create new task definition
- `r` - Register new revision
- `d` - Deregister task definition
- `j` - View/Edit as JSON
- `c` - Clone task definition
- `e` - Export to file
- `i` - Import from file
- `/` - Search families

## Status Indicators

- 🟢 `ACTIVE` - Can be used to run tasks
- ⚪ `INACTIVE` - Deregistered, cannot run new tasks

## Implementation Notes

```go
type TaskDefinitionsModel struct {
    families       []TaskDefinitionFamily
    selectedIndex  int
    view          ViewType
    detailFamily  *TaskDefinitionFamily
    revisions     []TaskDefinitionRevision
    containerView *ContainerDetail
    jsonView      *JSONViewer
    registerForm  *TaskDefForm
}
```

## Validation

The form includes real-time validation for:
- CPU/Memory combinations (Fargate requirements)
- Container port conflicts
- Essential container rules
- Image format validation
- Environment variable format