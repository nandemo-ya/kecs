#!/bin/bash

# Complete cleanup script for Multi-Container ALB example
# This script removes ALL resources created by the deployment

set -e

# Configuration
ENDPOINT_URL=${AWS_ENDPOINT_URL:-http://localhost:5373}
ALB_NAME="multi-container-alb-alb"
TG_NAME="multi-container-alb-tg"
CLUSTER_NAME="default"
SERVICE_NAME="multi-container-alb"
TASK_DEF_NAME="multi-container-alb"
LOG_GROUP_NAME="/ecs/multi-container-alb"

echo "=== Cleaning up ALL Multi-Container ALB Resources ==="
echo "Endpoint: $ENDPOINT_URL"

# Step 1: Delete ECS Service
echo ""
echo "Step 1: Deleting ECS service..."
SERVICE_EXISTS=$(aws ecs describe-services \
  --cluster $CLUSTER_NAME \
  --services $SERVICE_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'services[0].status' --output text 2>/dev/null || echo "")

if [ -n "$SERVICE_EXISTS" ] && [ "$SERVICE_EXISTS" != "None" ] && [ "$SERVICE_EXISTS" != "INACTIVE" ]; then
  echo "Deleting service: $SERVICE_NAME"
  aws ecs delete-service \
    --cluster $CLUSTER_NAME \
    --service $SERVICE_NAME \
    --force \
    --region us-east-1 \
  --endpoint-url $ENDPOINT_URL
  
  echo "Waiting for service deletion..."
  aws ecs wait services-inactive \
    --cluster $CLUSTER_NAME \
    --services $SERVICE_NAME \
    --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Service deleted"
else
  echo "Service not found or already inactive: $SERVICE_NAME"
fi

# Step 2: Stop all running tasks in cluster
echo ""
echo "Step 2: Stopping all running tasks..."
TASK_ARNS=$(aws ecs list-tasks \
  --cluster $CLUSTER_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'taskArns' --output json 2>/dev/null | jq -r '.[]' || echo "")

if [ -n "$TASK_ARNS" ]; then
  for TASK_ARN in $TASK_ARNS; do
    echo "Stopping task: $TASK_ARN"
    aws ecs stop-task \
      --cluster $CLUSTER_NAME \
      --task $TASK_ARN \
      --reason "Cleanup" \
      --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Task already stopped"
  done
else
  echo "No running tasks found"
fi

# Step 3: Delete ALB Resources
echo ""
echo "Step 3: Deleting Application Load Balancer resources..."

# Get ALB ARN
ALB_ARN=$(aws elbv2 describe-load-balancers \
  --names $ALB_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'LoadBalancers[0].LoadBalancerArn' --output text 2>/dev/null || echo "")

if [ -n "$ALB_ARN" ] && [ "$ALB_ARN" != "None" ]; then
  echo "Found ALB: $ALB_ARN"
  
  # Get and delete listeners
  LISTENER_ARNS=$(aws elbv2 describe-listeners \
    --load-balancer-arn $ALB_ARN \
    --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
    --query 'Listeners[*].ListenerArn' --output text 2>/dev/null || echo "")
  
  if [ -n "$LISTENER_ARNS" ]; then
    for LISTENER_ARN in $LISTENER_ARNS; do
      # Delete listener rules first (except default)
      echo "Deleting rules for listener: $LISTENER_ARN"
      RULE_ARNS=$(aws elbv2 describe-rules \
        --listener-arn $LISTENER_ARN \
        --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
        --query 'Rules[?Priority!=`default`].RuleArn' --output text 2>/dev/null || echo "")
      
      if [ -n "$RULE_ARNS" ]; then
        for RULE_ARN in $RULE_ARNS; do
          aws elbv2 delete-rule \
            --rule-arn $RULE_ARN \
            --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Rule already deleted"
        done
      fi
      
      # Delete listener
      echo "Deleting listener: $LISTENER_ARN"
      aws elbv2 delete-listener \
        --listener-arn $LISTENER_ARN \
        --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Listener already deleted"
    done
  fi
  
  # Delete Load Balancer
  echo "Deleting Application Load Balancer..."
  aws elbv2 delete-load-balancer \
    --load-balancer-arn $ALB_ARN \
    --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "ALB already deleted"
  
  # Wait for ALB deletion
  echo "Waiting for ALB deletion to complete..."
  sleep 5
else
  echo "Load Balancer not found: $ALB_NAME"
fi

# Step 4: Delete Target Group
echo ""
echo "Step 4: Deleting Target Group..."
TG_ARN=$(aws elbv2 describe-target-groups \
  --names $TG_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'TargetGroups[0].TargetGroupArn' --output text 2>/dev/null || echo "")

if [ -n "$TG_ARN" ] && [ "$TG_ARN" != "None" ]; then
  echo "Deleting Target Group: $TG_ARN"
  aws elbv2 delete-target-group \
    --target-group-arn $TG_ARN \
    --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Target Group already deleted"
else
  echo "Target Group not found: $TG_NAME"
fi

# Step 5: Deregister Task Definitions
echo ""
echo "Step 5: Deregistering task definitions..."
TASK_DEF_ARNS=$(aws ecs list-task-definitions \
  --family-prefix $TASK_DEF_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'taskDefinitionArns' --output json 2>/dev/null | jq -r '.[]' || echo "")

if [ -n "$TASK_DEF_ARNS" ]; then
  for TASK_DEF_ARN in $TASK_DEF_ARNS; do
    echo "Deregistering: $TASK_DEF_ARN"
    aws ecs deregister-task-definition \
      --task-definition $TASK_DEF_ARN \
      --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Already deregistered"
  done
else
  echo "No task definitions found"
fi

# Step 6: Delete CloudWatch Log Group
echo ""
echo "Step 6: Deleting CloudWatch Log Group..."
aws logs delete-log-group \
  --log-group-name $LOG_GROUP_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Log group not found or already deleted"

# Step 7: Delete ECS Cluster
echo ""
echo "Step 7: Deleting ECS cluster..."
CLUSTER_EXISTS=$(aws ecs describe-clusters \
  --clusters $CLUSTER_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'clusters[0].status' --output text 2>/dev/null || echo "")

if [ "$CLUSTER_EXISTS" == "ACTIVE" ]; then
  echo "Deleting cluster: $CLUSTER_NAME"
  aws ecs delete-cluster \
    --cluster $CLUSTER_NAME \
    --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Cluster already deleted"
else
  echo "Cluster not found or already inactive: $CLUSTER_NAME"
fi

# Step 8: Clean up generated files
echo ""
echo "Step 8: Cleaning up generated files..."
if [ -f "service_def_with_elb.json" ]; then
  echo "Removing service_def_with_elb.json"
  rm -f service_def_with_elb.json
fi

echo ""
echo "=== Cleanup Complete ==="
echo ""
echo "All resources have been removed:"
echo "✓ ECS Service: $SERVICE_NAME"
echo "✓ Running Tasks"
echo "✓ Application Load Balancer: $ALB_NAME"
echo "✓ Target Group: $TG_NAME"
echo "✓ Task Definitions: $TASK_DEF_NAME"
echo "✓ CloudWatch Log Group: $LOG_GROUP_NAME"
echo "✓ ECS Cluster: $CLUSTER_NAME"
echo "✓ Generated configuration files"
echo ""
echo "The environment is now clean and ready for a fresh deployment."