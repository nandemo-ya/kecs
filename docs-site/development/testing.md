# Testing Guide

This guide covers testing strategies, patterns, and best practices for KECS development.

## Testing Philosophy

KECS follows a comprehensive testing strategy:
- **Unit tests** for individual components
- **Integration tests** for component interactions
- **Scenario tests** for end-to-end workflows
- **Performance tests** for scalability validation

## Test Organization

```
kecs/
├── internal/
│   ├── controlplane/
│   │   ├── api/
│   │   │   ├── clusters_test.go      # Unit tests
│   │   │   └── services_test.go
│   │   └── managers/
│   │       └── cluster_manager_test.go
│   └── storage/
│       └── duckdb/
│           └── storage_test.go
├── tests/
│   ├── integration/                   # Integration tests
│   │   ├── api_test.go
│   │   └── storage_test.go
│   └── scenarios/                     # Scenario tests
│       ├── cluster_lifecycle_test.go
│       └── service_deployment_test.go
```

## Unit Testing

### Writing Unit Tests

Unit tests should be fast, isolated, and test a single unit of functionality.

```go
package api

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock dependencies
type MockClusterManager struct {
    mock.Mock
}

func (m *MockClusterManager) CreateCluster(ctx context.Context, input *CreateClusterInput) (*Cluster, error) {
    args := m.Called(ctx, input)
    return args.Get(0).(*Cluster), args.Error(1)
}

// Test handler
func TestClusterHandler_CreateCluster(t *testing.T) {
    tests := []struct {
        name           string
        input          interface{}
        mockSetup      func(*MockClusterManager)
        expectedStatus int
        expectedBody   interface{}
    }{
        {
            name: "successful creation",
            input: CreateClusterInput{
                ClusterName: "test-cluster",
            },
            mockSetup: func(m *MockClusterManager) {
                m.On("CreateCluster", mock.Anything, &CreateClusterInput{
                    ClusterName: "test-cluster",
                }).Return(&Cluster{
                    ClusterName: "test-cluster",
                    Status:      "ACTIVE",
                }, nil)
            },
            expectedStatus: http.StatusOK,
            expectedBody: CreateClusterOutput{
                Cluster: &Cluster{
                    ClusterName: "test-cluster",
                    Status:      "ACTIVE",
                },
            },
        },
        {
            name:           "invalid input",
            input:          "invalid",
            mockSetup:      func(m *MockClusterManager) {},
            expectedStatus: http.StatusBadRequest,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            mockManager := new(MockClusterManager)
            tt.mockSetup(mockManager)
            
            handler := &ClusterHandler{
                manager: mockManager,
            }
            
            // Create request
            body, _ := json.Marshal(tt.input)
            req := httptest.NewRequest("POST", "/v1/CreateCluster", bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")
            
            // Record response
            rr := httptest.NewRecorder()
            
            // Execute
            handler.CreateCluster(rr, req)
            
            // Assert
            assert.Equal(t, tt.expectedStatus, rr.Code)
            
            if tt.expectedBody != nil {
                var actual CreateClusterOutput
                err := json.NewDecoder(rr.Body).Decode(&actual)
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedBody, actual)
            }
            
            mockManager.AssertExpectations(t)
        })
    }
}
```

### Table-Driven Tests

Use table-driven tests for comprehensive coverage:

```go
func TestValidateClusterName(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        wantErr   bool
        errMsg    string
    }{
        {
            name:    "valid name",
            input:   "my-cluster",
            wantErr: false,
        },
        {
            name:    "empty name",
            input:   "",
            wantErr: true,
            errMsg:  "cluster name cannot be empty",
        },
        {
            name:    "name too long",
            input:   strings.Repeat("a", 256),
            wantErr: true,
            errMsg:  "cluster name cannot exceed 255 characters",
        },
        {
            name:    "invalid characters",
            input:   "my_cluster!",
            wantErr: true,
            errMsg:  "cluster name can only contain alphanumeric characters and hyphens",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateClusterName(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Testing with Ginkgo

KECS uses Ginkgo for BDD-style testing:

```go
package storage_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    
    "github.com/nandemo-ya/kecs/internal/storage"
)

var _ = Describe("DuckDBStorage", func() {
    var (
        store storage.Storage
        ctx   context.Context
    )
    
    BeforeEach(func() {
        ctx = context.Background()
        store = storage.NewDuckDBStorage(":memory:")
    })
    
    AfterEach(func() {
        store.Close()
    })
    
    Describe("Cluster Operations", func() {
        Context("when creating a cluster", func() {
            It("should store the cluster successfully", func() {
                cluster := &storage.Cluster{
                    Name:   "test-cluster",
                    Status: "ACTIVE",
                }
                
                err := store.CreateCluster(ctx, cluster)
                Expect(err).NotTo(HaveOccurred())
                
                retrieved, err := store.GetCluster(ctx, "test-cluster")
                Expect(err).NotTo(HaveOccurred())
                Expect(retrieved.Name).To(Equal("test-cluster"))
                Expect(retrieved.Status).To(Equal("ACTIVE"))
            })
            
            It("should return error for duplicate cluster", func() {
                cluster := &storage.Cluster{Name: "test-cluster"}
                
                err := store.CreateCluster(ctx, cluster)
                Expect(err).NotTo(HaveOccurred())
                
                err = store.CreateCluster(ctx, cluster)
                Expect(err).To(HaveOccurred())
                Expect(err.Error()).To(ContainSubstring("already exists"))
            })
        })
    })
})
```

## Integration Testing

### API Integration Tests

```go
// tests/integration/api_test.go
// +build integration

