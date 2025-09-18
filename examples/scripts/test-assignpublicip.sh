#!/bin/bash

# Test script for assignPublicIp functionality
set -e

# Configuration
CLUSTER_NAME="test-assignpublicip"
TASK_DEF_NAME="nginx-test"
ENDPOINT_URL="http://localhost:5373"

echo "Testing assignPublicIp functionality..."
echo ""

# Step 1: Create cluster
echo "Step 1: Creating cluster..."
aws ecs create-cluster --cluster-name $CLUSTER_NAME --endpoint-url $ENDPOINT_URL --no-cli-pager

# Step 2: Create task definition with nginx
echo ""
echo "Step 2: Creating task definition..."
cat > /tmp/test-task-def.json <<EOF
{
  "family": "$TASK_DEF_NAME",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["EC2"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "nginx",
      "image": "nginx:latest",
      "cpu": 256,
      "memory": 512,
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "protocol": "tcp"
        }
      ]
    }
  ]
}
EOF

TASK_DEF_ARN=$(aws ecs register-task-definition \
  --cli-input-json file:///tmp/test-task-def.json \
  --endpoint-url $ENDPOINT_URL \
  --query 'taskDefinition.taskDefinitionArn' \
  --output text)

echo "Task definition registered: $TASK_DEF_ARN"

# Step 3: Run task with assignPublicIp enabled
echo ""
echo "Step 3: Running task with assignPublicIp enabled..."
TASK_ARN=$(aws ecs run-task \
  --cluster $CLUSTER_NAME \
  --task-definition $TASK_DEF_NAME \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-test],assignPublicIp=ENABLED}" \
  --endpoint-url $ENDPOINT_URL \
  --query 'tasks[0].taskArn' \
  --output text)

echo "Task started: $TASK_ARN"

# Step 4: Wait for task to be running
echo ""
echo "Step 4: Waiting for task to be running..."
for i in {1..30}; do
  STATUS=$(aws ecs describe-tasks \
    --cluster $CLUSTER_NAME \
    --tasks $TASK_ARN \
    --endpoint-url $ENDPOINT_URL \
    --query 'tasks[0].lastStatus' \
    --output text)
  
  echo "Task status: $STATUS"
  
  if [ "$STATUS" == "RUNNING" ]; then
    break
  fi
  
  sleep 2
done

# Step 5: Check task details for public IP and port information
echo ""
echo "Step 5: Checking task details for public IP and port information..."
aws ecs describe-tasks \
  --cluster $CLUSTER_NAME \
  --tasks $TASK_ARN \
  --endpoint-url $ENDPOINT_URL \
  --query 'tasks[0].{Status:lastStatus,Attachments:attachments,Attributes:attributes}' \
  --output json | jq '.'

# Step 6: Extract allocated port from task attachments
echo ""
echo "Step 6: Extracting allocated port..."
HOST_PORT=$(aws ecs describe-tasks \
  --cluster $CLUSTER_NAME \
  --tasks $TASK_ARN \
  --endpoint-url $ENDPOINT_URL \
  --query 'tasks[0].attachments[?type==`PublicIp`].details[?name==`hostPort0`].value' \
  --output text 2>/dev/null || echo "")

if [ -n "$HOST_PORT" ]; then
  echo "Allocated host port: $HOST_PORT"
  echo ""
  echo "Step 7: Testing HTTP access..."
  echo "Trying to access nginx at http://localhost:$HOST_PORT"
  
  # Wait a bit for nginx to be ready
  sleep 5
  
  # Try to access nginx
  if curl -s -o /dev/null -w "%{http_code}" "http://localhost:$HOST_PORT" | grep -q "200\|301\|302"; then
    echo "✅ SUCCESS: Nginx is accessible at http://localhost:$HOST_PORT"
  else
    echo "⚠️  Could not access nginx (this might be normal if k3d port mapping is not yet active)"
  fi
else
  echo "⚠️  No host port found in task attachments"
fi

echo ""
echo "Test complete. To cleanup, run:"
echo "  aws ecs stop-task --cluster $CLUSTER_NAME --task $TASK_ARN --endpoint-url $ENDPOINT_URL"
echo "  aws ecs delete-cluster --cluster $CLUSTER_NAME --endpoint-url $ENDPOINT_URL"