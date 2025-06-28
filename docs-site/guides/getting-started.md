# Getting Started

Welcome to KECS! This guide will help you get up and running with a local ECS-compatible environment.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.21+**: Required for building from source
- **Docker**: Required for running containers
- **Kind**: For local Kubernetes cluster (optional but recommended)
- **kubectl**: For interacting with Kubernetes (optional)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/nandemo-ya/kecs.git
cd kecs

# Build the binary
make build

# The binary will be available at ./bin/kecs
```

### Using Docker

```bash
# Run KECS using Docker
docker run -p 8080:8080 -p 8081:8081 ghcr.io/nandemo-ya/kecs:latest
```

## Starting KECS

### Local Development

```bash
# Start the server
./bin/kecs server

# Or use make
make run
```

### With Kind

```bash
# Create a Kind cluster (if not exists)
kind create cluster --name kecs-dev

# Start KECS with Kind integration
./bin/kecs server --kubernetes-mode=kind
```

## Verifying Installation

### Health Check

```bash
# Check if KECS is running
curl http://localhost:8081/health
```

### API Test

```bash
# List clusters
curl -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -d '{}'
```

## Next Steps

Now that you have KECS running, you can:

1. [Create your first cluster](/guides/quick-start)
2. [Deploy a service](/guides/services)
3. [Learn about Task Definitions](/guides/task-definitions)

## Troubleshooting

### Port Already in Use

If you see an error about ports being in use:

```bash
# Check what's using port 8080
lsof -i :8080

# Run KECS on different ports
./bin/kecs server --api-port=9080 --admin-port=9081
```

### Kind Connection Issues

If KECS can't connect to Kind:

```bash
# Ensure Kind cluster is running
kind get clusters

# Check kubectl context
kubectl config current-context
```

For more troubleshooting tips, see our [Troubleshooting Guide](/guides/troubleshooting).