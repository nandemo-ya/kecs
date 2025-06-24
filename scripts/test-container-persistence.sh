#!/bin/bash

# Test script for container mode persistence
set -e

echo "=== KECS Container Mode Persistence Test ==="
echo

# Configuration
CONTAINER_NAME="kecs-test"
DATA_DIR="./test-kecs-data"
IMAGE="${KECS_IMAGE:-ghcr.io/nandemo-ya/kecs:latest}"

# Cleanup function
cleanup() {
    echo
    echo "Cleaning up..."
    docker rm -f $CONTAINER_NAME 2>/dev/null || true
    echo "Test data is preserved in: $DATA_DIR"
}

# Set trap for cleanup on exit
trap cleanup EXIT

# Create data directory
echo "1. Creating data directory: $DATA_DIR"
mkdir -p $DATA_DIR
chmod 755 $DATA_DIR

# Start KECS container
echo
echo "2. Starting KECS container with mounted volume..."
docker run -d \
    --name $CONTAINER_NAME \
    -p 8080:8080 \
    -p 8081:8081 \
    -e KECS_CONTAINER_MODE=true \
    -e KECS_DATA_DIR=/data \
    -e KECS_LOG_LEVEL=info \
    -v $(pwd)/$DATA_DIR:/data \
    -v /var/run/docker.sock:/var/run/docker.sock \
    $IMAGE

# Wait for KECS to start
echo
echo "3. Waiting for KECS to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        echo "   KECS is ready!"
        break
    fi
    echo -n "."
    sleep 1
done
echo

# Create a test cluster
echo
echo "4. Creating test cluster..."
aws --endpoint-url http://localhost:8080 \
    ecs create-cluster \
    --cluster-name persistence-test-cluster \
    --region us-east-1 \
    2>/dev/null | jq -r '.cluster.clusterArn'

# List clusters
echo
echo "5. Listing clusters (should show our test cluster)..."
aws --endpoint-url http://localhost:8080 \
    ecs list-clusters \
    --region us-east-1 \
    2>/dev/null | jq -r '.clusterArns[]'

# Check data file exists
echo
echo "6. Checking data file..."
if [ -f "$DATA_DIR/kecs.db" ]; then
    echo "   ✓ Database file exists: $DATA_DIR/kecs.db"
    ls -lh $DATA_DIR/kecs.db
else
    echo "   ✗ Database file not found!"
    exit 1
fi

# Stop container
echo
echo "7. Stopping KECS container..."
docker stop $CONTAINER_NAME
docker rm $CONTAINER_NAME

# Restart container with same volume
echo
echo "8. Starting new KECS container with same volume..."
docker run -d \
    --name $CONTAINER_NAME \
    -p 8080:8080 \
    -p 8081:8081 \
    -e KECS_CONTAINER_MODE=true \
    -e KECS_DATA_DIR=/data \
    -e KECS_LOG_LEVEL=info \
    -v $(pwd)/$DATA_DIR:/data \
    -v /var/run/docker.sock:/var/run/docker.sock \
    $IMAGE

# Wait for KECS to start again
echo
echo "9. Waiting for KECS to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        echo "   KECS is ready!"
        break
    fi
    echo -n "."
    sleep 1
done
echo

# List clusters again
echo
echo "10. Listing clusters (should still show our test cluster)..."
CLUSTERS=$(aws --endpoint-url http://localhost:8080 \
    ecs list-clusters \
    --region us-east-1 \
    2>/dev/null | jq -r '.clusterArns[]')

echo "$CLUSTERS"

# Verify persistence
echo
echo "11. Verifying persistence..."
if echo "$CLUSTERS" | grep -q "persistence-test-cluster"; then
    echo "   ✓ SUCCESS: Cluster data persisted across container restart!"
else
    echo "   ✗ FAILED: Cluster data was not persisted!"
    exit 1
fi

# Cleanup cluster
echo
echo "12. Cleaning up test cluster..."
aws --endpoint-url http://localhost:8080 \
    ecs delete-cluster \
    --cluster persistence-test-cluster \
    --region us-east-1 \
    2>/dev/null > /dev/null

echo
echo "=== Test completed successfully! ==="
echo "Data was successfully persisted in: $DATA_DIR"