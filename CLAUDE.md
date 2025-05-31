# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

KECS (Kubernetes-based ECS Compatible Service) is a standalone service that provides Amazon ECS compatible APIs running on Kubernetes. It enables a fully local ECS-compatible environment that operates independently of AWS environments.

## Common Development Commands

### Building and Running
```bash
make build          # Build the binary to bin/kecs
make run            # Build and run the application
make all            # Clean, format, vet, test, and build
```

### Testing and Code Quality
```bash
make test           # Run tests with race detection
make test-coverage  # Run tests with coverage report
make vet            # Run go vet
make fmt            # Format code with gofmt
```

### Docker Operations
```bash
make docker-build   # Build Docker image
make docker-push    # Build and push Docker image
```

### Development Workflow
```bash
make deps           # Download and verify dependencies
make clean          # Clean build artifacts and coverage files
```

## Architecture Overview

KECS implements a clean architecture with the following key components:

### Dual Server Design
- **API Server (port 8080)**: Handles ECS-compatible API requests at `/v1/<action>` endpoints
- **Admin Server (port 8081)**: Provides operational endpoints like `/health`

### Directory Structure
- `cmd/controlplane/`: Entry point for the control plane binary
- `internal/controlplane/cmd/`: CLI command implementations using Cobra
- `internal/controlplane/api/`: ECS API endpoint implementations
- `internal/controlplane/admin/`: Admin server with health checks
- `docs/adr/records/`: Architectural Decision Records

### API Implementation Pattern
Each ECS resource type has its own file in `internal/controlplane/api/` with:
- Request/Response struct definitions matching AWS ECS API
- Handler function registered in `api/server.go`
- Current implementations return mock responses with TODO comments

### Key Architectural Decisions
- Uses standard `net/http` for HTTP servers
- Graceful shutdown with 10-second timeout
- Context-based cancellation throughout
- Planned: DuckDB for persistence, Kubernetes client for container operations

## Development Notes

When implementing new ECS API endpoints:
1. Add type definitions to the appropriate file in `internal/controlplane/api/`
2. Implement the handler function following existing patterns
3. Register the handler in `api/server.go`
4. Follow AWS ECS API naming conventions exactly

The codebase is in early development with mock implementations. Actual business logic for Kubernetes integration is pending implementation.