# ELBv2 Integration in KECS

KECS provides a virtual implementation of Amazon Elastic Load Balancing v2 (Application Load Balancers) that works without requiring LocalStack Pro. This is achieved by mapping ECS load balancer concepts to Kubernetes Services.

## Overview

The ELBv2 integration provides:

- **Virtual Load Balancer Management**: Create and manage virtual ALBs
- **Target Group Management**: Configure target groups for container instances
- **Health Checks**: Monitor the health of your containers
- **Listener Configuration**: Set up routing rules for incoming traffic
- **No LocalStack Pro Required**: Uses Kubernetes-native features

## Architecture

```
┌─────────────────────┐     ┌──────────────────┐
│   ECS Service       │────▶│  Target Group    │
│                     │     │  (Virtual)       │
│  ┌──────────────┐  │     │  ┌────────────┐  │
│  │ Container 1  │──┼─────┼─▶│  Target 1  │  │
│  └──────────────┘  │     │  └────────────┘  │
│                     │     │                  │
│  ┌──────────────┐  │     │  ┌────────────┐  │
│  │ Container 2  │──┼─────┼─▶│  Target 2  │  │
│  └──────────────┘  │     │  └────────────┘  │
└─────────────────────┘     └──────────────────┘
                                      │
                                      ▼
                        ┌─────────────────────────┐
                        │   Virtual Load Balancer │
                        │   (K8s Service)         │
                        │  ┌──────────────────┐  │
                        │  │    Listener      │  │
                        │  │    Port: 80      │  │
                        │  └──────────────────┘  │
                        └─────────────────────────┘
```

## Implementation Details

Instead of using actual AWS ELBv2 APIs (which require LocalStack Pro), KECS provides:

1. **Virtual Load Balancers**: In-memory representation of ALBs
2. **Virtual Target Groups**: Tracking of container targets
3. **Simulated Health Checks**: State transitions for target health
4. **Kubernetes Service Mapping**: Maps to actual K8s Services for traffic routing

## Configuration

### Enabling ELBv2 Integration

The ELBv2 integration is enabled by default and doesn't require LocalStack:

```yaml
# config/kecs.yaml
elbv2:
  enabled: true
```

### Service Configuration

When creating an ECS service, you can configure load balancing:

```json
{
  "serviceName": "my-web-app",
  "taskDefinition": "my-web-app:1",
  "desiredCount": 3,
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-tg/50dc6c495c0c9188",
      "containerName": "web",
      "containerPort": 80
    }
  ]
}
```

## API Operations

### Load Balancer Operations

```go
// Create a virtual load balancer
lb, err := elbv2Integration.CreateLoadBalancer(ctx, "my-alb", subnets, securityGroups)

// Delete a virtual load balancer
err := elbv2Integration.DeleteLoadBalancer(ctx, lbArn)

// Get virtual load balancer details
lb, err := elbv2Integration.GetLoadBalancer(ctx, lbArn)
```

### Target Group Operations

```go
// Create a virtual target group
tg, err := elbv2Integration.CreateTargetGroup(ctx, "my-tg", 80, "HTTP", vpcId)

// Register targets (container IPs)
targets := []elbv2.Target{
    {Id: "10.0.1.10", Port: 80},
    {Id: "10.0.1.11", Port: 80},
}
err := elbv2Integration.RegisterTargets(ctx, tgArn, targets)

// Get target health status
health, err := elbv2Integration.GetTargetHealth(ctx, tgArn)
```

### Listener Operations

```go
// Create a virtual listener
listener, err := elbv2Integration.CreateListener(ctx, lbArn, 80, "HTTP", tgArn)

// Delete a virtual listener
err := elbv2Integration.DeleteListener(ctx, listenerArn)
```

## Integration with ECS Services

### Automatic Target Registration

When you create an ECS service with load balancer configuration, KECS automatically:

1. Validates the virtual target group exists
2. Registers container instances as targets
3. Updates targets when containers are added/removed
4. Simulates health check transitions

### Service Discovery

Virtual load balancers provide DNS names that follow AWS naming conventions:

```bash
# Example DNS name
my-alb-1234567890.us-east-1.elb.amazonaws.com
```

Note: These DNS names are virtual and won't resolve. Use Kubernetes Service endpoints for actual traffic routing.

## Testing

### 1. Create a Virtual Load Balancer

```bash
# Using KECS API
curl -X POST "http://localhost:8080/v1/CreateLoadBalancer" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-alb",
    "subnets": ["subnet-12345", "subnet-67890"],
    "securityGroups": ["sg-12345"]
  }'
```

### 2. Create a Virtual Target Group

```bash
curl -X POST "http://localhost:8080/v1/CreateTargetGroup" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-tg",
    "protocol": "HTTP",
    "port": 80,
    "vpcId": "vpc-12345"
  }'
```

### 3. Create an ECS Service with Load Balancer

```bash
curl -X POST "http://localhost:8080/v1/CreateService" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateService" \
  -d '{
    "serviceName": "web-app",
    "taskDefinition": "my-web-app:1",
    "desiredCount": 2,
    "loadBalancers": [
      {
        "targetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-tg/50dc6c495c0c9188",
        "containerName": "web",
        "containerPort": 80
      }
    ]
  }'
```

## Health Checks

Virtual target groups simulate health checks:

- **Initial State**: Targets start in "initial" state
- **Health Transition**: After ~5 seconds, targets transition to "healthy"
- **Health Check Settings**: 
  - Path: `/` (default)
  - Interval: 30 seconds (simulated)
  - Healthy Threshold: 2 checks
  - Unhealthy Threshold: 3 checks

## Kubernetes Service Mapping

Behind the scenes, KECS maps ECS load balancers to Kubernetes Services:

1. **ECS Service with Load Balancer** → **K8s Deployment + Service**
2. **Target Group** → **Service Endpoints**
3. **Health Checks** → **Readiness Probes**

Example Kubernetes Service created:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ecs-service-web-app
  labels:
    kecs.dev/service: web-app
spec:
  selector:
    kecs.dev/service: web-app
  ports:
    - port: 80
      targetPort: 80
  type: ClusterIP  # Or LoadBalancer for external access
```

## Advantages

1. **No LocalStack Pro Required**: Works with free LocalStack or without LocalStack
2. **Simplified Implementation**: Uses Kubernetes-native features
3. **Compatible API**: Maintains ECS API compatibility
4. **Local Development**: Perfect for local testing and development

## Limitations

1. **Virtual Only**: Load balancers don't actually route traffic (use K8s Services)
2. **No Advanced Features**: No WAF, SSL termination, or advanced routing
3. **In-Memory State**: Virtual resources are not persisted
4. **DNS Names**: Virtual DNS names don't resolve

## Future Enhancements

1. Persistence layer for virtual resources
2. Integration with actual Kubernetes Ingress controllers
3. Support for Network Load Balancers (NLB)
4. Advanced routing rules mapping to Ingress rules
5. SSL/TLS termination through cert-manager