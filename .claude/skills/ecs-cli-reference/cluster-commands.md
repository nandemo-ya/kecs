# ECS Cluster CLI Commands

## create-cluster

### Synopsis
```bash
aws ecs create-cluster \
    [--cluster-name <value>] \
    [--tags <value>] \
    [--settings <value>] \
    [--configuration <value>] \
    [--capacity-providers <value>] \
    [--default-capacity-provider-strategy <value>] \
    [--service-connect-defaults <value>]
```

### Key Options
| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `--cluster-name` | string | No | Cluster name (max 255 chars). Default: `default` |
| `--tags` | list | No | Key-value metadata (max 50) |
| `--settings` | list | No | Container Insights settings |
| `--configuration` | structure | No | Execute command and storage config |
| `--capacity-providers` | list | No | FARGATE, FARGATE_SPOT, or custom |
| `--default-capacity-provider-strategy` | list | No | Default strategy |
| `--service-connect-defaults` | structure | No | Service Connect namespace |

### Settings Structure
```json
[
  {
    "name": "containerInsights",
    "value": "enabled|disabled|enhanced"
  }
]
```

### Configuration Structure
```json
{
  "executeCommandConfiguration": {
    "kmsKeyId": "string",
    "logging": "NONE|DEFAULT|OVERRIDE",
    "logConfiguration": {
      "cloudWatchLogGroupName": "string",
      "cloudWatchEncryptionEnabled": true,
      "s3BucketName": "string",
      "s3EncryptionEnabled": true,
      "s3KeyPrefix": "string"
    }
  },
  "managedStorageConfiguration": {
    "kmsKeyId": "string",
    "fargateEphemeralStorageKmsKeyId": "string"
  }
}
```

### Example
```bash
aws ecs create-cluster \
    --cluster-name MyCluster \
    --settings name=containerInsights,value=enabled \
    --capacity-providers FARGATE FARGATE_SPOT \
    --default-capacity-provider-strategy capacityProvider=FARGATE,weight=1,base=1
```

### Output
```json
{
  "cluster": {
    "clusterArn": "arn:aws:ecs:region:account:cluster/MyCluster",
    "clusterName": "MyCluster",
    "status": "ACTIVE",
    "registeredContainerInstancesCount": 0,
    "runningTasksCount": 0,
    "pendingTasksCount": 0,
    "activeServicesCount": 0,
    "capacityProviders": ["FARGATE", "FARGATE_SPOT"],
    "defaultCapacityProviderStrategy": [...],
    "settings": [{"name": "containerInsights", "value": "enabled"}],
    "tags": []
  }
}
```

---

## describe-clusters

### Synopsis
```bash
aws ecs describe-clusters \
    [--clusters <value>] \
    [--include <value>]
```

### Options
| Option | Type | Description |
|--------|------|-------------|
| `--clusters` | list | Max 100 cluster names/ARNs. Default: default cluster |
| `--include` | list | Additional info: `ATTACHMENTS`, `CONFIGURATIONS`, `SETTINGS`, `STATISTICS`, `TAGS` |

### Example
```bash
aws ecs describe-clusters \
    --clusters MyCluster \
    --include SETTINGS STATISTICS TAGS
```

### Output
```json
{
  "clusters": [
    {
      "clusterArn": "string",
      "clusterName": "string",
      "status": "ACTIVE|PROVISIONING|DEPROVISIONING|FAILED|INACTIVE",
      "registeredContainerInstancesCount": 0,
      "runningTasksCount": 0,
      "pendingTasksCount": 0,
      "activeServicesCount": 0,
      "statistics": [
        {"name": "runningEC2TasksCount", "value": "0"},
        {"name": "runningFargateTasksCount", "value": "0"}
      ],
      "tags": []
    }
  ],
  "failures": []
}
```

---

## list-clusters

### Synopsis
```bash
aws ecs list-clusters \
    [--max-items <value>] \
    [--starting-token <value>]
```

### Options (Pagination)
| Option | Type | Description |
|--------|------|-------------|
| `--max-items` | integer | Max total items (1-100) |
| `--page-size` | integer | Items per API call |
| `--starting-token` | string | Pagination token |

### Example
```bash
aws ecs list-clusters --max-items 10
```

### Output
```json
{
  "clusterArns": [
    "arn:aws:ecs:us-west-2:123456789012:cluster/default",
    "arn:aws:ecs:us-west-2:123456789012:cluster/MyCluster"
  ],
  "nextToken": "string"
}
```

---

## delete-cluster

### Synopsis
```bash
aws ecs delete-cluster --cluster <value>
```

### Prerequisites
1. Deregister all container instances
2. Delete all services (set desiredCount=0 first)
3. Stop all tasks
4. Remove capacity providers

### Errors
- `ClusterContainsContainerInstancesException`
- `ClusterContainsServicesException`
- `ClusterContainsTasksException`
- `ClusterContainsCapacityProviderException`

### Example
```bash
aws ecs delete-cluster --cluster MyCluster
```

---

## update-cluster

### Synopsis
```bash
aws ecs update-cluster \
    --cluster <value> \
    [--settings <value>] \
    [--configuration <value>] \
    [--service-connect-defaults <value>]
```

### Example
```bash
aws ecs update-cluster \
    --cluster MyCluster \
    --settings name=containerInsights,value=enhanced
```

---

## update-cluster-settings

### Synopsis
```bash
aws ecs update-cluster-settings \
    --cluster <value> \
    --settings <value>
```

### Example
```bash
aws ecs update-cluster-settings \
    --cluster MyCluster \
    --settings name=containerInsights,value=disabled
```
