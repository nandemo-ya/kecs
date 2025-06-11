# Testcontainers Integration

KECS provides a [Testcontainers](https://golang.testcontainers.org/) integration package that makes it easy to write integration tests for applications that use Amazon ECS. The `testcontainers-go-kecs` package allows you to run KECS in Docker containers during tests, providing a local ECS-compatible environment.

## Installation

Add the testcontainers-go-kecs package to your Go module:

```bash
go get github.com/nandemo-ya/kecs/testcontainers-go-kecs
```

## Quick Start

Here's a simple example of using KECS with Testcontainers in your tests:

```go
package myapp_test

import (
    "context"
    "testing"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/ecs"
    "github.com/nandemo-ya/kecs/testcontainers-go-kecs"
    "github.com/stretchr/testify/require"
)

func TestECSIntegration(t *testing.T) {
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

The package provides several options to customize the KECS container:

### Test Mode

Enable test mode to run KECS without requiring Kubernetes. This is ideal for unit tests and CI environments:

```go
container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
```

### Custom Docker Image

Use a specific version of the KECS Docker image:

```go
container, err := kecs.StartContainer(ctx, 
    kecs.WithImage("ghcr.io/nandemo-ya/kecs:v1.0.0"),
)
```

### AWS Region

Set the AWS region for the ECS environment:

```go
container, err := kecs.StartContainer(ctx, 
    kecs.WithRegion("eu-west-1"),
)
```

### Environment Variables

Add custom environment variables to the container:

```go
container, err := kecs.StartContainer(ctx, 
    kecs.WithEnv(map[string]string{
        "LOG_LEVEL": "debug",
        "CUSTOM_CONFIG": "value",
    }),
)
```

### Log Consumer

Attach a log consumer to see container logs during tests:

```go
container, err := kecs.StartContainer(ctx, 
    kecs.WithLogConsumer(testcontainers.LogConsumerFunc(func(log testcontainers.Log) {
        t.Logf("KECS: %s", log.Content)
    })),
)
```

### Startup Timeout

Set a custom timeout for container startup:

```go
container, err := kecs.StartContainer(ctx, 
    kecs.WithWaitTimeout(2 * time.Minute),
)
```

## Helper Functions

The package includes several helper functions for common test scenarios:

### Wait Functions

Wait for resources to reach specific states:

```go
// Wait for a cluster to become active
err := kecs.WaitForCluster(ctx, client, "my-cluster", "ACTIVE", 30*time.Second)

// Wait for a service to become active
err := kecs.WaitForService(ctx, client, "my-cluster", "my-service", "ACTIVE", 30*time.Second)

// Wait for a task to start running
err := kecs.WaitForTask(ctx, client, "my-cluster", taskArn, "RUNNING", 30*time.Second)
```

### Resource Creation Helpers

Create test resources quickly:

```go
// Create a simple test task definition
taskDef, err := kecs.CreateTestTaskDefinition(ctx, client, "test-family")

// Create a test service
service, err := kecs.CreateTestService(ctx, client, "my-cluster", "test-service", "test-family:1")
```

### Cleanup Helpers

Clean up test resources:

```go
// Clean up a cluster and all its resources
err := kecs.CleanupCluster(ctx, client, "my-cluster")
```

## Complete Examples

### Testing Service Scaling

```go
func TestServiceScaling(t *testing.T) {
    ctx := context.Background()
    
    // Start KECS
    container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
    require.NoError(t, err)
    defer container.Cleanup(ctx)
    
    client, err := container.NewECSClient(ctx)
    require.NoError(t, err)
    
    // Create cluster
    clusterName := "scale-test"
    _, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
        ClusterName: aws.String(clusterName),
    })
    require.NoError(t, err)
    
    // Create task definition
    taskDef, err := kecs.CreateTestTaskDefinition(ctx, client, "scale-task")
    require.NoError(t, err)
    
    // Create service
    serviceName := "scale-service"
    _, err = client.CreateService(ctx, &ecs.CreateServiceInput{
        Cluster:        aws.String(clusterName),
        ServiceName:    aws.String(serviceName),
        TaskDefinition: taskDef.TaskDefinitionArn,
        DesiredCount:   aws.Int32(1),
    })
    require.NoError(t, err)
    
    // Scale up
    _, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
        Cluster:      aws.String(clusterName),
        Service:      aws.String(serviceName),
        DesiredCount: aws.Int32(3),
    })
    require.NoError(t, err)
    
    // Wait for service to stabilize
    err = kecs.WaitForService(ctx, client, clusterName, serviceName, "ACTIVE", 30*time.Second)
    require.NoError(t, err)
    
    // Verify task count
    tasks, err := client.ListTasks(ctx, &ecs.ListTasksInput{
        Cluster:     aws.String(clusterName),
        ServiceName: aws.String(serviceName),
    })
    require.NoError(t, err)
    assert.Len(t, tasks.TaskArns, 3)
}
```

### Testing Task Lifecycle

```go
func TestTaskLifecycle(t *testing.T) {
    ctx := context.Background()
    
    // Start KECS with debug logging
    container, err := kecs.StartContainer(ctx,
        kecs.WithTestMode(),
        kecs.WithEnv(map[string]string{"LOG_LEVEL": "debug"}),
    )
    require.NoError(t, err)
    defer container.Cleanup(ctx)
    
    client, err := container.NewECSClient(ctx)
    require.NoError(t, err)
    
    // Create cluster
    clusterName := "task-test"
    _, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
        ClusterName: aws.String(clusterName),
    })
    require.NoError(t, err)
    
    // Register task definition
    registerOutput, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
        Family: aws.String("lifecycle-task"),
        ContainerDefinitions: []types.ContainerDefinition{
            {
                Name:    aws.String("app"),
                Image:   aws.String("busybox:latest"),
                Memory:  aws.Int32(128),
                Command: []string{"sh", "-c", "echo 'Hello KECS' && sleep 30"},
            },
        },
    })
    require.NoError(t, err)
    
    // Run task
    runOutput, err := client.RunTask(ctx, &ecs.RunTaskInput{
        Cluster:        aws.String(clusterName),
        TaskDefinition: registerOutput.TaskDefinition.TaskDefinitionArn,
        Count:          aws.Int32(1),
    })
    require.NoError(t, err)
    require.Len(t, runOutput.Tasks, 1)
    
    taskArn := runOutput.Tasks[0].TaskArn
    
    // Wait for task to start
    err = kecs.WaitForTask(ctx, client, clusterName, *taskArn, "RUNNING", 30*time.Second)
    require.NoError(t, err)
    
    // Stop task
    _, err = client.StopTask(ctx, &ecs.StopTaskInput{
        Cluster: aws.String(clusterName),
        Task:    taskArn,
        Reason:  aws.String("Test completed"),
    })
    require.NoError(t, err)
    
    // Wait for task to stop
    err = kecs.WaitForTask(ctx, client, clusterName, *taskArn, "STOPPED", 30*time.Second)
    require.NoError(t, err)
}
```

## Best Practices

### 1. Use Test Mode for Unit Tests

For unit tests and CI environments where Kubernetes is not available, always use test mode:

```go
container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
```

### 2. Clean Up Resources

Always clean up resources after tests to prevent resource leaks:

```go
defer container.Cleanup(ctx)
// or
defer kecs.CleanupCluster(ctx, client, clusterName)
```

### 3. Use Appropriate Timeouts

Set realistic timeouts for operations based on your test environment:

```go
// Longer timeout for CI environments
timeout := 60 * time.Second
if os.Getenv("CI") == "true" {
    timeout = 2 * time.Minute
}
err := kecs.WaitForService(ctx, client, cluster, service, "ACTIVE", timeout)
```

### 4. Parallel Test Execution

Each test should create its own cluster to enable parallel execution:

```go
func TestFeatureA(t *testing.T) {
    t.Parallel()
    clusterName := fmt.Sprintf("test-a-%d", time.Now().UnixNano())
    // ... test code
}

