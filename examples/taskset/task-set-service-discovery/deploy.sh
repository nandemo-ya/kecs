#!/bin/bash

# Deploy TaskSet with Service Discovery example

echo "=== TaskSet Service Discovery Integration Example ==="
echo

# Configuration
CLUSTER_NAME=${CLUSTER_NAME:-default}
SERVICE_NAME="webapp-sd-service"
TASKSET_ID="ts-sd-$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -c1-8)"
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

# Step 3: Create TaskSet with Service Discovery configuration
echo "3. Creating TaskSet with Service Discovery..."
cat > taskset_request.json <<EOF
{
  "cluster": "$CLUSTER_NAME",
  "service": "$SERVICE_NAME",
  "taskDefinition": "webapp-sd:1",
  "scale": {
    "value": 2.0,
    "unit": "COUNT"
  },
  "serviceRegistries": [
    {
      "registryArn": "arn:aws:servicediscovery:us-east-1:000000000000:service/srv-webapp-sd",
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

echo "Endpoints:"
kubectl get endpoints -n $CLUSTER_NAME-us-east-1 | grep $SERVICE_NAME || true

echo "Pods:"
kubectl get pods -n $CLUSTER_NAME-us-east-1 | grep $SERVICE_NAME || true

# Step 7: Check Service Discovery annotations
echo "7. Checking Service Discovery configuration..."
DEPLOYMENT_NAME=$(echo "$SERVICE_NAME-ts" | tr '[:upper:]' '[:lower:]')
kubectl describe deployment -n $CLUSTER_NAME-us-east-1 $DEPLOYMENT_NAME 2>/dev/null | grep -E "Annotations:|service-discovery|service-registries" || true

echo
echo "8. Checking pod annotations for Service Discovery..."
POD_NAME=$(kubectl get pods -n $CLUSTER_NAME-us-east-1 | grep $SERVICE_NAME | head -1 | awk '{print $1}')
if [ ! -z "$POD_NAME" ]; then
    kubectl describe pod -n $CLUSTER_NAME-us-east-1 $POD_NAME | grep -E "Annotations:|kecs.io/sd-" || true
fi

# Clean up temporary files
rm -f taskset_request.json taskset_response.json

echo
echo "=== TaskSet Service Discovery Integration Example Complete ==="
echo "Note: Service Discovery endpoints are created and can be used for service-to-service communication"
echo "Pods are annotated with Service Discovery registry information"