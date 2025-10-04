# Service Discovery Design Document

## Overview

KECS implements AWS Cloud Map compatible service discovery, enabling ECS services to discover and communicate with each other using DNS names. This document describes the design and implementation details.

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        ECS API Layer                             │
│  ┌─────────────────┐        ┌──────────────────────────────┐   │
│  │ CreateService   │───────▶│ ServiceRegistries Integration│   │
│  └─────────────────┘        └──────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                                        │
┌─────────────────────────────────────────────────────────────────┐
│                   Service Discovery Layer                        │
│  ┌──────────────────┐    ┌─────────────────┐    ┌───────────┐  │
│  │ Cloud Map API    │───▶│ SD Manager      │───▶│ Storage   │  │
│  │ - Namespaces     │    │ - In-memory     │    │           │  │
│  │ - Services       │    │ - Lifecycle     │    └───────────┘  │
│  │ - Instances      │    │ - Health        │                    │
│  └──────────────────┘    └─────────────────┘                    │
└─────────────────────────────────────────────────────────────────┘
                                        │
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Layer                              │
│  ┌──────────────────┐    ┌─────────────────┐    ┌───────────┐  │
│  │ Headless Service │◀───│ K8s Integration │───▶│ Endpoints │  │
│  │ (sd-<service>)   │    │                 │    │           │  │
│  └──────────────────┘    └─────────────────┘    └───────────┘  │
└─────────────────────────────────────────────────────────────────┘
                                        │
┌─────────────────────────────────────────────────────────────────┐
│                      DNS Resolution                              │
│  ┌──────────────────┐    ┌─────────────────────────────────┐   │
│  │ CoreDNS          │    │ service.namespace.local        │   │
│  │ - Rewrite rules  │───▶│ → sd-service.default.svc      │   │
│  └──────────────────┘    └─────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

#### 1. Service Discovery Manager
- Central component managing namespaces, services, and instances
- Maintains in-memory storage with thread-safe operations
- Handles lifecycle management and health status tracking
- Location: `internal/servicediscovery/manager.go`

#### 2. Cloud Map API Handler
- Implements AWS Cloud Map API operations
- Handles API requests with AWS signature validation
- Converts between API formats and internal types
- Location: `internal/controlplane/api/servicediscovery_api.go`

#### 3. Kubernetes Integration
- Creates and manages headless services for DNS resolution
- Updates endpoints when instances register/deregister
- Uses label selectors for service identification
- Location: `internal/servicediscovery/kubernetes.go`

#### 4. ECS Integration
- Extended CreateService to support ServiceRegistries parameter
- Automatic instance registration when tasks start
- Automatic deregistration when tasks stop
- Location: `internal/controlplane/api/service_ecs_api.go`

## API Operations

### Namespace Operations

#### CreatePrivateDnsNamespace
```go
type CreatePrivateDnsNamespaceRequest struct {
    Name        string              `json:"Name"`
    Vpc         string              `json:"Vpc"`
    Description string              `json:"Description,omitempty"`
    Properties  *NamespaceProperties `json:"Properties,omitempty"`
}
```
- Creates a private DNS namespace (e.g., `demo.local`)
- Validates namespace name format
- Ensures unique namespace names
- Returns namespace ID and ARN

#### DeleteNamespace
- Deletes namespace if no services exist
- Validates namespace has no active services
- Cleans up associated resources

### Service Operations

#### CreateService
```go
type CreateServiceRequest struct {
    Name         string             `json:"Name"`
    NamespaceId  string             `json:"NamespaceId"`
    Description  string             `json:"Description,omitempty"`
    DnsConfig    *DnsConfig         `json:"DnsConfig"`
    HealthConfig *HealthCheckConfig `json:"HealthCheckConfig,omitempty"`
}
```
- Creates a service within a namespace
- Configures DNS records (A, SRV)
- Sets up health check configuration
- Creates corresponding Kubernetes resources

#### DeleteService
- Validates no active instances
- Removes Kubernetes resources
- Cleans up DNS configuration

### Instance Operations

#### RegisterInstance
```go
type RegisterInstanceRequest struct {
    ServiceId  string            `json:"ServiceId"`
    InstanceId string            `json:"InstanceId"`
    Attributes map[string]string `json:"Attributes"`
}
```
- Registers task as service instance
- Required attributes:
  - `AWS_INSTANCE_IPV4`: Task IP address
  - `AWS_INSTANCE_PORT`: Container port
  - `ECS_CLUSTER`: Cluster name
  - `ECS_SERVICE`: Service name
  - `ECS_TASK_ARN`: Task ARN
- Updates Kubernetes endpoints
- Sets initial health status

#### DeregisterInstance
- Removes instance from service
- Updates Kubernetes endpoints
- Cleans up health status

#### DiscoverInstances
```go
type DiscoverInstancesRequest struct {
    NamespaceName string `json:"NamespaceName"`
    ServiceName   string `json:"ServiceName"`
    MaxResults    int    `json:"MaxResults,omitempty"`
    HealthStatus  string `json:"HealthStatus,omitempty"`
}
```
- Returns healthy instances by default
- Supports filtering by health status
- Provides instance attributes and endpoints

