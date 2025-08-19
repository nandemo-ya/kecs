#!/bin/bash
set -e

# Blue/Green Deployment Script for KECS TaskSet
# This script demonstrates how to perform a Blue/Green deployment using ECS TaskSets

# Configuration
CLUSTER_NAME="${CLUSTER_NAME:-default}"
SERVICE_NAME="webapp-service"
AWS_REGION="${AWS_REGION:-us-east-1}"
AWS_ENDPOINT="${AWS_ENDPOINT_URL:-http://localhost:8080}"

# Colors for output
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_blue() {
    echo -e "${BLUE}[BLUE]${NC} $1"
}

log_green() {
    echo -e "${GREEN}[GREEN]${NC} $1"
}

# AWS CLI wrapper
aws_ecs() {
    aws ecs "$@" --region "$AWS_REGION" --endpoint-url "$AWS_ENDPOINT" --no-cli-pager
}

# Step 1: Create ECS Cluster
create_cluster() {
    log_info "Creating ECS cluster: $CLUSTER_NAME"
    aws_ecs create-cluster --cluster-name "$CLUSTER_NAME" || true
    log_info "Cluster created/exists: $CLUSTER_NAME"
}

# Step 2: Register Task Definitions
register_task_definitions() {
    log_info "Registering Blue task definition..."
    BLUE_TASK_DEF=$(aws_ecs register-task-definition \
        --cli-input-json file://task_def_blue.json \
        --query 'taskDefinition.taskDefinitionArn' \
        --output text)
    log_blue "Blue task definition: $BLUE_TASK_DEF"
    
    log_info "Registering Green task definition..."
    GREEN_TASK_DEF=$(aws_ecs register-task-definition \
        --cli-input-json file://task_def_green.json \
        --query 'taskDefinition.taskDefinitionArn' \
        --output text)
    log_green "Green task definition: $GREEN_TASK_DEF"
}

# Step 3: Create Service with EXTERNAL deployment controller
create_service() {
    log_info "Creating service with EXTERNAL deployment controller..."
    aws_ecs create-service \
        --cli-input-json file://service_def.json \
        --query 'service.serviceArn' \
        --output text || {
        log_warn "Service might already exist, continuing..."
    }
}

# Step 4: Create Blue TaskSet (Primary)
create_blue_taskset() {
    log_blue "Creating Blue TaskSet as PRIMARY..."
    
    BLUE_TASKSET=$(aws_ecs create-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --task-definition webapp:1 \
        --external-id "blue-deployment" \
        --network-configuration "awsvpcConfiguration={subnets=[subnet-12345,subnet-67890],securityGroups=[sg-webapp],assignPublicIp=ENABLED}" \
        --launch-type FARGATE \
        --scale "value=100,unit=PERCENT" \
        --query 'taskSet.taskSetArn' \
        --output text 2>/dev/null) || {
        log_warn "Blue TaskSet might already exist"
        BLUE_TASKSET=$(aws_ecs describe-task-sets \
            --cluster "$CLUSTER_NAME" \
            --service "$SERVICE_NAME" \
            --query "taskSets[?externalId=='blue-deployment'].taskSetArn | [0]" \
            --output text)
    }
    
    log_blue "Blue TaskSet ARN: $BLUE_TASKSET"
    
    # Update service to set Blue as PRIMARY
    log_info "Setting Blue TaskSet as PRIMARY..."
    aws_ecs update-service-primary-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --primary-task-set "$BLUE_TASKSET" || true
}

# Step 5: Create Green TaskSet (Standby)
create_green_taskset() {
    log_green "Creating Green TaskSet in standby (0% scale)..."
    
    GREEN_TASKSET=$(aws_ecs create-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --task-definition webapp:2 \
        --external-id "green-deployment" \
        --network-configuration "awsvpcConfiguration={subnets=[subnet-12345,subnet-67890],securityGroups=[sg-webapp],assignPublicIp=ENABLED}" \
        --launch-type FARGATE \
        --scale "value=0,unit=PERCENT" \
        --query 'taskSet.taskSetArn' \
        --output text 2>/dev/null) || {
        log_warn "Green TaskSet might already exist"
        GREEN_TASKSET=$(aws_ecs describe-task-sets \
            --cluster "$CLUSTER_NAME" \
            --service "$SERVICE_NAME" \
            --query "taskSets[?externalId=='green-deployment'].taskSetArn | [0]" \
            --output text)
    }
    
    log_green "Green TaskSet ARN: $GREEN_TASKSET"
}

