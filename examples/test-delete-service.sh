#!/bin/bash

# Test script for DeleteService API

echo "=== Test DeleteService API ==="

# Create a test cluster first
echo "1. Creating test cluster..."
curl -X POST http://localhost:8080/v1/createcluster \
  -H "Content-Type: application/json" \
  -d '{
    "clusterName": "test-delete-service"
  }'
echo -e "\n"

# Register a task definition
echo "2. Registering task definition..."
curl -X POST http://localhost:8080/v1/registertaskdefinition \
  -H "Content-Type: application/json" \
  -d '{
    "family": "test-delete-task",
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

# Create a service
echo "3. Creating service..."
curl -X POST http://localhost:8080/v1/createservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-delete-service",
    "serviceName": "test-service-to-delete",
    "taskDefinition": "test-delete-task:1",
    "desiredCount": 2
  }'
echo -e "\n"

# Describe the service to verify creation
echo "4. Describing service before update..."
curl -X POST http://localhost:8080/v1/describeservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-delete-service",
    "services": ["test-service-to-delete"]
  }' | jq '.services[0] | {serviceName, status, desiredCount, runningCount}'
echo -e "\n"

# Update service to set desired count to 0 (required for deletion)
echo "5. Updating service to set desired count to 0..."
curl -X POST http://localhost:8080/v1/updateservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-delete-service",
    "service": "test-service-to-delete",
    "desiredCount": 0
  }'
echo -e "\n"

# Delete the service
echo "6. Deleting service..."
curl -X POST http://localhost:8080/v1/deleteservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-delete-service",
    "service": "test-service-to-delete"
  }' | jq '.service | {serviceName, status}'
echo -e "\n"

# Try to describe the deleted service (should fail)
echo "7. Trying to describe deleted service (should fail)..."
curl -X POST http://localhost:8080/v1/describeservices \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-delete-service",
    "services": ["test-service-to-delete"]
  }' | jq '.'
echo -e "\n"

# Test force delete (create another service and delete with force=true)
echo "8. Creating another service for force delete test..."
curl -X POST http://localhost:8080/v1/createservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-delete-service",
    "serviceName": "test-service-force-delete",
    "taskDefinition": "test-delete-task:1",
    "desiredCount": 3
  }'
echo -e "\n"

# Force delete the service without setting desired count to 0
echo "9. Force deleting service with running tasks..."
curl -X POST http://localhost:8080/v1/deleteservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-delete-service",
    "service": "test-service-force-delete",
    "force": true
  }' | jq '.service | {serviceName, status, desiredCount}'
echo -e "\n"

# Clean up - delete the test cluster
echo "10. Cleaning up - deleting test cluster..."
curl -X POST http://localhost:8080/v1/deletecluster \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "test-delete-service"
  }'
echo -e "\n"

echo "=== Test completed ==="