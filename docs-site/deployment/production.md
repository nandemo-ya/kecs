# Production Deployment

## Overview

This guide covers deploying KECS in production environments with high availability, security, and scalability considerations.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Load Balancer                             │
│                    (AWS ALB / NLB / Nginx)                      │
└───────────────┬─────────────────────────────┬───────────────────┘
                │                             │
    ┌───────────▼───────────┐     ┌──────────▼────────────┐
    │   KECS Control Plane  │     │  KECS Control Plane   │
    │     (Primary)         │     │    (Secondary)        │
    └───────────┬───────────┘     └──────────┬────────────┘
                │                             │
    ┌───────────▼─────────────────────────────▼───────────┐
    │              Shared Storage (DuckDB)                 │
    │            (NFS / EFS / Cloud Storage)              │
    └─────────────────────────────────────────────────────┘
                │                             │
    ┌───────────▼───────────┐     ┌──────────▼────────────┐
    │  Kubernetes Cluster   │     │  Kubernetes Cluster   │
    │    (Region A)         │     │    (Region B)         │
    └───────────────────────┘     └───────────────────────┘
```

## Prerequisites

- Kubernetes cluster (EKS, GKE, AKS, or self-managed)
- Persistent storage solution
- Load balancer
- TLS certificates
- Monitoring infrastructure

## Deployment Options

### Option 1: Kubernetes Deployment

#### High Availability Setup

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: kecs-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: kecs-config
  namespace: kecs-system
data:
  config.yaml: |
    server:
      port: 8080
      adminPort: 8081
      logLevel: info
    
    storage:
      type: duckdb
      duckdb:
        path: /data/kecs.db
        backupEnabled: true
        backupSchedule: "0 2 * * *"
        backupRetention: 7
    
    kubernetes:
      inCluster: true
      cacheEnabled: true
      cacheTTL: 5m
    
    auth:
      enabled: true
      type: jwt
      jwt:
        issuer: kecs
        audience: kecs-api
        expirationTime: 24h
    
    metrics:
      enabled: true
      path: /metrics
    
    tracing:
      enabled: true
      jaeger:
        endpoint: http://jaeger-collector:14268/api/traces
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: kecs-data
  namespace: kecs-system
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: efs-storage-class
  resources:
    requests:
      storage: 100Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kecs-control-plane
  namespace: kecs-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: kecs-control-plane
  template:
    metadata:
      labels:
        app: kecs-control-plane
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: app
                    operator: In
                    values:
                      - kecs-control-plane
              topologyKey: kubernetes.io/hostname
      serviceAccountName: kecs-controller
      containers:
        - name: kecs
          image: ghcr.io/nandemo-ya/kecs:v1.0.0
          command: ["/kecs", "server"]
          env:
            - name: KECS_CONFIG_PATH
              value: /config/config.yaml
            - name: KECS_JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: kecs-secrets
                  key: jwt-secret
          ports:
            - containerPort: 8080
              name: api
            - containerPort: 8081
              name: admin
          volumeMounts:
            - name: config
              mountPath: /config
            - name: data
              mountPath: /data
            - name: tls
              mountPath: /tls
          livenessProbe:
            httpGet:
              path: /health
              port: admin
              scheme: HTTPS
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /health/ready
              port: admin
              scheme: HTTPS
            initialDelaySeconds: 10
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3
          resources:
            requests:
              memory: "1Gi"
              cpu: "500m"
            limits:
              memory: "4Gi"
              cpu: "2000m"
      volumes:
        - name: config
          configMap:
            name: kecs-config
        - name: data
          persistentVolumeClaim:
            claimName: kecs-data
        - name: tls
          secret:
            secretName: kecs-tls
---
apiVersion: v1
kind: Service
metadata:
  name: kecs-api
  namespace: kecs-system
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
spec:
  type: LoadBalancer
  selector:
    app: kecs-control-plane
  ports:
    - port: 443
      targetPort: api
      name: api
      protocol: TCP
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: kecs-control-plane
  namespace: kecs-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: kecs-control-plane
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

### Option 2: VM-Based Deployment

#### SystemD Service

```ini
[Unit]
Description=KECS Control Plane
Documentation=https://github.com/nandemo-ya/kecs
After=network.target

[Service]
Type=notify
ExecStart=/usr/local/bin/kecs server --config /etc/kecs/config.yaml
ExecReload=/bin/kill -s HUP $MAINPID
KillMode=mixed
KillSignal=SIGTERM
Restart=on-failure
RestartSec=5s
User=kecs
Group=kecs

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/kecs

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

#### HAProxy Configuration

