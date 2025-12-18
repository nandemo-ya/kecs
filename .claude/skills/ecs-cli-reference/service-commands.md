# ECS Service CLI Commands

## create-service

### Synopsis
```bash
aws ecs create-service \
    --service-name <value> \
    [--cluster <value>] \
    [--task-definition <value>] \
    [--desired-count <value>] \
    [--launch-type <value>] \
    [--capacity-provider-strategy <value>] \
    [--platform-version <value>] \
    [--scheduling-strategy <value>] \
    [--deployment-controller <value>] \
    [--deployment-configuration <value>] \
    [--network-configuration <value>] \
    [--load-balancers <value>] \
    [--service-registries <value>] \
    [--placement-constraints <value>] \
    [--placement-strategy <value>] \
    [--health-check-grace-period-seconds <value>] \
    [--enable-execute-command | --disable-execute-command] \
    [--enable-ecs-managed-tags | --no-enable-ecs-managed-tags] \
    [--propagate-tags <value>] \
    [--service-connect-configuration <value>] \
    [--volume-configurations <value>] \
    [--tags <value>] \
    [--client-token <value>]
```

### Required Parameters
| Option | Type | Description |
|--------|------|-------------|
| `--service-name` | string | Service name (max 255 chars) |

### Key Optional Parameters
| Option | Type | Description |
|--------|------|-------------|
| `--cluster` | string | Cluster name/ARN. Default: `default` |
| `--task-definition` | string | family:revision or ARN |
| `--desired-count` | integer | Number of tasks (required for REPLICA) |
| `--launch-type` | string | EC2, FARGATE, EXTERNAL, MANAGED_INSTANCES |
| `--scheduling-strategy` | string | REPLICA (default) or DAEMON |

### Network Configuration (awsvpc)
```json
{
  "awsvpcConfiguration": {
    "subnets": ["subnet-xxx"],
    "securityGroups": ["sg-xxx"],
    "assignPublicIp": "ENABLED|DISABLED"
  }
}
```

### Deployment Configuration
```json
{
  "deploymentCircuitBreaker": {
    "enable": true,
    "rollback": true
  },
  "maximumPercent": 200,
  "minimumHealthyPercent": 100,
  "alarms": {
    "alarmNames": ["string"],
    "enable": true,
    "rollback": true
  }
}
```

### Load Balancer Configuration
```json
[
  {
    "targetGroupArn": "arn:aws:elasticloadbalancing:...",
    "containerName": "app",
    "containerPort": 80
  }
]
```

### Placement Strategy
```json
[
  {"type": "spread", "field": "attribute:ecs.availability-zone"},
  {"type": "binpack", "field": "memory"}
]
```
Types: `spread`, `binpack`, `random`

### Service Connect Configuration
```json
{
  "enabled": true,
  "namespace": "my-namespace",
  "services": [
    {
      "portName": "http",
      "discoveryName": "backend",
      "clientAliases": [
        {"port": 80, "dnsName": "backend.local"}
      ]
    }
  ]
}
```

### Example
```bash
aws ecs create-service \
    --cluster MyCluster \
    --service-name MyService \
    --task-definition my-app:1 \
    --desired-count 2 \
    --launch-type FARGATE \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}" \
    --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:...,containerName=app,containerPort=80"
```

### Output
```json
{
  "service": {
    "serviceArn": "arn:aws:ecs:region:account:service/cluster/service",
    "serviceName": "MyService",
    "clusterArn": "string",
    "status": "ACTIVE",
    "desiredCount": 2,
    "runningCount": 0,
    "pendingCount": 0,
    "taskDefinition": "string",
    "deployments": [...],
    "events": []
  }
}
```

---

## describe-services

### Synopsis
```bash
aws ecs describe-services \
    --services <value> \
    [--cluster <value>] \
    [--include <value>]
```

### Options
| Option | Type | Description |
|--------|------|-------------|
| `--services` | list | Max 10 service names/ARNs |
| `--cluster` | string | Cluster name/ARN |
| `--include` | list | `TAGS` |

### Example
```bash
aws ecs describe-services \
    --cluster MyCluster \
    --services MyService \
    --include TAGS
```

