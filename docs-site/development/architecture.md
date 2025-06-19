# Development Architecture

This document provides a detailed overview of KECS architecture for developers who want to understand the codebase and contribute to the project.

## Project Structure

```
kecs/
├── cmd/                           # Command-line interfaces
│   └── controlplane/             # Main control plane CLI
├── internal/                     # Internal packages (not importable)
│   ├── controlplane/            # Control plane implementation
│   │   ├── api/                # ECS API handlers
│   │   ├── admin/              # Admin API (health, metrics)
│   │   └── cmd/                # CLI commands
│   ├── converters/              # ECS to K8s converters
│   ├── kubernetes/              # Kubernetes client and managers
│   ├── storage/                 # Storage abstraction
│   │   ├── interface.go        # Storage interface
│   │   └── duckdb/             # DuckDB implementation
│   └── websocket/               # WebSocket server
├── pkg/                          # Public packages
│   ├── types/                   # Shared types
│   └── utils/                   # Utility functions
├── api/                          # API specifications
│   └── openapi/                 # OpenAPI specs
├── web-ui/                       # React Web UI
├── mcp-server/                   # Model Context Protocol server
├── tests/                        # Test suites
│   ├── integration/             # Integration tests
│   └── scenarios/               # Scenario tests
├── scripts/                      # Build and deployment scripts
├── deployments/                  # Deployment configurations
└── docs/                         # Documentation
```

## Core Architecture Principles

### 1. Clean Architecture

KECS follows clean architecture principles with clear separation of concerns:

```go
// Domain layer - pure business logic
type Cluster struct {
    ARN        string
    Name       string
    Status     ClusterStatus
    CreatedAt  time.Time
}

// Use case layer - application business rules
type ClusterUseCase interface {
    CreateCluster(ctx context.Context, input CreateClusterInput) (*Cluster, error)
    GetCluster(ctx context.Context, name string) (*Cluster, error)
}

// Interface adapters - controllers, presenters
type ClusterHandler struct {
    useCase ClusterUseCase
}

// Infrastructure layer - frameworks and drivers
type DuckDBClusterRepository struct {
    db *sql.DB
}
```

### 2. Dependency Injection

Dependencies are injected through interfaces:

```go
// Define interface
type Storage interface {
    CreateCluster(ctx context.Context, cluster *Cluster) error
    GetCluster(ctx context.Context, name string) (*Cluster, error)
}

// Inject dependency
type ClusterManager struct {
    storage Storage
    k8s     kubernetes.Interface
}

func NewClusterManager(storage Storage, k8s kubernetes.Interface) *ClusterManager {
    return &ClusterManager{
        storage: storage,
        k8s:     k8s,
    }
}
```

### 3. Context-Driven

All operations use context for cancellation and timeout:

```go
func (m *ClusterManager) CreateCluster(ctx context.Context, input *CreateClusterInput) (*Cluster, error) {
    // Set timeout for operation
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Check context throughout operation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Perform operation
    return m.storage.CreateCluster(ctx, cluster)
}
```

## Key Components

### API Layer

The API layer handles HTTP requests and implements the ECS API specification:

```go
// internal/controlplane/api/server.go
type Server struct {
    router         *mux.Router
    clusterHandler *ClusterHandler
    serviceHandler *ServiceHandler
    taskHandler    *TaskHandler
}

func (s *Server) setupRoutes() {
    // ECS API endpoints
    s.router.HandleFunc("/v1/CreateCluster", s.clusterHandler.CreateCluster).Methods("POST")
    s.router.HandleFunc("/v1/ListClusters", s.clusterHandler.ListClusters).Methods("POST")
    // ... more routes
}
```

### Storage Layer

The storage layer provides persistence with pluggable backends:

