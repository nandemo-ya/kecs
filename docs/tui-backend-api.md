# TUI Backend API Documentation

This document describes the REST API endpoints added to the KECS admin server for TUI integration.

## Overview

The TUI backend API provides endpoints for:
- Instance management
- Proxying ECS API calls to specific instances
- CORS support for browser-based clients

All endpoints are served on the admin port (default: 8081).

## Authentication

If an API key is configured, include it in requests:
- Header: `X-API-Key: <api-key>`
- Query parameter: `?api_key=<api-key>`

Health check endpoints do not require authentication.

## Instance Management API

### List Instances
```
GET /api/instances
```

Returns all KECS instances.

**Response:**
```json
[
  {
    "name": "default",
    "status": "running",
    "clusters": 2,
    "services": 5,
    "tasks": 12,
    "apiPort": 8080,
    "adminPort": 8081,
    "createdAt": "2025-01-30T12:00:00Z"
  }
]
```

### Get Instance
```
GET /api/instances/{name}
```

Returns details for a specific instance.

**Response:**
```json
{
  "name": "default",
  "status": "running",
  "clusters": 2,
  "services": 5,
  "tasks": 12,
  "apiPort": 8080,
  "adminPort": 8081,
  "createdAt": "2025-01-30T12:00:00Z"
}
```

### Create Instance
```
POST /api/instances
```

Creates a new KECS instance (currently returns 501 Not Implemented).

**Request Body:**
```json
{
  "name": "dev",
  "apiPort": 8090,
  "adminPort": 8091,
  "localStack": true,
  "traefik": true,
  "devMode": false
}
```

### Delete Instance
```
DELETE /api/instances/{name}
```

Deletes an instance (currently returns 501 Not Implemented for non-default instances).

### Instance Health Check
```
GET /api/instances/{name}/health
```

Returns health status for an instance.

**Response:**
```json
{
  "status": "healthy",
  "version": "v0.0.1-alpha",
  "time": "2025-01-31T12:00:00Z"
}
```

## ECS API Proxy

The admin server proxies ECS API calls to the main API server for the specified instance.

### Generic Proxy Endpoint
```
POST /api/instances/{name}/{endpoint}
```

Proxies requests to the main API server. The endpoint is mapped to ECS actions:

| Endpoint | ECS Action |
|----------|------------|
| clusters | ListClusters |
| clusters/describe | DescribeClusters |
| services | ListServices |
| services/describe | DescribeServices |
| tasks | ListTasks |
| tasks/describe | DescribeTasks |
| tasks/run | RunTask |
| task-definitions | ListTaskDefinitions |
| task-definitions/register | RegisterTaskDefinition |

### Create Cluster
```
POST /api/instances/{name}/clusters
```

### Delete Cluster
```
DELETE /api/instances/{name}/clusters/{cluster}
```

### Delete Service
```
DELETE /api/instances/{name}/services/{service}
```

### Stop Task
```
DELETE /api/instances/{name}/tasks/{task}
```

## CORS Support

CORS headers are automatically added when `Server.AllowedOrigins` is configured:

```yaml
server:
  allowedOrigins:
    - "http://localhost:5173"
    - "http://localhost:3000"
```

Or allow all origins:
```yaml
server:
  allowedOrigins:
    - "*"
```

## Error Responses

All errors follow the AWS ECS error format:

```json
{
  "__type": "ErrorType",
  "message": "Error description"
}
```

Common error types:
- `MethodNotAllowed`: Invalid HTTP method
- `InvalidRequest`: Malformed request
- `InstanceNotFound`: Instance does not exist
- `NotImplemented`: Feature not yet implemented
- `UnauthorizedError`: Invalid or missing API key

## Integration with TUI

The TUI uses these endpoints through the API client:

1. Set the API endpoint:
   ```bash
   export KECS_API_ENDPOINT=http://localhost:8081
   ```

2. Run the TUI:
   ```bash
   ./bin/kecs tui
   ```

The TUI will automatically use the real API instead of mock data.

## Future Enhancements

- WebSocket support for real-time updates
- Multi-instance management
- API key authentication in config
- Metrics and monitoring endpoints