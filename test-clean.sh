#!/bin/bash
echo "=== Cleaning up KECS data ==="

# Kill any running processes
echo "Killing any running KECS processes..."
pkill -f "kecs server" || echo "No KECS processes found"

# Remove test database
echo -e "\nRemoving test database..."
rm -rf /tmp/kecs-test
ls -la /tmp/ | grep kecs || echo "No kecs directories in /tmp"

# Check default database location
echo -e "\nChecking default database location..."
if [ -d ~/.kecs/data ]; then
    echo "Found default data directory:"
    ls -la ~/.kecs/data/
    echo "Moving to backup..."
    mkdir -p ~/.kecs/data-backups
    mv ~/.kecs/data ~/.kecs/data-backups/backup-$(date +%s)
    echo "Default data directory backed up"
else
    echo "No default data directory found"
fi

echo -e "\n=== Cleanup complete ===\n"

# Now run a fresh test
echo "Starting fresh KECS instance..."
./bin/kecs server --data-dir /tmp/kecs-test-fresh --log-level debug &
KECS_PID=$!

# Wait for startup
echo "Waiting for KECS to start..."
for i in {1..30}; do
    if curl -s http://localhost:8081/health >/dev/null 2>&1; then
        echo "KECS is ready!"
        break
    fi
    sleep 1
done

# Register a task definition
echo -e "\nRegistering task definition..."
RESPONSE=$(curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/x-amz-json-1.1" \
    -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition" \
    -d '{
        "family": "test-clean",
        "containerDefinitions": [{
            "name": "busybox",
            "image": "busybox:latest",
            "memory": 128,
            "essential": true
        }],
        "requiresCompatibilities": ["EC2"],
        "networkMode": "bridge"
    }')

echo "Response:"
echo "$RESPONSE" | jq '.taskDefinition | {family, revision, taskDefinitionArn}'

# Check the ARN
ARN=$(echo "$RESPONSE" | jq -r '.taskDefinition.taskDefinitionArn')
echo -e "\nTask Definition ARN: $ARN"

if [[ "$ARN" == *"ap-northeast-1"* ]]; then
    echo "✓ SUCCESS: Region is correct!"
else
    echo "✗ FAIL: Region is incorrect (should be ap-northeast-1)"
fi

# Cleanup
echo -e "\nCleaning up..."
kill $KECS_PID
wait $KECS_PID 2>/dev/null || true

echo -e "\nTest complete!"