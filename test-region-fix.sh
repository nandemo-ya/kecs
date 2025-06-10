#!/bin/bash
set -e

# Clean up any existing processes
pkill -f "kecs server" || true
rm -rf /tmp/kecs-test
echo "Cleaned up database directory"
ls -la /tmp/ | grep kecs || echo "No kecs directories in /tmp"

echo "Starting KECS..."
./bin/kecs server --data-dir /tmp/kecs-test &
KECS_PID=$!

# Wait for KECS to start
echo "Waiting for KECS to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8081/health >/dev/null 2>&1; then
        echo "KECS is ready!"
        break
    fi
    sleep 1
done

# Create a cluster
echo -e "\nCreating test cluster..."
CLUSTER_RESPONSE=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/x-amz-json-1.1" \
    -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster" \
    -d '{"clusterName": "test-cluster"}')

echo "Cluster response:"
echo "$CLUSTER_RESPONSE" | jq .

# Register task definition
echo -e "\nRegistering task definition..."
TASK_DEF_RESPONSE=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/x-amz-json-1.1" \
    -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition" \
    -d '{
        "family": "test-simple",
        "containerDefinitions": [{
            "name": "busybox",
            "image": "busybox:latest",
            "memory": 128,
            "essential": true,
            "command": ["echo", "Hello from KECS"]
        }],
        "requiresCompatibilities": ["EC2"],
        "networkMode": "bridge"
    }')

echo "Task definition response:"
echo "$TASK_DEF_RESPONSE" | jq .

# Extract the task definition ARN to verify the region
TASK_DEF_ARN=$(echo "$TASK_DEF_RESPONSE" | jq -r '.taskDefinition.taskDefinitionArn')
echo -e "\nTask Definition ARN: $TASK_DEF_ARN"

# Check if it matches expected region
if [[ "$TASK_DEF_ARN" == *"ap-northeast-1"* ]]; then
    echo "✓ Region is correct (ap-northeast-1)"
else
    echo "✗ Region is incorrect (expected ap-northeast-1)"
fi

# Try to run task
echo -e "\nRunning task..."
RUN_TASK_RESPONSE=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/x-amz-json-1.1" \
    -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RunTask" \
    -d '{
        "cluster": "test-cluster",
        "taskDefinition": "test-simple",
        "count": 1,
        "launchType": "EC2"
    }' -w "\nHTTP_STATUS:%{http_code}")

# Extract HTTP status
HTTP_STATUS=$(echo "$RUN_TASK_RESPONSE" | grep -o "HTTP_STATUS:[0-9]*" | cut -d: -f2)
RESPONSE_BODY=$(echo "$RUN_TASK_RESPONSE" | sed 's/HTTP_STATUS:[0-9]*//')

echo "Response:"
echo "$RESPONSE_BODY" | jq . 2>/dev/null || echo "$RESPONSE_BODY"
echo "HTTP Status: $HTTP_STATUS"

# Cleanup
echo -e "\nCleaning up..."
kill $KECS_PID
wait $KECS_PID 2>/dev/null || true

echo -e "\nTest complete!"