# ECS Task Definition API Specifications

## RegisterTaskDefinition

### Required Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| family | String | Task definition family name (max 255 chars: alphanumeric, underscore, hyphen) |
| containerDefinitions | Array | Container definition array |

### Key Optional Parameters
| Parameter | Type | Description | Notes |
|-----------|------|-------------|-------|
| cpu | String | Task CPU units (e.g., `1024` or `1 vCPU`) | Required for Fargate |
| memory | String | Memory (MiB) (e.g., `512` or `1GB`) | Required for Fargate |
| networkMode | String | none/bridge/awsvpc/host | awsvpc required for Fargate |
| taskRoleArn | String | Task IAM Role ARN | Used by containers |
| executionRoleArn | String | Task execution Role ARN | For ECS Agent permissions |
| volumes | Array | Volume definitions (Docker/EFS/FSx, etc.) | - |
| tags | Array | Metadata (max 50) | Key/value pairs |
| ephemeralStorage | Object | Ephemeral storage allocation | Fargate Linux 1.4.0+ |
| requiresCompatibilities | Array | Launch type validation: EC2/FARGATE/EXTERNAL/MANAGED_INSTANCES | - |
| runtimePlatform | Object | OS/CPU architecture | cpuArchitecture/operatingSystemFamily |
| pidMode | String | Process namespace: host/task | Not supported on Windows/Fargate Windows |
| ipcMode | String | IPC namespace: host/task/none | Not supported on Windows/Fargate |
| proxyConfiguration | Object | App Mesh proxy settings | - |
| inferenceAccelerators | Array | Elastic Inference accelerators | - |
| placementConstraints | Array | Placement constraints (max 10) | - |
| enableFaultInjection | Boolean | Enable fault injection (default: false) | - |

### containerDefinitions Parameter Details
```json
{
  "name": "string",           // Required: Container name
  "image": "string",          // Required: Image URI
  "cpu": number,              // CPU units
  "memory": number,           // Memory (MiB)
  "memoryReservation": number,// Soft memory limit
  "essential": boolean,       // Default: true
  "portMappings": [
    {
      "containerPort": number,
      "hostPort": number,
      "protocol": "tcp|udp",
      "appProtocol": "string",
      "name": "string",
      "containerPortRange": "string"
    }
  ],
  "environment": [
    {
      "name": "string",
      "value": "string"
    }
  ],
  "environmentFiles": [
    {
      "type": "s3",
      "value": "arn:aws:s3:::..."
    }
  ],
  "command": ["string"],      // CMD instruction
  "entryPoint": ["string"],   // ENTRYPOINT
  "workingDirectory": "string",
  "user": "string",           // Execution user
  "logConfiguration": {
    "logDriver": "awslogs|splunk|awsfirelens",
    "options": {},
    "secretOptions": []       // From Secrets Manager
  },
  "healthCheck": {
    "command": ["CMD-SHELL", "curl -f http://localhost/"],
    "interval": 30,           // seconds
    "timeout": 5,             // seconds
    "retries": 3,
    "startPeriod": 0          // seconds
  },
  "mountPoints": [
    {
      "sourceVolume": "string",
      "containerPath": "string",
      "readOnly": boolean
    }
  ],
  "volumesFrom": [
    {
      "sourceContainer": "string",
      "readOnly": boolean
    }
  ],
  "linuxParameters": {
    "capabilities": {
      "add": ["string"],
      "drop": ["string"]
    },
    "devices": [],
    "initProcessEnabled": boolean,
    "maxSwap": number,
    "sharedMemorySize": number,
    "swappiness": number,
    "tmpfs": []
  },
  "privileged": boolean,
  "readonlyRootFilesystem": boolean,
  "interactive": boolean,
  "pseudoTerminal": boolean,
  "restartPolicy": {
    "enabled": boolean,
    "restartAttemptPeriod": number,
    "ignoredExitCodes": [number]
  },
  "secrets": [
    {
      "name": "string",
      "valueFrom": "arn:aws:secretsmanager:..."
    }
  ],
  "dependsOn": [
    {
      "containerName": "string",
      "condition": "START|COMPLETE|SUCCESS|HEALTHY"
    }
  ]
}
```

### Fargate Memory/CPU Combinations
```
CPU 0.25 vCPU (256):  512MB, 1GB, 2GB
CPU 0.5 vCPU (512):   1GB-4GB (1GB increments)
CPU 1 vCPU (1024):    2GB-8GB (1GB increments)
CPU 2 vCPU (2048):    4GB-16GB (1GB increments)
CPU 4 vCPU (4096):    8GB-30GB (1GB increments)
CPU 8 vCPU (8192):    16GB-60GB (4GB increments) *Linux 1.4.0+
CPU 16 vCPU (16384):  32GB-120GB (8GB increments) *Linux 1.4.0+
```

