# 20. Kubernetes to ECS State Synchronization

Date: 2025-07-24

## Status

Proposed

## Context

In the new KECS architecture, ECS resources (services and tasks) are backed by Kubernetes resources (deployments and pods). However, the current implementation does not synchronize the state between Kubernetes and the ECS API responses. This leads to:

1. `aws ecs list-tasks` returning empty results even when pods are running
2. Service status fields (runningCount, pendingCount) not reflecting actual Kubernetes state
3. Task lifecycle events not being captured in ECS API responses
4. Missing health check status and other runtime information

## Decision

We will implement a comprehensive state synchronization system that monitors Kubernetes resources and updates the corresponding ECS resources in storage. This system will use the Kubernetes informer framework for efficient resource watching and implement a controller pattern for state reconciliation.

### Monitored Kubernetes Resources

1. **Deployments** - Primary source for service state
   - `spec.replicas` → `desiredCount`
   - `status.readyReplicas` → `runningCount`
   - `status.replicas - status.readyReplicas` → `pendingCount`
   - `status.conditions` → service `status`

2. **ReplicaSets** - For deployment history and detailed replica tracking
   - Track deployment revisions
   - More granular replica state information

3. **Pods** - For individual task state
   - `status.phase` → task `lastStatus`
   - `status.containerStatuses` → container states
   - `status.podIP` → network interfaces
   - `metadata.creationTimestamp` → `startedAt`
   - Pod deletion timestamp → `stoppedAt`

4. **Events** - For service and task history
   - Convert to ECS events format
   - Track errors and warnings

### State Mappings

#### Service Status Mapping
```
Deployment State → ECS Service Status
- No deployment found → INACTIVE
- Replicas = 0 → DRAINING
- ReadyReplicas = 0 && Replicas > 0 → PROVISIONING
- ReadyReplicas < Replicas → UPDATING
- ReadyReplicas = Replicas → ACTIVE
- Deployment has failure condition → FAILED
```

#### Task Status Mapping
```
Pod Phase → ECS Task Status
- Pending + no containers → PROVISIONING
- Pending + containers creating → PENDING
- Running + containers not ready → ACTIVATING
- Running + all containers ready → RUNNING
- Succeeded → STOPPED (exitCode from container)
- Failed → STOPPED (with failure reason)
- Unknown → STOPPED (connection lost)
```

## Implementation Plan

### Phase 1: Core Synchronization Framework (Week 1-2)

1. **Create base controller structure**
```go
// pkg/controllers/sync/controller.go
type SyncController struct {
    kubeClient        kubernetes.Interface
    storage           storage.Storage
    deploymentLister  appslistersv1.DeploymentLister
    replicaSetLister  appslistersv1.ReplicaSetLister
    podLister         corelistersv1.PodLister
    eventLister       corelistersv1.EventLister
    workqueue         workqueue.RateLimitingInterface
}
```

2. **Implement informer setup**
```go
// pkg/controllers/sync/informers.go
func (c *SyncController) SetupInformers(stopCh <-chan struct{}) error {
    // Create shared informer factory
    factory := informers.NewSharedInformerFactory(c.kubeClient, time.Minute)
    
    // Setup deployment informer
    deploymentInformer := factory.Apps().V1().Deployments()
    deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc:    c.handleDeploymentAdd,
        UpdateFunc: c.handleDeploymentUpdate,
        DeleteFunc: c.handleDeploymentDelete,
    })
    
    // Similar setup for pods, replicasets, events
    // ...
}
```

3. **Create state mapper utilities**
```go
// pkg/controllers/sync/mappers/service_mapper.go
type ServiceStateMapper struct{}

func (m *ServiceStateMapper) MapDeploymentToServiceStatus(deployment *appsv1.Deployment) string {
    // Implementation of state mapping logic
}

func (m *ServiceStateMapper) MapDeploymentToServiceCounts(deployment *appsv1.Deployment) (desired, running, pending int) {
    // Extract counts from deployment status
}
```

### Phase 2: Service Synchronization (Week 2-3)

1. **Implement service sync logic**
```go
// pkg/controllers/sync/service_sync.go
func (c *SyncController) syncService(key string) error {
    // Parse namespace/name from key
    namespace, name, _ := cache.SplitMetaNamespaceKey(key)
    
    // Get deployment
    deployment, err := c.deploymentLister.Deployments(namespace).Get(name)
    
    // Map to ECS service name (remove ecs-service- prefix)
    serviceName := strings.TrimPrefix(name, "ecs-service-")
    
    // Get ECS cluster from namespace
    clusterName, region := parseNamespace(namespace)
    
    // Update service in storage
    return c.updateServiceState(clusterName, serviceName, deployment)
}
```

2. **Add deployment event handlers**
```go
func (c *SyncController) handleDeploymentUpdate(old, new interface{}) {
    newDep := new.(*appsv1.Deployment)
    oldDep := old.(*appsv1.Deployment)
    
    // Only sync if status changed
    if !reflect.DeepEqual(oldDep.Status, newDep.Status) {
        c.enqueueDeployment(newDep)
    }
}
```

3. **Implement batch updates**
```go
// pkg/controllers/sync/batch_updater.go
type BatchUpdater struct {
    updates   map[string]*storage.Service
    mu        sync.Mutex
    ticker    *time.Ticker
    storage   storage.Storage
}

func (b *BatchUpdater) AddUpdate(service *storage.Service) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.updates[service.ARN] = service
}
```