# Step 6: Show TaskSet Status
show_status() {
    log_info "Current TaskSet status:"
    aws_ecs describe-task-sets \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --query 'taskSets[*].[externalId,status,scale.value,scale.unit,computedDesiredCount,runningCount,pendingCount]' \
        --output table
}

# Step 7: Perform Blue to Green Switch
switch_to_green() {
    log_info "Starting Blue to Green deployment..."
    
    # Scale up Green TaskSet
    log_green "Scaling up Green TaskSet to 100%..."
    aws_ecs update-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --task-set "$GREEN_TASKSET" \
        --scale "value=100,unit=PERCENT"
    
    log_info "Waiting for Green TaskSet to stabilize (30 seconds)..."
    sleep 30
    
    # Make Green PRIMARY
    log_green "Setting Green TaskSet as PRIMARY..."
    aws_ecs update-service-primary-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --primary-task-set "$GREEN_TASKSET"
    
    log_info "Waiting for traffic shift (10 seconds)..."
    sleep 10
    
    # Scale down Blue TaskSet
    log_blue "Scaling down Blue TaskSet to 0%..."
    aws_ecs update-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --task-set "$BLUE_TASKSET" \
        --scale "value=0,unit=PERCENT"
    
    log_green "Blue to Green deployment completed!"
}

# Step 8: Rollback to Blue
rollback_to_blue() {
    log_warn "Rolling back to Blue deployment..."
    
    # Scale up Blue TaskSet
    log_blue "Scaling up Blue TaskSet to 100%..."
    aws_ecs update-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --task-set "$BLUE_TASKSET" \
        --scale "value=100,unit=PERCENT"
    
    log_info "Waiting for Blue TaskSet to stabilize (30 seconds)..."
    sleep 30
    
    # Make Blue PRIMARY
    log_blue "Setting Blue TaskSet as PRIMARY..."
    aws_ecs update-service-primary-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --primary-task-set "$BLUE_TASKSET"
    
    log_info "Waiting for traffic shift (10 seconds)..."
    sleep 10
    
    # Scale down Green TaskSet
    log_green "Scaling down Green TaskSet to 0%..."
    aws_ecs update-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --task-set "$GREEN_TASKSET" \
        --scale "value=0,unit=PERCENT"
    
    log_blue "Rollback to Blue completed!"
}

# Step 9: Cleanup
cleanup() {
    log_warn "Cleaning up resources..."
    
    # Delete TaskSets
    log_info "Deleting TaskSets..."
    aws_ecs delete-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --task-set "$BLUE_TASKSET" \
        --force || true
    
    aws_ecs delete-task-set \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --task-set "$GREEN_TASKSET" \
        --force || true
    
    # Delete Service
    log_info "Deleting service..."
    aws_ecs delete-service \
        --cluster "$CLUSTER_NAME" \
        --service "$SERVICE_NAME" \
        --force || true
    
    log_info "Cleanup completed!"
}

# Main execution
main() {
    case "${1:-}" in
        setup)
            create_cluster
            register_task_definitions
            create_service
            create_blue_taskset
            create_green_taskset
            show_status
            ;;
        deploy)
            switch_to_green
            show_status
            ;;
        rollback)
            rollback_to_blue
            show_status
            ;;
        status)
            show_status
            ;;
        cleanup)
            cleanup
            ;;
        *)
            echo "Usage: $0 {setup|deploy|rollback|status|cleanup}"
            echo ""
            echo "Commands:"
            echo "  setup    - Create cluster, service, and both TaskSets (Blue as PRIMARY)"
            echo "  deploy   - Switch from Blue to Green deployment"
            echo "  rollback - Switch back from Green to Blue"
            echo "  status   - Show current TaskSet status"
            echo "  cleanup  - Delete all resources"
            echo ""
            echo "Environment variables:"
            echo "  AWS_ENDPOINT_URL - KECS endpoint (default: http://localhost:8080)"
            echo "  AWS_REGION      - AWS region (default: us-east-1)"
            echo "  CLUSTER_NAME    - ECS cluster name (default: default)"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"