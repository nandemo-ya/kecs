#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Testing ELBv2 Integration with LocalStack${NC}"

# Navigate to controlplane directory
cd "$(dirname "$0")/.."

# 1. Run unit tests
echo -e "\n${YELLOW}Running ELBv2 integration unit tests...${NC}"
go test ./internal/integrations/elbv2/... -v

# 2. Run integration tests with LocalStack (if available)
if command -v localstack &> /dev/null; then
    echo -e "\n${YELLOW}Starting LocalStack for integration tests...${NC}"
    
    # Check if LocalStack is running
    if ! curl -s http://localhost:4566/_localstack/health | grep -q "running"; then
        echo -e "${YELLOW}LocalStack not running. Starting LocalStack...${NC}"
        localstack start -d
        
        # Wait for LocalStack to be ready
        echo -e "${YELLOW}Waiting for LocalStack to be ready...${NC}"
        for i in {1..30}; do
            if curl -s http://localhost:4566/_localstack/health | grep -q "running"; then
                echo -e "${GREEN}LocalStack is ready!${NC}"
                break
            fi
            echo -n "."
            sleep 2
        done
    else
        echo -e "${GREEN}LocalStack is already running${NC}"
    fi
    
    # Create a test load balancer
    echo -e "\n${YELLOW}Creating test load balancer...${NC}"
    aws --endpoint-url=http://localhost:4566 elbv2 create-load-balancer \
        --name test-lb \
        --subnets subnet-12345 subnet-67890 \
        --region us-east-1 || echo "Load balancer might already exist"
    
    # Create a test target group
    echo -e "\n${YELLOW}Creating test target group...${NC}"
    aws --endpoint-url=http://localhost:4566 elbv2 create-target-group \
        --name test-tg \
        --protocol HTTP \
        --port 80 \
        --vpc-id vpc-12345 \
        --region us-east-1 || echo "Target group might already exist"
    
    # List load balancers
    echo -e "\n${YELLOW}Listing load balancers...${NC}"
    aws --endpoint-url=http://localhost:4566 elbv2 describe-load-balancers --region us-east-1
    
    # List target groups
    echo -e "\n${YELLOW}Listing target groups...${NC}"
    aws --endpoint-url=http://localhost:4566 elbv2 describe-target-groups --region us-east-1
else
    echo -e "${YELLOW}LocalStack not installed. Skipping LocalStack integration tests.${NC}"
fi

echo -e "\n${GREEN}ELBv2 Integration tests completed!${NC}"