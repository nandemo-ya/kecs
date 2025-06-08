#!/bin/bash

# Test script for kind cluster reuse with existing cluster

echo "=== Testing kind cluster reuse with pre-existing ECS cluster ==="

CLUSTER_NAME="test-existing-cluster"

# First, create a cluster
echo -e "\n1. Creating ECS cluster '$CLUSTER_NAME'..."
curl -s -X POST http://localhost:8080/v1/CreateCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster" \
  -d '{
    "clusterName": "'$CLUSTER_NAME'",
    "tags": [{"key": "Test", "value": "existing"}]
  }' | jq -r '.cluster.clusterName // "Created"'

# Wait for kind cluster to be created
echo -e "\n2. Waiting for kind cluster creation..."
sleep 15

# Check kind clusters
echo -e "\n3. Current kind clusters:"
docker ps --filter "name=kecs-$CLUSTER_NAME" --format "table {{.Names}}\t{{.Status}}"

# Delete the kind cluster manually (simulate manual deletion)
echo -e "\n4. Deleting kind cluster manually (simulating accidental deletion)..."
docker rm -f "kecs-$CLUSTER_NAME-control-plane" 2>/dev/null

# Verify it's gone
echo -e "\n5. Kind clusters after manual deletion:"
docker ps --filter "name=kecs-$CLUSTER_NAME" --format "table {{.Names}}\t{{.Status}}"

# Try to use the cluster again - should recreate the kind cluster
echo -e "\n6. Trying to use the ECS cluster again (should recreate kind cluster)..."
DESCRIBE_RESPONSE=$(curl -s -X POST http://localhost:8080/v1/DescribeClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DescribeClusters" \
  -d '{"clusters": ["'$CLUSTER_NAME'"]}')

echo "Describe response:"
echo "$DESCRIBE_RESPONSE" | jq -r '.clusters[0].clusterName // "Not found"'

# Wait for potential recreation
echo -e "\n7. Waiting for potential kind cluster recreation..."
sleep 10

# Check if kind cluster was recreated
echo -e "\n8. Kind clusters after describe operation:"
docker ps --filter "name=kecs-$CLUSTER_NAME" --format "table {{.Names}}\t{{.Status}}"

# Count kind clusters with this name
COUNT=$(docker ps --filter "name=kecs-$CLUSTER_NAME" -q | wc -l)
echo -e "\nNumber of kind clusters for '$CLUSTER_NAME': $COUNT"

# Cleanup - delete the test cluster
echo -e "\n9. Cleaning up - deleting ECS cluster..."
curl -s -X POST http://localhost:8080/v1/DeleteCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DeleteCluster" \
  -d '{"cluster": "'$CLUSTER_NAME'"}' > /dev/null

echo "Done."