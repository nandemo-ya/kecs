#!/bin/bash
# Test script for Traefik AWS API routing

set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Testing Traefik AWS API routing...${NC}"

# Default values
ENDPOINT="${ENDPOINT:-http://localhost:4566}"
REGION="${AWS_REGION:-us-east-1}"

# Test ECS API routing
echo -e "\n${YELLOW}1. Testing ECS API routing:${NC}"
aws ecs list-clusters --endpoint-url "$ENDPOINT" --region "$REGION" --no-cli-pager || {
    echo -e "${RED}ECS API routing test failed${NC}"
    exit 1
}
echo -e "${GREEN}✓ ECS API routing works${NC}"

# Test ELBv2 API routing
echo -e "\n${YELLOW}2. Testing ELBv2 API routing:${NC}"
aws elbv2 describe-load-balancers --endpoint-url "$ENDPOINT" --region "$REGION" --no-cli-pager || {
    echo -e "${RED}ELBv2 API routing test failed${NC}"
    exit 1
}
echo -e "${GREEN}✓ ELBv2 API routing works${NC}"

# Test LocalStack routing (S3)
echo -e "\n${YELLOW}3. Testing LocalStack routing (S3):${NC}"
aws s3 ls --endpoint-url "$ENDPOINT" --region "$REGION" --no-cli-pager || {
    echo -e "${RED}LocalStack routing test failed${NC}"
    exit 1
}
echo -e "${GREEN}✓ LocalStack routing works${NC}"

# Check Traefik dashboard
echo -e "\n${YELLOW}4. Checking Traefik dashboard:${NC}"
DASHBOARD_URL="http://localhost:8080/dashboard/"
if curl -s -o /dev/null -w "%{http_code}" "$DASHBOARD_URL" | grep -q "200"; then
    echo -e "${GREEN}✓ Traefik dashboard is accessible at $DASHBOARD_URL${NC}"
else
    echo -e "${YELLOW}⚠ Traefik dashboard not accessible (this is expected with k3d port forwarding)${NC}"
fi

# Show routing information
echo -e "\n${YELLOW}5. Routing information:${NC}"
echo "- ECS APIs (X-Amz-Target: AmazonEC2ContainerServiceV20141113.*) → KECS control plane"
echo "- ELBv2 APIs (X-Amz-Target: ElasticLoadBalancingV2.*) → KECS control plane"
echo "- Other AWS APIs → LocalStack"

echo -e "\n${GREEN}All routing tests passed!${NC}"