# Container Commands Guide

This guide covers the container-based execution features of KECS, which allow you to run KECS in Docker containers similar to tools like kind and k3d.

## ⚠️ Security Notice

KECS requires Docker daemon access to manage local Kubernetes clusters. This provides significant capabilities including:
- Full access to Docker daemon (equivalent to root access)
- Ability to create, modify, and delete containers
- Access to the host filesystem through volume mounts

**Only use KECS in trusted environments**. See the [main README](../README.md#-security-notice---please-read) for detailed security information.

## Overview

KECS provides container commands that make it easy to:
- Run KECS in the background without keeping a terminal open
- Manage multiple instances with different configurations
- Automatically handle port conflicts
- Persist data between restarts

## Prerequisites

- Docker installed and running
- KECS binary in your PATH
- Sufficient permissions to run Docker commands
- A trusted environment (local development or CI/CD)

## Basic Usage

### Starting KECS

The simplest way to start KECS:

```bash
kecs start
```

This will:
- Pull the latest KECS image (if not present)
- Create a container named `kecs-server`
- Map ports 8080 (API) and 8081 (Admin)
- Mount `~/.kecs/data` for persistence
- Run in detached mode (background)

### Checking Status

To see if KECS is running:

```bash
kecs status
```

Output:
```
NAME                 STATUS     CREATED         PORTS                IMAGE                         
-----------------------------------------------------------------------------------------------
kecs-server          Up 2 mi... 2025-06-26 10:30 8080->8080, 8081-... ghcr.io/nandemo-ya/kecs:latest
```

### Viewing Logs

To see container logs:

```bash
# View recent logs
kecs logs

# Follow logs in real-time
kecs logs -f

# Show last 100 lines with timestamps
kecs logs --tail 100 -t
```

### Stopping KECS

To stop and remove the container:

```bash
kecs stop
```

The data directory is preserved when stopping.

## Advanced Usage

### Custom Ports

Run KECS on different ports:

```bash
kecs start --api-port 9080 --admin-port 9081
```

### Multiple Instances

Run multiple KECS instances with different names:

```bash
# Development instance
kecs start --name kecs-dev --api-port 8080 --admin-port 8081

# Staging instance
kecs start --name kecs-staging --api-port 8090 --admin-port 8091

# Test instance with auto-port assignment
kecs start --name kecs-test --auto-port
```

### Local Build

Build and use a local image:

```bash
kecs start --local-build
```

This builds the image from the current source code.

### Custom Data Directory

Specify a custom data directory:

```bash
kecs start --data-dir /path/to/data
```

## Configuration File

For complex setups, use a configuration file:

### Example Configuration

Create `~/.kecs/instances.yaml`:

```yaml
defaultInstance: dev

instances:
  - name: dev
    image: ghcr.io/nandemo-ya/kecs:latest
    ports:
      api: 8080
      admin: 8081
    dataDir: ~/.kecs/instances/dev/data
    autoStart: true
    env:
      KECS_LOG_LEVEL: debug
      KECS_TEST_MODE: "false"
    labels:
      environment: development
      team: backend

  - name: staging
    image: ghcr.io/nandemo-ya/kecs:v1.0.0
    ports:
      api: 8090
      admin: 8091
    dataDir: ~/.kecs/instances/staging/data
    autoStart: true
    env:
      KECS_LOG_LEVEL: info
    labels:
      environment: staging

  - name: test
    image: kecs:local
    ports:
      api: 8100
      admin: 8101
    dataDir: ~/.kecs/instances/test/data
    autoStart: false
    env:
      KECS_TEST_MODE: "true"
      KECS_LOG_LEVEL: warn
    labels:
      environment: test
```

### Using Configuration

Start a specific instance from config:

```bash
kecs start --config ~/.kecs/instances.yaml staging
```

Start the default instance:

```bash
kecs start --config ~/.kecs/instances.yaml
```

## Instance Management

### List All Instances

```bash
kecs instances list
```

Output shows both configured and running instances:

```
INSTANCE             STATUS     API PORT     ADMIN PORT   IMAGE                          DATA DIR
------------------------------------------------------------------------------------------------
dev *                running    8080         8081         ghcr.io/nandemo-ya/kecs:latest ~/.kecs/instances/dev/data
staging              running    8090         8091         ghcr.io/nandemo-ya/kecs:v1.0.0 ~/.kecs/instances/staging/data
test                 configured 8100         8101         kecs:local                     ~/.kecs/instances/test/data

* Default instance: dev
```

### Start All Instances

Start all instances with `autoStart: true`:

```bash
kecs instances start-all --config ~/.kecs/instances.yaml
```

### Stop All Instances

Stop all running KECS instances:

```bash
kecs instances stop-all
```

## Port Management

### Automatic Port Assignment

Use `--auto-port` to automatically find available ports:

```bash
kecs start --name test --auto-port
```

This will:
1. Check for used ports by other KECS instances
2. Find next available ports starting from defaults
3. Display assigned ports

### Port Conflict Detection

KECS automatically detects port conflicts:

```bash
$ kecs start --api-port 8080
Error: API port 8080 is already in use
```

## Troubleshooting

### Container Won't Start

1. Check Docker is running:
   ```bash
   docker version
   ```

2. Check for port conflicts:
   ```bash
   lsof -i :8080
   ```

3. Check container logs:
   ```bash
   docker logs kecs-server
   ```

### Permission Issues

If you get permission errors:

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Logout and login again
```

### Cleanup Orphaned Containers

If containers are left running:

```bash
# List all KECS containers
docker ps -a --filter "label=app=kecs"

# Remove all stopped KECS containers
docker container prune --filter "label=app=kecs"
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Test with KECS
on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Start KECS
        run: |
          kecs start --name ci-test --auto-port
          kecs status
      
      - name: Run Tests
        run: |
          # Your tests here
          npm test
      
      - name: Stop KECS
        run: kecs stop --name ci-test
```

### Docker Compose Example

```yaml
version: '3.8'

services:
  kecs:
    image: ghcr.io/nandemo-ya/kecs:latest
    container_name: kecs-compose
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      - kecs-data:/data
    environment:
      - KECS_CONTAINER_MODE=true
      - KECS_LOG_LEVEL=info
    labels:
      - app=kecs

volumes:
  kecs-data:
```

## Best Practices

1. **Use Named Instances**: Always use `--name` for better identification
2. **Configuration Files**: Use config files for reproducible setups
3. **Port Planning**: Plan port ranges for different environments
4. **Data Backup**: Regularly backup data directories
5. **Resource Limits**: Consider Docker resource limits for production use

## Environment Variables

When running in container mode, these environment variables are set:

- `KECS_CONTAINER_MODE=true` - Indicates container execution
- `KECS_DATA_DIR=/data` - Data directory inside container
- `KECS_API_PORT` - API server port
- `KECS_ADMIN_PORT` - Admin server port

## Security Considerations

1. **Network Isolation**: Containers run in Docker's default bridge network
2. **Data Permissions**: Data directory inherits host user permissions
3. **Port Exposure**: Only specified ports are exposed to host
4. **Image Security**: Use specific version tags in production

## Migration from Direct Execution

If you were running KECS directly:

1. Stop the direct process
2. Copy data to `~/.kecs/data` (or custom location)
3. Start with container command: `kecs start`
4. Verify data is accessible

## Summary

Container commands provide a convenient way to run KECS with:
- Simple start/stop commands
- Background execution
- Multiple instance support
- Automatic port management
- Data persistence
- Configuration file support

For more information, see the [main documentation](../README.md).