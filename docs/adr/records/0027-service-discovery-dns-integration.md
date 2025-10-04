# ADR 0027: Service Discovery DNS Integration using CoreDNS and Kubernetes Services

## Status

Accepted

## Context

ECS Service Discovery (AWS Cloud Map) allows services to discover each other using DNS names. KECS needs to provide compatible Service Discovery functionality without relying on LocalStack's paid Service Discovery feature.

### Requirements

1. Support ECS Service Discovery API (CreatePrivateDnsNamespace, CreateService, etc.)
2. Enable DNS-based service discovery (e.g., `backend-api.demo.local`)
3. Automatic instance registration when tasks start
4. Health checking and automatic deregistration
5. Work within Kubernetes clusters using existing resources
6. Avoid dependency on LocalStack's paid features

### Constraints

- LocalStack Service Discovery requires paid plan
- LocalStack Route53 is available in free tier
- K3d/K3s clusters have CoreDNS built-in
- ECS tasks run as Kubernetes Pods in cluster-specific namespaces (e.g., `default-us-east-1`)
- Service Discovery should work across namespaces

## Decision

We will implement Service Discovery using a combination of:
1. **CoreDNS** - For DNS resolution of service discovery zones
2. **Kubernetes Services** - For mapping DNS names to Pod IPs
3. **LocalStack Route53** - For metadata storage and future extensibility
4. **KECS Service Discovery Manager** - For orchestrating the integration

### Architecture

```
ECS CreateService with ServiceRegistries
  ↓
KECS Service Discovery Manager
  ├─ Store metadata in memory/database
  │   └─ Namespace/Service/Instance mappings
  │
  ├─ Create Kubernetes Resources
  │   ├─ ExternalName Service (in SD namespace)
  │   └─ ClusterIP/NodePort Service (in ECS namespace)
  │
  ├─ Update CoreDNS Configuration
  │   └─ Add zone mapping for demo.local → Kubernetes plugin
  │
  └─ (Optional) Register in Route53
      └─ For future multi-cluster scenarios
```

### DNS Resolution Flow

```
Application: nslookup backend-api.demo.local
  ↓
CoreDNS: demo.local zone → Kubernetes plugin
  ↓
Kubernetes: Look up "backend-api" Service in "default" namespace
  ↓
ExternalName Service → backend-api-service.default-us-east-1.svc.cluster.local
  ↓
Resolve: backend-api-service in default-us-east-1 namespace
  ↓
Return: Pod IPs (10.42.x.x)
```

### Resource Structure

#### 1. Service Discovery Namespace
Default Kubernetes namespace (`default`) represents the Service Discovery DNS zone.

#### 2. ExternalName Service (Service Discovery Layer)
```yaml
apiVersion: v1
kind: Service
metadata:
  name: backend-api  # Service Discovery service name
  namespace: default  # Service Discovery namespace
  labels:
    kecs.io/service-discovery: "true"
    kecs.io/namespace: demo.local
    kecs.io/service: backend-api
spec:
  type: ExternalName
  externalName: backend-api-service.default-us-east-1.svc.cluster.local
  ports:
  - port: 8080
```

#### 3. ClusterIP/NodePort Service (ECS Service Layer)
```yaml
apiVersion: v1
kind: Service
metadata:
  name: backend-api-service  # ECS service name
  namespace: default-us-east-1  # ECS cluster namespace
spec:
  type: NodePort  # or ClusterIP
  selector:
    app: backend-api-service
  ports:
  - port: 8080
    targetPort: 8080
```

#### 4. CoreDNS Configuration
```
# ConfigMap: coredns-custom in kube-system
demo.local:53 {
    errors
    health
    ready
    kubernetes default demo.local in-addr.arpa {
        pods insecure
        fallthrough
    }
    cache 30
    loop
    reload
    loadbalance
}
```

### Implementation Details

#### CreatePrivateDnsNamespace
1. Create namespace metadata in KECS storage
2. Create Route53 hosted zone in LocalStack (for metadata)
3. Create or update CoreDNS ConfigMap with new zone
4. Reload CoreDNS

#### CreateService (Service Discovery)
1. Store service metadata (namespace, name, DNS config)
2. Create ExternalName Service in Service Discovery namespace
3. Associate with Route53 hosted zone (metadata only)

#### RegisterInstance / DeregisterInstance
1. Automatically triggered by Pod lifecycle events
2. Update service endpoints
3. (Optional) Update Route53 A records for visibility

#### Instance Health Checking
- Leverage Kubernetes liveness/readiness probes
- Only READY Pods receive traffic through Services
- Automatic instance deregistration on Pod deletion

### Benefits of This Approach

1. **No LocalStack Paid Features**: Uses only free Route53 for metadata
2. **Native Kubernetes Integration**: Leverages built-in Service/DNS mechanisms
3. **Scalable**: CoreDNS handles high query volumes efficiently
4. **Reliable**: Kubernetes Services provide automatic load balancing
5. **Health Checking**: Built-in via Kubernetes probes
6. **Cross-Namespace**: ExternalName Services enable service discovery across namespaces
7. **Future-Proof**: Route53 integration allows multi-cluster scenarios later

### Limitations

1. **Two-Step DNS Resolution**: ExternalName → ClusterIP Service (minimal overhead)
2. **Namespace Coupling**: Service Discovery namespace is fixed to `default`
3. **No SRV Records**: Initial implementation focuses on A records
4. **Route53 Not Primary**: Route53 is metadata-only, not authoritative DNS

## Consequences

### Positive

- Service Discovery works without LocalStack paid features
- Uses Kubernetes-native mechanisms for reliability
- Simple to debug (standard kubectl commands)
- Automatic health checking and load balancing
- Compatible with ECS Service Discovery API

### Negative

- Requires CoreDNS configuration management
- ExternalName Services add one level of indirection
- Route53 integration is not authoritative (just metadata)

### Neutral

- Service Discovery namespace is `default` by convention
- ECS cluster namespaces remain separate (e.g., `default-us-east-1`)

## Implementation Tasks

1. Fix ExternalName Service to point to correct ECS Service FQDN
2. Ensure Kubernetes Services are created with proper selectors
3. Implement automatic instance registration on Pod events
4. Add health checking integration
5. Document DNS resolution behavior
6. Add integration tests for service-to-service communication

## Related

- ADR 0015: ECS Service Discovery API Implementation
- [AWS ECS Service Discovery](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/service-discovery.html)
- [CoreDNS Kubernetes Plugin](https://coredns.io/plugins/kubernetes/)
