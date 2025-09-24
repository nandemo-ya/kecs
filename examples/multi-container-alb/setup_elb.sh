#!/bin/bash

# ELB setup script for Multi-Container ALB example
# This script sets up Application Load Balancer and Target Group for the service

set -e

# Configuration
ENDPOINT_URL=${AWS_ENDPOINT_URL:-http://localhost:5373}
ALB_NAME="multi-container-alb-alb"
TG_NAME="multi-container-alb-tg"

echo "=== Setting up Application Load Balancer for Multi-Container ALB Example ==="
echo "Endpoint: $ENDPOINT_URL"

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
  --tags Key=Application,Value=multi-container-alb Key=Environment,Value=development \
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
  --tags Key=Application,Value=multi-container-alb Key=Environment,Value=development \
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

echo ""
echo "=== ELB Setup Complete ==="
echo ""
echo "Load Balancer Details:"
echo "  ALB: $ALB_NAME"
echo "  DNS: $ALB_DNS"
echo "  Target Group: $TG_NAME"
echo ""
echo "Next Steps:"
echo "  1. Run ./deploy.sh to deploy the service with ELB"
echo "  2. Port forward to Traefik to test:"
echo "     kubectl port-forward -n kecs-system svc/traefik 8888:80"
echo ""
echo "  3. Test endpoints:"
echo "     curl -H 'Host: $ALB_DNS' http://localhost:8888/"
echo "     curl -H 'Host: $ALB_DNS' http://localhost:8888/api/status"
echo "     curl -H 'Host: $ALB_DNS' http://localhost:8888/health"
echo ""
echo "To clean up all resources, run: ./cleanup_all.sh"