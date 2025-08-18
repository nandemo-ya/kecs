# KECS Registry Usage Guide

## Overview

KECS automatically creates a local Docker registry for each instance to facilitate development with locally built images. This registry can be accessed through multiple hostnames depending on where you're accessing it from.

## Registry Access Patterns

### From Host Machine (for pushing images)
```bash
# Push images using localhost:5000
docker tag myapp:latest localhost:5000/myapp:latest
docker push localhost:5000/myapp:latest
```

### From Within Kubernetes Cluster (for pulling images)
```yaml
# In your task definitions or Kubernetes manifests
image: registry.kecs.local:5000/myapp:latest
```

## Supported Registry Hostnames

The registry is accessible through the following hostnames:

| Hostname | Usage | Context |
|----------|-------|---------|
| `localhost:5000` | Pushing images from host machine | Development machine |
| `registry.kecs.local:5000` | **Recommended** for pulling images in Kubernetes | Pods/containers in cluster |
| `k3d-kecs-registry:5000` | Internal k3d container name | k3d internal networking (auto-configured) |

## Example Workflow

### 1. Build and Push Image (from host)
```bash
# Build your Docker image
docker build -t myapp:latest .

# Tag for local registry
docker tag myapp:latest localhost:5000/myapp:latest

# Push to KECS registry
docker push localhost:5000/myapp:latest
```

### 2. Use in ECS Task Definition
```json
{
  "family": "my-task",
  "containerDefinitions": [
    {
      "name": "my-container",
      "image": "registry.kecs.local:5000/myapp:latest",
      "memory": 512,
      "cpu": 256
    }
  ]
}
```

### 3. Use in Kubernetes Manifest
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      containers:
      - name: my-container
        image: registry.kecs.local:5000/myapp:latest
```

## Implementation Details

The registry configuration is automatically set up with:
- HTTP access (no TLS required for local development)
- Automatic DNS resolution via CoreDNS and /etc/hosts
- k3s registries.yaml configuration for proper mirror setup

## Troubleshooting

### Verify Registry is Running
```bash
# Check if registry container is running
docker ps | grep kecs-registry

# Test registry access from host
curl http://localhost:5000/v2/_catalog
```

### Check DNS Resolution in Cluster
```bash
# From within a pod
kubectl run test --rm -it --image=busybox -- sh
nslookup registry.kecs.local
```

### View Registry Contents
```bash
# List all repositories
curl http://localhost:5000/v2/_catalog

# List tags for a specific image
curl http://localhost:5000/v2/myapp/tags/list
```