# Microservice with ELB Example

This example demonstrates deploying a microservice API with Application Load Balancer (ALB) integration, including path-based routing and health checks.

## Overview

- **Purpose**: Show ECS service integration with ELB v2 (ALB)
- **Components**: 
  - Node.js API service with multiple endpoints
  - Application Load Balancer with path-based routing
  - Target group with health checks
- **Features**:
  - Load balancing across multiple tasks
  - Path-based routing rules
  - Health check configuration
  - Auto-scaling ready

## Architecture

```
                           Internet
                              │
                              ▼
                    ┌─────────────────┐
                    │      ALB        │
                    │ (Port 80/443)   │
                    └────────┬────────┘
                             │
                ┌────────────┴────────────┐
                │    Path-based Rules     │
                ├─────────────────────────┤
                │ /api/users → Target Grp │
                │ /api/products → Target  │
                │ /health → Target Group  │
                └────────────┬────────────┘
                             │
                    ┌────────▼────────┐
                    │  Target Group   │
                    │ (Port 3000)     │
                    └────────┬────────┘
                             │
           ┌─────────────────┼─────────────────┐
           ▼                 ▼                 ▼
     ┌───────────┐    ┌───────────┐    ┌───────────┐
     │  Task 1   │    │  Task 2   │    │  Task 3   │
     │ API:3000  │    │ API:3000  │    │ API:3000  │
     └───────────┘    └───────────┘    └───────────┘
```

## Prerequisites

1. KECS running locally
2. AWS CLI configured to point to KECS endpoint
3. ecspresso installed
4. LocalStack (optional, for full ALB functionality)

## Setup Instructions

### 1. Start KECS and LocalStack

```bash
# Start KECS
kecs start

# Optional: Start LocalStack for ALB support
docker run -d \
  --name localstack \
  -p 4566:4566 \
  -e SERVICES=elbv2,iam,logs \
  -e DEBUG=1 \
  localstack/localstack
```

### 2. Create the ECS Cluster

```bash
aws ecs create-cluster --cluster-name default \
  --endpoint-url http://localhost:8080
```

### 3. Create CloudWatch Log Group

```bash
aws logs create-log-group \
  --log-group-name /ecs/microservice-api \
  --endpoint-url http://localhost:8080
```

### 4. Create IAM Roles

```bash
# Task Execution Role
aws iam create-role \
  --role-name ecsTaskExecutionRole \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {"Service": "ecs-tasks.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }]
  }' \
  --endpoint-url http://localhost:8080

# Task Role
aws iam create-role \
  --role-name ecsTaskRole \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {"Service": "ecs-tasks.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }]
  }' \
  --endpoint-url http://localhost:8080
```

### 5. Create Load Balancer Resources

```bash
# Create VPC (if not exists)
VPC_ID=$(aws ec2 describe-vpcs \
  --endpoint-url http://localhost:8080 \
  --query 'Vpcs[0].VpcId' --output text)

# Create Application Load Balancer
ALB_ARN=$(aws elbv2 create-load-balancer \
  --name microservice-alb \
  --subnets subnet-12345678 subnet-87654321 \
  --security-groups sg-alb-public \
  --scheme internet-facing \
  --type application \
  --ip-address-type ipv4 \
  --endpoint-url http://localhost:8080 \
  --query 'LoadBalancers[0].LoadBalancerArn' --output text)

echo "ALB ARN: $ALB_ARN"

# Create Target Group
TG_ARN=$(aws elbv2 create-target-group \
  --name microservice-api-tg \
  --protocol HTTP \
  --port 3000 \
  --vpc-id $VPC_ID \
  --target-type ip \
  --health-check-enabled \
  --health-check-path /health \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3 \
  --matcher HttpCode=200 \
  --endpoint-url http://localhost:8080 \
  --query 'TargetGroups[0].TargetGroupArn' --output text)

echo "Target Group ARN: $TG_ARN"

# Create HTTP Listener
LISTENER_ARN=$(aws elbv2 create-listener \
  --load-balancer-arn $ALB_ARN \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=forward,TargetGroupArn=$TG_ARN \
  --endpoint-url http://localhost:8080 \
  --query 'Listeners[0].ListenerArn' --output text)

echo "Listener ARN: $LISTENER_ARN"

# Create path-based routing rules
# Rule for /api/users
aws elbv2 create-rule \
  --listener-arn $LISTENER_ARN \
  --priority 1 \
  --conditions Field=path-pattern,Values="/api/users*" \
  --actions Type=forward,TargetGroupArn=$TG_ARN \
  --endpoint-url http://localhost:8080

# Rule for /api/products
aws elbv2 create-rule \
  --listener-arn $LISTENER_ARN \
  --priority 2 \
  --conditions Field=path-pattern,Values="/api/products*" \
  --actions Type=forward,TargetGroupArn=$TG_ARN \
  --endpoint-url http://localhost:8080

# Rule for /api/*
aws elbv2 create-rule \
  --listener-arn $LISTENER_ARN \
  --priority 10 \
  --conditions Field=path-pattern,Values="/api/*" \
  --actions Type=forward,TargetGroupArn=$TG_ARN \
  --endpoint-url http://localhost:8080
```

