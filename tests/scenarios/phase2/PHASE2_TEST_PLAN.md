# Phase 2 Test Plan: Task Definitions and Services

## Overview
Phase 2 focuses on testing task definitions, services, and running tasks with real container workloads. Each test file will have its own KECS container and ECS cluster for complete isolation.

## Test Architecture
- **KECS Container**: One per _test.go file
- **ECS Cluster**: One per _test.go file (complete isolation)
- **Namespace**: One per service to ensure isolation within the cluster

## Test Categories

### 1. Single Container Patterns

#### 1.1 Simple Web Application (nginx)
**File**: `task_simple_web_test.go`
- **Task Definition**: nginx:alpine with custom index.html
- **Service**: Desired count 2, with load balancing
- **Tests**:
  - Register task definition
  - Create service
  - Verify 2 tasks are running
  - HTTP health check via curl
  - Scale service up/down
  - Update task definition (new nginx version)
  - Delete service and task definition

#### 1.2 Background Worker (Python)
**File**: `task_background_worker_test.go`
- **Task Definition**: Python script that processes queue (simulated)
- **Service**: Desired count 1
- **Tests**:
  - Register task definition with environment variables
  - Create service
  - Verify task is processing (check logs)
  - Stop task and verify restart
  - Update environment variables
  - Delete service and task definition

#### 1.3 Failure Test Application
**File**: `task_failure_handling_test.go`
- **Task Definition**: Container that exits with error after 30s
- **Service**: Desired count 1, with restart policy
- **Tests**:
  - Register task definition
  - Create service
  - Verify task starts
  - Wait for failure
  - Verify automatic restart
  - Check restart count in service events
  - Delete service and task definition

#### 1.4 Health Check Failure Test
**File**: `task_health_check_test.go`
- **Task Definition**: Web app with failing health check endpoint
- **Service**: Desired count 2
- **Tests**:
  - Register task definition with health check
  - Create service
  - Verify tasks are marked unhealthy
  - Verify service tries to maintain desired count
  - Fix health check endpoint (update task)
  - Verify tasks become healthy
  - Delete service and task definition

### 2. Two Container Patterns

#### 2.1 Nginx + Web Application
**File**: `task_nginx_webapp_test.go`
- **Task Definition**: 
  - Container 1: nginx as reverse proxy
  - Container 2: simple web app (e.g., httpd)
- **Service**: Desired count 1
- **Tests**:
  - Register task definition with container links
  - Create service
  - Verify both containers are running
  - Test nginx forwards requests to web app
  - Update web app container only
  - Verify nginx still works with new app
  - Delete service and task definition

#### 2.2 API Application + Redis
**File**: `task_api_redis_test.go`
- **Task Definition**:
  - Container 1: Simple API (e.g., Node.js)
  - Container 2: Redis
- **Service**: Desired count 1
- **Tests**:
  - Register task definition with container dependencies
  - Create service
  - Verify Redis starts before API
  - Test API can connect to Redis
  - Store/retrieve data via API
  - Restart task and verify data persistence
  - Delete service and task definition

#### 2.3 Init Container Pattern
**File**: `task_init_container_test.go`
- **Task Definition**:
  - Container 1: Init container (essential: false)
  - Container 2: Main application
- **Service**: Desired count 1
- **Tests**:
  - Register task definition with init container
  - Create service
  - Verify init container runs first
  - Verify init container exits successfully
  - Verify main container starts after init
  - Update init container script
  - Verify new initialization works
  - Delete service and task definition

## Test Implementation Structure

### Base Test Structure
```go
// Each test file will have its own BeforeSuite/AfterSuite
var _ = BeforeSuite(func() {
    // Start KECS container and create cluster for this test file
    sharedKECS = utils.StartKECS(GinkgoT())
    sharedClient = utils.NewECSClient(sharedKECS.Endpoint())
    
    // Create cluster specific to this test file
    clusterName = utils.GenerateTestName("phase2-cluster")
    err := sharedClient.CreateCluster(clusterName)
    Expect(err).NotTo(HaveOccurred())
    
    utils.AssertClusterActive(GinkgoT(), sharedClient, clusterName)
})

var _ = AfterSuite(func() {
    // Clean up cluster and KECS container
    if sharedClient != nil && clusterName != "" {
        _ = sharedClient.DeleteCluster(clusterName)
    }
    if sharedKECS != nil {
        sharedKECS.Cleanup()
    }
})

var _ = Describe("Task Definition: [Pattern Name]", Serial, func() {
    var (
        taskDefFamily  string
        serviceName    string
    )

    BeforeEach(func() {
        taskDefFamily = utils.GenerateTestName("td")
        serviceName = utils.GenerateTestName("svc")
    })

    AfterEach(func() {
        // Cleanup: Delete service first, then task definition
        _ = sharedClient.DeleteService(clusterName, serviceName)
        _ = sharedClient.DeregisterTaskDefinition(taskDefFamily)
    })

    // Test implementations...
})
```

### Common Test Patterns

1. **Task Definition Registration**
   ```go
   It("should register task definition", func() {
       taskDef := loadTaskDefinition("templates/simple-web.json")
       resp := client.RegisterTaskDefinition(taskDef)
       Expect(resp.Family).To(Equal(taskDefFamily))
   })
   ```

2. **Service Creation and Verification**
   ```go
   It("should create service and run tasks", func() {
       client.CreateService(clusterName, serviceName, taskDefFamily, 2)
       Eventually(func() int {
           return countRunningTasks(client, clusterName, serviceName)
       }, 60*time.Second).Should(Equal(2))
   })
   ```

3. **Application Testing**
   ```go
   It("should respond to HTTP requests", func() {
       taskIP := getTaskIP(client, clusterName, serviceName)
       resp := httpGet(fmt.Sprintf("http://%s:80", taskIP))
       Expect(resp.StatusCode).To(Equal(200))
   })
   ```

## Task Definition Templates

### Directory Structure
```
phase2/
├── templates/
│   ├── single-container/
│   │   ├── simple-web.json
│   │   ├── background-worker.json
│   │   ├── failure-app.json
│   │   └── health-check-fail.json
│   └── multi-container/
│       ├── nginx-webapp.json
│       ├── api-redis.json
│       └── init-container.json
├── task_simple_web_test.go
├── task_background_worker_test.go
├── task_failure_handling_test.go
├── task_health_check_test.go
├── task_nginx_webapp_test.go
├── task_api_redis_test.go
├── task_init_container_test.go
└── phase2_suite_test.go
```

## Success Criteria

1. **Basic Operations**
   - All task definitions can be registered/deregistered
   - All services can be created/updated/deleted
   - Task states are correctly reported

2. **Runtime Verification**
   - Single container apps work correctly
   - Multi-container apps can communicate
   - Health checks function properly
   - Failure handling works as expected

3. **API Compliance**
   - AWS CLI commands work correctly
   - Task/service states match ECS behavior
   - Error messages are appropriate

## Implementation Priority

1. **Phase 2.1**: Single container patterns (simple-web, background-worker)
2. **Phase 2.2**: Failure handling patterns
3. **Phase 2.3**: Multi-container patterns
4. **Phase 2.4**: Advanced patterns (init containers)

## Notes

- Use lightweight container images (alpine-based)
- Keep task definitions simple and focused
- Ensure proper cleanup after each test
- Use Eventually() for async operations
- Log important state transitions for debugging