# Integration Testing with KECS

This guide covers best practices and patterns for writing integration tests against ECS-compatible APIs using KECS. Whether you're testing microservices, batch jobs, or complex orchestrations, KECS provides a reliable testing environment.

## Overview

Integration testing with KECS allows you to:

- Test your application's ECS integration without AWS costs
- Verify task orchestration and service behavior
- Test failure scenarios and recovery mechanisms
- Validate your infrastructure as code
- Ensure your application works correctly with ECS APIs

## Testing Strategies

### 1. Component Integration Tests

Test individual components that interact with ECS:

```go
func TestTaskRunner_ExecuteJob(t *testing.T) {
    // Setup KECS
    ctx := context.Background()
    container, _ := kecs.StartContainer(ctx, kecs.WithTestMode())
    defer container.Cleanup(ctx)
    
    ecsClient, _ := container.NewECSClient(ctx)
    
    // Setup your component
    runner := NewTaskRunner(ecsClient)
    
    // Create test infrastructure
    cluster := createTestCluster(t, ecsClient)
    taskDef := registerJobTaskDefinition(t, ecsClient)
    
    // Test the component
    job := &Job{
        Name: "data-processing",
        TaskDefinition: taskDef,
        Environment: map[string]string{
            "INPUT_FILE": "s3://bucket/input.csv",
        },
    }
    
    result, err := runner.ExecuteJob(ctx, cluster, job)
    require.NoError(t, err)
    assert.Equal(t, "COMPLETED", result.Status)
}
```

### 2. Service Integration Tests

Test service deployments and updates:

```go
func TestServiceManager_Deploy(t *testing.T) {
    ctx := context.Background()
    container, _ := kecs.StartContainer(ctx, kecs.WithTestMode())
    defer container.Cleanup(ctx)
    
    ecsClient, _ := container.NewECSClient(ctx)
    manager := NewServiceManager(ecsClient)
    
    // Deploy a service
    deployment := &Deployment{
        Cluster: "production",
        Service: "api",
        Image: "myapp:v2.0",
        DesiredCount: 3,
        Environment: map[string]string{
            "API_VERSION": "v2",
        },
    }
    
    err := manager.Deploy(ctx, deployment)
    require.NoError(t, err)
    
    // Verify deployment
    service, err := manager.GetService(ctx, "production", "api")
    require.NoError(t, err)
    assert.Equal(t, int32(3), service.DesiredCount)
    assert.Equal(t, "myapp:v2.0", service.TaskDefinition.Image)
}
```

### 3. End-to-End Workflow Tests

Test complete workflows across multiple services:

```go
func TestOrderProcessingWorkflow(t *testing.T) {
    ctx := context.Background()
    container, _ := kecs.StartContainer(ctx, kecs.WithTestMode())
    defer container.Cleanup(ctx)
    
    // Setup services
    setupOrderService(t, container)
    setupPaymentService(t, container)
    setupShippingService(t, container)
    
    // Execute workflow
    order := createTestOrder()
    workflow := NewOrderWorkflow(container.Endpoint())
    
    result, err := workflow.ProcessOrder(ctx, order)
    require.NoError(t, err)
    
    // Verify all steps completed
    assert.Equal(t, "SHIPPED", result.Status)
    assert.NotEmpty(t, result.TrackingNumber)
    assert.True(t, result.PaymentProcessed)
}
```

## Testing Patterns

### Pattern: Test Fixtures

Create reusable test fixtures:

```go
type TestFixture struct {
    Container *kecs.Container
    Client    *ecs.Client
    Cluster   string
}

func NewTestFixture(t *testing.T) *TestFixture {
    ctx := context.Background()
    
    container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
    require.NoError(t, err)
    
    client, err := container.NewECSClient(ctx)
    require.NoError(t, err)
    
    cluster := fmt.Sprintf("test-%s", t.Name())
    _, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
        ClusterName: aws.String(cluster),
    })
    require.NoError(t, err)
    
    return &TestFixture{
        Container: container,
        Client:    client,
        Cluster:   cluster,
    }
}

func (f *TestFixture) Cleanup(t *testing.T) {
    ctx := context.Background()
    kecs.CleanupCluster(ctx, f.Client, f.Cluster)
    f.Container.Cleanup(ctx)
}
```

### Pattern: Test Data Builders

Use builders for complex test data:

