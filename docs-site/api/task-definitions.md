# Task Definition API Reference

## Overview

Task definition APIs allow you to register, manage, and query task definitions. A task definition is a blueprint that describes how a container should launch.

## RegisterTaskDefinition

Registers a new task definition from the supplied family and containerDefinitions.

### Request Syntax

```json
{
  "family": "string",
  "taskRoleArn": "string",
  "executionRoleArn": "string",
  "networkMode": "bridge|host|awsvpc|none",
  "containerDefinitions": [
    {
      "name": "string",
      "image": "string",
      "repositoryCredentials": {
        "credentialsParameter": "string"
      },
      "cpu": 123,
      "memory": 123,
      "memoryReservation": 123,
      "links": ["string"],
      "portMappings": [
        {
          "containerPort": 123,
          "hostPort": 123,
          "protocol": "tcp|udp",
          "name": "string",
          "appProtocol": "http|http2|grpc"
        }
      ],
      "essential": true,
      "entryPoint": ["string"],
      "command": ["string"],
      "environment": [
        {
          "name": "string",
          "value": "string"
        }
      ],
      "environmentFiles": [
        {
          "value": "string",
          "type": "s3"
        }
      ],
      "mountPoints": [
        {
          "sourceVolume": "string",
          "containerPath": "string",
          "readOnly": true
        }
      ],
      "volumesFrom": [
        {
          "sourceContainer": "string",
          "readOnly": true
        }
      ],
      "linuxParameters": {
        "capabilities": {
          "add": ["string"],
          "drop": ["string"]
        },
        "devices": [
          {
            "hostPath": "string",
            "containerPath": "string",
            "permissions": ["read", "write", "mknod"]
          }
        ],
        "initProcessEnabled": true,
        "sharedMemorySize": 123,
        "tmpfs": [
          {
            "containerPath": "string",
            "size": 123,
            "mountOptions": ["string"]
          }
        ],
        "maxSwap": 123,
        "swappiness": 123
      },
      "secrets": [
        {
          "name": "string",
          "valueFrom": "string"
        }
      ],
      "dependsOn": [
        {
          "containerName": "string",
          "condition": "START|COMPLETE|SUCCESS|HEALTHY"
        }
      ],
      "startTimeout": 123,
      "stopTimeout": 123,
      "hostname": "string",
      "user": "string",
      "workingDirectory": "string",
      "disableNetworking": true,
      "privileged": true,
      "readonlyRootFilesystem": true,
      "dnsServers": ["string"],
      "dnsSearchDomains": ["string"],
      "extraHosts": [
        {
          "hostname": "string",
          "ipAddress": "string"
        }
      ],
      "dockerSecurityOptions": ["string"],
      "interactive": true,
      "pseudoTerminal": true,
      "dockerLabels": {
        "string": "string"
      },
      "ulimits": [
        {
          "name": "core|cpu|data|fsize|locks|memlock|msgqueue|nice|nofile|nproc|rss|rtprio|rttime|sigpending|stack",
          "softLimit": 123,
          "hardLimit": 123
        }
      ],
      "logConfiguration": {
        "logDriver": "json-file|syslog|journald|gelf|fluentd|awslogs|splunk|awsfirelens",
        "options": {
          "string": "string"
        },
        "secretOptions": [
          {
            "name": "string",
            "valueFrom": "string"
          }
        ]
      },
      "healthCheck": {
        "command": ["string"],
        "interval": 123,
        "timeout": 123,
        "retries": 123,
        "startPeriod": 123
      },
      "systemControls": [
        {
          "namespace": "string",
          "value": "string"
        }
      ],
      "resourceRequirements": [
        {
          "value": "string",
          "type": "GPU|InferenceAccelerator"
        }
      ],
      "firelensConfiguration": {
        "type": "fluentd|fluentbit",
        "options": {
          "string": "string"
        }
      }
    }
  ],
  "volumes": [
    {
      "name": "string",
      "host": {
        "sourcePath": "string"
      },
      "dockerVolumeConfiguration": {
        "scope": "task|shared",
        "autoprovision": true,
        "driver": "string",
        "driverOpts": {
          "string": "string"
        },
        "labels": {
          "string": "string"
        }
      },
      "efsVolumeConfiguration": {
        "fileSystemId": "string",
        "rootDirectory": "string",
        "transitEncryption": "ENABLED|DISABLED",
        "transitEncryptionPort": 123,
        "authorizationConfig": {
          "accessPointId": "string",
          "iam": "ENABLED|DISABLED"
        }
      },
      "fsxWindowsFileServerVolumeConfiguration": {
        "fileSystemId": "string",
        "rootDirectory": "string",
        "authorizationConfig": {
          "credentialsParameter": "string",
          "domain": "string"
        }
      }
    }
  ],
  "placementConstraints": [
    {
      "type": "memberOf",
      "expression": "string"
    }
  ],
  "requiresCompatibilities": ["EC2", "FARGATE", "EXTERNAL"],
  "cpu": "string",
  "memory": "string",
  "tags": [
    {
      "key": "string",
      "value": "string"
    }
  ],
  "pidMode": "host|task",
  "ipcMode": "host|task|none",
  "proxyConfiguration": {
    "type": "APPMESH",
    "containerName": "string",
    "properties": [
      {
        "name": "string",
        "value": "string"
      }
    ]
  },
  "inferenceAccelerators": [
    {
      "deviceName": "string",
      "deviceType": "string"
    }
  ],
  "ephemeralStorage": {
    "sizeInGiB": 123
  },
  "runtimePlatform": {
    "cpuArchitecture": "X86_64|ARM64",
    "operatingSystemFamily": "WINDOWS_SERVER_2019_FULL|WINDOWS_SERVER_2019_CORE|WINDOWS_SERVER_2016_FULL|WINDOWS_SERVER_2004_CORE|WINDOWS_SERVER_2022_CORE|WINDOWS_SERVER_2022_FULL|WINDOWS_SERVER_20H2_CORE|LINUX"
  }
}
```

