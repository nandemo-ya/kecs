# ECS Service API Specifications

## CreateService

### Required Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| serviceName | String | Service name (max 255 chars) |

### Key Optional Parameters
| Parameter | Type | Description |
|-----------|------|-------------|
| cluster | String | Cluster name/ARN (default: default) |
| taskDefinition | String | family:revision format or ARN |
| desiredCount | Integer | Number of tasks to run (required for REPLICA) |
| launchType | String | EC2/FARGATE/EXTERNAL/MANAGED_INSTANCES |
| capacityProviderStrategy | Array | Max 20 capacity provider strategies |
| deploymentController | Object | ECS/CODE_DEPLOY/EXTERNAL |
| deploymentConfiguration | Object | Deployment settings |
| schedulingStrategy | String | REPLICA/DAEMON |
| networkConfiguration | Object | VPC settings (required for awsvpc) |
| loadBalancers | Array | Load balancer configuration |
| placementStrategy | Array | Placement strategies (max 5) |
| placementConstraints | Array | Placement constraints (max 10) |
| healthCheckGracePeriodSeconds | Integer | Health check grace period (seconds) |
| enableExecuteCommand | Boolean | Enable Execute Command |
| enableECSManagedTags | Boolean | Enable ECS managed tags |
| propagateTags | String | TASK_DEFINITION/SERVICE/NONE |
| serviceConnectConfiguration | Object | Service Connect settings |
| volumeConfigurations | Array | EBS volume settings (REPLICA only) |
| availabilityZoneRebalancing | String | ENABLED/DISABLED |
| clientToken | String | Idempotency token (max 36 chars) |

### Deployment Strategies

#### ROLLING (Rolling Update)
```json
{
  "deploymentConfiguration": {
    "strategy": "ROLLING",
    "maximumPercent": 200,
    "minimumHealthyPercent": 100
  }
}
```

#### BLUE_GREEN
```json
{
  "deploymentConfiguration": {
    "strategy": "BLUE_GREEN"
  },
  "loadBalancers": [{...}]  // Required
}
```

#### LINEAR
```json
{
  "deploymentConfiguration": {
    "strategy": "LINEAR",
    "linearConfiguration": {
      "stepPercent": 10,
      "stepBakeTimeInMinutes": 5
    }
  }
}
```

#### CANARY
```json
{
  "deploymentConfiguration": {
    "strategy": "CANARY",
    "canaryConfiguration": {
      "canaryPercent": 10,
      "canaryBakeTimeInMinutes": 5
    }
  }
}
```

### Network Configuration (required for awsvpc)
```json
{
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-xxx"],
      "securityGroups": ["sg-xxx"],
      "assignPublicIp": "ENABLED|DISABLED"
    }
  }
}
```

### Load Balancer Configuration
```json
{
  "loadBalancers": [{
    "targetGroupArn": "arn:aws:elasticloadbalancing:...",
    "containerName": "app",
    "containerPort": 80,
    "advancedConfiguration": {
      "alternateTargetGroupArn": "string",
      "productionListenerRule": "string",
      "testListenerRule": "string",
      "roleArn": "string"
    }
  }]
}
```

### Placement Strategies
```json
{
  "placementStrategy": [
    {"type": "spread", "field": "attribute:ecs.availability-zone"},
    {"type": "binpack", "field": "memory"}
  ],
  "placementConstraints": [
    {"type": "distinctInstance"}
  ]
}
```
Types:
- spread: Distribute across instances
- binpack: Maximize resource utilization
- random: Random placement
- distinctInstance: One task per instance
- memberOf: Attribute matching

### Service Connect Configuration
```json
{
  "serviceConnectConfiguration": {
    "enabled": true,
    "namespace": "my-namespace",
    "logConfiguration": {...},
    "services": [{
      "portName": "http",
      "discoveryName": "backend",
      "clientAliases": [{
        "port": 80,
        "dnsName": "backend.internal"
      }],
      "timeout": {
        "idleTimeoutSeconds": 60,
        "perRequestTimeoutSeconds": 5
      },
      "tls": {...}
    }]
  }
}
```

### Response - Service Object
```json
{
  "service": {
    "serviceArn": "string",
    "serviceName": "string",
    "clusterArn": "string",
    "status": "ACTIVE|DRAINING|INACTIVE",
    "desiredCount": number,
    "runningCount": number,
    "pendingCount": number,
    "taskDefinition": "string",
    "launchType": "string",
    "platformFamily": "string",
    "platformVersion": "string",
    "schedulingStrategy": "REPLICA|DAEMON",
    "deploymentController": {"type": "ECS|CODE_DEPLOY|EXTERNAL"},
    "deploymentConfiguration": {...},
    "deployments": [{
      "id": "string",
      "status": "PRIMARY|ACTIVE|INACTIVE",
      "taskDefinition": "string",
      "desiredCount": number,
      "pendingCount": number,
      "runningCount": number,
      "failedTasks": number,
      "createdAt": number,
      "updatedAt": number,
      "rolloutState": "COMPLETED|IN_PROGRESS|FAILED",
      "rolloutStateReason": "string"
    }],
    "loadBalancers": [...],
    "networkConfiguration": {...},
    "events": [...],
    "createdAt": number,
    "createdBy": "string",
    "tags": [...]
  }
}
```

---

## DescribeServices

### Request
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | No | Cluster name/ARN |
| services | Array[String] | Yes | Max 10 service names/ARNs |
| include | Array[String] | No | TAGS |

### Response
```json
{
  "services": [Service],
  "failures": [Failure]
}
```

---

## UpdateService

### Request (Key Parameters)
| Parameter | Type | Required | Triggers New Deployment |
|-----------|------|----------|------------------------|
| service | String | Yes | - |
| cluster | String | No | No |
| desiredCount | Integer | No | No |
| taskDefinition | String | No | Yes |
| forceNewDeployment | Boolean | No | Yes |
| networkConfiguration | Object | No | Yes |
| loadBalancers | Array | No | Yes |
| platformVersion | String | No | Yes |
| serviceConnectConfiguration | Object | No | Yes |

### forceNewDeployment Use Cases
- Pull new image with same image:tag
- Update Fargate platform
- Apply tag propagation settings to all tasks

### SIGTERM/SIGKILL Processing
1. When stopping tasks, SIGTERM is sent to containers
2. 30 second timeout
3. SIGKILL sent if not terminated within 30 seconds

---

## DeleteService

### Request
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | No | Cluster name/ARN |
| service | String | Yes | Service name |
| force | Boolean | No | Delete even if task count > 0 (REPLICA only) |

### Deletion Conditions
- No running tasks, or desiredCount=0
- Or force=true

### Status Transition
```
ACTIVE -> DRAINING -> INACTIVE
```

---

## ListServices

### Request
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | No | Cluster name/ARN |
| launchType | String | No | EC2/FARGATE/EXTERNAL/MANAGED_INSTANCES |
| maxResults | Integer | No | 1-100 (default 10) |
| nextToken | String | No | Pagination token |
| schedulingStrategy | String | No | REPLICA/DAEMON |
| resourceManagementType | String | No | CUSTOMER/ECS |

### Response
```json
{
  "serviceArns": ["string"],
  "nextToken": "string"
}
```

---

## Specific Errors
- AccessDeniedException
- PlatformTaskDefinitionIncompatibilityException
- PlatformUnknownException
- UnsupportedFeatureException
- NamespaceNotFoundException
- ServiceNotActiveException
