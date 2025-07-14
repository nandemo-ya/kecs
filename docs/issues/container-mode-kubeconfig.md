# Issue: Container Mode kubeconfig Host Access

## Problem Description

When KECS runs in container mode, the kubeconfig files generated for k3d clusters use the container's internal hostname (e.g., `k3d-kecs-test-cluster2-server-0`) as the API server endpoint. This hostname is not resolvable from the host machine, making it impossible to use kubectl directly from the host.

## Current Behavior

```yaml
# Current kubeconfig in container mode
server: https://k3d-kecs-test-cluster2-server-0:6443
```

This results in errors when trying to use kubectl from the host:
```
dial tcp: lookup k3d-kecs-test-cluster2-server-0: no such host
```

## Expected Behavior

When KECS is running in container mode, it should generate kubeconfig files that work from both inside the container and from the host machine. The kubeconfig should use `localhost` with the appropriate mapped port.

```yaml
# Expected kubeconfig for host access
server: https://localhost:<mapped-port>
```

## Workaround

Currently, users need to manually edit the kubeconfig file to replace the k3d container hostname with localhost and the correct port.

## Proposed Solution

1. Detect when KECS is running in container mode
2. When generating/updating kubeconfig files, create two versions:
   - Internal version: Uses k3d container hostnames (for KECS container use)
   - External version: Uses localhost with mapped ports (for host machine use)
3. Provide a command or API to retrieve the host-compatible kubeconfig

## Implementation Notes

- The k3d cluster manager already knows about port mappings
- Need to track which ports are mapped to which k3d API server ports
- Consider adding a new endpoint or command: `kecs get-kubeconfig --cluster <name> --host-access`

## Related Code

- `/controlplane/internal/kubernetes/k3d_cluster_manager.go` - Handles kubeconfig generation
- `/controlplane/internal/controlplane/cmd/start.go` - Container mode configuration

## Priority

Medium - This affects developer experience but has a manual workaround.