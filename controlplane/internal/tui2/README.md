# KECS TUI v2 Command Palette

The KECS TUI v2 includes a powerful command palette that provides quick access to common actions and commands.

## Features

### Activation
- Press `:` to activate command mode
- Press `Enter` with empty command input to open the command palette
- Press `Tab` while typing a command to see autocomplete suggestions

### Command Categories
The command palette organizes commands into categories:
- **General**: Common operations like help, refresh, and search
- **Navigation**: Quick navigation between views
- **Create**: Create new resources (instances, clusters, services)
- **Manage**: Start, stop, restart, and delete resources
- **Scale**: Scale services up or down
- **Debug**: Describe resources and execute commands in containers
- **Export**: Export data in JSON, YAML, or CSV formats

### Context-Aware Commands
Commands are filtered based on the current context:
- Only shows cluster commands when an instance is selected
- Only shows service commands when a cluster is selected
- Only shows task commands when a service is selected

### Command History
- Use `↑` and `↓` arrows to navigate through command history
- Recently executed commands are saved for quick access
- History is maintained across command executions

### Visual Feedback
- Commands are grouped by category for easy discovery
- Selected command is highlighted with a `▶` indicator
- Keyboard shortcuts are shown in yellow `[S]` format
- Command results are displayed in the footer for 3 seconds
- Errors are shown in red with helpful messages

### Search and Filter
- Start typing to filter commands by name or alias
- Partial matches are supported
- Aliases provide alternative ways to execute commands

## Usage Examples

### Quick Navigation
1. Press `:` to enter command mode
2. Type `gi` and press `Enter` to go to instances
3. Type `gc` and press `Enter` to go to clusters
4. Type `gs` and press `Enter` to go to services

### Resource Management
1. Navigate to the desired view
2. Press `:` and type `create` to see creation options
3. Use `delete` to remove selected resources
4. Use `start` or `stop` to manage resource states

### Scaling Services
1. Navigate to services view
2. Select a service
3. Press `:` and type `scale up` or `scale down`
4. Or use the shortcuts `su` and `sd`

### Export Data
1. Press `:` in any view
2. Type `export` to see export options
3. Choose `export json`, `export yaml`, or `export csv`

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `:` | Enter command mode |
| `Tab` | Show command palette / autocomplete |
| `↑` | Previous command from history |
| `↓` | Next command from history |
| `Enter` | Execute command |
| `Esc` | Cancel command mode |

## Available Commands

### General Commands
- `help` (aliases: `h`, `?`) - Show help documentation
- `refresh` (aliases: `r`, `reload`) - Refresh current data
- `search` (aliases: `find`, `f`) - Search in current view

### Navigation Commands
- `goto instances` (aliases: `instances`, `gi`) - Navigate to instances
- `goto clusters` (aliases: `clusters`, `gc`) - Navigate to clusters
- `goto services` (aliases: `services`, `gs`) - Navigate to services
- `goto tasks` (aliases: `tasks`, `gt`) - Navigate to tasks
- `logs` (aliases: `log`, `gl`) - View logs

### Create Commands
- `create instance` (aliases: `new instance`, `ci`) - Create a new instance
- `create cluster` (aliases: `new cluster`, `cc`) - Create a new cluster
- `create service` (aliases: `new service`, `cs`) - Create a new service

### Manage Commands
- `start` (alias: `run`) - Start selected resource
- `stop` (alias: `halt`) - Stop selected resource
- `restart` (alias: `reload service`) - Restart selected service
- `delete` (aliases: `remove`, `rm`) - Delete selected resource

### Scale Commands
- `scale up` (aliases: `scale+`, `su`) - Scale up service
- `scale down` (aliases: `scale-`, `sd`) - Scale down service
- `scale` - Scale to specific count

### Debug Commands
- `describe` (aliases: `desc`, `info`) - Show resource details
- `exec` (alias: `shell`) - Execute command in container

### Export Commands
- `export json` (alias: `ej`) - Export as JSON
- `export yaml` (alias: `ey`) - Export as YAML
- `export csv` (alias: `ec`) - Export as CSV