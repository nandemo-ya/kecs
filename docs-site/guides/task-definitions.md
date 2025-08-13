# Task Definitions Guide

Task definitions are blueprints for your applications that specify how containers should run. This guide covers creating and managing task definitions in KECS.

## Understanding Task Definitions

### What is a Task Definition?

A task definition is a JSON document that describes one or more containers that form your application. It specifies:
- Docker images to use
- CPU and memory requirements
- Networking mode
- Logging configuration
- Environment variables
- IAM roles

### Task Definition Families

Task definitions are grouped into families. Each revision of a task definition increments the revision number within the family.

```
webapp:1  → webapp:2  → webapp:3
   ↓          ↓           ↓
 First    Updated     Latest
revision   image      revision
```

## Creating Task Definitions

### Basic Task Definition

```json
{
  "family": "simple-app",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "nginx:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "protocol": "tcp"
        }
      ]
    }
  ]
}
```

### Multi-Container Task Definition

```json
{
  "family": "webapp-with-sidecar",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "containerDefinitions": [
    {
      "name": "webapp",
      "image": "myapp:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080
        }
      ],
      "environment": [
        {
          "name": "LOG_LEVEL",
          "value": "info"
        }
      ],
      "dependsOn": [
        {
          "containerName": "log-router",
          "condition": "START"
        }
      ]
    },
    {
      "name": "log-router",
      "image": "fluentd:latest",
      "essential": true,
      "firelensConfiguration": {
        "type": "fluentd"
      }
    }
  ]
}
```

## Container Definitions

### Essential Properties

```json
{
  "name": "myapp",
  "image": "myregistry/myapp:v1.2.3",
  "essential": true,
  "memory": 512,
  "memoryReservation": 256,
  "cpu": 256
}
```

- **name**: Unique name within the task
- **image**: Docker image to use
- **essential**: If true, task fails if container stops
- **memory**: Hard memory limit (MiB)
- **memoryReservation**: Soft memory limit
- **cpu**: CPU units (1024 = 1 vCPU)

### Port Mappings

```json
{
  "portMappings": [
    {
      "containerPort": 8080,
      "hostPort": 80,
      "protocol": "tcp",
      "name": "web"
    }
  ]
}
```

### Environment Configuration

#### Environment Variables

```json
{
  "environment": [
    {
      "name": "APP_ENV",
      "value": "production"
    },
    {
      "name": "API_URL",
      "value": "https://api.example.com"
    }
  ]
}
```

#### Secrets

```json
{
  "secrets": [
    {
      "name": "DB_PASSWORD",
      "valueFrom": "arn:aws:secretsmanager:region:account:secret:db-password"
    },
    {
      "name": "API_KEY",
      "valueFrom": "arn:aws:ssm:region:account:parameter/api-key"
    }
  ]
}
```

### Health Checks

```json
{
  "healthCheck": {
    "command": ["CMD-SHELL", "curl -f http://localhost/health || exit 1"],
    "interval": 30,
    "timeout": 5,
    "retries": 3,
    "startPeriod": 60
  }
}
```

### Logging Configuration

#### CloudWatch Logs

```json
{
  "logConfiguration": {
    "logDriver": "awslogs",
    "options": {
      "awslogs-group": "/ecs/myapp",
      "awslogs-region": "us-east-1",
      "awslogs-stream-prefix": "webapp"
    }
  }
}
```

#### FireLens

```json
{
  "logConfiguration": {
    "logDriver": "awsfirelens",
    "options": {
      "Name": "cloudwatch",
      "region": "us-east-1",
      "log_group_name": "/ecs/myapp",
      "log_stream_prefix": "firelens/"
    }
  }
}
```

## Advanced Features

### Container Dependencies

```json
{
  "containerDefinitions": [
    {
      "name": "database",
      "image": "postgres:13",
      "essential": true
    },
    {
      "name": "app",
      "image": "myapp:latest",
      "essential": true,
      "dependsOn": [
        {
          "containerName": "database",
          "condition": "HEALTHY"
        }
      ]
    }
  ]
}
```

Dependency conditions:
- **START**: Container has started
- **COMPLETE**: Container has run to completion
- **SUCCESS**: Container exited successfully
- **HEALTHY**: Container is healthy

### Volumes

#### Bind Mounts

```json
{
  "volumes": [
    {
      "name": "app-config",
      "host": {
        "sourcePath": "/etc/myapp"
      }
    }
  ],
  "containerDefinitions": [
    {
      "name": "app",
      "mountPoints": [
        {
          "sourceVolume": "app-config",
          "containerPath": "/config",
          "readOnly": true
        }
      ]
    }
  ]
}
```

#### EFS Volumes