### 6. Update service_def.json with actual Target Group ARN

```bash
# Replace the placeholder ARN in service_def.json
sed -i.bak "s|arn:aws:elasticloadbalancing:us-east-1:000000000000:targetgroup/microservice-api-tg/1234567890123456|$TG_ARN|" service_def.json
```

## Deployment

### Using ecspresso

```bash
# Deploy the service
ecspresso deploy --config ecspresso.yml

# Check deployment status
ecspresso status --config ecspresso.yml

# Scale the service
ecspresso scale --config ecspresso.yml --tasks 5
```

### Using AWS CLI

```bash
# Register task definition
aws ecs register-task-definition \
  --cli-input-json file://task_def.json \
  --endpoint-url http://localhost:8080

# Create service
aws ecs create-service \
  --cli-input-json file://service_def.json \
  --endpoint-url http://localhost:8080
```

## Verification

### 1. Check ALB and Target Health

```bash
# Get ALB DNS name
ALB_DNS=$(aws elbv2 describe-load-balancers \
  --names microservice-alb \
  --endpoint-url http://localhost:8080 \
  --query 'LoadBalancers[0].DNSName' --output text)

echo "ALB DNS: $ALB_DNS"

# Check target health
aws elbv2 describe-target-health \
  --target-group-arn $TG_ARN \
  --endpoint-url http://localhost:8080
```

### 2. Test API Endpoints through ALB

Since KECS runs in Kubernetes, we'll use port-forwarding to test:

```bash
# First, get the Traefik service that acts as the ALB
kubectl get svc -n kecs-system

# Port forward to access the "ALB" (Traefik)
kubectl port-forward -n kecs-system svc/traefik 8888:80 &
PF_PID=$!

# Test endpoints through the load balancer
# Health check
curl -H "Host: microservice-alb" http://localhost:8888/health
# Expected: {"status":"healthy","service":"microservice-api"}

# Users endpoint
curl -H "Host: microservice-alb" http://localhost:8888/api/users
# Expected: {"users":[{"id":1,"name":"John"},{"id":2,"name":"Jane"}]}

# Products endpoint
curl -H "Host: microservice-alb" http://localhost:8888/api/products
# Expected: {"products":[{"id":1,"name":"Widget","price":9.99},{"id":2,"name":"Gadget","price":19.99}]}

# Info endpoint (shows which instance handled the request)
curl -H "Host: microservice-alb" http://localhost:8888/api/info
# Expected: {"service":"microservice-api","version":"1.0.0","instance":"...","timestamp":"..."}

# Test load balancing by making multiple requests
for i in {1..10}; do
  echo "Request $i:"
  curl -s -H "Host: microservice-alb" http://localhost:8888/api/info | jq -r '.instance'
done

# Clean up port forward
kill $PF_PID
```

### 3. Direct Task Testing

```bash
# Get all task pods
kubectl get pods -n default -l app=microservice-api

# Test each task directly
for pod in $(kubectl get pods -n default -l app=microservice-api -o jsonpath='{.items[*].metadata.name}'); do
  echo "Testing pod: $pod"
  kubectl exec -n default $pod -- wget -q -O - http://localhost:3000/health
done
```

