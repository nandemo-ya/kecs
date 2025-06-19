# Service API Reference

## Overview

Service management APIs allow you to create, manage, and delete ECS services. A service enables you to run and maintain a specified number of instances of a task definition simultaneously.

## CreateService

Runs and maintains a desired number of tasks from a specified task definition.

### Request Syntax

```json
{
  "cluster": "string",
  "serviceName": "string",
  "taskDefinition": "string",
  "loadBalancers": [
    {
      "targetGroupArn": "string",
      "loadBalancerName": "string",
      "containerName": "string",
      "containerPort": 123
    }
  ],
  "serviceRegistries": [
    {
      "registryArn": "string",
      "port": 123,
      "containerName": "string",
      "containerPort": 123
    }
  ],
  "desiredCount": 123,
  "clientToken": "string",
  "launchType": "EC2|FARGATE|EXTERNAL",
  "capacityProviderStrategy": [
    {
      "capacityProvider": "string",
      "weight": 123,
      "base": 123
    }
  ],
  "platformVersion": "string",
  "role": "string",
  "deploymentConfiguration": {
    "deploymentCircuitBreaker": {
      "enable": true,
      "rollback": true
    },
    "maximumPercent": 123,
    "minimumHealthyPercent": 123
  },
  "placementConstraints": [
    {
      "type": "distinctInstance|memberOf",
      "expression": "string"
    }
  ],
  "placementStrategy": [
    {
      "type": "random|spread|binpack",
      "field": "string"
    }
  ],
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["string"],
      "securityGroups": ["string"],
      "assignPublicIp": "ENABLED|DISABLED"
    }
  },
  "healthCheckGracePeriodSeconds": 123,
  "schedulingStrategy": "REPLICA|DAEMON",
  "deploymentController": {
    "type": "ECS|CODE_DEPLOY|EXTERNAL"
  },
  "tags": [
    {
      "key": "string",
      "value": "string"
    }
  ],
  "enableECSManagedTags": true,
  "propagateTags": "TASK_DEFINITION|SERVICE|NONE",
  "enableExecuteCommand": true
}
```

### Request Parameters

- **cluster** (string): The cluster on which to run your service.
- **serviceName** (string, required): The name of your service.
- **taskDefinition** (string, required): The family and revision or full ARN of the task definition.
- **loadBalancers** (array): A load balancer object representing the load balancers to use.
- **serviceRegistries** (array): The details of the service discovery registry.
- **desiredCount** (integer): The number of instantiations of the task definition to place.
- **launchType** (string): The launch type on which to run your service.
- **platformVersion** (string): The platform version that your tasks run on.
- **deploymentConfiguration** (object): Optional deployment parameters.
- **placementConstraints** (array): An array of placement constraint objects.
- **placementStrategy** (array): The placement strategy objects to use.
- **networkConfiguration** (object): The network configuration for the service.
- **healthCheckGracePeriodSeconds** (integer): The period of time to ignore unhealthy load balancer health checks.
- **schedulingStrategy** (string): The scheduling strategy to use (REPLICA or DAEMON).
- **tags** (array): The metadata to apply to the service.
- **enableExecuteCommand** (boolean): Whether to enable ECS Exec for the service.

### Response Syntax

```json
{
  "service": {
    "serviceArn": "string",
    "serviceName": "string",
    "clusterArn": "string",
    "loadBalancers": [
      {
        "targetGroupArn": "string",
        "loadBalancerName": "string",
        "containerName": "string",
        "containerPort": 123
      }
    ],
    "serviceRegistries": [
      {
        "registryArn": "string",
        "port": 123,
        "containerName": "string",
        "containerPort": 123
      }
    ],
    "status": "ACTIVE|DRAINING|INACTIVE",
    "desiredCount": 123,
    "runningCount": 123,
    "pendingCount": 123,
    "launchType": "EC2|FARGATE|EXTERNAL",
    "platformVersion": "string",
    "platformFamily": "string",
    "taskDefinition": "string",
    "deploymentConfiguration": {
      "deploymentCircuitBreaker": {
        "enable": true,
        "rollback": true
      },
      "maximumPercent": 123,
      "minimumHealthyPercent": 123
    },
    "deployments": [
      {
        "id": "string",
        "status": "PRIMARY|ACTIVE|INACTIVE",
        "taskDefinition": "string",
        "desiredCount": 123,
        "pendingCount": 123,
        "runningCount": 123,
        "failedTasks": 123,
        "createdAt": "2024-01-01T00:00:00.000Z",
        "updatedAt": "2024-01-01T00:00:00.000Z",
        "launchType": "EC2|FARGATE|EXTERNAL",
        "platformVersion": "string",
        "platformFamily": "string",
        "networkConfiguration": {},
        "rolloutState": "COMPLETED|FAILED|IN_PROGRESS",
        "rolloutStateReason": "string"
      }
    ],
    "roleArn": "string",
    "events": [
      {
        "id": "string",
        "createdAt": "2024-01-01T00:00:00.000Z",
        "message": "string"
      }
    ],
    "createdAt": "2024-01-01T00:00:00.000Z",
    "placementConstraints": [],
    "placementStrategy": [],
    "networkConfiguration": {},
    "healthCheckGracePeriodSeconds": 123,
    "schedulingStrategy": "REPLICA|DAEMON",
    "deploymentController": {
      "type": "ECS|CODE_DEPLOY|EXTERNAL"
    },
    "tags": [],
    "createdBy": "string",
    "enableECSManagedTags": true,
    "propagateTags": "TASK_DEFINITION|SERVICE|NONE",
    "enableExecuteCommand": true
  }
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/CreateService \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateService" \
  -d '{
    "cluster": "production",
    "serviceName": "web-api",
    "taskDefinition": "webapp:1",
    "desiredCount": 3,
    "launchType": "FARGATE",
    "networkConfiguration": {
      "awsvpcConfiguration": {
        "subnets": ["subnet-12345", "subnet-67890"],
        "securityGroups": ["sg-12345"],
        "assignPublicIp": "ENABLED"
      }
    },
    "deploymentConfiguration": {
      "maximumPercent": 200,
      "minimumHealthyPercent": 100,
      "deploymentCircuitBreaker": {
        "enable": true,
        "rollback": true
      }
    },
    "tags": [
      {
        "key": "Environment",
        "value": "production"
      }
    ]
  }'
```

