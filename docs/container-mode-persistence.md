# Container Mode Persistence Guide

This guide explains how to configure KECS for data persistence when running in container mode.

## Overview

When running KECS in container mode (`KECS_CONTAINER_MODE=true`), the default data directory is `/data` inside the container. Without proper volume mounting, all data (clusters, services, tasks) will be lost when the container is removed.

## Configuration

### Environment Variables

- `KECS_DATA_DIR`: Path to the data directory (default: `/data` in container mode)
- `KECS_CONTAINER_MODE`: Set to `true` when running in a container

### Docker Compose

The recommended way to run KECS with persistence is using Docker Compose:

```yaml
version: '3.8'

services:
  kecs:
    image: ghcr.io/nandemo-ya/kecs:latest
    environment:
      - KECS_CONTAINER_MODE=true
      - KECS_DATA_DIR=/data
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      # Persist data between container restarts
      - ./kecs-data:/data
      # Required for k3d cluster management
      - /var/run/docker.sock:/var/run/docker.sock
```

### Docker Run

If using `docker run` directly:

```bash
docker run -d \
  --name kecs \
  -p 8080:8080 \
  -p 8081:8081 \
  -e KECS_CONTAINER_MODE=true \
  -e KECS_DATA_DIR=/data \
  -v $(pwd)/kecs-data:/data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/nandemo-ya/kecs:latest
```

### Kubernetes

For Kubernetes deployments, use a PersistentVolumeClaim:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: kecs-data-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kecs-controlplane
spec:
  template:
    spec:
      containers:
      - name: kecs
        image: ghcr.io/nandemo-ya/kecs:latest
        env:
        - name: KECS_CONTAINER_MODE
          value: "true"
        - name: KECS_DATA_DIR
          value: "/data"
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: kecs-data-pvc
```

## Data Directory Structure

The data directory contains:

```
/data/
├── kecs.db          # DuckDB database file
├── kecs.db.wal      # Write-ahead log (if enabled)
└── .lock            # Lock file for exclusive access
```

## Important Considerations

### 1. File Permissions

The KECS container runs as a non-root user (UID 65532). Ensure the mounted volume has appropriate permissions:

```bash
# Create directory with correct permissions
mkdir -p kecs-data
chmod 755 kecs-data

# If needed, change ownership (on Linux)
sudo chown 65532:65532 kecs-data
```

### 2. Docker Socket Access

KECS needs access to the Docker socket to manage k3d clusters. This requires:

1. Mounting `/var/run/docker.sock`
2. Ensuring the container user has access to the socket

On some systems, you may need to run the container with additional permissions:

```yaml
# docker-compose.yml
services:
  kecs:
    # ... other config ...
    group_add:
      - docker  # or the GID of the docker group
```

### 3. Backup and Recovery

#### Backup

To backup KECS data:

```bash
# Stop KECS to ensure consistency
docker-compose stop kecs

# Backup the data directory
tar -czf kecs-backup-$(date +%Y%m%d-%H%M%S).tar.gz kecs-data/

# Restart KECS
docker-compose start kecs
```

#### Recovery

To restore from backup:

```bash
# Stop KECS
docker-compose stop kecs

# Remove existing data
rm -rf kecs-data/

# Extract backup
tar -xzf kecs-backup-20240124-120000.tar.gz

# Restart KECS
docker-compose start kecs
```

### 4. Migration Between Environments

To migrate data between different environments:

1. **Export from source environment:**
   ```bash
   docker run --rm \
     -v source-kecs-data:/data \
     -v $(pwd):/backup \
     busybox tar czf /backup/kecs-export.tar.gz -C /data .
   ```

2. **Import to target environment:**
   ```bash
   docker run --rm \
     -v target-kecs-data:/data \
     -v $(pwd):/backup \
     busybox tar xzf /backup/kecs-export.tar.gz -C /data
   ```

## Troubleshooting

### Permission Denied Errors

If you see permission errors when starting KECS:

```bash
# Check ownership of data directory
ls -la kecs-data/

# Fix ownership (Linux)
sudo chown -R 65532:65532 kecs-data/

# Fix permissions
chmod -R 755 kecs-data/
```

### Database Lock Errors

If KECS reports database lock errors:

1. Ensure only one instance of KECS is running
2. Remove stale lock file if KECS crashed:
   ```bash
   rm kecs-data/.lock
   ```

### Docker Socket Access Errors

If KECS cannot access Docker:

1. Check Docker socket permissions:
   ```bash
   ls -la /var/run/docker.sock
   ```

2. Add user to docker group or run with appropriate GID:
   ```yaml
   # docker-compose.yml
   group_add:
     - $(stat -c '%g' /var/run/docker.sock)
   ```

## Best Practices

1. **Regular Backups**: Schedule regular backups of the data directory
2. **Monitor Disk Space**: Ensure adequate disk space for the data volume
3. **Use Named Volumes**: In production, use named Docker volumes for better management
4. **Separate Data and Logs**: Consider separate volumes for data and logs
5. **High Availability**: For HA setups, consider database replication strategies

## Limitations

1. **Single Instance**: Currently, KECS supports only single-instance deployments due to DuckDB limitations
2. **Live Migration**: Database files cannot be copied while KECS is running
3. **Cross-Platform**: Be careful with file permissions when sharing volumes between different OS