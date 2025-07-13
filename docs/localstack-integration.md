# LocalStack Integration Guide

KECS provides built-in integration with LocalStack to emulate AWS services locally, enabling comprehensive testing and development without requiring AWS credentials or incurring costs.

> **Note**: As of KECS v1.0, LocalStack is enabled by default when creating clusters. This provides immediate access to essential AWS services like IAM, Systems Manager Parameter Store, and Secrets Manager that are commonly required for ECS workloads.

## Overview

LocalStack integration in KECS provides:
- Local emulation of AWS services (IAM, CloudWatch Logs, SSM, Secrets Manager, ELB)
- Automatic lifecycle management of LocalStack containers
- Transparent AWS SDK configuration for ECS tasks
- Multiple proxy modes for different environments
- Dynamic service management

## Configuration

### Using Configuration File

LocalStack integration is configured in the KECS configuration file:

```yaml
# LocalStack configuration in production.yaml or development.yaml
localstack:
  enabled: true  # Enabled by default since v1.0
  services:
    - iam
    - logs
    - ssm
    - secretsmanager
    - elbv2
  persistence: true
  image: "localstack/localstack:latest"
  namespace: "aws-services"
  port: 4566
  edgePort: 4566
  resources:
    memory: "2Gi"
    cpu: "1000m"
    storageSize: "10Gi"
  debug: false
  dataDir: "/var/lib/localstack"
```

### Using Command Line Options

You can also enable LocalStack using command line flags:

```bash
# Start KECS with LocalStack enabled
kecs server --localstack-enabled --config controlplane/configs/production.yaml

# Or without a config file (uses defaults)
kecs server --localstack-enabled
```

### Configuration Precedence

1. Command line flags override config file settings
2. Config file settings override defaults
3. Default configuration is used if no config is specified

### Server Startup with LocalStack

When starting the KECS server with LocalStack enabled:

```bash
# Using config file
kecs server --config controlplane/configs/development.yaml

# Using command line flag
kecs server --localstack-enabled

# Specify custom ports
kecs server --localstack-enabled --port 8080 --admin-port 8081
```

The server will:
1. Initialize LocalStack manager
2. Create LocalStack deployment in Kubernetes
3. Wait for LocalStack to be healthy
4. Configure AWS proxy routing
5. Start accepting requests

### Proxy Configuration

Configure how ECS tasks connect to LocalStack:

```yaml
# AWS Proxy configuration  
proxy:
  mode: "environment"  # Options: environment, sidecar, disabled
  localstackEndpoint: "http://localstack.aws-services.svc.cluster.local:4566"
  fallbackEnabled: true
  fallbackOrder:
    - sidecar
    - environment
```

## CLI Commands

### Starting LocalStack

```bash
# Start LocalStack with default configuration
kecs localstack start

# Check status
kecs localstack status
```

### Managing Services

```bash
# List available services
kecs localstack services

# Enable additional services
kecs localstack enable s3 dynamodb

# Disable services
kecs localstack disable dynamodb

# Restart with new configuration
kecs localstack restart
```

### Monitoring

```bash
# Get detailed status
kecs localstack status

# Example output:
LocalStack Status:
  Running: true
  Healthy: true
  Endpoint: http://localstack.aws-services.svc.cluster.local:4566
  Uptime: 2h30m

Enabled Services:
  - iam
  - logs
  - ssm
  - secretsmanager
  - elbv2

Service Health:
  SERVICE         HEALTHY  ENDPOINT
  iam             true     http://localstack.aws-services.svc.cluster.local:4566
  logs            true     http://localstack.aws-services.svc.cluster.local:4566
  ssm             true     http://localstack.aws-services.svc.cluster.local:4566
  secretsmanager  true     http://localstack.aws-services.svc.cluster.local:4566
  elbv2           true     http://localstack.aws-services.svc.cluster.local:4566
```

## Proxy Modes

### Environment Variable Mode (Default)

Automatically injects AWS SDK configuration into ECS task containers:

```yaml
# Pod annotations to control proxy behavior
metadata:
  annotations:
    kecs.io/aws-proxy-enabled: "true"
    kecs.io/aws-proxy-mode: "environment"
    kecs.io/localstack-endpoint: "http://custom-localstack:4566"  # Optional custom endpoint
```

Injected environment variables:
- `AWS_ENDPOINT_URL`: LocalStack endpoint for all services
- `AWS_ENDPOINT_URL_*`: Service-specific endpoints
- `AWS_ACCESS_KEY_ID`: Test credentials
- `AWS_SECRET_ACCESS_KEY`: Test credentials
- `AWS_DEFAULT_REGION`: us-east-1

### Sidecar Mode (Future)

Adds a transparent proxy sidecar to intercept AWS API calls:
- No application changes required
- Works with any HTTP client
- Higher resource usage

