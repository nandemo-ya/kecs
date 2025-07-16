# KECS Admin API Documentation

The KECS Admin Server provides operational endpoints for health checks, metrics, and configuration management.

## Base URL

The Admin Server runs on port 8081 by default (configurable via `--admin-port` flag).

```
http://localhost:8081
```

## Endpoints

### Health Check Endpoints

#### GET /health
Basic health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-16T10:00:00Z"
}
```

#### GET /healthz
Legacy health check endpoint for backward compatibility.

**Response:**
```json
{
  "status": "OK",
  "timestamp": "2025-01-16T10:00:00Z",
  "version": "0.1.0"
}
```

#### GET /live
Kubernetes liveness probe endpoint.

**Response:**
- Status: 200 OK if the server is alive
- Status: 503 Service Unavailable if the server is not alive

#### GET /ready
Kubernetes readiness probe endpoint.

**Response:**
- Status: 200 OK if the server is ready to accept requests
- Status: 503 Service Unavailable if the server is not ready

#### GET /health/detailed
Detailed health check with component status.

**Response:**
```json
{
  "status": "healthy",
  "components": {
    "storage": "healthy",
    "kubernetes": "healthy",
    "localstack": "disabled"
  },
  "timestamp": "2025-01-16T10:00:00Z"
}
```

### Metrics Endpoints

#### GET /metrics
Basic metrics in JSON format.

**Response:**
```json
{
  "requests_total": 1000,
  "requests_per_second": 10.5,
  "active_connections": 25,
  "errors_total": 5
}
```

#### GET /metrics/prometheus
Metrics in Prometheus format.

**Response:**
```
# HELP kecs_requests_total Total number of requests
# TYPE kecs_requests_total counter
kecs_requests_total 1000
# HELP kecs_active_connections Number of active connections
# TYPE kecs_active_connections gauge
kecs_active_connections 25
```

### Configuration Endpoint

#### GET /config
Returns the current server configuration (non-sensitive values only).

**Response:**
```json
{
  "server": {
    "port": 8080,
    "adminPort": 8081,
    "dataDir": "/home/user/.kecs/data",
    "logLevel": "info",
    "endpoint": "",
    "allowedOrigins": []
  },
  "localstack": {
    "enabled": false,
    "useTraefik": false,
    "port": 4566,
    "edgePort": 4566
  },
  "kubernetes": {
    "kubeconfigPath": "",
    "k3dOptimized": false,
    "k3dAsync": false,
    "disableCoreDNS": false,
    "keepClustersOnShutdown": false
  },
  "features": {
    "testMode": false,
    "containerMode": false,
    "autoRecoverState": true,
    "traefik": false
  },
  "aws": {
    "defaultRegion": "us-east-1",
    "accountID": "000000000000"
  }
}
```

## Usage Examples

### Check Server Health
```bash
curl http://localhost:8081/health
```

### Get Current Configuration
```bash
curl http://localhost:8081/config | jq .
```

### Monitor Metrics
```bash
# JSON format
curl http://localhost:8081/metrics

# Prometheus format
curl http://localhost:8081/metrics/prometheus
```

### Kubernetes Probes
```yaml
# In your Kubernetes deployment
livenessProbe:
  httpGet:
    path: /live
    port: 8081
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /ready
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Security Considerations

- The Admin Server is intended for internal use only
- It should not be exposed to the public internet
- Consider using network policies or firewall rules to restrict access
- Sensitive configuration values (like secrets) are not exposed through the config endpoint