#!/bin/bash

# Set AWS endpoint to KECS
export AWS_ENDPOINT_URL=http://localhost:5373
export AWS_REGION=us-east-1

echo "Deploying nginx-with-log-generator example..."
echo "========================================="

# Check if default cluster exists
echo "1. Checking for default cluster..."
if ! aws ecs describe-clusters --clusters default 2>/dev/null | grep -q "ACTIVE"; then
    echo "   Creating default cluster..."
    aws ecs create-cluster --cluster-name default
else
    echo "   Default cluster already exists"
fi

# Register task definition
echo ""
echo "2. Registering task definition..."
aws ecs register-task-definition --cli-input-json file://task_def.json > /dev/null
if [ $? -eq 0 ]; then
    echo "   Task definition registered successfully"
else
    echo "   Failed to register task definition"
    exit 1
fi

# Create or update service
echo ""
echo "3. Creating service..."
# Check if service already exists
if aws ecs describe-services --cluster default --services nginx-with-log-generator 2>/dev/null | grep -q "serviceName"; then
    echo "   Service already exists, updating..."
    aws ecs update-service --cluster default --service nginx-with-log-generator --task-definition nginx-with-log-generator:1 --desired-count 1 > /dev/null
else
    echo "   Creating new service..."
    aws ecs create-service --cli-input-json file://service_def.json > /dev/null
fi

if [ $? -eq 0 ]; then
    echo "   Service deployed successfully"
else
    echo "   Failed to deploy service"
    exit 1
fi

echo ""
echo "Deployment completed!"
echo ""
echo "To view logs in TUI:"
echo "  kecs tui --instance <instance-name>"
echo ""
echo "To check pod status:"
echo "  kubectl get pods -n default-us-east-1"
echo ""
echo "To view container logs directly:"
echo "  kubectl logs -n default-us-east-1 <pod-name> nginx"
echo "  kubectl logs -n default-us-east-1 <pod-name> log-generator"