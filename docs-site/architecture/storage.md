# Storage Layer Architecture

## Overview

The storage layer in KECS provides persistent storage for ECS resources using DuckDB as the embedded database engine. This design choice enables high performance, ACID compliance, and zero external dependencies.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Storage Layer                             │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   Storage Interface                       │  │
│  │  - Abstract interface for all storage operations         │  │
│  │  - Enables swappable storage backends                    │  │
│  └────────────────────────┬─────────────────────────────────┘  │
│                           │                                      │
│  ┌────────────────────────▼─────────────────────────────────┐  │
│  │                  DuckDB Implementation                    │  │
│  │  ┌─────────────────┐  ┌─────────────────┐               │  │
│  │  │  Connection     │  │   Transaction   │               │  │
│  │  │     Pool        │  │   Management    │               │  │
│  │  └────────┬────────┘  └────────┬────────┘               │  │
│  │           │                     │                         │  │
│  │  ┌────────▼─────────────────────▼────────┐               │  │
│  │  │          Schema Management            │               │  │
│  │  │  - Migrations                         │               │  │
│  │  │  - Version Control                    │               │  │
│  │  └───────────────────────────────────────┘               │  │
│  │                                                           │  │
│  │  ┌───────────────────────────────────────────────────┐  │  │
│  │  │              Data Access Layer                     │  │  │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │  │  │
│  │  │  │ Cluster  │  │ Service  │  │  Task    │       │  │  │
│  │  │  │  Store   │  │  Store   │  │  Store   │       │  │  │
│  │  │  └──────────┘  └──────────┘  └──────────┘       │  │  │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │  │  │
│  │  │  │Task Def  │  │  Tags    │  │ Events   │       │  │  │
│  │  │  │  Store   │  │  Store   │  │  Store   │       │  │  │
│  │  │  └──────────┘  └──────────┘  └──────────┘       │  │  │
│  │  └───────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    Cache Layer                            │  │
│  │  ┌─────────────────┐  ┌─────────────────────────────┐   │  │
│  │  │  Memory Cache   │  │  Query Result Cache        │   │  │
│  │  │  (LRU + TTL)    │  │  (Prepared Statements)     │   │  │
│  │  └─────────────────┘  └─────────────────────────────┘   │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## DuckDB Selection Rationale

### Why DuckDB?

1. **Embedded Database**: No external dependencies or separate processes
2. **OLAP Optimized**: Excellent for analytical queries and aggregations
3. **ACID Compliant**: Full transaction support with isolation
4. **High Performance**: Columnar storage with vectorized execution
5. **SQL Support**: Rich SQL dialect for complex queries
6. **Low Memory Footprint**: Efficient memory usage with spill-to-disk
7. **Concurrent Access**: MVCC for concurrent reads with serialized writes

### Trade-offs

- **Write Performance**: Optimized for reads over writes (acceptable for ECS workloads)
- **Single Writer**: Only one write transaction at a time (mitigated by connection pooling)
- **File-based**: Requires persistent disk storage

## Schema Design

### Core Tables

#### Clusters Table

```sql
CREATE TABLE clusters (
    arn VARCHAR PRIMARY KEY,
    cluster_name VARCHAR NOT NULL UNIQUE,
    status VARCHAR NOT NULL,
    registered_container_instances_count INTEGER DEFAULT 0,
    running_tasks_count INTEGER DEFAULT 0,
    pending_tasks_count INTEGER DEFAULT 0,
    active_services_count INTEGER DEFAULT 0,
    capacity_providers JSON,
    default_capacity_provider_strategy JSON,
    settings JSON,
    configuration JSON,
    attachments JSON,
    attachments_status VARCHAR,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_clusters_name ON clusters(cluster_name);
CREATE INDEX idx_clusters_status ON clusters(status);
```

#### Services Table

