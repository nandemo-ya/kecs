# ECS Task Definition CLI Commands

## register-task-definition

### Synopsis
```bash
aws ecs register-task-definition \
    --family <value> \
    --container-definitions <value> \
    [--task-role-arn <value>] \
    [--execution-role-arn <value>] \
    [--network-mode <value>] \
    [--volumes <value>] \
    [--placement-constraints <value>] \
    [--requires-compatibilities <value>] \
    [--cpu <value>] \
    [--memory <value>] \
    [--tags <value>] \
    [--pid-mode <value>] \
    [--ipc-mode <value>] \
    [--proxy-configuration <value>] \
    [--inference-accelerators <value>] \
    [--ephemeral-storage <value>] \
    [--runtime-platform <value>] \
    [--enable-fault-injection | --no-enable-fault-injection]
```

### Required Parameters
| Option | Type | Description |
|--------|------|-------------|
| `--family` | string | Family name (max 255 chars: alphanumeric, underscore, hyphen) |
| `--container-definitions` | list | Container definitions in JSON |

### Key Optional Parameters
| Option | Type | Description |
|--------|------|-------------|
| `--task-role-arn` | string | IAM role ARN for containers |
| `--execution-role-arn` | string | IAM role for ECS agent (ECR pull, Secrets Manager) |
| `--network-mode` | string | `none`, `bridge`, `host`, `awsvpc` |
| `--cpu` | string | Task CPU (e.g., `256`, `1024`, `1 vCPU`) |
| `--memory` | string | Task memory (e.g., `512`, `2048`, `1 GB`) |
| `--requires-compatibilities` | list | `EC2`, `FARGATE`, `EXTERNAL`, `MANAGED_INSTANCES` |

### Container Definition Structure
```json
[
  {
    "name": "string",
    "image": "string",
    "cpu": 256,
    "memory": 512,
    "memoryReservation": 256,
    "essential": true,
    "portMappings": [
      {
        "containerPort": 80,
        "hostPort": 80,
        "protocol": "tcp",
        "name": "http",
        "appProtocol": "http"
      }
    ],
    "environment": [
      {"name": "KEY", "value": "value"}
    ],
    "environmentFiles": [
      {"type": "s3", "value": "arn:aws:s3:::bucket/file.env"}
    ],
    "secrets": [
      {"name": "DB_PASSWORD", "valueFrom": "arn:aws:secretsmanager:..."}
    ],
    "mountPoints": [
      {"sourceVolume": "data", "containerPath": "/data", "readOnly": false}
    ],
    "logConfiguration": {
      "logDriver": "awslogs",
      "options": {
        "awslogs-group": "/ecs/my-app",
        "awslogs-region": "us-west-2",
        "awslogs-stream-prefix": "ecs"
      }
    },
    "healthCheck": {
      "command": ["CMD-SHELL", "curl -f http://localhost/ || exit 1"],
      "interval": 30,
      "timeout": 5,
      "retries": 3,
      "startPeriod": 60
    },
    "dependsOn": [
      {"containerName": "init", "condition": "SUCCESS"}
    ],
    "command": ["./start.sh"],
    "entryPoint": ["/bin/sh", "-c"],
    "workingDirectory": "/app",
    "user": "1000:1000",
    "linuxParameters": {
      "initProcessEnabled": true
    }
  }
]
```

### Fargate CPU/Memory Combinations
| CPU (vCPU) | Memory Options |
|------------|----------------|
| 256 (0.25) | 512MB, 1GB, 2GB |
| 512 (0.5) | 1GB-4GB (1GB increments) |
| 1024 (1) | 2GB-8GB (1GB increments) |
| 2048 (2) | 4GB-16GB (1GB increments) |
| 4096 (4) | 8GB-30GB (1GB increments) |
| 8192 (8) | 16GB-60GB (4GB increments) |
| 16384 (16) | 32GB-120GB (8GB increments) |

### Runtime Platform
```json
{
  "cpuArchitecture": "X86_64|ARM64",
  "operatingSystemFamily": "LINUX|WINDOWS_SERVER_2019_FULL|WINDOWS_SERVER_2019_CORE|WINDOWS_SERVER_2022_FULL|WINDOWS_SERVER_2022_CORE"
}
```

### Ephemeral Storage (Fargate)
```json
{
  "sizeInGiB": 30
}
```
- Range: 21-200 GiB
- Requires Linux platform 1.4.0+

### Example: Register with JSON File
```bash
aws ecs register-task-definition --cli-input-json file://task-def.json
```

**task-def.json:**
```json
{
  "family": "my-app",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "executionRoleArn": "arn:aws:iam::123456789012:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "nginx:latest",
      "essential": true,
      "portMappings": [
        {"containerPort": 80, "protocol": "tcp"}
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/my-app",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

### Example: Register Inline
```bash
aws ecs register-task-definition \
    --family my-app \
    --network-mode awsvpc \
    --requires-compatibilities FARGATE \
    --cpu 256 \
    --memory 512 \
    --container-definitions '[{
      "name": "app",
      "image": "nginx:latest",
      "essential": true,
      "portMappings": [{"containerPort": 80}]
    }]'
