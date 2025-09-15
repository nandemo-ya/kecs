# TUI Interface Guide

KECS includes a powerful Terminal User Interface (TUI) for interactive management of your ECS resources. The TUI provides a visual, keyboard-driven interface for browsing and managing clusters, services, tasks, and more.

## Launching the TUI

```bash
# Launch TUI with default settings
kecs tui

# Connect to specific instance
kecs tui --instance dev

# Custom refresh interval
kecs tui --refresh 10

# Dark theme
kecs tui --theme dark
```

## Interface Overview

```
┌────────────────────────────────────────────────────────────────────────────────┐
│ KECS TUI v0.5.0                                              [q]uit [h]elp │
├─────────────────────────┬──────────────────────────┬────────────────────────────┤
│ Resources               │ Details                  │ Actions                    │
├─────────────────────────┼──────────────────────────┼────────────────────────────┤
│ ▶ Clusters (2)         │ Name: default            │ [Enter] Select             │
│   ▷ default            │ Status: ACTIVE           │ [n] New                    │
│   ▷ production         │ Services: 5              │ [d] Delete                 │
│ ▷ Services (5)         │ Tasks: 12                │ [e] Edit                   │
│ ▷ Tasks (12)           │ Created: 2024-01-15      │ [l] Logs                   │
│ ▷ Task Definitions (3) │                          │ [r] Refresh                │
│ ▷ Load Balancers (2)   │ Registered Instances: 3  │ [s] Scale                  │
│ ▷ Target Groups (2)    │ Container Instances: 3   │ [u] Update                 │
└─────────────────────────┴──────────────────────────┴────────────────────────────┘
│ Status: Connected to kecs-dev | Last refresh: 10s ago | Mode: Browse       │
└────────────────────────────────────────────────────────────────────────────────┘
```

## Navigation

### Keyboard Shortcuts

| Key | Action | Description |
|-----|--------|-------------|
| `↑`/`k` | Move up | Navigate up in the current list |
| `↓`/`j` | Move down | Navigate down in the current list |
| `←`/`h` | Move left | Navigate to parent/previous panel |
| `→`/`l` | Move right | Navigate to child/next panel |
| `Enter` | Select | Open/expand selected item |
| `Space` | Toggle | Expand/collapse tree node |
| `Tab` | Switch panel | Move between panels |
| `Esc` | Back | Go back to previous view |
| `q` | Quit | Exit the TUI |
| `?` | Help | Show help screen |

### Resource Navigation

```
Clusters
└── Services
    └── Tasks
        └── Containers
            └── Logs
```

## Resource Management

### Clusters

**View Cluster Details:**
- Navigate to cluster and press `Enter`
- Shows services, tasks, container instances
- Displays resource utilization

**Create New Cluster:**
1. Select "Clusters" in the resource tree
2. Press `n` for new
3. Enter cluster name
4. Configure settings
5. Press `Enter` to create

**Delete Cluster:**
1. Select cluster
2. Press `d` for delete
3. Confirm deletion

### Services

**View Service Details:**
- Shows task definition, desired/running count
- Displays deployments and events
- Shows associated load balancers

**Scale Service:**
1. Select service
2. Press `s` for scale
3. Enter desired count
4. Press `Enter` to apply

**Update Service:**
1. Select service
2. Press `u` for update
3. Choose new task definition
4. Configure update options
5. Apply changes

**Service Actions Menu:**
```
┌─────────────────────────┐
│ Service Actions         │
├─────────────────────────┤
│ [s] Scale               │
│ [u] Update              │
│ [r] Force Redeploy      │
│ [d] Delete              │
│ [l] View Logs           │
│ [e] Edit Tags           │
│ [m] Metrics             │
└─────────────────────────┘
```

### Tasks

**View Task Details:**
- Shows container status
- Displays network configuration
- Shows resource allocation
- Lists environment variables

**Stop Task:**
1. Select task
2. Press `x` to stop
3. Confirm action

**View Task Logs:**
1. Select task
2. Press `l` for logs
3. Choose container if multiple
4. View streaming logs

### Task Definitions

**Create Task Definition:**
1. Select "Task Definitions"
2. Press `n` for new
3. Use editor or form view
4. Configure containers
5. Save definition

**Task Definition Editor:**
```json
{
  "family": "my-app",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "nginx:alpine",
      "portMappings": [
        {
          "containerPort": 80
        }
      ]
    }
  ]
}
```

## Log Viewer

### Features
- Real-time log streaming
- Search and filter
- Color-coded log levels
- Export to file

### Log Viewer Controls

| Key | Action |
|-----|--------|
| `/` | Search |
| `f` | Filter |
| `c` | Clear filter |
| `w` | Wrap lines |
| `PgUp` | Page up |
| `PgDn` | Page down |
| `Home` | Go to start |
| `End` | Go to end |
| `s` | Save to file |

### Log Filtering

```
Filter: level:error container:app
┌──────────────────────────────────────────────────────────────────┐
│ 2024-01-15 10:30:15 ERROR [app] Connection refused to database  │
│ 2024-01-15 10:30:20 ERROR [app] Retry attempt 1 failed         │
│ 2024-01-15 10:30:25 ERROR [app] Retry attempt 2 failed         │
└──────────────────────────────────────────────────────────────────┘
```

## Load Balancer Management

### View Load Balancers
- Shows ALB/NLB list
- Displays DNS names
- Shows listener configuration
- Lists target groups

### Create Load Balancer
1. Navigate to "Load Balancers"
2. Press `n` for new
3. Choose type (ALB/NLB)
4. Configure settings
5. Add listeners
6. Create

