# CLI Commands Reference

KECS provides a comprehensive CLI for managing local ECS environments. This reference covers all available commands and their options.

## Core Commands

### kecs start

Starts a new KECS instance with a k3d cluster.

```bash
kecs start [flags]
```

**Flags:**
- `--instance string`: Instance name (default: auto-generated)
- `--api-port int`: API port for ECS/ELBv2 APIs (default: 5373)
- `--admin-port int`: Admin port for health/metrics (default: 5374)
- `--data-dir string`: Data directory (default: ~/.kecs/data)
- `--config string`: Configuration file path
- `--additional-localstack-services string`: Additional LocalStack services (comma-separated)
- `--timeout duration`: Timeout for cluster creation (default: 10m)

**Examples:**
```bash
# Start with default settings
kecs start

# Start with custom instance name
kecs start --instance dev

# Start with custom ports
kecs start --instance staging --api-port 6373 --admin-port 6374

# Start without LocalStack
kecs start --no-localstack
```

### kecs stop

Stops and removes a KECS instance.

```bash
kecs stop [flags]
```

**Flags:**
- `--instance string`: Instance name to stop (required)

**Examples:**
```bash
# Stop specific instance
kecs stop --instance dev

# Stop another instance
kecs stop --instance staging
```

## Kubernetes Integration

### kecs kubeconfig

Manages kubeconfig for KECS clusters.

```bash
kecs kubeconfig [subcommand] [flags]
```

**Subcommands:**
- `list`: List all available KECS clusters
- `get`: Get kubeconfig for an instance

**Examples:**
```bash
# List all available KECS clusters
kecs kubeconfig list

# Get kubeconfig path
kecs kubeconfig get --instance dev
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
✅ Port 5373 is available
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
| `KECS_ADMIN_PORT` | Default admin port | 5374 |
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
kecs start --api-port 6373
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