func TestFeatureB(t *testing.T) {
    t.Parallel()
    clusterName := fmt.Sprintf("test-b-%d", time.Now().UnixNano())
    // ... test code
}
```

### 5. Container Logs for Debugging

Enable log consumption when debugging test failures:

```go
if testing.Verbose() {
    opts = append(opts, kecs.WithLogConsumer(
        testcontainers.LogConsumerFunc(func(log testcontainers.Log) {
            t.Logf("KECS: %s", log.Content)
        }),
    ))
}
container, err := kecs.StartContainer(ctx, opts...)
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: 1.21
      
      - name: Run Integration Tests
        run: |
          go test -v ./... -tags=integration
        env:
          KECS_TEST_MODE: "true"
```

### GitLab CI Example

```yaml
integration-tests:
  image: golang:1.21
  services:
    - docker:dind
  variables:
    DOCKER_HOST: tcp://docker:2375
    KECS_TEST_MODE: "true"
  script:
    - go test -v ./... -tags=integration
```

## Advanced Usage

### Custom AWS Configuration

For more complex AWS SDK configurations:

```go
// Get the container
container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
require.NoError(t, err)

// Create custom AWS config
cfg, err := config.LoadDefaultConfig(ctx,
    config.WithRegion(container.Region()),
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
    config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
        func(service, region string, options ...interface{}) (aws.Endpoint, error) {
            if service == ecs.ServiceID {
                return aws.Endpoint{
                    URL:               container.Endpoint(),
                    HostnameImmutable: true,
                }, nil
            }
            return aws.Endpoint{}, &aws.EndpointNotFoundError{}
        },
    )),
    // Add custom retry configuration
    config.WithRetryMode(aws.RetryModeAdaptive),
    config.WithRetryMaxAttempts(5),
)
require.NoError(t, err)