### Response Structure
```json
{
  "taskDefinition": {
    "family": "string",
    "taskDefinitionArn": "arn:aws:ecs:region:account:task-definition/family:revision",
    "revision": number,
    "status": "ACTIVE|INACTIVE",
    "registeredAt": number,           // Unix timestamp
    "deregisteredAt": number,
    "registeredBy": "string",
    "containerDefinitions": [...],
    "cpu": "string",
    "memory": "string",
    "networkMode": "string",
    "taskRoleArn": "string",
    "executionRoleArn": "string",
    "requiresAttributes": [
      {
        "name": "com.amazonaws.ecs.capability.docker-remote-api.1.18",
        "targetId": "string",
        "targetType": "container-instance",
        "value": "string"
      }
    ],
    "requiresCompatibilities": ["EC2"|"FARGATE"|"EXTERNAL"],
    "runtimePlatform": {
      "cpuArchitecture": "X86_64|ARM64",
      "operatingSystemFamily": "LINUX|WINDOWS_SERVER_2019_*"
    },
    "volumes": [...],
    "placementConstraints": [...],
    "compatibilities": ["EC2"|"FARGATE"],
    "ephemeralStorage": {
      "sizeInGiB": number
    },
    "enableFaultInjection": boolean
  },
  "tags": [
    {
      "key": "string",
      "value": "string"
    }
  ]
}
```

### Important Constraints

#### Windows Containers
- Task-level CPU/Memory parameters are ignored -> **Must specify at container level**
- pidMode, ipcMode not supported

#### Fargate
- networkMode: **awsvpc required**
- cpu, memory: Required
- requiresCompatibilities: `["FARGATE"]` recommended

#### EC2
- cpu, memory: Optional
- CPU: 128-196608 CPU units (0.125-192 vCPUs)

#### Network Mode `awsvpc`
- NetworkConfiguration required in `RunTask`/`CreateService`

---

## DeregisterTaskDefinition

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| taskDefinition | String | Yes | family:revision or full ARN. Revision is required |

### Response
Returns TaskDefinition object with status: "INACTIVE"

### Allowed Operations
- Existing tasks/services continue running without interruption
- Services referencing INACTIVE task definitions can adjust scale (desiredCount changes)

### Disallowed Operations
- Run new tasks with INACTIVE task definitions
- Create new services with INACTIVE task definitions
- Update services to reference INACTIVE task definitions

### Important Notes
- **10-minute grace period**: Restrictions may not take effect for up to 10 minutes after deregistration
- **Persistence**: INACTIVE task definitions are currently discoverable indefinitely, but this may change
- **Deletion prerequisite**: Deregistration required before deletion (`DeleteTaskDefinitions`)

---

## DescribeTaskDefinition

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| taskDefinition | String | Yes | Identifier: `family` (latest ACTIVE), `family:revision`, or full ARN |
| include | Array[String] | No | `TAGS` to include resource tags |

### Response
```json
{
  "taskDefinition": {...},  // TaskDefinition object
  "tags": [...]             // Only if include: ["TAGS"]
}
```

### Important Notes
- INACTIVE task definitions can only be described if active tasks or services reference them

---

## ListTaskDefinitions

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| familyPrefix | String | No | Filter by family name |
| maxResults | Integer | No | 1-100 (default: max 100) |
| nextToken | String | No | Pagination token |
| sort | String | No | ASC (default) / DESC |
| status | String | No | ACTIVE (default) / INACTIVE / DELETE_IN_PROGRESS |

### Response
```json
{
  "taskDefinitionArns": [
    "arn:aws:ecs:region:account:task-definition/family:1",
    "arn:aws:ecs:region:account:task-definition/family:2"
  ],
  "nextToken": "string"
}
```

### Sort Behavior
- Default (ASC): Alphabetical by family name, ascending by revision (latest last)
- DESC: Reverse alphabetical, descending by revision (latest first)

---

## ListTaskDefinitionFamilies

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| familyPrefix | String | No | Filter by prefix |
| maxResults | Integer | No | 1-100 |
| nextToken | String | No | Pagination token |
| status | String | No | ACTIVE (default) / INACTIVE / ALL |

### Response
```json
{
  "families": ["family1", "family2"],
  "nextToken": "string"
}
```

---

## DeleteTaskDefinitions

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| taskDefinitions | Array[String] | Yes | Max 10 task definition ARNs |

### Prerequisites
- Task definition must be INACTIVE (deregistered)
- No running tasks or services referencing it

### Response
```json
{
  "taskDefinitions": [...],
  "failures": [...]
}
```