```go
type TaskDefinitionBuilder struct {
    family       string
    cpu          string
    memory       string
    containers   []types.ContainerDefinition
}

func NewTaskDefinitionBuilder(family string) *TaskDefinitionBuilder {
    return &TaskDefinitionBuilder{
        family: family,
        cpu:    "256",
        memory: "512",
    }
}

func (b *TaskDefinitionBuilder) WithContainer(name, image string) *TaskDefinitionBuilder {
    b.containers = append(b.containers, types.ContainerDefinition{
        Name:      aws.String(name),
        Image:     aws.String(image),
        Essential: aws.Bool(true),
    })
    return b
}

func (b *TaskDefinitionBuilder) Build() *ecs.RegisterTaskDefinitionInput {
    return &ecs.RegisterTaskDefinitionInput{
        Family:               aws.String(b.family),
        Cpu:                  aws.String(b.cpu),
        Memory:               aws.String(b.memory),
        ContainerDefinitions: b.containers,
    }
}

// Usage
taskDef := NewTaskDefinitionBuilder("web-app").
    WithContainer("nginx", "nginx:alpine").
    WithContainer("app", "myapp:latest").
    Build()
```

### Pattern: Assertion Helpers

Create domain-specific assertions:

```go
func AssertServiceHealthy(t *testing.T, client *ecs.Client, cluster, service string) {
    t.Helper()
    
    ctx := context.Background()
    resp, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
        Cluster:  aws.String(cluster),
        Services: []string{service},
    })
    require.NoError(t, err)
    require.Len(t, resp.Services, 1)
    
    svc := resp.Services[0]
    assert.Equal(t, "ACTIVE", *svc.Status)
    assert.Equal(t, *svc.DesiredCount, *svc.RunningCount)
    assert.Empty(t, svc.Events) // No recent error events
}

func AssertTaskCompleted(t *testing.T, client *ecs.Client, cluster, taskArn string) {
    t.Helper()
    
    err := kecs.WaitForTask(context.Background(), client, cluster, taskArn, "STOPPED", 60*time.Second)
    require.NoError(t, err)
    
    // Check exit code
    resp, err := client.DescribeTasks(context.Background(), &ecs.DescribeTasksInput{
        Cluster: aws.String(cluster),
        Tasks:   []string{taskArn},
    })
    require.NoError(t, err)
    
    task := resp.Tasks[0]
    for _, container := range task.Containers {
        assert.Equal(t, int32(0), *container.ExitCode)
    }
}
```

## Testing Scenarios

### Testing Service Discovery

```go
func TestServiceDiscovery(t *testing.T) {
    fixture := NewTestFixture(t)
    defer fixture.Cleanup(t)
    
    // Register services
    apiService := deployService(t, fixture, "api", 3)
    dbService := deployService(t, fixture, "database", 1)
    
    // Test service discovery
    discovery := NewServiceDiscovery(fixture.Client)
    
    // Find API endpoints
    endpoints, err := discovery.GetEndpoints(fixture.Cluster, "api")
    require.NoError(t, err)
    assert.Len(t, endpoints, 3)
    
    // Test connection between services
    for _, endpoint := range endpoints {
        assert.True(t, canConnect(endpoint, dbService.Endpoint))
    }
}
```

### Testing Rolling Updates

```go
func TestRollingUpdate(t *testing.T) {
    fixture := NewTestFixture(t)
    defer fixture.Cleanup(t)
    
    // Deploy initial version
    service := deployService(t, fixture, "web", 4)
    waitForStableService(t, fixture, service)
    
    // Capture initial state
    initialTasks := getRunningTasks(t, fixture, service)
    
    // Perform rolling update
    newTaskDef := registerNewVersion(t, fixture, "web:v2")
    _, err := fixture.Client.UpdateService(context.Background(), &ecs.UpdateServiceInput{
        Cluster:        aws.String(fixture.Cluster),
        Service:        service.ServiceName,
        TaskDefinition: newTaskDef.TaskDefinitionArn,
    })
    require.NoError(t, err)
    
    // Monitor rolling update
    Eventually(func() bool {
        tasks := getRunningTasks(t, fixture, service)
        
        // All tasks should be running new version
        for _, task := range tasks {
            if *task.TaskDefinitionArn != *newTaskDef.TaskDefinitionArn {
                return false
            }
        }
        
        // Should maintain desired count
        return len(tasks) == 4
    }, 2*time.Minute, 5*time.Second).Should(BeTrue())
    
    // Verify zero downtime
    assert.True(t, wasAlwaysHealthy(t, fixture, service))
}
```

### Testing Auto-Recovery

```go
func TestAutoRecovery(t *testing.T) {
    fixture := NewTestFixture(t)
    defer fixture.Cleanup(t)
    
    // Create service with health checks
    service := deployServiceWithHealthCheck(t, fixture, "api", 3)
    waitForStableService(t, fixture, service)
    
    // Simulate task failure
    tasks := getRunningTasks(t, fixture, service)
    failedTask := tasks[0]
    
    _, err := fixture.Client.StopTask(context.Background(), &ecs.StopTaskInput{
        Cluster: aws.String(fixture.Cluster),
        Task:    failedTask.TaskArn,
        Reason:  aws.String("Simulated failure"),
    })
    require.NoError(t, err)
    
    // Service should recover
    Eventually(func() int {
        return len(getRunningTasks(t, fixture, service))
    }, 30*time.Second, 1*time.Second).Should(Equal(3))
    
    // Verify new task is healthy
    AssertServiceHealthy(t, fixture.Client, fixture.Cluster, *service.ServiceName)
}
```

