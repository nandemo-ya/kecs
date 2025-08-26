#!/bin/bash

# Test script for ECS Task Management APIs (RunTask, StartTask, ListTasks)

ENDPOINT="http://localhost:5373"

echo "=== Prerequisites: Register a task definition ==="
aws ecs register-task-definition \
  --endpoint-url $ENDPOINT \
  --family test-app \
  --region ap-northeast-1 \
  --requires-compatibilities EC2 \
  --network-mode bridge \
  --cpu "256" \
  --memory "512" \
  --container-definitions '[
    {
      "name": "main",
      "image": "nginx:latest",
      "memory": 512,
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "hostPort": 8080,
          "protocol": "tcp"
        }
      ]
    }
  ]'

echo -e "\n=== Testing RunTask API ==="
aws ecs run-task \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --task-definition test-app:1 \
  --count 2 \
  --launch-type EC2 \
  --started-by test-user \
  --group test-group \
  --region ap-northeast-1

echo -e "\n=== Testing ListTasks API ==="
aws ecs list-tasks \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --region ap-northeast-1

echo -e "\n=== Testing ListTasks API with filters ==="
aws ecs list-tasks \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --desired-status RUNNING \
  --launch-type EC2 \
  --started-by test-user \
  --region ap-northeast-1

echo -e "\n=== Testing ListTasks API with pagination ==="
aws ecs list-tasks \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --max-results 2 \
  --region ap-northeast-1

echo -e "\n=== Get task ARN for further testing ==="
TASK_ARN=$(aws ecs list-tasks \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --region ap-northeast-1 \
  --query 'taskArns[0]' \
  --output text)

echo "Using task ARN: $TASK_ARN"

echo -e "\n=== Testing DescribeTasks API ==="
aws ecs describe-tasks \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --tasks "$TASK_ARN" \
  --region ap-northeast-1

echo -e "\n=== Testing StartTask API (requires container instance) ==="
# First get a container instance ARN
INSTANCE_ARN=$(aws ecs list-container-instances \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --region ap-northeast-1 \
  --query 'containerInstanceArns[0]' \
  --output text)

if [ "$INSTANCE_ARN" != "None" ] && [ -n "$INSTANCE_ARN" ]; then
  echo "Using container instance ARN: $INSTANCE_ARN"
  
  aws ecs start-task \
    --endpoint-url $ENDPOINT \
    --cluster default \
    --task-definition test-app:1 \
    --container-instances "$INSTANCE_ARN" \
    --started-by test-user \
    --group test-group \
    --region ap-northeast-1
else
  echo "No container instance available, skipping StartTask test"
fi

echo -e "\n=== Testing StopTask API ==="
if [ "$TASK_ARN" != "None" ] && [ -n "$TASK_ARN" ]; then
  aws ecs stop-task \
    --endpoint-url $ENDPOINT \
    --cluster default \
    --task "$TASK_ARN" \
    --reason "Test completed" \
    --region ap-northeast-1
else
  echo "No task available to stop"
fi

echo -e "\n=== Final ListTasks to see updated status ==="
aws ecs list-tasks \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --region ap-northeast-1