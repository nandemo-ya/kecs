# Control Plane Architecture

## Overview

The KECS control plane is the brain of the system, implementing the business logic for ECS resources and coordinating between the API layer and the underlying Kubernetes infrastructure.

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Control Plane                            │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    API Handlers                          │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐ │   │
│  │  │ Cluster  │  │ Service  │  │  Task    │  │  Task  │ │   │
│  │  │ Handler  │  │ Handler  │  │ Handler  │  │  Def   │ │   │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └───┬────┘ │   │
│  └───────┼──────────────┼─────────────┼────────────┼──────┘   │
│          │              │             │            │            │
│  ┌───────▼──────────────▼─────────────▼────────────▼──────┐   │
│  │                  Business Logic Layer                   │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐   │   │
│  │  │   Cluster   │  │   Service   │  │    Task      │   │   │
│  │  │  Manager    │  │   Manager   │  │   Manager    │   │   │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬───────┘   │   │
│  └─────────┼────────────────┼────────────────┼────────────┘   │
│            │                │                │                 │
│  ┌─────────▼────────────────▼────────────────▼─────────────┐  │
│  │              State Management & Persistence              │  │
│  │  ┌────────────┐  ┌────────────┐  ┌─────────────────┐  │  │
│  │  │   Cache    │  │  Storage   │  │   Event Bus     │  │  │
│  │  │   Layer    │  │  Interface │  │  (WebSocket)    │  │  │
│  │  └────────────┘  └────────────┘  └─────────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │                 Kubernetes Abstraction                    │ │
│  │  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │ │
│  │  │  Resource   │  │   Status     │  │   Event       │  │ │
│  │  │  Converter  │  │   Watcher    │  │   Handler     │  │ │
│  │  └─────────────┘  └──────────────┘  └───────────────┘  │ │
│  └──────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. API Handlers

API handlers are the entry points for all ECS API requests. They:

- Parse and validate incoming requests
- Transform request/response formats
- Handle AWS API conventions (headers, error formats)
- Delegate to appropriate business logic managers

#### Implementation Pattern

```go
type ClusterHandler struct {
    manager *managers.ClusterManager
    logger  *zap.Logger
}

func (h *ClusterHandler) CreateCluster(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request
    var req generated.CreateClusterInput
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, "InvalidParameterException", err.Error())
        return
    }

    // 2. Validate request
    if err := h.validateCreateClusterInput(&req); err != nil {
        writeError(w, "InvalidParameterException", err.Error())
        return
    }

    // 3. Call business logic
    cluster, err := h.manager.CreateCluster(r.Context(), &req)
    if err != nil {
        handleBusinessError(w, err)
        return
    }

    // 4. Return response
    writeResponse(w, &generated.CreateClusterOutput{
        Cluster: cluster,
    })
}
```

### 2. Business Logic Managers

Managers implement the core business logic for each resource type:

#### Cluster Manager

- Creates and manages ECS clusters
- Maps clusters to Kubernetes namespaces
- Handles cluster settings and configuration
- Manages cluster lifecycle (ACTIVE, PROVISIONING, etc.)

#### Service Manager

- Creates and manages long-running services
- Handles service scaling and updates
- Manages deployments and rollbacks
- Integrates with load balancers and service discovery

#### Task Manager

- Runs one-off tasks
- Manages task lifecycle and state transitions
- Handles task placement and scheduling
- Monitors container status

#### Task Definition Manager

- Stores and versions task definitions
- Validates container definitions
- Manages task definition families and revisions
- Handles deregistration and cleanup

### 3. State Management

#### Storage Interface

