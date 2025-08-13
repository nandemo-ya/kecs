#!/bin/bash

# Simple k3d performance test without color codes
set -e

echo "=== K3D Performance Test ==="
echo "Testing current k3d configuration..."

# Test 1: Standard k3d cluster (current implementation)
echo "Test 1: Standard k3d cluster creation"
CLUSTER_NAME="kecs-perf-standard-$(date +%s)"
START_TIME=$(date +%s)
k3d cluster create "$CLUSTER_NAME" \
    --servers 1 \
    --agents 0 \
    --wait
END_TIME=$(date +%s)
CREATION_TIME=$((END_TIME - START_TIME))
echo "Creation time: ${CREATION_TIME}s"

# Get memory usage
echo "Container stats:"
docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep "$CLUSTER_NAME" || true

# Delete cluster
START_TIME=$(date +%s)
k3d cluster delete "$CLUSTER_NAME"
END_TIME=$(date +%s)
DELETION_TIME=$((END_TIME - START_TIME))
echo "Deletion time: ${DELETION_TIME}s"

echo ""
echo "Test 2: Optimized k3d cluster creation"
CLUSTER_NAME="kecs-perf-optimized-$(date +%s)"
START_TIME=$(date +%s)
k3d cluster create "$CLUSTER_NAME" \
    --servers 1 \
    --agents 0 \
    --no-lb \
    --no-image-volume \
    --k3s-arg "--disable=traefik@server:0" \
    --k3s-arg "--disable=metrics-server@server:0" \
    --k3s-arg "--disable=servicelb@server:0" \
    --wait
END_TIME=$(date +%s)
CREATION_TIME_OPT=$((END_TIME - START_TIME))
echo "Creation time: ${CREATION_TIME_OPT}s"

# Get memory usage
echo "Container stats:"
docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep "$CLUSTER_NAME" || true

# Delete cluster
START_TIME=$(date +%s)
k3d cluster delete "$CLUSTER_NAME"
END_TIME=$(date +%s)
DELETION_TIME_OPT=$((END_TIME - START_TIME))
echo "Deletion time: ${DELETION_TIME_OPT}s"

echo ""
echo "=== Summary ==="
echo "Standard creation: ${CREATION_TIME}s, deletion: ${DELETION_TIME}s"
echo "Optimized creation: ${CREATION_TIME_OPT}s, deletion: ${DELETION_TIME_OPT}s"
echo "Improvement: $((CREATION_TIME - CREATION_TIME_OPT))s ($(( (CREATION_TIME - CREATION_TIME_OPT) * 100 / CREATION_TIME ))%)"