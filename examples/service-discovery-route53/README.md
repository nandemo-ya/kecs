# Service Discovery with Route53 Integration

This example demonstrates how to use KECS Service Discovery with Route53 integration for cross-cluster service communication.

## Prerequisites

1. KECS instance running (LocalStack with Route53 is automatically included)

## Setup

### 1. Start KECS with Route53 Integration

```bash
# Set environment variables
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1

# Start KECS (includes Route53 support)
kecs start --instance discovery-demo
```

### 2. Create Service Discovery Namespace

```bash
# Create a private DNS namespace
aws servicediscovery create-private-dns-namespace \
  --name production.local \
  --vpc vpc-default \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### 4. Create Service with Service Discovery

```bash
# Create service discovery service
aws servicediscovery create-service \
  --name backend \
  --namespace-id <namespace-id> \
  --dns-config "NamespaceId=<namespace-id>,DnsRecords=[{Type=A,TTL=60}]" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Create ECS service with service registry
aws ecs create-service \
  --cluster default \
  --service-name backend-service \
  --task-definition backend:1 \
  --service-registries "registryArn=arn:aws:servicediscovery:us-east-1:000000000000:service/<service-id>" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### 5. Discover Services

```bash
# Discover instances
aws servicediscovery discover-instances \
  --namespace-name production.local \
  --service-name backend \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

## Cross-Cluster Communication

With Route53 integration, services in different KECS instances can discover each other:

### Instance 1 (Port 5373)
```bash
kecs start --instance cluster1 --api-port 5373
export AWS_ENDPOINT_URL=http://localhost:5373

# Create namespace and service
aws servicediscovery create-private-dns-namespace --name shared.local --vpc vpc-default --region us-east-1
# ... create services ...
```

### Instance 2 (Port 8090)
```bash
kecs start --instance cluster2 --api-port 8090
export AWS_ENDPOINT_URL=http://localhost:8090

# Services can discover cluster1 services via Route53
aws servicediscovery discover-instances \
  --namespace-name shared.local \
  --service-name backend \
  --region us-east-1
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
| `AWS_ENDPOINT_URL` | KECS endpoint URL | http://localhost:5373 |
| `AWS_ACCESS_KEY_ID` | AWS access key | test |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key | test |
| `AWS_REGION` | AWS region | us-east-1 |

## Architecture

```
┌─────────────────────────────────────────────────┐
│              KECS with Route53                   │
│            (Integrated Service)                  │
└────────────────────┬────────────────────────────┘
                     │
    ┌────────────────┴────────────────┐
    │                                  │
┌───▼──────────────┐     ┌────────────▼─────────┐
│   KECS Cluster 1 │     │   KECS Cluster 2     │
│   Port: 5373     │     │   Port: 8090         │
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

1. Check KECS is running:
   ```bash
   kecs status
   ```

2. Verify environment variables are set
3. Check KECS logs for Route53 initialization messages

### DNS Resolution Fails

1. Check namespace and service are created
2. Verify instances are registered
3. Test with `dig` or `nslookup` commands

### Cross-Cluster Discovery Fails

1. Ensure both clusters are properly configured
2. Verify namespaces are created in both clusters
3. Check Route53 hosted zones:
   ```bash
   aws route53 list-hosted-zones --region us-east-1 --endpoint-url http://localhost:5373
   ```