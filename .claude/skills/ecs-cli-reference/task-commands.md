# ECS Task CLI Commands

## run-task

### Synopsis
```bash
aws ecs run-task \
    --task-definition <value> \
    [--cluster <value>] \
    [--count <value>] \
    [--launch-type <value>] \
    [--capacity-provider-strategy <value>] \
    [--platform-version <value>] \
    [--network-configuration <value>] \
    [--overrides <value>] \
    [--placement-constraints <value>] \
    [--placement-strategy <value>] \
    [--group <value>] \
    [--started-by <value>] \
    [--tags <value>] \
    [--enable-ecs-managed-tags | --no-enable-ecs-managed-tags] \
    [--propagate-tags <value>] \
    [--enable-execute-command | --disable-execute-command] \
    [--reference-id <value>] \
    [--volume-configurations <value>] \
    [--client-token <value>]
```

### Required Parameters
| Option | Type | Description |
|--------|------|-------------|
| `--task-definition` | string | family:revision or ARN. Uses latest ACTIVE if revision omitted |

### Key Optional Parameters
| Option | Type | Description |
|--------|------|-------------|
| `--cluster` | string | Cluster name/ARN. Default: `default` |
| `--count` | integer | Number of tasks (1-10 per call) |
| `--launch-type` | string | EC2, FARGATE, EXTERNAL, MANAGED_INSTANCES |
| `--platform-version` | string | Fargate platform. Default: `LATEST` |
| `--client-token` | string | Idempotency token (max 64 chars) |

### Network Configuration (required for awsvpc)
```json
{
  "awsvpcConfiguration": {
    "subnets": ["subnet-xxx"],
    "securityGroups": ["sg-xxx"],
    "assignPublicIp": "ENABLED|DISABLED"
  }
}
```
- Max 16 subnets, 5 security groups

### Task Overrides
```json
{
  "containerOverrides": [
    {
      "name": "container-name",
      "command": ["string"],
      "environment": [{"name": "ENV", "value": "val"}],
      "environmentFiles": [{"type": "s3", "value": "arn:..."}],
      "cpu": 256,
      "memory": 512,
      "memoryReservation": 256,
      "resourceRequirements": [
        {"type": "GPU", "value": "1"}
      ]
    }
  ],
  "cpu": "1024",
  "memory": "2048",
  "taskRoleArn": "arn:aws:iam::...",
  "executionRoleArn": "arn:aws:iam::...",
  "ephemeralStorage": {"sizeInGiB": 30}
}
```
- Total override character limit: 8192 bytes

### Capacity Provider Strategy
```json
[
  {
    "capacityProvider": "FARGATE",
    "weight": 1,
    "base": 1
  }
]
```
- Mutually exclusive with `--launch-type`
- Max 20 providers per strategy

### Placement Strategy
```json
[
  {"type": "spread", "field": "attribute:ecs.availability-zone"},
  {"type": "binpack", "field": "memory"}
]
```
- Max 5 strategies per task
- Not supported with Fargate

### Example: Run Fargate Task
```bash
aws ecs run-task \
    --cluster MyCluster \
    --task-definition my-app:1 \
    --launch-type FARGATE \
    --count 1 \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}"
```

### Example: Run with Overrides
```bash
aws ecs run-task \
    --cluster MyCluster \
    --task-definition my-app:1 \
    --overrides '{
      "containerOverrides": [{
        "name": "app",
        "command": ["./run-migration.sh"]
      }]
    }'
```

### Output
```json
{
  "tasks": [
    {
      "taskArn": "arn:aws:ecs:region:account:task/cluster/task-id",
      "clusterArn": "string",
      "taskDefinitionArn": "string",
      "lastStatus": "PENDING",
      "desiredStatus": "RUNNING",
      "containers": [
        {
          "containerArn": "string",
          "name": "string",
          "lastStatus": "PENDING"
        }
      ],
      "launchType": "FARGATE",
      "createdAt": "timestamp"
    }
  ],
  "failures": []
}
```

### Important Notes
- **Eventual consistency**: Results may not be immediately reflected. Use DescribeTasks with exponential backoff (max 5 minutes)
- **ConflictException**: Duplicate clientToken with different parameters

---

## stop-task

### Synopsis
```bash
aws ecs stop-task \
    --task <value> \
    [--cluster <value>] \
    [--reason <value>]
```

### Options
| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `--task` | string | Yes | Task ARN |
| `--cluster` | string | No | Cluster name/ARN |
| `--reason` | string | No | Stop reason message |

### Signal Processing
1. Sends SIGTERM to containers
2. 30-second timeout (configurable via `ECS_CONTAINER_STOP_TIMEOUT`)
3. SIGKILL if not terminated within timeout
4. Windows containers: CTRL_SHUTDOWN_EVENT

### Example
```bash
aws ecs stop-task \
    --cluster MyCluster \
    --task arn:aws:ecs:us-west-2:123456789012:task/MyCluster/abc123 \
    --reason "Scaling down"
```

