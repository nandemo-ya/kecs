# Architecture

**Date:** 2025-05-14

## Status

Proposed

## Context

KECS (Kubernetes-based ECS Compatible Service) is a standalone service that provides Amazon ECS compatible APIs running on Kubernetes. This ADR describes the architectural design of KECS. KECS must meet the following requirements:

- Implemented as a standalone Go application
- Containerizable and runnable on both Docker and Kubernetes
- Provides Amazon ECS compatible API interfaces
- Contains an internal data store for persisting task definitions and service definitions
- Capable of starting and stopping containers based on definitions

## Decision

KECS will adopt the following architecture:

### Overall Architecture

```mermaid
graph TD
    Client[AWS CLI/SDK Client] -->|ECS API Calls| ControlPlane
    
    subgraph KECS
        ControlPlane[Control Plane] -->|Store/Retrieve| DuckDB[(DuckDB)]
        ControlPlane -->|Manage Containers| ContainerRuntime[Container Runtime]
    end
    
    ContainerRuntime -->|Run| UserContainers[User Containers]
    
    subgraph Deployment Options
        Docker[Docker]
        Kubernetes[Kubernetes]
    end
    
    KECS -.->|Can be deployed on| Docker
    KECS -.->|Can be deployed on| Kubernetes
```

### Component Structure

```mermaid
graph TD
    subgraph "Control Plane"
        API[API Server] --> TaskManager[Task Manager]
        API --> ServiceManager[Service Manager]
        API --> ClusterManager[Cluster Manager]
        
        TaskManager --> ContainerAdapter[Container Adapter]
        ServiceManager --> TaskManager
        
        TaskManager --> Persistence[Persistence Layer]
        ServiceManager --> Persistence
        ClusterManager --> Persistence
        
        Persistence --> DuckDB[(DuckDB)]
    end
    
    ContainerAdapter -->|Docker API| DockerRuntime[Docker Runtime]
    ContainerAdapter -->|Kubernetes API| K8sRuntime[Kubernetes Runtime]
    
    DockerRuntime --> Containers[User Containers]
    K8sRuntime --> Pods[Kubernetes Pods]
```

### Sequence Diagram (Task Execution Example)

```mermaid
sequenceDiagram
    participant Client as AWS CLI/SDK Client
    participant CP as Control Plane
    participant DB as DuckDB
    participant Runtime as Container Runtime
    
    Client->>CP: RegisterTaskDefinition
    CP->>DB: Store Task Definition
    DB-->>CP: Confirmation
    CP-->>Client: Task Definition ARN
    
    Client->>CP: RunTask
    CP->>DB: Retrieve Task Definition
    DB-->>CP: Task Definition
    CP->>Runtime: Create Container(s)
    Runtime-->>CP: Container ID(s)
    CP->>DB: Store Task State
    DB-->>CP: Confirmation
    CP-->>Client: Task ARN
    
    Client->>CP: DescribeTasks
    CP->>DB: Retrieve Task State
    DB-->>CP: Task State
    CP-->>Client: Task Details
    
    Client->>CP: StopTask
    CP->>Runtime: Stop Container(s)
    Runtime-->>CP: Confirmation
    CP->>DB: Update Task State
    DB-->>CP: Confirmation
    CP-->>Client: Success
```

### Sequence Diagram (Service Management Example)

```mermaid
sequenceDiagram
    participant Client as AWS CLI/SDK Client
    participant CP as Control Plane
    participant DB as DuckDB
    participant Runtime as Container Runtime
    
    Client->>CP: CreateService
    CP->>DB: Store Service Definition
    DB-->>CP: Confirmation
    CP->>CP: Calculate Desired State
    CP->>Runtime: Create Container(s)
    Runtime-->>CP: Container ID(s)
    CP->>DB: Store Service State
    DB-->>CP: Confirmation
    CP-->>Client: Service ARN
    
    Note over CP: Service Controller Loop
    loop Reconciliation
        CP->>DB: Get Service Definition
        DB-->>CP: Service Definition
        CP->>DB: Get Current Tasks
        DB-->>CP: Current Tasks
        CP->>CP: Compare Desired vs Actual
        alt Need to scale up
            CP->>Runtime: Create Container(s)
            Runtime-->>CP: Container ID(s)
            CP->>DB: Update Service State
        else Need to scale down
            CP->>Runtime: Stop Container(s)
            Runtime-->>CP: Confirmation
            CP->>DB: Update Service State
        end
    end
```

## Consequences

### Benefits

- **AWS Compatibility**: KECS can be operated using AWS CLI and SDKs
- **Portability**: Can run on both Docker and Kubernetes
- **Standalone**: Minimal external dependencies, distributable as a single binary
- **Persistence**: Data can be persisted without external databases using DuckDB
- **Extensibility**: Support for different container runtimes through container adapters

### Challenges

- **Feature Limitations**: Not all ECS features will be supported
- **Performance**: Embedded DuckDB may become a bottleneck for large-scale workloads
- **Security**: Requires access permissions to container runtime

## Alternatives Considered

### Using External Databases

We considered using external databases (PostgreSQL, MySQL, etc.) instead of DuckDB, but chose embedded DuckDB for the following reasons:

- Simplified setup (no additional infrastructure required)
- Ability to distribute as a single binary
- Sufficient performance for small to medium workloads

### Serverless Architecture

We also considered architectures using serverless technologies like AWS Lambda, but rejected them for the following reasons:

- Dependency on AWS
- Complexity in local development environments
- Limited direct control over container management

## References

- [Amazon ECS API Reference](https://docs.aws.amazon.com/AmazonECS/latest/APIReference/Welcome.html)
- [DuckDB Documentation](https://duckdb.org/docs/)
- [Kubernetes API](https://kubernetes.io/docs/reference/kubernetes-api/)
