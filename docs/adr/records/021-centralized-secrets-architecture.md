# 21. Centralized Secrets Architecture

Date: 2025-01-10

## Status

Accepted

## Context

KECS needs to manage secrets from AWS Secrets Manager and SSM Parameter Store across multiple ECS cluster namespaces. Initially, we synchronized secrets directly to each namespace where they were needed, but this approach had several issues:

1. **Duplication**: The same secret would be synchronized multiple times across different namespaces
2. **Consistency**: Updates to secrets required multiple synchronization operations
3. **Resource Usage**: Each namespace maintained its own copy of secrets
4. **Complexity**: Synchronization logic had to track which namespaces needed which secrets

## Decision

We have decided to implement a centralized secrets architecture where:

1. All secrets from LocalStack (Secrets Manager and SSM Parameter Store) are synchronized to the `kecs-system` namespace as the single source of truth
2. User namespaces (ECS cluster namespaces) receive replicated copies of only the secrets they need
3. Secrets are replicated on-demand when pods require them
4. RBAC controls enable secure cross-namespace access

### Architecture Components

#### 1. Master Secrets in kecs-system
- All secrets from LocalStack are synchronized to `kecs-system` namespace
- Secrets Manager secrets become Kubernetes Secrets
- SSM Parameters become either Secrets or ConfigMaps based on sensitivity
- Simple naming convention without namespace prefix: `sm-<secret-name>` or `ssm-<parameter-name>`

#### 2. Secret Replication
- `SecretsReplicator` component handles replication from `kecs-system` to user namespaces
- Replicated secrets include metadata labels indicating their source
- Replication happens on-demand when pods need secrets
- Cleanup mechanism removes orphaned replicas

#### 3. RBAC Configuration
- Each user namespace gets a ServiceAccount with read permissions to `kecs-system` secrets
- ClusterRole defines read-only access to secrets and configmaps
- RoleBinding in `kecs-system` grants access to the namespace's ServiceAccount

### Implementation Details

```go
// Sync flow for Secrets Manager secrets
1. Pod created with secret annotations
2. SecretsController detects pod needs secrets
3. Secret synced from LocalStack to kecs-system
4. Secret replicated from kecs-system to pod's namespace
5. Pod can reference the local replica

// Replication metadata
Labels:
  kecs.io/managed-by: kecs
  kecs.io/replicated-from: kecs-system
  kecs.io/source: secretsmanager|ssm

Annotations:
  kecs.io/last-replicated: <timestamp>
  kecs.io/source-namespace: kecs-system
```

## Consequences

### Positive
- **Single Source of Truth**: All secrets managed centrally in `kecs-system`
- **Reduced Duplication**: Each secret stored once in `kecs-system`
- **Simplified Updates**: Update once in `kecs-system`, replicas updated automatically
- **Better Security**: Centralized access control and audit trail
- **Easier Monitoring**: All secret operations happen in one namespace

### Negative
- **Additional Complexity**: Replication layer adds complexity
- **RBAC Management**: Must maintain RBAC resources for each namespace
- **Potential Latency**: Extra step for replication could add latency
- **Cleanup Required**: Must handle orphaned replicas when source deleted

### Migration Path
1. Update sync controllers to write to `kecs-system`
2. Implement replication mechanism
3. Setup RBAC for existing namespaces
4. Test with new deployments
5. Migrate existing secrets (if any)

## References
- Kubernetes cross-namespace resource sharing patterns
- Secret replication controllers in multi-tenant environments
- RBAC best practices for Kubernetes