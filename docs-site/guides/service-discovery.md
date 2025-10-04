# Service Discovery

Service Discovery enables services to discover and communicate with each other using DNS names instead of hardcoded IP addresses. KECS provides AWS Cloud Map compatible service discovery that integrates seamlessly with ECS services.

## Overview

### What is Service Discovery?

Service Discovery is a mechanism that allows services to find and connect to each other dynamically. When you register a service with service discovery, it becomes accessible via a DNS name that automatically resolves to healthy instances of that service.

**Key Benefits:**
- **Dynamic Service Location**: No need to hardcode IP addresses
- **Automatic Health Checks**: Only healthy instances are returned
- **Load Balancing**: DNS returns multiple IP addresses for load distribution
- **Zero Configuration**: Services automatically register when they start

### How It Works

1. **Create a Namespace**: Define a DNS namespace (e.g., `app.local`)
2. **Register Services**: Services register themselves with their DNS name
3. **Health Monitoring**: Service Discovery monitors instance health
4. **DNS Resolution**: Other services resolve DNS names to healthy instances

```
┌─────────────────┐         DNS Query          ┌──────────────────┐
│  Frontend       │─────────────────────────────▶│  CoreDNS         │
│  Service        │         backend.app.local    │  (Service        │
└─────────────────┘                              │   Discovery)     │
                                                 └──────────────────┘
                                                          │
                                                          ▼
                                                  Returns healthy IPs:
                                                  - 10.42.1.5
                                                  - 10.42.1.6
```

## Architecture

### Components

KECS Service Discovery uses the following components:

1. **CoreDNS**: Kubernetes DNS server with custom configuration
2. **Service Discovery Manager**: Manages namespaces, services, and instances
3. **Kubernetes Services**: ExternalName Services for DNS aliases
4. **Health Checks**: Container health checks for instance health

### DNS Resolution Flow

```
Service Discovery DNS Name (e.g., backend-api.app.local)
           │
           ▼
    CoreDNS Rewrite Plugin
    (Rewrites to Kubernetes Service)
           │
           ▼
    Kubernetes Service (ClusterIP)
    (Returns Pod IPs)
           │
           ▼
    Healthy Pod IPs
```

## Getting Started

### 1. Create a Private DNS Namespace

First, create a namespace for your services:

```bash
aws servicediscovery create-private-dns-namespace \
  --name app.local \
  --vpc vpc-default \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

This creates a DNS namespace where your services will be registered.

### 2. Create a Service Discovery Service

Create a service discovery service for each application:

```bash
# Get the namespace ID
NAMESPACE_ID=$(aws servicediscovery list-namespaces \
  --query "Namespaces[?Name=='app.local'].Id" \
  --output text \
  --region us-east-1 \
  --endpoint-url http://localhost:5373)

# Create service discovery service
aws servicediscovery create-service \
  --name backend-api \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]" \
  --health-check-config "Type=HTTP,ResourcePath=/health,FailureThreshold=3" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### 3. Create ECS Service with Service Registry

When creating an ECS service, include the service registry configuration:

```json
{
  "cluster": "default",
  "serviceName": "backend-api-service",
  "taskDefinition": "backend-api:1",
  "desiredCount": 2,
  "launchType": "FARGATE",
  "serviceRegistries": [
    {
      "registryArn": "arn:aws:servicediscovery:us-east-1:123456789012:service/srv-xxxxx",
      "containerName": "backend",
      "containerPort": 8080
    }
  ],
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345"],
      "securityGroups": ["sg-12345"],
      "assignPublicIp": "ENABLED"
    }
  }
}
```

### 4. Access the Service

Other services can now access your service using DNS:

```bash
# From another container/service
curl http://backend-api.app.local:8080/api/data
```

## DNS Configuration

### DNS Name Format

Service Discovery DNS names follow this format:

```
<service-name>.<namespace-name>:<port>
```

Examples:
- `backend-api.app.local:8080`
- `frontend-web.app.local:3000`
- `database.staging.local:5432`

### CoreDNS Configuration

KECS automatically configures CoreDNS to handle Service Discovery DNS queries. The configuration includes:

1. **DNS Rewrite**: Rewrites Service Discovery DNS to Kubernetes Service DNS
2. **Kubernetes Plugin**: Resolves to actual Pod IPs
3. **Health Check Integration**: Only returns healthy instances

Example CoreDNS configuration:

