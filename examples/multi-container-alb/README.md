# Multi-Container Application with ALB Example

This example demonstrates a multi-container application with Application Load Balancer integration, featuring frontend, backend API, and sidecar logging containers.

## Overview

- **Purpose**: Demonstrate multi-container task with ELBv2 integration
- **Components**:
  - Frontend: Nginx web server
  - Backend: Node.js API server
  - Sidecar: Logging utility
  - Application Load Balancer with Target Group
- **Features**:
  - Container dependencies (frontend waits for backend)
  - Shared volumes between containers
  - Health checks via ALB Target Group
  - Multiple container logging
  - Load balancing across multiple tasks
  - Public IP assignment for direct task access

## Architecture

```
┌─────────────────────────────────────────┐
│         ECS Task (Fargate)              │
│                                         │
│  ┌─────────────┐   ┌─────────────┐    │
│  │  Frontend   │   │   Backend   │    │
│  │   (nginx)   │──▶│   (API)     │    │
│  │   Port 80   │   │  Port 3000  │    │
│  └──────┬──────┘   └──────┬──────┘    │
│         │                  │            │
│         ▼                  ▼            │
│  ┌─────────────────────────────────┐   │
│  │     Shared Volume (/data)       │   │
│  └─────────────────────────────────┘   │
│         ▲                              │
│         │                              │
│  ┌──────┴──────┐                      │
│  │   Sidecar   │                      │
│  │  (logger)   │                      │
│  └─────────────┘                      │
└─────────────────────────────────────────┘
```

## Prerequisites

1. KECS running locally
2. AWS CLI configured to point to KECS endpoint

## Quick Start

```bash
# 1. Start KECS
kecs start

# 2. Setup Application Load Balancer
./setup_elb.sh

# 3. Deploy the service with ELB
./deploy.sh
```

## Manual Setup Instructions

If you prefer to set up resources manually:

### 1. Start KECS

```bash
kecs start
```

### 2. Create the ECS cluster

```bash
aws ecs create-cluster --cluster-name default \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### 3. Create CloudWatch Log Group

```bash
aws logs create-log-group \
  --log-group-name /ecs/multi-container-alb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

Note: The `ecsTaskExecutionRole` is automatically created by KECS when it starts LocalStack. No need to create it manually.

## Deployment

### Deployment with Application Load Balancer

This example uses ELBv2 for production-like deployment with load balancing and automatic public IP assignment.

```bash
# First set up the load balancer infrastructure
./setup_elb.sh

# This will:
# - Create Application Load Balancer (ALB)
# - Create Target Group with health checks
# - Configure HTTP Listener with routing rules

# Then deploy the service
./deploy.sh

# The deploy script will:
# - Detect the existing Target Group
# - Create the service with load balancer configuration
# - Wait for deployment to stabilize
```


#### Architecture

```
                         Internet
                            │
                            ▼
                  ┌─────────────────┐
                  │      ALB        │
                  │   (Port 80)     │
                  └────────┬────────┘
                           │
              ┌────────────┴────────────┐
              │    Target Group        │
              │  (Health Check: /)     │
              └────────────┬────────────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
   ┌───────────┐    ┌───────────┐    ┌───────────┐
   │  Task 1   │    │  Task 2   │    │  Task 3   │
   │ nginx:80  │    │ nginx:80  │    │ nginx:80  │
   └───────────┘    └───────────┘    └───────────┘
```

## Verification

### For Standard Deployment

#### 1. Check Service and Tasks

```bash
# Check service status
aws ecs describe-services \
  --cluster default \
  --services multi-container-alb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'services[0].{Status:status,Desired:desiredCount,Running:runningCount}'

# List tasks
TASK_ARNS=$(aws ecs list-tasks \
  --cluster default \
  --service-name multi-container-alb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'taskArns' --output json)

# Describe tasks
aws ecs describe-tasks \
  --cluster default \
  --tasks $TASK_ARNS \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'tasks[*].{TaskArn:taskArn,Status:lastStatus,Containers:containers[*].{Name:name,Status:lastStatus}}'
```

#### 2. Test Container Communication

```bash
# Get a task's pod name
POD_NAME=$(kubectl get pods -n default -l app=multi-container-alb -o jsonpath='{.items[0].metadata.name}')

# Port forward to access the frontend
kubectl port-forward -n default $POD_NAME 8080:80 &
PF_PID=$!

# Port forward to access the backend API
kubectl port-forward -n default $POD_NAME 3000:3000 &
PF_API_PID=$!

# Test frontend (nginx)
curl http://localhost:8080/
# Note: This might show default nginx page or error if no content is served

# Test backend API
curl http://localhost:3000/
# Expected: {"status":"healthy","timestamp":"2024-01-20T..."}

# Check if containers can communicate
kubectl exec -n default $POD_NAME -c frontend-nginx -- wget -q -O - http://localhost:3000
# Expected: {"status":"healthy","timestamp":"2024-01-20T..."}

# Clean up port forwards
kill $PF_PID $PF_API_PID
```

#### 3. Verify Shared Volume

```bash
# Check shared data written by backend
kubectl exec -n default $POD_NAME -c backend-api -- cat /data/status.json
# Expected: {"status":"ok","message":"API Running"}

# Check sidecar logger output
kubectl exec -n default $POD_NAME -c sidecar-logger -- tail -n 5 /data/health.log
# Expected: Multiple timestamped health check entries

# Verify frontend can read shared data
kubectl exec -n default $POD_NAME -c frontend-nginx -- ls -la /usr/share/nginx/html/
```

#### 4. Check Container Dependencies

