# ECS Other APIs Specifications

## Tags API

### TagResource

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| resourceArn | String | Yes | Resource ARN to tag |
| tags | Array[Tag] | Yes | Tags to apply |

#### Supported Resources
- ECS capacity providers
- Tasks
- Services
- Task definitions
- Clusters
- Container instances

#### Important Notes
- Short ARN format not supported for services:
  - Invalid: `arn:aws:ecs:region:account:service/service-name`
  - Valid: `arn:aws:ecs:region:account:service/cluster-name/service-name`
- Existing tags not included in request are unchanged (add-only operation)

#### Response
HTTP 200 with empty body `{}`

#### Errors
- ResourceNotFoundException

---

### UntagResource

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| resourceArn | String | Yes | Resource ARN |
| tagKeys | Array[String] | Yes | Tag keys to remove |

#### Response
HTTP 200 with empty body `{}`

---

### ListTagsForResource

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| resourceArn | String | Yes | Resource ARN |

#### Response
```json
{
  "tags": [
    {
      "key": "string",
      "value": "string"
    }
  ]
}
```

---

## TaskSet API

### CreateTaskSet

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | Yes | Cluster name/ARN |
| service | String | Yes | Service name/ARN |
| taskDefinition | String | Yes | Task definition |
| capacityProviderStrategy | Array | No | Mutually exclusive with launchType |
| launchType | String | No | EC2/FARGATE/EXTERNAL/MANAGED_INSTANCES |
| loadBalancers | Array | No | ALB/NLB configuration |
| networkConfiguration | Object | No | VPC/subnet/security group settings |
| platformVersion | String | No | Fargate platform version (default: LATEST) |
| scale | Scale | No | Floating-point percentage of desired task count |
| serviceRegistries | Array | No | Service discovery registries |
| tags | Array | No | Metadata tags (max 50) |
| externalId | String | No | External system identifier |
| clientToken | String | No | Idempotency token (max 36 chars) |

#### Important Notes
- Used with services using EXTERNAL deployment controller type
- capacityProviderStrategy and launchType are mutually exclusive

#### Response
```json
{
  "taskSet": {
    "id": "string",
    "taskSetArn": "string",
    "clusterArn": "string",
    "serviceArn": "string",
    "status": "string",
    "taskDefinition": "string",
    "launchType": "string",
    "platformVersion": "string",
    "computedDesiredCount": number,
    "pendingCount": number,
    "runningCount": number,
    "createdAt": number,
    "updatedAt": number,
    "stabilityStatus": "string",
    "stabilityStatusAt": number,
    "scale": {
      "value": number,
      "unit": "string"
    },
    "loadBalancers": [...],
    "networkConfiguration": {...},
    "tags": [...]
  }
}
```

---

### DescribeTaskSets

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | Yes | Cluster name/ARN |
| service | String | Yes | Service name/ARN |
| taskSets | Array[String] | No | Task set IDs/ARNs to describe |
| include | Array[String] | No | TAGS |

#### Response
```json
{
  "taskSets": [TaskSet],
  "failures": [Failure]
}
```

---

### UpdateTaskSet

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | Yes | Cluster name/ARN |
| service | String | Yes | Service name/ARN |
| taskSet | String | Yes | Task set ID/ARN |
| scale | Scale | Yes | New scale value |

---

### DeleteTaskSet

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | Yes | Cluster name/ARN |
| service | String | Yes | Service name/ARN |
| taskSet | String | Yes | Task set ID/ARN |
| force | Boolean | No | Force delete |

---

## Attributes API

### PutAttributes

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| attributes | Array[Attribute] | Yes | Max 10 attributes per call |
| cluster | String | No | Cluster name/ARN |

#### Attribute Object
```json
{
  "name": "string",
  "targetId": "string",  // Container instance ARN, etc.
  "targetType": "string",
  "value": "string"
}
```

#### Limits
- Max 10 custom attributes per resource
- Max 10 attributes per single call

#### Response
```json
{
  "attributes": [Attribute]
}
```

#### Errors
- AttributeLimitExceededException

---

