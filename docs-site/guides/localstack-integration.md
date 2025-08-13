# LocalStack Integration Guide

KECS integrates with LocalStack to provide a complete local AWS environment. This guide covers setting up and using LocalStack with KECS.

## Overview

LocalStack integration enables:
- Local AWS service emulation (S3, DynamoDB, SQS, etc.)
- IAM role simulation
- CloudWatch logs and metrics
- Secrets Manager and SSM Parameter Store
- Service discovery with Route 53

## Setup

### Starting KECS with LocalStack

```bash
# Start KECS with LocalStack enabled
./bin/kecs server --localstack-enabled

# Or with custom LocalStack configuration
./bin/kecs server \
  --localstack-enabled \
  --localstack-endpoint http://localhost:4566 \
  --localstack-region us-east-1
```

### Configuration File

Create `kecs-config.yaml`:

```yaml
server:
  port: 8080
  adminPort: 8081

localstack:
  enabled: true
  endpoint: http://localhost:4566
  region: us-east-1
  services:
    - s3
    - dynamodb
    - sqs
    - sns
    - secretsmanager
    - ssm
    - iam
    - logs
    - cloudwatch
```

### Docker Compose Setup

```yaml
version: '3.8'
services:
  kecs:
    image: ghcr.io/nandemo-ya/kecs:latest
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      - KECS_LOCALSTACK_ENABLED=true
      - KECS_LOCALSTACK_ENDPOINT=http://localstack:4566
    depends_on:
      - localstack

  localstack:
    image: localstack/localstack:latest
    ports:
      - "4566:4566"
    environment:
      - SERVICES=s3,dynamodb,sqs,sns,secretsmanager,ssm,iam,logs,cloudwatch
      - DEBUG=1
    volumes:
      - ./localstack:/var/lib/localstack
      - /var/run/docker.sock:/var/run/docker.sock
```

## Using AWS Services

### IAM Integration

KECS automatically maps ECS task roles to Kubernetes ServiceAccounts:

```json
{
  "family": "webapp",
  "taskRoleArn": "arn:aws:iam::000000000000:role/webapp-task-role",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "myapp:latest",
      "environment": [
        {
          "name": "AWS_REGION",
          "value": "us-east-1"
        }
      ]
    }
  ]
}
```

The container can now access AWS services using the task role.

### S3 Integration

Access S3 buckets from your containers:

```python
import boto3

# No endpoint_url parameter needed! 
# KECS's transparent proxy automatically routes to LocalStack
s3 = boto3.client('s3')  # Works as-is, no AWS_ENDPOINT_URL required!

# List buckets
buckets = s3.list_buckets()

# Upload file
s3.upload_file('local.txt', 'my-bucket', 'remote.txt')

# Compare with typical LocalStack usage (NOT needed with KECS):
# s3 = boto3.client('s3', endpoint_url='http://localhost:4566')  # Not required!
```

### DynamoDB Integration

Use DynamoDB tables:

```python
import boto3

# Again, no endpoint configuration needed!
dynamodb = boto3.resource('dynamodb')  # Automatically uses LocalStack
table = dynamodb.Table('users')

# Put item
table.put_item(Item={
    'userId': '123',
    'name': 'John Doe',
    'email': 'john@example.com'
})

# Query
response = table.get_item(Key={'userId': '123'})

# Without KECS transparent proxy, you would need:
# dynamodb = boto3.resource('dynamodb', endpoint_url='http://localhost:4566')
```

### Secrets Manager

Store and retrieve secrets:

```bash
# Create secret via AWS CLI
aws secretsmanager create-secret \
  --name prod/db/password \
  --secret-string "mysecretpassword" \
  --endpoint-url http://localhost:4566

# Use in task definition
{
  "containerDefinitions": [
    {
      "name": "app",
      "secrets": [
        {
          "name": "DB_PASSWORD",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:000000000000:secret:prod/db/password"
        }
      ]
    }
  ]
}
```

### SSM Parameter Store

Store configuration parameters:

