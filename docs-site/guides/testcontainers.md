# Using Testcontainers with KECS

This guide explains how to use the `testcontainers-go-kecs` package to write integration tests for your ECS-compatible applications. Testcontainers provides a lightweight, throwaway instance of KECS that can run in Docker during your tests.

## Why Use Testcontainers?

- **No infrastructure required**: Run tests locally without setting up ECS or Kubernetes
- **Isolated environments**: Each test gets its own clean KECS instance
- **Fast feedback**: Quick startup times, especially in test mode
- **CI/CD friendly**: Works seamlessly in containerized CI environments
- **Real ECS API**: Test against actual ECS-compatible APIs, not mocks

## Installation

First, add the testcontainers-go-kecs package to your project:

```bash
go get github.com/nandemo-ya/kecs/testcontainers-go-kecs
```

## Basic Usage

Here's the simplest way to use KECS in your tests:

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

func TestMyECSApplication(t *testing.T) {
    ctx := context.Background()
    
    // Start KECS container
    container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
    require.NoError(t, err)
    defer container.Cleanup(ctx)
    
    // Get ECS client
    client, err := container.NewECSClient(ctx)
    require.NoError(t, err)
    
    // Now use the client as you would with real ECS
    _, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
        ClusterName: aws.String("my-test-cluster"),
    })
    require.NoError(t, err)
}
```

## Test Mode vs Full Mode

KECS can run in two modes:

### Test Mode (Recommended for most tests)
- No Kubernetes required
- Faster startup
- Simulated task execution
- Perfect for unit and integration tests

```go
container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
```

### Full Mode
- Requires Kubernetes (Kind)
- Real container execution
- Closer to production behavior
- Use for end-to-end tests

```go
container, err := kecs.StartContainer(ctx)  // No WithTestMode()
```

## Common Testing Patterns

### Testing Service Deployment

```go
func TestDeployService(t *testing.T) {
    ctx := context.Background()
    
    // Setup
    container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
    require.NoError(t, err)
    defer container.Cleanup(ctx)
    
    client, err := container.NewECSClient(ctx)
    require.NoError(t, err)
    
    // Create infrastructure
    _, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
        ClusterName: aws.String("prod-cluster"),
    })
    require.NoError(t, err)
    
    // Register task definition
    taskDefOutput, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
        Family: aws.String("web-app"),
        ContainerDefinitions: []types.ContainerDefinition{
            {
                Name:   aws.String("nginx"),
                Image:  aws.String("nginx:alpine"),
                Memory: aws.Int32(512),
                PortMappings: []types.PortMapping{
                    {
                        ContainerPort: aws.Int32(80),
                    },
                },
            },
        },
    })
    require.NoError(t, err)
    
    // Create service
    serviceOutput, err := client.CreateService(ctx, &ecs.CreateServiceInput{
        Cluster:        aws.String("prod-cluster"),
        ServiceName:    aws.String("web-service"),
        TaskDefinition: taskDefOutput.TaskDefinition.TaskDefinitionArn,
        DesiredCount:   aws.Int32(3),
    })
    require.NoError(t, err)
    
    // Verify service is running
    assert.Equal(t, "web-service", *serviceOutput.Service.ServiceName)
    assert.Equal(t, int32(3), *serviceOutput.Service.DesiredCount)
    
    // Wait for service to be active
    err = kecs.WaitForService(ctx, client, "prod-cluster", "web-service", "ACTIVE", 30*time.Second)
    require.NoError(t, err)
}
```

### Testing Auto-Scaling Behavior

```go
func TestAutoScaling(t *testing.T) {
    ctx := context.Background()
    
    container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
    require.NoError(t, err)
    defer container.Cleanup(ctx)
    
    client, err := container.NewECSClient(ctx)
    require.NoError(t, err)
    
    // Setup cluster and service
    clusterName := "scaling-test"
    _, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
        ClusterName: aws.String(clusterName),
    })
    require.NoError(t, err)
    
    // Create task definition and service (setup code omitted for brevity)
    service, err := kecs.CreateTestService(ctx, client, clusterName, "api-service", "api:1")
    require.NoError(t, err)
    
    // Test scaling up
    _, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
        Cluster:      aws.String(clusterName),
        Service:      service.ServiceArn,
        DesiredCount: aws.Int32(5),
    })
    require.NoError(t, err)
    
    // Verify tasks are created
    Eventually(func() int {
        tasks, _ := client.ListTasks(ctx, &ecs.ListTasksInput{
            Cluster:     aws.String(clusterName),
            ServiceName: service.ServiceName,
        })
        return len(tasks.TaskArns)
    }, 30*time.Second, 1*time.Second).Should(Equal(5))
}
```

### Testing Task Failures and Retries

```go
func TestTaskFailureHandling(t *testing.T) {
    ctx := context.Background()
    
    container, err := kecs.StartContainer(ctx, 
        kecs.WithTestMode(),
        kecs.WithEnv(map[string]string{"LOG_LEVEL": "debug"}),
    )
    require.NoError(t, err)
    defer container.Cleanup(ctx)
    
    client, err := container.NewECSClient(ctx)
    require.NoError(t, err)
    
    // Create cluster
    _, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
        ClusterName: aws.String("failure-test"),
    })
    require.NoError(t, err)
    
    // Register a task that will fail
    _, err = client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
        Family: aws.String("failing-task"),
        ContainerDefinitions: []types.ContainerDefinition{
            {
                Name:     aws.String("app"),
                Image:    aws.String("busybox"),
                Memory:   aws.Int32(128),
                Command:  []string{"sh", "-c", "exit 1"},
                Essential: aws.Bool(true),
            },
        },
    })
    require.NoError(t, err)
    
    // Run the task
    runOutput, err := client.RunTask(ctx, &ecs.RunTaskInput{
        Cluster:        aws.String("failure-test"),
        TaskDefinition: aws.String("failing-task"),
    })
    require.NoError(t, err)
    
    // Wait for task to fail
    taskArn := runOutput.Tasks[0].TaskArn
    err = kecs.WaitForTask(ctx, client, "failure-test", *taskArn, "STOPPED", 30*time.Second)
    require.NoError(t, err)
    
    // Verify task failed
    descOutput, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
        Cluster: aws.String("failure-test"),
        Tasks:   []string{*taskArn},
    })
    require.NoError(t, err)
    
    task := descOutput.Tasks[0]
    assert.Equal(t, "STOPPED", *task.LastStatus)
    assert.NotNil(t, task.StoppedReason)
}
```

## Testing Best Practices

### 1. Use Unique Resource Names

Prevent conflicts when running tests in parallel:

```go
clusterName := fmt.Sprintf("test-cluster-%s-%d", t.Name(), time.Now().UnixNano())
```

### 2. Always Clean Up Resources

Use defer statements to ensure cleanup:

```go
defer container.Cleanup(ctx)
defer kecs.CleanupCluster(ctx, client, clusterName)
```

### 3. Set Appropriate Timeouts

Adjust timeouts based on your environment:

```go
timeout := 30 * time.Second
if os.Getenv("CI") == "true" {
    timeout = 60 * time.Second  // Longer timeout in CI
}
```

### 4. Use Table-Driven Tests

Test multiple scenarios efficiently:

```go
func TestServiceCreation(t *testing.T) {
    testCases := []struct {
        name         string
        desiredCount int32
        launchType   string
        expectError  bool
    }{
        {"basic service", 1, "EC2", false},
        {"scaled service", 5, "EC2", false},
        {"fargate service", 2, "FARGATE", false},
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 5. Enable Debugging When Needed

```go
if testing.Verbose() {
    opts = append(opts, kecs.WithLogConsumer(
        testcontainers.LogConsumerFunc(func(log testcontainers.Log) {
            t.Logf("KECS: %s", log.Content)
        }),
    ))
}
```

## Integration with Testing Frameworks

### Using with Ginkgo

```go
var _ = Describe("ECS Service", func() {
    var (
        container *kecs.Container
        client    *ecs.Client
        ctx       context.Context
    )
    
    BeforeEach(func() {
        ctx = context.Background()
        var err error
        container, err = kecs.StartContainer(ctx, kecs.WithTestMode())
        Expect(err).NotTo(HaveOccurred())
        
        client, err = container.NewECSClient(ctx)
        Expect(err).NotTo(HaveOccurred())
    })
    
    AfterEach(func() {
        Expect(container.Cleanup(ctx)).To(Succeed())
    })
    
    It("should create a service", func() {
        // Test implementation
    })
})
```

### Using with testify/suite

```go
type ECSTestSuite struct {
    suite.Suite
    container *kecs.Container
    client    *ecs.Client
}

func (s *ECSTestSuite) SetupSuite() {
    ctx := context.Background()
    container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
    s.Require().NoError(err)
    s.container = container
    
    client, err := container.NewECSClient(ctx)
    s.Require().NoError(err)
    s.client = client
}

func (s *ECSTestSuite) TearDownSuite() {
    s.Require().NoError(s.container.Cleanup(context.Background()))
}

func (s *ECSTestSuite) TestServiceCreation() {
    // Test implementation
}

func TestECSSuite(t *testing.T) {
    suite.Run(t, new(ECSTestSuite))
}
```

## Advanced Configuration

### Custom Image and Ports

```go
container, err := kecs.StartContainer(ctx,
    kecs.WithImage("myregistry/kecs:custom"),
    kecs.WithAPIPort("9080"),
    kecs.WithAdminPort("9081"),
)
```

### Multiple Regions

```go
// Create containers for different regions
usContainer, _ := kecs.StartContainer(ctx,
    kecs.WithTestMode(),
    kecs.WithRegion("us-east-1"),
)

euContainer, _ := kecs.StartContainer(ctx,
    kecs.WithTestMode(),
    kecs.WithRegion("eu-west-1"),
)
```

### Custom AWS SDK Configuration

```go
// Use your own AWS config
cfg, err := config.LoadDefaultConfig(ctx,
    config.WithRegion("us-west-2"),
    config.WithRetryMode(aws.RetryModeAdaptive),
)

// Apply KECS endpoint
cfg.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(
    func(service, region string, options ...interface{}) (aws.Endpoint, error) {
        if service == ecs.ServiceID {
            return aws.Endpoint{
                URL: container.Endpoint(),
            }, nil
        }
        return aws.Endpoint{}, &aws.EndpointNotFoundError{}
    },
)

client := ecs.NewFromConfig(cfg)
```

## Troubleshooting

### Container Startup Issues

If the container fails to start:

1. Check Docker daemon is running
2. Verify port availability
3. Check Docker resources (memory/disk)
4. Enable debug logging

### Slow Tests

To speed up tests:

1. Use test mode instead of full mode
2. Reuse containers across tests in a suite
3. Run tests in parallel with unique resource names
4. Use smaller container images in task definitions

### Flaky Tests

To improve test reliability:

1. Use explicit waits instead of sleep
2. Increase timeouts in CI environments
3. Check for resource cleanup between tests
4. Use retry logic for transient failures

## Next Steps

- Explore the [complete examples](https://github.com/nandemo-ya/kecs/tree/main/testcontainers-go-kecs/examples)
- Read about [integration testing patterns](/guides/integration-testing)
- Learn about [KECS architecture](/architecture/)
- Check the [API reference](/api/) for all available operations