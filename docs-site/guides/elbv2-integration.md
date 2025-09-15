# ELBv2 Integration Guide

KECS provides full support for AWS Elastic Load Balancing v2 (ELBv2), including Application Load Balancers (ALB) and Network Load Balancers (NLB). This guide covers how to create and manage load balancers with your ECS services.

## Overview

KECS implements ELBv2 APIs that seamlessly integrate with Kubernetes:
- **Application Load Balancers (ALB)**: HTTP/HTTPS load balancing with path-based routing
- **Network Load Balancers (NLB)**: TCP/UDP load balancing for high-performance scenarios
- **Target Groups**: Map to Kubernetes Services for traffic distribution
- **Listeners**: Configure routing rules and protocols
- **Automatic Ingress Creation**: Kubernetes Ingress resources created automatically

## Architecture

```
┌──────────────────────────────────────────┐
│           External Traffic               │
└────────────────┬─────────────────────────┘
                 │
                 ▼ Port 5373
┌──────────────────────────────────────────┐
│         Traefik Gateway                  │
│   (Global Ingress Controller)            │
└────────────────┬─────────────────────────┘
                 │
                 ▼ Host-based routing
┌──────────────────────────────────────────┐
│        Kubernetes Ingress                │
│   (Created by KECS for each ALB)         │
└────────────────┬─────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────┐
│      Kubernetes Service                  │
│     (tg-<target-group-name>)             │
└────────────────┬─────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────┐
│         ECS Task Pods                    │
└──────────────────────────────────────────┘
```

## Creating Load Balancers

### Application Load Balancer (ALB)

```bash
# Create an ALB
aws elbv2 create-load-balancer \
  --name my-alb \
  --type application \
  --subnets subnet-12345 subnet-67890

# Response includes DNS name for routing
{
  "LoadBalancers": [{
    "LoadBalancerArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188",
    "DNSName": "my-alb.kecs.local",
    "Type": "application",
    "State": {
      "Code": "active"
    }
  }]
}
```

### Network Load Balancer (NLB)

```bash
# Create an NLB
aws elbv2 create-load-balancer \
  --name my-nlb \
  --type network \
  --subnets subnet-12345 subnet-67890
```

## Target Groups

Target Groups define the backend services that receive traffic from the load balancer.

### Create Target Group

```bash
# Create target group for HTTP traffic
aws elbv2 create-target-group \
  --name my-targets \
  --protocol HTTP \
  --port 80 \
  --vpc-id vpc-12345 \
  --target-type ip \
  --health-check-path /health
```

### Register ECS Tasks

Tasks are automatically registered when you create an ECS service with load balancer configuration:

```bash
# Create ECS service with ALB
aws ecs create-service \
  --cluster my-cluster \
  --service-name my-service \
  --task-definition my-app:1 \
  --desired-count 3 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345]}" \
  --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/50dc6c495c0c9188,containerName=my-app,containerPort=80"
```

## Listeners and Routing

### Create Listener

```bash
# Create HTTP listener
aws elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188 \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/50dc6c495c0c9188
```

### Path-Based Routing

```bash
# Add rule for path-based routing
aws elbv2 create-rule \
  --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-alb/50dc6c495c0c9188/50dc6c495c0c9188 \
  --priority 10 \
  --conditions Field=path-pattern,Values="/api/*" \
  --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-targets/50dc6c495c0c9188
```

## Host-Based Routing

KECS uses host headers to route traffic to different ALBs:

```bash
# Access ALB using its DNS name
curl -H "Host: my-alb.kecs.local" http://localhost:5373

# Or configure /etc/hosts
echo "127.0.0.1 my-alb.kecs.local" | sudo tee -a /etc/hosts
curl http://my-alb.kecs.local:5373
```

## HTTPS/TLS Configuration

### Create HTTPS Listener

```bash
# Create certificate (in real AWS, use ACM)
aws elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188 \
  --protocol HTTPS \
  --port 443 \
  --certificates CertificateArn=arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 \
  --default-actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/50dc6c495c0c9188
```

## Health Checks

### Configure Health Check

```bash
# Modify health check settings
aws elbv2 modify-target-group \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/50dc6c495c0c9188 \
  --health-check-protocol HTTP \
  --health-check-path /health \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3
```

### Check Target Health

```bash
# Describe target health
aws elbv2 describe-target-health \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/50dc6c495c0c9188
```

## Advanced Features

### Sticky Sessions

```bash
# Enable sticky sessions
aws elbv2 modify-target-group-attributes \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/50dc6c495c0c9188 \
  --attributes Key=stickiness.enabled,Value=true Key=stickiness.type,Value=lb_cookie
```

### Connection Draining