```
app.local:53 {
    errors
    health {
        lameduck 5s
    }
    ready

    # Rewrite Service Discovery queries to Kubernetes namespace
    rewrite stop {
        name regex (.*)\.app\.local {1}.default-us-east-1.svc.cluster.local
    }

    kubernetes cluster.local in-addr.arpa ip6.arpa {
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
    }

    cache 30
    loop
    reload
    loadbalance
}
```

## Health Checks

### Container Health Checks

Service Discovery integrates with ECS container health checks to determine instance health:

```json
{
  "containerDefinitions": [
    {
      "name": "backend",
      "image": "backend-api:latest",
      "healthCheck": {
        "command": ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ]
}
```

### Service Discovery Health Checks

Configure health checks when creating a service discovery service:

```bash
aws servicediscovery create-service \
  --name api-service \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]" \
  --health-check-config "Type=HTTP,ResourcePath=/health,FailureThreshold=3" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

Health check types:
- **HTTP**: HTTP health check endpoint
- **HTTPS**: HTTPS health check endpoint
- **TCP**: TCP connection check

## Service-to-Service Communication

### Example: Frontend to Backend

This example shows how to set up service-to-service communication using Service Discovery.

#### 1. Backend Service

```go
// backend/main.go
package main

import (
    "encoding/json"
    "net/http"
    "os"
    "time"
)

func main() {
    http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
        hostname, _ := os.Hostname()
        response := map[string]string{
            "hostname":  hostname,
            "timestamp": time.Now().Format(time.RFC3339),
            "message":   "Hello from backend",
        }
        json.NewEncoder(w).Encode(response)
    })

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
    })

    http.ListenAndServe(":8080", nil)
}
```

#### 2. Frontend Service

```go
// frontend/main.go
package main

import (
    "encoding/json"
    "net/http"
    "time"
)

const backendURL = "http://backend-api.app.local:8080"

func main() {
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        client := &http.Client{Timeout: 2 * time.Second}

        // Check backend connectivity
        backendStatus := "unknown"
        resp, err := client.Get(backendURL + "/health")
        if err != nil {
            backendStatus = "unreachable"
        } else {
            defer resp.Body.Close()
            if resp.StatusCode == http.StatusOK {
                backendStatus = "ok"
            } else {
                backendStatus = "unhealthy"
            }
        }

        response := map[string]string{
            "status":   "healthy",
            "frontend": "ok",
            "backend":  backendStatus,
        }
        json.NewEncoder(w).Encode(response)
    })

    http.ListenAndServe(":3000", nil)
}
```

#### 3. Deploy Services

```bash
# Create namespace
aws servicediscovery create-private-dns-namespace \
  --name app.local \
  --vpc vpc-default \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Create service discovery services
NAMESPACE_ID=$(aws servicediscovery list-namespaces \
  --query "Namespaces[?Name=='app.local'].Id" \
  --output text \
  --region us-east-1 \
  --endpoint-url http://localhost:5373)

# Backend service
aws servicediscovery create-service \
  --name backend-api \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]" \
  --health-check-config "Type=HTTP,ResourcePath=/health,FailureThreshold=3" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Frontend service
aws servicediscovery create-service \
  --name frontend-web \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]" \
  --health-check-config "Type=HTTP,ResourcePath=/health,FailureThreshold=3" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Deploy ECS services (with service registries)
# ... see examples/service-to-service-communication
```

## Important Notes

### Docker Image Requirements

For proper DNS resolution with Go's HTTP client, use **Debian-based Docker images with CGO enabled**:

```dockerfile
FROM golang:1.25-bookworm AS builder

WORKDIR /app
COPY main.go .
COPY go.mod .

