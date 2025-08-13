Build KECS Docker image and run it as a container. The container will expose ports 8080 (API) and 8081 (Admin).

Execute:
```bash
# Find project root
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || echo ".")

echo "Building KECS Docker image..."
(cd "$PROJECT_ROOT" && make docker-build) || { echo "Docker build failed"; exit 1; }

echo "Starting KECS container..."
docker run -d \
  --name kecs \
  -p 8080:8080 \
  -p 8081:8081 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $HOME/.kube:/home/nonroot/.kube:ro \
  -e KUBECONFIG=/home/nonroot/.kube/config \
  ghcr.io/nandemo-ya/kecs:latest

if [ $? -eq 0 ]; then
  echo "KECS container started successfully"
  echo "Container ID: $(docker ps -q -f name=kecs)"
  echo "API Server: http://localhost:8080"
  echo "Admin Server: http://localhost:8081"
  echo "Logs: docker logs -f kecs"
else
  echo "Failed to start KECS container"
  exit 1
fi
```