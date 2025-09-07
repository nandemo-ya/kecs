#!/bin/bash

# Test script for TaskSet API with Kubernetes integration

set -e

KECS_ENDPOINT="${KECS_ENDPOINT:-http://localhost:5373}"
CLUSTER_NAME="${CLUSTER_NAME:-default}"
SERVICE_NAME="webapp-service"
REGION="us-east-1"

echo "Testing TaskSet API with Kubernetes integration..."
echo "KECS Endpoint: $KECS_ENDPOINT"
echo ""

# Function to call KECS API
call_kecs() {
    local action=$1
    local params=$2
    aws ecs "$action" \
        --endpoint-url "$KECS_ENDPOINT" \
        --region "$REGION" \
        --no-cli-pager \
        $params 2>&1
}

echo "1. Creating cluster..."
aws ecs create-cluster \
    --cluster-name "$CLUSTER_NAME" \
    --endpoint-url "$KECS_ENDPOINT" \
    --region "$REGION" \
    --no-cli-pager || echo "Cluster may already exist"
echo ""

echo "2. Registering Blue task definition..."
BLUE_TASK_DEF=$(call_kecs register-task-definition \
    --cli-input-json file://task_def_blue.json \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text)
echo "Blue Task Definition: $BLUE_TASK_DEF"
echo ""

echo "3. Registering Green task definition..."
GREEN_TASK_DEF=$(call_kecs register-task-definition \
    --cli-input-json file://task_def_green.json \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text)
echo "Green Task Definition: $GREEN_TASK_DEF"
echo ""

echo "4. Creating service..."
call_kecs create-service \
    --cluster "$CLUSTER_NAME" \
    --service-name "$SERVICE_NAME" \
    --desired-count 2 \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}" || echo "Service may already exist"
echo ""

echo "5. Creating Blue TaskSet..."
BLUE_TASKSET_RESPONSE=$(call_kecs create-task-set \
    --cluster "$CLUSTER_NAME" \
    --service "$SERVICE_NAME" \
    --task-definition "$BLUE_TASK_DEF" \
    --external-id "blue-deployment" \
    --launch-type FARGATE \
    --scale "value=100,unit=PERCENT" \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}")

BLUE_TASKSET_ID=$(echo "$BLUE_TASKSET_RESPONSE" | grep -o '"id": "[^"]*' | head -1 | cut -d'"' -f4)
echo "Blue TaskSet ID: $BLUE_TASKSET_ID"
echo ""

echo "6. Creating Green TaskSet..."
GREEN_TASKSET_RESPONSE=$(call_kecs create-task-set \
    --cluster "$CLUSTER_NAME" \
    --service "$SERVICE_NAME" \
    --task-definition "$GREEN_TASK_DEF" \
    --external-id "green-deployment" \
    --launch-type FARGATE \
    --scale "value=0,unit=PERCENT" \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}")

GREEN_TASKSET_ID=$(echo "$GREEN_TASKSET_RESPONSE" | grep -o '"id": "[^"]*' | head -1 | cut -d'"' -f4)
echo "Green TaskSet ID: $GREEN_TASKSET_ID"
echo ""

echo "7. Describing TaskSets..."
call_kecs describe-task-sets \
    --cluster "$CLUSTER_NAME" \
    --service "$SERVICE_NAME" \
    --query 'taskSets[*].[id,externalId,status,stabilityStatus,runningCount,pendingCount]' \
    --output table
echo ""

echo "8. Checking Kubernetes resources..."
echo "Deployments:"
kubectl get deployments -n "$CLUSTER_NAME-$REGION" -l "kecs.io/service=$SERVICE_NAME" 2>/dev/null || echo "No deployments found"
echo ""

echo "Services:"
kubectl get services -n "$CLUSTER_NAME-$REGION" -l "kecs.io/service=$SERVICE_NAME" 2>/dev/null || echo "No services found"
echo ""

echo "9. Updating Green TaskSet to 50%..."
call_kecs update-task-set \
    --cluster "$CLUSTER_NAME" \
    --service "$SERVICE_NAME" \
    --task-set "$GREEN_TASKSET_ID" \
    --scale "value=50,unit=PERCENT"
echo ""

sleep 2

echo "10. Checking TaskSet status after update..."
call_kecs describe-task-sets \
    --cluster "$CLUSTER_NAME" \
    --service "$SERVICE_NAME" \
    --task-sets "$GREEN_TASKSET_ID" \
    --query 'taskSets[0].[id,externalId,scale.value,stabilityStatus,runningCount,pendingCount]' \
    --output json
echo ""

echo "11. Deleting Green TaskSet..."
call_kecs delete-task-set \
    --cluster "$CLUSTER_NAME" \
    --service "$SERVICE_NAME" \
    --task-set "$GREEN_TASKSET_ID"
echo ""

echo "12. Deleting Blue TaskSet..."
call_kecs delete-task-set \
    --cluster "$CLUSTER_NAME" \
    --service "$SERVICE_NAME" \
    --task-set "$BLUE_TASKSET_ID"
echo ""

echo "13. Deleting service..."
call_kecs delete-service \
    --cluster "$CLUSTER_NAME" \
    --service "$SERVICE_NAME" \
    --force || echo "Service deletion may have failed"
echo ""

echo "Test completed!"