package integration

import (
    "context"
    "testing"
    "time"
    
    "github.com/nandemo-ya/kecs/tests/testutil"
    "github.com/stretchr/testify/suite"
)

type APIIntegrationSuite struct {
    suite.Suite
    server *testutil.TestServer
    client *testutil.TestClient
}

func (s *APIIntegrationSuite) SetupSuite() {
    // Start test server
    s.server = testutil.StartTestServer(s.T())
    s.client = testutil.NewTestClient(s.server.URL)
}

func (s *APIIntegrationSuite) TearDownSuite() {
    s.server.Stop()
}

func (s *APIIntegrationSuite) TestClusterLifecycle() {
    ctx := context.Background()
    
    // Create cluster
    createResp, err := s.client.CreateCluster(ctx, &CreateClusterInput{
        ClusterName: "integration-test",
    })
    s.Require().NoError(err)
    s.Assert().Equal("ACTIVE", createResp.Cluster.Status)
    
    // List clusters
    listResp, err := s.client.ListClusters(ctx, &ListClustersInput{})
    s.Require().NoError(err)
    s.Assert().Contains(listResp.ClusterArns, createResp.Cluster.ClusterArn)
    
    // Delete cluster
    _, err = s.client.DeleteCluster(ctx, &DeleteClusterInput{
        Cluster: "integration-test",
    })
    s.Require().NoError(err)
    
    // Verify deletion
    _, err = s.client.DescribeClusters(ctx, &DescribeClustersInput{
        Clusters: []string{"integration-test"},
    })
    s.Assert().Error(err)
}

func TestAPIIntegration(t *testing.T) {
    suite.Run(t, new(APIIntegrationSuite))
}
```

### Storage Integration Tests

```go
// tests/integration/storage_test.go
// +build integration

