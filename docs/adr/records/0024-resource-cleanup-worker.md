# ADR-0024: Resource Cleanup Worker

Date: 2025-08-23

## Status

Proposed

## Context

KECS creates various ECS resources during normal operation, but some resources are not automatically cleaned up when they are no longer needed. This leads to accumulation of stale resources over time, particularly:

1. **Stopped Tasks**: Tasks that have been stopped but remain in the database indefinitely
2. **Deleted Services**: Service records that persist after the service has been deleted
3. **Stale Container Instances**: Container instances from nodes that no longer exist
4. **Orphaned TaskSets**: TaskSets without associated services
5. **Task Logs**: Log entries that grow unbounded over time

This issue was reported in #489, where old tasks were found to remain in the system indefinitely. A systematic approach to resource cleanup is needed to maintain system health and prevent unbounded resource growth.

## Decision

We will implement a Resource Cleanup Worker that periodically scans and removes stale resources based on configurable retention policies. The worker will:

1. **Run periodically** (default: every 5 minutes) in a background goroutine
2. **Handle multiple resource types** with type-specific cleanup logic
3. **Use configurable retention periods** for each resource type
4. **Provide safe cleanup** that respects resource dependencies
5. **Log cleanup operations** for observability

### Architecture

```
ResourceCleanupWorker
├── Start(ctx) - Starts the background worker
├── Stop() - Stops the worker gracefully
├── cleanupResources() - Main cleanup orchestrator
│   ├── cleanupStoppedTasks()
│   ├── cleanupDeletedServices()
│   ├── cleanupStaleContainerInstances()
│   ├── cleanupOrphanedTaskSets()
│   └── cleanupOldTaskLogs()
└── Configuration
    ├── Enabled (bool)
    ├── Interval (duration)
    └── RetentionPeriods (map[ResourceType]duration)
```

### Default Retention Periods

- **Stopped Tasks**: 1 hour (configurable via `KECS_CLEANUP_TASK_RETENTION`)
- **Deleted Services**: 24 hours (configurable via `KECS_CLEANUP_SERVICE_RETENTION`)
- **Stale Container Instances**: 1 hour (configurable via `KECS_CLEANUP_CONTAINER_INSTANCE_RETENTION`)
- **Orphaned TaskSets**: 24 hours (configurable via `KECS_CLEANUP_TASKSET_RETENTION`)
- **Task Logs**: 7 days (configurable via `KECS_CLEANUP_LOG_RETENTION`)

### Storage Interface Extensions

New methods will be added to the storage interfaces:

```go
// TaskStore interface
DeleteOlderThan(ctx context.Context, before time.Time, status string) (int, error)

// ServiceStore interface
DeleteMarkedForDeletion(ctx context.Context, before time.Time) (int, error)

// ContainerInstanceStore interface
DeleteStale(ctx context.Context, before time.Time) (int, error)

// TaskSetStore interface
DeleteOrphaned(ctx context.Context) (int, error)

// TaskLogStore interface (new)
DeleteOlderThan(ctx context.Context, before time.Time) (int, error)
```

## Consequences

### Positive

1. **Automatic cleanup** prevents unbounded resource growth
2. **Configurable retention** allows different policies for different environments
3. **Improved performance** by reducing database size
4. **Better observability** through cleanup logging
5. **Extensible design** allows adding new resource types easily

### Negative

1. **Additional background processing** adds some CPU/memory overhead
2. **Risk of premature deletion** if retention periods are too short
3. **Complexity** in handling resource dependencies correctly
4. **Testing complexity** for time-based cleanup logic

### Mitigation Strategies

1. **Conservative defaults**: Use longer retention periods by default
2. **Dry-run mode**: Add option to log what would be deleted without actually deleting
3. **Metrics**: Export cleanup metrics for monitoring
4. **Feature flag**: Allow disabling cleanup worker entirely if needed

## Implementation Plan

1. Create `resource_cleanup_worker.go` with worker structure
2. Add cleanup methods to storage interfaces
3. Implement DuckDB store methods for each cleanup operation
4. Add configuration loading from environment variables
5. Integrate worker into control plane startup
6. Add tests for cleanup logic
7. Document configuration options

## References

- Issue #489: Old tasks remain in the system
- Similar pattern: `test_mode_task_worker.go` for worker implementation reference
- AWS ECS cleanup behavior documentation