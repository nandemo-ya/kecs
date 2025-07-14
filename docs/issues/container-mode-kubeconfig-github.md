# Container Mode: kubeconfig not accessible from host machine

## Description

When running KECS in container mode (`kecs start`), the generated kubeconfig files use internal container hostnames that are not resolvable from the host machine, preventing direct kubectl access from the host.

## Steps to Reproduce

1. Start KECS in container mode:
   ```bash
   kecs start
   ```

2. Create a cluster:
   ```bash
   aws --endpoint-url http://localhost:8080 ecs create-cluster --cluster-name test-cluster
   ```

3. Try to access the cluster from the host:
   ```bash
   docker exec kecs-server cat /data/kubeconfig/kecs-test-cluster.config > ~/.kube/config
   kubectl get nodes
   ```

## Actual Result

```
Error from server (BadRequest): dial tcp: lookup k3d-kecs-test-cluster-server-0: no such host
```

## Expected Result

kubectl commands should work from the host machine when using the exported kubeconfig.

## Environment

- KECS Version: latest
- OS: macOS/Linux
- Container Runtime: Docker

## Proposed Solution

Generate host-compatible kubeconfig files that use `localhost:<mapped-port>` instead of internal container hostnames when KECS runs in container mode.

## Workaround

Manually edit the kubeconfig file to replace the k3d container hostname with `localhost` and the appropriate port (usually found in k3d cluster list).

## Additional Context

This is particularly important for developer workflows where users want to interact with the k3d clusters created by KECS directly from their host machine for debugging or development purposes.