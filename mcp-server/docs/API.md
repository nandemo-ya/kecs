# KECS MCP Server API Documentation

## Overview

The KECS MCP Server provides tools for interacting with KECS (Kubernetes-based ECS Compatible Service) through the Model Context Protocol. This document describes all available tools and their parameters.

## Tools Reference

### Cluster Management

#### list-clusters
List all ECS clusters.

**Parameters:**
- `maxResults` (number, optional): Maximum number of results to return (1-100)
- `nextToken` (string, optional): Token for pagination

**Example:**
```json
{
  "name": "list-clusters",
  "arguments": {
    "maxResults": 10
  }
}
```

#### describe-clusters
Get detailed information about one or more ECS clusters.

**Parameters:**
- `clusters` (string[], optional): Array of cluster names or ARNs
- `include` (string[], optional): Additional information to include: ATTACHMENTS, SETTINGS, STATISTICS, TAGS

**Example:**
```json
{
  "name": "describe-clusters",
  "arguments": {
    "clusters": ["default", "production"],
    "include": ["STATISTICS", "TAGS"]
  }
}
```

#### create-cluster
Create a new ECS cluster.

**Parameters:**
- `clusterName` (string, required): Name for the new cluster
- `tags` (object[], optional): Array of tags with `key` and `value` properties

**Example:**
```json
{
  "name": "create-cluster",
  "arguments": {
    "clusterName": "my-cluster",
    "tags": [
      {"key": "Environment", "value": "development"},
      {"key": "Team", "value": "backend"}
    ]
  }
}
```

#### delete-cluster
Delete an ECS cluster.

**Parameters:**
- `cluster` (string, required): Name or ARN of the cluster to delete

**Example:**
```json
{
  "name": "delete-cluster",
  "arguments": {
    "cluster": "my-cluster"
  }
}
```

### Service Management

#### list-services
List services in an ECS cluster.

**Parameters:**
- `cluster` (string, optional): Cluster name or ARN
- `maxResults` (number, optional): Maximum number of results (1-100)
- `nextToken` (string, optional): Token for pagination

#### describe-services
Get detailed information about one or more ECS services.

**Parameters:**
- `cluster` (string, optional): Cluster name or ARN
- `services` (string[], required): Array of service names or ARNs
- `include` (string[], optional): Additional information to include: TAGS

#### create-service
Create a new ECS service.

**Parameters:**
- `cluster` (string, optional): Cluster name or ARN
- `serviceName` (string, required): Name for the service
- `taskDefinition` (string, required): Task definition to use
- `desiredCount` (number, optional): Number of tasks to run
- `launchType` (string, optional): Launch type: EC2, FARGATE, EXTERNAL
- `platformVersion` (string, optional): Platform version for Fargate
- `deploymentConfiguration` (object, optional): Deployment configuration
  - `maximumPercent` (number): Maximum percent of tasks
  - `minimumHealthyPercent` (number): Minimum healthy percent
- `networkConfiguration` (object, optional): Network configuration for awsvpc mode
- `tags` (object[], optional): Tags to apply to the service

#### update-service
Update an existing ECS service.

**Parameters:**
- `cluster` (string, optional): Cluster name or ARN
- `service` (string, required): Service name or ARN
- `desiredCount` (number, optional): New desired count
- `taskDefinition` (string, optional): New task definition
- `forceNewDeployment` (boolean, optional): Force new deployment
- `deploymentConfiguration` (object, optional): New deployment configuration

#### delete-service
Delete an ECS service.

**Parameters:**
- `cluster` (string, optional): Cluster name or ARN
- `service` (string, required): Service name or ARN
- `force` (boolean, optional): Force deletion even if tasks are running

### Task Management

#### list-tasks
List tasks in an ECS cluster.

**Parameters:**
- `cluster` (string, optional): Cluster name or ARN
- `containerInstance` (string, optional): Filter by container instance
- `family` (string, optional): Filter by task definition family
- `serviceName` (string, optional): Filter by service name
- `desiredStatus` (string, optional): Filter by status: RUNNING, PENDING, STOPPED
- `startedBy` (string, optional): Filter by who started the task
- `maxResults` (number, optional): Maximum results (1-100)
- `nextToken` (string, optional): Pagination token

#### describe-tasks
Get detailed information about one or more ECS tasks.

**Parameters:**
- `cluster` (string, optional): Cluster name or ARN
- `tasks` (string[], required): Array of task IDs or ARNs
- `include` (string[], optional): Additional information: TAGS

#### run-task
Run a new task from a task definition.

**Parameters:**
- `cluster` (string, optional): Cluster name or ARN
- `taskDefinition` (string, required): Task definition to run
- `count` (number, optional): Number of tasks to run (1-10)
- `startedBy` (string, optional): Reference for who started the task
- `group` (string, optional): Task group name
- `launchType` (string, optional): Launch type: EC2, FARGATE, EXTERNAL
- `platformVersion` (string, optional): Platform version for Fargate
- `networkConfiguration` (object, optional): Network configuration
- `overrides` (object, optional): Task overrides
- `tags` (object[], optional): Tags for the task

#### stop-task
Stop a running ECS task.

**Parameters:**
- `cluster` (string, optional): Cluster name or ARN
- `task` (string, required): Task ID or ARN
- `reason` (string, optional): Reason for stopping

### Task Definition Management

#### list-task-definitions
List task definitions.

**Parameters:**
- `familyPrefix` (string, optional): Filter by family prefix
- `status` (string, optional): Filter by status: ACTIVE, INACTIVE
- `sort` (string, optional): Sort order: ASC, DESC
- `maxResults` (number, optional): Maximum results (1-100)
- `nextToken` (string, optional): Pagination token

#### describe-task-definition
Get detailed information about a task definition.

**Parameters:**
- `taskDefinition` (string, required): Task definition family:revision or ARN
- `include` (string[], optional): Additional information: TAGS

#### register-task-definition
Register a new task definition.

**Parameters:**
- `family` (string, required): Task definition family name
- `taskRoleArn` (string, optional): IAM role for tasks
- `executionRoleArn` (string, optional): IAM role for execution
- `networkMode` (string, optional): Network mode: bridge, host, awsvpc, none
- `containerDefinitions` (object[], required): Array of container definitions
- `volumes` (object[], optional): Volume definitions
- `placementConstraints` (object[], optional): Placement constraints
- `requiresCompatibilities` (string[], optional): Required compatibilities: EC2, FARGATE
- `cpu` (string, optional): CPU units for Fargate
- `memory` (string, optional): Memory for Fargate
- `tags` (object[], optional): Tags for the task definition

#### deregister-task-definition
Deregister a task definition.

**Parameters:**
- `taskDefinition` (string, required): Task definition to deregister

## Error Handling

All tools return errors in a consistent format:

```json
{
  "error": {
    "code": "ResourceNotFoundException",
    "message": "The specified cluster does not exist"
  }
}
```

Common error codes:
- `ResourceNotFoundException`: Resource not found
- `InvalidParameterException`: Invalid parameter value
- `ClientException`: Client-side error
- `ServerException`: Server-side error
- `NetworkError`: Network connectivity issue

## Rate Limiting

The MCP server respects KECS API rate limits. If rate limited, tools will automatically retry with exponential backoff.

## Authentication

If KECS requires authentication, set the `KECS_API_TOKEN` environment variable:

```bash
export KECS_API_TOKEN="your-auth-token"
```

The token will be included in the Authorization header for all API requests.