```json
{
  "volumes": [
    {
      "name": "efs-storage",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-12345678",
        "rootDirectory": "/data",
        "transitEncryption": "ENABLED",
        "authorizationConfig": {
          "accessPointId": "fsap-12345678",
          "iam": "ENABLED"
        }
      }
    }
  ]
}
```

### Resource Requirements

```json
{
  "requiresCompatibilities": ["EC2"],
  "placementConstraints": [
    {
      "type": "memberOf",
      "expression": "attribute:ecs.instance-type =~ c5.*"
    }
  ],
  "cpu": "2048",
  "memory": "4096",
  "gpuCount": 1
}
```

### Network Configuration

#### Network Modes

- **awsvpc**: Each task gets its own network interface
- **bridge**: Uses Docker's built-in bridge network
- **host**: Uses the host's network
- **none**: No networking

```json
{
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"]
}
```

### IAM Roles

#### Task Role

Grants containers permissions to AWS services:

```json
{
  "taskRoleArn": "arn:aws:iam::account:role/ecsTaskRole"
}
```

#### Execution Role

Grants ECS permissions to pull images and write logs:

```json
{
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole"
}
```

## Working with Task Definitions

### Register a Task Definition

```bash
# From file
aws ecs register-task-definition \
  --cli-input-json file://task-definition.json \
  --endpoint-url http://localhost:8080

# Inline
aws ecs register-task-definition \
  --family webapp \
  --network-mode awsvpc \
  --requires-compatibilities FARGATE \
  --cpu 256 \
  --memory 512 \
  --container-definitions '[
    {
      "name": "web",
      "image": "nginx:latest",
      "portMappings": [{"containerPort": 80}],
      "essential": true
    }
  ]' \
  --endpoint-url http://localhost:8080
```

### List Task Definitions

```bash
# List families
aws ecs list-task-definition-families \
  --endpoint-url http://localhost:8080

# List revisions
aws ecs list-task-definitions \
  --family-prefix webapp \
  --endpoint-url http://localhost:8080
```

### Describe Task Definition

```bash
# Latest revision
aws ecs describe-task-definition \
  --task-definition webapp \
  --endpoint-url http://localhost:8080

# Specific revision
aws ecs describe-task-definition \
  --task-definition webapp:3 \
  --endpoint-url http://localhost:8080
```

### Deregister Task Definition

```bash
aws ecs deregister-task-definition \
  --task-definition webapp:1 \
  --endpoint-url http://localhost:8080
```

## Best Practices

### 1. Container Images

- Use specific tags, not `latest`
- Keep images small and secure
- Use multi-stage builds
- Scan images for vulnerabilities

### 2. Resource Allocation

- Set both limits and requests
- Leave headroom for spikes
- Monitor actual usage
- Use resource reservations wisely

### 3. Configuration

- Use environment variables for configuration
- Store secrets in Secrets Manager or SSM
- Use parameter store for non-sensitive config
- Version your task definitions

### 4. Logging

- Always configure logging
- Use structured logging
- Set appropriate retention periods
- Consider log aggregation

### 5. Health Checks

- Implement application health endpoints
- Set reasonable timeouts and intervals
- Use startup periods for slow-starting apps
- Monitor health check metrics

## Common Patterns

### Sidecar Pattern

```json
{
  "containerDefinitions": [
    {
      "name": "app",
      "image": "myapp:latest"
    },
    {
      "name": "proxy",
      "image": "envoyproxy/envoy:latest",
      "links": ["app"]
    }
  ]
}
```

### Init Container Pattern

```json
{
  "containerDefinitions": [
    {
      "name": "init",
      "image": "busybox",
      "essential": false,
      "command": ["sh", "-c", "echo 'Initializing...'"]
    },
    {
      "name": "app",
      "image": "myapp:latest",
      "essential": true,
      "dependsOn": [{
        "containerName": "init",
        "condition": "SUCCESS"
      }]
    }
  ]
}
```

### Ambassador Pattern

```json
{
  "containerDefinitions": [
    {
      "name": "app",
      "image": "myapp:latest",
      "environment": [{
        "name": "PROXY_URL",
        "value": "http://localhost:8080"
      }]
    },
    {
      "name": "ambassador",
      "image": "ambassador:latest",
      "portMappings": [{
        "containerPort": 8080
      }]
    }
  ]
}
```

## Troubleshooting

### Task Definition Validation Errors

- Check JSON syntax
- Verify required fields
- Validate CPU/memory combinations
- Ensure image accessibility

### Container Start Failures

- Check image pull permissions
- Verify environment variables
- Review health check commands
- Check volume mount paths

### Performance Issues

- Monitor resource utilization
- Check for memory leaks
- Review CPU throttling
- Optimize container startup

For more troubleshooting help, see our [Troubleshooting Guide](/guides/troubleshooting).