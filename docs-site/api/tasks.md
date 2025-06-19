# Task API Reference

## Overview

Task management APIs allow you to run, stop, and manage ECS tasks. A task is the instantiation of a task definition within a cluster.

## RunTask

Starts a new task from the specified task definition on the specified cluster.

### Request Syntax

```json
{
  "capacityProviderStrategy": [
    {
      "capacityProvider": "string",
      "weight": 123,
      "base": 123
    }
  ],
  "cluster": "string",
  "count": 123,
  "enableECSManagedTags": true,
  "enableExecuteCommand": true,
  "group": "string",
  "launchType": "EC2|FARGATE|EXTERNAL",
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["string"],
      "securityGroups": ["string"],
      "assignPublicIp": "ENABLED|DISABLED"
    }
  },
  "overrides": {
    "containerOverrides": [
      {
        "name": "string",
        "command": ["string"],
        "environment": [
          {
            "name": "string",
            "value": "string"
          }
        ],
        "environmentFiles": [
          {
            "value": "string",
            "type": "s3"
          }
        ],
        "cpu": 123,
        "memory": 123,
        "memoryReservation": 123,
        "resourceRequirements": [
          {
            "type": "GPU|InferenceAccelerator",
            "value": "string"
          }
        ]
      }
    ],
    "cpu": "string",
    "inferenceAcceleratorOverrides": [
      {
        "deviceName": "string",
        "deviceType": "string"
      }
    ],
    "executionRoleArn": "string",
    "memory": "string",
    "taskRoleArn": "string",
    "ephemeralStorage": {
      "sizeInGiB": 123
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
  "propagateTags": "TASK_DEFINITION|SERVICE|NONE",
  "referenceId": "string",
  "startedBy": "string",
  "tags": [
    {
      "key": "string",
      "value": "string"
    }
  ],
  "taskDefinition": "string"
}
```

### Request Parameters

- **taskDefinition** (string, required): The family and revision or full ARN of the task definition to run.
- **cluster** (string): The cluster on which to run your task.
- **count** (integer): The number of instantiations of the specified task to place on your cluster.
- **launchType** (string): The infrastructure on which to run your standalone task.
- **networkConfiguration** (object): The network configuration for the task.
- **overrides** (object): A list of container overrides.
- **placementConstraints** (array): An array of placement constraint objects.
- **placementStrategy** (array): The placement strategy objects to use for the task.
- **platformVersion** (string): The platform version on which to run your task.
- **propagateTags** (string): Specifies whether to propagate tags from the task definition or service.
- **group** (string): The name of the task group to associate with the task.
- **tags** (array): The metadata that you apply to the task.
- **enableExecuteCommand** (boolean): Whether to enable Amazon ECS Exec for the tasks.

### Response Syntax

