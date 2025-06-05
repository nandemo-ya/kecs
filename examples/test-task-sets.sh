#!/bin/bash

# Test script for ECS Task Set APIs

ENDPOINT="http://localhost:8080"

echo "=== Prerequisites: Create a service ==="
aws ecs create-service \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --service-name test-service \
  --task-definition test-app:1 \
  --desired-count 3 \
  --deployment-controller type=EXTERNAL \
  --region ap-northeast-1

echo -e "\n=== Testing CreateTaskSet API ==="
aws ecs create-task-set \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --service test-service \
  --task-definition test-app:1 \
  --external-id "external-deployment-1" \
  --launch-type EC2 \
  --scale unit=PERCENT,value=100 \
  --region ap-northeast-1

echo -e "\n=== Testing DescribeTaskSets API (all task sets for service) ==="
aws ecs describe-task-sets \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --service test-service \
  --region ap-northeast-1

echo -e "\n=== Testing CreateTaskSet API with specific configuration ==="
aws ecs create-task-set \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --service test-service \
  --task-definition test-app:1 \
  --external-id "external-deployment-2" \
  --launch-type FARGATE \
  --platform-version "LATEST" \
  --scale unit=PERCENT,value=50 \
  --region ap-northeast-1

echo -e "\n=== Get task set ID for further testing ==="
TASK_SET_ID=$(aws ecs describe-task-sets \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --service test-service \
  --region ap-northeast-1 \
  --query 'taskSets[0].id' \
  --output text)

echo "Using task set ID: $TASK_SET_ID"

echo -e "\n=== Testing DescribeTaskSets API (specific task set) ==="
aws ecs describe-task-sets \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --service test-service \
  --task-sets "$TASK_SET_ID" \
  --region ap-northeast-1

echo -e "\n=== Testing UpdateTaskSet API ==="
aws ecs update-task-set \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --service test-service \
  --task-set "$TASK_SET_ID" \
  --scale unit=PERCENT,value=75 \
  --region ap-northeast-1

echo -e "\n=== Testing DeleteTaskSet API ==="
aws ecs delete-task-set \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --service test-service \
  --task-set "$TASK_SET_ID" \
  --region ap-northeast-1

echo -e "\n=== Final DescribeTaskSets to verify deletion status ==="
aws ecs describe-task-sets \
  --endpoint-url $ENDPOINT \
  --cluster default \
  --service test-service \
  --region ap-northeast-1