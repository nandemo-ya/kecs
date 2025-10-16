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
2. Terraform installed (>= 1.0)
3. AWS CLI configured

## Quick Start with Terraform (Recommended)

### 1. Start KECS

```bash
# Start KECS instance
kecs start

# Wait for KECS to be ready
kecs list
```

### 2. Deploy Infrastructure with Terraform

```bash
# Initialize Terraform
terraform init

# Review the planned changes
terraform plan

# Apply the configuration
terraform apply

# Type 'yes' when prompted
```

This will create:
- ECS Cluster: `multi-container-alb`
- CloudWatch Logs Log Group: `/ecs/multi-container-alb` (7 days retention)
- Application Load Balancer: `multi-container-alb-alb`
- Target Group: `multi-container-alb-tg` (with health checks)
- HTTP Listener on port 80
- Listener Rules for `/api/*`, `/static/*`, and `/health` paths

### 3. Get Target Group ARN

After Terraform completes, get the Target Group ARN for service deployment:

```bash
# Show all outputs
terraform output

# Get Target Group ARN specifically
terraform output target_group_arn
```

### 4. Update Service Definition

The `service_def_with_elb.json` file contains a placeholder Target Group ARN. Update it with the actual ARN from Terraform output:

```bash
# Get the ARN and update the file
TG_ARN=$(terraform output -raw target_group_arn)
echo "Target Group ARN: $TG_ARN"

# Update service_def_with_elb.json manually or use this command:
sed -i.bak "s|arn:aws:elasticloadbalancing:us-east-1:000000000000:targetgroup/multi-container-alb-tg/[^\"]*|$TG_ARN|" service_def_with_elb.json
```

### 5. Verify Infrastructure

```bash
# Verify ECS cluster
aws ecs describe-clusters \
  --cluster multi-container-alb \
  --endpoint-url http://localhost:5373 \
  --region us-east-1

# Verify ALB
aws elbv2 describe-load-balancers \
  --names multi-container-alb-alb \
  --endpoint-url http://localhost:5373 \
  --region us-east-1

# Verify Target Group
aws elbv2 describe-target-groups \
  --names multi-container-alb-tg \
  --endpoint-url http://localhost:5373 \
  --region us-east-1
```

### 6. Terraform Configuration

You can customize the configuration by creating a `terraform.tfvars` file:

```hcl
aws_region      = "us-east-1"
kecs_endpoint   = "http://localhost:5373"
cluster_name    = "multi-container-alb"
service_name    = "multi-container-alb"
environment     = "development"
vpc_id          = "vpc-12345678"
subnets         = ["subnet-12345678", "subnet-87654321"]
security_groups = ["sg-webapp"]
```

Or override via command line:

```bash
terraform apply -var="cluster_name=my-cluster" -var="environment=staging"
```

## Deployment

### Deploy ECS Service with Application Load Balancer

After setting up the infrastructure with Terraform, deploy the multi-container service:

```bash
export AWS_ENDPOINT_URL=http://localhost:5373
export AWS_REGION=us-east-1

# 1. Register the task definition
aws ecs register-task-definition \
  --cli-input-json file://task_def.json \
  --region us-east-1

# 2. Create the service with load balancer configuration
aws ecs create-service \
  --cli-input-json file://service_def_with_elb.json \
  --region us-east-1

# 3. Check service status
aws ecs describe-services \
  --cluster multi-container-alb \
  --services multi-container-alb \
  --region us-east-1 \
  --query 'services[0].{Status:status,Desired:desiredCount,Running:runningCount}'
```

The service will:
- Deploy 3 task replicas
- Register tasks with the Target Group automatically
- Enable deployment circuit breaker with rollback
- Use spread placement strategy across availability zones


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
  --cluster multi-container-alb \
  --services multi-container-alb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'services[0].{Status:status,Desired:desiredCount,Running:runningCount}'

# List tasks
TASK_ARNS=$(aws ecs list-tasks \
  --cluster multi-container-alb \
  --service-name multi-container-alb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'taskArns' --output json)

# Describe tasks
aws ecs describe-tasks \
  --cluster multi-container-alb \
  --tasks $TASK_ARNS \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'tasks[*].{TaskArn:taskArn,Status:lastStatus,Containers:containers[*].{Name:name,Status:lastStatus}}'
```

#### 2. Test Container Communication

```bash
# Get a task's pod name
POD_NAME=$(kubectl get pods -n multi-container-alb-us-east-1 -l app=multi-container-alb -o jsonpath='{.items[0].metadata.name}')

# Port forward to access the frontend
kubectl port-forward -n multi-container-alb-us-east-1 $POD_NAME 8080:80 &
PF_PID=$!

# Port forward to access the backend API
kubectl port-forward -n multi-container-alb-us-east-1 $POD_NAME 3000:3000 &
PF_API_PID=$!

# Test frontend (nginx)
curl http://localhost:8080/
# Note: This might show default nginx page or error if no content is served

# Test backend API
curl http://localhost:3000/
# Expected: {"status":"healthy","timestamp":"2024-01-20T..."}

