# Cluster API Reference

## Overview

Cluster management APIs allow you to create, manage, and delete ECS clusters. A cluster is a logical grouping of tasks or services.

## CreateCluster

Creates a new Amazon ECS cluster.

### Request Syntax

```json
{
  "clusterName": "string",
  "tags": [
    {
      "key": "string",
      "value": "string"
    }
  ],
  "settings": [
    {
      "name": "containerInsights",
      "value": "enabled"
    }
  ],
  "configuration": {
    "executeCommandConfiguration": {
      "logging": "NONE|DEFAULT|OVERRIDE",
      "logConfiguration": {
        "cloudWatchLogGroupName": "string",
        "s3BucketName": "string",
        "s3KeyPrefix": "string"
      }
    }
  },
  "capacityProviders": ["string"],
  "defaultCapacityProviderStrategy": [
    {
      "capacityProvider": "string",
      "weight": 123,
      "base": 123
    }
  ]
}
```

### Request Parameters

- **clusterName** (string): The name of your cluster. If not specified, a default name is used.
- **tags** (array): The metadata applied to the cluster as key-value pairs.
- **settings** (array): The setting to use when creating a cluster.
- **configuration** (object): The execute command configuration for the cluster.
- **capacityProviders** (array): The capacity providers to associate with the cluster.
- **defaultCapacityProviderStrategy** (array): The default capacity provider strategy for the cluster.

### Response Syntax

```json
{
  "cluster": {
    "clusterArn": "string",
    "clusterName": "string",
    "status": "ACTIVE|PROVISIONING|DEPROVISIONING|FAILED|INACTIVE",
    "registeredContainerInstancesCount": 123,
    "runningTasksCount": 123,
    "pendingTasksCount": 123,
    "activeServicesCount": 123,
    "statistics": [
      {
        "name": "string",
        "value": "string"
      }
    ],
    "tags": [
      {
        "key": "string",
        "value": "string"
      }
    ],
    "settings": [
      {
        "name": "containerInsights",
        "value": "enabled"
      }
    ],
    "capacityProviders": ["string"],
    "defaultCapacityProviderStrategy": [
      {
        "capacityProvider": "string",
        "weight": 123,
        "base": 123
      }
    ],
    "attachments": [
      {
        "id": "string",
        "type": "string",
        "status": "string",
        "details": [
          {
            "name": "string",
            "value": "string"
          }
        ]
      }
    ],
    "attachmentsStatus": "string"
  }
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/CreateCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster" \
  -d '{
    "clusterName": "production",
    "tags": [
      {
        "key": "Environment",
        "value": "prod"
      }
    ],
    "settings": [
      {
        "name": "containerInsights",
        "value": "enabled"
      }
    ]
  }'
```

## DeleteCluster

Deletes an Amazon ECS cluster. The cluster must have no active services or registered container instances.

### Request Syntax

```json
{
  "cluster": "string"
}
```

### Request Parameters

- **cluster** (string, required): The short name or full Amazon Resource Name (ARN) of the cluster to delete.

### Response Syntax

```json
{
  "cluster": {
    "clusterArn": "string",
    "clusterName": "string",
    "status": "ACTIVE|PROVISIONING|DEPROVISIONING|FAILED|INACTIVE",
    "registeredContainerInstancesCount": 123,
    "runningTasksCount": 123,
    "pendingTasksCount": 123,
    "activeServicesCount": 123
  }
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/DeleteCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DeleteCluster" \
  -d '{
    "cluster": "production"
  }'
```

## DescribeClusters

Describes one or more of your clusters.

### Request Syntax

```json
{
  "clusters": ["string"],
  "include": ["ATTACHMENTS", "CONFIGURATIONS", "SETTINGS", "STATISTICS", "TAGS"]
}
```

### Request Parameters

- **clusters** (array): A list of up to 100 cluster names or full cluster ARNs to describe.
- **include** (array): Additional information to include about your clusters.

### Response Syntax

