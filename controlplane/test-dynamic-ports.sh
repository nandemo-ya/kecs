#!/bin/bash
# Test script for dynamic Traefik port allocation

set -e

echo "Testing dynamic Traefik port allocation..."

# Test 1: Create first cluster - should get port 8090
echo -e "\n1. Creating first cluster (should get port 8090)..."
aws --endpoint-url http://localhost:8080 ecs create-cluster --cluster-name test-port-1

# Test 2: Create second cluster - should get port 8091
echo -e "\n2. Creating second cluster (should get port 8091)..."
aws --endpoint-url http://localhost:8080 ecs create-cluster --cluster-name test-port-2

# Test 3: Create third cluster - should get port 8092
echo -e "\n3. Creating third cluster (should get port 8092)..."
aws --endpoint-url http://localhost:8080 ecs create-cluster --cluster-name test-port-3

# Wait a bit for clusters to be ready
echo -e "\nWaiting for clusters to be ready..."
sleep 30

# Check port mappings
echo -e "\nChecking port mappings..."
echo "Port 8090 (cluster 1):"
docker ps | grep k3d-kecs-test-port-1-server | grep -o "0.0.0.0:[0-9]*->8090/tcp" || echo "Not found"

echo -e "\nPort 8091 (cluster 2):"
docker ps | grep k3d-kecs-test-port-2-server | grep -o "0.0.0.0:[0-9]*->8090/tcp" || echo "Not found"

echo -e "\nPort 8092 (cluster 3):"
docker ps | grep k3d-kecs-test-port-3-server | grep -o "0.0.0.0:[0-9]*->8090/tcp" || echo "Not found"

# List all clusters
echo -e "\nListing all clusters..."
aws --endpoint-url http://localhost:8080 ecs list-clusters

# Clean up
echo -e "\nCleaning up..."
aws --endpoint-url http://localhost:8080 ecs delete-cluster --cluster test-port-1
aws --endpoint-url http://localhost:8080 ecs delete-cluster --cluster test-port-2
aws --endpoint-url http://localhost:8080 ecs delete-cluster --cluster test-port-3

echo -e "\nTest completed!"