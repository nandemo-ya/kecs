# KECS Examples

This directory contains practical examples demonstrating how to use KECS (Kubernetes-based ECS Compatible Service) for various use cases.

## Overview

Each example is self-contained with:
- `task_def.json` - ECS task definition
- `service_def.json` - ECS service definition (except for batch jobs)
- `ecspresso.yml` - ecspresso configuration for easy deployment
- `README.md` - Detailed documentation including setup, deployment, and testing

## Available Examples

### 1. [single-task-nginx](./single-task-nginx/)
**Basic web server deployment**
- Simple nginx container
- Public network access
- Health checks
- Good starting point for learning KECS

### 2. [multi-container-webapp](./multi-container-webapp/)
**Multi-container application with dependencies**
- Frontend nginx + backend API + sidecar logger
- Container dependencies and startup order
- Shared volumes between containers
- Inter-container communication

### 3. [microservice-with-elb](./microservice-with-elb/)
**Microservice with load balancer integration**
- Application Load Balancer (ALB) setup
- Path-based routing rules
- Target group health checks
- Load distribution across multiple tasks

### 4. [service-with-secrets](./service-with-secrets/)
**Secure secret management**
- AWS Secrets Manager integration
- SSM Parameter Store usage
- Environment variable injection
- No hardcoded credentials

### 5. [batch-job-simple](./batch-job-simple/)
**One-off batch processing tasks**
- Standalone task execution (no service)
- Job scheduling patterns
- Parallel processing examples
- Task chaining with dependencies

## Getting Started

### Prerequisites

1. **KECS Installation**
   ```bash
   # Install KECS
   go install github.com/nandemo-ya/kecs/cmd/controlplane@latest
   
   # Or use Docker
   docker pull ghcr.io/nandemo-ya/kecs:latest
   ```

2. **Start KECS**
   ```bash
   kecs start
   ```

3. **Configure AWS CLI**
   ```bash
   # Point AWS CLI to KECS endpoint
   export AWS_ENDPOINT_URL=http://localhost:8080
   
   # Or use --endpoint-url flag with each command
   aws ecs list-clusters --endpoint-url http://localhost:8080
   ```

4. **Install ecspresso** (optional but recommended)
   ```bash
   brew install kayac/tap/ecspresso
   # Or download from https://github.com/kayac/ecspresso/releases
   ```

### Quick Start

1. Choose an example:
   ```bash
   cd examples/single-task-nginx
   ```

2. Follow the README in that directory for specific setup instructions

3. Deploy using ecspresso:
   ```bash
   ecspresso deploy --config ecspresso.yml
   ```

4. Or deploy using AWS CLI:
   ```bash
   aws ecs register-task-definition --cli-input-json file://task_def.json --endpoint-url http://localhost:8080
   aws ecs create-service --cli-input-json file://service_def.json --endpoint-url http://localhost:8080
   ```

## Common Patterns

### Using ecspresso

ecspresso simplifies ECS deployments with features like:
- Unified configuration management
- Deployment status tracking
- Log streaming
- Service scaling

Basic commands:
```bash
# Deploy service
ecspresso deploy --config ecspresso.yml

# Check status
ecspresso status --config ecspresso.yml

# View logs
ecspresso logs --config ecspresso.yml

# Scale service
ecspresso scale --config ecspresso.yml --tasks 5

# Run one-off task
ecspresso run --config ecspresso.yml
```

### Testing with Kubernetes

Since KECS runs tasks as Kubernetes pods, you can use kubectl for debugging:

```bash
# List pods for a service
kubectl get pods -n default -l app=<service-name>

# View pod details
kubectl describe pod -n default <pod-name>

# Access container logs
kubectl logs -n default <pod-name> -c <container-name>

# Port forward for testing
kubectl port-forward -n default <pod-name> 8080:80

# Exec into container
kubectl exec -it -n default <pod-name> -c <container-name> -- sh
```

### Working with LocalStack

Some examples require additional AWS services. Use LocalStack for local development:

```bash
# Start LocalStack with required services
docker run -d \
  --name localstack \
  -p 4566:4566 \
  -e SERVICES=secretsmanager,ssm,iam,elbv2 \
  localstack/localstack

# Use different endpoints for different services
aws ecs create-cluster --endpoint-url http://localhost:8080      # KECS for ECS
aws secretsmanager create-secret --endpoint-url http://localhost:4566  # LocalStack for Secrets
```

## Advanced Usage

### Environment-Specific Deployments

Use ecspresso with environment variables:
```bash
# Development
ENV=dev ecspresso deploy --config ecspresso.yml

# Production
ENV=prod ecspresso deploy --config ecspresso.yml
```

### CI/CD Integration

See [ci-cd/github-actions-kecs.yml](./ci-cd/github-actions-kecs.yml) for GitHub Actions example.

### Custom Task Definitions

Modify task definitions for your needs:
- Change CPU/memory allocations
- Add environment variables
- Configure volumes and mounts
- Set up sidecars

## Troubleshooting

### Common Issues

1. **Task fails to start**
   - Check IAM roles exist
   - Verify network configuration
   - Review task definition syntax

2. **Cannot access service**
   - Verify security groups
   - Check target group health
   - Use kubectl port-forward for testing

3. **Secrets not loading**
   - Ensure LocalStack is running
   - Check IAM permissions
   - Verify secret ARNs

### Debug Commands

```bash
# Check KECS status
kecs status

# View KECS logs
kecs logs -f

# List all resources
aws ecs list-clusters --endpoint-url http://localhost:8080
aws ecs list-services --cluster default --endpoint-url http://localhost:8080
aws ecs list-tasks --cluster default --endpoint-url http://localhost:8080

# Check Kubernetes resources
kubectl get all -n default
kubectl get events -n default --sort-by='.lastTimestamp'
```

## Contributing

To add a new example:

1. Create a new directory: `examples/your-example-name/`
2. Include all required files:
   - `task_def.json`
   - `service_def.json` (if applicable)
   - `ecspresso.yml`
   - `README.md`
3. Follow the existing examples' structure
4. Test thoroughly with both ecspresso and AWS CLI
5. Document any special requirements or dependencies

## Additional Resources

- [KECS Documentation](https://kecs.io)
- [ECS Best Practices Guide](https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/)
- [ecspresso Documentation](https://github.com/kayac/ecspresso)
- [LocalStack Documentation](https://docs.localstack.cloud/)

## Legacy Examples

The following directories contain older examples that may be migrated or updated:
- `task-definitions/` - Various task definition examples
- `services/` - Service definition examples
- `docker-compose/` - Docker Compose example

These are kept for reference but we recommend using the new structured examples above.