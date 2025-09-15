# CLI Commands Reference

KECS provides a comprehensive CLI for managing local ECS environments. This reference covers all available commands and their options.

## Global Flags

These flags can be used with any KECS command:

```bash
--config string     Config file (default is $HOME/.kecs/config.yaml)
--debug            Enable debug logging
--help             Help for the command
--version          Show version information
```

## Core Commands

### kecs start

Starts a new KECS instance with a k3d cluster.

```bash
kecs start [flags]
```

**Flags:**
- `--instance string`: Instance name (default: auto-generated)
- `--api-port int`: API port for ECS/ELBv2 APIs (default: 5373)
- `--admin-port int`: Admin port for health/metrics (default: 8081)
- `--localstack-port int`: LocalStack port (default: 4566)
- `--no-localstack`: Disable LocalStack integration
- `--no-traefik`: Disable Traefik gateway
- `--image string`: KECS Docker image to use
- `--k3d-args string`: Additional arguments for k3d cluster create

**Examples:**
```bash
# Start with default settings
kecs start

# Start with custom instance name
kecs start --instance dev

# Start with custom ports
kecs start --instance staging --api-port 8080 --admin-port 8081

# Start without LocalStack
kecs start --no-localstack
```

### kecs stop

Stops and removes a KECS instance.

```bash
kecs stop [flags]
```

**Flags:**
- `--instance string`: Instance name to stop
- `--all`: Stop all running instances
- `--force`: Force stop without confirmation

**Examples:**
```bash
# Stop with interactive selection
kecs stop

# Stop specific instance
kecs stop --instance dev

# Stop all instances
kecs stop --all
```

### kecs restart

Restarts a KECS instance.

```bash
kecs restart [flags]
```

**Flags:**
- `--instance string`: Instance name to restart

**Examples:**
```bash
# Restart with interactive selection
kecs restart

# Restart specific instance
kecs restart --instance dev
```

## Status Commands

### kecs status

Shows the status of KECS instances.

```bash
kecs status [flags]
```

**Flags:**
- `--instance string`: Show status for specific instance
- `--all`: Show status for all instances
- `--json`: Output in JSON format

**Examples:**
```bash
# Show status with interactive selection
kecs status

# Show status for specific instance
kecs status --instance dev

# Show all instances in JSON
kecs status --all --json
```

### kecs cluster info

Displays detailed information about a KECS cluster.

```bash
kecs cluster info [flags]
```

**Flags:**
- `--instance string`: Instance name
- `--kubeconfig`: Show kubeconfig path

**Example output:**
```
KECS Cluster Information:
========================
Instance: kecs-brave-wilson
Status: Running
API Endpoint: http://localhost:4566
Admin Endpoint: http://localhost:8081
Kubernetes Context: k3d-kecs-brave-wilson
Created: 2024-01-15 10:30:00
Uptime: 2h 15m
```

### kecs cluster list

Lists all KECS instances.

```bash
kecs cluster list [flags]
```

**Flags:**
- `--format string`: Output format (table, json, yaml)
- `--running`: Show only running instances
- `--stopped`: Show only stopped instances

**Examples:**
```bash
# List all instances
kecs cluster list

# List running instances in JSON
kecs cluster list --running --format json
```

## Log Commands

### kecs logs

Displays logs from KECS control plane.

```bash
kecs logs [flags]
```

**Flags:**
- `--instance string`: Instance name
- `-f, --follow`: Follow log output
- `--tail int`: Number of lines to show (default: 100)
- `--since string`: Show logs since timestamp (e.g., 2m, 1h)
- `--component string`: Filter by component (api, admin, storage)

**Examples:**
```bash
# Show last 100 lines
kecs logs

# Follow logs in real-time
kecs logs -f

# Show logs from last 5 minutes
kecs logs --since 5m

# Show only API server logs
kecs logs --component api
```

## Kubernetes Integration

### kecs kubeconfig

Manages kubeconfig for KECS clusters.

```bash
kecs kubeconfig [subcommand] [flags]
```

**Subcommands:**
- `get`: Get kubeconfig for an instance
- `merge`: Merge kubeconfig into ~/.kube/config
- `export`: Export kubeconfig to file

**Examples:**
```bash
# Get kubeconfig path
kecs kubeconfig get --instance dev

# Merge into default kubeconfig
kecs kubeconfig merge --instance dev

# Export to custom file
kecs kubeconfig export --instance dev --output ./dev-kubeconfig
```

### kecs kubectl

Runs kubectl commands against KECS cluster.

```bash
kecs kubectl [kubectl-args] [flags]
```

**Examples:**
```bash
# Get pods in kecs-system namespace
kecs kubectl get pods -n kecs-system

# Describe KECS deployment
kecs kubectl describe deployment kecs-controlplane -n kecs-system

# Get all resources
kecs kubectl get all --all-namespaces
```

## Development Commands

### kecs server

Runs KECS server directly (without k3d).

```bash
kecs server [flags]
```

