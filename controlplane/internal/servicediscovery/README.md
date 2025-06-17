# Service Discovery Integration

This package provides AWS Cloud Map compatible service discovery for KECS.

## Overview

The service discovery integration allows ECS services to discover and communicate with each other using DNS names, similar to AWS Cloud Map.

## Architecture

### Components

1. **Service Discovery Manager** - Core component that manages namespaces, services, and instances
2. **Kubernetes Integration** - Creates headless services and endpoints for DNS resolution
3. **CoreDNS Configuration** - Provides DNS resolution for service discovery
4. **API Handler** - Implements Cloud Map API operations

### How It Works

1. **Namespace Creation**: Creates a DNS namespace (e.g., `production.local`)
2. **Service Registration**: Registers services within a namespace
3. **Instance Registration**: When tasks start, they register as instances
4. **DNS Resolution**: Services can be accessed via `service-name.namespace-name`

## Usage

### Creating a Namespace

```bash
aws servicediscovery create-private-dns-namespace \
  --name production.local \
  --vpc vpc-12345678
```

### Creating a Service

```bash
aws servicediscovery create-service \
  --name backend \
  --namespace-id ns-xxxxx \
  --dns-config "NamespaceId=ns-xxxxx,DnsRecords=[{Type=A,TTL=60}]"
```

### ECS Service with Service Discovery

```bash
aws ecs create-service \
  --cluster my-cluster \
  --service-name my-service \
  --task-definition my-task:1 \
  --service-registries "registryArn=arn:aws:servicediscovery:region:account:service/srv-xxxxx,containerName=web,containerPort=80"
```

### Discovering Instances

```bash
aws servicediscovery discover-instances \
  --namespace-name production.local \
  --service-name backend
```

## DNS Resolution

Services registered with service discovery can be accessed using DNS:

- `backend.production.local` - Resolves to all healthy instances
- Works with both A records (IPv4) and SRV records (with port information)

## Kubernetes Integration

The service discovery manager creates:

1. **Headless Service**: For each Cloud Map service (prefixed with `sd-`)
2. **Endpoints**: Updated dynamically as instances register/deregister
3. **Labels**: For easy identification and management

## Health Checking

- Instance health status is tracked (HEALTHY, UNHEALTHY, UNKNOWN)
- Only healthy instances are returned in DNS queries
- Health status can be updated via the API

## Limitations

- Currently supports only private DNS namespaces
- Route 53 integration is simulated via Kubernetes Services
- Some advanced Cloud Map features may not be fully implemented