#!/bin/bash

# Service-to-Service Communication Demo Deployment Script
# This script deploys frontend and backend services with ECS Service Discovery

set -e

# Configuration
CLUSTER_NAME="default"
NAMESPACE_NAME="production.local"
VPC_ID="vpc-default"
KECS_ENDPOINT=${KECS_ENDPOINT:-"http://localhost:8080"}
LOCALSTACK_ENDPOINT=${LOCALSTACK_ENDPOINT:-"http://localhost:4566"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Deploying Service-to-Service Communication Demo${NC}"
echo "KECS Endpoint: $KECS_ENDPOINT"
echo "LocalStack Endpoint: $LOCALSTACK_ENDPOINT"

# Function to check if command exists
check_command() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}❌ $1 is not installed${NC}"
        exit 1
    fi
}

# Check required commands
check_command aws
check_command docker

# Set AWS endpoint for KECS
export AWS_ENDPOINT_URL=$KECS_ENDPOINT

# Step 1: Build Docker images
echo -e "\n${YELLOW}📦 Building Docker images...${NC}"

echo "Building backend image..."
docker build -t backend-api:latest ./backend

echo "Building frontend image..."
docker build -t frontend-web:latest ./frontend

# Step 2: Create ECS cluster if not exists
echo -e "\n${YELLOW}☁️  Creating ECS cluster...${NC}"
aws ecs create-cluster --cluster-name $CLUSTER_NAME 2>/dev/null || echo "Cluster already exists"

# Step 3: Create Service Discovery namespace
echo -e "\n${YELLOW}🔍 Creating Service Discovery namespace...${NC}"
NAMESPACE_OPERATION_ID=$(aws servicediscovery create-private-dns-namespace \
    --name $NAMESPACE_NAME \
    --vpc $VPC_ID \
    --query 'OperationId' \
    --output text 2>/dev/null || echo "")

if [ -n "$NAMESPACE_OPERATION_ID" ]; then
    echo "Waiting for namespace creation..."
    sleep 5
    
    # Get namespace ID
    NAMESPACE_ID=$(aws servicediscovery list-namespaces \
        --query "Namespaces[?Name=='$NAMESPACE_NAME'].Id" \
        --output text)
    echo "Namespace ID: $NAMESPACE_ID"
else
    # Namespace already exists, get its ID
    NAMESPACE_ID=$(aws servicediscovery list-namespaces \
        --query "Namespaces[?Name=='$NAMESPACE_NAME'].Id" \
        --output text)
    echo "Using existing namespace: $NAMESPACE_ID"
fi

# Step 4: Create Service Discovery services
echo -e "\n${YELLOW}🎯 Creating Service Discovery services...${NC}"

# Create backend service discovery service
echo "Creating backend service discovery..."
BACKEND_SERVICE_ARN=$(aws servicediscovery create-service \
    --name backend-api \
    --namespace-id $NAMESPACE_ID \
    --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]" \
    --health-check-config "Type=HTTP,ResourcePath=/health,FailureThreshold=3" \
    --query 'Service.Arn' \
    --output text 2>/dev/null || \
    aws servicediscovery list-services \
        --query "Services[?Name=='backend-api'].Arn" \
        --output text)

echo "Backend Service Discovery ARN: $BACKEND_SERVICE_ARN"

# Create frontend service discovery service
echo "Creating frontend service discovery..."
FRONTEND_SERVICE_ARN=$(aws servicediscovery create-service \
    --name frontend-web \
    --namespace-id $NAMESPACE_ID \
    --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]" \
    --health-check-config "Type=HTTP,ResourcePath=/health,FailureThreshold=3" \
    --query 'Service.Arn' \
    --output text 2>/dev/null || \
    aws servicediscovery list-services \
        --query "Services[?Name=='frontend-web'].Arn" \
        --output text)

echo "Frontend Service Discovery ARN: $FRONTEND_SERVICE_ARN"

# Step 5: Register task definitions
echo -e "\n${YELLOW}📝 Registering task definitions...${NC}"

aws ecs register-task-definition --cli-input-json file://backend-task-def.json
aws ecs register-task-definition --cli-input-json file://frontend-task-def.json

# Step 6: Update service definitions with actual Service Discovery ARNs
echo -e "\n${YELLOW}📝 Updating service definitions with Service Discovery ARNs...${NC}"

# Create temporary service definition files with actual ARNs
sed "s|arn:aws:servicediscovery:us-east-1:000000000000:service/srv-backend|$BACKEND_SERVICE_ARN|" \
    backend-service-def.json > /tmp/backend-service-def.json

sed "s|arn:aws:servicediscovery:us-east-1:000000000000:service/srv-frontend|$FRONTEND_SERVICE_ARN|" \
    frontend-service-def.json > /tmp/frontend-service-def.json

# Step 7: Create ECS services with service discovery
echo -e "\n${YELLOW}🔧 Creating ECS services...${NC}"

# Create backend service
echo "Creating backend ECS service..."
aws ecs create-service \
    --cli-input-json file:///tmp/backend-service-def.json \
    --cluster $CLUSTER_NAME \
    2>/dev/null || echo "Backend service already exists"

# Create frontend service
echo "Creating frontend ECS service..."
aws ecs create-service \
    --cli-input-json file:///tmp/frontend-service-def.json \
    --cluster $CLUSTER_NAME \
    2>/dev/null || echo "Frontend service already exists"

# Step 7: Wait for services to be active
echo -e "\n${YELLOW}⏳ Waiting for services to become active...${NC}"
sleep 10

# Step 8: Test service discovery
echo -e "\n${YELLOW}🔍 Testing Service Discovery...${NC}"

echo "Discovering backend instances:"
aws servicediscovery discover-instances \
    --namespace-name $NAMESPACE_NAME \
    --service-name backend-api \
    --query 'Instances[*].[InstanceId,Attributes]' \
    --output table

echo "Discovering frontend instances:"
aws servicediscovery discover-instances \
    --namespace-name $NAMESPACE_NAME \
    --service-name frontend-web \
    --query 'Instances[*].[InstanceId,Attributes]' \
    --output table

echo -e "\n${GREEN}✅ Deployment complete!${NC}"
echo -e "\nYou can access:"
echo "  - Frontend: http://localhost:3000 (after port forwarding)"
echo "  - Backend API: http://backend-api.production.local:8080"
echo ""
echo "To test service communication:"
echo "  1. Port forward the frontend service"
echo "  2. Open http://localhost:3000 in your browser"
echo "  3. Click 'Call Backend Service' to test communication"