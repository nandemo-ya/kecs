# Batch Job Simple Example

This example demonstrates running one-off batch jobs as ECS tasks without creating a service.

## Overview

- **Purpose**: Show how to run standalone tasks for batch processing
- **Components**: 
  - Single container that performs batch processing
  - No service definition (runs as standalone task)
- **Use Cases**:
  - Data processing jobs
  - Report generation
  - Database migrations
  - Cleanup tasks
  - Scheduled jobs (with external scheduler)

## Architecture

```
┌─────────────────┐
│   Run Task API  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   ECS Task      │
│                 │
│ • Start         │
│ • Process       │
│ • Complete      │
│ • Exit          │
└─────────────────┘
         │
         ▼
    Task Stopped
```

## Prerequisites

1. KECS running locally
2. AWS CLI configured to point to KECS endpoint

## Setup Instructions

### 1. Start KECS

```bash
kecs start
```

### 2. Create the ECS Cluster

```bash
aws ecs create-cluster --cluster-name default \
  --endpoint-url http://localhost:4566
```

### 3. Create CloudWatch Log Group

```bash
aws logs create-log-group \
  --log-group-name /ecs/batch-job-simple \
  --endpoint-url http://localhost:4566
```

Note: The `ecsTaskExecutionRole` is automatically created by KECS when it starts LocalStack. No need to create it manually.

## Running Batch Jobs

### Using AWS CLI

```bash
# Register task definition
aws ecs register-task-definition \
  --cli-input-json file://task_def.json \
  --endpoint-url http://localhost:4566

# Run a single task
aws ecs run-task \
  --cluster default \
  --task-definition batch-job-simple \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],securityGroups=[sg-batch-job],assignPublicIp=ENABLED}" \
  --endpoint-url http://localhost:4566

# Run task with overrides
aws ecs run-task \
  --cluster default \
  --task-definition batch-job-simple \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],securityGroups=[sg-batch-job],assignPublicIp=ENABLED}" \
  --overrides '{
    "containerOverrides": [{
      "name": "batch-processor",
      "environment": [
        {"name": "BATCH_SIZE", "value": "100"},
        {"name": "JOB_TYPE", "value": "hourly-report"}
      ]
    }]
  }' \
  --endpoint-url http://localhost:4566
```

## Monitoring and Verification

### 1. List Running Tasks

```bash
# List all tasks in the cluster
aws ecs list-tasks \
  --cluster default \
  --endpoint-url http://localhost:4566

# List only running tasks
aws ecs list-tasks \
  --cluster default \
  --desired-status RUNNING \
  --endpoint-url http://localhost:4566

# List stopped tasks
aws ecs list-tasks \
  --cluster default \
  --desired-status STOPPED \
  --endpoint-url http://localhost:4566
```

### 2. Monitor Task Execution

```bash
# Get task ARN
TASK_ARN=$(aws ecs list-tasks \
  --cluster default \
  --desired-status RUNNING \
  --endpoint-url http://localhost:4566 \
  --query 'taskArns[0]' --output text)

# Watch task status
watch -n 2 "aws ecs describe-tasks \
  --cluster default \
  --tasks $TASK_ARN \
  --endpoint-url http://localhost:4566 \
  --query 'tasks[0].{Status:lastStatus,Started:startedAt,Stopped:stoppedAt,Exit:containers[0].exitCode}'"
```

### 3. View Task Logs

```bash
# Stream logs in real-time
aws logs tail /ecs/batch-job-simple \
  --follow \
  --endpoint-url http://localhost:4566

# Get logs for a specific task
aws logs filter-log-events \
  --log-group-name /ecs/batch-job-simple \
  --filter-pattern "Batch job" \
  --endpoint-url http://localhost:4566
```

### 4. Check Task Output in Kubernetes

```bash
# Find the pod for the task
POD_NAME=$(kubectl get pods -n default -l app=batch-job-simple --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')

# If the task is still running, exec into it
kubectl exec -n default $POD_NAME -- cat /tmp/report.json

# View pod logs
kubectl logs -n default $POD_NAME

# For completed tasks, check completed pods
kubectl get pods -n default -l app=batch-job-simple --field-selector=status.phase=Succeeded
```

## Common Batch Job Patterns

### 1. Scheduled Jobs

```bash
# Run a job every hour using cron
# Add to crontab:
0 * * * * aws ecs run-task \
  --cluster default \
  --task-definition batch-job-simple \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],securityGroups=[sg-batch-job],assignPublicIp=ENABLED}" \
  --overrides '{"containerOverrides":[{"name":"batch-processor","environment":[{"name":"JOB_TYPE","value":"hourly-report"}]}]}' \
  --endpoint-url http://localhost:4566
```

### 2. Parallel Processing

