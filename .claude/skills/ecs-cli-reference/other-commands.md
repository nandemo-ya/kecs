# ECS Other CLI Commands

## Tags Commands

### tag-resource

```bash
aws ecs tag-resource \
    --resource-arn <value> \
    --tags <value>
```

**Options:**
| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `--resource-arn` | string | Yes | ARN of resource to tag |
| `--tags` | list | Yes | Key-value pairs (max 50) |

**Supported Resources:** capacity providers, tasks, services, task definitions, clusters, container instances

**Service ARN Format:** Must use long format:
```
arn:aws:ecs:region:account:service/cluster-name/service-name
```

**Tag Constraints:**
- Key: 1-128 chars
- Value: 0-256 chars
- Cannot use `aws:` prefix

**Example:**
```bash
aws ecs tag-resource \
    --resource-arn arn:aws:ecs:us-west-2:123456789012:cluster/MyCluster \
    --tags key=environment,value=production key=team,value=backend
```

**Output:** None (HTTP 200 on success)

---

### untag-resource

```bash
aws ecs untag-resource \
    --resource-arn <value> \
    --tag-keys <value>
```

**Example:**
```bash
aws ecs untag-resource \
    --resource-arn arn:aws:ecs:us-west-2:123456789012:cluster/MyCluster \
    --tag-keys environment team
```

**Output:** None (HTTP 200 on success)

---

### list-tags-for-resource

```bash
aws ecs list-tags-for-resource \
    --resource-arn <value>
```

**Example:**
```bash
aws ecs list-tags-for-resource \
    --resource-arn arn:aws:ecs:us-west-2:123456789012:cluster/MyCluster
```

**Output:**
```json
{
  "tags": [
    {"key": "environment", "value": "production"},
    {"key": "team", "value": "backend"}
  ]
}
```

---

## Attributes Commands

### put-attributes

```bash
aws ecs put-attributes \
    --attributes <value> \
    [--cluster <value>]
```

**Attribute Structure:**
```json
[
  {
    "name": "string",
    "value": "string",
    "targetType": "container-instance",
    "targetId": "string"
  }
]
```

**Constraints:**
- Max 10 attributes per resource
- Max 10 attributes per call
- Name: 1-128 chars
- Value: 1-128 chars

**Example:**
```bash
aws ecs put-attributes \
    --attributes name=stack,value=production,targetId=arn:aws:ecs:us-west-2:123456789012:container-instance/abc123
```

**Output:**
```json
{
  "attributes": [
    {"name": "stack", "value": "production", "targetId": "..."}
  ]
}
```

---

### list-attributes

```bash
aws ecs list-attributes \
    --target-type <value> \
    [--cluster <value>] \
    [--attribute-name <value>] \
    [--attribute-value <value>] \
    [--max-items <value>]
```

**Options:**
| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `--target-type` | string | Yes | `container-instance` |
| `--attribute-name` | string | No | Filter by name |
| `--attribute-value` | string | No | Filter by value (requires name) |

**Example:**
```bash
aws ecs list-attributes \
    --target-type container-instance \
    --attribute-name stack \
    --attribute-value production
```

---

### delete-attributes

```bash
aws ecs delete-attributes \
    --attributes <value> \
    [--cluster <value>]
```

**Example:**
```bash
aws ecs delete-attributes \
    --attributes name=stack,targetId=arn:aws:ecs:us-west-2:123456789012:container-instance/abc123
```

---

## Account Settings Commands

### put-account-setting

```bash
aws ecs put-account-setting \
    --name <value> \
    --value <value> \
    [--principal-arn <value>]
```

**Valid Setting Names:**
| Name | Description | Values |
|------|-------------|--------|
| `serviceLongArnFormat` | Service ARN format | enabled/disabled |
| `taskLongArnFormat` | Task ARN format | enabled/disabled |
| `containerInstanceLongArnFormat` | Container instance ARN format | enabled/disabled |
| `awsvpcTrunking` | ENI limit changes | enabled/disabled |
| `containerInsights` | Container Insights | enabled/disabled/enhanced |
| `dualStackIPv6` | IPv6 in dual-stack VPC | enabled/disabled |
| `fargateTaskRetirementWaitPeriod` | Task retirement wait | 0/7/14 (days) |
| `tagResourceAuthorization` | Tag authorization | enabled/disabled/on/off |
| `defaultLogDriverMode` | Log driver mode | blocking/non-blocking |
| `guardDutyActivate` | Runtime Monitoring | read-only |

**Example:**
```bash
aws ecs put-account-setting \
    --name containerInsights \
    --value enhanced
```

**Output:**
```json
{
  "setting": {
    "name": "containerInsights",
    "value": "enhanced",
    "principalArn": "arn:aws:iam::123456789012:user/admin",
    "type": "user"
  }
}
```

---

### put-account-setting-default

```bash
aws ecs put-account-setting-default \
    --name <value> \
    --value <value>
```

Sets default for entire account (all users/roles).

---

### list-account-settings

```bash
aws ecs list-account-settings \
    [--name <value>] \
    [--value <value>] \
    [--principal-arn <value>] \
    [--effective-settings | --no-effective-settings] \
    [--max-items <value>]
```

