# KECS Project Overview

## Purpose
KECS (Kubernetes-based ECS Compatible Service) is a standalone service that provides Amazon ECS compatible APIs running on Kubernetes. It enables a fully local ECS-compatible environment that operates independently of AWS environments.

## Tech Stack
- **Language**: Go (1.24.3)
- **Frameworks**: 
  - Kubernetes client-go for K8s integration
  - Standard net/http for HTTP servers
  - Cobra for CLI commands
  - Viper for configuration
- **Storage**: DuckDB for persistence
- **Testing**: Ginkgo (BDD-style testing framework) with Gomega matchers
- **UI**: Bubble Tea for TUI components
- **Container Runtime**: Docker with k3d for development
- **Documentation**: VitePress for static site generation

## Architecture
- **Dual Server Design**:
  - API Server (port 8080): ECS-compatible API requests at `/v1/<action>` endpoints
  - Admin Server (port 8081): Operational endpoints like `/health`
- **Clean Architecture** with separation of concerns
- **Context-based cancellation** throughout
- **Graceful shutdown** with 10-second timeout
- **WebSocket support** for real-time updates

## Key Features
- ECS API compatibility (Clusters, Services, Tasks, Task Definitions, Tags, Attributes)
- Hot reload development workflow
- Multiple instance support
- Docker container management commands (similar to kind/k3d)
- MCP Server for AI assistant integration
- LocalStack integration for AWS services simulation