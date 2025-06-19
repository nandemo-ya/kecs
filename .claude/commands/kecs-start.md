Build KECS and start it running in the background. The server will run on port 8080 (API) and 8081 (Admin).

Execute:
```bash
echo "Building KECS..."
cd .. && make build || { echo "Build failed"; exit 1; }

echo "Starting KECS in background..."
cd controlplane && nohup ../bin/kecs server > kecs.log 2>&1 &
echo $! > kecs.pid

echo "KECS started with PID: $(cat kecs.pid)"
echo "API Server: http://localhost:8080"
echo "Admin Server: http://localhost:8081"
echo "Logs: tail -f kecs.log"
```