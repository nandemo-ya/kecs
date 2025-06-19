Build KECS with embedded Web UI and start it running in the background. The server will run on port 8080 (API with Web UI) and 8081 (Admin).

Execute:
```bash
# Find project root by looking for go.mod
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || echo ".")

echo "Building Web UI..."
(cd "$PROJECT_ROOT" && make build-webui) || { echo "Web UI build failed"; exit 1; }

echo "Building KECS with embedded Web UI..."
(cd "$PROJECT_ROOT" && make build-with-ui) || { echo "Build failed"; exit 1; }

echo "Starting KECS with Web UI in background..."
nohup "$PROJECT_ROOT/bin/kecs" server > kecs.log 2>&1 &
echo $! > kecs.pid

echo "KECS with Web UI started with PID: $(cat kecs.pid)"
echo "Web UI: http://localhost:8080"
echo "API Server: http://localhost:8080"
echo "Admin Server: http://localhost:8081"
echo "Logs: tail -f kecs.log"
```