## Monitoring Dashboard

### Resource Metrics
```
┌────────────────────────────────────────────────────────────┐
│ Cluster: default                                             │
├───────────────────────────┬─────────────────────────────────┤
│ CPU Usage                │ Memory Usage                    │
│ ███████░░░ 35%          │ ████████████░░ 62%              │
├───────────────────────────┼─────────────────────────────────┤
│ Running Tasks: 12/15     │ Active Services: 5              │
│ Pending Tasks: 3         │ Container Instances: 3          │
└───────────────────────────┴─────────────────────────────────┘
```

### Events View
```
┌────────────────────────────────────────────────────────────┐
│ Recent Events                                                │
├────────────────────────────────────────────────────────────┤
│ 10:45:23 [INFO]  Service 'web-app' scaled to 5 tasks        │
│ 10:44:15 [INFO]  Task arn:aws:ecs:task/123 started          │
│ 10:43:00 [WARN]  Task arn:aws:ecs:task/122 stopped          │
│ 10:42:30 [INFO]  Deployment completed for 'api-service'     │
│ 10:41:00 [ERROR] Health check failed for task/121           │
└────────────────────────────────────────────────────────────┘
```

## Search and Filter

### Global Search
Press `/` to activate global search:
- Search across all resources
- Filter by resource type
- Search by tags
- Regex support

### Filter Syntax
```
type:service status:running
name:web-* cluster:production
tag:environment=prod
```

## Configuration

### TUI Settings
Press `c` to open configuration:

```
┌──────────────────────────────────────────┐
│ TUI Configuration                         │
├──────────────────────────────────────────┤
│ Theme:           [x] Dark [ ] Light       │
│ Refresh Rate:    [5] seconds              │
│ Show Timestamps: [x] Yes                  │
│ Auto-expand:     [ ] Yes                  │
│ Confirm Delete:  [x] Yes                  │
│ Log Buffer:      [1000] lines             │
└──────────────────────────────────────────┘
```

### Themes

**Dark Theme (Default):**
- High contrast colors
- Reduced eye strain
- Better for long sessions

**Light Theme:**
- Bright background
- Better in well-lit environments

## Advanced Features

### Multi-Instance Support

Switch between KECS instances:
1. Press `i` for instance switcher
2. Select instance from list
3. TUI reconnects automatically

### Export Functions

**Export Resources:**
- Press `e` on any resource
- Choose format (JSON, YAML)
- Save to file

**Export Logs:**
- In log viewer, press `s`
- Choose time range
- Select format
- Save to file

### Batch Operations

**Multi-select Mode:**
1. Press `m` to enter multi-select
2. Use `Space` to select items
3. Choose batch action
4. Confirm operation

**Batch Actions:**
- Stop multiple tasks
- Delete multiple services
- Update tags on multiple resources

## Troubleshooting

### Connection Issues

```bash
# Check KECS instance is running
kecs status

# Verify API endpoint
kecs cluster info

# Test connection
curl http://localhost:5373/v1/
```

### Display Issues

**Terminal Too Small:**
- Minimum size: 80x24
- Recommended: 120x40
- Resize terminal and restart TUI

**Colors Not Showing:**
```bash
# Check terminal supports colors
echo $TERM

# Set proper terminal type
export TERM=xterm-256color
```

### Performance Issues

**Slow Updates:**
- Increase refresh interval
- Check network latency
- Reduce log buffer size

**High CPU Usage:**
- Disable auto-refresh
- Close unused panels
- Reduce animation effects

## Tips and Tricks

### Productivity Tips

1. **Quick Navigation:**
   - Use number keys 1-9 to jump to sections
   - Press `g` for go-to menu
   - Use bookmarks with `b`

2. **Efficient Monitoring:**
   - Split view with `|` for side-by-side
   - Pin important resources with `p`
   - Set up alerts with `a`

3. **Keyboard Macros:**
   - Record with `Ctrl+r`
   - Play with `Ctrl+p`
   - Save frequently used sequences

### Common Workflows

**Deploy New Service:**
1. Create task definition (`n` in Task Definitions)
2. Create service (`n` in Services)
3. Configure auto-scaling
4. Monitor deployment in Events

**Troubleshoot Service:**
1. Select service
2. Check events panel
3. View task logs (`l`)
4. Check target health
5. Review metrics

**Rolling Update:**
1. Update task definition
2. Select service
3. Press `u` for update
4. Monitor deployment progress
5. Verify new tasks are healthy

## Integration with CLI

The TUI complements CLI commands:

```bash
# Create resources with CLI
aws ecs create-cluster --cluster-name prod

# Monitor with TUI
kecs tui --instance prod

# Make changes in TUI
# Verify with CLI
aws ecs describe-clusters --clusters prod
```

## Customization

### Custom Key Bindings

Edit `~/.kecs/tui-config.yaml`:

```yaml
keybindings:
  navigation:
    up: ["k", "Up"]
    down: ["j", "Down"]
    left: ["h", "Left"]
    right: ["l", "Right"]
  actions:
    delete: ["d", "Delete"]
    scale: ["s"]
    logs: ["L"]
```

### Custom Layouts

Save and load custom layouts:
1. Arrange panels as desired
2. Press `Ctrl+s` to save layout
3. Name the layout
4. Load with `Ctrl+l`

## Next Steps

- [CLI Commands](/guides/cli-commands) - Command-line reference
- [Services Guide](/guides/services) - Managing ECS services
- [Troubleshooting](/guides/troubleshooting) - Common issues