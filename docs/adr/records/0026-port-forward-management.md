# ADR-0026: Port Forward Management for KECS

## Status

Proposed

## Context

KECS needs to provide a way to access ECS services and tasks running in the local Kubernetes cluster from the host machine. The current challenges are:

1. **assignPublicIp not functional**: ECS's `assignPublicIp` flag is stored but not implemented
2. **Limited accessibility**: Services running in k3d cluster are not directly accessible from the host
3. **kubectl port-forward limitations**:
   - Runs in foreground by default, blocking the terminal
   - No persistent configuration management
   - Manual reconnection required when pods restart
   - Complex to manage multiple port forwards

4. **Developer experience**: Need for a simpler, more automated solution for local development

## Decision

We will implement a comprehensive `kecs port-forward` command that provides background port forwarding with automatic management and configuration file support.

### Architecture Overview

1. **NodePort Service Creation**:
   - When `assignPublicIp: ENABLED` is set in the task definition
   - Create Kubernetes Service with type NodePort instead of ClusterIP
   - Store NodePort information in the database

2. **k3d Integration**:
   - Use k3d's dynamic port configuration API (`k3d node edit`)
   - Map host ports to NodePort services dynamically
   - Handle serverlb container recreation (~10 seconds)

3. **Port Forward Agent**:
   - Single daemon process per KECS instance
   - Manages all port forwards through goroutines (not separate processes)
   - Provides gRPC/Unix socket API for CLI communication
   - Efficient resource usage with connection multiplexing
   - Automatic lifecycle management with instance start/stop

### Command Structure

#### Basic Commands

```bash
# Start a port forward for an ECS service
kecs port-forward start service <cluster>/<service-name> [--local-port <port>]

# Start a port forward for an ECS task
kecs port-forward start task <cluster>/<task-id> [--local-port <port>]

# List all active port forwards
kecs port-forward list

# Stop a specific port forward
kecs port-forward stop <forward-id>

# Stop all port forwards
kecs port-forward stop --all
```

#### Configuration File Commands

```bash
# Start all port forwards defined in a config file
kecs port-forward run --config <file-path>

# Stop all port forwards defined in a config file
kecs port-forward down --config <file-path>
```

### Configuration File Format

```yaml
# port-forward.yaml
version: "1"
forwards:
  - name: web-service
    type: service
    cluster: production
    service: webapp
    localPort: 8080
    targetPort: 80
    autoReconnect: true

  - name: api-service
    type: service
    cluster: production
    service: api
    localPort: 3000
    targetPort: 3000
    autoReconnect: true

  - name: debug-task
    type: task
    cluster: development
    tags:
      Environment: dev
      Component: worker
    localPort: 9229
    targetPort: 9229
    autoReconnect: false
```

### Auto-Reconnection and Discovery

1. **Service Discovery**:
   - ECS Services identified by cluster and service name
   - Automatic port forward to the service's NodePort

2. **Task Discovery**:
   - Tasks identified by ID or tags
   - When multiple tasks match tag criteria, automatically select the most recent one
   - Sort by task creation timestamp (newest first)
   - Provide option `--all` to error if multiple matches (strict mode)
   - Example: `tags: {Environment: dev, Component: api}`
   - Display selected task information for transparency

3. **Auto-Reconnection**:
   - Monitor target service/task health
   - Automatically reconnect when pods restart
   - Configurable retry logic with backoff
   - Optional notifications on reconnection

### Requirements and Constraints

1. **assignPublicIp Requirement**:
   - Tasks must have `assignPublicIp: ENABLED` for port forwarding
   - This triggers NodePort service creation
   - Without this flag, tasks use ClusterIP and are not accessible

2. **Port Management**:
   - Track allocated ports to prevent conflicts
   - Support automatic port selection with `--local-port 0`
   - Validate port availability before allocation

3. **Instance Isolation**:
   - Port forwards are scoped to KECS instances
   - Each instance maintains its own port forward state

### State Persistence

```
~/.kecs/instances/{instance-name}/
├── port-forwards/
│   ├── agent.sock          # Unix socket for agent communication
│   ├── agent.pid           # Agent process ID
│   ├── config.yaml         # Active port forward configurations
│   ├── state.yaml          # Agent state and connection tracking
│   └── logs/              # Port forward logs
│       ├── agent.log       # Agent daemon logs
│       ├── web-service.log
│       └── api-service.log
```