```bash
# View container startup order in pod events
kubectl describe pod -n default $POD_NAME | grep -A 20 "Events:"

# Check container health status
kubectl get pod -n default $POD_NAME -o json | jq '.status.containerStatuses[] | {name: .name, ready: .ready, started: .started}'
```

### For ELBv2 Deployment

#### 1. Check ALB and Target Health

```bash
# Get ALB details
ALB_ARN=$(aws elbv2 describe-load-balancers \
  --names multi-container-alb-alb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'LoadBalancers[0].LoadBalancerArn' --output text)

ALB_DNS=$(aws elbv2 describe-load-balancers \
  --load-balancer-arns $ALB_ARN \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'LoadBalancers[0].DNSName' --output text)

echo "ALB DNS: $ALB_DNS"

# Check target health
TG_ARN=$(aws elbv2 describe-target-groups \
  --names multi-container-alb-tg \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'TargetGroups[0].TargetGroupArn' --output text)

aws elbv2 describe-target-health \
  --target-group-arn $TG_ARN \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'TargetHealthDescriptions[*].{Target:Target.Id,Health:TargetHealth.State}' \
  --output table
```

#### 2. Test through Load Balancer

Since KECS runs in Kubernetes, access the ALB through port-forwarding:

```bash
# Port forward to Traefik (acting as ALB)
kubectl port-forward -n kecs-system svc/traefik 8888:80 &
PF_ALB=$!

# Test through load balancer
curl -H "Host: multi-container-alb-alb" http://localhost:8888/

# Test API endpoint through ALB
curl -H "Host: multi-container-alb-alb" http://localhost:8888/api/status

# Test health check endpoint
curl -H "Host: multi-container-alb-alb" http://localhost:8888/health

# Test load balancing across multiple tasks
for i in {1..10}; do
  echo "Request $i:"
  curl -s -H "Host: multi-container-alb-alb" http://localhost:8888/ | head -n 1
  sleep 0.5
done

# Monitor target group health
watch -n 5 "aws elbv2 describe-target-health \
  --target-group-arn $TG_ARN \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'TargetHealthDescriptions[*].{Target:Target.Id,Health:TargetHealth.State}' \
  --output table"

# Clean up port forward
kill $PF_ALB
```

#### 3. Test Failover and Auto-Recovery

```bash
# Get a task ARN
TASK_ARN=$(aws ecs list-tasks \
  --cluster multi-container-cluster \
  --service-name multi-container-alb-elb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'taskArns[0]' --output text)

# Stop one task to test failover
aws ecs stop-task \
  --cluster multi-container-cluster \
  --task $TASK_ARN \
  --reason "Testing failover" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Monitor service recovery (ECS should launch a new task automatically)
watch "aws ecs describe-services \
  --cluster multi-container-cluster \
  --services multi-container-alb-elb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'services[0].{Desired:desiredCount,Running:runningCount,Pending:pendingCount}'"

# Verify new task is registered with target group
aws elbv2 describe-target-health \
  --target-group-arn $TG_ARN \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'TargetHealthDescriptions[*].{Target:Target.Id,Health:TargetHealth.State,Description:TargetHealth.Description}' \
  --output table
```

## Key Points to Verify

1. **Container Dependencies**: Frontend should start only after backend is healthy
2. **Shared Volume**: All containers should access the same volume
3. **Inter-container Communication**: Frontend can reach backend on localhost:3000
4. **Health Checks**: Backend health check should pass
5. **Logging**: Each container logs to separate CloudWatch streams

## Troubleshooting

### Check Individual Container Logs

```bash
# Frontend logs
kubectl logs -n default $POD_NAME -c frontend-nginx

# Backend logs
kubectl logs -n default $POD_NAME -c backend-api

# Sidecar logs
kubectl logs -n default $POD_NAME -c sidecar-logger

# View CloudWatch logs
aws logs tail /ecs/multi-container-alb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --follow

# Filter logs by container
aws logs filter-log-events \
  --log-group-name /ecs/multi-container-alb \
  --log-stream-name-prefix "frontend-nginx" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### Verify Container Health

```bash
# Check if backend health check is passing
kubectl exec -n default $POD_NAME -c backend-api -- wget -q -O - http://localhost:3000
```

### Debug Shared Volume Issues

```bash
# List volume mounts in each container
kubectl describe pod -n default $POD_NAME | grep -A 5 "Mounts:"
```

## Cleanup

### For Standard Deployment

```bash
# Delete service
aws ecs delete-service \
  --cluster default \
  --service multi-container-alb \
  --force \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Wait for service deletion
aws ecs wait services-inactive \
  --cluster default \
  --services multi-container-alb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Deregister task definition
aws ecs deregister-task-definition \
  --task-definition multi-container-alb:1 \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/multi-container-alb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete cluster (if created for this example)
aws ecs delete-cluster \
  --cluster default \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### For ELBv2 Deployment

#### Complete Cleanup (Recommended)

```bash
# Remove ALL resources with a single script
./cleanup_all.sh

# This removes:
# - ECS Service
# - All running tasks
# - Application Load Balancer
# - Target Group and Listeners
# - Task Definitions
# - CloudWatch Log Group
# - ECS Cluster
# - Generated configuration files
```


## Advanced Testing

### Load Testing Multiple Containers

```bash
# Scale the service
aws ecs update-service \
  --cluster default \
  --service multi-container-alb \
  --desired-count 3 \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Verify all pods are running
kubectl get pods -n default -l app=multi-container-alb

# Test load distribution
for i in {1..10}; do
  POD=$(kubectl get pods -n default -l app=multi-container-alb -o jsonpath="{.items[$((i%3))].metadata.name}")
  echo "Testing pod: $POD"
  kubectl exec -n default $POD -c backend-api -- wget -q -O - http://localhost:3000
done
```