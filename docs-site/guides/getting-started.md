# Getting Started

Welcome to KECS! This guide will help you get up and running with a local ECS-compatible environment in minutes.

## Prerequisites

KECS requires a Docker-compatible environment to manage k3d clusters:
- **Docker Desktop** (macOS, Windows, Linux)
- **Rancher Desktop** (macOS, Windows, Linux)
- **OrbStack** (macOS)
- **Colima** (macOS, Linux)
- Or any other Docker-compatible runtime

Optional but recommended:
- **kubectl**: For direct Kubernetes cluster interaction
- **AWS CLI**: For interacting with KECS APIs

## Installation

### Using Homebrew (macOS/Linux)

```bash
# Install KECS
brew tap nandemo-ya/kecs
brew install kecs

# Verify installation
kecs version
```

### From Source

```bash
git clone https://github.com/nandemo-ya/kecs.git
cd kecs
make build
```

## Starting KECS

### Basic Start

```bash
# Start KECS with default settings
kecs start

# This will:
# 1. Create a k3d cluster named 'kecs-<random>'
# 2. Deploy KECS control plane
# 3. Deploy LocalStack for AWS services
# 4. Set up Traefik gateway for routing
# 5. Make everything available on port 5373
```

### Custom Instance

```bash
# Start with a specific instance name (uses default ports 5373/5374)
kecs start --instance dev

# Start with custom ports
kecs start --instance staging --api-port 5383 --admin-port 5384
```

### Check Status

```bash
# Check if KECS is running
kubectl get pods -n kecs-system

# Example output:
# NAME                                   READY   STATUS    RESTARTS   AGE
# kecs-server-7f8b9c5d4-x2klm     1/1     Running   0          5m
# localstack-6d7f8c9b5-p3qrs            1/1     Running   0          5m
```

## Using KECS

### Configure AWS CLI

```bash
# Set the endpoint URL for all AWS commands
export AWS_ENDPOINT_URL=http://localhost:5373

# Configure dummy credentials (required but not validated)
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1
```

### Create Your First Cluster

```bash
# Create an ECS cluster
aws ecs create-cluster --cluster-name my-first-cluster

# List clusters
aws ecs list-clusters

# Describe the cluster
aws ecs describe-clusters --clusters my-first-cluster
```

### Deploy a Simple Task

```bash
# Register a task definition
cat > task-definition.json << 'EOF'
{
  "family": "hello-world",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "hello",
      "image": "hello-world",
      "essential": true
    }
  ]
}
EOF

aws ecs register-task-definition --cli-input-json file://task-definition.json

# Run the task
aws ecs run-task \
  --cluster my-first-cluster \
  --task-definition hello-world \
  --launch-type FARGATE
```

## Managing KECS Instances

### List Running Instances

```bash
# Show all KECS instances
kecs list

# Example output:
# INSTANCE           STATUS    API PORT    ADMIN PORT
# kecs-dev           Running   5373        5374
# kecs-staging       Running   5383        5384
```

### Stop KECS

```bash
# Stop specific instance
kecs stop --instance dev

# Stop with interactive selection
kecs stop

# Stop specific instance
kecs stop --instance myinstance
```

## Advanced Configuration

### Environment Variables

```bash
# Custom k3d cluster name
export KECS_CLUSTER_NAME=my-kecs

# Custom namespace
export KECS_NAMESPACE=kecs-custom

# Debug logging
export KECS_LOG_LEVEL=debug
```

### Persistent Data

KECS stores instance data in `~/.kecs/instances/`. Each instance has:
- Configuration file
- k3d kubeconfig
- DuckDB database for ECS resources

### Using with LocalStack

KECS includes LocalStack, providing these AWS services:
- IAM (automatic `ecsTaskExecutionRole` creation)
- Secrets Manager
- Systems Manager (SSM)
- S3
- CloudWatch Logs
- And more...

All services are available through the same endpoint (port 5373):

```bash
# ECS (KECS)
aws ecs list-clusters

# S3 (LocalStack)
aws s3 mb s3://my-bucket
aws s3 ls

# Secrets Manager (LocalStack)
aws secretsmanager create-secret --name my-secret --secret-string "password"
```

## Interactive TUI

KECS includes an interactive Terminal User Interface:

```bash
# Launch TUI
kecs

# Features:
# - Browse clusters, services, and tasks
# - View real-time status updates
# - Check logs and events
# - Manage resources interactively
```

## Troubleshooting

### Port Already in Use

```bash
# Check what's using port 5373
lsof -i :5373

# Use different ports
kecs start --api-port 8080 --admin-port 8081
```

### k3d Issues

```bash
# Check k3d clusters
k3d cluster list

# Check Docker
docker ps

# Clean up orphaned clusters
k3d cluster delete kecs-<instance-name>
```

### Reset Everything

```bash
# Stop all KECS instances
kecs stop --all

# Clean up all data
rm -rf ~/.kecs/instances/

# Remove all k3d clusters
k3d cluster delete --all
```

## Next Steps

Now that you have KECS running:

1. [Deploy a Multi-Container Application](/guides/services)
2. [Work with Load Balancers](/guides/elbv2-integration)
3. [Use the Interactive TUI](/guides/tui-interface)
4. [Explore Examples](https://github.com/nandemo-ya/kecs/tree/main/examples)

## Getting Help

- Check the [Troubleshooting Guide](/guides/troubleshooting)
- Browse [GitHub Issues](https://github.com/nandemo-ya/kecs/issues)