```json
{
  "tasks": [
    {
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
      "attributes": [
        {
          "name": "string",
          "value": "string",
          "targetType": "container-instance",
          "targetId": "string"
        }
      ],
      "availabilityZone": "string",
      "capacityProviderName": "string",
      "clusterArn": "string",
      "connectivity": "CONNECTED|DISCONNECTED",
      "connectivityAt": "2024-01-01T00:00:00.000Z",
      "containerInstanceArn": "string",
      "containers": [
        {
          "containerArn": "string",
          "taskArn": "string",
          "name": "string",
          "image": "string",
          "imageDigest": "string",
          "runtimeId": "string",
          "lastStatus": "string",
          "exitCode": 123,
          "reason": "string",
          "networkBindings": [
            {
              "bindIP": "string",
              "containerPort": 123,
              "hostPort": 123,
              "protocol": "tcp|udp"
            }
          ],
          "networkInterfaces": [
            {
              "attachmentId": "string",
              "privateIpv4Address": "string",
              "ipv6Address": "string"
            }
          ],
          "healthStatus": "HEALTHY|UNHEALTHY|UNKNOWN",
          "managedAgents": [
            {
              "lastStartedAt": "2024-01-01T00:00:00.000Z",
              "name": "ExecuteCommandAgent",
              "reason": "string",
              "lastStatus": "string"
            }
          ],
          "cpu": "string",
          "memory": "string",
          "memoryReservation": "string",
          "gpuIds": ["string"]
        }
      ],
      "cpu": "string",
      "createdAt": "2024-01-01T00:00:00.000Z",
      "desiredStatus": "string",
      "enableExecuteCommand": true,
      "executionStoppedAt": "2024-01-01T00:00:00.000Z",
      "group": "string",
      "healthStatus": "HEALTHY|UNHEALTHY|UNKNOWN",
      "inferenceAccelerators": [
        {
          "deviceName": "string",
          "deviceType": "string"
        }
      ],
      "lastStatus": "string",
      "launchType": "EC2|FARGATE|EXTERNAL",
      "memory": "string",
      "overrides": {},
      "platformVersion": "string",
      "platformFamily": "string",
      "pullStartedAt": "2024-01-01T00:00:00.000Z",
      "pullStoppedAt": "2024-01-01T00:00:00.000Z",
      "startedAt": "2024-01-01T00:00:00.000Z",
      "startedBy": "string",
      "stopCode": "TaskFailedToStart|EssentialContainerExited|UserInitiated",
      "stoppedAt": "2024-01-01T00:00:00.000Z",
      "stoppedReason": "string",
      "stoppingAt": "2024-01-01T00:00:00.000Z",
      "tags": [],
      "taskArn": "string",
      "taskDefinitionArn": "string",
      "version": 123,
      "ephemeralStorage": {
        "sizeInGiB": 123
      }
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
curl -X POST http://localhost:8080/v1/RunTask \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RunTask" \
  -d '{
    "cluster": "production",
    "taskDefinition": "webapp:1",
    "count": 2,
    "launchType": "FARGATE",
    "networkConfiguration": {
      "awsvpcConfiguration": {
        "subnets": ["subnet-12345", "subnet-67890"],
        "securityGroups": ["sg-12345"],
        "assignPublicIp": "ENABLED"
      }
    },
    "overrides": {
      "containerOverrides": [
        {
          "name": "webapp",
          "environment": [
            {
              "name": "ENV",
              "value": "production"
            }
          ]
        }
      ]
    },
    "tags": [
      {
        "key": "Purpose",
        "value": "batch-job"
      }
    ]
  }'
```

## StopTask

Stops a running task.

### Request Syntax

```json
{
  "cluster": "string",
  "task": "string",
  "reason": "string"
}
```

### Request Parameters

- **task** (string, required): The task ID or full ARN of the task to stop.
- **cluster** (string): The cluster that hosts the task.
- **reason** (string): An optional message for why the task is being stopped.

### Response Syntax

```json
{
  "task": {
    "taskArn": "string",
    "clusterArn": "string",
    "taskDefinitionArn": "string",
    "containerInstanceArn": "string",
    "overrides": {},
    "lastStatus": "string",
    "desiredStatus": "STOPPED",
    "cpu": "string",
    "memory": "string",
    "containers": [],
    "startedBy": "string",
    "version": 123,
    "stoppedReason": "string",
    "stopCode": "TaskFailedToStart|EssentialContainerExited|UserInitiated",
    "connectivity": "CONNECTED|DISCONNECTED",
    "connectivityAt": "2024-01-01T00:00:00.000Z",
    "pullStartedAt": "2024-01-01T00:00:00.000Z",
    "pullStoppedAt": "2024-01-01T00:00:00.000Z",
    "executionStoppedAt": "2024-01-01T00:00:00.000Z",
    "createdAt": "2024-01-01T00:00:00.000Z",
    "startedAt": "2024-01-01T00:00:00.000Z",
    "stoppingAt": "2024-01-01T00:00:00.000Z",
    "stoppedAt": "2024-01-01T00:00:00.000Z",
    "group": "string",
    "launchType": "EC2|FARGATE|EXTERNAL",
    "platformVersion": "string",
    "platformFamily": "string",
    "attachments": [],
    "healthStatus": "HEALTHY|UNHEALTHY|UNKNOWN",
    "tags": [],
    "enableExecuteCommand": true
  }
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/StopTask \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.StopTask" \
  -d '{
    "cluster": "production",
    "task": "arn:aws:ecs:us-east-1:123456789012:task/production/1234567890abcdef",
    "reason": "Manual stop for maintenance"
  }'
```

