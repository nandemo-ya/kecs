# KECS Scenario Tests

## Overview

This directory contains end-to-end scenario tests for KECS.
These tests reproduce actual AWS ECS operational scenarios to verify KECS compatibility.

## Prerequisites

- Go 1.21 or higher
- Docker
- AWS CLI v2
- ecspresso (optional)

## Setup

```bash
# Install AWS CLI (macOS)
brew install awscli

# Install ecspresso (optional)
brew install kayac/tap/ecspresso

# Install dependencies
go mod download
```

## Running Tests

### Run all scenario tests
```bash
go test -v ./...
```

### Run specific scenarios
```bash
# Cluster management tests only
go test -v ./cluster

# Service tests only
go test -v ./service

# Failure tests only
go test -v ./failure
```

### Run with timeout
```bash
go test -v -timeout 30m ./...
```

### Run in parallel
```bash
go test -v -parallel 4 ./...
```

## Test Structure

```
scenarios/
├── README.md                   # This file
├── go.mod                      # Module definition
├── go.sum                      # Dependency lock file
├── cluster/                    # Cluster management tests
│   ├── cluster_lifecycle_test.go
│   └── multi_cluster_test.go
├── task_definition/            # Task definition tests
│   ├── register_test.go
│   ├── update_test.go
│   └── complex_task_test.go
├── service/                    # Service management tests
│   ├── service_lifecycle_test.go
│   ├── rolling_update_test.go
│   └── scaling_test.go
├── task/                       # Task execution tests
│   ├── task_lifecycle_test.go
│   ├── task_status_test.go
│   └── run_task_test.go
├── ecspresso/                  # ecspresso integration tests
│   ├── deploy_test.go
│   ├── rollback_test.go
│   ├── diff_test.go
│   └── verify_test.go
├── failure/                    # Failure scenario tests
│   ├── task_failure_test.go
│   └── health_check_test.go
├── fixtures/                   # Test data
│   ├── task-definitions/
│   ├── services/
│   └── ecspresso/
├── utils/                      # Utilities
│   ├── kecs_container.go       # Testcontainers wrapper
│   ├── ecs_client.go           # AWS CLI wrapper
│   ├── ecspresso_client.go     # ecspresso wrapper
│   ├── task_status_checker.go  # Task status monitoring
│   └── assertions.go           # Custom assertions
└── results/                    # Test results (gitignored)
```

## Writing Tests

### Basic Test Structure
```go
func TestServiceCreation(t *testing.T) {
    // 1. Setup KECS environment
    kecs := utils.StartKECS(t)
    defer kecs.Cleanup()
    
    // 2. Create client
    client := utils.NewECSClient(kecs.Endpoint())
    
    // 3. Run test
    t.Run("create service with multiple replicas", func(t *testing.T) {
        // Create cluster
        err := client.CreateCluster("test-cluster")
        require.NoError(t, err)
        
        // Register task definition
        err = client.RegisterTaskDefinition("fixtures/task-definitions/nginx.json")
        require.NoError(t, err)
        
        // Create service
        err = client.CreateService("test-cluster", "nginx-service", "nginx:1", 3)
        require.NoError(t, err)
        
        // Assertions
        utils.AssertServiceHealthy(t, client, "test-cluster", "nginx-service", 30*time.Second)
    })
}
```

### Custom Assertions
```go
// Wait for service to become healthy
utils.AssertServiceHealthy(t, client, cluster, service, timeout)

// Wait for specific task count
utils.AssertTaskCount(t, client, cluster, service, expectedCount, timeout)

// Verify rolling update completion
utils.AssertRollingUpdateComplete(t, client, cluster, service, newRevision, timeout)

// Track task status transitions
utils.AssertTaskStatusTransitions(t, client, cluster, taskArn, []string{
    "PROVISIONING", "PENDING", "ACTIVATING", "RUNNING",
})
```

## Debugging

### Enable detailed logging
```bash
KECS_LOG_LEVEL=debug go test -v ./...
```

### Check container logs
```go
// In test code
logs, err := kecs.GetLogs(ctx)
t.Logf("KECS logs:\n%s", logs)
```

### Test failure snapshots
When tests fail, the following information is saved to `results/` directory:
- KECS container logs
- Cluster state
- Service state
- Task state
- Event history

## CI/CD Integration

GitHub Actions example:
```yaml
- name: Run scenario tests
  run: |
    cd tests/scenarios
    go test -v -timeout 30m ./... -json > results/test-output.json
    
- name: Generate test report
  if: always()
  run: |
    go run github.com/jstemmer/go-junit-report < results/test-output.json > results/junit.xml
```

## Troubleshooting

### "Docker daemon not running"
```bash
# Check if Docker is running
docker ps

# Start Docker Desktop (macOS)
open -a Docker
```

### "AWS CLI not found"
```bash
# Verify installation
aws --version

# Add to PATH
export PATH=$PATH:/usr/local/bin
```

### Tests timing out
- Increase timeout: `go test -timeout 60m`
- Reduce parallel execution: `go test -parallel 1`
- Check resource limits: `docker system df`

### Container startup issues
- Check Docker resource allocation
- Verify KECS image exists: `docker images | grep kecs`
- Check port availability: `lsof -i :8080`

## Contributing

1. Add new scenarios in appropriate directories
2. Use existing utility functions
3. Ensure tests are independent (no dependencies on other tests)
4. Always implement proper cleanup
5. Update documentation for new test scenarios
6. Follow existing naming conventions:
   - Test files: `*_test.go`
   - Test functions: `TestScenarioName`
   - Helper functions: `assertCondition` or `waitForState`