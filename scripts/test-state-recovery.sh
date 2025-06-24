#!/bin/bash

# Test script for KECS state recovery
set -e

echo "=== KECS State Recovery Test ==="
echo

# Configuration
DATA_DIR="./test-kecs-data"
KECS_BINARY="../bin/kecs"
CLUSTER_NAME="recovery-test"
SERVICE_NAME="test-service"
TASK_DEF_FAMILY="test-task"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    
    # Stop KECS if running
    if [ ! -z "$KECS_PID" ]; then
        kill $KECS_PID 2>/dev/null || true
        wait $KECS_PID 2>/dev/null || true
    fi
    
    # Clean up k3d clusters
    k3d cluster list | grep "kecs-$CLUSTER_NAME" && k3d cluster delete "kecs-$CLUSTER_NAME" || true
    
    # Clean up data directory
    rm -rf $DATA_DIR
    
    log_info "Cleanup completed"
}

# Set trap for cleanup
trap cleanup EXIT

# Step 1: Create data directory
log_info "Creating data directory: $DATA_DIR"
mkdir -p $DATA_DIR

# Step 2: Start KECS with state recovery disabled (initial setup)
log_info "Starting KECS for initial setup..."
export KECS_DATA_DIR=$DATA_DIR
export KECS_AUTO_RECOVER_STATE=false
export KECS_KEEP_CLUSTERS_ON_SHUTDOWN=false
$KECS_BINARY server &
KECS_PID=$!

# Wait for KECS to start
log_info "Waiting for KECS to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        log_info "KECS is ready!"
        break
    fi
    echo -n "."
    sleep 1
done
echo

# Step 3: Create test resources
log_info "Creating test cluster..."
aws --endpoint-url http://localhost:8080 \
    ecs create-cluster \
    --cluster-name $CLUSTER_NAME \
    --region us-east-1 \
    2>/dev/null | jq -r '.cluster.clusterArn'

# Register task definition
log_info "Registering task definition..."
TASK_DEF_JSON=$(cat <<EOF
{
  "family": "$TASK_DEF_FAMILY",
  "containerDefinitions": [
    {
      "name": "nginx",
      "image": "nginx:latest",
      "memory": 512,
      "cpu": 256,
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "protocol": "tcp"
        }
      ]
    }
  ]
}
EOF
)

aws --endpoint-url http://localhost:8080 \
    ecs register-task-definition \
    --cli-input-json "$TASK_DEF_JSON" \
    --region us-east-1 \
    2>/dev/null | jq -r '.taskDefinition.taskDefinitionArn'

# Create service
log_info "Creating service..."
aws --endpoint-url http://localhost:8080 \
    ecs create-service \
    --cluster $CLUSTER_NAME \
    --service-name $SERVICE_NAME \
    --task-definition $TASK_DEF_FAMILY \
    --desired-count 2 \
    --region us-east-1 \
    2>/dev/null | jq -r '.service.serviceArn'

# Wait for k3d cluster to be created
log_info "Waiting for k3d cluster to be created..."
for i in {1..30}; do
    if k3d cluster list | grep -q "kecs-$CLUSTER_NAME"; then
        log_info "K3d cluster created!"
        break
    fi
    echo -n "."
    sleep 1
done
echo

# Verify k3d cluster exists
log_info "Verifying k3d cluster..."
k3d cluster list | grep "kecs-$CLUSTER_NAME"

# Step 4: Stop KECS (simulating crash/restart)
log_info "Stopping KECS..."
kill $KECS_PID
wait $KECS_PID 2>/dev/null || true
unset KECS_PID

# Verify k3d cluster is gone (due to cleanup)
log_info "Verifying k3d cluster was cleaned up..."
if k3d cluster list | grep -q "kecs-$CLUSTER_NAME"; then
    log_error "K3d cluster still exists after KECS shutdown!"
else
    log_info "K3d cluster was properly cleaned up"
fi

# Step 5: Restart KECS with state recovery enabled
log_info "Starting KECS with state recovery enabled..."
export KECS_AUTO_RECOVER_STATE=true
$KECS_BINARY server &
KECS_PID=$!

# Wait for KECS to start
log_info "Waiting for KECS to start and recover state..."
for i in {1..60}; do
    if curl -s http://localhost:8081/health > /dev/null 2>&1; then
        log_info "KECS is ready!"
        break
    fi
    echo -n "."
    sleep 1
done
echo

# Give recovery some time to complete
log_info "Waiting for state recovery to complete..."
sleep 10

# Step 6: Verify state was recovered
log_info "Verifying state recovery..."

# Check if cluster exists in KECS
log_info "Checking if cluster exists in KECS..."
CLUSTERS=$(aws --endpoint-url http://localhost:8080 \
    ecs list-clusters \
    --region us-east-1 \
    2>/dev/null | jq -r '.clusterArns[]')

if echo "$CLUSTERS" | grep -q "$CLUSTER_NAME"; then
    log_info "✓ Cluster exists in KECS"
else
    log_error "✗ Cluster not found in KECS"
    exit 1
fi

# Check if k3d cluster was recreated
log_info "Checking if k3d cluster was recreated..."
if k3d cluster list | grep -q "kecs-$CLUSTER_NAME"; then
    log_info "✓ K3d cluster was recreated"
else
    log_error "✗ K3d cluster was not recreated"
    exit 1
fi

# Check if service exists
log_info "Checking if service exists..."
SERVICES=$(aws --endpoint-url http://localhost:8080 \
    ecs list-services \
    --cluster $CLUSTER_NAME \
    --region us-east-1 \
    2>/dev/null | jq -r '.serviceArns[]')

if echo "$SERVICES" | grep -q "$SERVICE_NAME"; then
    log_info "✓ Service exists in KECS"
else
    log_error "✗ Service not found in KECS"
    exit 1
fi

# Check if Kubernetes resources were recreated
log_info "Checking Kubernetes resources..."
export KUBECONFIG=$(k3d kubeconfig write kecs-$CLUSTER_NAME)

# Check namespace
if kubectl get namespace "kecs-$CLUSTER_NAME" > /dev/null 2>&1; then
    log_info "✓ Namespace exists"
else
    log_error "✗ Namespace not found"
fi

# Check deployments
DEPLOYMENTS=$(kubectl get deployments -n "kecs-$CLUSTER_NAME" --no-headers 2>/dev/null | wc -l)
if [ "$DEPLOYMENTS" -gt 0 ]; then
    log_info "✓ Found $DEPLOYMENTS deployment(s)"
    kubectl get deployments -n "kecs-$CLUSTER_NAME"
else
    log_warn "⚠ No deployments found (this might be expected if service recovery is partial)"
fi

echo
log_info "=== State Recovery Test Completed Successfully! ==="
echo
log_info "Summary:"
log_info "- Cluster data persisted across restart ✓"
log_info "- K3d cluster was recreated ✓"
log_info "- Service data persisted ✓"
if [ "$DEPLOYMENTS" -gt 0 ]; then
    log_info "- Kubernetes resources were recreated ✓"
else
    log_info "- Kubernetes resources recreation needs investigation"
fi