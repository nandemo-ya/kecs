# Configuration Guide

## Overview

KECS can be configured through configuration files, environment variables, or command-line flags. This guide covers all configuration options and best practices.

## Configuration Hierarchy

Configuration is applied in the following order (later overrides earlier):

1. Default values
2. Configuration file
3. Environment variables
4. Command-line flags

## Configuration File

KECS supports YAML, JSON, and TOML configuration files.

### Basic Configuration

```yaml
# kecs-config.yaml
server:
  apiPort: 8080
  adminPort: 8081
  logLevel: info
  logFormat: json

storage:
  type: duckdb
  dataDir: /var/lib/kecs/data

kubernetes:
  kubeconfig: ~/.kube/config
  context: default
  cacheEnabled: true
```

### Complete Configuration Reference

```yaml
# Server Configuration
server:
  # API server port (default: 8080)
  apiPort: 8080
  
  # Admin server port for health/metrics (default: 8081)
  adminPort: 8081
  
  # Log level: trace, debug, info, warn, error (default: info)
  logLevel: info
  
  # Log format: json, text (default: json)
  logFormat: json
  
  # Enable pretty logging for development (default: false)
  prettyLog: false
  
  # Request timeout in seconds (default: 60)
  requestTimeout: 60
  
  # Graceful shutdown timeout in seconds (default: 30)
  shutdownTimeout: 30
  
  # CORS configuration
  cors:
    enabled: true
    allowOrigins: ["*"]
    allowMethods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowHeaders: ["*"]
    exposeHeaders: ["X-Total-Count"]
    allowCredentials: true
    maxAge: 86400

# TLS Configuration
tls:
  # Enable TLS (default: false)
  enabled: true
  
  # TLS certificate file path
  certFile: /etc/kecs/tls/server.crt
  
  # TLS key file path
  keyFile: /etc/kecs/tls/server.key
  
  # CA certificate file path for mTLS
  caFile: /etc/kecs/tls/ca.crt
  
  # Minimum TLS version: 1.0, 1.1, 1.2, 1.3 (default: 1.2)
  minVersion: "1.2"
  
  # Client authentication: NoClientCert, RequestClientCert, RequireAndVerifyClientCert
  clientAuth: NoClientCert
  
  # Cipher suites (if empty, uses secure defaults)
  cipherSuites: []

# Storage Configuration
storage:
  # Storage type: duckdb, postgres (default: duckdb)
  type: duckdb
  
  # Data directory (default: ~/.kecs/data)
  dataDir: /var/lib/kecs/data
  
  # DuckDB specific settings
  duckdb:
    # Database file path
    path: /var/lib/kecs/data/kecs.db
    
    # Memory limit for DuckDB
    memoryLimit: 4GB
    
    # Number of threads
    threads: 4
    
    # Enable WAL mode
    walEnabled: true
    
    # Checkpoint threshold
    checkpointThreshold: 1GB
    
    # Connection pool settings
    pool:
      maxConnections: 10
      maxIdleTime: 30m
      maxLifetime: 1h
  
  # PostgreSQL settings (if type: postgres)
  postgres:
    host: localhost
    port: 5432
    database: kecs
    username: kecs
    password: "${POSTGRES_PASSWORD}"
    sslMode: require
    
    # Connection pool settings
    pool:
      maxConnections: 25
      minConnections: 5
      maxIdleTime: 30m
      maxLifetime: 1h

# Kubernetes Configuration
kubernetes:
  # Kubeconfig file path (default: ~/.kube/config)
  kubeconfig: ~/.kube/config
  
  # Kubernetes context to use
  context: default
  
  # In-cluster configuration (when running inside Kubernetes)
  inCluster: false
  
  # Kubernetes API QPS
  qps: 100
  
  # Kubernetes API burst
  burst: 200
  
  # Request timeout
  timeout: 30s
  
  # Enable client-side caching
  cacheEnabled: true
  
  # Cache TTL
  cacheTTL: 5m
  
  # Namespace for KECS resources (default: kecs-{cluster-name})
  namespacePrefix: kecs
  
  # Resource labels
  labels:
    managed-by: kecs
    version: v1.0.0

# Authentication Configuration
auth:
  # Enable authentication (default: false)
  enabled: true
  
  # Authentication type: none, api-key, jwt, oauth2, mtls
  type: jwt
  
  # API Key authentication
  apiKey:
    # Header name for API key
    headerName: X-API-Key
    
    # API keys (can also be loaded from file or secrets)
    keys:
      - name: production
        key: "${API_KEY_PRODUCTION}"
        permissions: ["read", "write"]
      - name: readonly
        key: "${API_KEY_READONLY}"
        permissions: ["read"]
  
  # JWT authentication
  jwt:
    # JWT secret (or public key for RS256)
    secret: "${JWT_SECRET}"
    
    # JWT issuer
    issuer: kecs
    
    # JWT audience
    audience: kecs-api
    
    # Token expiration time
    expirationTime: 24h
    
    # Algorithm: HS256, RS256
    algorithm: HS256
  
  # OAuth2 configuration
  oauth2:
    provider: google
    clientId: "${OAUTH_CLIENT_ID}"
    clientSecret: "${OAUTH_CLIENT_SECRET}"
    redirectURL: https://kecs.example.com/auth/callback
    scopes: ["openid", "profile", "email"]

# LocalStack Integration
localstack:
  # Enable LocalStack integration
  enabled: false
  
  # LocalStack endpoint
  endpoint: http://localhost:4566
  
  # AWS region
  region: us-east-1
  
  # Services to use with LocalStack
  services:
    - s3
    - dynamodb
    - sqs
    - sns
    - secretsmanager
    - ssm
    - iam
    - logs
    - cloudwatch
  
  # Automatic sidecar injection
  sidecarInjection:
    enabled: true
    image: localstack/localstack:latest
    pullPolicy: IfNotPresent

# Metrics Configuration
metrics:
  # Enable metrics endpoint
  enabled: true
  
  # Metrics path
  path: /metrics
  
  # Include detailed metrics
  detailed: true
  
  # Metric namespaces to include
  namespaces:
    - kecs_api
    - kecs_storage
    - kecs_kubernetes

# Tracing Configuration
tracing:
  # Enable distributed tracing
  enabled: false
  
  # Tracing backend: jaeger, zipkin, otlp
  backend: jaeger
  
  # Jaeger configuration
  jaeger:
    # Collector endpoint
    endpoint: http://localhost:14268/api/traces
    
    # Agent host:port
    agentHost: localhost:6831
    
    # Sampling rate (0.0 to 1.0)
    samplingRate: 0.1
  
  # Service name for traces
  serviceName: kecs-control-plane

# Rate Limiting
rateLimiting:
  # Enable rate limiting
  enabled: true
  
  # Global rate limit (requests per second)
  global: 1000
  
  # Per-IP rate limit
  perIP: 100
  
  # Burst size
  burst: 50
  
  # Excluded IPs
  excludedIPs:
    - 127.0.0.1
    - 10.0.0.0/8

# Cache Configuration
cache:
  # Enable caching
  enabled: true
  
  # Cache type: memory, redis
  type: memory
  
  # Memory cache settings
  memory:
    # Maximum number of items
    maxItems: 10000
    
    # Default TTL
    defaultTTL: 5m
    
    # Cleanup interval
    cleanupInterval: 10m
  
  # Redis cache settings
  redis:
    # Redis address
    address: localhost:6379
    
    # Redis password
    password: "${REDIS_PASSWORD}"
    
    # Redis database
    database: 0
    
    # Key prefix
    keyPrefix: kecs:

# Feature Flags
features:
  # Enable Web UI
  webUI: true
  
  # Enable WebSocket support
  websocket: true
  
  # Enable GraphQL API
  graphql: false
  
  # Enable experimental features
  experimental:
    # Dynamic cluster creation
    dynamicClusters: true
    
    # Multi-region support
    multiRegion: false
    
    # Advanced scheduling
    advancedScheduling: false

# Audit Configuration
audit:
  # Enable audit logging
  enabled: true
  
  # Audit log file
  logFile: /var/log/kecs/audit.log
  
  # Audit events to log
  events:
    - cluster.create
    - cluster.delete
    - service.create
    - service.update
    - service.delete
    - task.run
    - task.stop
    - auth.login
    - auth.logout
  
  # Include request/response bodies
  includeBody: false
  
  # Maximum body size to log
  maxBodySize: 1024

# Backup Configuration
backup:
  # Enable automatic backups
  enabled: true
  
  # Backup schedule (cron format)
  schedule: "0 2 * * *"
  
  # Backup retention days
  retention: 7
  
  # Backup storage
  storage:
    # Storage type: local, s3
    type: s3
    
    # S3 configuration
    s3:
      bucket: kecs-backups
      region: us-east-1
      prefix: backups/
      
    # Local storage
    local:
      path: /var/backups/kecs

# Health Check Configuration
health:
  # Liveness probe configuration
  liveness:
    # Enable liveness endpoint
    enabled: true
    
    # Liveness path
    path: /health/live
  
  # Readiness probe configuration
  readiness:
    # Enable readiness endpoint
    enabled: true
    
    # Readiness path
    path: /health/ready
    
    # Check storage connectivity
    checkStorage: true
    
    # Check Kubernetes connectivity
    checkKubernetes: true
```