### 4. Verify Load Distribution

```bash
# Check ECS service metrics
aws ecs describe-services \
  --cluster default \
  --services microservice-api \
  --endpoint-url http://localhost:8080 \
  --query 'services[0].{Service:serviceName,Desired:desiredCount,Running:runningCount,Pending:pendingCount}'

# Monitor target group health
watch -n 5 "aws elbv2 describe-target-health \
  --target-group-arn $TG_ARN \
  --endpoint-url http://localhost:8080 \
  --query 'TargetHealthDescriptions[*].{Target:Target.Id,Health:TargetHealth.State}' \
  --output table"
```

## Key Points to Verify

1. **Load Balancing**: Requests should be distributed across all healthy tasks
2. **Health Checks**: All targets should show as "healthy" in the target group
3. **Path Routing**: Different paths should route correctly to the service
4. **Task Scaling**: Service should maintain desired count of tasks
5. **Failover**: If a task fails, traffic should route to healthy tasks only

## Advanced Testing

### Load Testing

```bash
# Install hey (HTTP load generator)
go install github.com/rakyll/hey@latest

# Run load test
hey -z 30s -c 10 -H "Host: microservice-alb" http://localhost:8888/api/users

# Monitor during load test
kubectl top pods -n default -l app=microservice-api
```

### Chaos Testing

```bash
# Kill one task to test failover
TASK_ARN=$(aws ecs list-tasks \
  --cluster default \
  --service-name microservice-api \
  --endpoint-url http://localhost:8080 \
  --query 'taskArns[0]' --output text)

aws ecs stop-task \
  --cluster default \
  --task $TASK_ARN \
  --reason "Chaos testing" \
  --endpoint-url http://localhost:8080

# Verify service recovers
watch "aws ecs describe-services \
  --cluster default \
  --services microservice-api \
  --endpoint-url http://localhost:8080 \
  --query 'services[0].runningCount'"
```

## Troubleshooting

### Check ALB Access Logs

```bash
# Enable access logs (if supported)
aws elbv2 modify-load-balancer-attributes \
  --load-balancer-arn $ALB_ARN \
  --attributes Key=access_logs.s3.enabled,Value=true \
    Key=access_logs.s3.bucket,Value=my-alb-logs \
  --endpoint-url http://localhost:8080
```

### Debug Unhealthy Targets

```bash
# Check why targets are unhealthy
aws elbv2 describe-target-health \
  --target-group-arn $TG_ARN \
  --endpoint-url http://localhost:8080 \
  --query 'TargetHealthDescriptions[?TargetHealth.State!=`healthy`]'

# Check task logs
aws logs tail /ecs/microservice-api --follow \
  --endpoint-url http://localhost:8080
```

### Verify Network Configuration

```bash
# Check security group rules
aws ec2 describe-security-groups \
  --group-ids sg-alb-public sg-api-service \
  --endpoint-url http://localhost:8080
```

## Cleanup

```bash
# Delete listener rules
aws elbv2 describe-rules \
  --listener-arn $LISTENER_ARN \
  --endpoint-url http://localhost:8080 \
  --query 'Rules[?Priority!=`default`].RuleArn' \
  --output text | xargs -n1 aws elbv2 delete-rule \
  --endpoint-url http://localhost:8080 \
  --rule-arn

# Delete listener
aws elbv2 delete-listener \
  --listener-arn $LISTENER_ARN \
  --endpoint-url http://localhost:8080

# Delete target group
aws elbv2 delete-target-group \
  --target-group-arn $TG_ARN \
  --endpoint-url http://localhost:8080

# Delete load balancer
aws elbv2 delete-load-balancer \
  --load-balancer-arn $ALB_ARN \
  --endpoint-url http://localhost:8080

# Delete ECS service
aws ecs delete-service \
  --cluster default \
  --service microservice-api \
  --force \
  --endpoint-url http://localhost:8080

# Deregister task definition
aws ecs deregister-task-definition \
  --task-definition microservice-api:1 \
  --endpoint-url http://localhost:8080

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/microservice-api \
  --endpoint-url http://localhost:8080
```