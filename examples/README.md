# KECS Examples

This directory contains practical examples demonstrating how to use KECS (Kubernetes-based ECS Compatible Service) for various use cases.

## Overview

Each example is self-contained with:
- `task_def.json` - ECS task definition
- `service_def.json` - ECS service definition (except for batch jobs)
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

### 6. [service-to-service-communication](./service-to-service-communication/)
**Inter-service communication with Service Discovery**
- Frontend and backend service architecture
- DNS-based service discovery (AWS Cloud Map compatible)
- Automatic instance registration and health checking
- Cross-service API calls using DNS names
- Load balancing across multiple instances

### 7. [service-discovery-route53](./service-discovery-route53/)
**Service Discovery with Route53 Integration**
- LocalStack Route53 integration (OSS compatible)
- Cross-cluster service communication
- Private DNS namespaces
- A and SRV record management

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
   
   KECS automatically creates the following IAM role in LocalStack on startup:
   - `ecsTaskExecutionRole` - Used by ECS agent to pull images and write logs
   
   This role is created with appropriate policies for local development.
   Note: Task roles should be created separately based on your specific task requirements.

3. **Configure AWS CLI**
   ```bash
   # Point AWS CLI to KECS endpoint
   export AWS_ENDPOINT_URL=http://localhost:5373
   
   # Or use --endpoint-url flag with each command
   aws ecs list-clusters --endpoint-url http://localhost:5373
   ```


### Quick Start

1. Choose an example:
   ```bash
   cd examples/single-task-nginx
   ```

2. Follow the README in that directory for specific setup instructions

3. Deploy using AWS CLI:
   ```bash
   aws ecs register-task-definition --cli-input-json file://task_def.json --endpoint-url http://localhost:5373
   aws ecs create-service --cli-input-json file://service_def.json --endpoint-url http://localhost:5373
   ```

## Common Patterns

### Using AWS CLI

Deploy and manage services using AWS CLI commands:

```bash
# Deploy service
aws ecs register-task-definition --cli-input-json file://task_def.json --endpoint-url http://localhost:5373
aws ecs create-service --cli-input-json file://service_def.json --endpoint-url http://localhost:5373

# Check service status
aws ecs describe-services --cluster default --services <service-name> --endpoint-url http://localhost:5373

# View task logs (via kubectl)
kubectl logs -n default <pod-name> -c <container-name>

# Scale service
aws ecs update-service --cluster default --service <service-name> --desired-count 5 --endpoint-url http://localhost:5373

# Run one-off task
aws ecs run-task --cluster default --task-definition <task-def> --endpoint-url http://localhost:5373
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

### Working with KECS

KECS automatically includes LocalStack support when you create an instance, providing integrated support for additional AWS services like Secrets Manager, SSM Parameter Store, IAM, and ELBv2.

```bash
# All AWS services are available through KECS endpoint
aws ecs create-cluster --endpoint-url http://localhost:5373
aws secretsmanager create-secret --endpoint-url http://localhost:5373
aws ssm put-parameter --endpoint-url http://localhost:5373
```

## Advanced Usage

### Environment-Specific Deployments

Use environment variables with your deployment scripts:
```bash
# Development
ENV=dev aws ecs update-service --cluster default --service my-service --task-definition my-task:dev --endpoint-url http://localhost:5373

# Production
ENV=prod aws ecs update-service --cluster default --service my-service --task-definition my-task:prod --endpoint-url http://localhost:5373
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
   - The `ecsTaskExecutionRole` is automatically created by KECS
   - Check if custom task roles are properly created (if used)
   - Verify network configuration
   - Review task definition syntax

2. **Cannot access service**
   - Verify security groups
   - Check target group health
   - Use kubectl port-forward for testing

3. **Secrets not loading**
   - Check if KECS instance is running (LocalStack is included)
   - Check IAM permissions
   - Verify secret ARNs

### Debug Commands

```bash
# Check KECS status
kecs status

# View KECS logs
kecs logs -f

# List all resources
aws ecs list-clusters --endpoint-url http://localhost:5373
aws ecs list-services --cluster default --endpoint-url http://localhost:5373
aws ecs list-tasks --cluster default --endpoint-url http://localhost:5373

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
   - `README.md`
3. Follow the existing examples' structure
4. Test thoroughly with AWS CLI
5. Document any special requirements or dependencies

## Additional Resources

- [KECS Documentation](https://kecs.io)
- [ECS Best Practices Guide](https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/)
- [LocalStack Documentation](https://docs.localstack.cloud/)

## Legacy Examples

The following directories contain older examples that may be migrated or updated:
- `task-definitions/` - Various task definition examples
- `services/` - Service definition examples
- `docker-compose/` - Docker Compose example

These are kept for reference but we recommend using the new structured examples above.