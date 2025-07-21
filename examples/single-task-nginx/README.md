# Single Task Nginx Example

This example demonstrates a simple nginx web server deployment using KECS with a single container task.

## Overview

- **Purpose**: Basic web server deployment
- **Components**: Single nginx container
- **Network**: Public IP with security group
- **Launch Type**: Fargate

## Prerequisites

Before deploying this example, ensure you have:

1. KECS running locally
2. AWS CLI configured to point to KECS endpoint

## Setup Instructions

### 1. Start KECS (if not already running)

```bash
kecs start
```

### 2. Create the ECS cluster

```bash
aws ecs create-cluster --cluster-name default \
  --endpoint-url http://localhost:4566
```

### 3. Create CloudWatch Log Group

```bash
aws logs create-log-group \
  --log-group-name /ecs/single-task-nginx \
  --endpoint-url http://localhost:4566
```

Note: The `ecsTaskExecutionRole` is automatically created by KECS when it starts LocalStack. No need to create it manually.

## Deployment

### Using AWS CLI

```bash
# Register task definition
aws ecs register-task-definition \
  --cli-input-json file://task_def.json \
  --endpoint-url http://localhost:4566

# Create service
aws ecs create-service \
  --cli-input-json file://service_def.json \
  --endpoint-url http://localhost:4566
```

## Verification

### 1. Check Service Status

```bash
aws ecs describe-services \
  --cluster default \
  --services single-task-nginx \
  --endpoint-url http://localhost:4566
```

### 2. List Running Tasks

```bash
aws ecs list-tasks \
  --cluster default \
  --service-name single-task-nginx \
  --endpoint-url http://localhost:4566
```

### 3. Get Task Details

```bash
# Get task ARN from list-tasks output
TASK_ARN=$(aws ecs list-tasks \
  --cluster default \
  --service-name single-task-nginx \
  --endpoint-url http://localhost:4566 \
  --query 'taskArns[0]' --output text)

# Describe task to get IP address
TASK_IP=$(aws ecs describe-tasks \
  --cluster default \
  --tasks $TASK_ARN \
  --endpoint-url http://localhost:4566 \
  --query 'tasks[0].containers[0].networkInterfaces[0].privateIpv4Address' \
  --output text)
```

### 4. Test the Nginx Server

Since KECS runs tasks in Kubernetes, you can access the nginx server through port-forwarding:

```bash
# Get the pod name
POD_NAME=$(kubectl get pods -n default -l app=single-task-nginx -o jsonpath='{.items[0].metadata.name}')

# Port forward to access nginx
kubectl port-forward -n default $POD_NAME 8888:80

# In another terminal, test nginx
curl http://localhost:8888/
# Expected output: Default nginx welcome page HTML
```

## Key Points to Verify

1. **Task Status**: Should be RUNNING
2. **Service Status**: desiredCount should match runningCount
3. **Health Checks**: Container should pass health checks
4. **Logs**: Check CloudWatch logs for any errors
5. **Network Access**: Nginx should respond on port 80

## Troubleshooting

### Check Task Logs

```bash
aws logs tail /ecs/single-task-nginx \
  --endpoint-url http://localhost:4566 \
  --follow
```

### Check Pod Status in Kubernetes

```bash
kubectl get pods -n default -l app=single-task-nginx
kubectl describe pod -n default $POD_NAME
```

## Cleanup

```bash
# Delete service
aws ecs delete-service \
  --cluster default \
  --service single-task-nginx \
  --force \
  --endpoint-url http://localhost:4566

# Deregister task definition
aws ecs deregister-task-definition \
  --task-definition single-task-nginx:1 \
  --endpoint-url http://localhost:4566

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/single-task-nginx \
  --endpoint-url http://localhost:4566
```