## DescribeTasks

Describes one or more of your tasks.

### Request Syntax

```json
{
  "cluster": "string",
  "tasks": ["string"],
  "include": ["TAGS"]
}
```

### Request Parameters

- **tasks** (array, required): A list of up to 100 task IDs or full ARN entries.
- **cluster** (string): The cluster that hosts the tasks.
- **include** (array): Specifies whether to see additional details about the tasks.

### Response Syntax

```json
{
  "tasks": [
    {
      "attachments": [],
      "attributes": [],
      "availabilityZone": "string",
      "capacityProviderName": "string",
      "clusterArn": "string",
      "connectivity": "CONNECTED|DISCONNECTED",
      "connectivityAt": "2024-01-01T00:00:00.000Z",
      "containerInstanceArn": "string",
      "containers": [
        {
          "containerArn": "string",
          "taskArn": "string",
          "name": "string",
          "image": "string",
          "imageDigest": "string",
          "runtimeId": "string",
          "lastStatus": "PENDING|RUNNING|STOPPED",
          "exitCode": 123,
          "reason": "string",
          "networkBindings": [],
          "networkInterfaces": [],
          "healthStatus": "HEALTHY|UNHEALTHY|UNKNOWN",
          "managedAgents": [],
          "cpu": "string",
          "memory": "string",
          "memoryReservation": "string",
          "gpuIds": []
        }
      ],
      "cpu": "string",
      "createdAt": "2024-01-01T00:00:00.000Z",
      "desiredStatus": "RUNNING|PENDING|STOPPED",
      "enableExecuteCommand": true,
      "executionStoppedAt": "2024-01-01T00:00:00.000Z",
      "group": "string",
      "healthStatus": "HEALTHY|UNHEALTHY|UNKNOWN",
      "inferenceAccelerators": [],
      "lastStatus": "PENDING|RUNNING|STOPPED",
      "launchType": "EC2|FARGATE|EXTERNAL",
      "memory": "string",
      "overrides": {},
      "platformVersion": "string",
      "platformFamily": "string",
      "pullStartedAt": "2024-01-01T00:00:00.000Z",
      "pullStoppedAt": "2024-01-01T00:00:00.000Z",
      "startedAt": "2024-01-01T00:00:00.000Z",
      "startedBy": "string",
      "stopCode": "TaskFailedToStart|EssentialContainerExited|UserInitiated",
      "stoppedAt": "2024-01-01T00:00:00.000Z",
      "stoppedReason": "string",
      "stoppingAt": "2024-01-01T00:00:00.000Z",
      "tags": [],
      "taskArn": "string",
      "taskDefinitionArn": "string",
      "version": 123,
      "ephemeralStorage": {
        "sizeInGiB": 123
      }
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
curl -X POST http://localhost:8080/v1/DescribeTasks \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DescribeTasks" \
  -d '{
    "cluster": "production",
    "tasks": [
      "arn:aws:ecs:us-east-1:123456789012:task/production/1234567890abcdef",
      "arn:aws:ecs:us-east-1:123456789012:task/production/fedcba0987654321"
    ],
    "include": ["TAGS"]
  }'
```

## ListTasks

Returns a list of tasks.

### Request Syntax

```json
{
  "cluster": "string",
  "containerInstance": "string",
  "family": "string",
  "group": "string",
  "serviceName": "string",
  "desiredStatus": "RUNNING|PENDING|STOPPED",
  "launchType": "EC2|FARGATE|EXTERNAL",
  "startedBy": "string",
  "maxResults": 123,
  "nextToken": "string"
}
```