**Options:**
| Option | Type | Description |
|--------|------|-------------|
| `--effective-settings` | boolean | Return root user or default settings |
| `--name` | string | Filter by setting name |
| `--principal-arn` | string | Filter by user/role ARN |

**Example:**
```bash
aws ecs list-account-settings --effective-settings
```

---

### delete-account-setting

```bash
aws ecs delete-account-setting \
    --name <value> \
    [--principal-arn <value>]
```

---

## Capacity Provider Commands

### create-capacity-provider

```bash
aws ecs create-capacity-provider \
    --name <value> \
    [--auto-scaling-group-provider <value>] \
    [--tags <value>]
```

**Auto Scaling Group Provider:**
```json
{
  "autoScalingGroupArn": "string",
  "managedScaling": {
    "status": "ENABLED|DISABLED",
    "targetCapacity": 100,
    "minimumScalingStepSize": 1,
    "maximumScalingStepSize": 10000,
    "instanceWarmupPeriod": 300
  },
  "managedTerminationProtection": "ENABLED|DISABLED",
  "managedDraining": "ENABLED|DISABLED"
}
```

**Example:**
```bash
aws ecs create-capacity-provider \
    --name MyCapacityProvider \
    --auto-scaling-group-provider \
      "autoScalingGroupArn=arn:aws:autoscaling:...,managedScaling={status=ENABLED,targetCapacity=100}"
```

---

### describe-capacity-providers

```bash
aws ecs describe-capacity-providers \
    [--capacity-providers <value>] \
    [--include <value>] \
    [--max-results <value>]
```

**Options:**
| Option | Type | Description |
|--------|------|-------------|
| `--capacity-providers` | list | Up to 100 names/ARNs |
| `--include` | list | `TAGS` |
| `--max-results` | integer | 1-10 |

---

### update-capacity-provider

```bash
aws ecs update-capacity-provider \
    --name <value> \
    --auto-scaling-group-provider <value>
```

---

### delete-capacity-provider

```bash
aws ecs delete-capacity-provider \
    --capacity-provider <value>
```

**Prerequisite:** Not associated with any cluster.

---

### put-cluster-capacity-providers

```bash
aws ecs put-cluster-capacity-providers \
    --cluster <value> \
    --capacity-providers <value> \
    --default-capacity-provider-strategy <value>
```

Associates capacity providers with cluster and sets default strategy.

**Example:**
```bash
aws ecs put-cluster-capacity-providers \
    --cluster MyCluster \
    --capacity-providers FARGATE FARGATE_SPOT \
    --default-capacity-provider-strategy \
      capacityProvider=FARGATE,weight=1,base=1 \
      capacityProvider=FARGATE_SPOT,weight=1
```

---

## TaskSet Commands

TaskSets are used with services using the `EXTERNAL` deployment controller.

### create-task-set

```bash
aws ecs create-task-set \
    --cluster <value> \
    --service <value> \
    --task-definition <value> \
    [--external-id <value>] \
    [--network-configuration <value>] \
    [--load-balancers <value>] \
    [--service-registries <value>] \
    [--launch-type <value>] \
    [--capacity-provider-strategy <value>] \
    [--platform-version <value>] \
    [--scale <value>] \
    [--client-token <value>] \
    [--tags <value>]
```

**Scale Structure:**
```json
{
  "value": 50.0,
  "unit": "PERCENT"
}
```

**Example:**
```bash
aws ecs create-task-set \
    --cluster MyCluster \
    --service MyService \
    --task-definition my-app:2 \
    --scale value=100,unit=PERCENT \
    --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345]}"
```

---

### describe-task-sets

```bash
aws ecs describe-task-sets \
    --cluster <value> \
    --service <value> \
    [--task-sets <value>] \
    [--include <value>]
```

**Output:**
```json
{
  "taskSets": [
    {
      "id": "string",
      "taskSetArn": "string",
      "status": "PRIMARY|ACTIVE|DRAINING",
      "taskDefinition": "string",
      "computedDesiredCount": 2,
      "pendingCount": 0,
      "runningCount": 2,
      "scale": {"value": 100.0, "unit": "PERCENT"},
      "stabilityStatus": "STEADY_STATE|STABILIZING"
    }
  ],
  "failures": []
}
```

---

### update-task-set

```bash
aws ecs update-task-set \
    --cluster <value> \
    --service <value> \
    --task-set <value> \
    --scale <value>
```

**Example:**
```bash
aws ecs update-task-set \
    --cluster MyCluster \
    --service MyService \
    --task-set ecs-svc/123456789 \
    --scale value=50,unit=PERCENT
```

---

### delete-task-set

```bash
aws ecs delete-task-set \
    --cluster <value> \
    --service <value> \
    --task-set <value> \
    [--force | --no-force]
```

**Options:**
| Option | Type | Description |
|--------|------|-------------|
| `--force` | boolean | Delete even if not scaled to zero |

**Example:**
```bash
aws ecs delete-task-set \
    --cluster MyCluster \
    --service MyService \
    --task-set ecs-svc/123456789 \
    --force
```

---

### update-service-primary-task-set

```bash
aws ecs update-service-primary-task-set \
    --cluster <value> \
    --service <value> \
    --primary-task-set <value>
```

Sets which task set is PRIMARY (receives production traffic).