## Environment Variables

All configuration options can be set via environment variables using the prefix `KECS_`:

```bash
# Server configuration
export KECS_SERVER_APIPORT=8080
export KECS_SERVER_ADMINPORT=8081
export KECS_SERVER_LOGLEVEL=debug

# Storage configuration
export KECS_STORAGE_TYPE=duckdb
export KECS_STORAGE_DATADIR=/var/lib/kecs/data

# Kubernetes configuration
export KECS_KUBERNETES_KUBECONFIG=/etc/kecs/kubeconfig
export KECS_KUBERNETES_CONTEXT=production

# Authentication
export KECS_AUTH_ENABLED=true
export KECS_AUTH_TYPE=jwt
export KECS_AUTH_JWT_SECRET=supersecret

# Cluster management
export KECS_KEEP_CLUSTERS_ON_SHUTDOWN=true  # Keep k3d clusters when KECS stops

# Data persistence
export KECS_DATA_DIR=/data  # Data directory path (useful in container mode)
```

### Environment Variable Mapping

| Configuration Path | Environment Variable |
|-------------------|---------------------|
| server.apiPort | KECS_SERVER_APIPORT |
| storage.type | KECS_STORAGE_TYPE |
| kubernetes.inCluster | KECS_KUBERNETES_INCLUSTER |
| auth.jwt.secret | KECS_AUTH_JWT_SECRET |

