#!/bin/bash

# Deployment script for Multi-Container ALB example
# This script requires ELB to be set up first via setup_elb.sh

set -e

# Configuration
ENDPOINT_URL=${AWS_ENDPOINT_URL:-http://localhost:5373}
CLUSTER_NAME="default"
SERVICE_NAME="multi-container-alb"
TASK_DEF_NAME="multi-container-alb"

echo "=== Deploying Multi-Container ALB Example ==="
echo "Endpoint: $ENDPOINT_URL"

# Step 1: Create ECS cluster (if not exists)
echo ""
echo "Step 1: Creating ECS cluster..."
aws ecs create-cluster --cluster-name $CLUSTER_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Cluster already exists"

# Step 2: Create CloudWatch Log Group
echo ""
echo "Step 2: Creating CloudWatch Log Group..."
aws logs create-log-group \
  --log-group-name /ecs/$TASK_DEF_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Log group already exists"

# Step 3: Register task definition
echo ""
echo "Step 3: Registering task definition..."
TASK_DEF_ARN=$(aws ecs register-task-definition \
  --cli-input-json file://task_def.json \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'taskDefinition.taskDefinitionArn' --output text)
echo "Task Definition ARN: $TASK_DEF_ARN"

# Step 4: Check for existing Target Group (from setup_elb.sh)
echo ""
echo "Step 4: Checking for existing ELB configuration..."

TG_NAME="multi-container-alb-tg"
TG_ARN=$(aws elbv2 describe-target-groups \
  --names $TG_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'TargetGroups[0].TargetGroupArn' --output text 2>/dev/null || echo "")

if [ -z "$TG_ARN" ] || [ "$TG_ARN" == "None" ]; then
  echo "Error: Target Group not found!"
  echo "Please run ./setup_elb.sh first to create the load balancer infrastructure."
  exit 1
fi

echo "Found Target Group: $TG_ARN"
echo "Creating service with load balancer configuration"

  # Generate service definition with ELB
  cat > service_def_with_elb.json <<EOF
{
  "serviceName": "$SERVICE_NAME",
  "cluster": "$CLUSTER_NAME",
  "taskDefinition": "$TASK_DEF_NAME",
  "desiredCount": 3,
  "launchType": "FARGATE",
  "platformVersion": "LATEST",
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345678", "subnet-87654321"],
      "securityGroups": ["sg-webapp"],
      "assignPublicIp": "ENABLED"
    }
  },
  "loadBalancers": [
    {
      "targetGroupArn": "$TG_ARN",
      "containerName": "frontend-nginx",
      "containerPort": 80
    }
  ],
  "healthCheckGracePeriodSeconds": 60,
  "deploymentConfiguration": {
    "maximumPercent": 200,
    "minimumHealthyPercent": 100,
    "deploymentCircuitBreaker": {
      "enable": true,
      "rollback": true
    }
  },
  "placementStrategy": [
    {
      "type": "spread",
      "field": "attribute:ecs.availability-zone"
    }
  ],
  "enableECSManagedTags": true,
  "propagateTags": "TASK_DEFINITION",
  "tags": [
    {
      "key": "Environment",
      "value": "development"
    },
    {
      "key": "Application",
      "value": "multi-container-alb"
    },
    {
      "key": "Type",
      "value": "webapp"
    },
    {
      "key": "LoadBalanced",
      "value": "true"
    }
  ]
}
EOF
SERVICE_DEF_FILE="service_def_with_elb.json"
DESIRED_COUNT=3

# Step 5: Create or update ECS service
echo ""
echo "Step 5: Creating/Updating ECS service..."

# Check if service exists
SERVICE_STATUS=$(aws ecs describe-services \
  --cluster $CLUSTER_NAME \
  --services $SERVICE_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'services[0].status' --output text 2>/dev/null || echo "")

if [ "$SERVICE_STATUS" == "ACTIVE" ]; then
  echo "Service exists, updating..."
  aws ecs update-service \
    --cluster $CLUSTER_NAME \
    --service $SERVICE_NAME \
    --desired-count $DESIRED_COUNT \
    --task-definition $TASK_DEF_NAME \
    --region us-east-1 \
    --endpoint-url $ENDPOINT_URL \
    --output table
else
  echo "Creating new service..."
  aws ecs create-service \
    --cli-input-json file://$SERVICE_DEF_FILE \
    --region us-east-1 \
    --endpoint-url $ENDPOINT_URL \
    --output table
fi

# Step 6: Wait for service to stabilize
echo ""
echo "Step 6: Waiting for service to stabilize..."
echo "This may take a few minutes..."

# Function to check service status
check_service_status() {
  aws ecs describe-services \
    --cluster $CLUSTER_NAME \
    --services $SERVICE_NAME \
    --region us-east-1 \
    --endpoint-url $ENDPOINT_URL \
    --query 'services[0].{Desired:desiredCount,Running:runningCount,Pending:pendingCount}' \
    --output json
}

# Wait for tasks to start
MAX_WAIT=60
WAIT_COUNT=0
while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
  STATUS=$(check_service_status)
  RUNNING=$(echo $STATUS | jq -r '.Running')
  DESIRED=$(echo $STATUS | jq -r '.Desired')

  echo "Tasks: Running=$RUNNING, Desired=$DESIRED"

  if [ "$RUNNING" == "$DESIRED" ]; then
    echo "Service is stable!"
    break
  fi

  sleep 5
  WAIT_COUNT=$((WAIT_COUNT + 1))
done

# Step 7: Verify deployment
echo ""
echo "Step 7: Verifying deployment..."

# Get task details
echo "Running tasks:"
aws ecs list-tasks \
  --cluster $CLUSTER_NAME \
  --service-name $SERVICE_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'taskArns' --output json | jq -r '.[]'

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Service Details:"
echo "  Cluster: $CLUSTER_NAME"
echo "  Service: $SERVICE_NAME"
echo "  Task Definition: $TASK_DEF_NAME"
echo "  Desired Count: $DESIRED_COUNT"
echo "  Load Balancer: Configured"
echo "  Target Group: $TG_NAME"
echo ""
echo "To clean up all resources, run: ./cleanup_all.sh"