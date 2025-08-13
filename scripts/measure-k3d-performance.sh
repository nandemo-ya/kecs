#!/bin/bash

# Script to measure k3d cluster creation and deletion performance
# Used as a baseline before implementing performance optimizations

set -e

# Colors for output - disable if not a terminal
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Configuration
CLUSTER_NAME="kecs-perf-test-$(date +%s)"
RESULTS_FILE="k3d-performance-baseline.txt"
ITERATIONS=3

echo -e "${BLUE}=== K3D Performance Baseline Measurement ===${NC}"
echo "Cluster name: $CLUSTER_NAME"
echo "Iterations: $ITERATIONS"
echo "Results file: $RESULTS_FILE"
echo ""

# Clear previous results
echo "K3D Performance Baseline Measurement" > "$RESULTS_FILE"
echo "Date: $(date)" >> "$RESULTS_FILE"
echo "========================================" >> "$RESULTS_FILE"
echo "" >> "$RESULTS_FILE"

# Function to measure time
measure_time() {
    local start_time=$(date +%s.%N)
    "$@"
    local end_time=$(date +%s.%N)
    echo "$end_time - $start_time" | bc
}

# Function to get memory usage
get_memory_usage() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS - get total memory used by k3d processes
        ps aux | grep -E "k3d|k3s" | grep -v grep | awk '{sum+=$6} END {print sum/1024 " MB"}'
    else
        # Linux
        ps aux | grep -E "k3d|k3s" | grep -v grep | awk '{sum+=$6} END {print sum/1024 " MB"}'
    fi
}

# Function to get container stats
get_container_stats() {
    local cluster=$1
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep "$cluster" || true
}

# Run performance tests
total_create_time=0
total_delete_time=0

for i in $(seq 1 $ITERATIONS); do
    echo -e "${YELLOW}--- Iteration $i/$ITERATIONS ---${NC}"
    echo "Iteration $i/$ITERATIONS" >> "$RESULTS_FILE"
    echo "-------------------" >> "$RESULTS_FILE"
    
    # Test cluster creation
    echo -e "${BLUE}Creating k3d cluster...${NC}"
    create_time=$(measure_time k3d cluster create "$CLUSTER_NAME-$i" \
        --servers 1 \
        --agents 0 \
        --no-lb \
        --k3s-arg "--disable=traefik@server:0" \
        --k3s-arg "--disable=metrics-server@server:0" \
        --wait)
    
    echo -e "${GREEN}Creation time: ${create_time}s${NC}"
    echo "Creation time: ${create_time}s" >> "$RESULTS_FILE"
    total_create_time=$(echo "$total_create_time + $create_time" | bc)
    
    # Wait for cluster to stabilize
    sleep 5
    
    # Get resource usage
    echo -e "${BLUE}Measuring resource usage...${NC}"
    memory_usage=$(get_memory_usage)
    echo "Memory usage: $memory_usage" >> "$RESULTS_FILE"
    
    echo -e "${BLUE}Container stats:${NC}"
    echo "Container stats:" >> "$RESULTS_FILE"
    get_container_stats "$CLUSTER_NAME-$i" | tee -a "$RESULTS_FILE"
    
    # Test cluster deletion
    echo -e "${BLUE}Deleting k3d cluster...${NC}"
    delete_time=$(measure_time k3d cluster delete "$CLUSTER_NAME-$i")
    
    echo -e "${GREEN}Deletion time: ${delete_time}s${NC}"
    echo "Deletion time: ${delete_time}s" >> "$RESULTS_FILE"
    echo "" >> "$RESULTS_FILE"
    total_delete_time=$(echo "$total_delete_time + $delete_time" | bc)
    
    # Wait between iterations
    if [ $i -lt $ITERATIONS ]; then
        echo -e "${BLUE}Waiting before next iteration...${NC}"
        sleep 5
    fi
done

# Calculate averages
avg_create_time=$(echo "scale=2; $total_create_time / $ITERATIONS" | bc)
avg_delete_time=$(echo "scale=2; $total_delete_time / $ITERATIONS" | bc)

echo -e "${GREEN}=== Summary ===${NC}"
echo "Average creation time: ${avg_create_time}s"
echo "Average deletion time: ${avg_delete_time}s"

echo "" >> "$RESULTS_FILE"
echo "Summary" >> "$RESULTS_FILE"
echo "=======" >> "$RESULTS_FILE"
echo "Average creation time: ${avg_create_time}s" >> "$RESULTS_FILE"
echo "Average deletion time: ${avg_delete_time}s" >> "$RESULTS_FILE"
echo "Total test duration: $(echo "$total_create_time + $total_delete_time" | bc)s" >> "$RESULTS_FILE"

# Additional information
echo "" >> "$RESULTS_FILE"
echo "Environment Information" >> "$RESULTS_FILE"
echo "======================" >> "$RESULTS_FILE"
echo "k3d version: $(k3d version | head -n1)" >> "$RESULTS_FILE"
echo "Docker version: $(docker version --format '{{.Server.Version}}')" >> "$RESULTS_FILE"
echo "OS: $(uname -s) $(uname -r)" >> "$RESULTS_FILE"

echo -e "${GREEN}Performance baseline saved to: $RESULTS_FILE${NC}"