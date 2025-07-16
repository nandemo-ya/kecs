#!/bin/bash
# End-to-end test script for KECS v2 architecture

set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
CLUSTER_NAME="${CLUSTER_NAME:-kecs-v2-test}"
TIMEOUT="${TIMEOUT:-10m}"
CLEANUP="${CLEANUP:-true}"

echo -e "${BLUE}=== KECS V2 Architecture End-to-End Test ===${NC}"
echo "Cluster name: $CLUSTER_NAME"
echo "Timeout: $TIMEOUT"
echo ""

# Cleanup function
cleanup() {
    if [ "$CLEANUP" = "true" ]; then
        echo -e "\n${YELLOW}Cleaning up...${NC}"
        kecs stop-v2 --name "$CLUSTER_NAME" --delete-data || true
    fi
}

# Set trap for cleanup
trap cleanup EXIT

# Step 1: Start KECS with v2 architecture
echo -e "${YELLOW}Step 1: Starting KECS with v2 architecture${NC}"
if ! kecs start-v2 --name "$CLUSTER_NAME" --timeout "$TIMEOUT"; then
    echo -e "${RED}Failed to start KECS${NC}"
    exit 1
fi
echo -e "${GREEN}✓ KECS started successfully${NC}\n"

# Step 2: Wait for components to be ready
echo -e "${YELLOW}Step 2: Waiting for components to be ready${NC}"
sleep 10  # Give components time to stabilize

# Step 3: Test admin health endpoint
echo -e "${YELLOW}Step 3: Testing admin health endpoint${NC}"
if curl -s -f http://localhost:8081/health > /dev/null; then
    echo -e "${GREEN}✓ Admin health endpoint is accessible${NC}"
else
    echo -e "${RED}✗ Admin health endpoint is not accessible${NC}"
    exit 1
fi

# Step 4: Test ECS API through Traefik
echo -e "\n${YELLOW}Step 4: Testing ECS API through Traefik${NC}"
echo "Creating test cluster..."
if aws ecs create-cluster --cluster-name test-cluster \
    --endpoint-url http://localhost:4566 \
    --region us-east-1 \
    --no-cli-pager > /dev/null; then
    echo -e "${GREEN}✓ ECS cluster created successfully${NC}"
else
    echo -e "${RED}✗ Failed to create ECS cluster${NC}"
    exit 1
fi

echo "Listing clusters..."
if aws ecs list-clusters \
    --endpoint-url http://localhost:4566 \
    --region us-east-1 \
    --no-cli-pager | grep -q "test-cluster"; then
    echo -e "${GREEN}✓ ECS cluster listed successfully${NC}"
else
    echo -e "${RED}✗ Failed to list ECS clusters${NC}"
    exit 1
fi

# Step 5: Test ELBv2 API through Traefik
echo -e "\n${YELLOW}Step 5: Testing ELBv2 API through Traefik${NC}"
if aws elbv2 describe-load-balancers \
    --endpoint-url http://localhost:4566 \
    --region us-east-1 \
    --no-cli-pager > /dev/null; then
    echo -e "${GREEN}✓ ELBv2 API is accessible${NC}"
else
    echo -e "${RED}✗ ELBv2 API is not accessible${NC}"
    exit 1
fi

# Step 6: Test LocalStack services through Traefik
echo -e "\n${YELLOW}Step 6: Testing LocalStack services through Traefik${NC}"
echo "Creating S3 bucket..."
BUCKET_NAME="test-bucket-$(date +%s)"
if aws s3 mb "s3://$BUCKET_NAME" \
    --endpoint-url http://localhost:4566 \
    --region us-east-1 > /dev/null; then
    echo -e "${GREEN}✓ S3 bucket created successfully${NC}"
else
    echo -e "${RED}✗ Failed to create S3 bucket${NC}"
    exit 1
fi

echo "Listing S3 buckets..."
if aws s3 ls \
    --endpoint-url http://localhost:4566 \
    --region us-east-1 \
    --no-cli-pager | grep -q "$BUCKET_NAME"; then
    echo -e "${GREEN}✓ S3 bucket listed successfully${NC}"
else
    echo -e "${RED}✗ Failed to list S3 buckets${NC}"
    exit 1
fi

# Step 7: Test task definition and task creation
echo -e "\n${YELLOW}Step 7: Testing task definition and task creation${NC}"
cat > /tmp/task-definition.json <<EOF
{
  "family": "test-task",
  "containerDefinitions": [
    {
      "name": "test-container",
      "image": "nginx:latest",
      "memory": 128,
      "essential": true
    }
  ]
}
EOF

echo "Registering task definition..."
if aws ecs register-task-definition \
    --cli-input-json file:///tmp/task-definition.json \
    --endpoint-url http://localhost:4566 \
    --region us-east-1 \
    --no-cli-pager > /dev/null; then
    echo -e "${GREEN}✓ Task definition registered successfully${NC}"
else
    echo -e "${RED}✗ Failed to register task definition${NC}"
    exit 1
fi

# Step 8: Check Kubernetes resources
echo -e "\n${YELLOW}Step 8: Checking Kubernetes resources${NC}"
export KUBECONFIG="$HOME/.k3d/kubeconfig-$CLUSTER_NAME.yaml"

echo "Checking kecs-system namespace..."
if kubectl get namespace kecs-system > /dev/null 2>&1; then
    echo -e "${GREEN}✓ kecs-system namespace exists${NC}"
else
    echo -e "${RED}✗ kecs-system namespace not found${NC}"
    exit 1
fi

echo "Checking deployments..."
kubectl get deployments -n kecs-system

echo "Checking services..."
kubectl get services -n kecs-system

# Summary
echo -e "\n${GREEN}=== All tests passed! ===${NC}"
echo -e "${GREEN}✓ KECS v2 architecture is working correctly${NC}"
echo ""
echo "To stop the cluster, run:"
echo "  kecs stop-v2 --name $CLUSTER_NAME"