```sql
CREATE TABLE services (
    arn VARCHAR PRIMARY KEY,
    service_name VARCHAR NOT NULL,
    cluster_arn VARCHAR NOT NULL,
    status VARCHAR NOT NULL,
    desired_count INTEGER NOT NULL DEFAULT 0,
    running_count INTEGER NOT NULL DEFAULT 0,
    pending_count INTEGER NOT NULL DEFAULT 0,
    task_definition VARCHAR NOT NULL,
    launch_type VARCHAR,
    platform_version VARCHAR,
    platform_family VARCHAR,
    deployment_configuration JSON,
    deployments JSON,
    role_arn VARCHAR,
    events JSON,
    placement_constraints JSON,
    placement_strategy JSON,
    network_configuration JSON,
    health_check_grace_period_seconds INTEGER,
    scheduling_strategy VARCHAR DEFAULT 'REPLICA',
    deployment_controller JSON,
    tags JSON,
    enable_ecs_managed_tags BOOLEAN DEFAULT FALSE,
    propagate_tags VARCHAR,
    enable_execute_command BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR,
    FOREIGN KEY (cluster_arn) REFERENCES clusters(arn) ON DELETE CASCADE,
    UNIQUE(cluster_arn, service_name)
);

CREATE INDEX idx_services_cluster ON services(cluster_arn);
CREATE INDEX idx_services_name ON services(service_name);
CREATE INDEX idx_services_status ON services(status);
CREATE INDEX idx_services_task_def ON services(task_definition);
```

#### Tasks Table

```sql
CREATE TABLE tasks (
    arn VARCHAR PRIMARY KEY,
    cluster_arn VARCHAR NOT NULL,
    task_definition_arn VARCHAR NOT NULL,
    service_arn VARCHAR,
    container_instance_arn VARCHAR,
    containers JSON NOT NULL,
    status VARCHAR NOT NULL,
    desired_status VARCHAR NOT NULL,
    cpu VARCHAR,
    memory VARCHAR,
    launch_type VARCHAR,
    platform_version VARCHAR,
    platform_family VARCHAR,
    availability_zone VARCHAR,
    group_name VARCHAR,
    started_by VARCHAR,
    started_at TIMESTAMP,
    stopped_at TIMESTAMP,
    stopped_reason VARCHAR,
    stopping_at TIMESTAMP,
    stop_code VARCHAR,
    connectivity VARCHAR,
    connectivity_at TIMESTAMP,
    pull_started_at TIMESTAMP,
    pull_stopped_at TIMESTAMP,
    execution_stopped_at TIMESTAMP,
    health_status VARCHAR,
    inference_accelerators JSON,
    attributes JSON,
    version BIGINT NOT NULL DEFAULT 1,
    ephemeral_storage JSON,
    enable_execute_command BOOLEAN DEFAULT FALSE,
    tags JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (cluster_arn) REFERENCES clusters(arn) ON DELETE CASCADE,
    FOREIGN KEY (service_arn) REFERENCES services(arn) ON DELETE SET NULL
);

CREATE INDEX idx_tasks_cluster ON tasks(cluster_arn);
CREATE INDEX idx_tasks_service ON tasks(service_arn);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_desired_status ON tasks(desired_status);
CREATE INDEX idx_tasks_started_by ON tasks(started_by);
CREATE INDEX idx_tasks_group ON tasks(group_name);
```

#### Task Definitions Table

