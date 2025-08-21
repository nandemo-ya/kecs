# 23. TUI Container Log Viewing

## Status

Proposed

## Context

The current KECS TUI allows users to view task status and detailed information, but lacks container log viewing functionality. Users must resort to one of the following methods to check logs:

1. Execute kubectl logs command directly
2. Check CloudWatch Logs (if configured) via AWS CLI or Management Console
3. Check LocalStack's CloudWatch Logs

These methods have the following challenges:
- Need to leave the TUI interface
- CloudWatch Logs lacks real-time capability (due to buffering and batching)
- Logs don't exist in CloudWatch Logs if LogGroup is not configured
- Difficult to retrieve logs after task termination

## Decision

We will implement an integrated container log viewing feature in the TUI with the following characteristics:

### 1. Hybrid Log Source Strategy

**Running Tasks:**
- Use Kubernetes API's Pod Logs API directly for real-time log retrieval
- Support streaming to display new logs in real-time
- Works regardless of CloudWatch Logs configuration

**Terminated Tasks:**
- Retrieve from logs stored in DuckDB
- Fetch logs from Kubernetes API and save to DuckDB when task terminates
- Enables long-term storage and fast searching

### 2. DuckDB Log Storage Design

```sql
CREATE TABLE IF NOT EXISTS task_logs (
    id TEXT PRIMARY KEY DEFAULT (gen_random_uuid()::TEXT),
    task_arn TEXT NOT NULL,
    container_name TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    log_line TEXT NOT NULL,
    log_level TEXT,  -- INFO, WARN, ERROR, DEBUG, etc. (when parseable)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    -- Note: DuckDB doesn't support CASCADE on foreign keys
);

-- Indexes
CREATE INDEX idx_task_logs_task_arn ON task_logs(task_arn);
CREATE INDEX idx_task_logs_timestamp ON task_logs(task_arn, timestamp);
CREATE INDEX idx_task_logs_container ON task_logs(task_arn, container_name);
```

### 3. Log Collection Architecture

```
┌─────────────┐
│   TUI       │
└──────┬──────┘
       │
       ▼
┌─────────────────────────┐
│  Log Viewer Component   │
└──────┬──────────────────┘
       │
       ├─────────────────────┐
       ▼                     ▼
┌──────────────┐      ┌──────────────┐
│ Live Logs    │      │ Stored Logs  │
│ (K8s API)    │      │ (DuckDB)     │
└──────────────┘      └──────────────┘
       │                     ▲
       │                     │
       └─────────────────────┘
         (Save on task termination)
```

### 4. TUI Implementation

**Log Viewer Component:**
```go
type LogViewer struct {
    taskArn       string
    containerName string
    logSource     LogSource  // LIVE or STORED
    followMode    bool
    filter        string
    logLevel      string
}
```

**Key Bindings:**
- `l` - Open log viewer from task detail screen
- `f` - Toggle follow mode (running tasks only)
- `/` - Log search/filter
- `Ctrl+C` - Close log viewer
- `↑/↓` - Scroll logs
- `PgUp/PgDn` - Page scroll

### 5. API Implementation

**New Endpoints:**
```go
// Get logs from running tasks (streaming support)
GET /api/tasks/{taskArn}/containers/{containerName}/logs/stream

// Get stored logs
GET /api/tasks/{taskArn}/containers/{containerName}/logs
  ?from={timestamp}
  &to={timestamp}
  &level={logLevel}
  &limit={limit}
  &offset={offset}
```

### 6. Log Lifecycle Management

**Log Collection Timing:**
1. Automatically fetch logs from Kubernetes API when task terminates
2. Save to DuckDB (batch insert)
3. Pod logs are naturally GC'd after save completion

**Log Retention Period:**
- Default: 7 days
- Configurable (environment variable: `KECS_LOG_RETENTION_DAYS`)
- Periodic cleanup job to delete old logs

## Consequences

### Positive
- Can view logs without leaving the TUI
- Improved debugging experience with real-time log display
- Log viewing available without CloudWatch Logs configuration
- Can view logs from terminated tasks
- Efficient problem investigation with log search/filtering features

### Negative
- Increased DuckDB storage usage
- Slight overhead from log collection processing
- Need to manage log retention period

## Implementation Plan

### Phase 1: Log Storage Foundation (1-2 days)
1. Add task_logs table to DuckDB
2. Implement log save/retrieve interfaces
3. Log collection feature on task termination

### Phase 2: API Implementation (1-2 days)
1. Live log streaming endpoint
2. Stored log retrieval endpoint
3. WebSocket support (optional)

### Phase 3: TUI Integration (2-3 days)
1. Log viewer component implementation
2. Key bindings and navigation
3. Filtering and search functionality
4. Follow mode (equivalent to tail -f)

### Phase 4: Operational Features (1 day)
1. Log rotation
2. Automatic deletion of old logs
3. Performance optimization

## Alternative Approaches Considered

### 1. CloudWatch Logs Only
- Pros: AWS compatibility, leverage existing implementation
- Cons: Lack of real-time capability, configuration required

### 2. Kubernetes API Only
- Pros: Simple, no additional storage needed
- Cons: Cannot retrieve logs from terminated tasks

### 3. Via Vector/Fluentd
- Pros: Advanced log processing capabilities
- Cons: Increased complexity, additional components required

## References

- [Kubernetes Pods Logs API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.28/#read-log-pod-v1-core)
- [DuckDB Text Functions](https://duckdb.org/docs/sql/functions/char)
- [Bubble Tea TUI Framework](https://github.com/charmbracelet/bubbletea)