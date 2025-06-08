#!/bin/bash

# Test script for kind cluster reuse feature

echo "=== Testing kind cluster reuse ==="

CLUSTER_NAME="test-reuse-cluster"

# First, create a cluster
echo -e "\n1. Creating ECS cluster '$CLUSTER_NAME'..."
RESPONSE1=$(curl -s -X POST http://localhost:8080/v1/CreateCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster" \
  -d '{
    "clusterName": "'$CLUSTER_NAME'",
    "tags": [{"key": "Test", "value": "reuse"}]
  }')

echo "Response:"
echo "$RESPONSE1" | jq -r '.cluster.clusterName // empty'

# Wait for kind cluster to be created
echo -e "\n2. Waiting for kind cluster creation..."
sleep 10

# Check kind clusters
echo -e "\n3. Current kind clusters:"
docker ps --filter "name=kecs-$CLUSTER_NAME" --format "table {{.Names}}\t{{.Status}}"

# Create the same cluster again
echo -e "\n4. Creating the same ECS cluster again..."
RESPONSE2=$(curl -s -X POST http://localhost:8080/v1/CreateCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster" \
  -d '{
    "clusterName": "'$CLUSTER_NAME'",
    "tags": [{"key": "Test", "value": "reuse"}]
  }')

echo "Response:"
if [ -z "$RESPONSE2" ]; then
  echo "(Empty response - cluster already exists)"
else
  echo "$RESPONSE2" | jq -r '.cluster.clusterName // empty'
fi

# Wait a bit
sleep 5

# Check kind clusters again
echo -e "\n5. Kind clusters after second create (should be the same):"
docker ps --filter "name=kecs-$CLUSTER_NAME" --format "table {{.Names}}\t{{.Status}}"

# Count kind clusters with this name
COUNT=$(docker ps --filter "name=kecs-$CLUSTER_NAME" -q | wc -l)
echo -e "\nNumber of kind clusters for '$CLUSTER_NAME': $COUNT"

if [ "$COUNT" -eq 1 ]; then
  echo "✓ SUCCESS: Only one kind cluster exists (reused successfully)"
else
  echo "✗ FAIL: Expected 1 kind cluster, found $COUNT"
fi

# Cleanup - delete the test cluster
echo -e "\n6. Cleaning up - deleting ECS cluster..."
curl -s -X POST http://localhost:8080/v1/DeleteCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DeleteCluster" \
  -d '{"cluster": "'$CLUSTER_NAME'"}' > /dev/null

echo "Done."