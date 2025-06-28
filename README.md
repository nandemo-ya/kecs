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

## ⚠️ Security Notice - Please Read

**KECS is designed exclusively for local development and CI environments.**

KECS requires access to the Docker daemon to create and manage local Kubernetes clusters. This provides significant capabilities:

- **Full access to Docker daemon** (equivalent to root access)
- **Ability to create, modify, and delete containers**
- **Access to the host filesystem through volume mounts**
- **Network configuration capabilities**

### ✅ Supported Environments
- Local development machines  
- CI/CD pipelines (GitHub Actions, GitLab CI, Jenkins, etc.)  
- Isolated test environments  

### ❌ NOT Supported/Unsafe
- Production environments  
- Public-facing deployments  
- Multi-tenant systems  
- Environments with untrusted users  

**By using KECS, you acknowledge that:**
1. You understand the security implications
2. You trust KECS with Docker daemon access
3. You will only use KECS in supported environments

### First Run

On first run, KECS will display a security disclaimer and request acknowledgment. To skip this in automated environments:

```bash
# Set environment variable
export KECS_SECURITY_ACKNOWLEDGED=true

# Or add to config file
echo "features.securityAcknowledged: true" >> ~/.kecs/config.yaml
```

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

## Quick Start

### Running KECS in a Container

The easiest way to run KECS is using the container-based execution:

```bash
# Start KECS in a container
kecs start

# Check status
kecs status

# View logs
kecs logs -f

# Stop KECS
kecs stop
```

### Running Multiple Instances

KECS supports running multiple instances with different configurations:

```bash
# Start with custom name and ports
kecs start --name dev --api-port 8080 --admin-port 8081
kecs start --name staging --api-port 8090 --admin-port 8091

# Or use auto-port assignment
kecs start --name test --auto-port

# List all instances
kecs instances list
```

## Installation

### From Source

```bash
git clone https://github.com/nandemo-ya/kecs.git
cd kecs/controlplane
make build
```

### Using Docker

```bash
docker pull ghcr.io/nandemo-ya/kecs:latest
```

## Usage

### Required Permissions

KECS requires the following permissions to function properly:

- **Docker Socket Access**: When running in a container, mount `/var/run/docker.sock`
- **Network Ports**: Default ports 8080 (API) and 8081 (Admin)
- **Local Storage**: For data persistence (default: `~/.kecs/data`)

```bash
# Container mode requires Docker socket mounting
docker run -v /var/run/docker.sock:/var/run/docker.sock \
           -p 8080:8080 -p 8081:8081 \
           ghcr.io/nandemo-ya/kecs:latest

# Binary mode requires Docker to be installed and accessible
kecs server
```

### Container Commands

KECS provides container-based execution similar to tools like kind and k3d, supporting both Docker and containerd runtimes:

#### Start Command

Starts KECS server in a container:

```bash
kecs start [flags]

Flags:
  --name string        Container name (default "kecs-server")
  --image string       Container image to use (default "ghcr.io/nandemo-ya/kecs:latest")
  --api-port int       API server port (default 8080)
  --admin-port int     Admin server port (default 8081)
  --data-dir string    Data directory (default "~/.kecs/data")
  -d, --detach         Run container in background (default true)
  --local-build        Build and use local image
  --config string      Path to configuration file
  --auto-port          Automatically find available ports
  --runtime string     Container runtime to use (docker, containerd, or auto)
```

Examples:

```bash
# Start with default settings (auto-detects runtime)
kecs start

# Start with custom ports
kecs start --api-port 9080 --admin-port 9081

# Start with local build
kecs start --local-build

# Start using configuration file
kecs start --config ~/.kecs/instances.yaml staging

# Use specific runtime
kecs start --runtime docker
kecs start --runtime containerd

# Works with k3d/k3s environments
kecs start --runtime containerd  # Automatically uses k3s socket
```

#### Stop Command