```go
type Storage interface {
    // Cluster operations
    CreateCluster(ctx context.Context, cluster *Cluster) error
    GetCluster(ctx context.Context, name string) (*Cluster, error)
    UpdateCluster(ctx context.Context, cluster *Cluster) error
    DeleteCluster(ctx context.Context, name string) error
    ListClusters(ctx context.Context, filter *ClusterFilter) ([]*Cluster, error)
    
    // Service operations
    CreateService(ctx context.Context, service *Service) error
    GetService(ctx context.Context, clusterName, serviceName string) (*Service, error)
    UpdateService(ctx context.Context, service *Service) error
    DeleteService(ctx context.Context, clusterName, serviceName string) error
    ListServices(ctx context.Context, clusterName string) ([]*Service, error)
    
    // Task operations
    CreateTask(ctx context.Context, task *Task) error
    GetTask(ctx context.Context, taskArn string) (*Task, error)
    UpdateTask(ctx context.Context, task *Task) error
    ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, error)
    
    // Task definition operations
    RegisterTaskDefinition(ctx context.Context, taskDef *TaskDefinition) error
    GetTaskDefinition(ctx context.Context, family string, revision int) (*TaskDefinition, error)
    DeregisterTaskDefinition(ctx context.Context, family string, revision int) error
    ListTaskDefinitions(ctx context.Context, family string) ([]*TaskDefinition, error)
}
```

#### Cache Layer

The cache layer provides:

- In-memory caching with LRU eviction
- TTL-based expiration
- Write-through caching for consistency
- Cache invalidation on updates

```go
type CachedStorage struct {
    storage Storage
    cache   *MemoryCache
}

func (cs *CachedStorage) GetCluster(ctx context.Context, name string) (*Cluster, error) {
    // Check cache first
    if cached, ok := cs.cache.Get(fmt.Sprintf("cluster:%s", name)); ok {
        return cached.(*Cluster), nil
    }
    
    // Fetch from storage
    cluster, err := cs.storage.GetCluster(ctx, name)
    if err != nil {
        return nil, err
    }
    
    // Update cache
    cs.cache.Set(fmt.Sprintf("cluster:%s", name), cluster, 5*time.Minute)
    return cluster, nil
}
```

### 4. Event System

The event system provides real-time updates via WebSocket:

#### Event Types

```go
type EventType string

const (
    EventTypeClusterCreated     EventType = "cluster.created"
    EventTypeClusterUpdated     EventType = "cluster.updated"
    EventTypeClusterDeleted     EventType = "cluster.deleted"
    EventTypeServiceCreated     EventType = "service.created"
    EventTypeServiceUpdated     EventType = "service.updated"
    EventTypeServiceDeleted     EventType = "service.deleted"
    EventTypeTaskStarted        EventType = "task.started"
    EventTypeTaskStopped        EventType = "task.stopped"
    EventTypeTaskStatusChanged  EventType = "task.status_changed"
    EventTypeDeploymentStarted  EventType = "deployment.started"
    EventTypeDeploymentComplete EventType = "deployment.complete"
    EventTypeDeploymentFailed   EventType = "deployment.failed"
)
```

#### Event Publishing

```go
type EventBus interface {
    Publish(event Event) error
    Subscribe(eventTypes []EventType, handler EventHandler) (Subscription, error)
    Unsubscribe(subscription Subscription) error
}

type Event struct {
    ID        string
    Type      EventType
    Timestamp time.Time
    Resource  string
    Data      interface{}
}
```

## Resource Lifecycle

### Service Lifecycle

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   CREATE    │────▶│   PENDING    │────▶│   ACTIVE    │
└─────────────┘     └──────────────┘     └──────┬──────┘
                                                 │
                                                 │ UPDATE
                                                 ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   DELETED   │◀────│   DRAINING   │◀────│  UPDATING   │
└─────────────┘     └──────────────┘     └─────────────┘
```

### Task Lifecycle

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│PROVISIONING │────▶│   PENDING    │────▶│ ACTIVATING  │
└─────────────┘     └──────────────┘     └──────┬──────┘
                                                 │
                                                 ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   STOPPED   │◀────│  STOPPING    │◀────│   RUNNING   │
└─────────────┘     └──────────────┘     └─────────────┘
```

## Concurrency Model

### Request Processing

- Each HTTP request is handled in its own goroutine
- Managers use context for cancellation and timeout
- Database operations are serialized through connection pool
- Kubernetes operations can be parallelized

