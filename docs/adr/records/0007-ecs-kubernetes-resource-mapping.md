# ADR-0007: ECS-Kubernetes Resource Mapping

Date: 2025-01-04

## Status
Proposed

## Context
KECS needs to map Amazon ECS resources to Kubernetes resources to provide ECS-compatible APIs on top of Kubernetes. Since ECS and Kubernetes have different resource models and concepts, we need to define clear mapping rules that maintain ECS API compatibility while leveraging Kubernetes capabilities effectively.

The key challenge is handling ECS clusters in a Kubernetes environment. Creating multiple Kubernetes clusters with kind for each ECS cluster would introduce significant overhead and complexity.

## Decision
We will map ECS resources to Kubernetes resources as follows:

### 1. ECS Cluster → Kubernetes Namespace
- Format: `<cluster-name>.<region>`
- Example: `my-cluster.us-east-1`
- Rationale: Namespaces provide logical isolation without the overhead of managing multiple Kubernetes clusters

### 2. ECS Service → Kubernetes Deployment + Service
- **Deployment**: Handles replica management, rolling updates, and pod lifecycle
- **Service**: Provides load balancing and service discovery
- Rationale: This combination provides equivalent functionality to ECS services

### 3. ECS Task → Kubernetes Pod
- Direct 1:1 mapping
- Represents a running instance of containers
- Rationale: Both represent the smallest deployable unit of containers

### 4. ECS Task Definition → Pod Template (in Deployment)
- Defines container specifications, resource requirements, environment variables
- Stored as part of Deployment spec
- Rationale: Pod templates serve the same purpose as task definitions

### 5. ECS Container → Kubernetes Container (in Pod)
- Direct 1:1 mapping
- Container specifications are nearly identical
- Rationale: Both platforms use the same container runtime standards

### 6. ECS Task Role → ServiceAccount + RBAC
- ServiceAccount provides pod identity
- RBAC rules define permissions
- Rationale: Kubernetes ServiceAccounts with RBAC provide similar IAM-like capabilities

### 7. ECS Service Discovery → Kubernetes Service
- Kubernetes Services provide DNS-based service discovery
- ClusterIP services for internal communication
- Rationale: Native Kubernetes service discovery aligns well with ECS service discovery

### 8. ECS Load Balancer → Kubernetes Service/Ingress
- ALB → Ingress controller
- NLB → Service with type LoadBalancer or NodePort
- Rationale: Kubernetes provides equivalent load balancing mechanisms

### 9. ECS Capacity Provider → Node Management Strategy
- EC2 capacity → Node pools/groups
- Fargate → Serverless node provisioning (if available)
- Rationale: Abstracts the compute provisioning strategy

## Consequences

### Positive
- Leverages native Kubernetes features effectively
- Avoids overhead of managing multiple Kubernetes clusters
- Provides clear mapping between ECS and Kubernetes concepts
- Enables efficient resource utilization through namespace isolation
- Maintains ECS API compatibility while using Kubernetes primitives

### Negative
- Some ECS features may not have perfect Kubernetes equivalents
- Namespace-based isolation is weaker than cluster-based isolation
- Region simulation through namespace naming is a convention, not enforcement
- Some ECS-specific behaviors may need to be emulated in the control plane

### Implementation Considerations
1. Namespace naming convention must be strictly enforced
2. Resource quotas and network policies should be used to enhance namespace isolation
3. The control plane must maintain mapping metadata to translate between ECS and Kubernetes resources
4. Some ECS API responses may need to be synthesized from multiple Kubernetes resources

## References
- [Amazon ECS Developer Guide](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/)
- [Kubernetes Concepts](https://kubernetes.io/docs/concepts/)
- ADR-0001: Product Concept
- ADR-0002: Architecture