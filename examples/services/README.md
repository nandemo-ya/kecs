# ECS Service Definition Examples

This directory contains example ECS service definitions for testing KECS service deployment functionality.

## Service Examples

### 1. `simple-service.json`
A minimal service configuration that demonstrates the basic required fields:
- Service name and cluster
- Task definition reference  
- Desired count
- Basic tags

**Use case**: Quick testing and development

### 2. `nginx-fargate-service.json`
A comprehensive Fargate service configuration featuring:
- FARGATE launch type with network configuration
- Load balancer integration
- Service discovery registration
- Deployment circuit breaker
- ECS managed tags

**Use case**: Production web applications with load balancing

### 3. `webapp-ec2-service.json`
An EC2-based service configuration with:
- EC2 launch type with service role
- Placement constraints and strategies
- Load balancer integration
- Health check grace period
- Task definition tag propagation

**Use case**: Applications requiring specific instance types or host resources

### 4. `redis-daemon-service.json`
A daemon service configuration that includes:
- DAEMON scheduling strategy (one task per container instance)
- Placement constraints for specific instance types
- Execute command enabled for debugging
- No load balancer (internal service)

**Use case**: System services that need to run on every host

### 5. `microservice-with-service-connect.json`
An advanced microservices configuration featuring:
- Service Connect for service mesh functionality
- Capacity provider strategy (mix of FARGATE and FARGATE_SPOT)
- Deployment alarms integration
- Private subnet configuration
- Complex tagging strategy

**Use case**: Production microservices architectures with advanced networking

## Usage with KECS

### Creating a Service

```bash
# Using AWS CLI with KECS endpoint
aws ecs create-service \
  --endpoint-url http://localhost:8080 \
  --cli-input-json file://examples/services/simple-service.json

# Or using curl
curl -X POST http://localhost:8080/v1/createservice \
  -H "Content-Type: application/json" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateService" \
  -d @examples/services/simple-service.json
```

### Listing Services

```bash
aws ecs list-services \
  --endpoint-url http://localhost:8080 \
  --cluster default
```

### Describing Services

```bash
aws ecs describe-services \
  --endpoint-url http://localhost:8080 \
  --cluster default \
  --services simple-service
```

### Updating a Service

```bash
aws ecs update-service \
  --endpoint-url http://localhost:8080 \
  --cluster default \
  --service simple-service \
  --desired-count 3
```

### Deleting a Service

```bash
aws ecs delete-service \
  --endpoint-url http://localhost:8080 \
  --cluster default \
  --service simple-service \
  --force
```

## Prerequisites

Before using these service examples:

1. **Register Task Definitions**: Ensure the referenced task definitions exist
   ```bash
   aws ecs register-task-definition \
     --endpoint-url http://localhost:8080 \
     --cli-input-json file://examples/task-definitions/nginx-fargate.json
   ```

2. **Create Clusters**: Ensure the target clusters exist
   ```bash
   aws ecs create-cluster \
     --endpoint-url http://localhost:8080 \
     --cluster-name default
   ```

3. **Kubernetes Context**: Ensure KECS has access to a Kubernetes cluster (kind, minikube, etc.)

## Testing Workflow

1. Start KECS server
2. Create a cluster
3. Register task definitions
4. Create services using these examples
5. Verify services are deployed as Kubernetes resources
6. Test service updates and deletions

## Notes

- **Load Balancers**: The load balancer ARNs in examples are placeholders
- **Security Groups**: Security group IDs should be updated for your environment  
- **Subnets**: Subnet IDs should match your VPC configuration
- **IAM Roles**: Role ARNs should exist in your AWS account or be mocked in KECS
- **Service Discovery**: Service discovery namespaces should be created beforehand

These examples demonstrate the full range of ECS service configuration options supported by KECS.