Stops and removes KECS container:

```bash
kecs stop [flags]

Flags:
  --name string     Container name (default "kecs-server")
  -f, --force       Force stop without graceful shutdown
```

#### Status Command

Shows KECS container status:

```bash
kecs status [flags]

Flags:
  --name string     Container name (empty for all KECS containers)
  -a, --all         Show all containers including stopped ones
```

#### Logs Command

Displays logs from KECS container:

```bash
kecs logs [flags]

Flags:
  --name string        Container name (default "kecs-server")
  -f, --follow         Follow log output
  --tail string        Number of lines to show from the end (default "all")
  -t, --timestamps     Show timestamps
```

### Multiple Instances Management

KECS supports running multiple instances with the `instances` command:

#### List Instances

```bash
kecs instances list [--config file]
```

Shows all configured and running instances with their status, ports, and configuration.

#### Start All Instances

```bash
kecs instances start-all [--config file]
```

Starts all instances marked with `autoStart: true` in the configuration file.

#### Stop All Instances

```bash
kecs instances stop-all
```

Stops all running KECS instances.

### Configuration File

KECS supports YAML configuration files for managing multiple instances:

```yaml
# ~/.kecs/instances.yaml
defaultInstance: dev

instances:
  - name: dev
    image: ghcr.io/nandemo-ya/kecs:latest
    ports:
      api: 8080
      admin: 8081
    dataDir: ~/.kecs/instances/dev/data
    autoStart: true
    env:
      KECS_LOG_LEVEL: debug
    labels:
      environment: development

  - name: staging
    image: ghcr.io/nandemo-ya/kecs:latest
    ports:
      api: 8090
      admin: 8091
    dataDir: ~/.kecs/instances/staging/data
    autoStart: true
    env:
      KECS_LOG_LEVEL: info
    labels:
      environment: staging

  - name: test
    image: ghcr.io/nandemo-ya/kecs:latest
    ports:
      api: 8100
      admin: 8101
    dataDir: ~/.kecs/instances/test/data
    autoStart: false
    env:
      KECS_TEST_MODE: "true"
    labels:
      environment: test
```

### Container Runtime Support

KECS supports multiple container runtimes:

#### Docker
- Docker Desktop
- Docker Engine
- Default runtime for most environments

#### Containerd
- k3s/k3d environments
- Rancher Desktop (containerd mode)
- Kind clusters
- Standard Kubernetes nodes

#### Auto-detection
KECS automatically detects the available runtime:
1. Checks for Docker first (backward compatibility)
2. Falls back to containerd if Docker is not available
3. Automatically finds k3s containerd socket at `/run/k3s/containerd/containerd.sock`

### Traditional Server Mode

You can also run KECS directly without containers:

```bash
# Run the server
kecs server

# Or with custom configuration
kecs server --port 8080 --admin-port 8081

# Run server (API-only mode)
kecs server
```

### Docker Deployment

#### Using Docker Compose

```bash
# Run KECS
docker compose up
```

#### Using KECS CLI

```bash
# Start API container
kecs start --name kecs-api --api-port 8080
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

## Security Considerations

KECS is a development tool that requires elevated permissions:

1. **Docker Daemon Access**: KECS needs access to the Docker daemon to create and manage k3d clusters. This is equivalent to root access on Linux systems.

2. **Network Access**: KECS creates virtual networks and exposes ports for service communication.

3. **Not for Production**: KECS is explicitly NOT designed for production use. It lacks the security features required for multi-tenant or public-facing deployments.

### Comparison with Similar Tools

| Tool | Purpose | Required Permissions |
|------|---------|---------------------|
| Docker Desktop | Container runtime | Root/Admin privileges |
| kind | Local Kubernetes | Docker socket access |
| k3d | Local k3s clusters | Docker socket access |
| LocalStack | AWS service emulation | Network ports |
| **KECS** | ECS emulation | Docker socket + Network ports |

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.