```bash
# Create parameter
aws ssm put-parameter \
  --name /myapp/database/host \
  --value "db.example.com" \
  --type String \
  --endpoint-url http://localhost:4566

# Use in task definition
{
  "containerDefinitions": [
    {
      "name": "app",
      "secrets": [
        {
          "name": "DB_HOST",
          "valueFrom": "arn:aws:ssm:us-east-1:000000000000:parameter/myapp/database/host"
        }
      ]
    }
  ]
}
```

### CloudWatch Logs

Container logs are automatically sent to CloudWatch:

```json
{
  "containerDefinitions": [
    {
      "name": "app",
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/myapp",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "webapp"
        }
      }
    }
  ]
}
```

View logs:
```bash
aws logs tail /ecs/myapp \
  --follow \
  --endpoint-url http://localhost:4566
```

## Automatic Sidecar Injection (Transparent Proxy)

KECS provides a powerful transparent proxy feature that automatically routes AWS API calls to LocalStack **without requiring AWS_ENDPOINT configuration**. This is a significant advantage over typical LocalStack setups.

### How It Works

1. KECS detects AWS SDK usage in your container
2. Automatically injects a transparent proxy sidecar
3. Routes all AWS API calls to LocalStack transparently
4. **No code changes or AWS_ENDPOINT settings required**

### Key Benefits

- **Zero Configuration**: Your existing AWS applications work without any modifications
- **No AWS_ENDPOINT Required**: Unlike standard LocalStack usage, you don't need to set `endpoint_url` or `AWS_ENDPOINT_URL`
- **Production-Ready Code**: The same code works in both local (with LocalStack) and production (with real AWS) environments
- **Automatic Detection**: KECS intelligently detects when proxy injection is needed

### How the Transparent Proxy Works

The transparent proxy mechanism works by:

1. **iptables Rules**: KECS configures iptables rules in the pod to intercept outbound HTTPS traffic to AWS domains
2. **DNS Resolution**: AWS service domains (e.g., `s3.amazonaws.com`) are resolved normally
3. **Traffic Interception**: The proxy sidecar intercepts connections to AWS endpoints
4. **Request Routing**: Requests are transparently forwarded to LocalStack while preserving all headers and authentication
5. **Response Handling**: LocalStack responses are returned to the application as if they came from AWS

This approach is superior to environment variable injection because:
- Works with any AWS SDK or tool (not just those that respect `AWS_ENDPOINT_URL`)
- No risk of environment variables being ignored or overridden
- Supports dynamic endpoint discovery (e.g., S3 virtual-hosted-style URLs)
- Zero application awareness required

### Manual Configuration

If you prefer to disable automatic injection and configure endpoints manually:
```json
{
  "containerDefinitions": [
    {
      "name": "app",
      "environment": [
        {
          "name": "KECS_DISABLE_PROXY",
          "value": "true"
        },
        {
          "name": "AWS_ENDPOINT_URL",
          "value": "http://localhost:4566"
        }
      ]
    }
  ]
}
```

## Service Discovery

### Private DNS Namespace

Create a Route 53 private hosted zone:

```bash
aws servicediscovery create-private-dns-namespace \
  --name prod.local \
  --vpc vpc-12345 \
  --endpoint-url http://localhost:4566
```

### Register Service

```json
{
  "serviceName": "api",
  "serviceRegistries": [
    {
      "registryArn": "arn:aws:servicediscovery:us-east-1:000000000000:service/srv-12345",
      "containerName": "api",
      "containerPort": 8080
    }
  ]
}
```

### Discover Services

Services can discover each other:
```python
# In your application
api_endpoint = "http://api.prod.local:8080"
```

## Testing with LocalStack

### Unit Tests

```python
import unittest
import boto3
from moto import mock_s3

class TestS3Integration(unittest.TestCase):
    @mock_s3
    def test_upload_file(self):
        # Create bucket
        s3 = boto3.client('s3', endpoint_url='http://localhost:4566')
        s3.create_bucket(Bucket='test-bucket')
        
        # Upload file
        s3.upload_file('test.txt', 'test-bucket', 'uploaded.txt')
        
        # Verify
        objects = s3.list_objects(Bucket='test-bucket')
        assert len(objects['Contents']) == 1
```

