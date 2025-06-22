#\!/bin/bash
echo "Starting test..."

# Start the test in background
ginkgo -v --focus "should create a service and run nginx containers" 2>&1 &
TEST_PID=$\!

# Wait a bit for container to start
sleep 5

# Find KECS container
KECS_CONTAINER=$(docker ps | grep "kecs:test" | awk '{print $1}')
if [ -n "$KECS_CONTAINER" ]; then
    echo "Found KECS container: $KECS_CONTAINER"
    echo "=== KECS Container Logs ==="
    docker logs -f $KECS_CONTAINER 2>&1 &
    LOGS_PID=$\!
fi

# Wait for test to complete
wait $TEST_PID
TEST_RESULT=$?

# Stop log following
if [ -n "$LOGS_PID" ]; then
    kill $LOGS_PID 2>/dev/null || true
fi

exit $TEST_RESULT