## UpdateService

Modifies the parameters of a service.

### Request Syntax

```json
{
  "cluster": "string",
  "service": "string",
  "desiredCount": 123,
  "taskDefinition": "string",
  "capacityProviderStrategy": [
    {
      "capacityProvider": "string",
      "weight": 123,
      "base": 123
    }
  ],
  "deploymentConfiguration": {
    "deploymentCircuitBreaker": {
      "enable": true,
      "rollback": true
    },
    "maximumPercent": 123,
    "minimumHealthyPercent": 123
  },
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["string"],
      "securityGroups": ["string"],
      "assignPublicIp": "ENABLED|DISABLED"
    }
  },
  "placementConstraints": [
    {
      "type": "distinctInstance|memberOf",
      "expression": "string"
    }
  ],
  "placementStrategy": [
    {
      "type": "random|spread|binpack",
      "field": "string"
    }
  ],
  "platformVersion": "string",
  "forceNewDeployment": true,
  "healthCheckGracePeriodSeconds": 123,
  "enableExecuteCommand": true,
  "enableECSManagedTags": true,
  "loadBalancers": [
    {
      "targetGroupArn": "string",
      "loadBalancerName": "string",
      "containerName": "string",
      "containerPort": 123
    }
  ],
  "propagateTags": "TASK_DEFINITION|SERVICE|NONE",
  "serviceRegistries": [
    {
      "registryArn": "string",
      "port": 123,
      "containerName": "string",
      "containerPort": 123
    }
  ]
}
```

### Request Parameters

- **cluster** (string): The cluster that hosts the service.
- **service** (string, required): The name of the service to update.
- **desiredCount** (integer): The number of instantiations of the task definition.
- **taskDefinition** (string): The task definition to run in your service.
- **forceNewDeployment** (boolean): Force a new deployment of the service.
- **deploymentConfiguration** (object): Optional deployment parameters.
- **networkConfiguration** (object): The network configuration for the service.
- **platformVersion** (string): The platform version on which your tasks run.
- **healthCheckGracePeriodSeconds** (integer): The period of time to ignore unhealthy health checks.

### Response Syntax

Same as CreateService response.

### Example

```bash
curl -X POST http://localhost:8080/v1/UpdateService \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.UpdateService" \
  -d '{
    "cluster": "production",
    "service": "web-api",
    "taskDefinition": "webapp:2",
    "desiredCount": 5,
    "forceNewDeployment": true
  }'
```

## DeleteService

Deletes a specified service. Services must have zero running tasks before deletion.

### Request Syntax

```json
{
  "cluster": "string",
  "service": "string",
  "force": true
}
```

### Request Parameters

- **cluster** (string): The cluster that hosts the service.
- **service** (string, required): The name of the service to delete.
- **force** (boolean): Force deletion even if the service has not scaled down to zero.

### Response Syntax

```json
{
  "service": {
    "serviceArn": "string",
    "serviceName": "string",
    "clusterArn": "string",
    "status": "ACTIVE|DRAINING|INACTIVE",
    "desiredCount": 0,
    "runningCount": 0,
    "pendingCount": 0
  }
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/DeleteService \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DeleteService" \
  -d '{
    "cluster": "production",
    "service": "web-api"
  }'
```

## DescribeServices

