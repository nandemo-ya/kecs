# testcontainers-go-kecs

A Go library that provides [Testcontainers](https://golang.testcontainers.org/) integration for [KECS](https://github.com/nandemo-ya/kecs) (Kubernetes-based ECS Compatible Service). This library makes it easy to write integration tests for applications that use Amazon ECS by providing a local, containerized ECS-compatible environment.

## Features

- Simple API for starting KECS containers in tests
- Full integration with AWS SDK Go v2
- Support for both test mode (no Kubernetes required) and full Kubernetes mode
- Helper functions for common test scenarios
- Automatic container lifecycle management
- Customizable container configuration

## Installation

```bash
go get github.com/nandemo-ya/kecs/testcontainers-go-kecs
```

## Quick Start

```go
package myapp_test

import (
    "context"
    "testing"
    
    "github.com/nandemo-ya/kecs/testcontainers-go-kecs"
    "github.com/stretchr/testify/require"
)

func TestMyECSIntegration(t *testing.T) {
    ctx := context.Background()
    
    // Start KECS container in test mode (no Kubernetes required)
    container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
    require.NoError(t, err)
    defer container.Cleanup(ctx)
    
    // Create ECS client
    client, err := container.NewECSClient(ctx)
    require.NoError(t, err)
    
    // Create a cluster
    _, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
        ClusterName: aws.String("test-cluster"),
    })
    require.NoError(t, err)
    
    // Your test code here...
}
```

## Configuration Options

### WithTestMode()
Enables test mode where KECS runs without requiring Kubernetes. Perfect for unit tests and CI environments.

```go
container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
```

### WithImage(image string)
Use a specific KECS Docker image version.

```go
container, err := kecs.StartContainer(ctx, 
    kecs.WithImage("ghcr.io/nandemo-ya/kecs:v1.0.0"),
)
```

### WithRegion(region string)
Set the AWS region for the ECS environment.

```go
container, err := kecs.StartContainer(ctx, 
    kecs.WithRegion("eu-west-1"),
)
```

### WithEnv(env map[string]string)
Set additional environment variables for the container.

```go
container, err := kecs.StartContainer(ctx, 
    kecs.WithEnv(map[string]string{
        "LOG_LEVEL": "debug",
    }),
)
```

### WithLogConsumer(consumer testcontainers.LogConsumer)
Attach a log consumer to see container logs during tests.

```go
container, err := kecs.StartContainer(ctx, 
    kecs.WithLogConsumer(testcontainers.LogConsumerFunc(func(log testcontainers.Log) {
        t.Logf("KECS: %s", log.Content)
    })),
)
```

### WithWaitTimeout(timeout time.Duration)
Set custom timeout for container startup.

```go
container, err := kecs.StartContainer(ctx, 
    kecs.WithWaitTimeout(2 * time.Minute),
)
```

## Helper Functions

The library provides several helper functions for common test scenarios:

### WaitForCluster
Wait for a cluster to reach a specific status.

```go
err := kecs.WaitForCluster(ctx, client, "my-cluster", "ACTIVE", 30*time.Second)
```

### WaitForService
Wait for a service to reach a specific status.

```go
err := kecs.WaitForService(ctx, client, "my-cluster", "my-service", "ACTIVE", 30*time.Second)
```

### WaitForTask
Wait for a task to reach a specific status.

```go
err := kecs.WaitForTask(ctx, client, "my-cluster", taskArn, "RUNNING", 30*time.Second)
```

### CreateTestTaskDefinition
Create a simple test task definition.

```go
taskDef, err := kecs.CreateTestTaskDefinition(ctx, client, "test-family")
```

### CreateTestService
Create a simple test service.

```go
service, err := kecs.CreateTestService(ctx, client, "my-cluster", "test-service", "test-family:1")
```

### CleanupCluster
Clean up a cluster and all its resources.

```go
err := kecs.CleanupCluster(ctx, client, "my-cluster")
```

## Complete Example

See the [examples](examples/) directory for complete working examples:

- [Basic Integration Test](examples/basic_test.go) - Simple cluster and service operations
- [Task Lifecycle Test](examples/task_lifecycle_test.go) - Testing task states and transitions
- [Service Scaling Test](examples/service_scaling_test.go) - Testing service scaling operations
- [Advanced Integration Test](examples/advanced_test.go) - Complex scenarios with multiple services

## Compatibility

- Go 1.21 or higher
- Docker
- Compatible with AWS SDK Go v2
- Works with standard Go testing framework and popular assertion libraries

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.