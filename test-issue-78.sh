#!/bin/bash

# Test script for issue #78 - CreateCluster missing attributes

echo "Testing CreateCluster with all attributes..."

# Generate unique cluster name
CLUSTER_NAME="test-issue-78-$(date +%s)"

# Create a cluster with all attributes
RESPONSE=$(curl -s -X POST http://localhost:8080/v1/CreateCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster" \
  -d '{
    "clusterName": "'$CLUSTER_NAME'",
    "tags": [
      {"key": "Environment", "value": "test"},
      {"key": "Team", "value": "backend"}
    ],
    "settings": [
      {"name": "containerInsights", "value": "enabled"}
    ],
    "capacityProviders": ["FARGATE", "FARGATE_SPOT"],
    "defaultCapacityProviderStrategy": [
      {"capacityProvider": "FARGATE", "weight": 1, "base": 0}
    ]
  }')

echo "Response:"
if [ -z "$RESPONSE" ]; then
  echo "Empty response - cluster might already exist"
else
  echo "$RESPONSE" | jq .
fi

echo -e "\nChecking for required fields..."

# Check for statistics
if echo "$RESPONSE" | jq -e '.cluster.statistics' > /dev/null; then
  echo "✓ statistics field present"
else
  echo "✗ statistics field missing"
fi

# Check for tags
if echo "$RESPONSE" | jq -e '.cluster.tags' > /dev/null; then
  echo "✓ tags field present"
else
  echo "✗ tags field missing"
fi

# Check for settings
if echo "$RESPONSE" | jq -e '.cluster.settings' > /dev/null; then
  echo "✓ settings field present"
else
  echo "✗ settings field missing"
fi

# Check for capacityProviders
if echo "$RESPONSE" | jq -e '.cluster.capacityProviders' > /dev/null; then
  echo "✓ capacityProviders field present"
else
  echo "✗ capacityProviders field missing"
fi

# Check for defaultCapacityProviderStrategy
if echo "$RESPONSE" | jq -e '.cluster.defaultCapacityProviderStrategy' > /dev/null; then
  echo "✓ defaultCapacityProviderStrategy field present"
else
  echo "✗ defaultCapacityProviderStrategy field missing"
fi