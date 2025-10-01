#!/bin/bash

set -e

# Configuration
CLUSTER_NAME="${CLUSTER_NAME:-default}"
REGION="${AWS_REGION:-us-east-1}"
ENDPOINT_URL="${AWS_ENDPOINT_URL:-http://localhost:5373}"

echo "=================================================="
echo "LocalStack S3 Job Example"
echo "=================================================="
echo "Cluster: $CLUSTER_NAME"
echo "Region: $REGION"
echo "Endpoint: $ENDPOINT_URL"
echo "=================================================="
echo ""

# 1. Create cluster if not exists
echo "Step 1: Ensuring cluster exists..."
aws ecs create-cluster \
  --cluster-name "$CLUSTER_NAME" \
  --region "$REGION" \
  --endpoint-url "$ENDPOINT_URL" \
  2>/dev/null || echo "Cluster already exists"
echo ""

# 2. Create CloudWatch Log Group
echo "Step 2: Creating CloudWatch Log Group..."
aws logs create-log-group \
  --log-group-name /ecs/localstack-s3-job \
  --region "$REGION" \
  --endpoint-url "$ENDPOINT_URL" \
  2>/dev/null || echo "Log group already exists"
echo ""

# 3. Register task definition
echo "Step 3: Registering task definition..."
aws ecs register-task-definition \
  --cli-input-json file://task_def.json \
  --region "$REGION" \
  --endpoint-url "$ENDPOINT_URL"
echo ""

# 4. Run task
echo "Step 4: Running task..."
TASK_ARN=$(aws ecs run-task \
  --cluster "$CLUSTER_NAME" \
  --task-definition localstack-s3-job \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],assignPublicIp=ENABLED}" \
  --region "$REGION" \
  --endpoint-url "$ENDPOINT_URL" \
  --query 'tasks[0].taskArn' \
  --output text)

echo "Task started: $TASK_ARN"
echo ""

# 5. Wait for task to complete
echo "Step 5: Waiting for task to complete..."
for i in {1..60}; do
  STATUS=$(aws ecs describe-tasks \
    --cluster "$CLUSTER_NAME" \
    --tasks "$TASK_ARN" \
    --region "$REGION" \
    --endpoint-url "$ENDPOINT_URL" \
    --query 'tasks[0].lastStatus' \
    --output text)

  echo "Task status: $STATUS (attempt $i/60)"

  if [ "$STATUS" = "STOPPED" ]; then
    EXIT_CODE=$(aws ecs describe-tasks \
      --cluster "$CLUSTER_NAME" \
      --tasks "$TASK_ARN" \
      --region "$REGION" \
      --endpoint-url "$ENDPOINT_URL" \
      --query 'tasks[0].containers[0].exitCode' \
      --output text)

    echo "Task completed with exit code: $EXIT_CODE"
    break
  fi

  sleep 5
done
echo ""

# 6. Show logs
echo "Step 6: Retrieving task logs..."
aws logs tail /ecs/localstack-s3-job \
  --region "$REGION" \
  --endpoint-url "$ENDPOINT_URL" \
  --since 5m
echo ""

# 7. Verify S3 bucket contents
echo "Step 7: Verifying S3 bucket contents..."
echo "Listing objects in test-bucket:"
aws s3 ls s3://test-bucket/ \
  --endpoint-url "$ENDPOINT_URL" \
  --region "$REGION"
echo ""

echo "Downloading output file:"
aws s3 cp s3://test-bucket/output.txt - \
  --endpoint-url "$ENDPOINT_URL" \
  --region "$REGION"
echo ""

echo "=================================================="
echo "LocalStack S3 Job completed successfully!"
echo "=================================================="
