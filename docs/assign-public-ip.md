# AssignPublicIp Support in KECS

## Overview

KECS supports the `assignPublicIp` parameter for ECS tasks, enabling external access to containers without requiring `kubectl port-forward`. This feature dynamically allocates host ports and creates Kubernetes NodePort services to expose containers to the host network.

## How It Works

When a task is run with `assignPublicIp: ENABLED`, KECS:

1. **Allocates Host Ports**: Dynamically assigns available ports from the range 32000-32999
2. **Creates NodePort Service**: Maps host ports to NodePort range 30000-30999
3. **Configures k3d Mapping**: Requires k3d cluster to have port range pre-configured

### Port Mapping Flow

```
External Client → Host Port (32000-32999) → k3d → NodePort (30000-30999) → Pod Container Port
```

## Prerequisites

### k3d Cluster Configuration

The k3d cluster must be started with the appropriate port range mapping:

```bash
k3d cluster create kecs-cluster \
  --port "32000-32999:30000-30999@server:0" \
  --port "8080:8080@server:0" \
  --port "5373:5373@server:0"
```

**Important**: Port mappings cannot be added dynamically to a running k3d cluster from within the control plane container. They must be configured when the cluster is created or added manually from the host.

## Usage Example

### 1. Create a Task Definition

```json
{
  "family": "web-app",
  "containerDefinitions": [
    {
      "name": "nginx",
      "image": "nginx:latest",
      "portMappings": [
        {
          "containerPort": 80,
          "protocol": "tcp"
        }
      ]
    }
  ]
}
```

### 2. Run Task with AssignPublicIp

```bash
aws ecs run-task \
  --cluster my-cluster \
  --task-definition web-app:1 \
  --network-configuration '{
    "awsvpcConfiguration": {
      "assignPublicIp": "ENABLED"
    }
  }' \
  --endpoint-url http://localhost:8080
```

### 3. Access the Service

After the task starts, KECS will return the allocated host ports in the task details:

```bash
# Get task details to find allocated ports
aws ecs describe-tasks \
  --cluster my-cluster \
  --tasks <task-id> \
  --endpoint-url http://localhost:8080

# Access the service (example with port 32000)
curl http://localhost:32000
```

## Implementation Details

### Port Allocation

- **Host Port Range**: 32000-32999 (1000 ports available)
- **NodePort Range**: 30000-30999 (mapped 1:1 with host ports)
- **Allocation Strategy**: Thread-safe sequential allocation with mutex locks
- **Port Tracking**: Maintains task-to-port mappings for cleanup on task termination

### Kubernetes Resources

For each task with `assignPublicIp: ENABLED`, KECS creates:

1. **Pod**: Standard pod with container definitions
2. **NodePort Service**: Exposes pod ports as NodePorts
   - Service name: `task-<task-id>`
   - Labels:
     - `kecs.dev/task-id`: Task ID
     - `kecs.dev/managed-by`: "kecs"
     - `kecs.dev/type`: "task-nodeport"
   - Annotations:
     - `kecs.dev/task-arn`: Full task ARN

### Limitations

1. **Static k3d Port Range**: k3d port mappings must be configured at cluster creation time
2. **Port Range Limit**: Maximum 1000 concurrent tasks with public IPs
3. **No Dynamic Port Addition**: Cannot add new port mappings to running k3d cluster from container
4. **Port Cleanup**: Released ports remain mapped in k3d until cluster recreation

## Troubleshooting

### Task Cannot Be Accessed Externally

1. **Verify k3d port mapping**:
   ```bash
   docker inspect k3d-<cluster>-serverlb | grep -A 10 "Ports"
   ```

2. **Check NodePort service**:
   ```bash
   kubectl get svc -n <cluster>-<region> task-<task-id>
   ```

3. **Verify pod is running**:
   ```bash
   kubectl get pods -n <cluster>-<region> -l kecs.dev/task-id=<task-id>
   ```

### Manual Port Mapping (Workaround)

If the k3d cluster wasn't created with port mappings, you can add them manually from the host:

```bash
# Add single port mapping
k3d node edit k3d-<cluster>-serverlb --port-add 32000:30000

# Add range (must be done individually for each port)
for i in {32000..32010}; do
  k3d node edit k3d-<cluster>-serverlb --port-add $i:$((i-2000))
done
```

**Note**: This must be done from the host machine, not from within the KECS container.

## Testing

A test script is provided at `examples/test-assignpublicip.sh` that demonstrates:

1. Creating a cluster
2. Registering a task definition with port mappings
3. Running a task with assignPublicIp enabled
4. Verifying external access

Run the test:
```bash
./examples/test-assignpublicip.sh
```

## Future Improvements

1. **Automatic Port Range Configuration**: Investigate methods to configure k3d port ranges during KECS startup
2. **Dynamic Port Management**: Explore alternatives to k3d for dynamic port allocation
3. **Port Range Extension**: Support configurable port ranges beyond 32000-32999
4. **Load Balancer Integration**: Add support for AWS Load Balancer type services