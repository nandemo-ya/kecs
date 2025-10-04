#!/bin/bash

# Test script for service-to-service communication

set -e

# Configuration
NAMESPACE_NAME="demo.local"
KECS_ENDPOINT=${KECS_ENDPOINT:-"http://localhost:5373"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 Testing Service-to-Service Communication${NC}"

# Set AWS endpoint for KECS
export AWS_ENDPOINT_URL=$KECS_ENDPOINT

# Function to test endpoint
test_endpoint() {
    local url=$1
    local service=$2
    
    echo -e "\n${YELLOW}Testing $service at $url${NC}"
    
    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" $url 2>/dev/null || echo "FAILED")
    
    if [[ "$response" == *"FAILED"* ]]; then
        echo -e "${RED}❌ Failed to connect to $service${NC}"
        return 1
    fi
    
    http_status=$(echo "$response" | grep "HTTP_STATUS" | cut -d: -f2)
    body=$(echo "$response" | sed '/HTTP_STATUS/d')
    
    if [ "$http_status" = "200" ]; then
        echo -e "${GREEN}✅ $service is responding (HTTP $http_status)${NC}"
        echo "Response: $body"
        return 0
    else
        echo -e "${RED}❌ $service returned HTTP $http_status${NC}"
        echo "Response: $body"
        return 1
    fi
}

# Step 1: Discover backend service instances
echo -e "\n${YELLOW}🔍 Discovering backend service instances...${NC}"
backend_instances=$(aws servicediscovery discover-instances \
    --namespace-name $NAMESPACE_NAME \
    --service-name backend-api \
    --query 'Instances[*].Attributes.AWS_INSTANCE_IPV4' \
    --output text 2>/dev/null || echo "")

if [ -z "$backend_instances" ]; then
    echo -e "${RED}❌ No backend instances found${NC}"
    echo "Make sure the backend service is deployed and registered with service discovery"
    exit 1
fi

echo "Found backend instances: $backend_instances"

# Step 2: Test backend health directly
for ip in $backend_instances; do
    test_endpoint "http://$ip:8080/health" "Backend health check"
    test_endpoint "http://$ip:8080/api/data" "Backend API"
done

# Step 3: Discover frontend service instances
echo -e "\n${YELLOW}🔍 Discovering frontend service instances...${NC}"
frontend_instances=$(aws servicediscovery discover-instances \
    --namespace-name $NAMESPACE_NAME \
    --service-name frontend-web \
    --query 'Instances[*].Attributes.AWS_INSTANCE_IPV4' \
    --output text 2>/dev/null || echo "")

if [ -z "$frontend_instances" ]; then
    echo -e "${RED}❌ No frontend instances found${NC}"
    echo "Make sure the frontend service is deployed and registered with service discovery"
    exit 1
fi

echo "Found frontend instances: $frontend_instances"

# Step 4: Test frontend health
for ip in $frontend_instances; do
    test_endpoint "http://$ip:3000/health" "Frontend health check"
done

# Step 5: Test DNS resolution (if inside the cluster)
echo -e "\n${YELLOW}🌐 Testing DNS Resolution (requires cluster access)...${NC}"
echo "DNS names configured:"
echo "  - backend-api.$NAMESPACE_NAME:8080"
echo "  - frontend-web.$NAMESPACE_NAME:3000"

# Step 6: Test service-to-service communication
echo -e "\n${YELLOW}🔄 Testing Service-to-Service Communication...${NC}"
for ip in $frontend_instances; do
    echo "Testing frontend ($ip) calling backend..."
    response=$(curl -s "http://$ip:3000/call-backend" 2>/dev/null || echo "FAILED")
    
    if [[ "$response" == *"Backend Response"* ]]; then
        echo -e "${GREEN}✅ Frontend successfully communicated with backend!${NC}"
    elif [[ "$response" == *"FAILED"* ]]; then
        echo -e "${RED}❌ Failed to connect to frontend${NC}"
    else
        echo -e "${YELLOW}⚠️  Frontend responded but backend communication may have failed${NC}"
        echo "Check the frontend logs for details"
    fi
done

echo -e "\n${GREEN}🎉 Communication test complete!${NC}"
echo ""
echo "To view the frontend UI:"
echo "  kubectl port-forward service/frontend-web-service 3000:3000"
echo "  Then open http://localhost:3000 in your browser"