## Performance Testing

### Load Testing with KECS

```go
func TestHighLoadScenario(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test")
    }
    
    fixture := NewTestFixture(t)
    defer fixture.Cleanup(t)
    
    // Deploy service
    service := deployService(t, fixture, "api", 10)
    
    // Generate load
    var wg sync.WaitGroup
    errors := make(chan error, 100)
    
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            // Simulate client operations
            for j := 0; j < 10; j++ {
                _, err := fixture.Client.ListTasks(context.Background(), &ecs.ListTasksInput{
                    Cluster:     aws.String(fixture.Cluster),
                    ServiceName: service.ServiceName,
                })
                if err != nil {
                    errors <- err
                }
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    var errorCount int
    for err := range errors {
        t.Logf("Error during load test: %v", err)
        errorCount++
    }
    
    // Allow some errors but not too many
    assert.Less(t, errorCount, 10, "Too many errors during load test")
}
```

## Debugging Test Failures

### Enable Verbose Logging

```go
func TestWithDebugging(t *testing.T) {
    ctx := context.Background()
    
    // Enable all debug options
    container, err := kecs.StartContainer(ctx,
        kecs.WithTestMode(),
        kecs.WithEnv(map[string]string{
            "LOG_LEVEL": "trace",
            "DEBUG":     "true",
        }),
        kecs.WithLogConsumer(testcontainers.LogConsumerFunc(func(log testcontainers.Log) {
            t.Logf("[KECS] %s", strings.TrimSpace(string(log.Content)))
        })),
    )
    require.NoError(t, err)
    defer container.Cleanup(ctx)
    
    // Your test code...
}
```

### Capture State on Failure

```go
func TestWithDiagnostics(t *testing.T) {
    fixture := NewTestFixture(t)
    defer func() {
        if t.Failed() {
            captureDiagnostics(t, fixture)
        }
        fixture.Cleanup(t)
    }()
    
    // Test code...
}

func captureDiagnostics(t *testing.T, fixture *TestFixture) {
    t.Helper()
    
    ctx := context.Background()
    
    // List all resources
    clusters, _ := fixture.Client.ListClusters(ctx, &ecs.ListClustersInput{})
    t.Logf("Clusters: %v", clusters.ClusterArns)
    
    services, _ := fixture.Client.ListServices(ctx, &ecs.ListServicesInput{
        Cluster: aws.String(fixture.Cluster),
    })
    t.Logf("Services: %v", services.ServiceArns)
    
    tasks, _ := fixture.Client.ListTasks(ctx, &ecs.ListTasksInput{
        Cluster: aws.String(fixture.Cluster),
    })
    t.Logf("Tasks: %v", tasks.TaskArns)
    
    // Get container logs if available
    if logs, err := fixture.Container.Logs(ctx); err == nil {
        t.Logf("Container logs:\n%s", logs)
    }
}
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration-test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run Integration Tests
      run: |
        go test -v -race -tags=integration ./...
      env:
        KECS_TEST_MODE: "true"
        TEST_TIMEOUT: "10m"
```

### Makefile Integration

```makefile
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	KECS_TEST_MODE=true go test -v -timeout=10m -tags=integration ./...

.PHONY: test-integration-verbose
test-integration-verbose:
	@echo "Running integration tests with verbose output..."
	KECS_TEST_MODE=true go test -v -timeout=10m -tags=integration ./... -args -test.v

.PHONY: test-integration-specific
test-integration-specific:
	@echo "Running specific integration test..."
	KECS_TEST_MODE=true go test -v -timeout=10m -tags=integration -run=$(TEST) ./...
```

## Best Practices Summary

1. **Isolate Test Environments**: Each test should create its own cluster
2. **Use Test Mode**: For faster feedback during development
3. **Clean Up Resources**: Always defer cleanup operations
4. **Handle Timeouts Gracefully**: Adjust for CI environments
5. **Make Tests Deterministic**: Avoid timing-dependent assertions
6. **Test Error Scenarios**: Verify your error handling
7. **Use Parallel Testing**: With unique resource names
8. **Monitor Test Performance**: Track test execution times
9. **Document Test Requirements**: Clearly state what each test verifies
10. **Version Your Test Data**: Keep test fixtures in version control

## Next Steps

- Learn about [using Testcontainers](/guides/testcontainers) specifically
- Explore [example tests](https://github.com/nandemo-ya/kecs/tree/main/testcontainers-go-kecs/examples)
- Read about [KECS architecture](/architecture/) to understand the testing environment
- Check the [API reference](/api/) for available operations