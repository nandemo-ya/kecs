# KECS Directory Structure

## Root Level
- `controlplane/` - Main Go application code
- `docs-site/` - VitePress documentation site
- `mcp-server/` - TypeScript MCP server for AI integration
- `tests/` - Scenario tests with testcontainers
- `examples/` - Example configurations and use cases
- `scripts/` - Build and deployment scripts
- `.github/` - GitHub Actions workflows
- `deployments/` - Deployment configurations

## Controlplane Directory (`controlplane/`)
- `cmd/controlplane/` - Entry point for the control plane binary
- `internal/controlplane/cmd/` - CLI command implementations using Cobra
- `internal/controlplane/api/` - ECS API endpoint implementations
- `internal/controlplane/admin/` - Admin server with health checks
- `internal/converters/` - Task definition to Kubernetes resource converters
- `internal/kubernetes/` - Kubernetes client and resource managers
- `internal/storage/` - Storage interfaces and DuckDB implementation
- `test/` - Test files and test utilities
- `awsproxy/` - AWS proxy service implementation

## Documentation
- `docs/adr/records/` - Architectural Decision Records
- `docs-site/` - VitePress-based documentation site
  - `.vitepress/config.js` - Site configuration
  - `guides/` - User guides and tutorials
  - `api/` - API reference documentation
  - `architecture/` - Architecture documentation
  - `deployment/` - Deployment guides
  - `development/` - Developer documentation

## Testing Structure
- Ginkgo tests alongside code (`*_test.go` files)
- Scenario tests in `tests/scenarios/` using testcontainers
- Test configurations in `controlplane/test/`