**Flags:**
- `--port int`: API server port (default: 8080)
- `--admin-port int`: Admin server port (default: 8081)
- `--kubeconfig string`: Path to kubeconfig file
- `--namespace string`: Kubernetes namespace (default: kecs-system)
- `--storage string`: Storage backend (duckdb, memory)
- `--storage-path string`: Path for storage files

**Examples:**
```bash
# Run with default settings
kecs server

# Run with custom ports
kecs server --port 9090 --admin-port 9091

# Run with memory storage
kecs server --storage memory
```

### kecs dev

Development utilities for KECS.

```bash
kecs dev [subcommand] [flags]
```

**Subcommands:**
- `reload`: Hot reload control plane
- `build`: Build and push development image
- `reset`: Reset development environment

**Examples:**
```bash
# Hot reload after code changes
kecs dev reload

# Build and deploy dev image
kecs dev build --push

# Reset development cluster
kecs dev reset --instance dev
```

## Interactive TUI

### kecs tui

Launches the Terminal User Interface.

```bash
kecs tui [flags]
```

**Flags:**
- `--instance string`: Connect to specific instance
- `--theme string`: Color theme (default, dark, light)
- `--refresh int`: Refresh interval in seconds (default: 5)

**Features:**
- Browse clusters, services, and tasks
- View real-time status updates
- Check logs and events
- Manage resources interactively
- Keyboard shortcuts for navigation

**Key Bindings:**
- `↑/↓`: Navigate items
- `Enter`: Select/expand item
- `Tab`: Switch panels
- `l`: View logs
- `d`: Delete resource
- `r`: Refresh
- `q`: Quit

## Configuration Management

### kecs config

Manages KECS configuration.

```bash
kecs config [subcommand] [flags]
```

**Subcommands:**
- `init`: Initialize configuration
- `show`: Display current configuration
- `set`: Set configuration value
- `get`: Get configuration value

**Examples:**
```bash
# Initialize config
kecs config init

# Show all configuration
kecs config show

# Set default instance
kecs config set default.instance dev

# Get API endpoint
kecs config get api.endpoint
```

## Utility Commands

### kecs version

Shows version information.

```bash
kecs version [flags]
```

**Flags:**
- `--short`: Show only version number
- `--json`: Output in JSON format

**Example output:**
```
KECS Version: v0.5.0
Git Commit: abc123def
Build Date: 2024-01-15
Go Version: go1.21.5
Platform: darwin/arm64
```

### kecs doctor

Checks system requirements and diagnoses issues.

```bash
kecs doctor [flags]
```

**Checks:**
- Docker daemon status
- k3d installation
- Port availability
- Storage permissions
- Network connectivity

**Example output:**
```
✅ Docker daemon is running
✅ k3d is installed (v5.4.6)
✅ Port 4566 is available
✅ Storage directory is writable
✅ Network connectivity OK

All checks passed!
```

### kecs cleanup

Cleans up orphaned resources.

```bash
kecs cleanup [flags]
```

**Flags:**
- `--dry-run`: Show what would be cleaned
- `--force`: Skip confirmation
- `--all`: Clean all resources including data

**Examples:**
```bash
# Preview cleanup
kecs cleanup --dry-run

# Clean orphaned clusters
kecs cleanup

# Clean everything including data
kecs cleanup --all --force
```

## Environment Variables

KECS respects the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|  
| `KECS_INSTANCE` | Default instance name | - |
| `KECS_API_PORT` | Default API port | 5373 |
| `KECS_ADMIN_PORT` | Default admin port | 8081 |
| `KECS_NAMESPACE` | Kubernetes namespace | kecs-system |
| `KECS_LOG_LEVEL` | Log level (debug, info, warn, error) | info |
| `KECS_STORAGE_PATH` | Storage directory | ~/.kecs/data |
| `KECS_CONFIG_PATH` | Config file path | ~/.kecs/config.yaml |
| `KECS_LOCALSTACK_ENABLED` | Enable LocalStack | true |
| `KECS_FEATURES_TRAEFIK` | Enable Traefik | true |

## AWS CLI Integration

To use AWS CLI with KECS:

```bash
# Set endpoint URL
export AWS_ENDPOINT_URL=http://localhost:5373

# Configure dummy credentials
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1

# Use AWS CLI normally
aws ecs list-clusters
aws elbv2 describe-load-balancers
```

## Troubleshooting

### Common Issues

**Port already in use:**
```bash
# Check what's using the port
lsof -i :5373

# Use different port
kecs start --api-port 8080
```

**k3d cluster not found:**
```bash
# List k3d clusters
k3d cluster list

# Clean up orphaned cluster
k3d cluster delete kecs-<instance>
```

**Permission denied:**
```bash
# Fix Docker permissions
sudo usermod -aG docker $USER
newgrp docker
```

### Debug Mode

Enable debug logging for troubleshooting:

```bash
# Via flag
kecs start --debug

# Via environment variable
export KECS_LOG_LEVEL=debug
kecs start
```

## Next Steps

- [Services Guide](/guides/services) - Deploy ECS services
- [ELBv2 Integration](/guides/elbv2-integration) - Configure load balancers
- [TUI Interface](/guides/tui-interface) - Interactive management
- [Troubleshooting Guide](/guides/troubleshooting) - Resolve common issues