```sql
CREATE TABLE task_definitions (
    arn VARCHAR PRIMARY KEY,
    family VARCHAR NOT NULL,
    revision INTEGER NOT NULL,
    status VARCHAR NOT NULL DEFAULT 'ACTIVE',
    task_role_arn VARCHAR,
    execution_role_arn VARCHAR,
    network_mode VARCHAR DEFAULT 'bridge',
    container_definitions JSON NOT NULL,
    volumes JSON,
    placement_constraints JSON,
    requires_compatibilities JSON,
    cpu VARCHAR,
    memory VARCHAR,
    tags JSON,
    pid_mode VARCHAR,
    ipc_mode VARCHAR,
    proxy_configuration JSON,
    inference_accelerators JSON,
    ephemeral_storage JSON,
    runtime_platform JSON,
    registered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    registered_by VARCHAR,
    deregistered_at TIMESTAMP,
    requires_attributes JSON,
    UNIQUE(family, revision)
);

CREATE INDEX idx_task_definitions_family ON task_definitions(family);
CREATE INDEX idx_task_definitions_status ON task_definitions(status);
CREATE INDEX idx_task_definitions_revision ON task_definitions(family, revision DESC);
```

#### Tags Table

```sql
CREATE TABLE resource_tags (
    resource_arn VARCHAR NOT NULL,
    tag_key VARCHAR NOT NULL,
    tag_value VARCHAR,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (resource_arn, tag_key)
);

CREATE INDEX idx_tags_resource ON resource_tags(resource_arn);
CREATE INDEX idx_tags_key_value ON resource_tags(tag_key, tag_value);
```

#### Events Table

```sql
CREATE TABLE events (
    id VARCHAR PRIMARY KEY,
    event_type VARCHAR NOT NULL,
    resource_arn VARCHAR NOT NULL,
    resource_type VARCHAR NOT NULL,
    action VARCHAR NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    data JSON,
    correlation_id VARCHAR,
    user_identity VARCHAR
);

CREATE INDEX idx_events_resource ON events(resource_arn);
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX idx_events_correlation ON events(correlation_id);
```

## Connection Pool Implementation

### Pool Configuration

```go
type PoolConfig struct {
    DSN             string
    MaxConnections  int           // Default: 10
    MaxIdleTime     time.Duration // Default: 30 minutes
    MaxLifetime     time.Duration // Default: 1 hour
    CheckInterval   time.Duration // Default: 1 minute
}

type ConnectionPool struct {
    config      PoolConfig
    connections chan *pooledConnection
    mu          sync.Mutex
    closed      bool
}

type pooledConnection struct {
    conn        *sql.DB
    lastUsed    time.Time
    created     time.Time
    inUse       bool
}
```

### Connection Lifecycle

```go
func (p *ConnectionPool) GetConnection(ctx context.Context) (*sql.DB, error) {
    select {
    case conn := <-p.connections:
        if p.isConnectionValid(conn) {
            conn.lastUsed = time.Now()
            conn.inUse = true
            return conn.conn, nil
        }
        // Connection expired, close and create new
        conn.conn.Close()
        return p.createConnection()
        
    case <-ctx.Done():
        return nil, ctx.Err()
        
    default:
        // No available connections, create new if under limit
        if len(p.connections) < p.config.MaxConnections {
            return p.createConnection()
        }
        // Wait for available connection
        return p.waitForConnection(ctx)
    }
}

func (p *ConnectionPool) ReturnConnection(conn *sql.DB) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if p.closed {
        conn.Close()
        return
    }
    
    // Find the pooled connection
    for c := range p.connections {
        if c.conn == conn {
            c.inUse = false
            c.lastUsed = time.Now()
            break
        }
    }
}
```

## Transaction Management

### Transaction Patterns

```go
type TransactionFunc func(*sql.Tx) error

func (s *DuckDBStorage) WithTransaction(ctx context.Context, fn TransactionFunc) error {
    conn, err := s.pool.GetConnection(ctx)
    if err != nil {
        return err
    }
    defer s.pool.ReturnConnection(conn)
    
    tx, err := conn.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelSerializable,
    })
    if err != nil {
        return err
    }
    
    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)
        }
    }()
    
    if err := fn(tx); err != nil {
        tx.Rollback()
        return err
    }
    
    return tx.Commit()
}
```

### Optimistic Locking

