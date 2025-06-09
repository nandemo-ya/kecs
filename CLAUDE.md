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

#### Testing Framework
KECS uses **Ginkgo** as the primary testing framework for Go tests:
- Ginkgo provides BDD-style testing with descriptive test specifications
- Tests should be written using Ginkgo's `Describe`, `Context`, `It` patterns
- Use Gomega matchers for assertions
- Place test files as `*_test.go` alongside the code they test

Example test structure:
```go
var _ = Describe("ClusterHandler", func() {
    Context("when creating a cluster", func() {
        It("should return existing cluster when name already exists", func() {
            // Test implementation
            Expect(response).To(Equal(expectedResponse))
        })
    })
})
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
- **Web UI (embedded)**: React-based dashboard served at `/` (when enabled)

### Directory Structure
- `cmd/controlplane/`: Entry point for the control plane binary
- `internal/controlplane/cmd/`: CLI command implementations using Cobra
- `internal/controlplane/api/`: ECS API endpoint implementations
- `internal/controlplane/admin/`: Admin server with health checks
- `internal/converters/`: Task definition to Kubernetes resource converters
- `internal/kubernetes/`: Kubernetes client and resource managers
- `internal/storage/`: Storage interfaces and DuckDB implementation
- `docs/adr/records/`: Architectural Decision Records
- `web-ui/`: React/TypeScript Web UI application
- `docs-site/`: VitePress-based documentation site

### API Implementation Pattern
Each ECS resource type has its own file in `internal/controlplane/api/` with:
- Request/Response struct definitions matching AWS ECS API
- Handler function registered in `api/server.go`
- Current implementations return mock responses with TODO comments

### Key Architectural Decisions
- Uses standard `net/http` for HTTP servers
- Graceful shutdown with 10-second timeout
- Context-based cancellation throughout
- DuckDB for persistence (storage layer implemented)
- Kubernetes client for container operations (Kind integration)
- WebSocket support for real-time updates
- Embedded Web UI with conditional compilation

## Web UI Development

KECS includes a modern React/TypeScript Web UI for managing ECS resources:

### Web UI Features
- **Dashboard**: Real-time overview of clusters, services, tasks
- **Resource Management**: Create, view, update services and task definitions
- **Real-time Updates**: WebSocket integration for live status updates
- **Visualization**: Service topology, metrics charts, log viewer
- **Responsive Design**: Works on desktop and mobile devices

### Web UI Build Process
```bash
# Build Web UI (from web-ui directory)
cd web-ui && npm run build

# Build control plane with embedded UI
./scripts/build-webui.sh  # Builds UI and embeds into binary
```

### Web UI Architecture
- React 19 with TypeScript for type safety
- React Router for SPA navigation
- Custom hooks for API integration and WebSocket connections
- Component-based architecture with reusable UI elements
- Real-time data synchronization with control plane

## Documentation Site

KECS documentation is built using VitePress (SSG - Static Site Generator):

### Documentation Development
```bash
# Install dependencies and start dev server
cd docs-site
npm install
npm run docs:dev  # Access at http://localhost:5173
```

### Documentation Build
```bash
# Build documentation site
./scripts/build-docs.sh  # Output in docs-site/.vitepress/dist/