# Check if containers can communicate
kubectl exec -n multi-container-alb-us-east-1 $POD_NAME -c frontend-nginx -- wget -q -O - http://localhost:3000
# Expected: {"status":"healthy","timestamp":"2024-01-20T..."}

# Clean up port forwards
kill $PF_PID $PF_API_PID
```

#### 3. Verify Shared Volume

```bash
# Check shared data written by backend
kubectl exec -n multi-container-alb-us-east-1 $POD_NAME -c backend-api -- cat /data/status.json
# Expected: {"status":"ok","message":"API Running"}

# Check sidecar logger output
kubectl exec -n multi-container-alb-us-east-1 $POD_NAME -c sidecar-logger -- tail -n 5 /data/health.log
# Expected: Multiple timestamped health check entries

# Verify frontend can read shared data
kubectl exec -n multi-container-alb-us-east-1 $POD_NAME -c frontend-nginx -- ls -la /usr/share/nginx/html/
```

#### 4. Check Container Dependencies

```bash
# View container startup order in pod events
kubectl describe pod -n multi-container-alb-us-east-1 $POD_NAME | grep -A 20 "Events:"

# Check container health status
kubectl get pod -n multi-container-alb-us-east-1 $POD_NAME -o json | jq '.status.containerStatuses[] | {name: .name, ready: .ready, started: .started}'
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
kubectl logs -n multi-container-alb-us-east-1 $POD_NAME -c frontend-nginx

# Backend logs
kubectl logs -n multi-container-alb-us-east-1 $POD_NAME -c backend-api

# Sidecar logs
kubectl logs -n multi-container-alb-us-east-1 $POD_NAME -c sidecar-logger

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
kubectl exec -n multi-container-alb-us-east-1 $POD_NAME -c backend-api -- wget -q -O - http://localhost:3000
```

### Debug Shared Volume Issues

```bash
# List volume mounts in each container
kubectl describe pod -n multi-container-alb-us-east-1 $POD_NAME | grep -A 5 "Mounts:"
```

## Cleanup

### Using Terraform (Recommended)

First, delete the ECS service manually (Terraform doesn't manage the service):

```bash
# Delete the ECS service
aws ecs delete-service \
  --cluster multi-container-alb \
  --service multi-container-alb \
  --force \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Wait for service deletion (optional)
aws ecs wait services-inactive \
  --cluster multi-container-alb \
  --services multi-container-alb \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

Then destroy all infrastructure with Terraform:

```bash
# Destroy all infrastructure
terraform destroy

# Type 'yes' when prompted
```

This will remove:
- Application Load Balancer
- Target Group and Listener Rules
- ECS Cluster
- CloudWatch Logs Log Group

### Manual Cleanup (Alternative)

<details>
<summary>Click to expand manual cleanup instructions</summary>

```bash
export AWS_ENDPOINT_URL=http://localhost:5373
export AWS_REGION=us-east-1

# 1. Delete ECS service
aws ecs delete-service \
  --cluster multi-container-alb \
  --service multi-container-alb \
  --force \
  --region us-east-1

# 2. Get ALB and Target Group ARNs
ALB_ARN=$(aws elbv2 describe-load-balancers \
  --names multi-container-alb-alb \
  --region us-east-1 \
  --query 'LoadBalancers[0].LoadBalancerArn' --output text)

TG_ARN=$(aws elbv2 describe-target-groups \
  --names multi-container-alb-tg \
  --region us-east-1 \
  --query 'TargetGroups[0].TargetGroupArn' --output text)

# 3. Delete Listener
LISTENER_ARN=$(aws elbv2 describe-listeners \
  --load-balancer-arn $ALB_ARN \
  --region us-east-1 \
  --query 'Listeners[0].ListenerArn' --output text)

aws elbv2 delete-listener \
  --listener-arn $LISTENER_ARN \
  --region us-east-1

# 4. Delete Load Balancer
aws elbv2 delete-load-balancer \
  --load-balancer-arn $ALB_ARN \
  --region us-east-1

# 5. Delete Target Group
aws elbv2 delete-target-group \
  --target-group-arn $TG_ARN \
  --region us-east-1

# 6. Delete Log Group
aws logs delete-log-group \
  --log-group-name /ecs/multi-container-alb \
  --region us-east-1

# 7. Delete ECS Cluster
aws ecs delete-cluster \
  --cluster multi-container-alb \
  --region us-east-1
```

</details>


## Advanced Testing

### Load Testing Multiple Containers

```bash
# Scale the service
aws ecs update-service \
  --cluster multi-container-alb \
  --service multi-container-alb \
  --desired-count 3 \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Verify all pods are running
kubectl get pods -n multi-container-alb-us-east-1 -l app=multi-container-alb

# Test load distribution
for i in {1..10}; do
  POD=$(kubectl get pods -n multi-container-alb-us-east-1 -l app=multi-container-alb -o jsonpath="{.items[$((i%3))].metadata.name}")
  echo "Testing pod: $POD"
  kubectl exec -n multi-container-alb-us-east-1 $POD -c backend-api -- wget -q -O - http://localhost:3000
done
```