# KECS New Architecture (v2)

## Overview

The new KECS architecture runs the control plane inside a k3d cluster, providing a unified AWS API endpoint accessible from all containers. This solves the previous limitation where application containers couldn't access KECS control plane APIs.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                   k3d Cluster                       │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │            kecs-system namespace             │   │
│  │                                              │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────┐ │   │
│  │  │    KECS    │  │ LocalStack │  │Traefik │ │   │
│  │  │Control Plane│  │            │  │Gateway │ │   │
│  │  └─────┬──────┘  └─────┬──────┘  └───┬────┘ │   │
│  │        │               │              │      │   │
│  │        └───────────────┴──────────────┘      │   │
│  │                       │                      │   │
│  └───────────────────────┼──────────────────────┘   │
│                          │                          │
│  ┌───────────────────────┼──────────────────────┐   │
│  │    Application namespace(s)                  │   │
│  │                       │                      │   │
│  │    ┌─────────────────▼─────────────────┐    │   │
│  │    │     Application Containers         │    │   │
│  │    │  (Can access AWS APIs via Traefik) │    │   │
│  │    └────────────────────────────────────┘    │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
                           │
                           │ :4566 (Traefik)
                     ┌─────▼─────┐
                     │ AWS CLI   │
                     │ /SDK      │
                     └───────────┘
```

## Quick Start

### Starting KECS v2

```bash
# Start KECS with new architecture
kecs start-v2

# Or with custom settings
kecs start-v2 --name my-cluster --api-port 4566 --admin-port 8081

# Check status
kubectl get all -n kecs-system
```

### Using KECS v2

All AWS API calls go through the unified endpoint on port 4566:

```bash
# Configure AWS CLI
export AWS_ENDPOINT_URL=http://localhost:4566
export AWS_REGION=us-east-1

# ECS operations (handled by KECS)
aws ecs create-cluster --cluster-name my-cluster
aws ecs list-clusters

# ELBv2 operations (handled by KECS)
aws elbv2 create-load-balancer --name my-alb --subnets subnet-12345

# Other AWS services (handled by LocalStack)
aws s3 mb s3://my-bucket
aws dynamodb create-table --table-name my-table
```

### Application Access

Applications running inside the cluster can access AWS APIs through the internal endpoint:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
spec:
  containers:
  - name: app
    image: my-app:latest
    env:
    - name: AWS_ENDPOINT_URL
      value: http://traefik.kecs-system.svc.cluster.local:4566
```

## Components

### 1. KECS Control Plane
- Provides ECS and ELBv2 APIs
- Deployed as a Deployment in kecs-system namespace
- Uses DuckDB for persistence
- Health checks on `/health` and `/ready`

### 2. Traefik Gateway
- Routes AWS API requests based on X-Amz-Target header
- ECS APIs → KECS control plane
- ELBv2 APIs → KECS control plane
- Other APIs → LocalStack
- Provides unified endpoint on port 4566

### 3. LocalStack
- Provides other AWS services (S3, DynamoDB, etc.)
- Deployed in kecs-system namespace
- Integrated with Traefik routing

## Migration from v1

The main differences from the previous architecture:

1. **Unified Endpoint**: Single endpoint (4566) for all AWS APIs
2. **In-Cluster Access**: Applications can access KECS APIs
3. **Simplified Networking**: No need for complex port forwarding
4. **Better Integration**: Seamless LocalStack integration

## Commands

### start-v2
Starts KECS with the new architecture:
```bash
kecs start-v2 [flags]

Flags:
  --name string        Cluster name (default "kecs")
  --api-port int       AWS API port (default 4566)
  --admin-port int     Admin API port (default 8081)
  --data-dir string    Data directory for persistence
  --no-localstack      Disable LocalStack deployment
  --no-traefik         Disable Traefik deployment
  --timeout duration   Timeout for cluster creation (default 10m)
```

### stop-v2
Stops and deletes the k3d cluster:
```bash
kecs stop-v2 [flags]

Flags:
  --name string     Cluster name to stop (default "kecs")
  --delete-data     Delete persistent data
```

### cluster
Manage k3d clusters:
```bash
kecs cluster create [name]    # Create a new cluster
kecs cluster delete [name]    # Delete a cluster
kecs cluster list             # List all clusters
kecs cluster info [name]      # Show cluster information
```

## Testing

Test the setup with the provided script:
```bash
./controlplane/scripts/test-v2-architecture.sh
```

Or manually test each component:
```bash
# Test routing
./controlplane/scripts/test-traefik-routing.sh

# Check component health
curl http://localhost:8081/health
curl http://localhost:8081/health/detailed
```

## Troubleshooting

### Check component status
```bash
kubectl get all -n kecs-system
kubectl logs -n kecs-system -l app=kecs-controlplane
kubectl logs -n kecs-system -l app=traefik
kubectl logs -n kecs-system -l app=localstack
```

### Port conflicts
If port 4566 is already in use:
```bash
kecs start-v2 --api-port 4567
```

### Access cluster directly
```bash
export KUBECONFIG=~/.k3d/kubeconfig-kecs.yaml
kubectl get nodes
```