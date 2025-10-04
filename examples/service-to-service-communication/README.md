# Service-to-Service Communication Example

This example demonstrates how services deployed on KECS can discover and communicate with each other using ECS Service Discovery (AWS Cloud Map compatible).

## Overview

This demo demonstrates service-to-service communication where:
1. **Frontend Service** → calls → **Backend Service**
2. Communication happens via Service Discovery DNS names
3. No hardcoded IP addresses are used

### Components

- **Backend API Service** (`backend-api`): 
  - REST API running on port 8080
  - Provides `/api/data` endpoint
  - Returns JSON responses with hostname and timestamp
  
- **Frontend Web Service** (`frontend-web`): 
  - Web UI running on port 3000
  - Has a button to call the backend API
  - Discovers backend via DNS: `backend-api.demo.local:8080`
  - Displays the backend response in the UI

- **Service Discovery**: 
  - Automatic DNS-based service discovery
  - Services register themselves when starting
  - Health checks ensure only healthy instances are discoverable

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     User Browser                        │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼ Port Forward (3000)
┌─────────────────────────────────────────────────────────┐
│                    Frontend Service                      │
│                  (frontend-web:3000)                     │
│                                                          │
│  Discovers backend via: backend-api.demo.local    │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼ Service Discovery DNS
┌─────────────────────────────────────────────────────────┐
│                    Backend Service                       │
│                  (backend-api:8080)                      │
│                                                          │
│              Returns JSON API responses                  │
└──────────────────────────────────────────────────────────┘
```

## Prerequisites

1. KECS instance running
2. Docker installed (for building images)
3. AWS CLI configured
4. (Optional) LocalStack for Route53 integration

## Important Notes

### Docker Images

The example services use **Debian-based Docker images with Go 1.25 and CGO enabled**. This is important for proper DNS resolution with Go's HTTP client when using Service Discovery.

**Why Debian + CGO?**
- Alpine Linux (musl libc) has compatibility issues with Go's DNS resolver
- CGO-enabled builds ensure proper DNS resolution for Service Discovery names
- Health checks use `wget` which is included in the Debian image

If you modify the Dockerfiles, ensure you:
1. Use Debian or Ubuntu base images (not Alpine)
2. Build with `CGO_ENABLED=1`
3. Include `wget` for health checks

## Quick Start

### 1. Start KECS

```bash
# Start KECS instance
kecs start --instance service-demo

# Set environment variable
export KECS_ENDPOINT=http://localhost:5373
```

### 2. Build and Push Docker Images

```bash
# Navigate to example directory
cd examples/service-to-service-communication

# Build both service images
docker build -t backend-api:latest ./backend
docker build -t frontend-web:latest ./frontend

# Tag images for k3d registry
docker tag backend-api:latest localhost:5000/kecs-example-backend:latest
docker tag frontend-web:latest localhost:5000/kecs-example-frontend:latest

# Push images to k3d registry
docker push localhost:5000/kecs-example-backend:latest
docker push localhost:5000/kecs-example-frontend:latest
```

### 3. Deploy Services with Service Discovery

```bash
# Deploy both services (this will create namespace, services, and ECS services)
./scripts/deploy.sh
```

This script will:
- Create ECS cluster (if not exists)
- Create Service Discovery namespace `demo.local`
- Register both services in Service Discovery
- Deploy ECS task definitions
- Create ECS services with service registry configuration

### 4. Test Communication

```bash
# Test service discovery and communication
./scripts/test-communication.sh
```

### 5. Access Frontend UI

Since KECS runs tasks as Kubernetes pods, you can access the frontend:

```bash
# Find the frontend pod
kubectl get pods -l app=frontend-web-service

# Port forward to access the UI
kubectl port-forward pod/<frontend-pod-name> 3000:3000

# Open in browser
open http://localhost:3000
```

Click the "Call Backend Service" button to test service-to-service communication.

### What Happens When You Click "Call Backend Service"

1. Frontend receives the button click
2. Frontend resolves `backend-api.demo.local` using Service Discovery
3. DNS returns IP addresses of healthy backend instances
4. Frontend makes HTTP request to `http://backend-api.demo.local:8080/api/data`
5. Backend responds with JSON data
6. Frontend displays the response in the UI

## Service Discovery Details

### DNS Names

Services are accessible via DNS names in the format:
```
<service-name>.<namespace>:<port>
```

- Backend: `backend-api.demo.local:8080`
- Frontend: `frontend-web.demo.local:3000`

### How It Works

1. **Service Registration**: When services start, they automatically register with Service Discovery
2. **Health Checking**: Service Discovery monitors health endpoints (`/health`)
3. **DNS Resolution**: Services can resolve each other using DNS names
4. **Load Balancing**: Multiple instances of a service are automatically load balanced

## Manual Deployment Steps

If you prefer to deploy manually:

### 1. Build and Push Docker Images

```bash
# Build backend
docker build -t backend-api:latest ./backend
docker tag backend-api:latest localhost:5000/kecs-example-backend:latest
docker push localhost:5000/kecs-example-backend:latest

# Build frontend
docker build -t frontend-web:latest ./frontend
docker tag frontend-web:latest localhost:5000/kecs-example-frontend:latest
docker push localhost:5000/kecs-example-frontend:latest
```

### 2. Create ECS Cluster

```bash
aws ecs create-cluster \
  --cluster-name default \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT
```

