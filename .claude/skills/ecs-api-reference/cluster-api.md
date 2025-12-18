# ECS Cluster API Specifications

## CreateCluster

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| clusterName | String | No | Cluster name (max 255 chars). Defaults to `default` |
| capacityProviders | Array[String] | No | Capacity providers (FARGATE/FARGATE_SPOT, etc.) |
| defaultCapacityProviderStrategy | Array[CapacityProviderStrategyItem] | No | Default strategy |
| configuration | ClusterConfiguration | No | Execute Command/Managed Storage settings |
| serviceConnectDefaults | ClusterServiceConnectDefaultsRequest | No | Service Connect default namespace |
| settings | Array[ClusterSetting] | No | Container Insights settings, etc. |
| tags | Array[Tag] | No | Metadata tags (max 50) |

### ClusterConfiguration Structure
```json
{
  "executeCommandConfiguration": {
    "kmsKeyId": "string",
    "logging": "NONE|DEFAULT|OVERRIDE",
    "logConfiguration": {
      "cloudWatchLogGroupName": "string",
      "cloudWatchEncryptionEnabled": boolean,
      "s3BucketName": "string",
      "s3EncryptionEnabled": boolean,
      "s3KeyPrefix": "string"
    }
  },
  "managedStorageConfiguration": {
    "kmsKeyId": "string",
    "fargateEphemeralStorageKmsKeyId": "string"
  }
}
```

### Response - Cluster Object
```json
{
  "cluster": {
    "clusterArn": "string",
    "clusterName": "string",
    "status": "ACTIVE|PROVISIONING|DEPROVISIONING|FAILED|INACTIVE",
    "activeServicesCount": number,
    "runningTasksCount": number,
    "pendingTasksCount": number,
    "registeredContainerInstancesCount": number,
    "capacityProviders": ["string"],
    "defaultCapacityProviderStrategy": [...],
    "configuration": {...},
    "serviceConnectDefaults": {"namespace": "string"},
    "settings": [{"name": "containerInsights", "value": "enabled|disabled|enhanced"}],
    "tags": [...]
  }
}
```

### Important Notes
- Attempts to automatically create service-linked role
- Auto Scaling group-based capacity providers cannot be shared with other clusters
- Returns HTTP 200 on success

---

## DescribeClusters

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| clusters | Array[String] | No | Max 100 cluster names/ARNs |
| include | Array[String] | No | ATTACHMENTS, CONFIGURATIONS, SETTINGS, STATISTICS, TAGS |

### Response
```json
{
  "clusters": [Cluster],
  "failures": [
    {
      "arn": "string",
      "reason": "string",
      "detail": "string"
    }
  ]
}
```

### STATISTICS Content
Task/service counts by launch type

---

## ListClusters

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| maxResults | Integer | No | 1-100 (default 100) |
| nextToken | String | No | Pagination token |

### Response
```json
{
  "clusterArns": ["arn:aws:ecs:region:account:cluster/name"],
  "nextToken": "string"
}
```

---

## DeleteCluster

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | Yes | Cluster name or ARN |

### Prerequisites for Deletion
1. Deregister all container instances
2. Delete all services (desiredCount=0 -> DeleteService)
3. Delete all capacity providers
4. Stop all active tasks

### Specific Errors
- ClusterContainsCapacityProviderException
- ClusterContainsContainerInstancesException
- ClusterContainsServicesException
- ClusterContainsTasksException
- UpdateInProgressException

### Important Notes
- Remains discoverable in INACTIVE state for a period after deletion
- Behavior may change in the future

---

## UpdateCluster

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | Yes | Cluster name |
| configuration | ClusterConfiguration | No | Update Execute Command settings |
| serviceConnectDefaults | Object | No | Service Connect namespace |
| settings | Array[ClusterSetting] | No | Cluster settings |

### ClusterSetting
```json
{
  "name": "containerInsights",
  "value": "enabled|disabled|enhanced"
}
```

---

## UpdateClusterSettings

### Request Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| cluster | String | Yes | Cluster name |
| settings | Array[ClusterSetting] | Yes | Settings to update |