### Request Parameters

- **family** (string, required): The family name for task definitions of the same family.
- **taskRoleArn** (string): The IAM role that containers can assume for AWS permissions.
- **executionRoleArn** (string): The IAM role that ECS can assume to pull images and write logs.
- **networkMode** (string): The Docker networking mode (default: bridge).
- **containerDefinitions** (array, required): A list of container definitions.
- **volumes** (array): A list of volume definitions.
- **placementConstraints** (array): An array of placement constraint objects.
- **requiresCompatibilities** (array): The task launch types the task definition supports.
- **cpu** (string): The CPU units for the task (256, 512, 1024, 2048, 4096).
- **memory** (string): The memory for the task in MiB.
- **tags** (array): Metadata tags to apply to the task definition.
- **pidMode** (string): The process namespace to use (host or task).
- **ipcMode** (string): The IPC resource namespace to use.
- **proxyConfiguration** (object): Configuration for App Mesh proxy.
- **inferenceAccelerators** (array): Elastic Inference accelerators.
- **ephemeralStorage** (object): The ephemeral storage settings.
- **runtimePlatform** (object): The runtime platform configuration.

### Response Syntax

```json
{
  "taskDefinition": {
    "taskDefinitionArn": "string",
    "containerDefinitions": [],
    "family": "string",
    "taskRoleArn": "string",
    "executionRoleArn": "string",
    "networkMode": "bridge|host|awsvpc|none",
    "revision": 123,
    "volumes": [],
    "status": "ACTIVE|INACTIVE",
    "requiresAttributes": [
      {
        "name": "string",
        "value": "string",
        "targetType": "container-instance",
        "targetId": "string"
      }
    ],
    "placementConstraints": [],
    "compatibilities": ["EC2", "FARGATE", "EXTERNAL"],
    "runtimePlatform": {
      "cpuArchitecture": "X86_64|ARM64",
      "operatingSystemFamily": "LINUX"
    },
    "requiresCompatibilities": ["EC2", "FARGATE", "EXTERNAL"],
    "cpu": "string",
    "memory": "string",
    "inferenceAccelerators": [],
    "pidMode": "host|task",
    "ipcMode": "host|task|none",
    "proxyConfiguration": {},
    "registeredAt": "2024-01-01T00:00:00.000Z",
    "deregisteredAt": "2024-01-01T00:00:00.000Z",
    "registeredBy": "string",
    "ephemeralStorage": {
      "sizeInGiB": 123
    }
  },
  "tags": []
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/RegisterTaskDefinition \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition" \
  -d '{
    "family": "webapp",
    "networkMode": "awsvpc",
    "requiresCompatibilities": ["FARGATE"],
    "cpu": "256",
    "memory": "512",
    "containerDefinitions": [
      {
        "name": "web",
        "image": "nginx:latest",
        "essential": true,
        "portMappings": [
          {
            "containerPort": 80,
            "protocol": "tcp"
          }
        ],
        "environment": [
          {
            "name": "ENV",
            "value": "production"
          }
        ],
        "logConfiguration": {
          "logDriver": "awslogs",
          "options": {
            "awslogs-group": "/ecs/webapp",
            "awslogs-region": "us-east-1",
            "awslogs-stream-prefix": "web"
          }
        },
        "healthCheck": {
          "command": ["CMD-SHELL", "curl -f http://localhost/ || exit 1"],
          "interval": 30,
          "timeout": 5,
          "retries": 3,
          "startPeriod": 60
        }
      }
    ]
  }'
```