```json
{
  "clusters": [
    {
      "clusterArn": "string",
      "clusterName": "string",
      "status": "ACTIVE|PROVISIONING|DEPROVISIONING|FAILED|INACTIVE",
      "registeredContainerInstancesCount": 123,
      "runningTasksCount": 123,
      "pendingTasksCount": 123,
      "activeServicesCount": 123,
      "statistics": [
        {
          "name": "string",
          "value": "string"
        }
      ],
      "tags": [
        {
          "key": "string",
          "value": "string"
        }
      ],
      "settings": [
        {
          "name": "containerInsights",
          "value": "enabled"
        }
      ],
      "capacityProviders": ["string"],
      "defaultCapacityProviderStrategy": [
        {
          "capacityProvider": "string",
          "weight": 123,
          "base": 123
        }
      ]
    }
  ],
  "failures": [
    {
      "arn": "string",
      "reason": "string",
      "detail": "string"
    }
  ]
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/DescribeClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DescribeClusters" \
  -d '{
    "clusters": ["production", "staging"],
    "include": ["TAGS", "SETTINGS", "STATISTICS"]
  }'
```

## ListClusters

Returns a list of existing clusters.

### Request Syntax

```json
{
  "nextToken": "string",
  "maxResults": 123
}
```

### Request Parameters

- **nextToken** (string): The nextToken value returned from a previous paginated request.
- **maxResults** (integer): The maximum number of cluster results returned. Default: 100, Maximum: 100.

### Response Syntax

```json
{
  "clusterArns": ["string"],
  "nextToken": "string"
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -d '{
    "maxResults": 10
  }'
```

## UpdateCluster

Updates an Amazon ECS cluster.

### Request Syntax

```json
{
  "cluster": "string",
  "settings": [
    {
      "name": "containerInsights",
      "value": "enabled"
    }
  ],
  "configuration": {
    "executeCommandConfiguration": {
      "logging": "NONE|DEFAULT|OVERRIDE",
      "logConfiguration": {
        "cloudWatchLogGroupName": "string",
        "s3BucketName": "string",
        "s3KeyPrefix": "string"
      }
    }
  }
}
```

### Request Parameters

- **cluster** (string, required): The name or ARN of the cluster to update.
- **settings** (array): The cluster settings to update.
- **configuration** (object): The execute command configuration for the cluster.

### Response Syntax

```json
{
  "cluster": {
    "clusterArn": "string",
    "clusterName": "string",
    "status": "ACTIVE|PROVISIONING|DEPROVISIONING|FAILED|INACTIVE",
    "registeredContainerInstancesCount": 123,
    "runningTasksCount": 123,
    "pendingTasksCount": 123,
    "activeServicesCount": 123,
    "settings": [
      {
        "name": "containerInsights",
        "value": "enabled"
      }
    ],
    "configuration": {
      "executeCommandConfiguration": {
        "logging": "DEFAULT"
      }
    }
  }
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/UpdateCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.UpdateCluster" \
  -d '{
    "cluster": "production",
    "settings": [
      {
        "name": "containerInsights",
        "value": "disabled"
      }
    ]
  }'
```

## Error Responses

### ClusterNotFoundException

Returned when the specified cluster could not be found.

```json
{
  "__type": "ClusterNotFoundException",
  "message": "The specified cluster could not be found."
}
```

### InvalidParameterException

Returned when request parameters are invalid.

```json
{
  "__type": "InvalidParameterException",
  "message": "Invalid cluster name format."
}
```

### UpdateInProgressException

Returned when a cluster update is already in progress.

```json
{
  "__type": "UpdateInProgressException",
  "message": "The cluster cannot be updated while another update is in progress."
}
```

## Best Practices

1. **Naming Conventions**: Use meaningful names for clusters (e.g., `production`, `staging`, `dev-team-a`).

2. **Tags**: Always tag clusters with metadata for better organization:
   - Environment: prod, staging, dev
   - Team: backend, frontend, data
   - Purpose: web-apps, batch-processing

3. **Container Insights**: Enable container insights for production clusters to get detailed metrics.

4. **Capacity Providers**: Configure capacity providers for better resource management.

5. **Error Handling**: Always handle potential errors in your applications:
   ```python
   try:
       response = ecs.create_cluster(clusterName='production')
   except ecs.exceptions.ClientException as e:
       print(f"Error creating cluster: {e}")
   ```