```go
func (s *DuckDBStorage) UpdateTask(ctx context.Context, task *Task) error {
    return s.WithTransaction(ctx, func(tx *sql.Tx) error {
        // Check version for optimistic locking
        var currentVersion int64
        err := tx.QueryRowContext(ctx,
            "SELECT version FROM tasks WHERE arn = ?",
            task.Arn,
        ).Scan(&currentVersion)
        
        if err != nil {
            return err
        }
        
        if currentVersion != task.Version {
            return ErrConcurrentModification
        }
        
        // Update with version increment
        _, err = tx.ExecContext(ctx, `
            UPDATE tasks SET
                status = ?,
                desired_status = ?,
                containers = ?,
                updated_at = CURRENT_TIMESTAMP,
                version = version + 1
            WHERE arn = ? AND version = ?
        `, task.Status, task.DesiredStatus, 
           jsonMarshal(task.Containers), task.Arn, task.Version)
        
        return err
    })
}
```

## Query Optimization

### Prepared Statements

```go
type preparedStatements struct {
    getCluster        *sql.Stmt
    listClusters      *sql.Stmt
    getService        *sql.Stmt
    listServices      *sql.Stmt
    getTask           *sql.Stmt
    listTasks         *sql.Stmt
    getTaskDefinition *sql.Stmt
}

func (s *DuckDBStorage) prepareSt statements(conn *sql.DB) error {
    var err error
    
    s.prepared.getCluster, err = conn.Prepare(`
        SELECT arn, cluster_name, status, 
               registered_container_instances_count,
               running_tasks_count, pending_tasks_count,
               active_services_count, settings, configuration,
               created_at, updated_at
        FROM clusters
        WHERE cluster_name = ?
    `)
    if err != nil {
        return err
    }
    
    // Prepare other statements...
    return nil
}
```

### Query Plans and Indexes

```sql
-- Analyze query performance
EXPLAIN ANALYZE
SELECT s.*, COUNT(t.arn) as task_count
FROM services s
LEFT JOIN tasks t ON s.arn = t.service_arn
WHERE s.cluster_arn = ? AND s.status = 'ACTIVE'
GROUP BY s.arn;

-- Create covering indexes for common queries
CREATE INDEX idx_tasks_service_status 
ON tasks(service_arn, status) 
INCLUDE (arn, desired_status);

-- Partial indexes for active resources
CREATE INDEX idx_services_active 
ON services(cluster_arn, service_name) 
WHERE status = 'ACTIVE';
```

## Data Consistency

### Foreign Key Constraints

- Cascade deletes for cluster → services → tasks
- Set NULL for optional relationships (task → service)
- Prevent orphaned resources

### Data Validation

```go
func (s *DuckDBStorage) validateTaskDefinition(taskDef *TaskDefinition) error {
    // Validate required fields
    if taskDef.Family == "" {
        return ValidationError{Field: "family", Message: "Family is required"}
    }
    
    if len(taskDef.ContainerDefinitions) == 0 {
        return ValidationError{Field: "containerDefinitions", Message: "At least one container required"}
    }
    
    // Validate container definitions
    for i, container := range taskDef.ContainerDefinitions {
        if container.Name == "" {
            return ValidationError{
                Field: fmt.Sprintf("containerDefinitions[%d].name", i),
                Message: "Container name is required",
            }
        }
        if container.Image == "" {
            return ValidationError{
                Field: fmt.Sprintf("containerDefinitions[%d].image", i),
                Message: "Container image is required",
            }
        }
    }
    
    // Validate CPU/memory combinations
    if err := validateResourceCombination(taskDef.Cpu, taskDef.Memory); err != nil {
        return err
    }
    
    return nil
}
```

## Backup and Recovery

### Backup Strategy