```
global
    log /dev/log local0
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    daemon

    # TLS
    ssl-default-bind-ciphers ECDHE+AESGCM:ECDHE+AES256:ECDHE+AES128:!PSK:!DHE:!RSA
    ssl-default-bind-options no-sslv3 no-tlsv10 no-tlsv11

defaults
    log     global
    mode    http
    option  httplog
    option  dontlognull
    timeout connect 5000
    timeout client  50000
    timeout server  50000
    errorfile 400 /etc/haproxy/errors/400.http
    errorfile 403 /etc/haproxy/errors/403.http
    errorfile 408 /etc/haproxy/errors/408.http
    errorfile 500 /etc/haproxy/errors/500.http
    errorfile 502 /etc/haproxy/errors/502.http
    errorfile 503 /etc/haproxy/errors/503.http
    errorfile 504 /etc/haproxy/errors/504.http

frontend kecs_frontend
    bind *:443 ssl crt /etc/haproxy/certs/kecs.pem
    redirect scheme https if !{ ssl_fc }
    
    # API routing
    use_backend kecs_api if { path_beg /v1/ }
    use_backend kecs_websocket if { path_beg /ws }
    use_backend kecs_ui if { path_beg /ui }
    
    default_backend kecs_api

backend kecs_api
    balance roundrobin
    option httpchk GET /health
    http-check expect status 200
    
    server kecs1 10.0.1.10:8080 check ssl verify none
    server kecs2 10.0.1.11:8080 check ssl verify none
    server kecs3 10.0.1.12:8080 check ssl verify none

backend kecs_websocket
    balance source
    option http-server-close
    option forceclose
    
    server kecs1 10.0.1.10:8080 check ssl verify none
    server kecs2 10.0.1.11:8080 check ssl verify none
    server kecs3 10.0.1.12:8080 check ssl verify none

backend kecs_ui
    balance roundrobin
    
    server kecs1 10.0.1.10:8080 check ssl verify none
    server kecs2 10.0.1.11:8080 check ssl verify none
    server kecs3 10.0.1.12:8080 check ssl verify none
```

## Security Configuration

### TLS Configuration

```yaml
# kecs-config.yaml
server:
  tls:
    enabled: true
    certFile: /tls/server.crt
    keyFile: /tls/server.key
    caFile: /tls/ca.crt
    minVersion: "1.2"
    cipherSuites:
      - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
      - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
      - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
      - TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kecs-control-plane
  namespace: kecs-system
spec:
  podSelector:
    matchLabels:
      app: kecs-control-plane
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080
    - from:
        - namespaceSelector:
            matchLabels:
              name: monitoring
      ports:
        - protocol: TCP
          port: 8081
  egress:
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443
        - protocol: TCP
          port: 6443
    - to:
        - namespaceSelector:
            matchLabels:
              name: kecs-system
      ports:
        - protocol: TCP
          port: 5432
```

### RBAC Configuration

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kecs-controller
  namespace: kecs-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kecs-controller
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: [""]
    resources: ["pods", "pods/log", "pods/exec"]
    verbs: ["*"]
  - apiGroups: [""]
    resources: ["services", "endpoints"]
    verbs: ["*"]
  - apiGroups: [""]
    resources: ["configmaps", "secrets"]
    verbs: ["*"]
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets"]
    verbs: ["*"]
  - apiGroups: ["batch"]
    resources: ["jobs", "cronjobs"]
    verbs: ["*"]
  - apiGroups: ["autoscaling"]
    resources: ["horizontalpodautoscalers"]
    verbs: ["*"]
  - apiGroups: ["policy"]
    resources: ["poddisruptionbudgets"]
    verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kecs-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kecs-controller
subjects:
  - kind: ServiceAccount
    name: kecs-controller
    namespace: kecs-system
```

## Storage Configuration

### Database Replication

For production, consider database replication:

```yaml
storage:
  type: postgres
  postgres:
    primary:
      host: postgres-primary.kecs-system.svc.cluster.local
      port: 5432
      database: kecs
      sslMode: require
    replicas:
      - host: postgres-replica-1.kecs-system.svc.cluster.local
        port: 5432
      - host: postgres-replica-2.kecs-system.svc.cluster.local
        port: 5432
    pool:
      maxConnections: 100
      minConnections: 10
      maxIdleTime: 30m
```

### Backup Strategy

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: kecs-backup
  namespace: kecs-system
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: ghcr.io/nandemo-ya/kecs-backup:latest
              command:
                - /scripts/backup.sh
              env:
                - name: S3_BUCKET
                  value: kecs-backups
                - name: AWS_REGION
                  value: us-east-1
              volumeMounts:
                - name: data
                  mountPath: /data
          volumes:
            - name: data
              persistentVolumeClaim:
                claimName: kecs-data
          restartPolicy: OnFailure
```

## Monitoring and Observability

### Prometheus Configuration