```

### Output
```json
{
  "taskDefinition": {
    "taskDefinitionArn": "arn:aws:ecs:region:account:task-definition/my-app:1",
    "family": "my-app",
    "revision": 1,
    "status": "ACTIVE",
    "containerDefinitions": [...],
    "cpu": "256",
    "memory": "512",
    "networkMode": "awsvpc",
    "requiresCompatibilities": ["FARGATE"],
    "compatibilities": ["EC2", "FARGATE"],
    "registeredAt": "timestamp",
    "registeredBy": "arn:aws:iam::..."
  }
}
```

---

## deregister-task-definition

### Synopsis
```bash
aws ecs deregister-task-definition \
    --task-definition <value>
```

### Options
| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `--task-definition` | string | Yes | family:revision or full ARN. Revision required. |

### Example
```bash
aws ecs deregister-task-definition --task-definition my-app:1
```

### Output
```json
{
  "taskDefinition": {
    "taskDefinitionArn": "string",
    "family": "my-app",
    "revision": 1,
    "status": "INACTIVE",
    ...
  }
}
```

### Important Notes
- Existing tasks/services continue running
- Cannot run new tasks or create new services with INACTIVE definitions
- 10-minute grace period before restrictions take effect
- INACTIVE definitions remain discoverable indefinitely
- Required before `delete-task-definitions`

---

## describe-task-definition

### Synopsis
```bash
aws ecs describe-task-definition \
    --task-definition <value> \
    [--include <value>]
```

### Options
| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `--task-definition` | string | Yes | `family` (latest ACTIVE), `family:revision`, or ARN |
| `--include` | list | No | `TAGS` |

### Example
```bash
aws ecs describe-task-definition \
    --task-definition my-app:1 \
    --include TAGS
```

### Output
```json
{
  "taskDefinition": {
    "taskDefinitionArn": "string",
    "family": "string",
    "revision": 1,
    "status": "ACTIVE",
    "containerDefinitions": [...],
    "volumes": [...],
    "cpu": "string",
    "memory": "string",
    "networkMode": "string",
    "requiresCompatibilities": [...],
    "compatibilities": [...],
    "runtimePlatform": {...},
    "registeredAt": "timestamp",
    "registeredBy": "string"
  },
  "tags": []
}
```

### Notes
- INACTIVE definitions can only be described if active tasks/services reference them

---

## list-task-definitions

### Synopsis
```bash
aws ecs list-task-definitions \
    [--family-prefix <value>] \
    [--status <value>] \
    [--sort <value>] \
    [--max-items <value>] \
    [--starting-token <value>]
```

### Options
| Option | Type | Description |
|--------|------|-------------|
| `--family-prefix` | string | Filter by family name |
| `--status` | string | `ACTIVE` (default), `INACTIVE`, `DELETE_IN_PROGRESS` |
| `--sort` | string | `ASC` (default) or `DESC` |
| `--max-items` | integer | Max items (1-100) |

### Sort Behavior
- ASC: Alphabetical by family, ascending by revision (latest last)
- DESC: Reverse alphabetical, descending by revision (latest first)

### Example
```bash
aws ecs list-task-definitions --family-prefix my-app --sort DESC
```

### Output
```json
{
  "taskDefinitionArns": [
    "arn:aws:ecs:region:account:task-definition/my-app:3",
    "arn:aws:ecs:region:account:task-definition/my-app:2",
    "arn:aws:ecs:region:account:task-definition/my-app:1"
  ],
  "nextToken": "string"
}
```

---

## list-task-definition-families

### Synopsis
```bash
aws ecs list-task-definition-families \
    [--family-prefix <value>] \
    [--status <value>] \
    [--max-items <value>] \
    [--starting-token <value>]
```

### Options
| Option | Type | Description |
|--------|------|-------------|
| `--family-prefix` | string | Filter by prefix |
| `--status` | string | `ACTIVE` (default), `INACTIVE`, `ALL` |
| `--max-items` | integer | Max items (1-100) |

### Example
```bash
aws ecs list-task-definition-families --family-prefix my
```

### Output
```json
{
  "families": [
    "my-app",
    "my-worker",
    "my-cronjob"
  ],
  "nextToken": "string"
}
```

---

## delete-task-definitions

### Synopsis
```bash
aws ecs delete-task-definitions \
    --task-definitions <value>
```

### Options
| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `--task-definitions` | list | Yes | Max 10 task definition ARNs |

### Prerequisites
- Task definition must be INACTIVE (deregistered)
- No running tasks or services referencing it

### Example
```bash
aws ecs delete-task-definitions \
    --task-definitions \
      arn:aws:ecs:region:account:task-definition/my-app:1 \
      arn:aws:ecs:region:account:task-definition/my-app:2
```

### Output
```json
{
  "taskDefinitions": [...],
  "failures": [...]
}
```
