# WebSocket API Reference

## Overview

KECS provides a WebSocket API for real-time updates and bidirectional communication. This allows clients to receive immediate notifications about cluster events, task status changes, and service updates.

## Connection

### Endpoint

```
ws://localhost:8080/ws
wss://localhost:8080/ws  (when TLS is enabled)
```

### Authentication

WebSocket connections can be authenticated using:

1. **Query Parameters**
   ```
   ws://localhost:8080/ws?token=<auth-token>
   ```

2. **Initial Authentication Message**
   ```json
   {
     "type": "auth",
     "token": "your-auth-token"
   }
   ```

### Connection Example

```javascript
// JavaScript/Browser
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('Connected to KECS WebSocket');
  
  // Authenticate
  ws.send(JSON.stringify({
    type: 'auth',
    token: 'your-auth-token'
  }));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('Disconnected from KECS WebSocket');
};
```

```python
# Python
import websocket
import json

def on_message(ws, message):
    data = json.loads(message)
    print(f"Received: {data}")

def on_open(ws):
    print("Connected to KECS WebSocket")
    # Authenticate
    ws.send(json.dumps({
        "type": "auth",
        "token": "your-auth-token"
    }))

def on_error(ws, error):
    print(f"WebSocket error: {error}")

def on_close(ws):
    print("Disconnected from KECS WebSocket")

ws = websocket.WebSocketApp("ws://localhost:8080/ws",
                            on_message=on_message,
                            on_open=on_open,
                            on_error=on_error,
                            on_close=on_close)
ws.run_forever()
```

## Message Format

All messages are JSON-encoded with the following structure:

### Client to Server

```json
{
  "id": "unique-message-id",
  "type": "message-type",
  "action": "action-name",
  "payload": {
    // Action-specific data
  }
}
```

### Server to Client

```json
{
  "id": "message-id",
  "type": "message-type",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    // Event-specific data
  }
}
```

## Subscriptions

### Subscribe to Events

Subscribe to specific event types to receive real-time updates.

#### Subscribe Request

```json
{
  "id": "sub-001",
  "type": "subscribe",
  "action": "events",
  "payload": {
    "eventTypes": ["cluster", "service", "task"],
    "clusters": ["production", "staging"],
    "services": ["web-api"],
    "filters": {
      "status": ["RUNNING", "PENDING"]
    }
  }
}
```

#### Subscribe Response

```json
{
  "id": "sub-001",
  "type": "subscribe_ack",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    "subscriptionId": "sub-001",
    "status": "active"
  }
}
```

### Event Types

Available event types for subscription:

- **cluster**: Cluster lifecycle events
- **service**: Service state changes
- **task**: Task status updates
- **deployment**: Deployment progress
- **scaling**: Auto-scaling events
- **health**: Health check status changes
- **container**: Container state changes

### Unsubscribe

```json
{
  "id": "unsub-001",
  "type": "unsubscribe",
  "action": "events",
  "payload": {
    "subscriptionId": "sub-001"
  }
}
```

## Event Messages

### Cluster Events

```json
{
  "id": "evt-001",
  "type": "event",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    "eventType": "cluster",
    "action": "CREATE|UPDATE|DELETE",
    "cluster": {
      "clusterArn": "arn:aws:ecs:us-east-1:000000000000:cluster/production",
      "clusterName": "production",
      "status": "ACTIVE|PROVISIONING|DEPROVISIONING|FAILED|INACTIVE",
      "registeredContainerInstancesCount": 5,
      "runningTasksCount": 20,
      "pendingTasksCount": 2,
      "activeServicesCount": 3
    }
  }
}
```

### Service Events

