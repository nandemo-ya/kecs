# Production Deployment Guide

This guide covers deploying KECS in production environments.

## Overview

KECS can be deployed in various production environments:
- Kubernetes clusters
- Docker Swarm
- Standalone Docker hosts
- Cloud platforms (AWS ECS, GKE, AKS, etc.)

## Prerequisites

- Docker 20.10+ (with buildx support for multi-platform builds)
- Kubernetes 1.24+ (if deploying to Kubernetes)
- Persistent storage for DuckDB database

## Building for Production

### 1. Build Production Docker Image

```bash
# Build for current platform
./scripts/build-production.sh

# Build and push multi-platform image
PUSH=true PLATFORMS=linux/amd64,linux/arm64 ./scripts/build-production.sh

# Build with specific version
VERSION=v1.0.0 ./scripts/build-production.sh
```

### 2. Configuration

#### Environment Variables

Copy `.env.production.example` to `.env.production` and customize:

```bash
cp .env.production.example .env.production
# Edit .env.production with your settings
```

Key environment variables:
- `PORT`: API server port (default: 8080)
- `ADMIN_PORT`: Admin server port (default: 8081)
- `KECS_UI_BASE_PATH`: Web UI base path (default: /ui)
- `KECS_STORAGE_PATH`: Database file path (default: /data/kecs.db)
- `KECS_LOG_LEVEL`: Logging level (default: info)

## Deployment Options

### Kubernetes Deployment

1. Create namespace:
```bash
kubectl create namespace kecs-system
```

2. Deploy KECS:
```bash
kubectl apply -f deployments/kubernetes/production/
```

3. Access the service:
```bash
# Port forward for local access
kubectl port-forward -n kecs-system svc/kecs-controlplane 8080:80

# Or use Ingress (configure domain in ingress.yaml first)
kubectl apply -f deployments/kubernetes/production/ingress.yaml
```

### Docker Compose Deployment

```yaml
version: '3.8'

services:
  kecs:
    image: ghcr.io/nandemo-ya/kecs:latest
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      - KECS_LOG_LEVEL=info
      - KECS_UI_ENABLED=true
    volumes:
      - kecs-data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
      interval: 30s
      timeout: 3s
      retries: 3

volumes:
  kecs-data:
```

### Standalone Docker

```bash
# Run with persistent storage
docker run -d \
  --name kecs \
  -p 8080:8080 \
  -p 8081:8081 \
  -v /path/to/data:/data \
  -e KECS_LOG_LEVEL=info \
  ghcr.io/nandemo-ya/kecs:latest

# Run with custom configuration
docker run -d \
  --name kecs \
  -p 8080:8080 \
  -p 8081:8081 \
  -v /path/to/data:/data \
  -v /path/to/config.yaml:/config/production.yaml \
  -e KECS_CONFIG=/config/production.yaml \
  ghcr.io/nandemo-ya/kecs:latest
```

## Health Checks and Monitoring

### Health Check Endpoints

- `/health` - Simple health check
- `/health/detailed` - Detailed health information
- `/ready` - Readiness probe (checks critical dependencies)
- `/live` - Liveness probe (basic aliveness check)

### Metrics Endpoints

- `/metrics` - JSON format metrics
- `/metrics/prometheus` - Prometheus format metrics

### Example Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'kecs'
    static_configs:
    - targets: ['kecs-controlplane:8081']
    metrics_path: '/metrics/prometheus'
```

## Security Considerations

### 1. Network Security

- Use TLS/SSL for all external communications
- Configure proper Ingress rules
- Restrict admin port (8081) access

### 2. Authentication & Authorization

- Enable authentication for production use
- Use service accounts for Kubernetes deployments
- Implement RBAC policies

### 3. Data Security

- Encrypt persistent volumes
- Regular backups of DuckDB database
- Secure environment variables and secrets

## Performance Tuning

### 1. Resource Allocation

Recommended resources:
```yaml
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 2Gi
```

### 2. Database Optimization

- Use SSD storage for DuckDB
- Configure appropriate connection pool size
- Regular database maintenance

### 3. Scaling

- Horizontal scaling: Run multiple replicas behind a load balancer
- Vertical scaling: Increase resources per instance
- Use HPA (Horizontal Pod Autoscaler) in Kubernetes

## Backup and Recovery

### Database Backup

```bash
# Backup DuckDB database
docker exec kecs cp /data/kecs.db /data/kecs-backup-$(date +%Y%m%d).db

# Restore from backup
docker exec kecs cp /data/kecs-backup-20240604.db /data/kecs.db
```

### Kubernetes Backup

Use Velero or similar tools for cluster-level backups:
```bash
velero backup create kecs-backup --include-namespaces kecs-system
```

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Check persistent volume permissions
   - Verify storage class availability
   - Check disk space

2. **WebSocket Connection Issues**
   - Ensure Ingress supports WebSocket
   - Check proxy timeout settings
   - Verify CORS configuration

3. **High Memory Usage**
   - Monitor goroutine count
   - Check for memory leaks
   - Adjust GC settings if needed

### Debug Mode

Enable debug logging:
```bash
docker run -e KECS_LOG_LEVEL=debug ...
```

### Performance Profiling

Access pprof endpoints (admin port):
- `/debug/pprof/` - Profile index
- `/debug/pprof/heap` - Heap profile
- `/debug/pprof/goroutine` - Goroutine profile

## Upgrading

### Zero-Downtime Upgrade (Kubernetes)

1. Update image tag in deployment
2. Apply rolling update:
```bash
kubectl set image -n kecs-system deployment/kecs-controlplane \
  controlplane=ghcr.io/nandemo-ya/kecs:v1.1.0
```

### Database Migration

Database schema migrations are handled automatically on startup.

## Support

- GitHub Issues: https://github.com/nandemo-ya/kecs/issues
- Documentation: https://github.com/nandemo-ya/kecs/docs
- Community: [Join our Discord/Slack]