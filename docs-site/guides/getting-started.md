# Getting Started

Welcome to KECS! This guide will help you get up and running with a local ECS-compatible environment in minutes.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Docker**: Required for running containers and k3d
- **kubectl** (optional): For direct Kubernetes cluster interaction
- **AWS CLI** (recommended): For interacting with KECS APIs

## Installation

### macOS (Homebrew)

```bash
brew install nandemo-ya/tap/kecs
```

### Linux/macOS (Direct Download)

```bash
# Download the latest release
curl -L https://github.com/nandemo-ya/kecs/releases/latest/download/kecs-$(uname -s)-$(uname -m) -o kecs
chmod +x kecs
sudo mv kecs /usr/local/bin/
```

### From Source

```bash
# Clone the repository
git clone https://github.com/nandemo-ya/kecs.git
cd kecs

# Build the binary
make build

# Install to system path
sudo mv bin/kecs /usr/local/bin/
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
# Start with a specific instance name
kecs start --instance dev

# Start with custom ports  
kecs start --instance staging --api-port 8080 --admin-port 8081
```

### Check Status

```bash
# Get cluster information
kecs cluster info

# Example output:
# KECS Cluster Information:
# ========================
# Instance: kecs-brave-wilson
# Status: Running
# API Endpoint: http://localhost:5373
# Admin Endpoint: http://localhost:8081
# Kubernetes Context: k3d-kecs-brave-wilson
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
kecs cluster list

# Example output:
# KECS Instances:
# ===============
# • kecs-dev (Running)
# • kecs-staging (Running)
# • kecs-prod-test (Stopped)
```

### View Logs

```bash
# Stream logs from control plane
kecs logs -f

# Show last 100 lines
kecs logs --tail 100
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
kecs tui

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
4. [Set up Hot Reload Development](/development/hot-reload)
5. [Explore Examples](https://github.com/nandemo-ya/kecs/tree/main/examples)

## Getting Help

- Check the [Troubleshooting Guide](/guides/troubleshooting)
- Browse [GitHub Issues](https://github.com/nandemo-ya/kecs/issues)
- Ask in [Discussions](https://github.com/nandemo-ya/kecs/discussions)
- Review [API Documentation](/api/)