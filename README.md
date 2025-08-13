<div align="center">
  <img src="./assets/logo.svg" alt="KECS Logo" width="200" />
  
  # KECS
  
  **Kubernetes-based ECS Compatible Service**
</div>

<div align="center">

[![CI/CD Pipeline](https://github.com/nandemo-ya/kecs/actions/workflows/ci.yml/badge.svg)](https://github.com/nandemo-ya/kecs/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/nandemo-ya/kecs/branch/main/graph/badge.svg)](https://codecov.io/gh/nandemo-ya/kecs)
[![Go Version](https://img.shields.io/badge/Go-1.24.3-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Release](https://img.shields.io/github/release/nandemo-ya/kecs.svg)](https://github.com/nandemo-ya/kecs/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/nandemo-ya/kecs)](https://goreportcard.com/report/github.com/nandemo-ya/kecs)
[![GoDoc](https://pkg.go.dev/badge/github.com/nandemo-ya/kecs)](https://pkg.go.dev/github.com/nandemo-ya/kecs)

</div>

## Development Tool Notice

**KECS is designed exclusively for local development and CI environments.**

KECS runs its control plane inside a k3d cluster, providing a clean and isolated environment for ECS workloads. The CLI manages k3d cluster lifecycle using the Docker API.

### ✅ Supported Environments
- Local development machines  
- CI/CD pipelines (GitHub Actions, GitLab CI, Jenkins, etc.)  
- Isolated test environments  

### ❌ NOT Supported
- Production environments  
- Public-facing deployments  
- Multi-tenant systems

## Overview

KECS (Kubernetes-based ECS Compatible Service) is a standalone service that provides Amazon ECS compatible APIs running on Kubernetes. It enables a fully local ECS-compatible environment that operates independently of AWS environments.

### Key Features

- **ECS API Compatibility**: Provides API endpoints compatible with Amazon ECS
- **Kubernetes Backend**: Leverages Kubernetes for container orchestration
- **Local Execution**: Runs completely locally without AWS dependencies
- **Container Runtime Support**: Works with both Docker and containerd (k3s, k3d, Rancher Desktop)
- **Container-based Background Execution**: Run KECS in containers with simple commands
- **Multiple Instance Support**: Run multiple KECS instances with different configurations
- **CI/CD Integration**: Easily integrates with CI/CD pipelines
- **Built-in LocalStack Integration**: Automatically provides local AWS services (IAM, SSM, Secrets Manager, etc.) for ECS workloads
- **Automatic IAM Role**: Creates `ecsTaskExecutionRole` on startup for pulling images and writing logs

## Quick Start

### Running KECS

KECS runs its control plane inside a k3d cluster, providing better integration and a unified AWS API endpoint:

```bash
# Start KECS
kecs start

# This creates a k3d cluster with:
# - KECS control plane (ECS/ELBv2 APIs)
# - LocalStack (other AWS services)
# - Traefik gateway (unified routing)

# Check status
kecs cluster info

# Stop KECS
kecs stop
```

All AWS APIs are accessible through port 4566:
```bash
export AWS_ENDPOINT_URL=http://localhost:4566
aws ecs list-clusters         # → KECS
aws elbv2 describe-load-balancers  # → KECS  
aws s3 ls                     # → LocalStack
```

### Running Multiple Instances

KECS supports running multiple instances with different configurations:

```bash
# Start with custom instance name and ports
kecs start --instance dev --api-port 8080 --admin-port 8081
kecs start --instance staging --api-port 8090 --admin-port 8091

# Or use auto-generated instance name
kecs start  # Generates a random instance name

# Stop a specific instance
kecs stop --instance dev

# Stop instance with interactive selection
kecs stop  # Shows a list to select from
```

## Installation

### From Source

```bash
git clone https://github.com/nandemo-ya/kecs.git
cd kecs/controlplane
make build
```

### Development Setup

For contributors, we use [Lefthook](https://github.com/evilmartians/lefthook) to ensure code quality with git hooks:

```bash
# Run the setup script (installs Lefthook and configures git hooks)
./scripts/setup-lefthook.sh

# Or install manually
brew install lefthook  # macOS
# or
curl -sSfL https://raw.githubusercontent.com/evilmartians/lefthook/master/install.sh | sh -s -- -b /usr/local/bin  # Linux

# Install git hooks
lefthook install
```

The git hooks will:
- **Pre-commit**: Run unit tests, go fmt, and go vet for changed files
- **Pre-push**: Run the full test suite with race detection

To skip hooks temporarily: `git commit --no-verify` or `git push --no-verify`

### Using Docker

```bash
docker pull ghcr.io/nandemo-ya/kecs:latest
```

## Usage

### Requirements

KECS requires the following to function properly:

- **Docker**: For managing k3d clusters (the CLI uses Docker API)
- **Network Ports**: Port 4566 for unified AWS API endpoint
- **Local Storage**: For data persistence (default: `~/.kecs/data`)

```bash
# Start KECS (creates and manages k3d cluster)
kecs start

# Access AWS APIs through the unified endpoint
export AWS_ENDPOINT_URL=http://localhost:4566
aws ecs list-clusters
```


### Configuration

#### Default Configuration Changes

As of the latest version, KECS has updated its default configuration:

- **LocalStack**: Now **enabled by default** to provide AWS service emulation
- **Traefik**: Now **enabled by default** for advanced routing capabilities

To disable these features (e.g., for testing or lightweight deployments):

```bash
# Via environment variables
export KECS_LOCALSTACK_ENABLED=false
export KECS_FEATURES_TRAEFIK=false

# Or via configuration file
localstack:
  enabled: false
features:
  traefik: false
```

For more details, see the [Configuration Guide](docs/configuration.md).


### Server Mode

You can also run KECS server directly (useful for development):

```bash
# Run the server
kecs server

# Or with custom configuration
kecs server --port 8080 --admin-port 8081
```

### Docker Deployment

#### Using Docker Compose

```bash
# Run KECS
docker compose up
```


#### Building Docker Images

```bash
# Build API image
make docker-build-api
```

## API Endpoints

KECS provides ECS-compatible API endpoints:

- **API Server** (default port 8080): ECS API endpoints at `/v1/<action>`
- **Admin Server** (default port 8081): Health checks at `/health`

## Documentation

- Architectural Decision Records (ADRs) are stored in the `docs/adr/records` directory
- API documentation is available in the `docs/api` directory
- For more detailed documentation, visit our [documentation site](https://nandemo-ya.github.io/kecs/)

## Architecture

KECS uses a modern architecture where the control plane runs inside a k3d cluster:

1. **CLI Tool**: Manages k3d cluster lifecycle (start, stop, status)
2. **Control Plane**: Runs as pods inside the k3d cluster, providing ECS and ELBv2 APIs
3. **Unified Gateway**: Traefik routes AWS API calls to appropriate services (KECS or LocalStack)

### Comparison with Similar Tools

| Tool | Purpose | Architecture |
|------|---------|--------------|
| Docker Desktop | Container runtime | System daemon |
| k3d | Local k3s clusters | CLI + OCI containers |
| LocalStack | AWS service emulation | Standalone container |
| **KECS** | ECS emulation | Control plane in k3d cluster |

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.