```bash
# Configure deregistration delay
aws elbv2 modify-target-group-attributes \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/50dc6c495c0c9188 \
  --attributes Key=deregistration_delay.timeout_seconds,Value=30
```

### Cross-Zone Load Balancing

```bash
# Enable cross-zone load balancing
aws elbv2 modify-load-balancer-attributes \
  --load-balancer-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188 \
  --attributes Key=load_balancing.cross_zone.enabled,Value=true
```

## Kubernetes Integration Details

### Automatic Resources

When you create ELBv2 resources, KECS automatically creates corresponding Kubernetes resources:

1. **Target Group** → Kubernetes Service
   - Name: `tg-<target-group-name>`
   - Namespace: `kecs-services`
   - Selector: Matches ECS task pods

2. **ALB + Listener** → Kubernetes Ingress
   - Automatically created when listener is added
   - Host-based routing using ALB DNS name
   - Managed by Traefik ingress controller

### Direct Kubernetes Access

```bash
# View created services
kubectl get services -n kecs-services

# View ingress resources
kubectl get ingress -n kecs-services

# Describe target group service
kubectl describe service tg-my-targets -n kecs-services
```

## Monitoring and Logs

### View Load Balancer Metrics

```bash
# Describe load balancer
aws elbv2 describe-load-balancers \
  --load-balancer-arns arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188

# Get target group attributes
aws elbv2 describe-target-group-attributes \
  --target-group-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-targets/50dc6c495c0c9188
```

### Access Logs

```bash
# View KECS logs for ELBv2 operations
kecs logs --component elbv2 -f

# View Traefik access logs
kubectl logs -n kecs-system deployment/traefik -f
```

## Complete Example

Here's a complete example of deploying an application with ALB:

```bash
# 1. Create ALB
ALB_ARN=$(aws elbv2 create-load-balancer \
  --name demo-alb \
  --type application \
  --subnets subnet-12345 subnet-67890 \
  --query 'LoadBalancers[0].LoadBalancerArn' \
  --output text)

# 2. Create target group
TG_ARN=$(aws elbv2 create-target-group \
  --name demo-targets \
  --protocol HTTP \
  --port 80 \
  --vpc-id vpc-12345 \
  --target-type ip \
  --query 'TargetGroups[0].TargetGroupArn' \
  --output text)

# 3. Create listener
aws elbv2 create-listener \
  --load-balancer-arn $ALB_ARN \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=forward,TargetGroupArn=$TG_ARN

# 4. Create task definition
cat > task-def.json << EOF
{
  "family": "demo-app",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "nginx:alpine",
      "portMappings": [
        {
          "containerPort": 80,
          "protocol": "tcp"
        }
      ],
      "essential": true
    }
  ]
}
EOF

aws ecs register-task-definition --cli-input-json file://task-def.json

# 5. Create ECS service with ALB
aws ecs create-service \
  --cluster default \
  --service-name demo-service \
  --task-definition demo-app:1 \
  --desired-count 3 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345]}" \
  --load-balancers "targetGroupArn=$TG_ARN,containerName=app,containerPort=80"

# 6. Test the application
curl -H "Host: demo-alb.kecs.local" http://localhost:5373
```

## Troubleshooting

### Load Balancer Not Accessible

```bash
# Check if Ingress was created
kubectl get ingress -n kecs-services

# Check Traefik logs
kubectl logs -n kecs-system deployment/traefik | grep error

# Verify DNS resolution
nslookup my-alb.kecs.local
```

### Targets Unhealthy

```bash
# Check target health
aws elbv2 describe-target-health --target-group-arn $TG_ARN

# Check pod status
kubectl get pods -n kecs-services

# View pod logs
kubectl logs -n kecs-services <pod-name>
```

### Routing Issues

```bash
# Test with explicit host header
curl -v -H "Host: my-alb.kecs.local" http://localhost:5373

# Check Ingress configuration
kubectl describe ingress -n kecs-services

# Verify Service endpoints
kubectl get endpoints -n kecs-services
```

## Best Practices

1. **Use meaningful names**: Name your ALBs and target groups descriptively
2. **Configure health checks**: Always set appropriate health check paths
3. **Monitor logs**: Use `kecs logs` to troubleshoot issues
4. **Test locally**: Use host headers or /etc/hosts for local testing
5. **Clean up resources**: Delete unused load balancers to free resources

## Limitations

Current limitations in KECS ELBv2 implementation:

- WAF integration not supported
- Some advanced ALB features may be limited
- Certificate management simplified (no ACM integration)
- CloudWatch metrics not available

## Next Steps

- [Services Guide](/guides/services) - Create ECS services with load balancers
- [Networking Guide](/guides/networking) - Understand KECS networking
- [API Reference](/api/elbv2) - Complete ELBv2 API documentation