# IMPORTANT: CGO must be enabled for proper DNS resolution
RUN CGO_ENABLED=1 go build -o app main.go

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates wget && rm -rf /var/lib/apt/lists/*

WORKDIR /root/
COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]
```

**Why Debian + CGO?**
- Alpine Linux (musl libc) has compatibility issues with Go's DNS resolver
- CGO-enabled builds ensure proper DNS resolution for Service Discovery names
- Health checks use `wget` which is included in the Debian image

If you modify Dockerfiles, ensure you:
1. Use Debian or Ubuntu base images (not Alpine)
2. Build with `CGO_ENABLED=1`
3. Include `wget` or `curl` for health checks

## Monitoring and Troubleshooting

### List Namespaces

```bash
aws servicediscovery list-namespaces \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### List Services

```bash
aws servicediscovery list-services \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### Discover Instances

```bash
aws servicediscovery discover-instances \
  --namespace-name app.local \
  --service-name backend-api \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### Test DNS Resolution

From within a pod:

```bash
# Exec into a pod
kubectl exec -it <pod-name> -- sh

# Test DNS resolution
nslookup backend-api.app.local

# Test HTTP connectivity
curl http://backend-api.app.local:8080/health
```

### Check CoreDNS Configuration

```bash
# Get CoreDNS configmap
kubectl get configmap -n kube-system coredns-custom -o yaml

# Check CoreDNS logs
kubectl logs -n kube-system -l k8s-app=kube-dns
```

### Common Issues

#### DNS Resolution Fails

1. **Check CoreDNS is running:**
   ```bash
   kubectl get pods -n kube-system | grep coredns
   ```

2. **Verify namespace exists:**
   ```bash
   aws servicediscovery list-namespaces --region us-east-1 --endpoint-url http://localhost:5373
   ```

3. **Check service is registered:**
   ```bash
   aws servicediscovery list-services --region us-east-1 --endpoint-url http://localhost:5373
   ```

#### Connection Refused

1. **Verify services are running:**
   ```bash
   kubectl get pods -n default-us-east-1
   ```

2. **Check Kubernetes Services:**
   ```bash
   kubectl get svc -n default-us-east-1
   ```

3. **Verify health checks are passing:**
   ```bash
   kubectl describe pod <pod-name> -n default-us-east-1
   ```

#### Wrong DNS Name

Ensure you're using the correct format:
```
<service-name>.<namespace-name>:<port>
```

Example: `backend-api.app.local:8080`

## Advanced Features

### Multiple Namespaces

You can create multiple namespaces for different environments:

```bash
# Production namespace
aws servicediscovery create-private-dns-namespace \
  --name app.local \
  --vpc vpc-default

# Staging namespace
aws servicediscovery create-private-dns-namespace \
  --name staging.local \
  --vpc vpc-default

# Development namespace
aws servicediscovery create-private-dns-namespace \
  --name dev.local \
  --vpc vpc-default
```

### Custom TTL

Configure DNS record TTL:

```bash
aws servicediscovery create-service \
  --name api-service \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=30}]" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

Lower TTL = faster failover, but more DNS queries
Higher TTL = fewer DNS queries, but slower failover

### Service Discovery with Load Balancing

Combine Service Discovery with ELBv2 for external access:

```json
{
  "serviceName": "web-app",
  "taskDefinition": "webapp:1",
  "desiredCount": 3,
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:...",
      "containerName": "web",
      "containerPort": 80
    }
  ],
  "serviceRegistries": [
    {
      "registryArn": "arn:aws:servicediscovery:...",
      "containerName": "web",
      "containerPort": 80
    }
  ]
}
```

This provides:
- **External access** via ALB
- **Internal service-to-service** communication via Service Discovery

## Best Practices

### 1. Namespace Design

- Use separate namespaces for different environments
- Use meaningful namespace names (e.g., `app.local`, `staging.local`)
- Keep namespace names consistent across deployments

### 2. Service Naming

- Use descriptive service names (e.g., `user-api`, `payment-service`)
- Follow a consistent naming convention
- Avoid special characters in service names

### 3. Health Checks

- Always implement `/health` endpoints
- Set appropriate failure thresholds
- Use meaningful health check responses
- Include dependency checks in health endpoints

### 4. DNS TTL

- Use lower TTL (30-60s) for frequently changing services
- Use higher TTL (120-300s) for stable services
- Balance between failover speed and DNS load

### 5. Error Handling

- Implement retry logic with exponential backoff
- Set appropriate timeouts for service calls
- Handle DNS resolution failures gracefully
- Log connection failures for debugging

### 6. Monitoring

- Monitor service registration status
- Track DNS query patterns
- Alert on health check failures
- Monitor service-to-service latency

## Examples

See the complete working example in:
- [examples/service-to-service-communication](https://github.com/nandemo-ya/kecs/tree/main/examples/service-to-service-communication)

This example includes:
- Complete deployment scripts
- Frontend and backend services
- Service Discovery configuration
- Health check implementation
- Testing procedures

## Next Steps

- [ELBv2 Integration](/guides/elbv2-integration) - Combine with load balancing
- [Services Guide](/guides/services) - Learn more about ECS services
- [Task Definitions](/guides/task-definitions) - Configure your tasks
- [Examples](/guides/examples) - See more examples