## DNS Resolution

### DNS Name Format
- Pattern: `<service-name>.<namespace-name>`
- Example: `backend.demo.local`
- Resolves to all healthy instance IPs

### Kubernetes Service Naming
- Headless services prefixed with `sd-`
- Example: `sd-backend` for service `backend`
- No cluster IP assigned (headless)
- Endpoints managed dynamically

### CoreDNS Integration
```
demo.local:53 {
    rewrite name backend.demo.local sd-backend.default.svc.cluster.local
    kubernetes cluster.local
    forward . /etc/resolv.conf
}
```

## ECS Integration Details

### CreateService with Service Discovery
```json
{
  "serviceName": "my-service",
  "taskDefinition": "my-task:1",
  "desiredCount": 2,
  "serviceRegistries": [{
    "registryArn": "arn:aws:servicediscovery:region:account:service/srv-xxxxx",
    "containerName": "web",
    "containerPort": 80
  }]
}
```

### Task Lifecycle Integration
1. **Task Start**:
   - Extract service registry metadata from service
   - Register instance with task IP and port
   - Update Kubernetes endpoints

2. **Task Running**:
   - Health status tracked and updated
   - Instance discoverable via DNS

3. **Task Stop**:
   - Deregister instance automatically
   - Remove from Kubernetes endpoints
   - Clean up health status

## Data Model

### Namespace
```go
type Namespace struct {
    ID          string
    ARN         string
    Name        string
    Type        string // DNS_PRIVATE or DNS_PUBLIC
    Description string
    Properties  *NamespaceProperties
    CreateDate  time.Time
}
```

### Service
```go
type Service struct {
    ID           string
    ARN          string
    Name         string
    NamespaceID  string
    Description  string
    InstanceCount int
    DnsConfig    *DnsConfig
    HealthConfig *HealthCheckConfig
    CreateDate   time.Time
}
```

### Instance
```go
type Instance struct {
    ID           string
    ServiceID    string
    Attributes   map[string]string
    HealthStatus string // HEALTHY, UNHEALTHY, UNKNOWN
    CreateDate   time.Time
}
```

## Health Checking

### Health Status Values
- `HEALTHY`: Instance is healthy and receiving traffic
- `UNHEALTHY`: Instance failed health checks
- `UNKNOWN`: Initial state or health check pending

### Health Check Configuration
```go
type HealthCheckConfig struct {
    Type              string // HTTP, HTTPS, TCP
    ResourcePath      string // For HTTP/HTTPS
    FailureThreshold  int32
}
```

### Health Status Updates
- Can be updated via API
- Affects instance discovery results
- Integrated with ECS task health

## Implementation Considerations

### Thread Safety
- All manager operations use mutex protection
- Safe for concurrent access
- No race conditions in instance updates

### Performance
- In-memory storage for fast lookups
- O(1) instance registration/deregistration
- Efficient Kubernetes endpoint updates

### Scalability
- Supports hundreds of services per namespace
- Thousands of instances per service
- Minimal memory footprint per instance

### Error Handling
- Idempotent operations where possible
- Clear error messages for validation failures
- Graceful handling of Kubernetes API errors

## Security Considerations

### Access Control
- Namespace isolation at DNS level
- Service-level access policies (future)
- Instance attributes validation

### Network Security
- Private DNS namespaces only
- No external DNS exposure
- Kubernetes network policies apply

## Future Enhancements

### Planned Features
1. **Public DNS namespaces**: For external service discovery
2. **Route 53 integration**: Real DNS zone management
3. **Custom health checks**: HTTP/TCP health checking
4. **Service mesh integration**: Istio/Linkerd support
5. **Multi-region support**: Cross-region discovery

### API Extensions
1. **UpdateService**: Modify service configuration
2. **GetOperation**: Track async operations
3. **ListInstances**: Paginated instance listing
4. **Custom attributes**: Extended instance metadata

## Testing Strategy

### Unit Tests
- Manager operations with mock Kubernetes
- API handler request/response validation
- DNS configuration generation

### Integration Tests
- End-to-end service discovery flow
- ECS service creation with registries
- Instance discovery scenarios

### Scenario Tests
- Multi-service communication
- Health status transitions
- Concurrent registration/deregistration

## Monitoring and Observability

### Metrics
- Namespace/service/instance counts
- Registration/deregistration rates
- Discovery request latency
- Health check results

### Logging
- API operation logs
- Instance lifecycle events
- Kubernetes integration errors
- DNS resolution debugging

## References

- [AWS Cloud Map API Reference](https://docs.aws.amazon.com/cloud-map/latest/api/Welcome.html)
- [ECS Service Discovery](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/service-discovery.html)
- [Kubernetes Headless Services](https://kubernetes.io/docs/concepts/services-networking/service/#headless-services)
- [CoreDNS Documentation](https://coredns.io/manual/toc/)