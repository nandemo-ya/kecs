# WebSocket Integration Guide

This document describes the WebSocket integration in the KECS Web UI for real-time communication.

## Overview

The WebSocket integration provides real-time bidirectional communication between the web UI and the KECS backend, enabling features like:

- Live log streaming
- Real-time metrics updates
- Task status notifications
- System alerts and notifications
- Interactive debugging

## Architecture

### Core Components

1. **WebSocket Service** (`src/services/websocket.ts`)
   - Manages WebSocket connections
   - Handles reconnection logic
   - Provides message queuing
   - Implements heartbeat mechanism

2. **React Hooks**
   - `useWebSocket` - Base hook for WebSocket connections
   - `useWebSocketLogStream` - Specialized hook for log streaming
   - `useWebSocketMetrics` - Real-time metrics updates
   - `useWebSocketNotifications` - System notifications
   - `useWebSocketTaskUpdates` - Task status updates

3. **WebSocket Provider** (`src/contexts/WebSocketContext.tsx`)
   - Global WebSocket connection management
   - Shared connection state

## Usage Examples

### Basic WebSocket Connection

```typescript
import { useWebSocket } from '../hooks/useWebSocket';

function MyComponent() {
  const ws = useWebSocket({
    path: '/ws/endpoint',
    autoConnect: true,
    onMessage: (message) => {
      console.log('Received:', message);
    },
  });

  const sendMessage = () => {
    ws.send({
      type: 'my_message',
      payload: { data: 'Hello' },
    });
  };

  return (
    <div>
      <p>Status: {ws.isConnected ? 'Connected' : 'Disconnected'}</p>
      <button onClick={sendMessage}>Send Message</button>
    </div>
  );
}
```

### Log Streaming

```typescript
import { useWebSocketLogStream } from '../hooks/useWebSocketLogStream';

function LogViewer({ taskId }) {
  const {
    logs,
    isConnected,
    pause,
    resume,
    clear,
  } = useWebSocketLogStream({
    taskId,
    follow: true,
    maxBufferSize: 1000,
  });

  return (
    <div>
      {logs.map(log => (
        <div key={log.id}>
          [{log.timestamp}] {log.level}: {log.message}
        </div>
      ))}
    </div>
  );
}
```

### Real-time Metrics

```typescript
import { useWebSocketMetrics } from '../hooks/useWebSocketMetrics';

function MetricsDisplay({ taskIds }) {
  const { metrics } = useWebSocketMetrics({
    taskIds,
    metrics: ['cpu', 'memory'],
    interval: 5000,
  });

  return (
    <div>
      {metrics.map(resource => (
        <div key={resource.taskId}>
          <h3>{resource.taskId}</h3>
          <p>CPU: {resource.cpu.points[0]?.value}%</p>
          <p>Memory: {resource.memory.points[0]?.value} MB</p>
        </div>
      ))}
    </div>
  );
}
```

### Notifications

```typescript
import { useWebSocketNotifications } from '../hooks/useWebSocketNotifications';

function NotificationCenter() {
  const {
    notifications,
    unreadCount,
    markAsRead,
    dismiss,
  } = useWebSocketNotifications({
    maxNotifications: 50,
  });

  return (
    <div>
      <h2>Notifications ({unreadCount})</h2>
      {notifications.map(notif => (
        <div key={notif.id}>
          <h4>{notif.title}</h4>
          <p>{notif.message}</p>
          <button onClick={() => markAsRead(notif.id)}>
            Mark as Read
          </button>
        </div>
      ))}
    </div>
  );
}
```

## WebSocket Protocol

### Message Format

All WebSocket messages follow this format:

```typescript
interface WebSocketMessage {
  type: string;          // Message type identifier
  payload?: any;         // Message data
  id?: string;          // Unique message ID
  timestamp?: Date;     // Message timestamp
}
```

### Common Message Types

#### Client to Server

- `ping` - Heartbeat message
- `subscribe` - Subscribe to updates
- `unsubscribe` - Unsubscribe from updates
- `request_history` - Request historical data
- `update_filter` - Update data filters

#### Server to Client

- `pong` - Heartbeat response
- `log_entry` - Single log entry
- `log_batch` - Batch of log entries
- `metric_update` - Metric data update
- `notification` - System notification
- `task_update` - Task status change

## Features

### Auto-Reconnection

The WebSocket service automatically attempts to reconnect when the connection is lost:

- Exponential backoff strategy
- Maximum retry attempts configurable
- Connection state tracking

### Message Queuing

Messages sent while disconnected are queued and sent when the connection is restored.

### Heartbeat

Regular ping/pong messages ensure the connection is alive and detect disconnections quickly.

### Error Handling

Comprehensive error handling with:
- Connection errors
- Message parsing errors
- Handler errors
- Timeout handling

## Configuration

### Environment Variables

```bash
REACT_APP_WS_HOST=localhost:8080  # WebSocket server host
```

### Connection Options

```typescript
interface WebSocketConfig {
  url: string;
  protocols?: string | string[];
  reconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  heartbeatInterval?: number;
  messageTimeout?: number;
}
```

## Demo

Access the WebSocket demo at `/websocket-demo` to see all features in action:

- Connection management
- Real-time messaging
- Notifications
- Metrics updates
- Task updates

## Security Considerations

1. **Authentication**: WebSocket connections should be authenticated
2. **Message Validation**: All incoming messages should be validated
3. **Rate Limiting**: Implement rate limiting to prevent abuse
4. **Encryption**: Use WSS (WebSocket Secure) in production

## Performance Tips

1. **Batch Updates**: Group multiple updates into single messages
2. **Throttling**: Limit update frequency for high-volume data
3. **Selective Subscription**: Only subscribe to needed data
4. **Buffer Management**: Set appropriate buffer sizes

## Troubleshooting

### Connection Issues

1. Check browser console for errors
2. Verify WebSocket endpoint URL
3. Check for proxy/firewall issues
4. Ensure backend WebSocket support

### Message Issues

1. Verify message format
2. Check for parsing errors
3. Ensure handlers are registered
4. Check subscription status

## Future Enhancements

- Binary message support
- Compression
- Multi-channel support
- Offline message persistence
- Advanced filtering options