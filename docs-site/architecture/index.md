# Architecture Overview

KECS has evolved to a modern architecture where the control plane runs inside a k3d cluster, providing better isolation, scalability, and integration with Kubernetes-native tools.

## Current Architecture (k3d-based)

```
┌─────────────────────────────────────────────────────────────┐
│                     User's Machine                           │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │   AWS CLI   │  │   AWS SDKs   │  │   KECS CLI      │  │
│  └──────┬──────┘  └──────┬───────┘  └────────┬─────────┘  │
└─────────┼─────────────────┼──────────────────┼─────────────┘
          │                 │                   │
          ▼                 ▼                   ▼
      Port 5373         Port 5373        Docker API
          │                 │                   │
┌─────────┼─────────────────┼──────────────────┼─────────────┐
│         │          k3d Cluster                │             │
│         ▼                 ▼                   ▼             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │            Traefik Gateway (Port 5373)               │  │
│  │         (Unified AWS API Endpoint)                   │  │
│  └────────────┬─────────────────┬───────────────────────┘  │
│               │                 │                           │
│               ▼                 ▼                           │
│  ┌──────────────────┐  ┌──────────────────┐               │
│  │   KECS Control   │  │    LocalStack    │               │
│  │      Plane       │  │                  │               │
│  │  - ECS APIs      │  │  - IAM           │               │
│  │  - ELBv2 APIs    │  │  - S3            │               │
│  │  - DuckDB Store  │  │  - SSM           │               │
│  │                  │  │  - Secrets Mgr   │               │
│  └──────────────────┘  └──────────────────┘               │
│               │                                             │
│               ▼                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Kubernetes Resources (k3s)                   │  │
│  │  - Pods (ECS Tasks)                                  │  │
│  │  - Services (Target Groups)                          │  │
│  │  - Ingresses (ALB/NLB)                               │  │
│  │  - ConfigMaps/Secrets                                │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. KECS CLI

The CLI tool manages the entire KECS lifecycle:

- **Cluster Management**: Creates and manages k3d clusters
- **Instance Control**: Start, stop, restart KECS instances
- **Configuration**: Manages instance configurations and port mappings
- **Log Access**: Streams logs from control plane components
- **Interactive TUI**: Provides terminal-based management interface

### 2. k3d Cluster

KECS runs inside a k3d cluster for better isolation:

- **k3s Kubernetes**: Lightweight Kubernetes distribution
- **Docker Runtime**: Container execution environment
- **Pre-configured Networking**: Port mappings and load balancer
- **Local Registry**: For development image management
- **Persistent Volumes**: Data storage across restarts

### 3. Traefik Gateway

Unified entry point for all AWS APIs:

- **Single Port (5373)**: All AWS services through one endpoint
- **Path-based Routing**: Routes `/v1/*` to KECS, others to LocalStack
- **Host-based Routing**: ALB/NLB DNS name routing
- **TLS Termination**: HTTPS support for secure connections
- **Load Balancing**: Distributes traffic across pods

### 4. KECS Control Plane

The heart of ECS compatibility:

#### API Server
- **ECS APIs**: Full compatibility with AWS ECS API specification
- **ELBv2 APIs**: Application and Network Load Balancer support
- **WebSocket**: Real-time updates for resource changes
- **Request Validation**: AWS-compatible error responses

#### Resource Managers
- **Cluster Manager**: ECS cluster lifecycle
- **Service Manager**: Service deployments and scaling
- **Task Manager**: Task execution and monitoring
- **Task Definition Manager**: Task definition storage and versioning

#### Storage Layer (DuckDB)
- **Persistent State**: Survives control plane restarts
- **ACID Transactions**: Consistent resource state
- **Fast Queries**: Optimized for ECS resource lookups
- **Automatic Backups**: Data protection

#### Kubernetes Integration
- **Resource Converters**: ECS to Kubernetes translation
- **Pod Management**: Task execution as Kubernetes pods
- **Service Creation**: Target groups as Kubernetes services
- **Ingress Management**: ALB/NLB as Kubernetes ingresses
- **Secret Management**: Task credentials and environment

### 5. LocalStack Integration

Provides complementary AWS services:

- **IAM**: Role management including `ecsTaskExecutionRole`
- **Secrets Manager**: Secure credential storage
- **Systems Manager (SSM)**: Parameter store
- **S3**: Object storage for artifacts
- **CloudWatch Logs**: Centralized logging
- **And more**: SQS, SNS, DynamoDB, etc.

## Key Architecture Changes

### Migration to k3d

The architecture has evolved from running as a standalone binary to running inside k3d:

| Aspect | Old Architecture | New Architecture |
|--------|-----------------|------------------|
| **Deployment** | Standalone binary | Control plane in k3d |
| **Port Management** | Direct host ports | Traefik gateway on 5373 |
| **Persistence** | Local filesystem | k3d volumes |
| **Networking** | Host networking | k3d networking |
| **Development** | Binary restart | Hot reload with `make dev` |
| **Multi-instance** | Port conflicts | Isolated k3d clusters |

### Benefits of k3d Architecture

1. **Isolation**: Each KECS instance runs in its own k3d cluster
2. **Portability**: Consistent environment across platforms
3. **Integration**: Native Kubernetes tools work out-of-the-box
4. **Development**: Hot reload without losing state
5. **Testing**: Easy CI/CD integration with containers

## Data Flow

### Service Creation Flow

```
AWS CLI → Port 5373 → Traefik → KECS API → DuckDB
                                    ↓
                              Kubernetes API
                                    ↓
                            Create Deployment
                            Create Service
                            Create Ingress
```

### Task Execution Flow

```
RunTask Request → KECS API → Task Manager
                      ↓
                Task Definition
                      ↓
                Pod Specification
                      ↓
                Kubernetes Scheduler
                      ↓
                Pod Running
                      ↓
                Status Updates → DuckDB
```

### Load Balancer Creation Flow

```
Create ALB → KECS ELBv2 API → Store in DuckDB
                    ↓
            Create Target Group
                    ↓
            K8s Service (tg-*)
                    ↓
            Create Listener
                    ↓
            K8s Ingress → Traefik
```

## Networking Architecture

### Port Mappings

```
External Port → k3d LoadBalancer → Internal Service
    5373     →    NodePort 30373 →  Traefik:80
    8081     →    NodePort 30881 →  Admin:8081
```

### DNS Resolution

```
my-alb.kecs.local → Traefik (Host Header)
                        ↓
                  Kubernetes Ingress
                        ↓
                  Target Group Service
                        ↓
                    ECS Task Pods
```

## Storage Architecture

### DuckDB Schema

```sql
-- Clusters table
CREATE TABLE clusters (
    arn TEXT PRIMARY KEY,
    name TEXT UNIQUE,
    status TEXT,
    created_at TIMESTAMP
);

-- Services table  
CREATE TABLE services (
    arn TEXT PRIMARY KEY,
    cluster_arn TEXT,
    name TEXT,
    task_definition TEXT,
    desired_count INTEGER,
    running_count INTEGER,
    status TEXT,
    created_at TIMESTAMP,
    FOREIGN KEY (cluster_arn) REFERENCES clusters(arn)
);

-- Tasks table
CREATE TABLE tasks (
    arn TEXT PRIMARY KEY,
    cluster_arn TEXT,
    service_arn TEXT,
    task_definition TEXT,
    status TEXT,
    started_at TIMESTAMP,
    FOREIGN KEY (cluster_arn) REFERENCES clusters(arn),
    FOREIGN KEY (service_arn) REFERENCES services(arn)
);

-- Load Balancers table
CREATE TABLE load_balancers (
    arn TEXT PRIMARY KEY,
    name TEXT UNIQUE,
    type TEXT,
    dns_name TEXT,
    created_at TIMESTAMP
);
```

### Data Persistence

- **Location**: `~/.kecs/instances/<name>/data/`
- **Backup**: Automatic on graceful shutdown
- **Recovery**: Restored on instance start
- **Migration**: Portable across machines

## Security Model

### Local Development

- **No Authentication**: Designed for local use only
- **Network Isolation**: k3d network boundaries
- **Resource Limits**: Container resource constraints

### Production Considerations

For production deployments (not recommended):

1. **API Security**: Add authentication layer
2. **TLS**: Enable HTTPS on all endpoints
3. **Network Policies**: Restrict pod communication
4. **RBAC**: Kubernetes role-based access
5. **Secrets**: External secret management

## Performance Characteristics

### Resource Usage

| Component | CPU | Memory | Storage |
|-----------|-----|--------|---------|
| k3d cluster | ~200m | ~512Mi | ~1GB |
| KECS control plane | ~100m | ~256Mi | ~100MB |
| LocalStack | ~100m | ~512Mi | ~500MB |
| Traefik | ~50m | ~128Mi | - |
| Per ECS task | Variable | Variable | Variable |

### Scalability Limits

- **Clusters**: 10-20 per instance
- **Services**: 100-200 per cluster
- **Tasks**: 500-1000 concurrent
- **Task Definitions**: Unlimited (DuckDB)
- **Load Balancers**: 50-100

## Development Workflow

### Hot Reload Architecture

```
Code Change → make dev → Build Image
                ↓
          Push to k3d registry
                ↓
          Update Deployment
                ↓
          Rolling Update (zero downtime)
                ↓
          Preserve DuckDB state
```

### CI/CD Integration

```yaml
# GitHub Actions example
- name: Start KECS
  run: kecs start --instance ci
  
- name: Run tests
  run: |
    export AWS_ENDPOINT_URL=http://localhost:5373
    ./run-integration-tests.sh
    
- name: Cleanup
  run: kecs stop --instance ci
```

## Next Steps

- [Control Plane Details](/architecture/control-plane)
- [Storage Layer Design](/architecture/storage)
- [Kubernetes Integration](/architecture/kubernetes)