## Command-Line Flags

```bash
# Server flags
./bin/kecs server \
  --api-port 8080 \
  --admin-port 8081 \
  --log-level debug \
  --config /etc/kecs/config.yaml

# All available flags
./bin/kecs server --help
```

### Common Flags

| Flag | Description | Default |
|------|-------------|---------|
| --api-port | API server port | 8080 |
| --admin-port | Admin server port | 8081 |
| --log-level | Log level | info |
| --log-format | Log format (json/text) | json |
| --config | Configuration file path | |
| --data-dir | Data directory | ~/.kecs/data |
| --kubeconfig | Kubernetes config file | ~/.kube/config |

## Configuration Best Practices

### 1. Use Configuration Files

Store configuration in files for reproducibility:

```yaml
# base-config.yaml - Shared configuration
server:
  logFormat: json
  requestTimeout: 60

storage:
  type: duckdb
```

```yaml
# production-config.yaml - Production overrides
!include base-config.yaml

server:
  logLevel: info
  
tls:
  enabled: true
  certFile: /etc/kecs/tls/server.crt
  keyFile: /etc/kecs/tls/server.key
```

### 2. Use Environment Variables for Secrets

Never store secrets in configuration files:

```yaml
auth:
  jwt:
    secret: "${JWT_SECRET}"

storage:
  postgres:
    password: "${POSTGRES_PASSWORD}"
```

### 3. Validate Configuration

Use the validate command before deploying:

```bash
./bin/kecs validate --config /etc/kecs/config.yaml
```

### 4. Configuration Templates

Use templating for dynamic configuration:

```bash
# Using envsubst
envsubst < config.yaml.template > config.yaml

# Using Helm
helm template kecs ./charts/kecs -f values.yaml
```

## Configuration Examples

### Development Configuration

```yaml
server:
  apiPort: 8080
  adminPort: 8081
  logLevel: debug
  prettyLog: true

storage:
  type: duckdb
  dataDir: ./data

kubernetes:
  kubeconfig: ~/.kube/config

auth:
  enabled: false

features:
  webUI: true
  experimental:
    dynamicClusters: true
```

