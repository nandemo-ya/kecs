# Service Discovery Integration

This package provides AWS Cloud Map compatible service discovery for KECS with Route53 integration.

## Overview

The service discovery integration allows ECS services to discover and communicate with each other using DNS names, similar to AWS Cloud Map. It now includes optional Route53 integration for enhanced DNS resolution, particularly useful with LocalStack.

## Architecture

### Components

1. **Service Discovery Manager** - Core component that manages namespaces, services, and instances
2. **Kubernetes Integration** - Creates headless services and endpoints for DNS resolution
3. **CoreDNS Configuration** - Provides DNS resolution for service discovery
4. **Route53 Integration** - Optional integration with LocalStack Route53 for cross-cluster DNS
5. **DNS Resolver** - Dual DNS resolution strategy with fallback support
6. **API Handler** - Implements Cloud Map API operations

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

## Route53 Integration

### Configuration

Enable Route53 integration by setting environment variables:

```bash
# LocalStack endpoint (required for Route53 integration)
export LOCALSTACK_ENDPOINT=http://localhost:4566
# or
export AWS_ENDPOINT_URL=http://localhost:4566

# AWS credentials (for LocalStack)
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1
```

### Features

- **Automatic Hosted Zone Creation**: Creates Route53 hosted zones for each namespace
- **A Record Management**: Automatically updates A records when instances register/deregister
- **SRV Record Support**: Supports SRV records for service port discovery
- **Dual DNS Resolution**: Falls back between Kubernetes DNS and Route53
- **Cross-Cluster Communication**: Enables service discovery across KECS instances

### DNS Resolution Strategy

1. **Kubernetes DNS** (Primary for internal services)
   - Fast resolution within the same cluster
   - Format: `service.namespace.svc.cluster.local`

2. **Route53** (Primary for cross-cluster)
   - Resolution across different KECS instances
   - Format: `service.namespace.domain`

3. **Standard DNS** (Fallback)
   - Uses system DNS resolver as last resort

## Limitations

- Currently supports only private DNS namespaces
- Route53 integration requires LocalStack or AWS
- Some advanced Cloud Map features may not be fully implemented
- SRV records require client support for port discovery