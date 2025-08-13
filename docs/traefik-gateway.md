# Traefik API Gateway

This document describes the Traefik-based API gateway implementation in KECS v2 architecture.

## Overview

In the new architecture, Traefik serves as the unified AWS API gateway that routes requests to appropriate backends:
- ECS API requests → KECS control plane
- ELBv2 API requests → KECS control plane  
- Other AWS service requests → LocalStack

## Architecture

```
┌─────────────────┐
│ AWS CLI/SDK     │
└────────┬────────┘
         │ :4566
┌────────▼────────┐
│ Traefik Gateway │
│  (kecs-system)  │
└────────┬────────┘
         │ Route by X-Amz-Target header
         │
    ┌────┴────┬─────────┐
    │         │         │
┌───▼──┐  ┌──▼──┐  ┌──▼────────┐
│ ECS  │  │ELBv2│  │LocalStack │
│ API  │  │ API │  │ (S3, etc) │
└──────┘  └─────┘  └───────────┘
```

## Routing Rules

Traefik routes requests based on the `X-Amz-Target` header:

1. **ECS APIs**: `X-Amz-Target: AmazonEC2ContainerServiceV20141113.*`
   - Routes to KECS control plane on port 8080
   - Priority: 100

2. **ELBv2 APIs**: `X-Amz-Target: ElasticLoadBalancingV2.*`
   - Routes to KECS control plane on port 8080
   - Priority: 90

3. **Default**: All other requests
   - Routes to LocalStack on port 4566
   - Priority: 10

## Middleware

### aws-headers
Manages AWS-specific headers:
- Preserves original host header for signature validation
- Adds KECS identification headers
- Configures CORS for browser-based clients

### aws-auth
Preserves AWS authentication headers:
- Authorization
- X-Amz-Date
- X-Amz-Security-Token
- X-Amz-Content-SHA256

### rate-limit
Protects against abuse:
- Average: 100 requests/second
- Burst: 200 requests
- Per-host rate limiting

## Deployment

Traefik is deployed as part of the KECS control plane setup:

```bash
# Start KECS with new architecture
kecs start-v2 --name my-cluster

# Traefik is automatically deployed and configured
```

## Testing

Test the routing with the provided script:

```bash
./controlplane/scripts/test-traefik-routing.sh
```

Or manually:

```bash
# Test ECS routing
aws ecs list-clusters --endpoint-url http://localhost:4566

# Test ELBv2 routing  
aws elbv2 describe-load-balancers --endpoint-url http://localhost:4566

# Test LocalStack routing
aws s3 ls --endpoint-url http://localhost:4566
```

## Monitoring

### Dashboard
Access the Traefik dashboard at http://localhost:8080 (when port-forwarded).

### Metrics
Prometheus metrics are exposed on port 8082:
- Request counts by service
- Response times
- Error rates

## Troubleshooting

### Check Traefik logs
```bash
kubectl logs -n kecs-system -l app=traefik
```

### Verify routing rules
```bash
kubectl get ingressroute -n kecs-system
```

### Test connectivity
```bash
kubectl port-forward -n kecs-system svc/traefik 8080:8080
curl http://localhost:8080/api/rawdata
```