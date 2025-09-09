# ADR-0025: Velero-based State Persistence for KECS Instances

## Status

Proposed

## Context

Currently, KECS faces challenges with state persistence when stopping and starting instances:

1. **DuckDB restoration not working**: The existing DuckDB-based state restoration mechanism is not functioning properly
2. **Lost ECS resources**: When an instance is stopped and restarted, ECS clusters, services, and tasks are not properly restored
3. **Complex restoration logic**: Implementing comprehensive state restoration from scratch is error-prone and may miss edge cases
4. **Kubernetes state management**: Need a Kubernetes-native solution for backing up and restoring cluster state

## Decision

We will implement state persistence using Velero, a mature Kubernetes backup and restore solution, with the following architecture:

1. **Velero for backup/restore**: Use Velero to backup and restore Kubernetes resources
2. **MinIO as storage backend**: Deploy MinIO within each k3d cluster as S3-compatible storage
3. **Host filesystem persistence**: Mount MinIO data to host filesystem for persistence across cluster restarts
4. **Hybrid approach**: Combine in-cluster MinIO with host-based data persistence

### Storage Architecture

```
~/.kecs/instances/{instance-name}/
├── minio-data/              # MinIO data (persisted on host)
│   └── velero/              # Velero backups
├── velero-config/           # Velero configuration
│   ├── credentials          # MinIO credentials
│   └── backup-location.yaml # Backup location config
└── config.json              # Instance configuration
```

## Consequences

### Positive

1. **Proven solution**: Velero is a mature, production-ready Kubernetes backup solution
2. **Comprehensive backup**: Captures all Kubernetes resources, including ConfigMaps, Secrets, and PersistentVolumes
3. **Selective restore**: Can restore specific namespaces or resource types
4. **Future flexibility**: Easy to migrate to cloud storage (S3, GCS, Azure Blob) if needed
5. **Kubernetes-native**: Works directly with Kubernetes API, ensuring consistency
6. **Incremental backups**: Velero supports incremental backups for efficiency

### Negative

1. **Additional complexity**: Introduces Velero and MinIO as new dependencies
2. **Storage overhead**: Backups consume additional disk space
3. **Startup time**: Initial setup of Velero and MinIO adds to instance startup time
4. **Resource consumption**: MinIO and Velero consume CPU and memory within the cluster

## Implementation Plan

### Phase 1: Infrastructure (Week 1)
- Create `controlplane/internal/velero` package
- Implement MinIO deployment manager
- Implement Velero installation manager
- Create Kubernetes manifests for MinIO

### Phase 2: Backup/Restore (Week 2)
- Integrate Velero backup into instance Stop operation
- Integrate Velero restore into instance Start operation
- Implement backup validation and error handling
- Add backup lifecycle management (retention, cleanup)

### Phase 3: Testing & Migration (Week 3)
- Create comprehensive test suite
- Add feature flag for gradual rollout
- Document migration path from DuckDB
- Performance optimization

## Technical Details

### MinIO Deployment

MinIO will be deployed as a single-replica StatefulSet within each k3d cluster:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: minio
  namespace: velero
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: minio
        image: minio/minio:latest
        args:
        - server
        - /data
        env:
        - name: MINIO_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: minio-credentials
              key: access-key
        - name: MINIO_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: minio-credentials
              key: secret-key
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        hostPath:
          path: /var/lib/kecs/instances/{instance-name}/minio-data
          type: DirectoryOrCreate
```

### Velero Configuration

Velero will be configured to use MinIO as its backup storage location:

```yaml
apiVersion: velero.io/v1
kind: BackupStorageLocation
metadata:
  name: default
  namespace: velero
spec:
  provider: aws
  objectStorage:
    bucket: velero
  config:
    region: minio
    s3ForcePathStyle: "true"
    s3Url: http://minio.velero.svc:9000
```

### Backup Strategy

1. **On Stop**: Create a full backup of all namespaces (excluding system namespaces)
2. **Backup naming**: `stop-backup-{instance-name}-{timestamp}`
3. **Retention**: Keep last 5 backups per instance
4. **Validation**: Verify backup completion before stopping cluster

### Restore Strategy

1. **On Start**: Check for existing backups
2. **Latest backup**: Restore from the most recent successful backup
3. **Failure handling**: Log warning but continue startup if restore fails
4. **Selective restore**: Option to restore specific namespaces or resources

## Alternatives Considered

### 1. Direct DuckDB Enhancement
- **Pros**: No new dependencies, simpler architecture
- **Cons**: Complex implementation, potential for missing resources, not Kubernetes-native

### 2. etcd Snapshot
- **Pros**: Native Kubernetes backup mechanism
- **Cons**: Low-level, requires manual resource reconstruction, complex for k3s

### 3. External Cloud Storage
- **Pros**: No local storage management, professional backup solution
- **Cons**: Requires internet connectivity, potential costs, privacy concerns

### 4. Custom Backup Solution
- **Pros**: Tailored to KECS needs
- **Cons**: High development effort, maintenance burden, reliability concerns

## Migration Path

### Version 1.0 (Opt-in)
- Velero backup disabled by default
- Enable via `--use-velero-backup` flag
- DuckDB restoration remains as fallback

### Version 1.1 (Default)
- Velero backup enabled by default
- Can disable via `--no-velero-backup` flag
- Deprecation warning for DuckDB restoration

### Version 1.2 (Cleanup)
- Remove DuckDB restoration code
- Velero becomes the only backup mechanism

## Monitoring and Metrics

- Backup success/failure rates
- Backup size and duration
- Restore success/failure rates
- Storage consumption per instance
- Time to complete stop/start operations

## Security Considerations

1. **MinIO credentials**: Generated per instance, stored securely
2. **Backup encryption**: Optional encryption at rest in MinIO
3. **Network isolation**: MinIO only accessible within cluster
4. **Access control**: RBAC for Velero service account

## References

- [Velero Documentation](https://velero.io/docs/)
- [MinIO Documentation](https://docs.min.io/)
- [Kubernetes Backup Best Practices](https://kubernetes.io/docs/concepts/cluster-administration/backup/)
- [ADR-0020: Kubernetes-ECS State Synchronization](./0020-kubernetes-ecs-state-synchronization.md)