### Agent Architecture

1. **Agent Lifecycle**:
   - Automatically started when first port-forward command is issued
   - Runs as daemon process bound to KECS instance
   - Gracefully shuts down when instance stops
   - Survives individual port-forward connection failures

2. **Communication Protocol**:
   - Unix socket for low-latency local communication
   - Optional gRPC for future remote management
   - JSON-RPC over socket for simple CLI integration

3. **Resource Management**:
   - Single process with goroutine pool
   - Connection multiplexing for efficient port usage
   - Automatic cleanup of stale connections
   - Memory-efficient streaming for large transfers

### Integration Points

1. **Service Creation**:
   - Modify `ServiceManager` to create NodePort when `assignPublicIp: ENABLED`
   - Store NodePort information in DuckDB

2. **k3d Management**:
   - Use existing k3d Go library (already in dependencies)
   - Implement wrapper for `k3d node edit` operations

3. **CLI Integration**:
   - Add `port-forward` command to main KECS CLI
   - Integrate with existing instance management

## Consequences

### Positive

1. **Better Developer Experience**:
   - Background execution doesn't block terminal
   - Configuration files enable reproducible setups
   - Auto-reconnection reduces manual intervention

2. **ECS Compatibility**:
   - Implements `assignPublicIp` functionality
   - Maintains ECS semantics for service access

3. **Flexibility**:
   - Supports both imperative and declarative approaches
   - Tag-based task discovery for dynamic environments
   - Automatic selection of newest task improves resilience

4. **Production-like Testing**:
   - Can simulate public IP assignment locally
   - Compatible with ELBv2 for advanced scenarios

5. **Resilience**:
   - Handles task restarts gracefully by selecting newest task
   - Reduces brittleness when tasks are recreated

6. **Resource Efficiency**:
   - Single agent process instead of multiple port-forward processes
   - Goroutine-based connection management
   - Reduced system overhead with connection pooling

### Negative

1. **Complexity**:
   - Adds agent daemon management complexity
   - Requires Unix socket/IPC communication
   - More complex than simple process spawning

2. **Performance**:
   - k3d node edit causes brief serverlb restart (~10 seconds)
   - Agent process consumes memory even when idle

3. **Platform Dependencies**:
   - Requires Docker Desktop or compatible Docker environment
   - k3d version compatibility requirements
   - Unix socket support required (may affect Windows compatibility)

### Alternatives Considered

1. **kubectl port-forward from controlplane**: Only works within container, not accessible from host
2. **Docker socket mount to controlplane**: Security concerns with container accessing Docker API
3. **LoadBalancer with MetalLB**: Overly complex for local development use case
4. **SSH tunneling**: Additional complexity and security considerations
5. **Multiple process approach**: Less efficient resource usage, harder to manage state
6. **Direct TCP proxy in controlplane**: Would require exposing controlplane ports to host

## Implementation Plan

### Phase 1: Core Infrastructure
- [ ] Implement NodePort service creation for `assignPublicIp: ENABLED`
- [ ] Add port management to ServiceManager
- [ ] Create port forward state storage

### Phase 2: Basic Commands
- [ ] Implement `kecs port-forward start` for services
- [ ] Implement `kecs port-forward list`
- [ ] Implement `kecs port-forward stop`

### Phase 3: Advanced Features
- [ ] Add task-based port forwarding
- [ ] Implement tag-based task discovery
- [ ] Add configuration file support

### Phase 4: Auto-Management
- [ ] Implement auto-reconnection logic
- [ ] Add health monitoring
- [ ] Create background process management

### Phase 5: Polish and Documentation
- [ ] Add comprehensive tests
- [ ] Write user documentation
- [ ] Add examples and tutorials

## References

- Issue #588: Implement assignPublicIp support with kecs port-forward command
- ECS assignPublicIp documentation
- k3d port configuration: https://k3d.io/v5.0.0/usage/exposing_services/
- kubectl port-forward documentation
- Related code: `controlplane/internal/kubernetes/service_manager.go`