## DeregisterTaskDefinition

Deregisters the specified task definition.

### Request Syntax

```json
{
  "taskDefinition": "string"
}
```

### Request Parameters

- **taskDefinition** (string, required): The family and revision (family:revision) or full ARN of the task definition.

### Response Syntax

```json
{
  "taskDefinition": {
    "taskDefinitionArn": "string",
    "containerDefinitions": [],
    "family": "string",
    "taskRoleArn": "string",
    "executionRoleArn": "string",
    "networkMode": "bridge|host|awsvpc|none",
    "revision": 123,
    "volumes": [],
    "status": "ACTIVE|INACTIVE|DELETE_IN_PROGRESS",
    "requiresAttributes": [],
    "placementConstraints": [],
    "compatibilities": ["EC2", "FARGATE", "EXTERNAL"],
    "runtimePlatform": {},
    "requiresCompatibilities": ["EC2", "FARGATE", "EXTERNAL"],
    "cpu": "string",
    "memory": "string",
    "inferenceAccelerators": [],
    "pidMode": "host|task",
    "ipcMode": "host|task|none",
    "proxyConfiguration": {},
    "registeredAt": "2024-01-01T00:00:00.000Z",
    "deregisteredAt": "2024-01-01T00:00:00.000Z",
    "registeredBy": "string",
    "ephemeralStorage": {}
  }
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/DeregisterTaskDefinition \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DeregisterTaskDefinition" \
  -d '{
    "taskDefinition": "webapp:1"
  }'
```

## DescribeTaskDefinition

Describes a task definition.

### Request Syntax

```json
{
  "taskDefinition": "string",
  "include": ["TAGS"]
}
```

### Request Parameters

- **taskDefinition** (string, required): The family for the latest ACTIVE revision, family and revision for a specific revision, or full ARN.
- **include** (array): Specifies whether to see additional information about the task definition.

### Response Syntax

```json
{
  "taskDefinition": {
    "taskDefinitionArn": "string",
    "containerDefinitions": [],
    "family": "string",
    "taskRoleArn": "string",
    "executionRoleArn": "string",
    "networkMode": "bridge|host|awsvpc|none",
    "revision": 123,
    "volumes": [],
    "status": "ACTIVE|INACTIVE|DELETE_IN_PROGRESS",
    "requiresAttributes": [],
    "placementConstraints": [],
    "compatibilities": ["EC2", "FARGATE", "EXTERNAL"],
    "runtimePlatform": {},
    "requiresCompatibilities": ["EC2", "FARGATE", "EXTERNAL"],
    "cpu": "string",
    "memory": "string",
    "inferenceAccelerators": [],
    "pidMode": "host|task",
    "ipcMode": "host|task|none",
    "proxyConfiguration": {},
    "registeredAt": "2024-01-01T00:00:00.000Z",
    "deregisteredAt": "2024-01-01T00:00:00.000Z",
    "registeredBy": "string",
    "ephemeralStorage": {}
  },
  "tags": []
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/DescribeTaskDefinition \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DescribeTaskDefinition" \
  -d '{
    "taskDefinition": "webapp",
    "include": ["TAGS"]
  }'
```