### DeleteAttributes

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| attributes | Array[Attribute] | Yes | Attributes to delete |
| cluster | String | No | Cluster name/ARN |

---

### ListAttributes

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| targetType | String | Yes | container-instance |
| attributeName | String | No | Filter by attribute name |
| attributeValue | String | No | Filter by value |
| cluster | String | No | Cluster name/ARN |
| maxResults | Integer | No | 1-100 |
| nextToken | String | No | Pagination token |

#### Response
```json
{
  "attributes": [Attribute],
  "nextToken": "string"
}
```

---

## Account Settings API

### PutAccountSetting

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| name | String | Yes | Setting name |
| value | String | Yes | Setting value |
| principalArn | String | No | User/role/root user ARN |

#### Valid Setting Names
| Name | Description | Values |
|------|-------------|--------|
| serviceLongArnFormat | Service ARN/ID format | enabled/disabled |
| taskLongArnFormat | Task ARN/ID format | enabled/disabled |
| containerInstanceLongArnFormat | Container instance ARN/ID format | enabled/disabled |
| awsvpcTrunking | ENI limit changes | enabled/disabled |
| containerInsights | Container Insights monitoring | enabled/disabled/enhanced |
| dualStackIPv6 | IPv6 address assignment in dual-stack VPC | enabled/disabled |
| fargateTaskRetirementWaitPeriod | Fargate task retirement wait time | 0/7/14 (days) |
| tagResourceAuthorization | Tag authorization on resource creation | enabled/disabled/on/off |
| defaultLogDriverMode | Default log driver delivery mode | blocking/non-blocking |
| guardDutyActivate | ECS Runtime Monitoring (read-only) | - |

#### Important Notes
- fargateTaskRetirementWaitPeriod requires root user principalArn
- Federated users cannot specify principalArn
- Settings are region-specific

---

### PutAccountSettingDefault

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| name | String | Yes | Setting name |
| value | String | Yes | Setting value |

Sets default for entire account (all users/roles)

---

### ListAccountSettings

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| effectiveSettings | Boolean | No | Return root user or default settings |
| maxResults | Integer | No | 1-10 (default: 10) |
| name | String | No | Setting name filter |
| nextToken | String | No | Pagination token |
| principalArn | String | No | User/role ARN |
| value | String | No | Filter by value (requires name) |

#### Response
```json
{
  "settings": [
    {
      "name": "string",
      "principalArn": "string",
      "type": "string",
      "value": "string"
    }
  ],
  "nextToken": "string"
}
```

---

### DeleteAccountSetting

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| name | String | Yes | Setting name |
| principalArn | String | No | User/role ARN |

---

## Capacity Provider API

### CreateCapacityProvider

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| name | String | Yes | Capacity provider name |
| autoScalingGroupProvider | Object | Yes | Auto Scaling group settings |
| tags | Array | No | Tags |

#### autoScalingGroupProvider Structure
```json
{
  "autoScalingGroupArn": "string",
  "managedScaling": {
    "status": "ENABLED|DISABLED",
    "targetCapacity": number,
    "minimumScalingStepSize": number,
    "maximumScalingStepSize": number,
    "instanceWarmupPeriod": number
  },
  "managedTerminationProtection": "ENABLED|DISABLED",
  "managedDraining": "ENABLED|DISABLED"
}
```

---

### DescribeCapacityProviders

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| capacityProviders | Array[String] | No | Capacity provider names/ARNs |
| include | Array[String] | No | TAGS |
| maxResults | Integer | No | 1-100 |
| nextToken | String | No | Pagination token |

---

### UpdateCapacityProvider

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| name | String | Yes | Capacity provider name |
| autoScalingGroupProvider | Object | Yes | Updated settings |

---

### DeleteCapacityProvider

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| capacityProvider | String | Yes | Capacity provider name/ARN |

#### Prerequisites
- Not associated with any cluster

---

### PutClusterCapacityProviders

#### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | Yes | Cluster name |
| capacityProviders | Array[String] | Yes | Capacity providers to associate |
| defaultCapacityProviderStrategy | Array | Yes | Default strategy |

Associates capacity providers with a cluster and sets default strategy.
