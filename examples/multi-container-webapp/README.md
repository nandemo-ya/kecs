# Multi-Container Web Application Example

This example demonstrates a multi-container web application with frontend, backend API, and sidecar logging containers.

## Overview

- **Purpose**: Show multi-container task with dependencies and shared volumes
- **Components**: 
  - Frontend: Nginx web server
  - Backend: Node.js API server
  - Sidecar: Logging utility
- **Features**:
  - Container dependencies (frontend waits for backend)
  - Shared volumes between containers
  - Health checks
  - Multiple container logging

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
3. ecspresso installed

## Setup Instructions

### 1. Start KECS

```bash
kecs start
```

### 2. Create the ECS cluster

```bash
aws ecs create-cluster --cluster-name default \
  --endpoint-url http://localhost:8080
```

### 3. Create CloudWatch Log Group

```bash
aws logs create-log-group \
  --log-group-name /ecs/multi-container-webapp \
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

# Attach policy for ECR and CloudWatch Logs
aws iam attach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy \
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

## Deployment

### Using ecspresso

```bash
# Deploy the service
ecspresso deploy --config ecspresso.yml

# Check deployment status
ecspresso status --config ecspresso.yml

# View logs
ecspresso logs --config ecspresso.yml --container frontend-nginx
ecspresso logs --config ecspresso.yml --container backend-api
ecspresso logs --config ecspresso.yml --container sidecar-logger
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

### 1. Check Service and Tasks

```bash
# Check service status
aws ecs describe-services \
  --cluster default \
  --services multi-container-webapp \
  --endpoint-url http://localhost:8080 \
  --query 'services[0].{Status:status,Desired:desiredCount,Running:runningCount}'

# List tasks
TASK_ARNS=$(aws ecs list-tasks \
  --cluster default \
  --service-name multi-container-webapp \
  --endpoint-url http://localhost:8080 \
  --query 'taskArns' --output json)

# Describe tasks
aws ecs describe-tasks \
  --cluster default \
  --tasks $TASK_ARNS \
  --endpoint-url http://localhost:8080 \
  --query 'tasks[*].{TaskArn:taskArn,Status:lastStatus,Containers:containers[*].{Name:name,Status:lastStatus}}'
```

### 2. Test Container Communication

```bash
# Get a task's pod name
POD_NAME=$(kubectl get pods -n default -l app=multi-container-webapp -o jsonpath='{.items[0].metadata.name}')

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

### 3. Verify Shared Volume

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

### 4. Check Container Dependencies

```bash
# View container startup order in pod events
kubectl describe pod -n default $POD_NAME | grep -A 20 "Events:"

# Check container health status
kubectl get pod -n default $POD_NAME -o json | jq '.status.containerStatuses[] | {name: .name, ready: .ready, started: .started}'
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

```bash
# Delete service
aws ecs delete-service \
  --cluster default \
  --service multi-container-webapp \
  --force \
  --endpoint-url http://localhost:8080

# Wait for service deletion
aws ecs wait services-inactive \
  --cluster default \
  --services multi-container-webapp \
  --endpoint-url http://localhost:8080

# Deregister task definition
aws ecs deregister-task-definition \
  --task-definition multi-container-webapp:1 \
  --endpoint-url http://localhost:8080

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/multi-container-webapp \
  --endpoint-url http://localhost:8080

# Delete IAM roles (if created for this example)
aws iam detach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy \
  --endpoint-url http://localhost:8080

aws iam delete-role --role-name ecsTaskExecutionRole --endpoint-url http://localhost:8080
aws iam delete-role --role-name ecsTaskRole --endpoint-url http://localhost:8080
```

## Advanced Testing

### Load Testing Multiple Containers

```bash
# Scale the service
aws ecs update-service \
  --cluster default \
  --service multi-container-webapp \
  --desired-count 3 \
  --endpoint-url http://localhost:8080

# Verify all pods are running
kubectl get pods -n default -l app=multi-container-webapp

# Test load distribution
for i in {1..10}; do
  POD=$(kubectl get pods -n default -l app=multi-container-webapp -o jsonpath="{.items[$((i%3))].metadata.name}")
  echo "Testing pod: $POD"
  kubectl exec -n default $POD -c backend-api -- wget -q -O - http://localhost:3000
done
```