# KECS Port Forward Management

KECS provides a comprehensive port forwarding system that allows you to access services and tasks running in your KECS clusters from your local machine. This feature is analogous to AWS ECS's `assignPublicIp` functionality.

## Overview

The port-forward management system enables:
- Local access to services and tasks running in KECS clusters
- Automatic port mapping through k3d for services with NodePort
- Direct kubectl port-forwarding for tasks
- Auto-reconnection on failure
- Health monitoring of active connections
- Persistent state management across KECS restarts

## Prerequisites

- KECS instance must be running
- Service must have `assignPublicIp: ENABLED` in network configuration (for service port-forwarding)
- kubectl and k3d must be accessible from your system

## Basic Commands

### Starting Port Forwarding

#### For Services
```bash
# Forward a service with auto-assigned local port
kecs port-forward start service <cluster>/<service-name>

# Forward a service with specific local and target ports
kecs port-forward start service <cluster>/<service-name> --local-port 8080 --target-port 80
```

#### For Tasks
```bash
# Forward a task with auto-assigned local port
kecs port-forward start task <cluster>/<task-id>

# Forward a task with specific ports
kecs port-forward start task <cluster>/<task-id> --local-port 9090 --target-port 8080

# Forward using task tags (forwards to newest matching task)
kecs port-forward start task <cluster> --tags app=nginx,env=prod --local-port 3000
```

### Managing Port Forwards

#### List Active Port Forwards
```bash
kecs port-forward list
```

Output example:
```
ID                              TYPE     CLUSTER    TARGET                   LOCAL    TARGET    STATUS    CREATED
svc-default-nginx-1234567890    service  default    nginx-service           8080     80        active    2025-09-27 12:00:00
task-default-abc123             task     default    task-abc123def456       9090     8080      active    2025-09-27 12:01:00
```

#### Stop Port Forwarding
```bash
# Stop a specific forward by ID
kecs port-forward stop <forward-id>

# Stop all forwards
kecs port-forward stop --all
```


## How It Works

### Service Port Forwarding

For services with `assignPublicIp: ENABLED`:

1. KECS creates a service in Kubernetes (NodePort or LoadBalancer type)
2. The port-forward manager uses `k3d node edit` to map the NodePort to a host port
3. kubectl establishes a port-forward connection from the host port to the service
4. The connection is monitored and automatically reconnected if it fails

**Note**: Port forwarding works with both NodePort and LoadBalancer service types. Services with ELB/ALB integration (LoadBalancer type) are fully supported.

### Task Port Forwarding

For individual tasks:

1. KECS identifies the pod corresponding to the ECS task
2. kubectl establishes a direct port-forward to the pod
3. The connection is monitored and automatically reconnected if the task restarts

## Advanced Features

### Auto-Reconnection

Port forwards are automatically reconnected when:
- The connection is lost due to network issues
- The target pod/service is restarted
- The KECS instance is restarted (persistent state)

You can disable auto-reconnection:
```bash
kecs port-forward start service <cluster>/<service> --no-auto-reconnect
```

### Health Monitoring

The port-forward manager continuously monitors connection health:
- Checks connection status every 30 seconds
- Attempts reconnection after 3 failed health checks
- Logs all connection events for debugging

### Tag-Based Task Selection

When using tags to select tasks:
- The newest task matching all specified tags is selected
- If the task is replaced, the forward automatically switches to the new task
- Useful for blue-green deployments and rolling updates

Example:
```bash
# This will always forward to the newest task with these tags
kecs port-forward start task default --tags app=api,version=stable --local-port 8080
```

## Integration with ECS Services

### Enabling Public Access

To enable port forwarding for a service, set `assignPublicIp: ENABLED` in your service definition:

```json
{
  "serviceName": "my-service",
  "taskDefinition": "my-task:1",
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345678"],
      "assignPublicIp": "ENABLED"
    }
  }
}
```

### Using with ALB/ELB

Port forwarding works alongside ELB/ALB configurations:
- ALB target groups can point to NodePort services
- Health checks work through the forwarded ports
- Multiple services can be exposed through different ports

## Troubleshooting

### Common Issues

#### Port Already in Use
```
Error: port 8080 is already in use by forward svc-default-nginx-1234567890
```
Solution: Use a different local port or stop the existing forward.

#### Service Missing NodePort
```
Error: service nginx does not have NodePort configured. Ensure assignPublicIp is enabled
```
Solution: Update your service definition to include `assignPublicIp: ENABLED`.

#### Connection Refused
```
Error: connection refused on port 8080
```
Solution:
- Check if the target service/task is running
- Verify the target port is correct
- Check KECS controlplane logs: `kubectl logs -n kecs-system deployment/kecs-controlplane -f`

### Debug Mode

Enable verbose logging for troubleshooting:
```bash
KECS_LOG_LEVEL=debug kecs port-forward start service default/nginx
```

### Checking Forward Status

View detailed forward information:
```bash
kecs port-forward list --format json | jq '.[] | select(.id=="<forward-id>")'
```

## Best Practices

1. **Document Port Mappings**: Keep a record of standard port assignments for consistency across teams.

2. **Monitor Health**: Regularly check `kecs port-forward list` to ensure connections are healthy.

3. **Resource Management**: Stop unused port forwards to free up ports and system resources.

4. **Security**: Only forward ports that need local access. Keep sensitive services protected.

5. **Naming Conventions**: Use descriptive service names to make port forward management easier.

## Examples

### Development Workflow

```bash
# Start your KECS instance
kecs start

# Deploy your service with public IP enabled
aws ecs create-service --service-name web --assign-public-ip ENABLED

# Forward the service to local port 3000
kecs port-forward start service default/web --local-port 3000

# Access your service
curl http://localhost:3000

# When done, stop the forward
kecs port-forward stop --all
```

### Multi-Service Setup

```bash
# Start multiple port forwards
kecs port-forward start service default/frontend --local-port 3000 --target-port 80
kecs port-forward start service default/api --local-port 8080 --target-port 8080
kecs port-forward start service default/admin --local-port 9090 --target-port 3000

# Access services
curl http://localhost:3000  # Frontend
curl http://localhost:8080  # API
curl http://localhost:9090  # Admin panel
```

### ALB/ELB Service Setup

```bash
# Deploy service with ALB integration
aws ecs create-service \
  --service-name webapp-alb \
  --task-definition webapp:1 \
  --network-configuration "awsvpcConfiguration={assignPublicIp=ENABLED}" \
  --load-balancers targetGroupArn=arn:aws:elasticloadbalancing:us-east-1:000000000000:targetgroup/webapp-tg/xxx

# Forward the LoadBalancer service
kecs port-forward start service default/webapp-alb --local-port 8080 --target-port 80

# Access the service locally
curl http://localhost:8080
```

### Task Debugging

```bash
# Find a problematic task
aws ecs list-tasks --cluster default

# Forward to the specific task for debugging
kecs port-forward start task default/arn:aws:ecs:task:abc123 --local-port 5005 --target-port 5005

# Connect debugger to localhost:5005
```

## Architecture

The port-forward system consists of:

1. **Manager**: Central component that tracks all forwards and manages their lifecycle
2. **Forwarder**: Individual goroutine managing a single kubectl port-forward process
3. **Health Monitor**: Background process checking connection health
4. **Reconnection Monitor**: Background process handling auto-reconnection
5. **State Persistence**: JSON files storing forward configurations

For more details, see [ADR-0026: Port Forward Management](./adr/records/0026-port-forward-management.md).