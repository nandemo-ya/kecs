#!/bin/bash

# Test script for ECS task with artifacts

# Set the endpoint (adjust if KECS is running on a different port)
ENDPOINT="http://localhost:8080"

# Register task definition with artifacts
echo "Registering task definition with artifacts..."
curl -X POST "$ENDPOINT/v1/RegisterTaskDefinition" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition" \
  -d @task-definition-with-artifacts.json

echo -e "\n\nTask definition registered. When you run a task with this definition:"
echo "1. KECS will create init containers to download artifacts"
echo "2. Artifacts from S3 will be downloaded (requires LocalStack with S3 enabled)"
echo "3. Artifacts from HTTP/HTTPS will be downloaded using wget"
echo "4. The main container will have access to artifacts at /artifacts/*"

echo -e "\n\nTo run a task with this definition:"
echo 'curl -X POST "$ENDPOINT/v1/RunTask" \'
echo '  -H "Content-Type: application/x-amz-json-1.1" \'
echo '  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RunTask" \'
echo '  -d "{'
echo '    \"cluster\": \"default\",'
echo '    \"taskDefinition\": \"webapp-with-artifacts\"'
echo '  }"'