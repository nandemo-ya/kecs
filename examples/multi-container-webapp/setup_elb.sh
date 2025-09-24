#!/bin/bash

# ELB setup script for Multi-Container WebApp
# This script configures Application Load Balancer for an existing ECS service

set -e

# Configuration
ENDPOINT_URL=${AWS_ENDPOINT_URL:-http://localhost:5373}
ALB_NAME="multi-container-webapp-alb"
TG_NAME="multi-container-webapp-tg"
CLUSTER_NAME="default"
SERVICE_NAME="multi-container-webapp"
TASK_DEF_NAME="multi-container-webapp"

echo "=== Setting up Application Load Balancer for Multi-Container WebApp ==="
echo "Endpoint: $ENDPOINT_URL"

# Check if service exists
echo ""
echo "Checking if service exists..."
SERVICE_STATUS=$(aws ecs describe-services \
  --cluster $CLUSTER_NAME \
  --services $SERVICE_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'services[0].status' --output text 2>/dev/null || echo "")

if [ "$SERVICE_STATUS" != "ACTIVE" ]; then
  echo "Error: Service $SERVICE_NAME not found in cluster $CLUSTER_NAME"
  echo "Please run ./deploy.sh first to create the service"
  exit 1
fi

# Get latest task definition revision
TASK_DEF_ARN=$(aws ecs describe-services \
  --cluster $CLUSTER_NAME \
  --services $SERVICE_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'services[0].taskDefinition' --output text)
TASK_DEF_REVISION=$(echo $TASK_DEF_ARN | sed 's/.*://')
echo "Using Task Definition: $TASK_DEF_NAME:$TASK_DEF_REVISION"

# Step 1: Setup ELB resources
echo ""
echo "Step 1: Creating Application Load Balancer..."

# KECS doesn't require VPC - use dummy VPC ID
VPC_ID="vpc-12345678"
echo "Using default VPC ID: $VPC_ID"

# Create ALB
ALB_ARN=$(aws elbv2 create-load-balancer \
  --name $ALB_NAME \
  --subnets subnet-12345678 subnet-87654321 \
  --security-groups sg-webapp \
  --scheme internet-facing \
  --type application \
  --ip-address-type ipv4 \
  --tags Key=Application,Value=$TASK_DEF_NAME Key=Environment,Value=development \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'LoadBalancers[0].LoadBalancerArn' --output text 2>/dev/null) || \
ALB_ARN=$(aws elbv2 describe-load-balancers \
  --names $ALB_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'LoadBalancers[0].LoadBalancerArn' --output text)

echo "ALB ARN: $ALB_ARN"

# Get ALB DNS
ALB_DNS=$(aws elbv2 describe-load-balancers \
  --load-balancer-arns $ALB_ARN \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'LoadBalancers[0].DNSName' --output text)
echo "ALB DNS: $ALB_DNS"

# Step 2: Create Target Group
echo ""
echo "Step 2: Creating Target Group..."
TG_ARN=$(aws elbv2 create-target-group \
  --name $TG_NAME \
  --protocol HTTP \
  --port 80 \
  --vpc-id $VPC_ID \
  --target-type ip \
  --health-check-enabled \
  --health-check-path / \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3 \
  --matcher 'HttpCode="200,301,302,404"' \
  --tags Key=Application,Value=$TASK_DEF_NAME Key=Environment,Value=development \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'TargetGroups[0].TargetGroupArn' --output text 2>/dev/null) || \
TG_ARN=$(aws elbv2 describe-target-groups \
  --names $TG_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'TargetGroups[0].TargetGroupArn' --output text)

echo "Target Group ARN: $TG_ARN"

# Step 3: Create Listener
echo ""
echo "Step 3: Creating HTTP Listener..."
LISTENER_ARN=$(aws elbv2 create-listener \
  --load-balancer-arn $ALB_ARN \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=forward,TargetGroupArn=$TG_ARN \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'Listeners[0].ListenerArn' --output text 2>/dev/null) || \
LISTENER_ARN=$(aws elbv2 describe-listeners \
  --load-balancer-arn $ALB_ARN \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'Listeners[?Port==`80`].ListenerArn' --output text | head -n1)

echo "Listener ARN: $LISTENER_ARN"

# Step 4: Create path-based routing rules
echo ""
echo "Step 4: Creating routing rules..."

# Check if rules already exist before creating
EXISTING_RULES=$(aws elbv2 describe-rules \
  --listener-arn $LISTENER_ARN \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'Rules[?Priority!=`default`].Priority' --output text 2>/dev/null || echo "")

if [ -z "$EXISTING_RULES" ]; then
  # Rule for API endpoints
  aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 1 \
    --conditions Field=path-pattern,Values="/api/*" \
    --actions Type=forward,TargetGroupArn=$TG_ARN \
    --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "API rule already exists"

  # Rule for static assets
  aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 2 \
    --conditions Field=path-pattern,Values="/static/*" \
    --actions Type=forward,TargetGroupArn=$TG_ARN \
    --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Static rule already exists"

  # Rule for health checks
  aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 3 \
    --conditions Field=path-pattern,Values="/health" \
    --actions Type=forward,TargetGroupArn=$TG_ARN \
    --region us-east-1 \
  --endpoint-url $ENDPOINT_URL 2>/dev/null || echo "Health rule already exists"
else
  echo "Routing rules already configured"
fi

# Step 5: Update ECS service with load balancer
echo ""
echo "Step 5: Updating ECS service to use load balancer..."

# Note: AWS ECS doesn't allow adding load balancer to existing service via update-service
# We need to recreate the service with load balancer configuration
echo "Note: To attach a load balancer, the service needs to be recreated."
echo "This will cause a brief downtime."

# Delete existing service
echo "Deleting existing service..."
aws ecs delete-service \
  --cluster $CLUSTER_NAME \
  --service $SERVICE_NAME \
  --force \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL > /dev/null 2>&1

# Wait for service deletion
echo "Waiting for service deletion..."
sleep 10

# Create new service with load balancer
echo "Creating service with load balancer configuration..."
cat > service_def_with_elb.json <<EOF
{
  "serviceName": "$SERVICE_NAME",
  "cluster": "$CLUSTER_NAME",
  "taskDefinition": "$TASK_DEF_NAME:$TASK_DEF_REVISION",
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
      "value": "$TASK_DEF_NAME"
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

aws ecs create-service \
  --cli-input-json file://service_def_with_elb.json \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --output table

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

# Check target health
echo "Checking target health..."
aws elbv2 describe-target-health \
  --target-group-arn $TG_ARN \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'TargetHealthDescriptions[*].{Target:Target.Id,Health:TargetHealth.State}' \
  --output table

# Get task details
echo ""
echo "Running tasks:"
aws ecs list-tasks \
  --cluster $CLUSTER_NAME \
  --service-name $SERVICE_NAME \
  --region us-east-1 \
  --endpoint-url $ENDPOINT_URL \
  --query 'taskArns' --output json | jq -r '.[]'

echo ""
echo "=== ELB Setup Complete ==="
echo ""
echo "Service Details:"
echo "  Cluster: $CLUSTER_NAME"
echo "  Service: $SERVICE_NAME"
echo "  Task Definition: $TASK_DEF_NAME"
echo "  Desired Count: 3"
echo ""
echo "Load Balancer Details:"
echo "  ALB: $ALB_NAME"
echo "  DNS: $ALB_DNS"
echo "  Target Group: $TG_NAME"
echo ""
echo "Testing Instructions:"
echo "  1. Port forward to Traefik:"
echo "     kubectl port-forward -n kecs-system svc/traefik 8888:80"
echo ""
echo "  2. Test endpoints:"
echo "     curl -H 'Host: $ALB_DNS' http://localhost:8888/"
echo "     curl -H 'Host: $ALB_DNS' http://localhost:8888/api/status"
echo "     curl -H 'Host: $ALB_DNS' http://localhost:8888/health"
echo ""
echo "To clean up all resources, run: ./cleanup_all.sh"