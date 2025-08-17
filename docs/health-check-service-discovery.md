# Health Check and Service Discovery Integration

## Overview

KECS implements ECS-compatible behavior where container health checks automatically affect Service Discovery, ensuring that only healthy instances receive traffic through DNS resolution.

## Key Features

### 1. Container Health Checks
- **Task Definition**: Health checks are defined in the task definition's container definitions
- **Kubernetes Integration**: Health checks are converted to Kubernetes liveness/readiness probes
- **Status Tracking**: Task health status is tracked based on container health

### 2. Health Status Propagation
- **Automatic Updates**: When a container's health status changes, it's automatically propagated to Service Discovery
- **Real-time Sync**: Health status changes trigger immediate updates to DNS records
- **Status Mapping**: ECS health statuses (HEALTHY, UNHEALTHY, UNKNOWN) are mapped to Service Discovery

### 3. DNS Response Filtering
- **Healthy Only**: Only healthy instances are included in DNS A/AAAA record responses
- **Immediate Effect**: Unhealthy instances are immediately excluded from DNS resolution
- **Kubernetes Endpoints**: Kubernetes Endpoints are updated to reflect health status

## Implementation Details

### Task Manager Updates

The TaskManager now includes health status synchronization with Service Discovery:

```go
// UpdateTaskStatus in task_manager.go
func (tm *TaskManager) UpdateTaskStatus(ctx context.Context, taskARN string, pod *corev1.Pod) error {
    // ... existing status update logic ...
    
    // Update health status
    previousHealthStatus := task.HealthStatus
    task.HealthStatus = tm.getHealthStatus(pod)
    
    // Update health status in Service Discovery if changed
    if task.LastStatus == "RUNNING" && previousHealthStatus != task.HealthStatus {
        go tm.updateServiceDiscoveryHealth(context.Background(), task)
    }
}
```

### Service Discovery Manager Updates

The Service Discovery manager filters instances based on health status:

```go
// DiscoverInstances in manager.go
func (m *manager) DiscoverInstances(ctx context.Context, namespaceName, serviceName string) ([]*Instance, error) {
    // Get only healthy instances for DNS resolution
    var instances []*Instance
    for _, instance := range m.instances[serviceID] {
        // Only include healthy instances in DNS responses
        if instance.HealthStatus == "HEALTHY" || instance.HealthStatus == "" {
            instances = append(instances, instance)
        }
    }
    return instances, nil
}
```

### Kubernetes Endpoints Sync

Kubernetes Endpoints are updated to categorize instances by health:

```go
// buildEndpointSubsets in kubernetes.go
func (m *manager) buildEndpointSubsets(instances map[string]*Instance, service *Service) []corev1.EndpointSubset {
    addresses := []corev1.EndpointAddress{}
    notReadyAddresses := []corev1.EndpointAddress{}
    
    for _, instance := range instances {
        // Categorize by health status
        if instance.HealthStatus == "HEALTHY" || instance.HealthStatus == "" {
            addresses = append(addresses, endpointAddress)
        } else {
            notReadyAddresses = append(notReadyAddresses, endpointAddress)
        }
    }
}
```

## Health Check Configuration

### Task Definition Example

```json
{
  "family": "my-service",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "myapp:latest",
      "healthCheck": {
        "command": ["CMD-SHELL", "wget -q --spider http://localhost:8080/health || exit 1"],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ]
}
```

### Health Check Parameters
- **command**: Command to run to check health
- **interval**: Seconds between health checks (default: 30)
- **timeout**: Seconds to wait for health check (default: 5)
- **retries**: Consecutive failures before marking unhealthy (default: 3)
- **startPeriod**: Grace period before health checks start (default: 0)

## Service Discovery Behavior

### Healthy Instances
- Included in DNS A/AAAA records
- Listed in Kubernetes Endpoints as ready addresses
- Receive traffic from load balancers and service mesh

### Unhealthy Instances
- Excluded from DNS A/AAAA records
- Listed in Kubernetes Endpoints as not-ready addresses
- Do not receive new traffic
- Existing connections may persist until terminated

### Unknown/Initial State
- Treated as healthy by default (to allow initial startup)
- Included in DNS responses until first health check completes
- Transitions to HEALTHY or UNHEALTHY based on health check results

## Testing

Use the provided test script to verify the integration:

```bash
./test-health-service-discovery.sh
```

The script will:
1. Create a Service Discovery namespace and service
2. Deploy an ECS service with health checks
3. Verify instances are registered with correct health status
4. Demonstrate DNS resolution filtering based on health

## Benefits

1. **Automatic Failover**: Unhealthy instances are automatically removed from rotation
2. **Zero Downtime Deployments**: New instances only receive traffic when healthy
3. **Service Reliability**: Prevents routing traffic to failing containers
4. **ECS Compatibility**: Matches AWS ECS Service Discovery behavior

## Architecture

```
┌─────────────────┐
│  ECS Task       │
│  Health Check   │
└────────┬────────┘
         │
         v
┌─────────────────┐
│  Task Manager   │
│  Status Update  │
└────────┬────────┘
         │
         v
┌─────────────────┐
│Service Discovery│
│  Health Update  │
└────────┬────────┘
         │
         v
┌─────────────────┐
│   Kubernetes    │
│   Endpoints     │
└────────┬────────┘
         │
         v
┌─────────────────┐
│    CoreDNS      │
│  DNS Resolution │
└─────────────────┘
```

## Limitations

- Health check commands must be compatible with the container's environment
- Initial health status is assumed healthy until first check completes
- Health check intervals affect how quickly unhealthy instances are removed

## Future Enhancements

- Support for custom health check grace periods per service
- Health check metrics and monitoring
- Configurable health status thresholds
- Integration with external health check systems