```go
// internal/storage/interface.go
type Storage interface {
    ClusterStore
    ServiceStore
    TaskStore
    TaskDefinitionStore
    TagStore
}

type ClusterStore interface {
    CreateCluster(ctx context.Context, cluster *Cluster) error
    GetCluster(ctx context.Context, name string) (*Cluster, error)
    UpdateCluster(ctx context.Context, cluster *Cluster) error
    DeleteCluster(ctx context.Context, name string) error
    ListClusters(ctx context.Context, filter *ClusterFilter) ([]*Cluster, error)
}
```

### Kubernetes Integration

The Kubernetes layer manages container orchestration:

```go
// internal/kubernetes/client_manager.go
type ClientManager interface {
    GetClient(clusterName string) (kubernetes.Interface, error)
    CreateNamespace(ctx context.Context, cluster *Cluster) error
    DeleteNamespace(ctx context.Context, clusterName string) error
}

// internal/converters/task_converter.go
func ConvertTaskDefinitionToPod(taskDef *TaskDefinition) *v1.Pod {
    // Convert ECS task definition to Kubernetes pod spec
}
```

## Development Patterns

### Error Handling

Use typed errors for better error handling:

```go
// Define error types
type ErrorCode string

const (
    ErrCodeClusterNotFound      ErrorCode = "ClusterNotFoundException"
    ErrCodeInvalidParameter     ErrorCode = "InvalidParameterException"
    ErrCodeResourceInUse        ErrorCode = "ResourceInUseException"
)

type APIError struct {
    Code    ErrorCode
    Message string
    Details map[string]interface{}
}

func (e *APIError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Use in handlers
func (h *ClusterHandler) GetCluster(w http.ResponseWriter, r *http.Request) {
    cluster, err := h.manager.GetCluster(ctx, name)
    if err != nil {
        var apiErr *APIError
        if errors.As(err, &apiErr) {
            writeError(w, string(apiErr.Code), apiErr.Message)
            return
        }
        writeError(w, "ServerException", "Internal server error")
        return
    }
}
```

### Testing Patterns

Use interfaces for testability:

```go
// Define mock
type MockStorage struct {
    mock.Mock
}

func (m *MockStorage) GetCluster(ctx context.Context, name string) (*Cluster, error) {
    args := m.Called(ctx, name)
    return args.Get(0).(*Cluster), args.Error(1)
}

// Use in tests
func TestClusterManager_GetCluster(t *testing.T) {
    mockStorage := new(MockStorage)
    manager := NewClusterManager(mockStorage, nil)
    
    expectedCluster := &Cluster{Name: "test"}
    mockStorage.On("GetCluster", mock.Anything, "test").Return(expectedCluster, nil)
    
    cluster, err := manager.GetCluster(context.Background(), "test")
    assert.NoError(t, err)
    assert.Equal(t, expectedCluster, cluster)
    mockStorage.AssertExpectations(t)
}
```

### Logging

Use structured logging with zap:

```go
logger := zap.NewProduction()
defer logger.Sync()

// Log with context
logger.Info("Creating cluster",
    zap.String("clusterName", input.ClusterName),
    zap.String("requestID", requestID),
    zap.Int("tagCount", len(input.Tags)),
)

// Log errors with stack trace
logger.Error("Failed to create cluster",
    zap.Error(err),
    zap.String("clusterName", input.ClusterName),
    zap.Stack("stacktrace"),
)
```

## Adding New Features

### 1. Adding a New API Endpoint

1. Define types in `internal/controlplane/api/types.go`:
```go
type StartTaskInput struct {
    Cluster        string   `json:"cluster"`
    TaskDefinition string   `json:"taskDefinition"`
    Count          int      `json:"count,omitempty"`
}

type StartTaskOutput struct {
    Tasks    []*Task   `json:"tasks"`
    Failures []*Failure `json:"failures"`
}
```

2. Implement handler in `internal/controlplane/api/tasks.go`:
```go
func (h *TaskHandler) StartTask(w http.ResponseWriter, r *http.Request) {
    var input StartTaskInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        writeError(w, "InvalidParameterException", err.Error())
        return
    }
    
    output, err := h.manager.StartTask(r.Context(), &input)
    if err != nil {
        handleError(w, err)
        return
    }
    
    writeResponse(w, output)
}
```

