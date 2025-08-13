# ECS API Response Fields Requiring Kubernetes Synchronization

This document analyzes the ECS API response structures and identifies which fields need to be synchronized with Kubernetes resources.

## Service Response Fields

### Fields Requiring Real-time Kubernetes Sync

| Field | ECS Type | Kubernetes Source | Update Trigger |
|-------|----------|-------------------|----------------|
| `runningCount` | int32 | Deployment.Status.ReadyReplicas | Pod Ready status changes |
| `pendingCount` | int32 | Deployment.Status.Replicas - ReadyReplicas | Pod phase changes |
| `desiredCount` | int32 | Deployment.Spec.Replicas | User updates (already synced) |
| `status` | string | Derived from Deployment conditions | Deployment status changes |
| `deployments[].runningCount` | int32 | ReplicaSet.Status.ReadyReplicas | Pod Ready status changes |
| `deployments[].pendingCount` | int32 | ReplicaSet.Status.Replicas - ReadyReplicas | Pod phase changes |
| `deployments[].failedTasks` | int32 | Count of failed pods in ReplicaSet | Pod Failed status |
| `deployments[].rolloutState` | string | Deployment rollout status | Deployment progress |
| `events[]` | ServiceEvent | Kubernetes Events for Deployment/Pods | Event creation |

### Static Fields (Set at Creation/Update)

- `serviceArn`, `serviceName`, `clusterArn` - Identifiers
- `taskDefinition` - Updated via UpdateService API
- `launchType`, `platformVersion` - Configuration
- `networkConfiguration`, `loadBalancers` - Networking config
- `createdAt`, `createdBy` - Metadata

## Task Response Fields

### Fields Requiring Real-time Kubernetes Sync

| Field | ECS Type | Kubernetes Source | Update Trigger |
|-------|----------|-------------------|----------------|
| `lastStatus` | string | Pod.Status.Phase mapping | Pod phase changes |
| `containers[].lastStatus` | string | Container.State | Container state changes |
| `containers[].exitCode` | int32 | Container.State.Terminated.ExitCode | Container termination |
| `containers[].reason` | string | Container.State.Waiting/Terminated.Reason | Container state changes |
| `containers[].networkInterfaces[]` | NetworkInterface | Pod.Status.PodIP | Pod IP assignment |
| `containers[].runtimeId` | string | Container.Status.ContainerID | Container creation |
| `startedAt` | time | Pod.Status.StartTime | Pod start |
| `stoppedAt` | time | Container.State.Terminated.FinishedAt | All containers terminated |
| `stoppingAt` | time | Pod.DeletionTimestamp | Pod deletion initiated |
| `pullStartedAt` | time | First container pulling event | Event: Pulling image |
| `pullStoppedAt` | time | Last container pulled event | Event: Pulled image |
| `connectivity` | string | Pod network readiness | Pod conditions |
| `connectivityAt` | time | When connectivity established | Pod Ready condition |
| `healthStatus` | string | Container health check results | Container probe results |
| `stoppedReason` | string | Pod/Container termination reason | Pod/Container state |
| `stopCode` | string | Mapped from exit codes | Container exit code |
| `executionStoppedAt` | time | Last container stopped | Container termination |
| `attachments[].status` | string | ENI attachment status (for awsvpc) | Pod network ready |

### Container Status Mapping

```
ECS Status -> Kubernetes State
PENDING -> Container.State.Waiting
RUNNING -> Container.State.Running
STOPPED -> Container.State.Terminated
```

### Task Status Mapping

```
ECS LastStatus -> Kubernetes Pod Phase
PROVISIONING -> Pod.Status.Phase = Pending && no containers started
PENDING -> Pod.Status.Phase = Pending && containers creating
ACTIVATING -> Pod.Status.Phase = Pending && some containers ready
RUNNING -> Pod.Status.Phase = Running && all containers ready
DEACTIVATING -> Pod.Status.Phase = Running && DeletionTimestamp set
STOPPING -> Pod being terminated
STOPPED -> Pod.Status.Phase = Succeeded/Failed
```

## Kubernetes Resources to Monitor

### For Service Updates

1. **Deployments**
   - Watch for: spec changes, status updates
   - Updates: desiredCount, status, deployments array

2. **ReplicaSets** 
   - Watch for: status changes
   - Updates: deployment running/pending counts

3. **Pods** (filtered by service selector)
   - Watch for: phase changes, ready status
   - Updates: running/pending counts, task status

4. **Events** (filtered by involved object)
   - Watch for: deployment/pod events
   - Updates: service events array

### For Task Updates

1. **Pods** (by task ID label)
   - Watch for: all status changes
   - Updates: task status, container status, timestamps

2. **Events** (by pod)
   - Watch for: image pull events, failures
   - Updates: pull timestamps, failure reasons

## Implementation Recommendations

### Watch Strategy

1. **Service Controller**
   - Watch Deployments in all namespaces with label selector
   - Watch ReplicaSets owned by tracked Deployments
   - Aggregate Pod counts from ReplicaSet status (more efficient than watching all pods)
   - Buffer updates to avoid excessive storage writes

2. **Task Controller**
   - Watch individual Pods by name (already implemented in TaskManager)
   - Track container state transitions
   - Update task status based on pod phase and container states

3. **Event Aggregation**
   - Watch Events with field selectors for tracked resources
   - Convert Kubernetes events to ECS ServiceEvents
   - Implement event deduplication and rate limiting

### Performance Considerations

1. Use label selectors to limit watch scope
2. Implement informers with shared caches
3. Batch storage updates
4. Use field selectors where possible
5. Implement exponential backoff for retries

### Critical Sync Points

1. **Service Creation**: Set initial status to PENDING, then ACTIVE when deployment ready
2. **Service Update**: Track deployment rollout progress
3. **Task Launch**: Update through PROVISIONING -> PENDING -> RUNNING states
4. **Task Stop**: Set STOPPING status immediately, STOPPED when pod deleted
5. **Health Checks**: Update container and task health status from probes

## Fields That Don't Need Sync

These fields are managed entirely by KECS and don't need Kubernetes synchronization:

- Service: `capacityProviderStrategy`, `placementConstraints`, `placementStrategy`, `tags`
- Task: `capacityProviderName`, `group`, `startedBy`, `tags`, `version`
- Administrative: `enableECSManagedTags`, `enableExecuteCommand`, `propagateTags`