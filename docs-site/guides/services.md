# Working with Services

Services in ECS are long-running applications that maintain a desired number of tasks. This guide covers creating, managing, and monitoring services in KECS.

## Service Concepts

### What is a Service?

A service allows you to run and maintain a specified number of instances of a task definition simultaneously in an ECS cluster. If any of your tasks fail or stop, the service scheduler launches another instance to replace it.

### Key Features

- **Desired Count**: Maintains the specified number of running tasks
- **Load Balancing**: Distributes traffic across tasks
- **Service Discovery**: Enables service-to-service communication
- **Rolling Updates**: Updates tasks with zero downtime
- **Auto Scaling**: Scales based on metrics

## Creating Services

### Basic Service Creation

```bash
# Create a simple service
aws ecs create-service \
  --cluster production \
  --service-name web-app \
  --task-definition webapp:1 \
  --desired-count 3 \
  --launch-type FARGATE \
  --endpoint-url http://localhost:8080
```

### Service with Load Balancer

```json
{
  "cluster": "production",
  "serviceName": "web-app",
  "taskDefinition": "webapp:1",
  "desiredCount": 3,
  "launchType": "FARGATE",
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:region:account-id:targetgroup/my-targets/1234567890123456",
      "containerName": "web",
      "containerPort": 80
    }
  ],
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345", "subnet-67890"],
      "securityGroups": ["sg-12345"],
      "assignPublicIp": "ENABLED"
    }
  }
}
```

### Service with Service Discovery

```json
{
  "cluster": "production",
  "serviceName": "api-service",
  "taskDefinition": "api:1",
  "desiredCount": 2,
  "launchType": "FARGATE",
  "serviceRegistries": [
    {
      "registryArn": "arn:aws:servicediscovery:region:account-id:service/srv-1234567890",
      "containerName": "api",
      "containerPort": 8080
    }
  ],
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345"],
      "securityGroups": ["sg-12345"]
    }
  }
}
```

## Service Configuration

### Deployment Configuration

Control how services are updated:

```json
{
  "deploymentConfiguration": {
    "maximumPercent": 200,
    "minimumHealthyPercent": 100,
    "deploymentCircuitBreaker": {
      "enable": true,
      "rollback": true
    }
  }
}
```

- **maximumPercent**: Maximum number of tasks during deployment (% of desired count)
- **minimumHealthyPercent**: Minimum number of healthy tasks during deployment
- **deploymentCircuitBreaker**: Automatically roll back failed deployments

### Placement Strategies

Distribute tasks across your cluster:

```json
{
  "placementStrategy": [
    {
      "type": "spread",
      "field": "attribute:ecs.availability-zone"
    },
    {
      "type": "binpack",
      "field": "memory"
    }
  ]
}
```

Strategy types:
- **spread**: Distribute evenly based on field
- **binpack**: Pack tasks based on resource utilization
- **random**: Place tasks randomly

### Placement Constraints

Control where tasks can run:

```json
{
  "placementConstraints": [
    {
      "type": "memberOf",
      "expression": "attribute:ecs.instance-type =~ t3.*"
    }
  ]
}
```

## Managing Services

### Updating a Service

Update service configuration or task definition:

```bash
# Update to new task definition
aws ecs update-service \
  --cluster production \
  --service web-app \
  --task-definition webapp:2 \
  --endpoint-url http://localhost:8080

# Update desired count
aws ecs update-service \
  --cluster production \
  --service web-app \
  --desired-count 5 \
  --endpoint-url http://localhost:8080
```

### Scaling Services

#### Manual Scaling

```bash
aws ecs update-service \
  --cluster production \
  --service web-app \
  --desired-count 10 \
  --endpoint-url http://localhost:8080
```

#### Auto Scaling

Set up auto scaling policies:

