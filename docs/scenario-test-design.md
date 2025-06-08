# KECS Scenario Test Design Document

## Overview

This document describes the detailed design of KECS scenario tests. The scenario tests aim to reproduce actual AWS ECS operational scenarios and ensure KECS compatibility and quality.

## Test Scenario Details

### 1. Cluster Management Scenarios

#### 1.1 Cluster Lifecycle
```gherkin
Scenario: Cluster lifecycle from creation to deletion
  Given KECS is running
  When I create a cluster named "test-cluster"
  Then the cluster should be created with "ACTIVE" status
  And kind cluster "kecs-test-cluster" should exist
  
  When I describe the cluster
  Then cluster details should be returned correctly
  
  When I delete the cluster
  Then the cluster should be deleted
  And the kind cluster should also be deleted
```

#### 1.2 Multiple Cluster Management
```gherkin
Scenario: Managing multiple clusters simultaneously
  Given KECS is running
  When I create 3 clusters
  Then all clusters should be in "ACTIVE" status
  
  When I list clusters
  Then 3 clusters should be displayed
```

### 2. Task Definition Scenarios

#### 2.1 Task Definition Registration and Updates
```gherkin
Scenario: Task definition registration and revision management
  Given a cluster exists
  When I register a task definition with nginx:1.19
  Then task definition "nginx:1" should be created
  
  When I register an updated task definition with nginx:1.20
  Then task definition "nginx:2" should be created
  
  When I list task definitions
  Then 2 revisions should be displayed
```

#### 2.2 Complex Task Definitions
```gherkin
Scenario: Multi-container task definition
  Given a cluster exists
  When I register a task definition with the following containers:
    | Container Name | Image | Port | Environment Variables |
    | web | nginx:latest | 80 | - |
    | app | app:latest | 3000 | DB_HOST=localhost |
    | sidecar | fluentd:latest | - | - |
  Then the task definition should be registered correctly
```

### 3. Service Management Scenarios

#### 3.1 Basic Service Creation
```gherkin
Scenario: Service creation and task launch
  Given a cluster and task definition exist
  When I create a service with desiredCount=2
  Then the service should become "ACTIVE"
  And tasks should transition through the following statuses:
    | Status | Description | Expected Time |
    | PROVISIONING | Resource allocation | 0-5s |
    | PENDING | Container image pull | 5-30s |
    | ACTIVATING | Container startup | 30-45s |
    | RUNNING | Normal operation | 45s+ |
  And 2 tasks should be in "RUNNING" state
  And all tasks should pass health checks
  And task details should be retrievable:
    | Field | Validation |
    | privateIpAddress | Valid IP address |
    | startedAt | Close to current time |
    | cpu/memory | As defined |
```

#### 3.2 Rolling Update
```gherkin
Scenario: Service rolling update
  Given nginx service is running with revision 1
  And 2 tasks are running normally
  
  When I update the service to revision 2
  Then new tasks should start launching
  And old and new tasks should run concurrently
  And new tasks should pass health checks
  And old tasks should stop gradually
  And eventually all tasks should be revision 2
  And the service should remain available throughout
```

#### 3.3 Service Scaling
```gherkin
Scenario: Service scale up and scale down
  Given a service is running with desiredCount=2
  
  When I change desiredCount to 5
  Then 3 new tasks should launch
  And total 5 tasks should be running
  
  When I change desiredCount to 1
  Then 4 tasks should stop
  And only 1 task should remain running
```

### 4. Failure Scenarios

#### 4.1 Task Failure
```gherkin
Scenario: Automatic recovery when task fails
  Given a service running an application that exits with exit(1)
  And desiredCount=3
  
  When tasks launch
  Then tasks should transition from "RUNNING" to "STOPPED"
  And stop reason should be "Essential container in task exited"
  And new tasks should launch automatically
  And this restart cycle should repeat
```

#### 4.2 Health Check Failure
```gherkin
Scenario: Task replacement due to health check failure
  Given a service with health check endpoint returning 404
  And health check configuration:
    | Parameter | Value |
    | interval | 30s |
    | timeout | 5s |
    | retries | 3 |
    | startPeriod | 60s |
  
  When I launch the service
  Then tasks should become "RUNNING"
  And health checks should start after startPeriod
  And health checks should fail 3 consecutive times
  And tasks should be marked as "UNHEALTHY"
  And tasks should be stopped
  And new tasks should be launched
```