### Output
```json
{
  "task": {
    "taskArn": "string",
    "lastStatus": "STOPPED",
    "desiredStatus": "STOPPED",
    "stopCode": "UserInitiated",
    "stoppedReason": "Scaling down",
    "stoppedAt": "timestamp"
  }
}
```

### Stop Codes
- `UserInitiated`: Stopped by user
- `TaskFailedToStart`: Task failed during startup
- `EssentialContainerExited`: Essential container exited
- `ServiceSchedulerInitiated`: Service scheduler stopped task
- `SpotInterruption`: Spot instance interrupted
- `TerminationNotice`: Instance termination notice

---

## describe-tasks

### Synopsis
```bash
aws ecs describe-tasks \
    --tasks <value> \
    [--cluster <value>] \
    [--include <value>]
```

### Options
| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `--tasks` | list | Yes | Max 100 task IDs/ARNs |
| `--cluster` | string | No | Cluster name/ARN |
| `--include` | list | No | `TAGS` |

### Example
```bash
aws ecs describe-tasks \
    --cluster MyCluster \
    --tasks abc123 def456 \
    --include TAGS
```

### Output Structure
```json
{
  "tasks": [
    {
      "taskArn": "string",
      "clusterArn": "string",
      "taskDefinitionArn": "string",
      "lastStatus": "RUNNING",
      "desiredStatus": "RUNNING",
      "healthStatus": "HEALTHY|UNHEALTHY|UNKNOWN",
      "launchType": "FARGATE",
      "cpu": "256",
      "memory": "512",
      "containers": [
        {
          "containerArn": "string",
          "name": "string",
          "image": "string",
          "lastStatus": "RUNNING",
          "exitCode": 0,
          "healthStatus": "HEALTHY",
          "networkInterfaces": [
            {"privateIpv4Address": "10.0.0.1"}
          ]
        }
      ],
      "attachments": [...],
      "createdAt": "timestamp",
      "startedAt": "timestamp",
      "tags": []
    }
  ],
  "failures": []
}
```

### Task Status Values
- `PROVISIONING`
- `PENDING`
- `ACTIVATING`
- `RUNNING`
- `DEACTIVATING`
- `STOPPING`
- `DEPROVISIONING`
- `STOPPED`

### Notes
- Stopped tasks visible for at least 1 hour
- Tagged tasks from deleted clusters don't appear with same-named new clusters

---

## list-tasks

### Synopsis
```bash
aws ecs list-tasks \
    [--cluster <value>] \
    [--container-instance <value>] \
    [--family <value>] \
    [--started-by <value>] \
    [--service-name <value>] \
    [--desired-status <value>] \
    [--launch-type <value>] \
    [--max-items <value>] \
    [--starting-token <value>]
```

### Options
| Option | Type | Description |
|--------|------|-------------|
| `--cluster` | string | Cluster name/ARN |
| `--container-instance` | string | Filter by container instance |
| `--family` | string | Filter by task definition family |
| `--started-by` | string | Filter by startedBy (must be only filter) |
| `--service-name` | string | Filter by service |
| `--desired-status` | string | RUNNING (default), PENDING, STOPPED |
| `--launch-type` | string | EC2, FARGATE, EXTERNAL |

### Example
```bash
aws ecs list-tasks \
    --cluster MyCluster \
    --service-name MyService \
    --desired-status RUNNING
```

### Output
```json
{
  "taskArns": [
    "arn:aws:ecs:region:account:task/cluster/task-id-1",
    "arn:aws:ecs:region:account:task/cluster/task-id-2"
  ],
  "nextToken": "string"
}
```

### Notes
- `--started-by` cannot be combined with other filters
- `PENDING` filter returns no results
- Recently stopped tasks may appear

---

## execute-command

### Synopsis
```bash
aws ecs execute-command \
    --task <value> \
    --command <value> \
    --interactive \
    [--cluster <value>] \
    [--container <value>]
```

### Required Options
| Option | Type | Description |
|--------|------|-------------|
| `--task` | string | Task ARN or ID |
| `--command` | string | Command to run on container |
| `--interactive` | boolean | Must be true |

### Optional
| Option | Type | Description |
|--------|------|-------------|
| `--cluster` | string | Cluster name/ARN |
| `--container` | string | Container name (required for multi-container tasks) |

### Prerequisites
- Task started with `enableExecuteCommand=true`
- AWS Systems Manager Session Manager plugin installed
- Appropriate IAM permissions

### Example
```bash
aws ecs execute-command \
    --cluster MyCluster \
    --task abc123 \
    --container app \
    --interactive \
    --command "/bin/sh"
```

### Common Errors
- `TargetNotConnectedException`:
  - Invalid IAM permissions
  - SSM agent not installed/running
  - Session Manager VPC endpoint not configured