```go
func (s *DuckDBStorage) Backup(ctx context.Context, backupPath string) error {
    conn, err := s.pool.GetConnection(ctx)
    if err != nil {
        return err
    }
    defer s.pool.ReturnConnection(conn)
    
    _, err = conn.ExecContext(ctx, "EXPORT DATABASE ?", backupPath)
    return err
}

func (s *DuckDBStorage) ScheduledBackup() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            backupPath := fmt.Sprintf("/backups/kecs_%s.db", 
                time.Now().Format("20060102_150405"))
            
            if err := s.Backup(context.Background(), backupPath); err != nil {
                log.Error("Backup failed", zap.Error(err))
            } else {
                log.Info("Backup completed", zap.String("path", backupPath))
                s.cleanOldBackups()
            }
            
        case <-s.stopCh:
            return
        }
    }
}
```

### Recovery Procedures

1. **Point-in-Time Recovery**
   - Transaction log replay
   - Consistent snapshots
   - Minimal data loss

2. **Disaster Recovery**
   - Regular backups to object storage
   - Cross-region replication
   - Automated restore testing

## Performance Tuning

### DuckDB Configuration

```sql
-- Set memory limit
PRAGMA memory_limit='4GB';

-- Enable parallel execution
PRAGMA threads=4;

-- Configure checkpoint behavior
PRAGMA checkpoint_threshold='1GB';
PRAGMA wal_autocheckpoint='1000';

-- Optimize for concurrent reads
PRAGMA enable_optimizer=true;
PRAGMA enable_profiling=false;
PRAGMA enable_progress_bar=false;
```

### Query Optimization

```go
// Batch operations for better performance
func (s *DuckDBStorage) BatchCreateTasks(ctx context.Context, tasks []*Task) error {
    return s.WithTransaction(ctx, func(tx *sql.Tx) error {
        stmt, err := tx.PrepareContext(ctx, `
            INSERT INTO tasks (
                arn, cluster_arn, task_definition_arn, 
                status, desired_status, containers
            ) VALUES (?, ?, ?, ?, ?, ?)
        `)
        if err != nil {
            return err
        }
        defer stmt.Close()
        
        for _, task := range tasks {
            _, err := stmt.ExecContext(ctx,
                task.Arn, task.ClusterArn, task.TaskDefinitionArn,
                task.Status, task.DesiredStatus, jsonMarshal(task.Containers),
            )
            if err != nil {
                return err
            }
        }
        
        return nil
    })
}
```

## Monitoring

### Storage Metrics

```go
var (
    dbConnectionsActive = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "kecs_db_connections_active",
            Help: "Number of active database connections",
        },
    )
    
    dbQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "kecs_db_query_duration_seconds",
            Help: "Database query duration",
        },
        []string{"operation", "table"},
    )
    
    dbErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kecs_db_errors_total",
            Help: "Total number of database errors",
        },
        []string{"operation", "error_type"},
    )
)
```

### Health Checks

```go
func (s *DuckDBStorage) HealthCheck(ctx context.Context) error {
    conn, err := s.pool.GetConnection(ctx)
    if err != nil {
        return err
    }
    defer s.pool.ReturnConnection(conn)
    
    // Simple query to verify connectivity
    var result int
    err = conn.QueryRowContext(ctx, "SELECT 1").Scan(&result)
    if err != nil {
        return err
    }
    
    // Check table existence
    tables := []string{"clusters", "services", "tasks", "task_definitions"}
    for _, table := range tables {
        var count int
        err = conn.QueryRowContext(ctx, 
            "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?",
            table,
        ).Scan(&count)
        if err != nil || count == 0 {
            return fmt.Errorf("table %s not found", table)
        }
    }
    
    return nil
}
```

## Future Enhancements

1. **Partitioning**
   - Time-based partitioning for events
   - Automatic partition management
   - Improved query performance

2. **Replication**
   - Read replicas for scaling
   - Multi-region support
   - Conflict resolution

3. **Advanced Features**
   - Full-text search indexes
   - Materialized views for analytics
   - Change data capture (CDC)

4. **Alternative Backends**
   - PostgreSQL adapter
   - CockroachDB for distributed deployments
   - Cloud-native storage options