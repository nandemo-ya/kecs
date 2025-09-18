# KECS State Recovery Guide

This guide explains how KECS recovers k3d clusters and Kubernetes resources after a restart.

## Overview

When KECS is restarted (planned or unplanned), the k3d clusters it manages are typically lost. State Recovery automatically recreates these clusters and redeploys services based on the persisted state in the database.

## How It Works

### 1. State Persistence
KECS stores the following information in its database:
- Cluster configurations (including k3d cluster names)
- Service definitions
- Task definitions
- Deployment metadata

### 2. Recovery Process
On startup, if state recovery is enabled:

1. **Cluster Recovery**: KECS checks each cluster in the database
   - If the k3d cluster doesn't exist, it recreates it
   - Waits for the cluster to be ready before proceeding

2. **Service Recovery**: For each recovered cluster
   - Recreates Kubernetes namespaces
   - Redeploys services based on stored configurations
   - Restores deployment metadata

### 3. Recovery Modes

#### Automatic Recovery (Default)
```bash
# State recovery is enabled by default
./bin/kecs server

# Or explicitly enable it
export KECS_AUTO_RECOVER_STATE=true
./bin/kecs server
```

#### Disable Recovery
```bash
# Disable state recovery
export KECS_AUTO_RECOVER_STATE=false
./bin/kecs server
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KECS_AUTO_RECOVER_STATE` | `true` | Enable/disable automatic state recovery on startup |
| `KECS_KEEP_CLUSTERS_ON_SHUTDOWN` | `false` | Keep k3d clusters when KECS stops (useful with recovery) |

### Best Practices

#### For Development
```bash
# Keep clusters between restarts for faster development
export KECS_KEEP_CLUSTERS_ON_SHUTDOWN=true
export KECS_AUTO_RECOVER_STATE=false
```

#### For Production
```bash
# Clean shutdown and automatic recovery
export KECS_KEEP_CLUSTERS_ON_SHUTDOWN=false
export KECS_AUTO_RECOVER_STATE=true
```

## Recovery Scenarios

### Scenario 1: Planned Restart
1. KECS receives shutdown signal
2. K3d clusters are cleaned up (unless `KECS_KEEP_CLUSTERS_ON_SHUTDOWN=true`)
3. On restart, clusters and services are recreated
4. Services resume normal operation

### Scenario 2: Crash Recovery
1. KECS crashes unexpectedly
2. K3d clusters remain running (orphaned)
3. On restart, KECS detects existing clusters and reuses them
4. Services are reconnected to existing deployments

### Scenario 3: Container Restart
```yaml
# docker-compose.yml
services:
  kecs:
    image: ghcr.io/nandemo-ya/kecs:latest
    restart: unless-stopped
    volumes:
      - ./kecs-data:/data  # Persistent state
    environment:
      - KECS_CONTAINER_MODE=true
      - KECS_DATA_DIR=/data
      - KECS_AUTO_RECOVER_STATE=true
```

## Monitoring Recovery

### Log Messages
Watch for these log messages during startup:

```
Starting state recovery...
Found 3 clusters in storage, checking which need recovery...
Recovering k3d cluster kecs-production for ECS cluster production...
Successfully recovered k3d cluster kecs-production
Found 5 services to recover for cluster production
Successfully recovered service web-app
State recovery summary: 3 recovered, 0 skipped, 0 failed
State recovery completed
```

### Health Checks
Verify recovery success:

```bash
# Check cluster status
aws --endpoint-url http://localhost:8080 ecs list-clusters

# Check k3d clusters
k3d cluster list

# Check services
aws --endpoint-url http://localhost:8080 ecs list-services --cluster <cluster-name>
```

## Troubleshooting

### Recovery Failures

#### Symptom: "Failed to recreate k3d cluster"
**Cause**: Docker daemon issues or resource constraints

**Solution**:
1. Check Docker daemon is running
2. Verify sufficient resources (CPU, memory, disk)
3. Check for port conflicts
4. Manually clean up orphaned clusters: `k3d cluster delete <cluster-name>`

#### Symptom: "Failed to recover services"
**Cause**: Task definition missing or invalid

**Solution**:
1. Verify task definitions exist in database
2. Check task definition validity
3. Review service configurations
4. Check Kubernetes API connectivity

### Manual Recovery

If automatic recovery fails, you can manually recover:

```bash
# 1. List clusters from database
aws --endpoint-url http://localhost:8080 ecs list-clusters

# 2. Manually create k3d cluster
k3d cluster create kecs-<cluster-name>

# 3. Restart KECS to reconnect
./bin/kecs server
```

### Disable Recovery for Specific Clusters

Currently, recovery is all-or-nothing. To skip specific clusters:

1. Delete them from the database before restart
2. Or set their status to INACTIVE

## Performance Considerations

### Recovery Time
- Cluster creation: 30-60 seconds per cluster
- Service deployment: 5-10 seconds per service
- Total time: Depends on number of resources

### Parallel Recovery
Currently, clusters are recovered sequentially. Future improvements may include:
- Parallel cluster creation
- Batch service deployment
- Progressive recovery with health checks

### Resource Usage
During recovery:
- High CPU usage for k3d cluster creation
- Increased memory for multiple Kubernetes API connections
- Network bandwidth for pulling container images

## Limitations

1. **Task State**: Individual task states are not recovered
2. **In-Progress Operations**: Operations interrupted by shutdown are not resumed
3. **External Resources**: Resources outside KECS control (e.g., load balancers) need manual intervention
4. **Secrets**: Kubernetes secrets need to be recreated if they contained dynamic values

## Best Practices

1. **Regular Backups**: Backup the database regularly
   ```bash
   cp /data/kecs.db /backup/kecs-$(date +%Y%m%d).db
   ```

2. **Health Monitoring**: Monitor recovery success
   ```bash
   # Add to monitoring system
   curl -f http://localhost:8081/health || alert "KECS unhealthy"
   ```

3. **Gradual Rollout**: Test recovery in staging before production

4. **Resource Planning**: Ensure sufficient resources for recovery
   - CPU: 2 cores minimum
   - Memory: 4GB minimum
   - Disk: 20GB for k3d images

5. **Recovery Testing**: Regularly test recovery procedures
   ```bash
   ./scripts/test/test-state-recovery.sh
   ```

## Future Improvements

Planned enhancements for state recovery:

1. **Selective Recovery**: Choose which resources to recover
2. **Progressive Recovery**: Recover critical services first
3. **Recovery Webhooks**: Notify external systems of recovery status
4. **State Export/Import**: Backup and restore specific configurations
5. **Multi-Region Recovery**: Support for distributed deployments