## ListTaskDefinitions

Returns a list of task definitions.

### Request Syntax

```json
{
  "familyPrefix": "string",
  "status": "ACTIVE|INACTIVE|DELETE_IN_PROGRESS",
  "sort": "ASC|DESC",
  "nextToken": "string",
  "maxResults": 123
}
```

### Request Parameters

- **familyPrefix** (string): The full family name to filter the results.
- **status** (string): The task definition status (ACTIVE, INACTIVE, or DELETE_IN_PROGRESS).
- **sort** (string): The order to sort results (ASC or DESC).
- **nextToken** (string): The nextToken value from a previous paginated request.
- **maxResults** (integer): The maximum number of task definition results.

### Response Syntax

```json
{
  "taskDefinitionArns": ["string"],
  "nextToken": "string"
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/ListTaskDefinitions \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListTaskDefinitions" \
  -d '{
    "familyPrefix": "webapp",
    "status": "ACTIVE",
    "sort": "DESC",
    "maxResults": 10
  }'
```

## ListTaskDefinitionFamilies

Returns a list of task definition families.

### Request Syntax

```json
{
  "familyPrefix": "string",
  "status": "ACTIVE|INACTIVE|ALL",
  "nextToken": "string",
  "maxResults": 123
}
```

### Request Parameters

- **familyPrefix** (string): The family prefix to filter results.
- **status** (string): The task definition family status (ACTIVE, INACTIVE, or ALL).
- **nextToken** (string): The nextToken value from a previous paginated request.
- **maxResults** (integer): The maximum number of results.

### Response Syntax

```json
{
  "families": ["string"],
  "nextToken": "string"
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/ListTaskDefinitionFamilies \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListTaskDefinitionFamilies" \
  -d '{
    "status": "ACTIVE",
    "maxResults": 10
  }'
```

## Error Responses

### InvalidParameterException

```json
{
  "__type": "InvalidParameterException",
  "message": "Invalid container definition."
}
```

### ServerException

```json
{
  "__type": "ServerException",
  "message": "Server error occurred."
}
```

### ClientException

```json
{
  "__type": "ClientException",
  "message": "Task definition does not exist."
}
```

## CPU and Memory Combinations

Valid CPU/memory combinations for Fargate:

| CPU | Memory Values (MiB) |
|-----|-------------------|
| 256 | 512, 1024, 2048 |
| 512 | 1024-4096 (increments of 1024) |
| 1024 | 2048-8192 (increments of 1024) |
| 2048 | 4096-16384 (increments of 1024) |
| 4096 | 8192-30720 (increments of 1024) |
| 8192 | 16384-61440 (increments of 4096) |
| 16384 | 32768-122880 (increments of 8192) |

## Best Practices

1. **Versioning**: Always version your task definitions by family and revision.

2. **Resource Limits**: Set appropriate CPU and memory limits:
   ```json
   {
     "cpu": "256",
     "memory": "512",
     "containerDefinitions": [{
       "memory": 512,
       "memoryReservation": 256
     }]
   }
   ```

3. **Health Checks**: Always configure health checks for production workloads.

4. **Secrets Management**: Use Secrets Manager or SSM Parameter Store for sensitive data:
   ```json
   {
     "secrets": [
       {
         "name": "DB_PASSWORD",
         "valueFrom": "arn:aws:secretsmanager:region:account-id:secret:db-password"
       }
     ]
   }
   ```

5. **Logging**: Configure centralized logging:
   ```json
   {
     "logConfiguration": {
       "logDriver": "awslogs",
       "options": {
         "awslogs-group": "/ecs/app",
         "awslogs-region": "us-east-1",
         "awslogs-stream-prefix": "container"
       }
     }
   }
   ```

6. **Container Dependencies**: Use dependencies for multi-container tasks.

7. **Tags**: Tag task definitions for better organization and cost tracking.