```json
{
  "id": "evt-002",
  "type": "event",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    "eventType": "service",
    "action": "CREATE|UPDATE|DELETE|DEPLOYMENT_START|DEPLOYMENT_COMPLETE|DEPLOYMENT_FAILED",
    "service": {
      "serviceArn": "arn:aws:ecs:us-east-1:000000000000:service/production/web-api",
      "serviceName": "web-api",
      "clusterArn": "arn:aws:ecs:us-east-1:000000000000:cluster/production",
      "status": "ACTIVE|DRAINING|INACTIVE",
      "desiredCount": 3,
      "runningCount": 3,
      "pendingCount": 0,
      "deployments": [
        {
          "id": "deploy-001",
          "status": "PRIMARY|ACTIVE|INACTIVE",
          "taskDefinition": "webapp:2",
          "desiredCount": 3,
          "runningCount": 2,
          "pendingCount": 1,
          "rolloutState": "IN_PROGRESS",
          "rolloutStateReason": "Rolling update in progress"
        }
      ]
    }
  }
}
```

### Task Events

```json
{
  "id": "evt-003",
  "type": "event",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    "eventType": "task",
    "action": "START|STOP|STATUS_CHANGE",
    "task": {
      "taskArn": "arn:aws:ecs:us-east-1:000000000000:task/production/1234567890abcdef",
      "clusterArn": "arn:aws:ecs:us-east-1:000000000000:cluster/production",
      "taskDefinitionArn": "arn:aws:ecs:us-east-1:000000000000:task-definition/webapp:1",
      "lastStatus": "PENDING|RUNNING|STOPPED",
      "desiredStatus": "RUNNING|STOPPED",
      "stoppedReason": "Essential container exited",
      "containers": [
        {
          "containerArn": "arn:aws:ecs:us-east-1:000000000000:container/...",
          "name": "webapp",
          "lastStatus": "RUNNING",
          "exitCode": null,
          "reason": null
        }
      ]
    }
  }
}
```

### Deployment Events

```json
{
  "id": "evt-004",
  "type": "event",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    "eventType": "deployment",
    "action": "PROGRESS",
    "deployment": {
      "serviceArn": "arn:aws:ecs:us-east-1:000000000000:service/production/web-api",
      "deploymentId": "deploy-001",
      "status": "IN_PROGRESS",
      "taskDefinition": "webapp:2",
      "desiredCount": 3,
      "runningCount": 2,
      "pendingCount": 1,
      "failedTasks": 0,
      "rolloutState": "IN_PROGRESS",
      "rolloutStateReason": "Rolling update in progress",
      "createdAt": "2024-01-01T00:00:00.000Z",
      "updatedAt": "2024-01-01T00:01:00.000Z"
    }
  }
}
```

## Commands

### Execute Command

Execute commands on running containers (ECS Exec).

#### Request

```json
{
  "id": "cmd-001",
  "type": "command",
  "action": "execute",
  "payload": {
    "cluster": "production",
    "task": "arn:aws:ecs:us-east-1:000000000000:task/production/1234567890abcdef",
    "container": "webapp",
    "command": "/bin/bash",
    "interactive": true
  }
}
```

#### Response

```json
{
  "id": "cmd-001",
  "type": "command_ack",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    "sessionId": "exec-session-001",
    "websocketUrl": "ws://localhost:8080/exec/exec-session-001"
  }
}
```

### Get Metrics

Request real-time metrics for resources.

#### Request

```json
{
  "id": "metrics-001",
  "type": "command",
  "action": "get_metrics",
  "payload": {
    "resource": "service",
    "resourceArn": "arn:aws:ecs:us-east-1:000000000000:service/production/web-api",
    "metrics": ["CPUUtilization", "MemoryUtilization"],
    "period": 60,
    "statistics": ["Average", "Maximum"]
  }
}
```

#### Response