### Production Configuration

```yaml
server:
  apiPort: 8080
  adminPort: 8081
  logLevel: info
  logFormat: json

tls:
  enabled: true
  certFile: /etc/kecs/tls/server.crt
  keyFile: /etc/kecs/tls/server.key
  minVersion: "1.2"

storage:
  type: postgres
  postgres:
    host: postgres.kecs.svc.cluster.local
    port: 5432
    database: kecs
    username: kecs
    password: "${POSTGRES_PASSWORD}"
    sslMode: require
    pool:
      maxConnections: 50

kubernetes:
  inCluster: true
  cacheEnabled: true
  cacheTTL: 5m

auth:
  enabled: true
  type: jwt
  jwt:
    secret: "${JWT_SECRET}"
    issuer: kecs
    expirationTime: 24h

metrics:
  enabled: true
  detailed: true

audit:
  enabled: true
  logFile: /var/log/kecs/audit.log

backup:
  enabled: true
  schedule: "0 2 * * *"
  retention: 30
  storage:
    type: s3
    s3:
      bucket: kecs-backups-prod
```

### High Availability Configuration

```yaml
server:
  apiPort: 8080
  adminPort: 8081
  logLevel: info

storage:
  type: postgres
  postgres:
    # Use connection pooler
    host: pgbouncer.kecs.svc.cluster.local
    port: 6432
    database: kecs
    pool:
      maxConnections: 100
      minConnections: 20

kubernetes:
  inCluster: true
  # Increase limits for HA
  qps: 200
  burst: 400
  cacheEnabled: true

# Enable distributed caching
cache:
  enabled: true
  type: redis
  redis:
    # Redis Sentinel for HA
    address: redis-sentinel.kecs.svc.cluster.local:26379
    sentinelMasterName: mymaster

# Rate limiting for stability
rateLimiting:
  enabled: true
  global: 10000
  perIP: 1000
```

## Troubleshooting Configuration

### Configuration Validation

```bash
# Validate configuration file
./bin/kecs validate --config config.yaml

# Test configuration loading
./bin/kecs server --config config.yaml --dry-run
```

### Debug Configuration Loading

```bash
# Enable debug logging for configuration
KECS_LOG_LEVEL=debug ./bin/kecs server --config config.yaml 2>&1 | grep config
```

### Common Issues

1. **Environment variables not working**
   - Check variable naming (KECS_ prefix)
   - Ensure no spaces around = in exports
   - Use quotes for values with spaces

2. **Configuration file not found**
   - Use absolute paths
   - Check file permissions
   - Verify working directory

3. **Merge conflicts**
   - Later sources override earlier ones
   - Use --print-config to see final configuration

## Cluster Lifecycle Management

KECS manages k3d clusters automatically for running ECS tasks. By default, these clusters are cleaned up when KECS shuts down.

### Graceful Shutdown Behavior

When KECS receives a shutdown signal (SIGTERM/SIGINT):

1. **Default behavior**: All k3d clusters are deleted
   - Ensures clean system state
   - Prevents resource leaks
   - Suitable for development and CI/CD environments

2. **Keep clusters on shutdown**: Set `KECS_KEEP_CLUSTERS_ON_SHUTDOWN=true`
   - k3d clusters remain after KECS stops
   - Useful for debugging or when managing clusters manually
   - Clusters can be reused when KECS restarts

### Configuration Options

```bash
# Keep k3d clusters when KECS stops (default: false)
export KECS_KEEP_CLUSTERS_ON_SHUTDOWN=true

# Skip k3d operations in test mode (default: false)
export KECS_TEST_MODE=true
```

### Manual Cluster Cleanup

If clusters are retained, you can clean them up manually:

```bash
# List all KECS k3d clusters
k3d cluster list | grep kecs-

# Delete specific cluster
k3d cluster delete kecs-my-cluster

# Delete all KECS clusters
k3d cluster list | grep kecs- | awk '{print $1}' | xargs -I {} k3d cluster delete {}
```

## Next Steps

- [Security Configuration](/guides/security) - Security hardening
- [Performance Tuning](/guides/performance) - Optimization guide
- [Monitoring Setup](/guides/monitoring) - Metrics and monitoring