#!/bin/bash
# Test script to verify LocalStack IAM role initialization

set -e

echo "Testing LocalStack IAM role initialization..."

# Start KECS with LocalStack
echo "Starting KECS..."
./bin/kecs start --name test-iam-init

# Wait for LocalStack to be ready
echo "Waiting for LocalStack to be ready..."
sleep 30

# Check if IAM roles were created
echo "Checking for ecsTaskExecutionRole..."
aws --endpoint-url=http://localhost:4566 iam get-role --role-name ecsTaskExecutionRole || {
    echo "ERROR: ecsTaskExecutionRole not found"
    ./bin/kecs stop --name test-iam-init
    exit 1
}

echo "Checking for ecsTaskRole..."
aws --endpoint-url=http://localhost:4566 iam get-role --role-name ecsTaskRole || {
    echo "ERROR: ecsTaskRole not found"
    ./bin/kecs stop --name test-iam-init
    exit 1
}

echo "Success! Both IAM roles were created automatically."

# Cleanup
echo "Stopping KECS..."
./bin/kecs stop --name test-iam-init

echo "Test completed successfully!"