```json
{
  "id": "metrics-001",
  "type": "metrics",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    "resourceArn": "arn:aws:ecs:us-east-1:000000000000:service/production/web-api",
    "metrics": {
      "CPUUtilization": {
        "Average": 45.2,
        "Maximum": 78.5,
        "Unit": "Percent",
        "Datapoints": [
          {
            "Timestamp": "2024-01-01T00:00:00.000Z",
            "Value": 45.2
          }
        ]
      },
      "MemoryUtilization": {
        "Average": 62.8,
        "Maximum": 85.3,
        "Unit": "Percent",
        "Datapoints": [
          {
            "Timestamp": "2024-01-01T00:00:00.000Z",
            "Value": 62.8
          }
        ]
      }
    }
  }
}
```

## Error Handling

### Error Message Format

```json
{
  "id": "original-message-id",
  "type": "error",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    "code": "INVALID_REQUEST",
    "message": "Invalid subscription parameters",
    "details": {
      "field": "eventTypes",
      "reason": "Unknown event type: 'invalid-type'"
    }
  }
}
```

### Error Codes

- **AUTH_FAILED**: Authentication failed
- **INVALID_REQUEST**: Invalid message format or parameters
- **SUBSCRIPTION_ERROR**: Failed to create subscription
- **RESOURCE_NOT_FOUND**: Requested resource not found
- **PERMISSION_DENIED**: Insufficient permissions
- **RATE_LIMIT_EXCEEDED**: Too many requests
- **INTERNAL_ERROR**: Server error

## Connection Management

### Heartbeat/Ping

The server sends periodic ping messages to keep the connection alive.

#### Server Ping

```json
{
  "type": "ping",
  "timestamp": "2024-01-01T00:00:00.000Z"
}
```

#### Client Pong

```json
{
  "type": "pong"
}
```

### Connection Status

#### Connected

```json
{
  "type": "connected",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    "sessionId": "ws-session-001",
    "serverVersion": "1.0.0"
  }
}
```

#### Disconnecting

```json
{
  "type": "disconnecting",
  "timestamp": "2024-01-01T00:00:00.000Z",
  "payload": {
    "reason": "Server shutdown",
    "code": 1001
  }
}
```

## Best Practices

1. **Connection Management**
   - Implement automatic reconnection with exponential backoff
   - Handle connection drops gracefully
   - Clean up subscriptions on disconnect

2. **Message Handling**
   - Always include unique message IDs for request/response correlation
   - Implement timeout handling for commands
   - Buffer messages during reconnection

3. **Subscriptions**
   - Subscribe only to necessary event types
   - Use filters to reduce message volume
   - Unsubscribe when no longer needed

4. **Error Handling**
   - Implement comprehensive error handling
   - Log errors for debugging
   - Provide user-friendly error messages

5. **Performance**
   - Batch subscriptions when possible
   - Implement client-side throttling
   - Use compression for large payloads (when supported)

## Client Libraries

### JavaScript/TypeScript

```typescript
import { KECSWebSocket } from '@kecs/websocket-client';

const client = new KECSWebSocket({
  url: 'ws://localhost:8080/ws',
  token: 'your-auth-token',
  reconnect: true,
  reconnectInterval: 5000
});

client.on('connected', () => {
  console.log('Connected to KECS');
});

client.on('event', (event) => {
  console.log('Received event:', event);
});

client.subscribe({
  eventTypes: ['task'],
  clusters: ['production']
});
```

### Python

```python
from kecs_websocket import KECSWebSocketClient

client = KECSWebSocketClient(
    url='ws://localhost:8080/ws',
    token='your-auth-token',
    auto_reconnect=True
)

@client.on('event')
def handle_event(event):
    print(f"Received event: {event}")

client.subscribe(
    event_types=['task'],
    clusters=['production']
)

client.start()
```

## Rate Limiting

WebSocket connections are subject to rate limiting:

- **Connection Rate**: 10 connections per minute per IP
- **Message Rate**: 100 messages per second per connection
- **Subscription Limit**: 50 active subscriptions per connection

Exceeding rate limits results in:
1. Warning message for first violation
2. Temporary throttling for repeated violations
3. Connection termination for persistent violations