### Request Parameters

- **cluster** (string): The cluster to query.
- **containerInstance** (string): The container instance ID or full ARN.
- **family** (string): The name of the task definition family.
- **group** (string): The name of the task group.
- **serviceName** (string): The name of the service.
- **desiredStatus** (string): The task desired status (RUNNING, PENDING, or STOPPED).
- **launchType** (string): The launch type for services to list.
- **startedBy** (string): The value to filter results by.
- **maxResults** (integer): The maximum number of task results returned.
- **nextToken** (string): The nextToken value from a previous paginated request.

### Response Syntax

```json
{
  "taskArns": ["string"],
  "nextToken": "string"
}
```

### Example

```bash
curl -X POST http://localhost:8080/v1/ListTasks \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListTasks" \
  -d '{
    "cluster": "production",
    "serviceName": "web-api",
    "desiredStatus": "RUNNING",
    "maxResults": 10
  }'
```

## Task States

Tasks transition through several states:

1. **PROVISIONING**: Resources being allocated
2. **PENDING**: Waiting to be placed on container instance
3. **ACTIVATING**: Performing final steps before running
4. **RUNNING**: Task is running
5. **DEACTIVATING**: Performing cleanup steps
6. **STOPPING**: Task is in the process of stopping
7. **DEPROVISIONING**: Resources being released
8. **STOPPED**: Task has stopped

### Task Stop Reasons

Common stop codes and reasons:

- **TaskFailedToStart**: Task failed to start
- **EssentialContainerExited**: Essential container in task exited
- **UserInitiated**: User requested the task to stop
- **ServiceSchedulerInitiated**: Service scheduler stopped the task
- **SpotInterruption**: Spot instance was interrupted
- **TerminationNotice**: Container instance received termination notice

## Error Responses

### ClusterNotFoundException

```json
{
  "__type": "ClusterNotFoundException",
  "message": "The specified cluster could not be found."
}
```

### InvalidParameterException

```json
{
  "__type": "InvalidParameterException",
  "message": "Invalid task definition."
}
```

### AccessDeniedException

```json
{
  "__type": "AccessDeniedException",
  "message": "User is not authorized to perform this operation."
}
```

### ServiceNotActiveException

```json
{
  "__type": "ServiceNotActiveException",
  "message": "The specified service is not active."
}
```

### PlatformUnknownException

```json
{
  "__type": "PlatformUnknownException",
  "message": "The specified platform version does not exist."
}
```

### PlatformTaskDefinitionIncompatibilityException

```json
{
  "__type": "PlatformTaskDefinitionIncompatibilityException",
  "message": "The specified task definition is incompatible with the compute platform."
}
```

### UnsupportedFeatureException

```json
{
  "__type": "UnsupportedFeatureException",
  "message": "The requested feature is not supported."
}
```

## Best Practices

1. **Task Placement**: Use placement strategies and constraints to optimize resource utilization.

2. **Container Overrides**: Use overrides to customize task behavior without creating new task definitions:
   ```json
   {
     "overrides": {
       "containerOverrides": [
         {
           "name": "app",
           "environment": [
             {
               "name": "CONFIG_OVERRIDE",
               "value": "custom-value"
             }
           ]
         }
       ]
     }
   }
   ```

3. **Task Groups**: Use task groups to organize related tasks for easier management.

4. **Monitoring**: Always monitor task health and status:
   ```bash
   # Monitor task status
   aws ecs describe-tasks \
     --cluster production \
     --tasks $TASK_ARN \
     --query 'tasks[0].{Status:lastStatus,Health:healthStatus}'
   ```

5. **Resource Allocation**: Set appropriate CPU and memory for your tasks based on actual usage.

6. **Error Handling**: Implement proper error handling and retry logic in your applications.

7. **Logging**: Configure comprehensive logging for debugging and monitoring.