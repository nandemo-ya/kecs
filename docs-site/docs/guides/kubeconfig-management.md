# Kubeconfig Management

This guide explains how to use KECS's kubeconfig command to access k3d clusters created by KECS.

## Overview

When KECS creates k3d clusters, the generated kubeconfig files often require manual fixes to work properly from your local machine. The `kecs kubeconfig` command automatically handles these fixes for you.

## Common Issues with k3d Kubeconfig

k3d-generated kubeconfig files may have these issues when accessed from outside containers:

1. **host.docker.internal**: This hostname is not resolvable on Linux hosts
2. **Incorrect ports**: The API server port may not be properly exposed
3. **Empty port values**: Sometimes the port is missing, leaving just a trailing colon

## Using the kubeconfig Command

### List Available Clusters

To see all KECS clusters with corresponding k3d clusters:

```bash
kecs kubeconfig list
```

Example output:
```
Available KECS clusters:
  test-cluster
  microservices-cluster
```

### Get Kubeconfig for a Cluster

To get a properly configured kubeconfig:

```bash
# Print to stdout
kecs kubeconfig get test-cluster

# Save to a file
kecs kubeconfig get test-cluster -o ~/.kube/kecs-test-cluster

# Use directly with kubectl
kecs kubeconfig get test-cluster | kubectl --kubeconfig=/dev/stdin get nodes
```

### Get Raw k3d Kubeconfig

If you need the original k3d kubeconfig without fixes:

```bash
kecs kubeconfig get test-cluster --raw
```

## What the Command Fixes

The `kecs kubeconfig get` command automatically:

1. Replaces `host.docker.internal` with `127.0.0.1`
2. Extracts the correct API server port from the k3d loadbalancer container
3. Fixes malformed server URLs (e.g., `https://host.docker.internal:` → `https://127.0.0.1:50715`)

## Integration with kubectl

### Using Environment Variable

```bash
# Set KUBECONFIG environment variable
export KUBECONFIG=$(kecs kubeconfig get test-cluster -o /tmp/kecs-test.kubeconfig && echo /tmp/kecs-test.kubeconfig)

# Now kubectl will use this config by default
kubectl get nodes
```

### Using kubectl Context

```bash
# Save kubeconfig and merge with existing config
kecs kubeconfig get test-cluster -o ~/.kube/kecs-test-cluster
export KUBECONFIG=~/.kube/config:~/.kube/kecs-test-cluster
kubectl config view --flatten > ~/.kube/config.new
mv ~/.kube/config.new ~/.kube/config

# Use the context
kubectl config use-context k3d-kecs-test-cluster
```

### One-liner for Quick Access

```bash
# Create an alias for quick access
alias kube-test='kubectl --kubeconfig=<(kecs kubeconfig get test-cluster)'

# Use it
kube-test get pods -A
```

## Troubleshooting

### Cluster Not Found

If you get an error like "k3d cluster 'kecs-test-cluster' does not exist":

1. Check if the cluster exists in KECS:
   ```bash
   curl -s http://localhost:8080/v1/DescribeClusters | jq
   ```

2. Check if the k3d cluster exists:
   ```bash
   k3d cluster list
   ```

3. If the k3d cluster is missing, KECS may need to recreate it:
   ```bash
   # Restart KECS to trigger cluster recreation
   kecs stop && kecs start
   ```

### Connection Refused

If kubectl commands fail with "connection refused":

1. Verify the k3d cluster is running:
   ```bash
   docker ps | grep k3d-kecs
   ```

2. Check if the API server port is exposed:
   ```bash
   docker ps --format "table {{.Names}}\t{{.Ports}}" | grep serverlb
   ```

### Port Extraction Failed

If the command fails to extract the API server port:

1. Check the loadbalancer container name:
   ```bash
   docker ps --format "{{.Names}}" | grep serverlb
   ```

2. Manually get the port:
   ```bash
   docker ps --format "{{.Ports}}" --filter "name=k3d-kecs-.*-serverlb"
   ```

## Best Practices

1. **Save kubeconfig files**: Instead of running the command each time, save the kubeconfig to a file
2. **Use specific contexts**: When working with multiple clusters, use kubectl contexts to switch between them
3. **Automate with scripts**: Create shell scripts or aliases for frequently accessed clusters

## Related Commands

- `kecs cluster create`: Create a new ECS cluster
- `k3d cluster list`: List all k3d clusters
- `kubectl config`: Manage kubectl configuration