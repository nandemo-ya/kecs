# Local Development

## Overview

This guide covers setting up KECS for local development, including building from source and running locally.

## Prerequisites

- Go 1.21 or later
- Docker
- Make
- Git

## Building from Source

### Clone the Repository

```bash
git clone https://github.com/nandemo-ya/kecs.git
cd kecs
```

### Build the Binary

```bash
# Build the control plane
make build

# The binary will be created at ./bin/kecs
```


## Running Locally

### Basic Setup

```bash
# Run KECS with default settings
kecs server

# KECS will start on:
# - API Server: http://localhost:8080
# - Admin Server: http://localhost:8081
```

### Custom Configuration

```bash
# Run with custom ports
kecs server --api-port 9080 --admin-port 9081

# Run with debug logging
kecs server --log-level debug

# Run with custom data directory
kecs server --data-dir ./data
```

### Environment Variables

```bash
# Set log level
export KECS_LOG_LEVEL=debug

# Set data directory
export KECS_DATA_DIR=/path/to/data

# Enable LocalStack integration
export KECS_LOCALSTACK_ENABLED=true
export KECS_LOCALSTACK_ENDPOINT=http://localhost:4566
```

## Development Workflow

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./internal/controlplane/api/...
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make vet

# Run all checks
make all
```

### Hot Reload

For development, use `air` for hot reloading:

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air -c .air.toml
```

Example `.air.toml`:
```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = ["server"]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./controlplane/cmd/controlplane"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_error = true

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
```

## Using KECS Container Commands

KECS provides convenient container commands for running instances:

```bash
# Start a KECS instance with k3d cluster
kecs start

# Start with custom name and port
kecs start --name myinstance --api-port 9090

# View logs from control plane
kubectl logs -n kecs-system deployment/kecs-server -f

# Stop the instance
kecs stop
```

For more details, see [Container Commands documentation](../guides/container-commands.md).

## IDE Setup

### VS Code

Recommended extensions:
- Go
- Prettier
- ESLint
- Docker
- GitLens

Example `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch KECS",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/controlplane/cmd/controlplane",
      "args": ["server", "--log-level", "debug"],
      "env": {
        "KECS_DATA_DIR": "${workspaceFolder}/data"
      }
    }
  ]
}
```

### GoLand

1. Open the project root
2. Configure Go modules support
3. Set up run configuration:
   - Program arguments: `server --log-level debug`
   - Environment variables: `KECS_DATA_DIR=/path/to/data`

## Data Persistence in Container Mode

When running KECS in Docker, data is stored in `/data` by default. To persist data between container restarts:

1. **Mount a volume**: Map a host directory to `/data` in the container
2. **Set environment variables**:
   - `KECS_CONTAINER_MODE=true`
   - `KECS_DATA_DIR=/data`

For detailed information, see the [Container Mode Persistence Guide](/guides/container-persistence).

## Common Development Tasks

### Adding a New API Endpoint

1. Define types in `internal/controlplane/api/types.go`
2. Implement handler in appropriate file (e.g., `clusters.go`)
3. Register handler in `internal/controlplane/api/server.go`
4. Add tests in `*_test.go` file

### Modifying the Database Schema

1. Update schema in `internal/storage/duckdb/schema.sql`
2. Add migration in `internal/storage/duckdb/migrations/`
3. Update storage interface if needed
4. Run tests to verify changes

```

## Troubleshooting

### Port Already in Use

```bash
# Find process using port
lsof -i :8080

# Kill process
kill -9 <PID>
```

### Docker Socket Permission

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Apply changes
newgrp docker
```

### Build Errors

```bash
# Clean build artifacts
make clean

# Update dependencies
go mod tidy
go mod download

# Rebuild
make build
```

### Database Issues

```bash
# Remove database file
rm -rf ~/.kecs/data/kecs.db

# KECS will recreate the database on next start
```

## Next Steps

- [Contributing](/development/contributing) - Contribute to KECS