### Output
```json
{
  "services": [
    {
      "serviceArn": "string",
      "serviceName": "string",
      "status": "ACTIVE|DRAINING|INACTIVE",
      "desiredCount": 2,
      "runningCount": 2,
      "pendingCount": 0,
      "taskDefinition": "string",
      "deploymentConfiguration": {...},
      "deployments": [
        {
          "id": "string",
          "status": "PRIMARY|ACTIVE|INACTIVE",
          "taskDefinition": "string",
          "desiredCount": 2,
          "runningCount": 2,
          "rolloutState": "COMPLETED|IN_PROGRESS|FAILED"
        }
      ],
      "events": [
        {
          "id": "string",
          "createdAt": "timestamp",
          "message": "string"
        }
      ]
    }
  ],
  "failures": []
}
```

---

## list-services

### Synopsis
```bash
aws ecs list-services \
    [--cluster <value>] \
    [--launch-type <value>] \
    [--scheduling-strategy <value>] \
    [--max-items <value>] \
    [--starting-token <value>]
```

### Options
| Option | Type | Description |
|--------|------|-------------|
| `--cluster` | string | Cluster name/ARN |
| `--launch-type` | string | EC2, FARGATE, EXTERNAL |
| `--scheduling-strategy` | string | REPLICA or DAEMON |
| `--max-items` | integer | Max items (1-100, default 10) |

### Example
```bash
aws ecs list-services --cluster MyCluster --launch-type FARGATE
```

### Output
```json
{
  "serviceArns": [
    "arn:aws:ecs:region:account:service/cluster/service1",
    "arn:aws:ecs:region:account:service/cluster/service2"
  ],
  "nextToken": "string"
}
```

---

## update-service

### Synopsis
```bash
aws ecs update-service \
    --service <value> \
    [--cluster <value>] \
    [--desired-count <value>] \
    [--task-definition <value>] \
    [--capacity-provider-strategy <value>] \
    [--deployment-configuration <value>] \
    [--network-configuration <value>] \
    [--placement-constraints <value>] \
    [--placement-strategy <value>] \
    [--platform-version <value>] \
    [--force-new-deployment | --no-force-new-deployment] \
    [--health-check-grace-period-seconds <value>] \
    [--enable-execute-command | --disable-execute-command] \
    [--enable-ecs-managed-tags | --no-enable-ecs-managed-tags] \
    [--load-balancers <value>] \
    [--propagate-tags <value>] \
    [--service-registries <value>] \
    [--service-connect-configuration <value>] \
    [--volume-configurations <value>]
```

### Triggers New Deployment
- `--task-definition`
- `--force-new-deployment`
- `--network-configuration`
- `--load-balancers`
- `--platform-version`
- `--service-connect-configuration`

### Force New Deployment Use Cases
- Pull new image with same tag
- Update Fargate platform version
- Apply tag propagation to existing tasks

### Example: Scale Service
```bash
aws ecs update-service \
    --cluster MyCluster \
    --service MyService \
    --desired-count 4
```

### Example: Update Task Definition
```bash
aws ecs update-service \
    --cluster MyCluster \
    --service MyService \
    --task-definition my-app:2
```

### Example: Force Redeploy
```bash
aws ecs update-service \
    --cluster MyCluster \
    --service MyService \
    --force-new-deployment
```

---

## delete-service

### Synopsis
```bash
aws ecs delete-service \
    --service <value> \
    [--cluster <value>] \
    [--force | --no-force]
```

### Options
| Option | Type | Description |
|--------|------|-------------|
| `--service` | string | Service name/ARN |
| `--cluster` | string | Cluster name/ARN |
| `--force` | boolean | Delete even with running tasks (REPLICA only) |

### Deletion Process
```
ACTIVE -> DRAINING -> INACTIVE
```

### Example
```bash
# Scale to 0 first (recommended)
aws ecs update-service --cluster MyCluster --service MyService --desired-count 0

# Then delete
aws ecs delete-service --cluster MyCluster --service MyService

# Or force delete
aws ecs delete-service --cluster MyCluster --service MyService --force
```