3. Register route in `internal/controlplane/api/server.go`:
```go
s.router.HandleFunc("/v1/StartTask", s.taskHandler.StartTask).Methods("POST")
```

### 2. Adding Storage Support

1. Update storage interface:
```go
type TaskStore interface {
    CreateTask(ctx context.Context, task *Task) error
    GetTask(ctx context.Context, arn string) (*Task, error)
    UpdateTaskStatus(ctx context.Context, arn string, status TaskStatus) error
    ListTasks(ctx context.Context, filter *TaskFilter) ([]*Task, error)
}
```

2. Implement for DuckDB:
```go
func (s *duckDBStorage) CreateTask(ctx context.Context, task *Task) error {
    query := `
        INSERT INTO tasks (arn, cluster_arn, task_definition_arn, status, created_at)
        VALUES (?, ?, ?, ?, ?)
    `
    _, err := s.db.ExecContext(ctx, query, 
        task.ARN, task.ClusterARN, task.TaskDefinitionARN, 
        task.Status, task.CreatedAt,
    )
    return err
}
```

### 3. Adding Kubernetes Resources

1. Create converter in `internal/converters/`:
```go
func ConvertTaskToPod(task *Task, taskDef *TaskDefinition) *v1.Pod {
    return &v1.Pod{
        ObjectMeta: metav1.ObjectMeta{
            Name:      task.Name,
            Namespace: task.ClusterName,
            Labels: map[string]string{
                "kecs.io/task-arn": task.ARN,
                "kecs.io/task-family": taskDef.Family,
            },
        },
        Spec: convertTaskDefinitionToPodSpec(taskDef),
    }
}
```

2. Implement Kubernetes operations:
```go
func (m *TaskManager) createPod(ctx context.Context, pod *v1.Pod) error {
    _, err := m.k8s.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
    return err
}
```

## Performance Considerations

### 1. Connection Pooling

```go
type ConnectionPool struct {
    connections chan *sql.DB
    maxSize     int
}

func (p *ConnectionPool) Get(ctx context.Context) (*sql.DB, error) {
    select {
    case conn := <-p.connections:
        return conn, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### 2. Caching

```go
type Cache struct {
    data map[string]interface{}
    mu   sync.RWMutex
    ttl  time.Duration
}

func (c *Cache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    item, exists := c.data[key]
    return item, exists
}
```

### 3. Batch Operations

```go
func (m *TaskManager) StartTasks(ctx context.Context, count int) ([]*Task, error) {
    tasks := make([]*Task, count)
    errCh := make(chan error, count)
    
    var wg sync.WaitGroup
    for i := 0; i < count; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            task, err := m.startSingleTask(ctx)
            if err != nil {
                errCh <- err
                return
            }
            tasks[idx] = task
        }(i)
    }
    
    wg.Wait()
    close(errCh)
    
    // Check for errors
    for err := range errCh {
        if err != nil {
            return nil, err
        }
    }
    
    return tasks, nil
}
```

## Debugging Tips

### 1. Enable Debug Logging

```go
if os.Getenv("KECS_DEBUG") == "true" {
    logger = logger.With(zap.AddCaller())
}
```

### 2. Request Tracing

```go
func TraceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        traceID := uuid.New().String()
        ctx := context.WithValue(r.Context(), "traceID", traceID)
        
        logger.Info("Request started",
            zap.String("traceID", traceID),
            zap.String("method", r.Method),
            zap.String("path", r.URL.Path),
        )
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 3. Profiling

```go
import _ "net/http/pprof"

// In admin server
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// Profile CPU
go tool pprof http://localhost:6060/debug/pprof/profile

// Profile memory
go tool pprof http://localhost:6060/debug/pprof/heap
```

## Next Steps

- [Testing Guide](./testing) - Writing and running tests
- [Building Guide](./building) - Build and packaging
- [Contributing](./contributing) - How to contribute