# API Reference

KECS implements the Amazon ECS API specification, providing full compatibility with existing ECS tools and SDKs.

## Overview

All API requests follow the AWS API conventions:

- **Endpoint**: `http://localhost:8080/v1/<Action>`
- **Method**: POST
- **Content-Type**: `application/x-amz-json-1.1`
- **Target Header**: `X-Amz-Target: AmazonEC2ContainerServiceV20141113.<Action>`

## Authentication

Currently, KECS does not require authentication for local development. In production deployments, you can configure authentication through:

- API Keys
- JWT Tokens
- mTLS

See [Authentication Guide](/api/authentication) for details.

## Available APIs

### Cluster Management
- [CreateCluster](/api/clusters#createcluster)
- [DeleteCluster](/api/clusters#deletecluster)
- [DescribeClusters](/api/clusters#describeclusters)
- [ListClusters](/api/clusters#listclusters)
- [UpdateCluster](/api/clusters#updatecluster)

### Service Management
- [CreateService](/api/services#createservice)
- [DeleteService](/api/services#deleteservice)
- [DescribeServices](/api/services#describeservices)
- [ListServices](/api/services#listservices)
- [UpdateService](/api/services#updateservice)

### Task Management
- [RunTask](/api/tasks#runtask)
- [StopTask](/api/tasks#stoptask)
- [DescribeTasks](/api/tasks#describetasks)
- [ListTasks](/api/tasks#listtasks)

### Task Definition Management
- [RegisterTaskDefinition](/api/task-definitions#registertaskdefinition)
- [DeregisterTaskDefinition](/api/task-definitions#deregistertaskdefinition)
- [DescribeTaskDefinition](/api/task-definitions#describetaskdefinition)
- [ListTaskDefinitions](/api/task-definitions#listtaskdefinitions)

## Example Request

```bash
curl -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -d '{
    "maxResults": 10
  }'
```

## Response Format

All responses follow the standard AWS API response format:

```json
{
  "clusterArns": [
    "arn:aws:ecs:us-east-1:123456789012:cluster/default"
  ],
  "nextToken": null
}
```

## Error Handling

Errors are returned with appropriate HTTP status codes and error details:

```json
{
  "__type": "ClientException",
  "message": "Cluster not found"
}
```

Common error types:
- `ClientException`: Client-side errors (400)
- `ServerException`: Server-side errors (500)
- `ResourceNotFoundException`: Resource not found (404)
- `InvalidParameterException`: Invalid parameters (400)

## SDK Usage

KECS is compatible with AWS SDKs. Configure the endpoint:

### AWS CLI
```bash
aws ecs list-clusters --endpoint-url http://localhost:8080
```

### Python (boto3)
```python
import boto3

ecs = boto3.client('ecs', endpoint_url='http://localhost:8080')
clusters = ecs.list_clusters()
```

### Go SDK
```go
import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/ecs"
)

sess := session.Must(session.NewSession(&aws.Config{
    Endpoint: aws.String("http://localhost:8080"),
}))

svc := ecs.New(sess)
```

## Rate Limiting

KECS implements rate limiting to prevent abuse:
- Default: 100 requests per second per IP
- Configurable via `--rate-limit` flag

## WebSocket API

For real-time updates, KECS provides a WebSocket endpoint:
- **Endpoint**: `ws://localhost:8080/ws`
- **Protocol**: JSON messages
- See [WebSocket Guide](/api/websocket) for details