# ADR-0022: TaskSet and Kubernetes Integration

## Status
Proposed

## Context
Amazon ECS TaskSets are a crucial component for advanced deployment strategies, particularly Blue/Green deployments and canary releases. A TaskSet represents a group of tasks running the same task definition version within a service. This allows multiple versions of an application to run simultaneously with controlled traffic distribution.

In KECS, we need to map this ECS concept to Kubernetes resources while maintaining API compatibility and providing similar functionality.

### ECS TaskSet Concepts
- **TaskSet**: A collection of tasks (containers) running the same task definition
- **Scale**: Percentage or count of desired tasks relative to service's desired count
- **Primary TaskSet**: The TaskSet receiving production traffic
- **Active TaskSet**: A TaskSet that is running and can receive traffic
- **Service Registry**: Integration with service discovery for traffic routing

### Use Cases
1. **Blue/Green Deployment**: Running two TaskSets simultaneously and switching traffic
2. **Canary Deployment**: Gradually shifting traffic from one TaskSet to another
3. **A/B Testing**: Running multiple versions with specific traffic distribution
4. **Rollback**: Quick reversion to previous TaskSet without redeployment

## Decision

### Kubernetes Resource Mapping

We will map ECS TaskSets to Kubernetes resources as follows:

#### 1. TaskSet → Deployment + Service
Each TaskSet will be represented by:
- **Kubernetes Deployment**: Manages the pods (tasks) for this TaskSet
- **Kubernetes Service** (optional): For TaskSet-specific service discovery
- **Labels**: Used to identify and group resources belonging to a TaskSet

```yaml
# Deployment for TaskSet
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {service-name}-{taskset-id}
  namespace: {cluster-name}-{region}
  labels:
    kecs.io/cluster: {cluster-name}
    kecs.io/service: {service-name}
    kecs.io/taskset: {taskset-id}
    kecs.io/taskset-external-id: {external-id}
    kecs.io/role: taskset
spec:
  replicas: {computed-desired-count}
  selector:
    matchLabels:
      kecs.io/taskset: {taskset-id}
  template:
    metadata:
      labels:
        kecs.io/cluster: {cluster-name}
        kecs.io/service: {service-name}
        kecs.io/taskset: {taskset-id}
    spec:
      containers: [...]
```

#### 2. Traffic Management

For traffic distribution between TaskSets, we'll use:

**Option A: Service with Endpoints Management** (Initial Implementation)
- Main service points to all pods across TaskSets
- Use label selectors to control which TaskSets receive traffic
- Simple but limited traffic distribution control

**Option B: Ingress/Gateway Based** (Future Enhancement)
- Use Ingress controllers or Service Mesh for advanced traffic management
- Supports percentage-based traffic splitting
- Better for production Blue/Green deployments

#### 3. Scale Management

TaskSet scale will be calculated as:
```
taskset.computedDesiredCount = service.desiredCount * (taskset.scale.value / 100)
```

For example:
- Service desired count: 10
- Blue TaskSet scale: 100% → 10 replicas
- Green TaskSet scale: 0% → 0 replicas (standby)

### State Management

#### TaskSet Lifecycle States
1. **PROVISIONING**: Creating Kubernetes resources
2. **ACTIVE**: Deployment is ready and pods are running
3. **PRIMARY**: This TaskSet is receiving production traffic
4. **DRAINING**: Removing from service discovery, terminating tasks

#### Stability Status
- **STABILIZING**: Deployment is rolling out
- **STEADY_STATE**: All replicas are ready
- **UNSTABLE**: Deployment has issues

### Service Discovery Integration

TaskSets will integrate with service discovery through:
1. **Service selectors**: Updated to include/exclude TaskSet labels
2. **DNS records**: TaskSet-specific DNS entries for testing
3. **Health checks**: Readiness probes ensure only healthy pods receive traffic

### API Implementation

The existing TaskSet API handlers will be enhanced to:

1. **CreateTaskSet**:
   - Create Kubernetes Deployment
   - Configure service discovery
   - Set initial scale

2. **UpdateTaskSet**:
   - Update Deployment replicas for scale changes
   - Modify service selectors for traffic shifts

3. **DeleteTaskSet**:
   - Gracefully drain traffic
   - Delete Deployment and associated resources

4. **DescribeTaskSets**:
   - Aggregate status from Kubernetes resources
   - Calculate running/pending/desired counts

## Consequences

### Positive
- Enables Blue/Green deployments in KECS
- Maintains ECS API compatibility
- Leverages native Kubernetes capabilities
- Supports gradual rollouts and quick rollbacks

### Negative
- Complex state synchronization between ECS model and Kubernetes
- Initial implementation limited to basic traffic distribution
- Requires careful coordination of multiple Kubernetes resources

### Neutral
- Additional Kubernetes resources per TaskSet increase cluster overhead
- Need to implement cleanup for orphaned TaskSets

## Implementation Plan

### Phase 1: Basic TaskSet Support
- Create/Delete TaskSet with Deployment creation
- Scale management
- Status reporting

### Phase 2: Traffic Management
- Service selector updates for PRIMARY TaskSet
- Basic Blue/Green switching

### Phase 3: Advanced Features
- Percentage-based traffic splitting
- Canary deployment support
- Integration with Ingress/Service Mesh

## References
- [AWS ECS TaskSets Documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/deployment-type-external.html)
- [Kubernetes Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [Kubernetes Service](https://kubernetes.io/docs/concepts/services-networking/service/)