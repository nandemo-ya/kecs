Build KECS Docker image with embedded Web UI and run it as a container. The container will expose ports 8080 (API with Web UI) and 8081 (Admin).

Execute:
```bash
# Find project root
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || echo ".")

echo "Building Web UI..."
(cd "$PROJECT_ROOT" && make build-webui) || { echo "Web UI build failed"; exit 1; }

echo "Building KECS Docker image with Web UI..."
(cd "$PROJECT_ROOT" && docker build -f controlplane/Dockerfile.production -t ghcr.io/nandemo-ya/kecs:latest-ui .) || { echo "Docker build failed"; exit 1; }

echo "Starting KECS container with Web UI..."
docker run -d \
  --name kecs-ui \
  -p 8080:8080 \
  -p 8081:8081 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $HOME/.kube:/home/nonroot/.kube:ro \
  -e KUBECONFIG=/home/nonroot/.kube/config \
  ghcr.io/nandemo-ya/kecs:latest-ui

if [ $? -eq 0 ]; then
  echo "KECS container with Web UI started successfully"
  echo "Container ID: $(docker ps -q -f name=kecs-ui)"
  echo "Web UI: http://localhost:8080"
  echo "API Server: http://localhost:8080"
  echo "Admin Server: http://localhost:8081"
  echo "Logs: docker logs -f kecs-ui"
else
  echo "Failed to start KECS container"
  exit 1
fi
```