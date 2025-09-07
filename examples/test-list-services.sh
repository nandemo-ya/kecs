#!/bin/bash

# Test script for ListServices API

echo "=== Test ListServices API ==="

# Create a test cluster first
echo "1. Creating test cluster..."
curl -X POST http://localhost:5373/v1/createcluster \
  -H "Content-Type: application/json" \
  -d '{
    "clusterName": "test-list-services"
  }'
echo -e "\n"

# Register task definitions
echo "2. Registering task definitions..."
# First task definition
curl -X POST http://localhost:5373/v1/registertaskdefinition \
  -H "Content-Type: application/json" \
  -d '{
    "family": "test-task-fargate",
    "requiresCompatibilities": ["FARGATE"],
    "networkMode": "awsvpc",
    "cpu": "256",
    "memory": "512",
    "containerDefinitions": [
      {
        "name": "app",
        "image": "nginx:latest",
        "memory": 512,
        "cpu": 256
      }
    ]
  }'
echo -e "\n"

# Second task definition
curl -X POST http://localhost:5373/v1/registertaskdefinition \
  -H "Content-Type: application/json" \
  -d '{
    "family": "test-task-ec2",
    "requiresCompatibilities": ["EC2"],
    "containerDefinitions": [
      {
        "name": "app",
        "image": "httpd:latest",
        "memory": 512,
        "cpu": 256
      }
    ]
  }'
echo -e "\n"

# Create multiple services with different configurations
echo "3. Creating services..."

# FARGATE service 1
curl -X POST http://localhost:5373/v1/createservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-list-services",
    "serviceName": "fargate-service-1",
    "taskDefinition": "test-task-fargate:1",
    "desiredCount": 2,
    "launchType": "FARGATE",
    "schedulingStrategy": "REPLICA"
  }'
echo -e "\n"

# FARGATE service 2
curl -X POST http://localhost:5373/v1/createservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-list-services",
    "serviceName": "fargate-service-2",
    "taskDefinition": "test-task-fargate:1",
    "desiredCount": 1,
    "launchType": "FARGATE",
    "schedulingStrategy": "REPLICA"
  }'
echo -e "\n"

# EC2 service
curl -X POST http://localhost:5373/v1/createservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-list-services",
    "serviceName": "ec2-service-1",
    "taskDefinition": "test-task-ec2:1",
    "desiredCount": 3,
    "launchType": "EC2",
    "schedulingStrategy": "REPLICA"
  }'
echo -e "\n"

# Test 1: List all services
echo "4. Testing: List all services..."
curl -X POST http://localhost:5373/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-list-services"
  }' | jq '.'
echo -e "\n"

# Test 2: List services with maxResults (pagination)
echo "5. Testing: List services with maxResults=2..."
curl -X POST http://localhost:5373/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-list-services",
    "maxResults": 2
  }' | jq '.'
echo -e "\n"

# Test 3: List services filtered by launch type
echo "6. Testing: List only FARGATE services..."
curl -X POST http://localhost:5373/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-list-services",
    "launchType": "FARGATE"
  }' | jq '.'
echo -e "\n"

# Test 4: List services filtered by launch type EC2
echo "7. Testing: List only EC2 services..."
curl -X POST http://localhost:5373/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-list-services",
    "launchType": "EC2"
  }' | jq '.'
echo -e "\n"

# Test 5: List services from non-existent cluster
echo "8. Testing: List services from non-existent cluster..."
curl -X POST http://localhost:5373/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "non-existent-cluster"
  }' | jq '.'
echo -e "\n"

# Test 6: Test pagination with nextToken
echo "9. Testing: Pagination with nextToken..."
echo "First page:"
RESPONSE=$(curl -s -X POST http://localhost:5373/v1/listservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-list-services",
    "maxResults": 1
  }')
echo "$RESPONSE" | jq '.'
NEXT_TOKEN=$(echo "$RESPONSE" | jq -r '.nextToken // empty')

if [ -n "$NEXT_TOKEN" ]; then
  echo -e "\nNext page using nextToken:"
  curl -X POST http://localhost:5373/v1/listservices \
    -H "Content-Type: application/json" \
    -d "{
      \"cluster\": \"test-list-services\",
      \"maxResults\": 1,
      \"nextToken\": \"$NEXT_TOKEN\"
    }" | jq '.'
fi
echo -e "\n"

# Clean up - delete services and cluster
echo "10. Cleaning up..."
# Delete services
for service in "fargate-service-1" "fargate-service-2" "ec2-service-1"; do
  # First update to desired count 0
  curl -X POST http://localhost:5373/v1/updateservice \
    -H "Content-Type: application/json" \
    -d "{
      \"cluster\": \"test-list-services\",
      \"service\": \"$service\",
      \"desiredCount\": 0
    }" > /dev/null 2>&1
  
  # Then delete
  curl -X POST http://localhost:5373/v1/deleteservice \
    -H "Content-Type: application/json" \
    -d "{
      \"cluster\": \"test-list-services\",
      \"service\": \"$service\"
    }" > /dev/null 2>&1
done

# Delete cluster
curl -X POST http://localhost:5373/v1/deletecluster \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-list-services"
  }'
echo -e "\n"

echo "=== Test completed ==="