func TestStorageConcurrency(t *testing.T) {
    storage := setupTestStorage(t)
    defer storage.Close()
    
    const numGoroutines = 10
    const numOperations = 100
    
    var wg sync.WaitGroup
    errors := make(chan error, numGoroutines*numOperations)
    
    // Concurrent writes
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            for j := 0; j < numOperations; j++ {
                cluster := &Cluster{
                    Name: fmt.Sprintf("cluster-%d-%d", id, j),
                }
                if err := storage.CreateCluster(context.Background(), cluster); err != nil {
                    errors <- err
                }
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    // Check for errors
    for err := range errors {
        t.Errorf("Concurrent operation failed: %v", err)
    }
    
    // Verify all clusters created
    clusters, err := storage.ListClusters(context.Background(), nil)
    assert.NoError(t, err)
    assert.Len(t, clusters, numGoroutines*numOperations)
}
```

## Scenario Testing

### End-to-End Tests

```go
// tests/scenarios/service_deployment_test.go
package scenarios

import (
    "testing"
    "time"
    
    "github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

func TestServiceDeploymentScenario(t *testing.T) {
    // Start KECS
    kecs := utils.StartKECS(t)
    defer kecs.Cleanup()
    
    // Create ECS client
    client := utils.NewECSClient(kecs.Endpoint())
    
    // 1. Create cluster
    err := client.CreateCluster("test-cluster")
    require.NoError(t, err)
    
    // 2. Register task definition
    taskDefArn, err := client.RegisterTaskDefinition(&TaskDefinition{
        Family: "webapp",
        ContainerDefinitions: []ContainerDefinition{
            {
                Name:  "nginx",
                Image: "nginx:latest",
                PortMappings: []PortMapping{
                    {ContainerPort: 80},
                },
            },
        },
    })
    require.NoError(t, err)
    
    // 3. Create service
    serviceArn, err := client.CreateService(&CreateServiceInput{
        Cluster:        "test-cluster",
        ServiceName:    "webapp-service",
        TaskDefinition: taskDefArn,
        DesiredCount:   3,
    })
    require.NoError(t, err)
    
    // 4. Wait for service to stabilize
    utils.WaitForServiceStable(t, client, "test-cluster", "webapp-service", 2*time.Minute)
    
    // 5. Verify running tasks
    tasks, err := client.ListTasks(&ListTasksInput{
        Cluster:     "test-cluster",
        ServiceName: "webapp-service",
    })
    require.NoError(t, err)
    assert.Len(t, tasks, 3)
    
    // 6. Update service
    err = client.UpdateService(&UpdateServiceInput{
        Cluster:      "test-cluster",
        Service:      "webapp-service",
        DesiredCount: 5,
    })
    require.NoError(t, err)
    
    // 7. Verify scaling
    utils.WaitForTaskCount(t, client, "test-cluster", "webapp-service", 5, 2*time.Minute)
    
    // 8. Delete service
    err = client.DeleteService(&DeleteServiceInput{
        Cluster: "test-cluster",
        Service: "webapp-service",
    })
    require.NoError(t, err)
}
```

## Performance Testing

### Load Tests

```go
// tests/performance/load_test.go
// +build performance

func TestAPILoadPerformance(t *testing.T) {
    server := setupTestServer(t)
    defer server.Stop()
    
    const (
        numWorkers   = 50
        numRequests  = 1000
        duration     = 30 * time.Second
    )
    
    results := make(chan time.Duration, numWorkers*numRequests)
    errors := make(chan error, numWorkers*numRequests)
    
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()
    
    start := time.Now()
    var wg sync.WaitGroup
    
    // Start workers
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            client := NewTestClient(server.URL)
            
            for j := 0; j < numRequests; j++ {
                select {
                case <-ctx.Done():
                    return
                default:
                }
                
                reqStart := time.Now()
                _, err := client.ListClusters(ctx, &ListClustersInput{})
                reqDuration := time.Since(reqStart)
                
                if err != nil {
                    errors <- err
                } else {
                    results <- reqDuration
                }
            }
        }()
    }
    
    wg.Wait()
    close(results)
    close(errors)
    
    // Analyze results
    var (
        totalRequests int
        totalDuration time.Duration
        maxDuration   time.Duration
        durations     []time.Duration
    )
    
    for d := range results {
        totalRequests++
        totalDuration += d
        durations = append(durations, d)
        if d > maxDuration {
            maxDuration = d
        }
    }
    
    avgDuration := totalDuration / time.Duration(totalRequests)
    elapsed := time.Since(start)
    rps := float64(totalRequests) / elapsed.Seconds()
    
    // Calculate percentiles
    sort.Slice(durations, func(i, j int) bool {
        return durations[i] < durations[j]
    })
    
    p50 := durations[len(durations)*50/100]
    p95 := durations[len(durations)*95/100]
    p99 := durations[len(durations)*99/100]
    
    t.Logf("Performance Results:")
    t.Logf("Total requests: %d", totalRequests)
    t.Logf("Requests per second: %.2f", rps)
    t.Logf("Average latency: %v", avgDuration)
    t.Logf("P50 latency: %v", p50)
    t.Logf("P95 latency: %v", p95)
    t.Logf("P99 latency: %v", p99)
    t.Logf("Max latency: %v", maxDuration)
    
    // Assert performance requirements
    assert.Greater(t, rps, 1000.0, "Should handle >1000 RPS")
    assert.Less(t, p95, 100*time.Millisecond, "P95 should be <100ms")
}
```

## Test Utilities

### Test Helpers

```go
// tests/testutil/helpers.go
package testutil

// Start test server with custom configuration
func StartTestServer(t *testing.T, opts ...ServerOption) *TestServer {
    t.Helper()
    
    config := &ServerConfig{
        Port:     0, // Random port
        LogLevel: "error",
        Storage:  ":memory:",
    }
    
    for _, opt := range opts {
        opt(config)
    }
    
    server := &TestServer{
        config: config,
    }
    
    server.Start(t)
    return server
}

// Wait for condition with timeout
func WaitFor(t *testing.T, condition func() bool, timeout time.Duration) {
    t.Helper()
    
    deadline := time.Now().Add(timeout)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if condition() {
                return
            }
            if time.Now().After(deadline) {
                t.Fatal("Timeout waiting for condition")
            }
        }
    }
}
```

## Running Tests

### Commands

```bash
# Unit tests
make test

# Integration tests
make test-integration

# Scenario tests
cd tests/scenarios
make test

# Performance tests
make test-performance

# All tests with coverage
make test-all-coverage

# Specific package
go test -v ./internal/storage/...

# With race detector
go test -race ./...

# Ginkgo tests
cd controlplane
ginkgo -r

```

### CI/CD Integration

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Install dependencies
        run: |
          make deps
      
      - name: Run unit tests
        run: make test-coverage
      
      - name: Run integration tests
        run: make test-integration
      
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
```

## Best Practices

1. **Fast Tests**: Keep unit tests under 100ms
2. **Isolation**: Tests should not depend on external services
3. **Deterministic**: Tests should produce same results every run
4. **Clear Names**: Test names should describe what they test
5. **Test Coverage**: Aim for >80% coverage
6. **Mock External Dependencies**: Use interfaces for testability
7. **Parallel Execution**: Use `t.Parallel()` where appropriate
8. **Cleanup**: Always clean up resources after tests

## Next Steps

- [Building Guide](./building) - Build and packaging
- [Contributing](./contributing) - Contribution guidelines