```bash
# Register scalable target
aws application-autoscaling register-scalable-target \
  --service-namespace ecs \
  --scalable-dimension ecs:service:DesiredCount \
  --resource-id service/production/web-app \
  --min-capacity 2 \
  --max-capacity 10

# Create scaling policy
aws application-autoscaling put-scaling-policy \
  --policy-name cpu-scaling \
  --service-namespace ecs \
  --scalable-dimension ecs:service:DesiredCount \
  --resource-id service/production/web-app \
  --policy-type TargetTrackingScaling \
  --target-tracking-scaling-policy-configuration file://scaling-policy.json
```

### Service Health Checks

Services use health checks to determine task health:

```json
{
  "healthCheckGracePeriodSeconds": 60,
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:...",
      "containerName": "web",
      "containerPort": 80
    }
  ]
}
```

## Monitoring Services

### Service Metrics

View service status:

```bash
# Describe service
aws ecs describe-services \
  --cluster production \
  --services web-app \
  --endpoint-url http://localhost:8080
```

Key metrics to monitor:
- **runningCount**: Number of running tasks
- **pendingCount**: Number of pending tasks
- **desiredCount**: Desired number of tasks
- **deployments**: Active deployments

### Service Events

View service events:

```bash
aws ecs describe-services \
  --cluster production \
  --services web-app \
  --endpoint-url http://localhost:8080 \
  | jq '.services[0].events[:5]'
```

### Task Status

Check individual task status:

```bash
# List tasks for a service
aws ecs list-tasks \
  --cluster production \
  --service-name web-app \
  --endpoint-url http://localhost:8080

# Describe tasks
aws ecs describe-tasks \
  --cluster production \
  --tasks <task-arn> \
  --endpoint-url http://localhost:8080
```

## Service Patterns

### Blue/Green Deployments

1. Create new task definition
2. Create new service with new version
3. Test new service
4. Switch traffic to new service
5. Delete old service

### Canary Deployments

Use weighted target groups:

```json
{
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:...:targetgroup/blue/...",
      "containerName": "app",
      "containerPort": 80
    },
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:...:targetgroup/green/...",
      "containerName": "app",
      "containerPort": 80
    }
  ]
}
```

### Sidecar Pattern

Deploy multiple containers in a task:

```json
{
  "containerDefinitions": [
    {
      "name": "app",
      "image": "myapp:latest",
      "portMappings": [{"containerPort": 8080}]
    },
    {
      "name": "envoy",
      "image": "envoyproxy/envoy:latest",
      "portMappings": [{"containerPort": 9901}]
    }
  ]
}
```

## Service Discovery

### Private DNS Namespace

Create a namespace for service discovery:

```bash
aws servicediscovery create-private-dns-namespace \
  --name local \
  --vpc vpc-12345 \
  --endpoint-url http://localhost:8080
```

### Register Service

```json
{
  "serviceRegistries": [
    {
      "registryArn": "arn:aws:servicediscovery:...",
      "containerName": "app",
      "containerPort": 8080
    }
  ]
}
```

### Discover Services

Services can discover each other using DNS:
```
http://service-name.namespace.local:8080
```

## Best Practices

### 1. Resource Allocation

- Set appropriate CPU and memory limits
- Use resource reservations for critical services
- Monitor resource utilization

### 2. Health Checks

- Configure appropriate health check intervals
- Set reasonable grace periods
- Use container health checks

### 3. Deployment Strategy

- Use rolling updates for zero-downtime deployments
- Enable circuit breaker for automatic rollback
- Test deployments in staging first

### 4. Monitoring

- Set up CloudWatch alarms
- Monitor service events
- Track deployment success rates

### 5. Security

- Use IAM roles for tasks
- Restrict security groups
- Enable encryption for sensitive data

## Troubleshooting

### Service Won't Start

1. Check task definition is valid
2. Verify cluster has available resources
3. Check security groups and network configuration
4. Review service events for errors

### Tasks Keep Failing

1. Check container logs
2. Verify image is accessible
3. Check resource constraints
4. Review task stop reasons

### Slow Deployments

1. Adjust deployment configuration
2. Check health check settings
3. Monitor resource availability
4. Review placement constraints

For more detailed troubleshooting, see our [Troubleshooting Guide](/guides/troubleshooting).