Describes the specified services running in your cluster.

### Request Syntax

```json
{
  "cluster": "string",
  "services": ["string"],
  "include": ["TAGS"]
}
```

### Request Parameters

- **cluster** (string): The cluster that hosts the services.
- **services** (array, required): A list of services to describe.
- **include** (array): Specifies whether to include additional details.

### Response Syntax

```json
{
  "services": [
    {
      "serviceArn": "string",
      "serviceName": "string",
      "clusterArn": "string",
      "loadBalancers": [],
      "serviceRegistries": [],
      "status": "ACTIVE|DRAINING|INACTIVE",
      "desiredCount": 123,
      "runningCount": 123,
      "pendingCount": 123,
      "launchType": "EC2|FARGATE|EXTERNAL",
      "platformVersion": "string",
      "platformFamily": "string",
      "taskDefinition": "string",
      "deploymentConfiguration": {},
      "deployments": [],
      "roleArn": "string",
      "events": [],
      "createdAt": "2024-01-01T00:00:00.000Z",
      "placementConstraints": [],
      "placementStrategy": [],
      "networkConfiguration": {},
      "healthCheckGracePeriodSeconds": 123,
      "schedulingStrategy": "REPLICA|DAEMON",
      "deploymentController": {},
      "tags": [],
      "createdBy": "string",
      "enableECSManagedTags": true,
      "propagateTags": "TASK_DEFINITION|SERVICE|NONE",
      "enableExecuteCommand": true
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
curl -X POST http://localhost:8080/v1/DescribeServices \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DescribeServices" \
  -d '{
    "cluster": "production",
    "services": ["web-api", "backend-api"],
    "include": ["TAGS"]
  }'
```

## ListServices

Returns a list of services.

### Request Syntax

```json
{
  "cluster": "string",
  "nextToken": "string",
  "maxResults": 123,
  "launchType": "EC2|FARGATE|EXTERNAL",
  "schedulingStrategy": "REPLICA|DAEMON"
}
```

### Request Parameters

- **cluster** (string): The cluster to list services for.
- **nextToken** (string): The nextToken value from a previous paginated request.
- **maxResults** (integer): The maximum number of service results returned.
- **launchType** (string): Filter services by launch type.
- **schedulingStrategy** (string): Filter services by scheduling strategy.

### Response Syntax

```json
{
  "serviceArns": ["string"],
  "nextToken": "string"
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/ListServices \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListServices" \
  -d '{
    "cluster": "production",
    "maxResults": 10
  }'
```

## Error Responses

### ServiceNotFoundException

Returned when the specified service could not be found.

```json
{
  "__type": "ServiceNotFoundException",
  "message": "The specified service could not be found."
}
```

### ServiceNotActiveException

Returned when the specified service is not active.

```json
{
  "__type": "ServiceNotActiveException",
  "message": "The specified service is not active."
}
```

### InvalidParameterException

Returned when request parameters are invalid.

```json
{
  "__type": "InvalidParameterException",
  "message": "Invalid service configuration."
}
```

### ClusterNotFoundException

Returned when the specified cluster could not be found.

```json
{
  "__type": "ClusterNotFoundException",
  "message": "The specified cluster could not be found."
}
```

### PlatformUnknownException

Returned when the specified platform version is not valid.

```json
{
  "__type": "PlatformUnknownException",
  "message": "The specified platform version does not exist."
}
```

### PlatformTaskDefinitionIncompatibilityException

Returned when the task definition is incompatible with the platform version.

```json
{
  "__type": "PlatformTaskDefinitionIncompatibilityException",
  "message": "The specified task definition is incompatible with the platform version."
}
```

## Best Practices

1. **Service Naming**: Use descriptive names that indicate the service's purpose (e.g., `web-frontend`, `api-backend`, `worker-queue`).

2. **Deployment Configuration**: Always configure deployment parameters:
   ```json
   {
     "deploymentConfiguration": {
       "maximumPercent": 200,
       "minimumHealthyPercent": 100,
       "deploymentCircuitBreaker": {
         "enable": true,
         "rollback": true
       }
     }
   }
   ```

3. **Health Checks**: Set appropriate health check grace periods for services with load balancers.

4. **Placement Strategies**: Use placement strategies to optimize resource utilization:
   ```json
   {
     "placementStrategy": [
       {
         "type": "spread",
         "field": "attribute:ecs.availability-zone"
       }
     ]
   }
   ```

5. **Rolling Updates**: Use rolling updates for zero-downtime deployments:
   ```bash
   # Update service with new task definition
   aws ecs update-service \
     --cluster production \
     --service web-api \
     --task-definition webapp:2 \
     --force-new-deployment
   ```

6. **Service Discovery**: Enable service discovery for service-to-service communication.

7. **Auto Scaling**: Configure auto scaling for production services to handle varying loads.