# ADR-0010: Docker Proxy Support for Kind Containers

Date: 2025-06-07

## Status

Proposed

## Context

Users familiar with Docker but not Kubernetes need a way to view and manage containers running inside Kind clusters. Currently, they must:
- Use `kubectl` commands (requires Kubernetes knowledge)
- Execute `crictl` commands inside Kind nodes
- Cannot use familiar `docker ps`, `docker logs`, etc.

## Decision

Implement Docker command proxy functionality in KECS that:

1. **Docker API Compatibility Layer**
   - Expose Docker Engine API-compatible endpoints
   - Translate Docker API calls to Kind/crictl operations
   - Support common Docker commands (ps, logs, exec, inspect)

2. **Implementation Approaches**

   ### Option A: Docker CLI Plugin
   - Create `docker-kecs` plugin for Docker CLI
   - Users can run: `docker kecs ps`, `docker kecs logs <container>`
   - Leverages existing Docker CLI infrastructure

   ### Option B: Standalone CLI with Docker-like Interface
   - Create `kecs docker` subcommand
   - Mimics Docker CLI syntax: `kecs docker ps`, `kecs docker logs`
   - Full control over implementation

   ### Option C: Docker Engine API Proxy
   - Implement Docker Engine API endpoints in KECS
   - Users can set `DOCKER_HOST=tcp://localhost:2375`
   - Existing Docker tools work transparently

## Proposed Solution

Implement a hybrid approach:

1. **Phase 1**: Standalone CLI (`kecs docker` subcommand)
   - Quick to implement
   - No Docker CLI modifications needed
   - Good for testing the concept

2. **Phase 2**: Docker Engine API subset
   - Implement key Docker API endpoints
   - Enable `DOCKER_HOST` compatibility
   - Support for Docker ecosystem tools

3. **Phase 3**: Docker CLI plugin (optional)
   - Native Docker CLI integration
   - Best user experience

## Implementation Details

### API Endpoints (Phase 2)
```
GET  /containers/json                 # docker ps
GET  /containers/{id}/logs           # docker logs
POST /containers/{id}/exec           # docker exec
GET  /containers/{id}/json           # docker inspect
```

### Command Mapping
```
docker ps     -> crictl ps
docker logs   -> crictl logs
docker exec   -> crictl exec
docker inspect -> crictl inspect
```

### Cluster Selection
- Environment variable: `KECS_CLUSTER=<cluster-name>`
- CLI flag: `--cluster <cluster-name>`
- Default: active Kind cluster

## Consequences

### Positive
- Familiar interface for Docker users
- No Kubernetes knowledge required
- Leverages existing Docker ecosystem
- Gradual migration path to Kubernetes

### Negative
- Additional complexity in KECS
- Not all Docker commands map cleanly
- Potential confusion about container runtime differences
- Maintenance burden for API compatibility

## References
- [Docker Engine API](https://docs.docker.com/engine/api/)
- [CRI-tools (crictl)](https://github.com/kubernetes-sigs/cri-tools)
- [Docker CLI Plugins](https://docs.docker.com/engine/extend/plugin_api/)