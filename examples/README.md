# KECS Examples

This directory contains example configurations for testing KECS (Kubernetes-based ECS Compatible Service) functionality.

## Directory Structure

### `/task-definitions/`
Contains ECS task definition examples for various scenarios:

- **`nginx-fargate.json`** - Simple nginx container with Fargate compatibility
- **`webapp-ec2.json`** - Node.js web application with EC2 compatibility
- **`multi-container.json`** - Multiple containers in a single task
- **`batch-job.json`** - Batch processing task configuration
- **`test-new.json`** & **`test-deregister.json`** - Testing configurations

### `/services/`
Contains ECS service definition examples for deployment scenarios:

- **`simple-service.json`** - Minimal service configuration for quick testing
- **`nginx-fargate-service.json`** - Production-ready web service with load balancing
- **`webapp-ec2-service.json`** - EC2-based service with placement strategies
- **`redis-daemon-service.json`** - Daemon service that runs on every host
- **`microservice-with-service-connect.json`** - Advanced microservices configuration

## Quick Start

### 1. Start KECS Server
```bash
# From the project root
make run
```

### 2. Create a Cluster
```bash
aws ecs create-cluster \
  --endpoint-url http://localhost:8080 \
  --cluster-name default
```

### 3. Register a Task Definition
```bash
aws ecs register-task-definition \
  --endpoint-url http://localhost:8080 \
  --cli-input-json file://examples/task-definitions/nginx-fargate.json
```

### 4. Create a Service
```bash
aws ecs create-service \
  --endpoint-url http://localhost:8080 \
  --cli-input-json file://examples/services/simple-service.json
```

### 5. Run a Task
```bash
aws ecs run-task \
  --endpoint-url http://localhost:8080 \
  --cluster default \
  --task-definition nginx-fargate
```

## Testing Container Deployment

The examples in this directory are designed to test the complete container deployment workflow:

1. **Task Definitions** → **Kubernetes Pod Templates**
2. **Services** → **Kubernetes Deployments**
3. **Tasks** → **Kubernetes Pods**

Each example demonstrates different ECS features and how they map to Kubernetes resources.

## Prerequisites

- KECS server running (port 8080)
- Kubernetes cluster access (kind, minikube, etc.)
- AWS CLI configured to use KECS endpoint

For more detailed information, see the README files in each subdirectory.