# Or manually from docs-site directory
cd docs-site && npm run docs:build
```

### Documentation Structure
- `docs-site/`: VitePress documentation root
  - `.vitepress/config.js`: Site configuration
  - `index.md`: Home page
  - `guides/`: User guides and tutorials
  - `api/`: API reference documentation
  - `architecture/`: Architecture documentation
  - `deployment/`: Deployment guides
  - `development/`: Developer documentation

## Development Notes

### Development Workflow Rules
1. **Always create a feature branch before starting implementation work**
   ```bash
   git checkout -b feat/feature-name  # For features
   git checkout -b fix/bug-name       # For bug fixes
   ```

2. **Run all tests before creating a Pull Request**
   ```bash
   # Control plane tests (using Ginkgo)
   cd controlplane && ginkgo -r
   # Or using go test
   cd controlplane && go test ./...
   
   # Web UI tests
   cd web-ui && npm test
   ```

3. **Only create PR after all tests pass**
   - Both controlplane and web-ui unit tests must pass
   - Fix any failing tests before proceeding with PR

4. **Test CI/CD changes locally with act before committing**
   ```bash
   # Test GitHub Actions workflow locally
   act -W .github/workflows/workflow-name.yml -j job-name --container-architecture linux/amd64
   ```
   - ALWAYS verify CI changes work locally before pushing
   - This prevents breaking the CI pipeline for other developers

### When implementing new ECS API endpoints:
1. Add type definitions to the appropriate file in `internal/controlplane/api/`
2. Implement the handler function following existing patterns
3. Register the handler in `api/server.go`
4. Follow AWS ECS API naming conventions exactly
5. Update Web UI types in `web-ui/src/types/` if needed
6. Add corresponding UI components if user-facing
7. Write Ginkgo tests for the new endpoint covering:
   - Success cases
   - Error cases
   - Edge cases (e.g., idempotency, empty inputs)
   - AWS ECS compatibility behavior

### Current Implementation Status
- **Implemented**: Clusters, Services, Tasks, Task Definitions, Tags, Attributes
- **Storage**: DuckDB integration for persistence
- **Kubernetes**: Task converter with secret management
- **Web UI**: Dashboard, detail views, WebSocket support
- **MCP Server**: TypeScript-based Model Context Protocol server for AI assistant integration
- **In Progress**: Full Kubernetes task lifecycle management

## MCP Server Development

KECS includes a Model Context Protocol (MCP) server for AI assistant integration:

### MCP Server Overview
The MCP server enables AI assistants like Claude to interact with KECS through natural language:
- Located in `mcp-server/` directory
- Built with TypeScript and the official MCP SDK
- Provides tools for all ECS operations (clusters, services, tasks, task definitions)
- Supports Claude Desktop and Claude Code (VS Code) integration

### MCP Server Development
```bash
# Install dependencies
cd mcp-server
npm install

# Development mode with hot-reloading
npm run dev

# Build for production
npm run build

# Run tests
npm test
```

### MCP Server Configuration
- **Claude Desktop**: Configure in `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Claude Code**: Configure in `~/Library/Application Support/Claude/claude_code_config.json`
- Documentation available in `mcp-server/docs/` and `docs-site/docs/mcp-server/`

## Scenario Tests

### Running Scenario Tests
```bash
# Navigate to scenario tests directory
cd tests/scenarios

# Run all scenario tests
make test

# Run only cluster tests
make test-cluster

# Run with debug logging
make test-verbose

# Run specific test
make test-one TEST=TestClusterCreateAndDelete
```

### Test Structure
- **Phase 1 (Foundation)**: Basic cluster management tests - COMPLETED
  - Testcontainers integration for isolated test environments
  - AWS CLI wrapper for ECS operations
  - Test helpers and utilities
- **Phase 2**: Task definition and basic service operations
- **Phase 3**: Task lifecycle and status tracking
- **Phase 4**: Advanced service operations (rolling updates, scaling)
- **Phase 5**: Failure scenarios
- **Phase 6**: ecspresso integration
- **Phase 7**: Performance tests
- **Phase 8**: CI/CD integration

### Prerequisites
- Docker
- AWS CLI v2
- Go 1.21+

### Test Implementation Pattern
Scenario tests use standard Go testing with helper utilities:
```go
func TestClusterLifecycle(t *testing.T) {
    // Start KECS container
    kecs := utils.StartKECS(t)
    defer kecs.Cleanup()
    
    // Create ECS client
    client := utils.NewECSClient(kecs.Endpoint())
    
    // Test implementation
    err := client.CreateCluster("test-cluster")
    require.NoError(t, err)
    
    // Assertions
    utils.AssertClusterActive(t, client, "test-cluster")
}
```