// Create ECS client with custom config
client := ecs.NewFromConfig(cfg)
```

### Testing with Multiple KECS Instances

For testing distributed scenarios:

```go
func TestMultiRegionDeployment(t *testing.T) {
    ctx := context.Background()
    
    // Start KECS instances for different regions
    usEast, err := kecs.StartContainer(ctx,
        kecs.WithTestMode(),
        kecs.WithRegion("us-east-1"),
        kecs.WithAPIPort("8080"),
    )
    require.NoError(t, err)
    defer usEast.Cleanup(ctx)
    
    euWest, err := kecs.StartContainer(ctx,
        kecs.WithTestMode(),
        kecs.WithRegion("eu-west-1"),
        kecs.WithAPIPort("8081"),
    )
    require.NoError(t, err)
    defer euWest.Cleanup(ctx)
    
    // Create clients for each region
    usClient, err := usEast.NewECSClient(ctx)
    require.NoError(t, err)
    
    euClient, err := euWest.NewECSClient(ctx)
    require.NoError(t, err)
    
    // Test cross-region scenarios...
}
```

## Troubleshooting

### Container Fails to Start

If the KECS container fails to start:

1. Check Docker is running: `docker ps`
2. Enable debug logging: `kecs.WithEnv(map[string]string{"LOG_LEVEL": "debug"})`
3. Increase startup timeout: `kecs.WithWaitTimeout(5 * time.Minute)`
4. Check for port conflicts

### Tests Timeout

If tests timeout waiting for resources:

1. Increase wait timeouts in helper functions
2. Check container logs for errors
3. Ensure sufficient system resources
4. Use test mode for faster startup

### Resource Cleanup Issues

If resources aren't cleaned up properly:

1. Use `defer` statements for cleanup
2. Use unique resource names with timestamps
3. Implement test cleanup in `TestMain` if needed

## More Examples

The testcontainers-go-kecs package includes extensive examples:

- [Basic Operations](https://github.com/nandemo-ya/kecs/blob/main/testcontainers-go-kecs/examples/basic_test.go)
- [Task Lifecycle](https://github.com/nandemo-ya/kecs/blob/main/testcontainers-go-kecs/examples/task_lifecycle_test.go)
- [Service Scaling](https://github.com/nandemo-ya/kecs/blob/main/testcontainers-go-kecs/examples/service_scaling_test.go)
- [Advanced Scenarios](https://github.com/nandemo-ya/kecs/blob/main/testcontainers-go-kecs/examples/advanced_test.go)