### Resource Locking

```go
type ResourceLock struct {
    mu    sync.RWMutex
    locks map[string]*sync.Mutex
}

func (rl *ResourceLock) Lock(resourceID string) {
    rl.mu.Lock()
    if _, exists := rl.locks[resourceID]; !exists {
        rl.locks[resourceID] = &sync.Mutex{}
    }
    lock := rl.locks[resourceID]
    rl.mu.Unlock()
    
    lock.Lock()
}

func (rl *ResourceLock) Unlock(resourceID string) {
    rl.mu.RLock()
    lock, exists := rl.locks[resourceID]
    rl.mu.RUnlock()
    
    if exists {
        lock.Unlock()
    }
}
```

## Error Handling

### Error Types

```go
type ErrorCode string

const (
    ErrCodeClusterNotFound      ErrorCode = "ClusterNotFoundException"
    ErrCodeServiceNotFound      ErrorCode = "ServiceNotFoundException"
    ErrCodeTaskNotFound         ErrorCode = "TaskNotFoundException"
    ErrCodeInvalidParameter     ErrorCode = "InvalidParameterException"
    ErrCodeResourceInUse        ErrorCode = "ResourceInUseException"
    ErrCodeLimitExceeded        ErrorCode = "LimitExceededException"
    ErrCodeAccessDenied         ErrorCode = "AccessDeniedException"
    ErrCodeServerException      ErrorCode = "ServerException"
)

type BusinessError struct {
    Code    ErrorCode
    Message string
    Details map[string]interface{}
}
```

### Error Propagation

- Business errors are converted to appropriate HTTP status codes
- Kubernetes errors are wrapped with context
- All errors are logged with request correlation IDs
- Client receives AWS-compatible error responses

## Performance Optimizations

### 1. Connection Pooling

- DuckDB connection pool with configurable size
- Kubernetes client connection reuse
- HTTP keep-alive for API responses

### 2. Caching Strategy

- Resource metadata cached with 5-minute TTL
- List operations cached with 30-second TTL
- Cache invalidation on write operations
- Bloom filters for existence checks

### 3. Batch Operations

- Batch Kubernetes API calls where possible
- Bulk database inserts for multiple tasks
- Aggregated status updates

### 4. Async Processing

- Background task status monitoring
- Asynchronous event publishing
- Non-blocking WebSocket writes

## Monitoring and Observability

### Metrics

```go
var (
    apiRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "kecs_api_request_duration_seconds",
            Help: "API request duration in seconds",
        },
        []string{"method", "endpoint", "status"},
    )
    
    activeServices = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "kecs_active_services",
            Help: "Number of active services",
        },
        []string{"cluster"},
    )
    
    taskStateTransitions = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kecs_task_state_transitions_total",
            Help: "Total number of task state transitions",
        },
        []string{"from_state", "to_state"},
    )
)
```

### Logging

- Structured logging with zap
- Request correlation IDs
- Contextual information (cluster, service, task)
- Log levels: debug, info, warn, error

### Tracing

- OpenTelemetry integration
- Distributed tracing across components
- Trace propagation to Kubernetes operations
- Performance profiling endpoints

## Testing Strategy

### Unit Tests

- Mock interfaces for all dependencies
- Table-driven tests for API handlers
- Property-based testing for converters
- Concurrent operation testing

### Integration Tests

- Real DuckDB for storage tests
- Test Kubernetes cluster (Kind)
- End-to-end API testing
- WebSocket event verification

### Load Testing

- Simulate concurrent API requests
- Test resource limits
- Measure response times under load
- Verify system stability

## Future Enhancements

1. **Multi-Region Support**
   - Cross-region replication
   - Region-aware scheduling
   - Global resource management

2. **Advanced Scheduling**
   - Custom placement strategies
   - Resource bin packing
   - Spot instance support

3. **Enhanced Security**
   - Fine-grained RBAC
   - Audit logging
   - Encryption at rest

4. **Operational Features**
   - Automated backups
   - Disaster recovery
   - Capacity planning