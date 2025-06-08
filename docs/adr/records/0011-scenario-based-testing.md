# ADR-0011: Scenario-Based Testing Framework

Date: 2025-06-08

## Status

Proposed

## Context

To ensure KECS quality and validate its behavior in production environments, we need end-to-end scenario tests in addition to unit tests. These tests should mimic real operational scenarios and verify AWS ECS compatibility.

### Requirements
- Execute test scenarios that simulate production usage
- Verify compatibility with AWS ECS CLI tools
- Run automated tests in isolated environments
- Enable integration with CI pipelines

## Decision

### Test Execution Environment

1. **Adopt Testcontainers**
   - Run KECS control plane as a container
   - Provide isolated environment for each test
   - Automatic cleanup after test completion

2. **Operation Interfaces**
   - Use AWS ECS CLI (`aws ecs`) commands
   - Support ecspresso tool operations
   - Ensure compatibility by testing with production tools

3. **ecspresso Scenarios**
   - Deploy using service definition files
   - Simultaneous task definition and service updates
   - Real-time deployment monitoring
   - Rollback operations

### Test Scenario Structure

#### 1. Cluster Management
- **Create**: `aws ecs create-cluster`
- **Describe**: `aws ecs describe-clusters`
- **List**: `aws ecs list-clusters`
- **Delete**: `aws ecs delete-cluster`

#### 2. Task Definition Management
- **Register**: `aws ecs register-task-definition`
- **Describe**: `aws ecs describe-task-definition`
- **List**: `aws ecs list-task-definitions`
- **Update**: Register new revision
- **Deregister**: `aws ecs deregister-task-definition`

#### 3. Service Management
- **Create**: `aws ecs create-service`
- **Describe**: `aws ecs describe-services`
- **Update**: `aws ecs update-service`
- **Delete**: `aws ecs delete-service`

#### 4. Happy Path Scenarios
- **Service Launch Flow**
  1. Register task definition
  2. Create service
  3. Verify task status transitions
     - PROVISIONING → PENDING → RUNNING
     - Appropriate wait times for each status
  4. Verify health check success
  5. Verify task details (IP address, start time, etc.)
  
- **Rolling Update**
  1. Register new task definition revision
  2. Update service
  3. Verify old and new tasks running concurrently
  4. Verify gradual stop of old tasks
  5. Verify complete migration to new tasks

#### 5. Failure Scenarios
- **Task Failure**
  1. Use container image that exits abnormally
  2. Verify automatic task restart
  3. Verify restart limit enforcement
  
- **Health Check Failure**
  1. Launch task with failing health check
  2. Verify unhealthy task detection
  3. Verify automatic task replacement
  4. Verify service maintains minimum task count

### Test Framework Structure

```
tests/
├── scenarios/
│   ├── cluster/
│   │   ├── create_test.go
│   │   ├── describe_test.go
│   │   └── delete_test.go
│   ├── task_definition/
│   │   ├── register_test.go
│   │   ├── update_test.go
│   │   └── deregister_test.go
│   ├── service/
│   │   ├── create_test.go
│   │   ├── update_test.go
│   │   ├── rolling_update_test.go
│   │   └── delete_test.go
│   ├── task/
│   │   ├── task_lifecycle_test.go
│   │   ├── task_status_test.go
│   │   └── run_task_test.go
│   ├── ecspresso/
│   │   ├── deploy_test.go
│   │   ├── rollback_test.go
│   │   ├── diff_test.go
│   │   └── verify_test.go
│   └── failure/
│       ├── task_failure_test.go
│       └── health_check_failure_test.go
├── fixtures/
│   ├── task-definitions/
│   │   ├── nginx.json
│   │   ├── failing-app.json
│   │   └── unhealthy-app.json
│   ├── services/
│   │   ├── basic-service.json
│   │   └── multi-replica-service.json
│   └── ecspresso/
│       ├── ecs-service-def.json
│       ├── ecs-task-def.json
│       └── config.yaml
├── utils/
│   ├── testcontainers.go
│   ├── aws_ecs_client.go
│   ├── ecspresso_client.go
│   ├── task_status_checker.go
│   └── assertions.go
└── README.md
```

### Implementation Approach

1. **Testcontainers Setup**
   ```go
   func setupKECS(t *testing.T) (string, func()) {
       ctx := context.Background()
       
       // Start KECS container
       req := testcontainers.ContainerRequest{
           Image:        "kecs:latest",
           ExposedPorts: []string{"8080/tcp"},
           WaitingFor:   wait.ForHTTP("/health").WithPort("8080"),
       }
       
       container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
           ContainerRequest: req,
           Started:          true,
       })
       require.NoError(t, err)
       
       // Get endpoint URL
       endpoint, err := container.Endpoint(ctx, "http")
       require.NoError(t, err)
       
       // Cleanup function
       cleanup := func() {
           container.Terminate(ctx)
       }
       
       return endpoint, cleanup
   }
   ```

2. **AWS CLI Wrapper**
   ```go
   type ECSClient struct {
       endpoint string
   }
   
   func (c *ECSClient) CreateCluster(name string) error {
       cmd := exec.Command("aws", "ecs", "create-cluster",
           "--cluster-name", name,
           "--endpoint-url", c.endpoint)
       return cmd.Run()
   }
   ```

3. **Assertions**
   ```go
   func assertTaskRunning(t *testing.T, client *ECSClient, cluster, service string) {
       // Periodically check task status
       require.Eventually(t, func() bool {
           tasks := client.ListTasks(cluster, service)
           for _, task := range tasks {
               if task.LastStatus == "RUNNING" {
                   return true
               }
           }
           return false
       }, 30*time.Second, 1*time.Second)
   }
   ```

## Consequences

### Positive
- High reliability by testing with production tools
- Parallel execution possible with isolated environments
- Easy CI pipeline integration
- End-to-end behavior validation

### Negative
- Docker required in test environment due to Testcontainers
- Longer test execution time compared to unit tests
- AWS CLI installation prerequisite

### Risks
- Kind cluster startup may fail in test environments
- Network issues may cause unstable container communication

## Implementation Plan

1. **Phase 1: Foundation**
   - Testcontainers setup
   - AWS CLI wrapper implementation
   - Basic assertion functions

2. **Phase 2: Basic Scenarios**
   - Cluster management tests
   - Task definition management tests
   - Service management tests (basic operations)

3. **Phase 3: Advanced Scenarios**
   - Rolling update tests
   - Failure scenario tests
   - Performance tests

4. **Phase 4: CI Integration**
   - GitHub Actions integration
   - Test result reporting
   - Coverage measurement

## References

- [Testcontainers for Go](https://golang.testcontainers.org/)
- [AWS ECS CLI Reference](https://docs.aws.amazon.com/cli/latest/reference/ecs/)
- [ecspresso](https://github.com/kayac/ecspresso)