# KECS Terminal User Interface (TUI)

The KECS TUI provides a powerful, interactive terminal-based interface for managing ECS resources with real-time updates and intuitive keyboard navigation.

## Features

### Core Features
- **Instance Management**: Create, start, stop, and switch between KECS instances
- **Dashboard View**: Real-time overview of all ECS resources with current instance indicator
- **Resource Management**: Full CRUD operations for clusters and services
- **Task Monitoring**: View and track running/stopped tasks
- **Task Definitions**: Browse and inspect task definition configurations
- **Real-time Updates**: Auto-refresh every 30 seconds with manual refresh option
- **Keyboard-First Design**: Complete keyboard navigation without mouse

### Advanced Features
- **Search**: Real-time search across all resource types
- **Filtering**: Multi-select status filters for refined views
- **Context-Sensitive Help**: Intelligent help system that adapts to current view
- **Create Forms**: Interactive forms for creating clusters and services
- **Detail Views**: Comprehensive resource information displays
- **Status Indicators**: Color-coded status for quick visual feedback

## Quick Start

```bash
# Start the TUI with default settings (auto-detects running instances)
kecs tui

# Connect to a specific endpoint
kecs tui --endpoint http://localhost:8080

# Connect to a specific instance by name
kecs tui --instance dev-cluster

# Connect to remote KECS instance
kecs tui --endpoint http://remote-kecs:8080
```

## Key Features in Action

### Search (`/`)
- Real-time search as you type
- Case-insensitive matching
- Searches multiple fields per resource type
- `Esc` to clear and exit search

### Filter (`f`)
- Multi-select status filters
- Quick select all (`a`) or clear (`c`)
- Shows filtered count (e.g., "Showing 5 of 10")
- Combines with search for precise results

### Create Resources (`n`)
- Interactive forms with validation
- Tab navigation between fields
- `Ctrl+s` to submit, `Esc` to cancel
- Automatic refresh after creation

### Context Help (`?`)
- Shows relevant shortcuts for current view
- Full documentation overlay
- Context-aware guidance
- Toggle on/off design

## Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `1-5` | Switch between resource views |
| `6` | Instance management |
| `i` | Quick instance switch |
| `↑/↓` or `j/k` | Navigate up/down |
| `Enter` | Select/View details |
| `Esc` | Go back/Cancel |
| `PgUp/PgDn` | Page navigation |
| `g/G` | Go to start/end |

### Actions
| Key | Action |
|-----|--------|
| `n` | Create new resource/instance |
| `s` | Stop instance (in instance view) |
| `d` | Destroy instance (in instance view) |
| `r` | Refresh data |
| `/` | Search |
| `f` | Filter |
| `?` | Toggle help |
| `q` | Quit |

## Documentation

- [User Guide](./USER_GUIDE.md) - Comprehensive user documentation
- [Developer Guide](./DEVELOPER_GUIDE.md) - Architecture and development guide

## Requirements

- Terminal with UTF-8 support
- 256 color terminal recommended
- Minimum 80x24 terminal size

## Architecture

Built with modern Go TUI libraries:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - The Elm Architecture for terminals
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI component library
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal UIs

## Status

The TUI implementation follows [ADR-0016](../adr/records/0016-web-ui-deprecation-and-tui-adoption.md) which deprecated the Web UI in favor of a Terminal UI for better performance and simplified architecture.