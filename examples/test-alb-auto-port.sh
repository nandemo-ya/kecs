#!/bin/bash

# Test script for automatic port mapping with ALB listener
# This script tests the new feature that automatically adds k3d port mappings
# when creating an ELBv2 listener

set -e

# Configuration
ENDPOINT_URL=${AWS_ENDPOINT_URL:-http://localhost:5373}
CLUSTER_NAME="alb-port-test-cluster"
ALB_NAME="test-auto-port-alb"
TG_NAME="test-auto-port-tg"

echo "=== Testing ALB Automatic Port Mapping ==="
echo "This test will create an ALB with listener and verify direct access without kubectl port-forward"
echo ""

# Step 1: Create ECS cluster
echo "Step 1: Creating ECS cluster..."
aws ecs create-cluster --cluster-name $CLUSTER_NAME \
  --endpoint-url $ENDPOINT_URL || echo "Cluster already exists"

# Step 2: Create ALB
echo ""
echo "Step 2: Creating Application Load Balancer..."
ALB_ARN=$(aws elbv2 create-load-balancer \
  --name $ALB_NAME \
  --subnets subnet-12345678 subnet-87654321 \
  --security-groups sg-test \
  --scheme internet-facing \
  --type application \
  --endpoint-url $ENDPOINT_URL \
  --query 'LoadBalancers[0].LoadBalancerArn' --output text 2>/dev/null) || \
ALB_ARN=$(aws elbv2 describe-load-balancers \
  --names $ALB_NAME \
  --endpoint-url $ENDPOINT_URL \
  --query 'LoadBalancers[0].LoadBalancerArn' --output text)

echo "ALB ARN: $ALB_ARN"

# Get ALB DNS
ALB_DNS=$(aws elbv2 describe-load-balancers \
  --load-balancer-arns $ALB_ARN \
  --endpoint-url $ENDPOINT_URL \
  --query 'LoadBalancers[0].DNSName' --output text)
echo "ALB DNS: $ALB_DNS"

# Step 3: Create Target Group
echo ""
echo "Step 3: Creating Target Group..."
TG_ARN=$(aws elbv2 create-target-group \
  --name $TG_NAME \
  --protocol HTTP \
  --port 80 \
  --vpc-id vpc-12345 \
  --target-type ip \
  --endpoint-url $ENDPOINT_URL \
  --query 'TargetGroups[0].TargetGroupArn' --output text 2>/dev/null) || \
TG_ARN=$(aws elbv2 describe-target-groups \
  --names $TG_NAME \
  --endpoint-url $ENDPOINT_URL \
  --query 'TargetGroups[0].TargetGroupArn' --output text)

echo "Target Group ARN: $TG_ARN"

# Step 4: Create Listener (THIS SHOULD TRIGGER PORT MAPPING!)
echo ""
echo "Step 4: Creating HTTP Listener on port 80..."
echo ">>> This should automatically add k3d port mapping 8080:30880 <<<"
LISTENER_ARN=$(aws elbv2 create-listener \
  --load-balancer-arn $ALB_ARN \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=forward,TargetGroupArn=$TG_ARN \
  --endpoint-url $ENDPOINT_URL \
  --query 'Listeners[0].ListenerArn' --output text)

echo "Listener ARN: $LISTENER_ARN"

# Step 5: Check if port mapping was added
echo ""
echo "Step 5: Checking k3d port mappings..."
echo "Looking for port mapping 8080->30880..."

# Check docker container ports
CONTAINER_NAME=$(docker ps --format "table {{.Names}}" | grep serverlb | head -1)
if [ -n "$CONTAINER_NAME" ]; then
  echo "Checking ports on container: $CONTAINER_NAME"
  docker port $CONTAINER_NAME | grep -E "8080|30880" || echo "Port mapping not found in docker port output"
  
  # Alternative check with docker inspect
  echo ""
  echo "Detailed port inspection:"
  docker inspect $CONTAINER_NAME --format '{{range $p, $conf := .NetworkSettings.Ports}}{{$p}} -> {{(index $conf 0).HostPort}}{{println}}{{end}}' | grep -E "8080|30880" || true
fi

# Step 6: Test direct access
echo ""
echo "Step 6: Testing direct access to ALB (without kubectl port-forward)..."
echo "Attempting to access: http://localhost:8080"
echo "Using Host header: $ALB_DNS"
echo ""

# First, check if Traefik is responding
echo "Testing if Traefik is accessible on port 8080..."
if curl -s -o /dev/null -w "%{http_code}" -H "Host: $ALB_DNS" http://localhost:8080/healthz 2>/dev/null | grep -q "404\|200\|301\|302"; then
  echo "✅ SUCCESS: ALB is directly accessible on localhost:8080!"
  echo "   No kubectl port-forward needed!"
else
  echo "❌ FAILED: Cannot access ALB on localhost:8080"
  echo "   Port mapping may not have been added successfully"
  echo ""
  echo "   Fallback: You can still use kubectl port-forward:"
  echo "   kubectl port-forward -n kecs-system svc/traefik 8888:80"
fi

# Step 7: Cleanup
echo ""
echo "Step 7: Cleanup (optional)"
echo "To clean up resources, run:"
echo "  aws elbv2 delete-listener --listener-arn $LISTENER_ARN --endpoint-url $ENDPOINT_URL"
echo "  aws elbv2 delete-target-group --target-group-arn $TG_ARN --endpoint-url $ENDPOINT_URL"
echo "  aws elbv2 delete-load-balancer --load-balancer-arn $ALB_ARN --endpoint-url $ENDPOINT_URL"
echo "  aws ecs delete-cluster --cluster $CLUSTER_NAME --endpoint-url $ENDPOINT_URL"

echo ""
echo "=== Test Complete ==="