#### 4.3 Resource Constraints
```gherkin
Scenario: Tasks cannot be placed due to resource constraints
  Given a cluster with limited available resources
  When I create a service requesting large amounts of memory
  Then tasks should remain in "PENDING" state
  And events should contain "no container instances meet all requirements"
```

### 5. Task Management Scenarios

#### 5.1 RunTask for One-off Execution
```gherkin
Scenario: Batch job execution with RunTask
  Given a cluster and batch job task definition exist
  When I launch one task with RunTask
  Then the task should transition through statuses:
    | Status | Verification |
    | PROVISIONING | Task ARN is generated |
    | PENDING | Container instance is assigned |
    | RUNNING | startedAt is recorded |
    | STOPPED | stoppedAt is recorded, exitCode is 0 |
  And task execution time should be recorded
  And logs should be output to CloudWatch Logs
```

#### 5.2 Force Stop Task
```gherkin
Scenario: Force stop a running task
  Given a long-running task is active
  When I execute StopTask
  Then task desiredStatus should become "STOPPED"
  And the task should be cleaned up properly
  And stopCode should be "UserInitiated"
```

### 6. ecspresso Scenarios

#### 6.1 Basic Deployment
```gherkin
Scenario: Service deployment with ecspresso
  Given ecspresso configuration file exists:
    """
    region: ap-northeast-1
    cluster: test-cluster
    service: webapp
    service_definition: ecs-service-def.json
    task_definition: ecs-task-def.json
    """
  
  When I execute ecspresso deploy
  Then task definition should be registered
  And service should be created or updated
  And deployment progress should be displayed
  And all tasks should launch successfully
```

#### 6.2 Deployment Verification and Rollback
```gherkin
Scenario: Deployment verification and rollback with ecspresso
  Given current service is running with revision 1
  
  When I execute ecspresso diff
  Then differences between current state and new definition should be displayed
  
  When I execute ecspresso verify
  Then task definition and service definition validity should be verified
  
  When I deploy problematic revision 2
  And tasks fail to launch
  
  When I execute ecspresso rollback
  Then service should revert to revision 1
  And all tasks should run normally
```

#### 6.3 Task Execution with ecspresso
```gherkin
Scenario: Task execution and monitoring with ecspresso
  Given ecspresso configuration and task definition exist
  
  When I execute ecspresso run --watch
  Then task should launch
  And task logs should be displayed in real-time
  And monitoring should continue until task completion
  And exit code should be displayed
```

### 7. Advanced Scenarios

#### 7.1 Blue/Green Deployment
```gherkin
Scenario: Blue/Green deployment pattern
  Given "app-blue" service is running
  And load balancer is pointing to "app-blue"
  
  When I launch "app-green" service with new version
  Then both services should run concurrently
  
  When I switch load balancer to "app-green"
  Then traffic should flow to new version
  
  When I delete "app-blue" service
  Then only "app-green" should be running
```

#### 7.2 Service Discovery
```gherkin
Scenario: Inter-service communication
  Given "backend" service is running
  When I launch "frontend" service
  And "frontend" attempts to connect to "backend"
  Then connection should succeed through service discovery
```

## Test Implementation Examples

### Testcontainers Setup
```go
func TestServiceLifecycle(t *testing.T) {
    ctx := context.Background()
    
    // Start KECS container
    kecsContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "kecs:latest",
            ExposedPorts: []string{"8080/tcp"},
            Env: map[string]string{
                "LOG_LEVEL": "debug",
            },
            WaitingFor: wait.ForAll(
                wait.ForHTTP("/health").WithPort("8081"),
                wait.ForLog("API server started"),
            ),
        },
        Started: true,
    })
    require.NoError(t, err)
    defer kecsContainer.Terminate(ctx)
    
    // Get endpoint
    endpoint, err := kecsContainer.Endpoint(ctx, "http")
    require.NoError(t, err)
    
    // Configure AWS CLI
    os.Setenv("AWS_ENDPOINT_URL", endpoint)
    
    // Execute test scenarios
    t.Run("CreateCluster", func(t *testing.T) {
        // ...
    })
}
```

### Assertion Examples
```go
func assertServiceHealthy(t *testing.T, cluster, service string, timeout time.Duration) {
    deadline := time.Now().Add(timeout)
    
    for time.Now().Before(deadline) {
        output, err := exec.Command("aws", "ecs", "describe-services",
            "--cluster", cluster,
            "--services", service,
        ).Output()
        
        if err == nil {
            var result map[string]interface{}
            json.Unmarshal(output, &result)
            
            services := result["services"].([]interface{})
            if len(services) > 0 {
                svc := services[0].(map[string]interface{})
                if svc["status"] == "ACTIVE" && 
                   svc["runningCount"] == svc["desiredCount"] {
                    return
                }
            }
        }
        
        time.Sleep(1 * time.Second)
    }
    
    t.Fatalf("Service %s/%s did not become healthy within %v", cluster, service, timeout)
}
```

