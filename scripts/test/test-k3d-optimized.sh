#!/bin/bash

# Test script to compare k3d cluster creation performance
# Compares the old CreateCluster vs new CreateClusterOptimized

set -e

echo "=== K3D Performance Comparison Test ==="
echo "Date: $(date)"
echo ""

# Function to run KECS and create a cluster
test_cluster_creation() {
    local test_name=$1
    local use_optimized=$2
    local cluster_name="test-cluster-$(date +%s)"
    
    echo "--- Test: $test_name ---"
    
    # Start KECS with appropriate settings
    export KECS_TEST_MODE="false"  # Use real k3d clusters
    export KECS_K3D_OPTIMIZED="$use_optimized"
    export LOG_LEVEL="info"
    
    # Start KECS in background
    ./bin/kecs server --api-port 8080 --admin-port 8081 > /tmp/kecs-$test_name.log 2>&1 &
    KECS_PID=$!
    
    # Wait for KECS to be ready
    echo "Waiting for KECS to start..."
    for i in {1..30}; do
        if curl -s http://localhost:8081/health > /dev/null 2>&1; then
            echo "KECS is ready"
            break
        fi
        sleep 1
    done
    
    # Create cluster and measure time
    echo "Creating cluster: $cluster_name"
    START_TIME=$(date +%s)
    
    aws ecs create-cluster --cluster-name "$cluster_name" \
        --endpoint-url http://localhost:8080 \
        --region us-east-1 \
        --no-cli-pager 2>&1 | jq .
    
    # Wait for cluster to be fully ready
    for i in {1..60}; do
        STATUS=$(aws ecs describe-clusters --clusters "$cluster_name" \
            --endpoint-url http://localhost:8080 \
            --region us-east-1 \
            --no-cli-pager 2>&1 | jq -r '.clusters[0].status // "UNKNOWN"')
        
        if [ "$STATUS" = "ACTIVE" ]; then
            break
        fi
        sleep 1
    done
    
    END_TIME=$(date +%s)
    CREATION_TIME=$((END_TIME - START_TIME))
    echo "Cluster creation time: ${CREATION_TIME}s"
    echo "Cluster status: $STATUS"
    
    # Check k3d cluster
    echo "K3d clusters:"
    k3d cluster list | grep kecs || true
    
    # Get container stats
    echo "Container stats:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep kecs || true
    
    # Delete cluster
    echo "Deleting cluster..."
    aws ecs delete-cluster --cluster "$cluster_name" \
        --endpoint-url http://localhost:8080 \
        --region us-east-1 \
        --no-cli-pager > /dev/null 2>&1
    
    # Stop KECS
    kill $KECS_PID 2>/dev/null || true
    wait $KECS_PID 2>/dev/null || true
    
    # Clean up any remaining k3d clusters
    k3d cluster list -o json | jq -r '.[].name' | grep '^kecs-' | xargs -I {} k3d cluster delete {} 2>/dev/null || true
    
    echo "Test completed"
    echo ""
    
    return $CREATION_TIME
}

# Run tests
echo "Running standard cluster creation test..."
test_cluster_creation "standard" "false"
STANDARD_TIME=$?

sleep 5

echo "Running optimized cluster creation test..."
test_cluster_creation "optimized" "true"
OPTIMIZED_TIME=$?

# Summary
echo "=== Performance Comparison Summary ==="
echo "Standard cluster creation: ${STANDARD_TIME}s"
echo "Optimized cluster creation: ${OPTIMIZED_TIME}s"
IMPROVEMENT=$((STANDARD_TIME - OPTIMIZED_TIME))
if [ $STANDARD_TIME -gt 0 ]; then
    IMPROVEMENT_PCT=$(( (IMPROVEMENT * 100) / STANDARD_TIME ))
    echo "Improvement: ${IMPROVEMENT}s (${IMPROVEMENT_PCT}%)"
fi

# Check logs for any errors
echo ""
echo "Checking for errors in logs..."
grep -i error /tmp/kecs-*.log | head -10 || echo "No errors found"