### 3. Create Service Discovery Namespace

```bash
aws servicediscovery create-private-dns-namespace \
  --name demo.local \
  --vpc vpc-default \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT
```

### 4. Create Service Discovery Services

```bash
# Get namespace ID
NAMESPACE_ID=$(aws servicediscovery list-namespaces \
  --query "Namespaces[?Name=='demo.local'].Id" \
  --output text \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT)

# Create backend discovery service and get ARN
BACKEND_SERVICE_ARN=$(aws servicediscovery create-service \
  --name backend-api \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]" \
  --health-check-config "Type=HTTP,ResourcePath=/health,FailureThreshold=3" \
  --query 'Service.Arn' \
  --output text \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT)

# Create frontend discovery service and get ARN
FRONTEND_SERVICE_ARN=$(aws servicediscovery create-service \
  --name frontend-web \
  --namespace-id $NAMESPACE_ID \
  --dns-config "NamespaceId=$NAMESPACE_ID,DnsRecords=[{Type=A,TTL=60}]" \
  --health-check-config "Type=HTTP,ResourcePath=/health,FailureThreshold=3" \
  --query 'Service.Arn' \
  --output text \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT)

echo "Backend Service Discovery ARN: $BACKEND_SERVICE_ARN"
echo "Frontend Service Discovery ARN: $FRONTEND_SERVICE_ARN"
```

### 5. Register Task Definitions

```bash
aws ecs register-task-definition \
  --cli-input-json file://backend-task-def.json \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT

aws ecs register-task-definition \
  --cli-input-json file://frontend-task-def.json \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT
```

### 6. Create ECS Services

```bash
# Create backend service
aws ecs create-service \
  --cluster default \
  --service-name backend-api-service \
  --task-definition backend-api:1 \
  --desired-count 2 \
  --launch-type FARGATE \
  --service-registries "registryArn=$BACKEND_SERVICE_ARN,containerName=backend,containerPort=8080" \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}" \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT

# Create frontend service
aws ecs create-service \
  --cluster default \
  --service-name frontend-web-service \
  --task-definition frontend-web:1 \
  --desired-count 1 \
  --launch-type FARGATE \
  --service-registries "registryArn=$FRONTEND_SERVICE_ARN,containerName=frontend,containerPort=3000" \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}" \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT
```

## Testing Service Discovery

### Discover Instances

```bash
# Discover backend instances
aws servicediscovery discover-instances \
  --namespace-name demo.local \
  --service-name backend-api \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT

# Discover frontend instances
aws servicediscovery discover-instances \
  --namespace-name demo.local \
  --service-name frontend-web \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT
```

### Test DNS Resolution (from within cluster)

```bash
# Exec into a pod
kubectl exec -it <pod-name> -- sh

# Test DNS resolution
nslookup backend-api.demo.local
curl http://backend-api.demo.local:8080/api/data
```

## Monitoring

### View Service Logs

```bash
# Backend logs
kubectl logs -l app=backend-api -f

# Frontend logs
kubectl logs -l app=frontend-web -f
```

### Check Service Health

```bash
# Backend health
curl http://backend-api.demo.local:8080/health

# Frontend health
curl http://frontend-web.demo.local:3000/health
```

## Cleanup

Remove all resources:

```bash
./scripts/cleanup.sh
```

## Troubleshooting

### Services Can't Find Each Other

1. Check namespace exists:
   ```bash
   aws servicediscovery list-namespaces --region us-east-1 --endpoint-url $KECS_ENDPOINT
   ```

2. Check services are registered:
   ```bash
   aws servicediscovery list-services --region us-east-1 --endpoint-url $KECS_ENDPOINT
   ```

3. Check instances are healthy:
   ```bash
   aws servicediscovery discover-instances \
     --namespace-name demo.local \
     --service-name backend-api \
     --region us-east-1 --endpoint-url $KECS_ENDPOINT
   ```

### Connection Refused

1. Verify services are running:
   ```bash
   aws ecs list-services --cluster default --region us-east-1 --endpoint-url $KECS_ENDPOINT
   ```

2. Check task status:
   ```bash
   aws ecs list-tasks --cluster default --region us-east-1 --endpoint-url $KECS_ENDPOINT
   ```

3. Verify health checks are passing

### DNS Resolution Fails

1. Check CoreDNS is running:
   ```bash
   kubectl get pods -n kube-system | grep coredns
   ```

2. Verify service endpoints exist:
   ```bash
   kubectl get endpoints
   ```

## Advanced Features

### Scaling Services

```bash
# Scale backend to 3 instances
aws ecs update-service \
  --cluster default \
  --service backend-api-service \
  --desired-count 3 \
  --region us-east-1 --endpoint-url $KECS_ENDPOINT
```

### Cross-Cluster Communication

With Route53 integration, services can communicate across different KECS clusters:

1. KECS includes Route53 integration automatically

2. Services in different clusters can use the same namespace for discovery

### Custom Health Checks

Modify the health check configuration in task definitions:

```json
"healthCheck": {
  "command": ["CMD-SHELL", "custom-health-check.sh"],
  "interval": 30,
  "timeout": 5,
  "retries": 3,
  "startPeriod": 60
}
```

## Learn More

- [ECS Service Discovery Documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/service-discovery.html)
- [KECS Service Discovery Implementation](../../controlplane/internal/servicediscovery/README.md)