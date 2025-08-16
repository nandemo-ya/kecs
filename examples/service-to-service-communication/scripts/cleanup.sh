#!/bin/bash

# Cleanup script for Service-to-Service Communication Demo

set -e

# Configuration
CLUSTER_NAME="default"
NAMESPACE_NAME="production.local"
KECS_ENDPOINT=${KECS_ENDPOINT:-"http://localhost:8080"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🧹 Cleaning up Service-to-Service Communication Demo${NC}"

# Set AWS endpoint for KECS
export AWS_ENDPOINT_URL=$KECS_ENDPOINT

# Step 1: Delete ECS services
echo -e "\n${YELLOW}Deleting ECS services...${NC}"
aws ecs delete-service --cluster $CLUSTER_NAME --service backend-api-service --force 2>/dev/null || echo "Backend service not found"
aws ecs delete-service --cluster $CLUSTER_NAME --service frontend-web-service --force 2>/dev/null || echo "Frontend service not found"

# Wait for services to be deleted
echo "Waiting for services to be deleted..."
sleep 5

# Step 2: Get namespace ID
NAMESPACE_ID=$(aws servicediscovery list-namespaces \
    --query "Namespaces[?Name=='$NAMESPACE_NAME'].Id" \
    --output text 2>/dev/null || echo "")

if [ -n "$NAMESPACE_ID" ]; then
    # Step 3: Get service IDs
    echo -e "\n${YELLOW}Finding Service Discovery services...${NC}"
    
    BACKEND_SERVICE_ID=$(aws servicediscovery list-services \
        --filters Name=NAMESPACE_ID,Values=$NAMESPACE_ID \
        --query "Services[?Name=='backend-api'].Id" \
        --output text 2>/dev/null || echo "")
    
    FRONTEND_SERVICE_ID=$(aws servicediscovery list-services \
        --filters Name=NAMESPACE_ID,Values=$NAMESPACE_ID \
        --query "Services[?Name=='frontend-web'].Id" \
        --output text 2>/dev/null || echo "")
    
    # Step 4: Delete Service Discovery services
    echo -e "\n${YELLOW}Deleting Service Discovery services...${NC}"
    
    if [ -n "$BACKEND_SERVICE_ID" ]; then
        aws servicediscovery delete-service --id $BACKEND_SERVICE_ID 2>/dev/null || echo "Backend discovery service not found"
    fi
    
    if [ -n "$FRONTEND_SERVICE_ID" ]; then
        aws servicediscovery delete-service --id $FRONTEND_SERVICE_ID 2>/dev/null || echo "Frontend discovery service not found"
    fi
    
    # Wait for services to be deleted
    sleep 5
    
    # Step 5: Delete namespace
    echo -e "\n${YELLOW}Deleting Service Discovery namespace...${NC}"
    aws servicediscovery delete-namespace --id $NAMESPACE_ID 2>/dev/null || echo "Namespace not found"
fi

# Step 6: Deregister task definitions
echo -e "\n${YELLOW}Deregistering task definitions...${NC}"
aws ecs deregister-task-definition --task-definition backend-api:1 2>/dev/null || echo "Backend task definition not found"
aws ecs deregister-task-definition --task-definition frontend-web:1 2>/dev/null || echo "Frontend task definition not found"

echo -e "\n${GREEN}✅ Cleanup complete!${NC}"