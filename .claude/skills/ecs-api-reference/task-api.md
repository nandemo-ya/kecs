# ECS Task API Specifications

## RunTask

### Required Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| taskDefinition | String | family:revision or ARN (uses latest ACTIVE if not specified) |

### Key Optional Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| cluster | String | Cluster name/ARN |
| count | Integer | Number of tasks to start (max 10) |
| launchType | String | EC2/FARGATE/EXTERNAL/MANAGED_INSTANCES |
| capacityProviderStrategy | Array | Max 20 (mutually exclusive with launchType) |
| networkConfiguration | Object | Required for awsvpc |
| overrides | Object | Container setting overrides |
| placementConstraints | Array | Placement constraints (max 10) |
| placementStrategy | Array | Placement strategies (max 5) |
| tags | Array | Tags (max 50) |
| enableExecuteCommand | Boolean | Enable Execute Command |
| propagateTags | String | TASK_DEFINITION/SERVICE/NONE |
| clientToken | String | Idempotency ID (max 64 chars) |

### Overrides Structure
```json
{
  "overrides": {
    "containerOverrides": [{
      "name": "container-name",
      "command": ["string"],
      "environment": [{"name": "ENV", "value": "val"}],
      "environmentFiles": [{"type": "s3", "value": "arn:..."}],
      "cpu": number,
      "memory": number,
      "memoryReservation": number,
      "resourceRequirements": [{
        "type": "GPU|InferenceAccelerator",
        "value": "string"
      }]
    }],
    "cpu": "string",
    "memory": "string",
    "taskRoleArn": "string",
    "executionRoleArn": "string",
    "ephemeralStorage": {"sizeInGiB": number},
    "inferenceAcceleratorOverrides": [...]
  }
}
```

### Response
```json
{
  "tasks": [{
    "taskArn": "string",
    "taskDefinitionArn": "string",
    "clusterArn": "string",
    "desiredStatus": "RUNNING|PENDING|STOPPED",
    "lastStatus": "string",
    "containers": [{
      "containerArn": "string",
      "name": "string",
      "lastStatus": "string"
    }],
    "createdAt": number,
    "launchType": "string",
    "platformVersion": "string",
    "startedAt": number,
    "stoppedAt": number,
    "tags": [...]
  }],
  "failures": [{
    "arn": "string",
    "reason": "string",
    "detail": "string"
  }]
}
```

### Specific Errors
- BlockedException (AWS account blocked)
- ConflictException (another RunTask with same clientToken in progress)

### Important Notes
1. **Eventual Consistency Model**: Results may not be immediately reflected
   - Solution: Check state with DescribeTasks (exponential backoff, max 5 minutes)
2. **capacityProviderStrategy vs launchType**: Mutually exclusive, cannot specify both
3. **Maximum 10 tasks per single call**

---

## StopTask

### Request
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| task | String | Yes | Task ARN |
| cluster | String | No | Cluster name/ARN |
| reason | String | No | Stop reason message |

### Response
```json
{
  "task": {
    "taskArn": "string",
    "desiredStatus": "STOPPED",
    "lastStatus": "string",
    "stopCode": "UserInitiated|...",
    "stoppedReason": "string",
    "stoppingAt": number,
    ...
  }
}
```

### Signal Processing
1. Issues docker stop equivalent
2. Sends SIGTERM to container
3. 30 second timeout
4. SIGKILL if not terminated within 30 seconds
- Timeout setting: ECS_CONTAINER_STOP_TIMEOUT
- Windows containers: CTRL_SHUTDOWN_EVENT

### Important Notes
- All tags are deleted when task stops

---

## DescribeTasks

### Request
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | No | Cluster name/ARN |
| tasks | Array[String] | Yes | Max 100 task IDs/ARNs |
| include | Array[String] | No | TAGS |

### Task Object Key Fields
| Field | Type | Description |
|-------|------|-------------|
| taskArn | String | Task ARN |
| clusterArn | String | Cluster ARN |
| taskDefinitionArn | String | Task definition ARN |
| containers | Array | Container information |
| lastStatus | String | Last status |
| desiredStatus | String | Desired state |
| launchType | String | Launch type |
| platformVersion | String | Platform version |
| platformFamily | String | Platform family |
| cpu | String | CPU allocation |
| memory | String | Memory allocation |
| createdAt | Number | Creation time |
| startedAt | Number | Start time |
| stoppedAt | Number | Stop time |
| stoppedReason | String | Stop reason |
| healthStatus | String | UNKNOWN/HEALTHY/UNHEALTHY |
| attachments | Array | Connection info (ENI, etc.) |
| ephemeralStorage | Object | Ephemeral storage |
| enableExecuteCommand | Boolean | ExecuteCommand enabled |

### lastStatus Values
- PROVISIONING
- PENDING
- ACTIVATING
- RUNNING
- DEACTIVATING
- STOPPING
- DEPROVISIONING
- STOPPED

### Container Object
```json
{
  "containerArn": "string",
  "name": "string",
  "image": "string",
  "imageDigest": "string",
  "runtimeId": "string",
  "lastStatus": "string",
  "exitCode": number,
  "reason": "string",
  "healthStatus": "string",
  "cpu": "string",
  "memory": "string",
  "networkBindings": [...],
  "networkInterfaces": [...],
  "managedAgents": [...]
}
```

### Important Notes
- Stopped tasks are included in response for at least 1 hour
- Tagged tasks from deleted clusters are not returned even with same-named new clusters

---

## ListTasks

### Request
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | No | Cluster name/ARN |
| containerInstance | String | No | Container instance ID/ARN |
| desiredStatus | String | No | RUNNING/PENDING/STOPPED (default: RUNNING) |
| family | String | No | Task definition family name |
| launchType | String | No | EC2/FARGATE/EXTERNAL/MANAGED_INSTANCES |
| maxResults | Integer | No | 1-100 (default 100) |
| nextToken | String | No | Pagination token |
| serviceName | String | No | Service name |
| startedBy | String | No | Task starter (standalone filter only) |

### Response
```json
{
  "taskArns": ["string"],
  "nextToken": "string"
}
```

### Important Notes
- desiredStatus=PENDING can be specified but returns no results
- startedBy cannot be combined with other filters
- Recently stopped tasks may be included

---

## ExecuteCommand

### Request
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | No | Cluster name/ARN |
| command | String | Yes | Command to execute |
| container | String | No | Container name (required for multi-container tasks) |
| interactive | Boolean | Yes | Must be true |
| task | String | Yes | Task ARN/ID |

### Response
```json
{
  "clusterArn": "string",
  "taskArn": "string",
  "containerArn": "string",
  "containerName": "string",
  "interactive": true,
  "session": {
    "sessionId": "string",
    "streamUrl": "wss://...",
    "tokenValue": "string"
  }
}
```

### Specific Errors
- TargetNotConnectedException
  - Invalid IAM permissions
  - SSM agent not installed/running
  - Session Manager VPC endpoint not configured

### Prerequisites
- Task started with enableExecuteCommand=true
- AWS SSM Session Manager integration