```bash
# Run multiple tasks in parallel for large datasets
TOTAL_ITEMS=1000
BATCH_SIZE=100
TASKS=$((TOTAL_ITEMS / BATCH_SIZE))

for i in $(seq 0 $((TASKS-1))); do
  START=$((i * BATCH_SIZE))
  END=$((START + BATCH_SIZE - 1))
  
  aws ecs run-task \
    --cluster default \
    --task-definition batch-job-simple \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],securityGroups=[sg-batch-job],assignPublicIp=ENABLED}" \
    --overrides "{
      \"containerOverrides\": [{
        \"name\": \"batch-processor\",
        \"environment\": [
          {\"name\": \"START_INDEX\", \"value\": \"$START\"},
          {\"name\": \"END_INDEX\", \"value\": \"$END\"},
          {\"name\": \"JOB_ID\", \"value\": \"batch-$i\"}
        ]
      }]
    }" \
    --endpoint-url http://localhost:4566 &
done

wait
echo "All batch jobs submitted"
```

### 3. Chain Jobs with Dependencies

```bash
# Run job 1
TASK1=$(aws ecs run-task \
  --cluster default \
  --task-definition batch-job-simple \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],securityGroups=[sg-batch-job],assignPublicIp=ENABLED}" \
  --overrides '{"containerOverrides":[{"name":"batch-processor","environment":[{"name":"JOB_TYPE","value":"stage-1"}]}]}' \
  --endpoint-url http://localhost:4566 \
  --query 'tasks[0].taskArn' --output text)

# Wait for job 1 to complete
aws ecs wait tasks-stopped \
  --cluster default \
  --tasks $TASK1 \
  --endpoint-url http://localhost:4566

# Check exit code
EXIT_CODE=$(aws ecs describe-tasks \
  --cluster default \
  --tasks $TASK1 \
  --endpoint-url http://localhost:4566 \
  --query 'tasks[0].containers[0].exitCode' --output text)

if [ "$EXIT_CODE" -eq "0" ]; then
  echo "Stage 1 successful, running stage 2..."
  aws ecs run-task \
    --cluster default \
    --task-definition batch-job-simple \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],securityGroups=[sg-batch-job],assignPublicIp=ENABLED}" \
    --overrides '{"containerOverrides":[{"name":"batch-processor","environment":[{"name":"JOB_TYPE","value":"stage-2"}]}]}' \
    --endpoint-url http://localhost:4566
else
  echo "Stage 1 failed with exit code $EXIT_CODE"
fi
```

## Key Points to Verify

1. **Task Completion**: Tasks should complete and exit with code 0
2. **Log Output**: All processing steps should be logged
3. **Resource Cleanup**: Completed tasks should release resources
4. **Error Handling**: Failed tasks should exit with non-zero code
5. **Idempotency**: Tasks should be safe to retry

## Troubleshooting

### Task Fails to Start

```bash
# Check task failure reason
aws ecs describe-tasks \
  --cluster default \
  --tasks $TASK_ARN \
  --endpoint-url http://localhost:4566 \
  --query 'tasks[0].{StoppedReason:stoppedReason,StopCode:stopCode}'

# Check container status
kubectl describe pod -n default $POD_NAME
```

### Task Runs Forever

```bash
# Stop a running task
aws ecs stop-task \
  --cluster default \
  --task $TASK_ARN \
  --reason "Task timeout" \
  --endpoint-url http://localhost:4566
```

### Debug Task Environment

```bash
# Run interactive debug task
aws ecs run-task \
  --cluster default \
  --task-definition batch-job-simple \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],securityGroups=[sg-batch-job],assignPublicIp=ENABLED}" \
  --overrides '{
    "containerOverrides": [{
      "name": "batch-processor",
      "command": ["sh", "-c", "sleep 3600"]
    }]
  }' \
  --endpoint-url http://localhost:4566

# Then exec into the container
kubectl exec -it -n default $POD_NAME -- sh
```

## Cleanup

```bash
# Stop all running tasks
aws ecs list-tasks \
  --cluster default \
  --desired-status RUNNING \
  --endpoint-url http://localhost:4566 \
  --query 'taskArns[]' --output text | \
xargs -n1 aws ecs stop-task \
  --cluster default \
  --endpoint-url http://localhost:4566 \
  --task

# Deregister task definition
aws ecs deregister-task-definition \
  --task-definition batch-job-simple:1 \
  --endpoint-url http://localhost:4566

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/batch-job-simple \
  --endpoint-url http://localhost:4566

# Clean up completed pods in Kubernetes
kubectl delete pods -n default -l app=batch-job-simple --field-selector=status.phase=Succeeded
kubectl delete pods -n default -l app=batch-job-simple --field-selector=status.phase=Failed
```

## Best Practices

1. **Idempotency**: Design jobs to be safely retryable
2. **Timeout**: Set appropriate timeouts for long-running jobs
3. **Error Handling**: Exit with proper codes for monitoring
4. **Resource Limits**: Set CPU and memory limits appropriately
5. **Logging**: Log progress for debugging and monitoring
6. **Cleanup**: Ensure temporary files and resources are cleaned up