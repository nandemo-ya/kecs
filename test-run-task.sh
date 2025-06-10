#!/bin/bash
set -e

echo "Starting KECS in background..."
./bin/kecs server --data-dir /tmp/kecs-test &
KECS_PID=$!

# Wait for KECS to start
echo "Waiting for KECS to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:8081/health >/dev/null; then
        echo "KECS is ready!"
        break
    fi
    sleep 1
done

# Create a cluster
echo "Creating test cluster..."
curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/x-amz-json-1.1" \
    -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster" \
    -d '{"clusterName": "test-cluster"}' | jq .

# Register task definition
echo "Registering task definition..."
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

# Try to run task
echo "Running task..."
RUN_TASK_RESPONSE=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/x-amz-json-1.1" \
    -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RunTask" \
    -d '{
        "cluster": "test-cluster",
        "taskDefinition": "test-simple",
        "count": 1,
        "launchType": "EC2"
    }')

echo "Run task response:"
echo "$RUN_TASK_RESPONSE" | jq .

# Cleanup
echo "Cleaning up..."
kill $KECS_PID
wait $KECS_PID 2>/dev/null || true

echo "Test complete!"