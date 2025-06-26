# 1. Container-based Background Execution

Date: 2025-06-26

## Status

Accepted

## Context

KECS initially required users to run the server process directly, which meant keeping a terminal session open or setting up systemd services. This approach had several limitations:

- Required terminal session to remain open
- Complex setup for background execution
- Difficult to manage multiple instances
- No built-in port conflict resolution
- Manual process management required

Similar tools in the Kubernetes ecosystem (kind, k3d, minikube) provide container-based execution, which has become the expected pattern for development tools.

## Decision

We will implement container-based background execution for KECS using Docker, providing simple commands to start, stop, and manage KECS instances in containers.

The implementation includes:

1. **Basic Container Commands**:
   - `kecs start` - Start KECS in a Docker container
   - `kecs stop` - Stop and remove the container
   - `kecs status` - Show container status
   - `kecs logs` - Display container logs

2. **Multiple Instance Support**:
   - Custom container names for multiple instances
   - Port conflict detection and auto-assignment
   - Container labeling for identification
   - Configuration file support

3. **Instance Management**:
   - `kecs instances list` - List all instances
   - `kecs instances start-all` - Start configured instances
   - `kecs instances stop-all` - Stop all instances

## Consequences

### Positive

- **Improved User Experience**: Simple commands familiar to Kubernetes developers
- **Background Execution**: No need to keep terminal sessions open
- **Multiple Instances**: Easy to run dev, staging, and test environments
- **Port Management**: Automatic port conflict resolution
- **Data Persistence**: Volume mounts preserve data between restarts
- **Container Isolation**: Each instance runs in its own container

### Negative

- **Docker Dependency**: Requires Docker to be installed and running
- **Additional Complexity**: More code to maintain for container management
- **Resource Overhead**: Each instance consumes container resources

### Implementation Details

The container implementation uses:
- Docker Go SDK for container management
- Health checks with configurable timeout
- Volume mounts for data persistence
- Environment variables for container mode detection
- Labels for instance identification
- YAML configuration files for multi-instance setups

## References

- Issue #252: Add container-based background execution
- Similar patterns in: kind, k3d, minikube
- Docker Go SDK documentation