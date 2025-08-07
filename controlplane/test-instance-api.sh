#\!/bin/bash

# Test instance API by simulating what TUI does

echo "Testing instance list from admin server..."

# Start a temporary admin server
../bin/kecs controlplane --admin-port 8083 &
ADMIN_PID=$\!
sleep 2

# Test instance list endpoint
echo "Calling /api/instances..."
curl -s http://localhost:8083/api/instances | jq

# Kill the admin server
kill $ADMIN_PID 2>/dev/null

echo "Done"
