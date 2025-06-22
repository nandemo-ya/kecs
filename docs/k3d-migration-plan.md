# KECS k3d Migration Plan

## Overview
This document outlines the plan to migrate KECS from Kind to k3d for local Kubernetes cluster management.

## Background
KECS currently uses Kind (Kubernetes in Docker) but faces challenges:
- Requires Docker CLI inside containers
- Slower performance compared to alternatives
- Container deployment complexity

k3d offers better performance and uses Docker API directly (no CLI dependency).

## Implementation Status

### ✅ Phase 1: Interface Abstraction (COMPLETED)
- [x] Created `ClusterManager` interface for cluster operations
- [x] Implemented `KindClusterManager` wrapper for existing Kind functionality
- [x] Implemented `K3dClusterManager` for k3d operations
- [x] Added factory method `NewClusterManager()` for runtime switching

### 🔄 Phase 2: k3d Implementation (IN PROGRESS)
- [x] Added k3d dependencies to go.mod
- [x] Implemented basic k3d cluster operations (create/delete/exists)
- [x] Implemented kubeconfig management for container mode
- [x] Added cluster info and readiness checking
- [ ] Test k3d implementation in isolation
- [ ] Handle edge cases and error scenarios

### ⏳ Phase 3: Integration & Testing (PENDING)
- [ ] Update existing code to use ClusterManager interface
- [ ] Update scenario tests for k3d compatibility
- [ ] Performance benchmarking (k3d vs Kind)
- [ ] Container mode validation
- [ ] Cross-platform testing

### ⏳ Phase 4: Migration & Cleanup (PENDING)
- [ ] Switch default implementation to k3d
- [ ] Update configuration and documentation
- [ ] Remove Kind dependencies (optional)
- [ ] Update Docker images and deployment scripts

## File Structure

```
controlplane/internal/kubernetes/
├── cluster_manager.go           # Interface definition and factory
├── kind_cluster_manager.go      # Kind implementation (wrapper)
├── k3d_cluster_manager.go       # k3d implementation (new)
├── kind_manager.go             # Original Kind manager (legacy)
└── service_manager.go          # Uses ClusterManager interface
```

## API Compatibility

The new `ClusterManager` interface maintains compatibility with existing code:

```go
// Old usage
kindManager := kubernetes.NewKindManager()
kindManager.CreateCluster(ctx, "my-cluster")

// New usage
clusterManager, _ := kubernetes.NewClusterManager(&kubernetes.ClusterManagerConfig{
    Provider: "k3d", // or "kind"
})
clusterManager.CreateCluster(ctx, "my-cluster")
```

## Configuration

### Environment Variables
- `KECS_CLUSTER_PROVIDER`: Set to "k3d" or "kind" (default: "k3d")
- `KECS_CONTAINER_MODE`: Enable container mode (existing)
- `KECS_KUBECONFIG_PATH`: Custom kubeconfig directory (existing)

### Example Configuration
```go
config := &kubernetes.ClusterManagerConfig{
    Provider:      "k3d",
    ContainerMode: true,
    KubeconfigPath: "/kecs/kubeconfig",
}
```

## Migration Benefits

### Immediate Benefits
- **No Docker CLI dependency**: Resolves container deployment issues
- **Better performance**: Faster cluster creation and deletion
- **Lower resource usage**: k3s is more lightweight than full Kubernetes

### Long-term Benefits
- **Improved developer experience**: Faster local development cycles
- **Simplified deployment**: Smaller container images, fewer dependencies
- **Better maintainability**: More stable API, active development

## Testing Strategy

### Unit Tests
- Test ClusterManager interface implementations
- Test configuration and factory methods
- Test error handling and edge cases

### Integration Tests
- Update scenario tests to use k3d
- Test container mode functionality
- Validate kubeconfig generation and networking

### Performance Tests
- Compare cluster creation/deletion times
- Measure resource usage (CPU, memory)
- Test concurrent cluster operations

## Rollback Plan

If issues arise, we can easily rollback:
1. Change default provider from "k3d" to "kind"
2. Set `KECS_CLUSTER_PROVIDER=kind` environment variable
3. The Kind implementation remains fully functional

## Next Steps

1. **Test k3d implementation**:
   ```bash
   cd controlplane
   go test ./internal/kubernetes -run TestK3dClusterManager
   ```

2. **Update service managers**:
   ```go
   // Replace KindManager with ClusterManager
   clusterManager, _ := kubernetes.NewClusterManager(config)
   serviceManager := kubernetes.NewServiceManager(storage, clusterManager)
   ```

3. **Run scenario tests**:
   ```bash
   KECS_CLUSTER_PROVIDER=k3d make test
   ```

4. **Performance benchmarking**:
   ```bash
   ./scripts/benchmark-cluster-managers.sh
   ```

## Timeline

- **Week 1**: Complete k3d implementation and unit tests
- **Week 2**: Integration testing and scenario test updates
- **Week 3**: Performance testing and optimization
- **Week 4**: Documentation and migration

## Risk Mitigation

- **Interface abstraction**: Allows easy switching between implementations
- **Gradual migration**: Can test k3d alongside Kind
- **Comprehensive testing**: Validates functionality before migration
- **Documentation**: Clear migration path and troubleshooting guide