### Disabled Mode

No automatic configuration - applications must manually configure AWS SDK endpoints.

## Using LocalStack with ECS Tasks

### Example Task Definition

```json
{
  "family": "my-app",
  "taskDefinitionArn": "arn:aws:ecs:us-east-1:123456789012:task-definition/my-app:1",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "my-app:latest",
      "memory": 512,
      "cpu": 256,
      "essential": true,
      "environment": [
        {
          "name": "APP_ENV",
          "value": "development"
        }
      ]
    }
  ]
}
```

When deployed with LocalStack enabled, the container automatically receives AWS SDK configuration.

### Application Code Example

```python
import boto3

# No endpoint configuration needed - handled by environment variables
s3 = boto3.client('s3')
iam = boto3.client('iam')
logs = boto3.client('logs')

# Use AWS services normally
s3.create_bucket(Bucket='my-bucket')
iam.create_role(
    RoleName='my-role',
    AssumeRolePolicyDocument=policy_doc
)
```

## Manual LocalStack Deployment

If you need to deploy LocalStack manually:

```bash
# Apply Kubernetes manifests
kubectl apply -f deployments/kubernetes/localstack/

# Check deployment
kubectl -n aws-services get pods
kubectl -n aws-services get svc
```

## Troubleshooting

### LocalStack Not Starting

1. Check pod status:
   ```bash
   kubectl -n aws-services describe pod -l app=localstack
   ```

2. Check logs:
   ```bash
   kubectl -n aws-services logs -l app=localstack
   ```

3. Verify resources:
   ```bash
   kubectl -n aws-services get pvc
   kubectl -n aws-services get events
   ```

### Connection Issues

1. Verify service endpoint:
   ```bash
   kubectl -n aws-services get svc localstack
   ```

2. Test connectivity from a pod:
   ```bash
   kubectl run test --rm -it --image=curlimages/curl -- \
     curl http://localstack.aws-services.svc.cluster.local:4566/_localstack/health
   ```

3. Check proxy configuration:
   ```bash
   # Check if environment variables are injected
   kubectl describe pod <your-ecs-task-pod>
   ```

### Service Not Available

1. Check enabled services:
   ```bash
   kecs localstack status
   ```

2. Enable required service:
   ```bash
   kecs localstack enable <service-name>
   ```

3. Restart LocalStack:
   ```bash
   kecs localstack restart
   ```

## Advanced Configuration

### Custom Environment Variables

Add custom LocalStack configuration:

```yaml
localstack:
  environment:
    LAMBDA_EXECUTOR: "docker"
    LAMBDA_DOCKER_NETWORK: "bridge"
    SKIP_SSL_CERT_DOWNLOAD: "1"
    HOSTNAME_EXTERNAL: "localstack"
```

### Persistence

LocalStack data is persisted in a PVC by default. To disable:

```yaml
localstack:
  persistence: false
```

### Resource Limits

Adjust resources based on your needs:

```yaml
localstack:
  resources:
    memory: "4Gi"      # Increase for heavy workloads
    cpu: "2000m"       # Increase for better performance
    storageSize: "20Gi" # Increase for more data
```

## Best Practices

1. **Start Small**: Begin with essential services only (IAM, Logs, SSM)
2. **Monitor Resources**: LocalStack can be resource-intensive
3. **Use Persistence**: Enable persistence for development continuity
4. **Test Locally**: Verify LocalStack connectivity before deploying applications
5. **Clean Up**: Stop LocalStack when not in use to free resources

## Disabling LocalStack

While LocalStack is enabled by default, you can disable it in specific scenarios:

### For Unit Tests
```bash
# Set environment variable
export KECS_LOCALSTACK_ENABLED=false

# Or use test configuration
kecs server --config controlplane/configs/test.yaml
```

### For Production without LocalStack
```yaml
# In your custom configuration file
localstack:
  enabled: false
```

### Via Command Line
```bash
# Explicitly disable LocalStack
kecs server --localstack-enabled=false
```

## Limitations

- Not all AWS services are fully implemented in LocalStack
- Some service behaviors may differ from real AWS
- Performance is slower than real AWS services
- Lambda execution requires additional configuration
- Some advanced features may not be supported

## Security Considerations

- LocalStack uses test credentials (`test`/`test`)
- Do not use production data with LocalStack
- LocalStack is for development/testing only
- Network policies can restrict LocalStack access

## Additional Resources

- [LocalStack Documentation](https://docs.localstack.cloud/)
- [AWS SDK Configuration](https://docs.aws.amazon.com/sdkref/latest/guide/feature-ss-endpoints.html)
- [KECS Architecture](./architecture.md)
- [ADR-0012: LocalStack Integration](./adr/records/0012-kecs-localstack-adr.md)