### Phase 3: Task Synchronization (Week 3-4)

1. **Implement pod to task mapping**
```go
// pkg/controllers/sync/task_sync.go
func (c *SyncController) syncTask(key string) error {
    namespace, name, _ := cache.SplitMetaNamespaceKey(key)
    
    // Get pod
    pod, err := c.podLister.Pods(namespace).Get(name)
    
    // Check if this is an ECS-managed pod
    if !isECSManagedPod(pod) {
        return nil
    }
    
    // Create or update task
    task := c.mapPodToTask(pod)
    return c.storage.TaskStore().CreateOrUpdate(context.TODO(), task)
}
```

2. **Map pod labels to ECS metadata**
```go
func (c *SyncController) mapPodToTask(pod *corev1.Pod) *storage.Task {
    return &storage.Task{
        ARN:               generateTaskARN(pod),
        ClusterARN:        getClusterARNFromNamespace(pod.Namespace),
        TaskDefinitionARN: pod.Labels["ecs.amazonaws.com/task-definition-arn"],
        DesiredStatus:     mapPodPhaseToDesiredStatus(pod.Status.Phase),
        LastStatus:        mapPodPhaseToLastStatus(pod.Status.Phase),
        LaunchType:        "FARGATE",
        StartedAt:         getPodStartTime(pod),
        StoppedAt:         getPodStopTime(pod),
        StoppedReason:     getPodStopReason(pod),
        Containers:        mapPodContainers(pod),
    }
}
```

3. **Container state synchronization**
```go
func mapPodContainers(pod *corev1.Pod) []storage.TaskContainer {
    var containers []storage.TaskContainer
    
    for i, container := range pod.Spec.Containers {
        status := getContainerStatus(pod, container.Name)
        
        containers = append(containers, storage.TaskContainer{
            Name:        container.Name,
            Image:       container.Image,
            LastStatus:  mapContainerState(status),
            ExitCode:    getExitCode(status),
            Reason:      getContainerReason(status),
            NetworkInterfaces: []storage.NetworkInterface{{
                PrivateIPv4Address: pod.Status.PodIP,
            }},
        })
    }
    
    return containers
}
```

### Phase 4: Event and History Tracking (Week 4-5)

1. **Event collection and transformation**
```go
// pkg/controllers/sync/event_sync.go
func (c *SyncController) syncEvents() {
    // Get events for ECS-managed resources
    selector := labels.Set{"kecs.dev/managed-by": "kecs"}.AsSelector()
    events, _ := c.eventLister.List(selector)
    
    // Group by service/task
    eventMap := groupEventsByResource(events)
    
    // Update storage
    for resourceARN, events := range eventMap {
        c.updateResourceEvents(resourceARN, events)
    }
}
```

2. **Health check monitoring**
```go
func (c *SyncController) extractHealthStatus(pod *corev1.Pod) string {
    for _, container := range pod.Status.ContainerStatuses {
        if container.Ready {
            continue
        }
        
        // Check if health check failed
        if isHealthCheckFailure(container) {
            return "UNHEALTHY"
        }
    }
    
    return "HEALTHY"
}
```

### Phase 5: Optimization and Monitoring (Week 5-6)

1. **Add metrics and observability**
```go
var (
    syncDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "kecs_sync_duration_seconds",
            Help: "Time taken to sync resource state",
        },
        []string{"resource_type"},
    )
    
    syncErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "kecs_sync_errors_total",
            Help: "Total number of sync errors",
        },
        []string{"resource_type", "error_type"},
    )
)
```

2. **Implement rate limiting and backoff**
```go
func (c *SyncController) processNextWorkItem() bool {
    key, quit := c.workqueue.Get()
    if quit {
        return false
    }
    defer c.workqueue.Done(key)
    
    err := c.syncByKey(key.(string))
    if err == nil {
        c.workqueue.Forget(key)
        return true
    }
    
    // Requeue with backoff
    if c.workqueue.NumRequeues(key) < 5 {
        c.workqueue.AddRateLimited(key)
        return true
    }
    
    c.workqueue.Forget(key)
    return true
}
```

## Configuration

Add configuration options to control sync behavior:

```yaml
sync:
  enabled: true
  workers: 4
  resyncPeriod: 5m
  batchSize: 100
  batchDelay: 2s
  rateLimit:
    qps: 10
    burst: 20
```

## Testing Strategy

1. **Unit tests** for state mappers and converters
2. **Integration tests** using fake Kubernetes client
3. **E2E tests** with real Kubernetes cluster
4. **Performance tests** with large numbers of resources
5. **Chaos tests** to verify resilience

## Consequences

### Positive
- ECS API responses accurately reflect Kubernetes state
- Real-time updates as Kubernetes resources change
- Efficient resource utilization through informer caching
- Comprehensive state tracking including health and events

### Negative
- Additional complexity in the control plane
- Increased memory usage for informer caches
- Potential for sync lag during high update rates
- Need to handle edge cases and race conditions

### Mitigation
- Use rate limiting to prevent overwhelming the storage
- Implement circuit breakers for storage failures
- Add comprehensive metrics for monitoring sync health
- Document sync behavior and limitations

## References
- [Kubernetes Informer Pattern](https://pkg.go.dev/k8s.io/client-go/informers)
- [ECS Task Definition](https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_Task.html)
- [Controller Best Practices](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/controllers.md)