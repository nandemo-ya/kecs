Build KECS and start it running in the background. The server will run on port 8080 (API) and 8081 (Admin).

Execute:
```bash
# Find project root
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || echo ".")

echo "Building KECS..."
(cd "$PROJECT_ROOT" && make build) || { echo "Build failed"; exit 1; }

echo "Starting KECS in background..."
nohup "$PROJECT_ROOT/bin/kecs" server > kecs.log 2>&1 &
echo $! > kecs.pid

echo "KECS started with PID: $(cat kecs.pid)"
echo "API Server: http://localhost:8080"
echo "Admin Server: http://localhost:8081"
echo "Logs: tail -f kecs.log"
```