### Task Status Checker
```go
type TaskStatusChecker struct {
    client *ECSClient
}

func (c *TaskStatusChecker) WaitForStatus(t *testing.T, cluster, taskArn string, expectedStatus string, timeout time.Duration) {
    statusHistory := []string{}
    deadline := time.Now().Add(timeout)
    
    for time.Now().Before(deadline) {
        task, err := c.client.DescribeTask(cluster, taskArn)
        if err != nil {
            t.Logf("Error describing task: %v", err)
            time.Sleep(1 * time.Second)
            continue
        }
        
        currentStatus := task.LastStatus
        if len(statusHistory) == 0 || statusHistory[len(statusHistory)-1] != currentStatus {
            statusHistory = append(statusHistory, currentStatus)
            t.Logf("Task status transition: %s -> %s", 
                strings.Join(statusHistory, " -> "), currentStatus)
        }
        
        if currentStatus == expectedStatus {
            // Additional validations
            assert.NotEmpty(t, task.TaskArn)
            if expectedStatus == "RUNNING" {
                assert.NotNil(t, task.StartedAt)
                assert.NotEmpty(t, task.Containers[0].NetworkInterfaces[0].PrivateIpv4Address)
            }
            return
        }
        
        time.Sleep(2 * time.Second)
    }
    
    t.Fatalf("Task did not reach %s status within %v. Status history: %v", 
        expectedStatus, timeout, statusHistory)
}
```

### ecspresso Client
```go
type EcspressoClient struct {
    configPath string
    endpoint   string
}

func (e *EcspressoClient) Deploy(t *testing.T) error {
    cmd := exec.Command("ecspresso", "deploy",
        "--config", e.configPath,
        "--endpoint", e.endpoint,
        "--no-wait") // Don't wait in tests
    
    output, err := cmd.CombinedOutput()
    t.Logf("ecspresso deploy output:\n%s", string(output))
    return err
}

func (e *EcspressoClient) Verify(t *testing.T) error {
    cmd := exec.Command("ecspresso", "verify",
        "--config", e.configPath,
        "--endpoint", e.endpoint)
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        t.Logf("ecspresso verify failed:\n%s", string(output))
    }
    return err
}

func (e *EcspressoClient) Run(t *testing.T, overrides map[string]string) (string, error) {
    args := []string{"run",
        "--config", e.configPath,
        "--endpoint", e.endpoint,
        "--watch"} // Monitor logs
    
    // Add overrides
    for k, v := range overrides {
        args = append(args, "--overrides", fmt.Sprintf("%s=%s", k, v))
    }
    
    cmd := exec.Command("ecspresso", args...)
    output, err := cmd.CombinedOutput()
    
    // Extract task ARN
    taskArn := extractTaskArn(string(output))
    return taskArn, err
}
```

## CI/CD Integration

### GitHub Actions Example
```yaml
name: Scenario Tests

on:
  pull_request:
  push:
    branches: [main]

jobs:
  scenario-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: 1.21
      
      - name: Install AWS CLI
        run: |
          curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
          unzip awscliv2.zip
          sudo ./aws/install
      
      - name: Build KECS image
        run: |
          cd controlplane
          docker build -t kecs:latest .
      
      - name: Run scenario tests
        run: |
          cd tests/scenarios
          go test -v -timeout 30m ./...
      
      - name: Upload test results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: tests/scenarios/results/
```

## Metrics and Reporting

### Measurement Items
- Execution time for each scenario
- Task startup time
- Rolling update completion time
- API response time
- Resource usage (CPU, memory)

### Report Format
```json
{
  "timestamp": "2025-01-06T10:00:00Z",
  "duration": "15m32s",
  "scenarios": {
    "cluster_lifecycle": {
      "status": "passed",
      "duration": "45s",
      "steps": [
        {
          "name": "create_cluster",
          "duration": "12s",
          "status": "passed"
        }
      ]
    }
  },
  "metrics": {
    "task_startup_time_p50": "8s",
    "task_startup_time_p99": "15s",
    "api_response_time_p50": "50ms",
    "api_response_time_p99": "200ms"
  }
}