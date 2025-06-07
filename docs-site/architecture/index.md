# Architecture Overview

KECS is designed with a clean, modular architecture that provides ECS compatibility while leveraging Kubernetes as the underlying container orchestration platform.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                          │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │   AWS CLI   │  │   AWS SDKs   │  │     Web UI       │  │
│  └──────┬──────┘  └──────┬───────┘  └────────┬─────────┘  │
└─────────┼─────────────────┼──────────────────┼─────────────┘
          │                 │                   │
          ▼                 ▼                   ▼
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         ECS API Handler (Port 8080)                  │  │
│  │  - REST API Endpoints                                │  │
│  │  - WebSocket Connections                             │  │
│  │  - Request Validation                                │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│                    Control Plane                             │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────────────┐   │
│  │   Service   │  │    Task     │  │  Task Definition │   │
│  │  Manager    │  │   Manager   │  │     Manager      │   │
│  └──────┬──────┘  └──────┬──────┘  └────────┬─────────┘   │
│         │                 │                   │              │
│  ┌──────▼─────────────────▼──────────────────▼──────────┐  │
│  │              Resource Converters                      │  │
│  │  - ECS to Kubernetes Translation                     │  │
│  │  - State Reconciliation                              │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
          │                                      │
          ▼                                      ▼
┌──────────────────────┐              ┌──────────────────────┐
│   Storage Layer      │              │  Kubernetes Client   │
│  ┌────────────────┐  │              │  ┌────────────────┐  │
│  │    DuckDB      │  │              │  │   Deployment   │  │
│  │  - Clusters    │  │              │  │   Management   │  │
│  │  - Services    │  │              │  │                │  │
│  │  - Tasks       │  │              │  │   Pod/Service  │  │
│  │  - Task Defs   │  │              │  │   Creation     │  │
│  └────────────────┘  │              │  └────────────────┘  │
└──────────────────────┘              └──────────────────────┘
```

## Core Components

### 1. API Gateway Layer

The API Gateway handles all incoming requests and routes them appropriately:

- **ECS API Server** (Port 8080)
  - Implements AWS ECS API specification
  - Handles REST API requests
  - Manages WebSocket connections for real-time updates
  - Serves the embedded Web UI

- **Admin Server** (Port 8081)
  - Health checks (`/health`)
  - Metrics endpoint (`/metrics`)
  - Operational endpoints

### 2. Control Plane

The control plane implements the business logic for ECS resources:

- **Service Manager**: Handles service lifecycle (create, update, delete)
- **Task Manager**: Manages task execution and state
- **Task Definition Manager**: Stores and retrieves task definitions
- **Cluster Manager**: Manages ECS cluster resources

### 3. Resource Converters

Translates between ECS and Kubernetes concepts:

- Converts ECS Task Definitions to Kubernetes Deployments/Pods
- Maps ECS Services to Kubernetes Services
- Handles resource requirements and limits
- Manages secrets and environment variables

### 4. Storage Layer

DuckDB-based persistence layer:

- Stores ECS resource definitions
- Maintains state consistency
- Provides fast queries for resource lookups
- Handles concurrent access with ACID guarantees

### 5. Kubernetes Integration

Interfaces with the Kubernetes API:

- Creates and manages Pods, Services, ConfigMaps
- Monitors resource status
- Handles container lifecycle events
- Supports multiple Kubernetes backends (Kind, standard clusters)

## Design Principles

### 1. ECS API Compatibility
- Exact API compatibility with Amazon ECS
- Support for existing tools and SDKs
- Consistent error messages and responses

### 2. Clean Architecture
- Clear separation of concerns
- Dependency injection
- Interface-based design
- Testable components

### 3. Kubernetes Native
- Leverages Kubernetes primitives
- Respects Kubernetes patterns
- Compatible with Kubernetes ecosystem

### 4. Developer Experience
- Simple local setup
- Comprehensive Web UI
- Real-time updates via WebSocket
- Clear error messages

## Data Flow

### Creating a Service

1. Client sends `CreateService` request to API Gateway
2. API Gateway validates request and forwards to Service Manager
3. Service Manager creates service record in DuckDB
4. Resource Converter translates to Kubernetes resources
5. Kubernetes Client creates Deployment and Service
6. Status updates flow back through WebSocket

### Running a Task

1. Client sends `RunTask` request
2. Task Manager validates task definition exists
3. Resource Converter creates Pod specification
4. Kubernetes Client creates Pod
5. Task Manager monitors Pod status
6. Updates are stored in DuckDB and sent via WebSocket

## Security Considerations

- No authentication required for local development
- Production deployments should implement:
  - API authentication (JWT, API keys)
  - TLS for all communications
  - Network policies in Kubernetes
  - Role-based access control

## Scalability

- Horizontal scaling of control plane components
- DuckDB handles thousands of resources efficiently
- Kubernetes provides natural scaling for workloads
- WebSocket connections use efficient pub/sub model

## Next Steps

- [Control Plane Details](/architecture/control-plane)
- [Storage Layer Design](/architecture/storage)
- [Kubernetes Integration](/architecture/kubernetes)
- [Web UI Architecture](/architecture/web-ui)