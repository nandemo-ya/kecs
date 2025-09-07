#!/bin/bash

set -e

echo "Testing CloudWatch Logs integration with KECS"
echo "============================================"

# Configuration
CLUSTER_NAME=${CLUSTER_NAME:-"default"}
REGION=${REGION:-"us-east-1"}
AWS_ENDPOINT_URL=${AWS_ENDPOINT_URL:-"http://localhost:5373"}

# Create log group in LocalStack
echo "1. Creating CloudWatch log group in LocalStack..."
aws logs create-log-group \
  --log-group-name /ecs/nginx-app \
  --region $REGION \
  --endpoint-url $LOCALSTACK_ENDPOINT 2>/dev/null || echo "Log group already exists"

# Register task definition
echo "2. Registering task definition with CloudWatch logging..."
aws ecs register-task-definition \
  --cli-input-json file://task_def_with_logs.json \
  --endpoint-url $AWS_ENDPOINT_URL \
  --region $REGION

# Run task
echo "3. Running task..."
TASK_ARN=$(aws ecs run-task \
  --cluster $CLUSTER_NAME \
  --task-definition nginx-with-logs:1 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}" \
  --endpoint-url $AWS_ENDPOINT_URL \
  --region $REGION \
  --query 'tasks[0].taskArn' \
  --output text)

echo "Task ARN: $TASK_ARN"

# Extract task ID from ARN
TASK_ID=$(echo $TASK_ARN | awk -F'/' '{print $NF}')
echo "Task ID: $TASK_ID"

# Wait for task to start
echo "4. Waiting for task to start..."
sleep 10

# Check pod in Kubernetes
echo "5. Checking Kubernetes pod..."
NAMESPACE="${CLUSTER_NAME}-${REGION}"
kubectl get pod -n $NAMESPACE ecs-task-$TASK_ID -o wide

# Check pod annotations for CloudWatch configuration
echo "6. Checking pod annotations for CloudWatch configuration..."
kubectl get pod -n $NAMESPACE ecs-task-$TASK_ID -o jsonpath='{.metadata.annotations}' | jq '.' | grep -E 'kecs.dev/container.*logs' || true

# Check Vector DaemonSet
echo "7. Checking Vector DaemonSet..."
kubectl get daemonset -n kecs-system vector -o wide 2>/dev/null || echo "Vector DaemonSet not found in kecs-system"

# Check Vector ConfigMap
echo "8. Checking Vector ConfigMap..."
kubectl get configmap -n kecs-system vector-config -o wide 2>/dev/null || echo "Vector ConfigMap not found in kecs-system"

# Generate some logs from nginx
echo "9. Generating logs from nginx container..."
kubectl exec -n $NAMESPACE ecs-task-$TASK_ID -c nginx -- sh -c "echo 'Test log message from nginx' && nginx -t"

# Wait a bit for logs to be collected
sleep 5

# Check CloudWatch logs in LocalStack
echo "10. Checking CloudWatch logs in LocalStack..."
aws logs describe-log-streams \
  --log-group-name /ecs/nginx-app \
  --region $REGION \
  --endpoint-url $LOCALSTACK_ENDPOINT \
  --query 'logStreams[*].[logStreamName,lastEventTimestamp]' \
  --output table

# Try to get log events
echo "11. Attempting to retrieve log events..."
LOG_STREAM=$(aws logs describe-log-streams \
  --log-group-name /ecs/nginx-app \
  --region $REGION \
  --endpoint-url $LOCALSTACK_ENDPOINT \
  --query 'logStreams[0].logStreamName' \
  --output text)

if [ "$LOG_STREAM" != "None" ] && [ ! -z "$LOG_STREAM" ]; then
  echo "Fetching logs from stream: $LOG_STREAM"
  aws logs get-log-events \
    --log-group-name /ecs/nginx-app \
    --log-stream-name "$LOG_STREAM" \
    --region $REGION \
    --endpoint-url $LOCALSTACK_ENDPOINT \
    --query 'events[*].[timestamp,message]' \
    --output table
else
  echo "No log streams found yet"
fi

# Cleanup
echo ""
echo "Cleanup:"
echo "To stop the task: aws ecs stop-task --cluster $CLUSTER_NAME --task $TASK_ARN --endpoint-url $AWS_ENDPOINT_URL --region $REGION"
echo "To delete the pod: kubectl delete pod -n $NAMESPACE ecs-task-$TASK_ID"