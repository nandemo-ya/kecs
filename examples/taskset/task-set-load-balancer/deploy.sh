#!/bin/bash

# Deploy TaskSet with Load Balancer example

echo "=== TaskSet Load Balancer Integration Example ==="
echo

# Configuration
CLUSTER_NAME=${CLUSTER_NAME:-default}
SERVICE_NAME="webapp-lb-service"
TASKSET_ID="ts-lb-$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -c1-8)"
ENDPOINT=${KECS_ENDPOINT:-http://localhost:8080}

# Step 1: Register task definition
echo "1. Registering task definition..."
aws ecs register-task-definition \
    --cli-input-json file://task_def.json \
    --endpoint-url $ENDPOINT \
    --region us-east-1

# Step 2: Create service with EXTERNAL deployment controller
echo "2. Creating service with EXTERNAL deployment controller..."
aws ecs create-service \
    --cluster $CLUSTER_NAME \
    --service-name $SERVICE_NAME \
    --deployment-controller '{"type": "EXTERNAL"}' \
    --desired-count 0 \
    --endpoint-url $ENDPOINT \
    --region us-east-1

# Step 3: Create TaskSet with load balancer configuration
echo "3. Creating TaskSet with load balancer..."
cat > taskset_request.json <<EOF
{
  "cluster": "$CLUSTER_NAME",
  "service": "$SERVICE_NAME",
  "taskDefinition": "webapp-lb:1",
  "scale": {
    "value": 3.0,
    "unit": "COUNT"
  },
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:000000000000:targetgroup/webapp-lb-tg/1234567890abcdef",
      "containerName": "webapp",
      "containerPort": 80
    }
  ],
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345"],
      "securityGroups": ["sg-12345"],
      "assignPublicIp": "ENABLED"
    }
  },
  "launchType": "FARGATE",
  "platformVersion": "LATEST"
}
EOF

aws ecs create-task-set \
    --cli-input-json file://taskset_request.json \
    --endpoint-url $ENDPOINT \
    --region us-east-1 > taskset_response.json

# Extract TaskSet ID
TASKSET_ARN=$(jq -r '.taskSet.taskSetArn' taskset_response.json)
echo "Created TaskSet: $TASKSET_ARN"

# Step 4: Wait for TaskSet to stabilize
echo "4. Waiting for TaskSet to stabilize..."
sleep 5

# Step 5: Check TaskSet status
echo "5. Checking TaskSet status..."
aws ecs describe-task-sets \
    --cluster $CLUSTER_NAME \
    --service $SERVICE_NAME \
    --endpoint-url $ENDPOINT \
    --region us-east-1

# Step 6: Check Kubernetes resources
echo "6. Checking Kubernetes resources..."
echo "Deployments:"
kubectl get deployments -n $CLUSTER_NAME-us-east-1 | grep $SERVICE_NAME || true

echo "Services:"
kubectl get services -n $CLUSTER_NAME-us-east-1 | grep $SERVICE_NAME || true

echo "Pods:"
kubectl get pods -n $CLUSTER_NAME-us-east-1 | grep $SERVICE_NAME || true

# Step 7: Check service type
echo "7. Checking service configuration..."
SERVICE_NAME_K8S=$(echo "$SERVICE_NAME-ts" | tr '[:upper:]' '[:lower:]')
kubectl describe service -n $CLUSTER_NAME-us-east-1 $SERVICE_NAME_K8S 2>/dev/null | grep -E "Type:|LoadBalancer|Annotations:" || true

# Clean up temporary files
rm -f taskset_request.json taskset_response.json

echo
echo "=== TaskSet Load Balancer Integration Example Complete ==="
echo "Note: In a real environment, the LoadBalancer service would get an external IP"
echo "For local testing with k3d, you can use port-forward or NodePort to access the service"