```yaml
apiVersion: v1
kind: ServiceMonitor
metadata:
  name: kecs-metrics
  namespace: kecs-system
spec:
  selector:
    matchLabels:
      app: kecs-control-plane
  endpoints:
    - port: admin
      path: /metrics
      interval: 30s
      scheme: https
      tlsConfig:
        insecureSkipVerify: true
```

### Grafana Dashboard

Import the KECS dashboard (ID: 12345) or create custom dashboards monitoring:

- API request rates and latencies
- Resource utilization (CPU, memory, disk)
- Active clusters, services, and tasks
- Error rates and types
- Database performance metrics

### Alerting Rules

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: kecs-alerts
  namespace: kecs-system
spec:
  groups:
    - name: kecs
      rules:
        - alert: KECSHighErrorRate
          expr: rate(kecs_api_errors_total[5m]) > 0.1
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: High error rate in KECS API
            description: "Error rate is {{ $value }} errors per second"
        
        - alert: KECSHighMemoryUsage
          expr: container_memory_usage_bytes{pod=~"kecs-.*"} / container_spec_memory_limit_bytes > 0.9
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: High memory usage in KECS pod
            description: "Memory usage is above 90% for pod {{ $labels.pod }}"
```

## Performance Tuning

### Database Optimization

```sql
-- Optimize DuckDB for production
PRAGMA memory_limit='8GB';
PRAGMA threads=8;
PRAGMA checkpoint_threshold='2GB';

-- Create indexes for common queries
CREATE INDEX idx_tasks_cluster_status ON tasks(cluster_arn, status);
CREATE INDEX idx_services_cluster_active ON services(cluster_arn) WHERE status = 'ACTIVE';

-- Analyze tables periodically
ANALYZE tasks;
ANALYZE services;
ANALYZE clusters;
```

### Kubernetes Resource Tuning

```yaml
resources:
  requests:
    memory: "2Gi"
    cpu: "1000m"
  limits:
    memory: "8Gi"
    cpu: "4000m"

# Pod Disruption Budget
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: kecs-pdb
  namespace: kecs-system
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: kecs-control-plane
```

## Disaster Recovery

### Multi-Region Setup

Deploy KECS across multiple regions with data replication:

```yaml
regions:
  primary:
    name: us-east-1
    endpoint: https://kecs-us-east-1.example.com
    database:
      primary: true
  
  secondary:
    - name: us-west-2
      endpoint: https://kecs-us-west-2.example.com
      database:
        replicateFrom: us-east-1
    
    - name: eu-west-1
      endpoint: https://kecs-eu-west-1.example.com
      database:
        replicateFrom: us-east-1
```

### Backup and Restore Procedures

```bash
#!/bin/bash
# backup.sh

# Variables
BACKUP_DIR="/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
S3_BUCKET="kecs-backups"

# Create backup
duckdb /data/kecs.db "EXPORT DATABASE '${BACKUP_DIR}/kecs_${TIMESTAMP}'"

# Compress backup
tar -czf "${BACKUP_DIR}/kecs_${TIMESTAMP}.tar.gz" "${BACKUP_DIR}/kecs_${TIMESTAMP}"

# Upload to S3
aws s3 cp "${BACKUP_DIR}/kecs_${TIMESTAMP}.tar.gz" "s3://${S3_BUCKET}/backups/"

# Clean up old backups
find ${BACKUP_DIR} -name "kecs_*.tar.gz" -mtime +7 -delete
```

## Maintenance

### Rolling Updates

```bash
# Update KECS deployment
kubectl set image deployment/kecs-control-plane \
  kecs=ghcr.io/nandemo-ya/kecs:v1.1.0 \
  -n kecs-system

# Monitor rollout
kubectl rollout status deployment/kecs-control-plane -n kecs-system
```

### Database Maintenance

```bash
# Vacuum database
kubectl exec -n kecs-system deployment/kecs-control-plane -- \
  duckdb /data/kecs.db "VACUUM; ANALYZE;"

# Check database integrity
kubectl exec -n kecs-system deployment/kecs-control-plane -- \
  duckdb /data/kecs.db "PRAGMA integrity_check;"
```

## Security Hardening

1. **Enable audit logging**
2. **Implement rate limiting**
3. **Use Pod Security Standards**
4. **Enable encryption at rest**
5. **Regular security scanning**
6. **Implement Zero Trust networking**

## Compliance

For compliance requirements:

1. **Data Residency**: Configure storage locations
2. **Audit Trails**: Enable comprehensive logging
3. **Access Control**: Implement fine-grained RBAC
4. **Encryption**: TLS for transit, encryption for storage
5. **Retention Policies**: Configure data retention

## Next Steps

- [Configuration Guide](./configuration) - Detailed configuration options
- [Monitoring Guide](/guides/monitoring) - Set up comprehensive monitoring
- [Security Best Practices](/guides/security) - Security hardening guide