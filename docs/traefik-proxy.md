# Traefik Proxy Integration

KECS supports Traefik as a reverse proxy to solve the LocalStack DNS resolution issues. When enabled, Traefik runs inside the k3d cluster and provides a single entry point for accessing services.

## Architecture

```
[KECS Host] → [localhost:8090] → [Traefik in k3d] → [LocalStack/Services]
                                        ↓
                                Port Forward (8090)
```

## Configuration

Enable Traefik by setting the environment variable:

```bash
export KECS_FEATURES_TRAEFIK=true
```

Or in your configuration file:

```yaml
features:
  traefik: true
```

## How it Works

1. When creating a k3d cluster, KECS maps port 8090 from the host to the cluster
2. After cluster creation, KECS deploys Traefik
3. Traefik routes requests to LocalStack and other services based on the path
4. LocalStack health checks and API calls go through `http://localhost:8090`

## Testing

1. Start KECS with Traefik enabled:
   ```bash
   KECS_FEATURES_TRAEFIK=true KECS_LOCALSTACK_ENABLED=true ./bin/kecs server
   ```

2. Create a cluster:
   ```bash
   aws --endpoint-url http://localhost:8080 ecs create-cluster --cluster-name test
   ```

3. Check LocalStack status:
   ```bash
   curl http://localhost:8090/_localstack/health
   ```

4. Access LocalStack services through Traefik:
   ```bash
   aws --endpoint-url http://localhost:8090 s3 ls
   ```

## Troubleshooting

### Check Traefik deployment
```bash
kubectl -n kecs-system get pods
kubectl -n kecs-system logs -l app=traefik
```

### Check port forwarding
```bash
docker ps | grep k3d
netstat -an | grep 8090
```

### Access Traefik dashboard
```bash
curl http://localhost:8090/dashboard/
```

## Benefits

- Solves DNS resolution issues permanently
- Single port for all services
- Easy to extend with more services
- Built-in load balancing and routing
- No need for complex port forwarding setup