### Integration Tests

```bash
# Start LocalStack and KECS
docker-compose up -d

# Run tests
pytest tests/integration/

# Clean up
docker-compose down
```

## Monitoring and Debugging

### LocalStack Dashboard

Access the LocalStack UI:
1. Open http://localhost:8080/localstack/dashboard
2. View:
   - Service health status
   - API call logs
   - Resource listings
   - Configuration

### Debugging AWS SDK Calls

Enable debug logging:

```python
import logging
import boto3

# Enable debug logging
boto3.set_stream_logger('boto3.resources', logging.DEBUG)

# Your code here
s3 = boto3.client('s3')
```

### Viewing Proxy Logs

Check sidecar proxy logs:
```bash
kubectl logs <pod-name> -c localstack-proxy -n <namespace>
```

## Best Practices

### 1. Resource Initialization

Create resources on startup:

```python
# init_resources.py
import boto3

def initialize():
    s3 = boto3.client('s3', endpoint_url='http://localhost:4566')
    
    # Create buckets
    buckets = ['uploads', 'processed', 'archive']
    for bucket in buckets:
        try:
            s3.create_bucket(Bucket=bucket)
        except s3.exceptions.BucketAlreadyExists:
            pass
    
    # Create DynamoDB tables
    dynamodb = boto3.client('dynamodb', endpoint_url='http://localhost:4566')
    # ... create tables

if __name__ == '__main__':
    initialize()
```

### 2. Environment Parity

Keep local and production similar:
- Use same resource names
- Match IAM policies
- Replicate bucket structures
- Use consistent parameter paths

### 3. CI/CD Integration

```yaml
# .github/workflows/test.yml
name: Tests
on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      localstack:
        image: localstack/localstack
        ports:
          - 4566:4566
    
    steps:
      - uses: actions/checkout@v2
      - name: Start KECS
        run: |
          docker run -d \
            -p 8080:8080 \
            -e KECS_LOCALSTACK_ENDPOINT=http://localstack:4566 \
            ghcr.io/nandemo-ya/kecs:latest
      
      - name: Run tests
        run: make test
```

### 4. Cost Optimization

LocalStack Pro features:
- Use free tier for development
- Pro for advanced services (RDS, EKS, etc.)
- Share LocalStack instance across team

## Troubleshooting

### Connection Issues

If containers can't reach LocalStack:

1. Check LocalStack is running:
   ```bash
   curl http://localhost:4566/_localstack/health
   ```

2. Verify network connectivity:
   ```bash
   kubectl exec <pod> -- nslookup localstack-proxy
   ```

3. Check proxy injection:
   ```bash
   kubectl describe pod <pod> -n <namespace>
   ```

### Authentication Errors

For IAM-related issues:

1. Verify task role ARN
2. Check ServiceAccount creation
3. Review IAM policies in LocalStack
4. Enable IAM debug logging

### Service Discovery Issues

If services can't find each other:

1. Check DNS namespace creation
2. Verify service registration
3. Test DNS resolution:
   ```bash
   kubectl exec <pod> -- nslookup api.prod.local
   ```

## Advanced Configuration

### Custom Endpoints

Override specific service endpoints:

```yaml
localstack:
  services:
    s3:
      endpoint: http://custom-s3:4566
    dynamodb:
      endpoint: http://custom-dynamodb:4566
```

### Persistence

Enable LocalStack persistence:

```yaml
services:
  localstack:
    environment:
      - PERSISTENCE=1
    volumes:
      - ./localstack-data:/var/lib/localstack
```

### Multi-Region Support

```yaml
localstack:
  regions:
    - us-east-1
    - eu-west-1
    - ap-northeast-1
```

For more details, see the [LocalStack documentation](https://docs.localstack.cloud/).