# DeleteService API

## Overview

The DeleteService API removes an Amazon ECS service from a cluster. The service is marked for deletion and enters the DRAINING state before removal. You can optionally force delete a service even if it has running tasks.

## API Specification

### Request

```json
{
  "cluster": "string",
  "service": "string",
  "force": boolean
}
```

#### Parameters

- **cluster** (string, optional): The short name or full Amazon Resource Name (ARN) of the cluster that hosts the service to delete. If you do not specify a cluster, the default cluster is assumed.
- **service** (string, required): The name of the service to delete.
- **force** (boolean, optional): If true, allows you to delete a service even if it has not been scaled down to zero tasks. It is only necessary to use this if the service is using the REPLICA scheduling strategy.

### Response

```json
{
  "service": {
    "serviceArn": "string",
    "serviceName": "string",
    "clusterArn": "string",
    "status": "DRAINING",
    "desiredCount": 0,
    "runningCount": number,
    "pendingCount": number,
    // ... other service fields
  }
}
```

## Behavior

1. **Standard Deletion**:
   - The service must have a desired count of 0 before deletion
   - The service status is set to DRAINING
   - Associated Kubernetes Deployment and Service resources are deleted
   - The service is removed from storage

2. **Force Deletion**:
   - When `force=true`, the service can be deleted even with running tasks
   - The desired count is automatically set to 0
   - The service enters DRAINING state and is deleted

3. **Kubernetes Integration**:
   - Deletes the associated Kubernetes Deployment
   - Deletes the associated Kubernetes Service (if exists)
   - Continues with deletion even if Kubernetes resource deletion fails

## Error Conditions

- **Service Not Found**: Returns error if the specified service does not exist
- **Cluster Not Found**: Returns error if the specified cluster does not exist
- **Non-Zero Desired Count**: Returns error if desired count > 0 and force=false

## Example Usage

### Standard Delete (after scaling down)
```bash
# First scale down the service
curl -X POST http://localhost:8080/v1/updateservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "my-cluster",
    "service": "my-service",
    "desiredCount": 0
  }'

# Then delete the service
curl -X POST http://localhost:8080/v1/deleteservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "my-cluster",
    "service": "my-service"
  }'
```

### Force Delete
```bash
curl -X POST http://localhost:8080/v1/deleteservice \
  -H "Content-Type: application/json" \
  -d '{
    "cluster": "my-cluster",
    "service": "my-service",
    "force": true
  }'
```