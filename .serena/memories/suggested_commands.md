# KECS Essential Development Commands

## Building and Running
```bash
make build          # Build the binary to bin/kecs
make run            # Build and run the application
make all            # Clean, format, vet, test, and build
```

## Container-based Execution
```bash
kecs start          # Start KECS in a Docker container
kecs stop           # Stop and remove KECS container
kecs status         # Show container status
kecs logs -f        # Follow container logs

# Multiple instances
kecs start --name dev --api-port 8080
kecs start --name staging --api-port 8090 --auto-port
kecs instances list # List all instances
```

## Development Workflow (Hot Reload)
```bash
# 1. Start KECS instance
./bin/kecs start

# 2. Make code changes and hot reload
make dev            # Build and hot reload controlplane
make dev-logs       # Build, hot reload, and tail logs

# For specific instance
KECS_INSTANCE=myinstance make dev
```

## Testing
```bash
make test           # Run tests with race detection
make test-coverage  # Run tests with coverage report

# Control plane tests (using Ginkgo)
cd controlplane && ginkgo -r
# Or using go test
cd controlplane && go test ./...

# Scenario tests
cd tests/scenarios && make test
```

## Code Quality
```bash
make fmt            # Format code with gofmt and organize imports
make vet            # Run go vet
make lint           # Run golangci-lint
make lint-fix       # Run golangci-lint and fix issues automatically
```

## Docker Operations
```bash
make docker-build   # Build Docker image
make docker-push    # Build and push Docker image
make docker-build-dev # Build Docker image for k3d registry (dev mode)
```

## Dependencies
```bash
make deps           # Download and verify dependencies
make clean          # Clean build artifacts and coverage files
```

## Documentation
```bash
# Documentation development
cd docs-site
npm install
npm run docs:dev    # Access at http://localhost:5173

# Build documentation
./scripts/build-docs.sh
```

## System Commands (macOS Darwin)
- `ls` - List directory contents
- `find` - Find files and directories
- `grep` - Search text patterns (prefer `rg` ripgrep if available)
- `git` - Version control
- `docker` - Container operations
- `kubectl` - Kubernetes operations