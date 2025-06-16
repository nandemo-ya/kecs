# ELBv2 Integration in KECS

KECS supports Amazon Elastic Load Balancing v2 (Application Load Balancers) through LocalStack integration. This enables you to create and manage load balancers for your ECS services in a local environment.

## Overview

The ELBv2 integration provides:

- **Application Load Balancer Management**: Create and manage ALBs
- **Target Group Management**: Configure target groups for container instances
- **Health Checks**: Monitor the health of your containers
- **Listener Configuration**: Set up routing rules for incoming traffic
- **LocalStack Integration**: Full compatibility with LocalStack's ELBv2 service

## Architecture

```
┌─────────────────────┐     ┌──────────────────┐
│   ECS Service       │────▶│  Target Group    │
│                     │     │                  │
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
                            ┌──────────────────┐
                            │   Load Balancer  │
                            │                  │
                            │  ┌────────────┐  │
                            │  │  Listener  │  │
                            │  │  Port: 80  │  │
                            │  └────────────┘  │
                            └──────────────────┘
```

## Configuration

### Enabling ELBv2 Integration

To enable ELBv2 integration, ensure LocalStack is running with the ELBv2 service:

```yaml
# config/localstack.yaml
localstack:
  enabled: true
  services:
    - elbv2
    - ec2  # Required for VPC and subnet operations
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
// Create a load balancer
lb, err := elbv2Integration.CreateLoadBalancer(ctx, "my-alb", subnets, securityGroups)

// Delete a load balancer
err := elbv2Integration.DeleteLoadBalancer(ctx, lbArn)

// Get load balancer details
lb, err := elbv2Integration.GetLoadBalancer(ctx, lbArn)
```

### Target Group Operations

```go
// Create a target group
tg, err := elbv2Integration.CreateTargetGroup(ctx, "my-tg", 80, "HTTP", vpcId)

// Register targets
targets := []elbv2.Target{
    {Id: "10.0.1.10", Port: 80},
    {Id: "10.0.1.11", Port: 80},
}
err := elbv2Integration.RegisterTargets(ctx, tgArn, targets)

// Get target health
health, err := elbv2Integration.GetTargetHealth(ctx, tgArn)
```

### Listener Operations

```go
// Create a listener
listener, err := elbv2Integration.CreateListener(ctx, lbArn, 80, "HTTP", tgArn)

// Delete a listener
err := elbv2Integration.DeleteListener(ctx, listenerArn)
```

## Integration with ECS Services

### Automatic Target Registration

When you create an ECS service with load balancer configuration, KECS automatically:

1. Validates the target group exists
2. Registers container instances as targets
3. Updates targets when containers are added/removed
4. Handles health check configurations

### Service Discovery

Load balancers provide service discovery through DNS names:

```bash
# Access your service through the load balancer
curl http://my-alb-1234567890.us-east-1.elb.amazonaws.com
```

## Testing with LocalStack

### 1. Start LocalStack with ELBv2

```bash
localstack start -d
```

### 2. Create a Load Balancer

```bash
aws --endpoint-url=http://localhost:4566 elbv2 create-load-balancer \
  --name my-alb \
  --subnets subnet-12345 subnet-67890 \
  --region us-east-1
```

### 3. Create a Target Group

```bash
aws --endpoint-url=http://localhost:4566 elbv2 create-target-group \
  --name my-tg \
  --protocol HTTP \
  --port 80 \
  --vpc-id vpc-12345 \
  --region us-east-1
```

### 4. Create a Listener

```bash
aws --endpoint-url=http://localhost:4566 elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188 \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-tg/50dc6c495c0c9188 \
  --region us-east-1
```

### 5. Create an ECS Service with Load Balancer

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

Target groups automatically perform health checks on registered targets:

- **Health Check Path**: `/` (default)
- **Health Check Interval**: 30 seconds
- **Timeout**: 5 seconds
- **Healthy Threshold**: 2 consecutive successful checks
- **Unhealthy Threshold**: 3 consecutive failed checks

You can monitor health status:

```bash
aws --endpoint-url=http://localhost:4566 elbv2 describe-target-health \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-tg/50dc6c495c0c9188 \
  --region us-east-1
```

## Limitations

Current limitations in the KECS ELBv2 integration:

1. **LocalStack Only**: Currently only works with LocalStack, not real AWS ELBv2
2. **ALB Only**: Only Application Load Balancers are supported (no NLB/GLB)
3. **Basic Features**: Advanced features like WAF, authentication, etc. are not yet implemented
4. **Static Registration**: Dynamic target registration based on container lifecycle is simplified

## Future Enhancements

Planned improvements:

1. Network Load Balancer (NLB) support
2. Dynamic target registration/deregistration
3. Advanced routing rules (path-based, host-based)
4. SSL/TLS termination support
5. Integration with AWS Certificate Manager
6. WebSocket and HTTP/2 support
7. Cross-zone load balancing configuration