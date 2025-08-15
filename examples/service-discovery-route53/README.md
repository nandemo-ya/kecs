# Service Discovery with Route53 Integration

This example demonstrates how to use KECS Service Discovery with Route53 integration for cross-cluster service communication.

## Prerequisites

1. LocalStack running with Route53 enabled
2. KECS instance running with Route53 integration configured

## Setup

### 1. Start LocalStack

```bash
docker run -d \
  --name localstack \
  -p 4566:4566 \
  -e SERVICES=route53 \
  localstack/localstack
```

### 2. Start KECS with Route53 Integration

```bash
# Set environment variables
export LOCALSTACK_ENDPOINT=http://localhost:4566
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1

# Start KECS
kecs start --name discovery-demo
```

### 3. Create Service Discovery Namespace

```bash
# Create a private DNS namespace
aws servicediscovery create-private-dns-namespace \
  --name production.local \
  --vpc vpc-default \
  --endpoint-url http://localhost:8080
```

### 4. Create Service with Service Discovery

```bash
# Create service discovery service
aws servicediscovery create-service \
  --name backend \
  --namespace-id <namespace-id> \
  --dns-config "NamespaceId=<namespace-id>,DnsRecords=[{Type=A,TTL=60}]" \
  --endpoint-url http://localhost:8080

# Create ECS service with service registry
aws ecs create-service \
  --cluster default \
  --service-name backend-service \
  --task-definition backend:1 \
  --service-registries "registryArn=arn:aws:servicediscovery:us-east-1:000000000000:service/<service-id>" \
  --endpoint-url http://localhost:8080
```

### 5. Discover Services

```bash
# Discover instances
aws servicediscovery discover-instances \
  --namespace-name production.local \
  --service-name backend \
  --endpoint-url http://localhost:8080
```

## Cross-Cluster Communication

With Route53 integration, services in different KECS instances can discover each other:

### Instance 1 (Port 8080)
```bash
kecs start --name cluster1 --api-port 8080
export AWS_ENDPOINT_URL=http://localhost:8080

# Create namespace and service
aws servicediscovery create-private-dns-namespace --name shared.local --vpc vpc-default
# ... create services ...
```

### Instance 2 (Port 8090)
```bash
kecs start --name cluster2 --api-port 8090
export AWS_ENDPOINT_URL=http://localhost:8090

# Services can discover cluster1 services via Route53
aws servicediscovery discover-instances \
  --namespace-name shared.local \
  --service-name backend
```

## DNS Resolution

Services can be accessed using standard DNS:

```bash
# Within containers
curl http://backend.production.local

# With port discovery (SRV records)
dig SRV _http._tcp.backend.production.local
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LOCALSTACK_ENDPOINT` | LocalStack endpoint URL | - |
| `AWS_ENDPOINT_URL` | Alternative endpoint setting | - |
| `AWS_ACCESS_KEY_ID` | AWS access key (test for LocalStack) | - |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key (test for LocalStack) | - |
| `AWS_REGION` | AWS region | us-east-1 |

## Architecture

```
┌─────────────────────────────────────────────────┐
│                 LocalStack                       │
│                 Route53 Service                  │
└────────────────────┬────────────────────────────┘
                     │
    ┌────────────────┴────────────────┐
    │                                  │
┌───▼──────────────┐     ┌────────────▼─────────┐
│   KECS Cluster 1 │     │   KECS Cluster 2     │
│   Port: 8080     │     │   Port: 8090         │
│                  │     │                      │
│  ┌─────────────┐ │     │  ┌─────────────┐    │
│  │  Service A  │ │     │  │  Service B  │    │
│  └─────────────┘ │     │  └─────────────┘    │
│                  │     │                      │
│  Kubernetes DNS  │     │  Kubernetes DNS     │
│  + Route53       │     │  + Route53          │
└──────────────────┘     └────────────────────┘
```

## Troubleshooting

### Route53 Not Working

1. Check LocalStack is running:
   ```bash
   curl http://localhost:4566/_localstack/health
   ```

2. Verify environment variables are set
3. Check KECS logs for Route53 initialization messages

### DNS Resolution Fails

1. Check namespace and service are created
2. Verify instances are registered
3. Test with `dig` or `nslookup` commands

### Cross-Cluster Discovery Fails

1. Ensure both clusters use the same LocalStack instance
2. Verify namespaces are created in both clusters
3. Check Route53 hosted zones:
   ```bash
   aws route53 list-hosted-zones --endpoint-url http://localhost:4566
   ```