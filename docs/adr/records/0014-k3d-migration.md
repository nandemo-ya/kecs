# ADR-0014: Migration from Kind to k3d for Local Kubernetes Clusters

## Status
Proposed

## Context
KECS currently uses Kind (Kubernetes in Docker) for creating local Kubernetes clusters to back ECS clusters. However, we've encountered several limitations:

1. **Docker CLI Dependency**: Kind requires Docker CLI to be available in the container, causing deployment complexity
2. **Performance**: Kind has slower startup times compared to alternatives
3. **Resource Usage**: Higher memory and CPU overhead
4. **Container Mode Issues**: Running Kind inside KECS containers requires Docker CLI installation

## Decision
We will migrate from Kind to k3d for local Kubernetes cluster management.

## Rationale

### k3d Advantages
1. **Docker API Only**: k3d uses Docker Engine API directly, no Docker CLI required
2. **Better Performance**: Faster cluster startup and lower resource usage
3. **Lightweight**: Based on k3s, which is designed for edge/IoT scenarios
4. **Container-Friendly**: Works seamlessly in container environments without additional dependencies
5. **Active Development**: Well-maintained by Rancher/SUSE with regular updates

### Technical Benefits
- **Smaller Container Images**: No need to install Docker CLI in KECS containers
- **Better Error Handling**: Structured API responses vs command-line parsing
- **Cross-Platform**: Consistent behavior across different host operating systems
- **Security**: Fewer binary dependencies reduce attack surface

## Implementation Plan

### Phase 1: Interface Abstraction
- Create `ClusterManager` interface to abstract cluster operations
- Implement interface for both Kind and k3d
- Allow runtime switching between implementations

### Phase 2: k3d Implementation
- Implement `K3dManager` using k3d Go SDK
- Add k3d dependencies to go.mod
- Implement all required cluster operations

### Phase 3: Testing & Validation
- Update scenario tests to work with k3d
- Performance benchmarking comparison
- Container mode validation

### Phase 4: Migration & Cleanup
- Switch default implementation to k3d
- Remove Kind dependencies
- Update documentation

## Consequences

### Positive
- Simplified container deployment (no Docker CLI needed)
- Better performance and resource usage
- Improved development experience
- Future-proof solution

### Negative
- Migration effort required
- Potential compatibility issues during transition
- Team needs to learn k3d specifics
- Temporary maintenance of two implementations

### Risks
- k3d API stability (mitigated by interface abstraction)
- Ecosystem compatibility differences between Kind and k3d
- Integration testing complexity during transition

## Migration Timeline
- Phase 1: 1-2 days (interface design and abstraction)
- Phase 2: 3-5 days (k3d implementation)
- Phase 3: 2-3 days (testing and validation)
- Phase 4: 1-2 days (migration and cleanup)

**Total Estimated Time**: 1-2 weeks

## Success Criteria
- [ ] KECS containers can create k3d clusters without Docker CLI
- [ ] Performance improvement measurable (startup time, resource usage)
- [ ] All existing functionality preserved
- [ ] Scenario tests pass with k3d
- [ ] Container mode works seamlessly