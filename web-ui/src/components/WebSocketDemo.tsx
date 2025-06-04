import React, { useState } from 'react';
import { useWebSocket } from '../hooks/useWebSocket';
import { useWebSocketNotifications } from '../hooks/useWebSocketNotifications';
import { useWebSocketMetrics } from '../hooks/useWebSocketMetrics';
import { useWebSocketTaskUpdates } from '../hooks/useWebSocketTaskUpdates';

export function WebSocketDemo() {
  const [message, setMessage] = useState('');
  const [responses, setResponses] = useState<string[]>([]);

  // Basic WebSocket connection
  const ws = useWebSocket({
    path: '/ws/demo',
    autoConnect: true,
    onMessage: (msg) => {
      setResponses(prev => [...prev, `Received: ${JSON.stringify(msg)}`]);
    },
  });

  // Notifications
  const notifications = useWebSocketNotifications({
    maxNotifications: 50,
  });

  // Metrics
  const metrics = useWebSocketMetrics({
    metrics: ['cpu', 'memory'],
    interval: 5000,
  });

  // Task updates
  const taskUpdates = useWebSocketTaskUpdates({
    autoSubscribe: true,
  });

  const sendMessage = () => {
    if (message.trim()) {
      ws.send({
        type: 'message',
        payload: { text: message },
      });
      setMessage('');
    }
  };

  return (
    <div style={{ padding: '20px', fontFamily: 'Arial, sans-serif' }}>
      <h1>WebSocket Integration Demo</h1>

      {/* Connection Status */}
      <section style={{ marginBottom: '30px' }}>
        <h2>Connection Status</h2>
        <div style={{ 
          display: 'inline-flex', 
          alignItems: 'center', 
          gap: '10px',
          padding: '10px',
          background: ws.isConnected ? '#d4edda' : ws.isReconnecting ? '#fff3cd' : '#f8d7da',
          borderRadius: '5px',
        }}>
          <div style={{
            width: '10px',
            height: '10px',
            borderRadius: '50%',
            background: ws.isConnected ? '#28a745' : ws.isReconnecting ? '#ffc107' : '#dc3545',
          }} />
          <span>
            {ws.isConnected ? 'Connected' : ws.isReconnecting ? 'Reconnecting...' : 'Disconnected'}
          </span>
        </div>
        {ws.error && (
          <div style={{ color: '#dc3545', marginTop: '10px' }}>
            Error: {ws.error.message}
          </div>
        )}
      </section>

      {/* Basic Messaging */}
      <section style={{ marginBottom: '30px' }}>
        <h2>Basic Messaging</h2>
        <div style={{ display: 'flex', gap: '10px', marginBottom: '10px' }}>
          <input
            type="text"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && sendMessage()}
            placeholder="Type a message..."
            style={{ 
              flex: 1, 
              padding: '8px', 
              border: '1px solid #ddd',
              borderRadius: '4px',
            }}
          />
          <button 
            onClick={sendMessage}
            disabled={!ws.isConnected}
            style={{
              padding: '8px 16px',
              background: ws.isConnected ? '#007bff' : '#6c757d',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: ws.isConnected ? 'pointer' : 'not-allowed',
            }}
          >
            Send
          </button>
        </div>
        <div style={{
          height: '150px',
          overflow: 'auto',
          border: '1px solid #ddd',
          borderRadius: '4px',
          padding: '10px',
          background: '#f8f9fa',
        }}>
          {responses.length === 0 ? (
            <div style={{ color: '#6c757d' }}>No messages yet...</div>
          ) : (
            responses.map((resp, idx) => (
              <div key={idx} style={{ marginBottom: '5px' }}>{resp}</div>
            ))
          )}
        </div>
      </section>

      {/* Notifications */}
      <section style={{ marginBottom: '30px' }}>
        <h2>Notifications ({notifications.unreadCount} unread)</h2>
        {notifications.notifications.length === 0 ? (
          <div style={{ color: '#6c757d' }}>No notifications</div>
        ) : (
          <div style={{ maxHeight: '200px', overflow: 'auto' }}>
            {notifications.notifications.slice(0, 5).map(notif => (
              <div 
                key={notif.id}
                style={{
                  padding: '10px',
                  marginBottom: '5px',
                  background: notif.read ? '#f8f9fa' : '#e3f2fd',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                }}
              >
                <div>
                  <strong>{notif.title}</strong>
                  <div style={{ fontSize: '14px', color: '#666' }}>{notif.message}</div>
                  <div style={{ fontSize: '12px', color: '#999' }}>
                    {new Date(notif.timestamp).toLocaleTimeString()}
                  </div>
                </div>
                {!notif.read && (
                  <button
                    onClick={() => notifications.markAsRead(notif.id)}
                    style={{
                      padding: '4px 8px',
                      background: '#28a745',
                      color: 'white',
                      border: 'none',
                      borderRadius: '4px',
                      fontSize: '12px',
                      cursor: 'pointer',
                    }}
                  >
                    Mark Read
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Metrics */}
      <section style={{ marginBottom: '30px' }}>
        <h2>Resource Metrics</h2>
        {metrics.metrics.length === 0 ? (
          <div style={{ color: '#6c757d' }}>No metrics data</div>
        ) : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', gap: '10px' }}>
            {metrics.metrics.map((resource, idx) => (
              <div 
                key={idx}
                style={{
                  padding: '15px',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  background: '#f8f9fa',
                }}
              >
                <h4 style={{ margin: '0 0 10px 0', color: '#333' }}>
                  {resource.taskId || resource.serviceName || resource.containerId}
                </h4>
                <div style={{ marginBottom: '5px' }}>
                  <span style={{ fontWeight: 'bold' }}>CPU:</span>{' '}
                  {resource.cpu.points.length > 0 
                    ? `${resource.cpu.points[resource.cpu.points.length - 1].value.toFixed(2)}%`
                    : 'N/A'
                  }
                </div>
                <div>
                  <span style={{ fontWeight: 'bold' }}>Memory:</span>{' '}
                  {resource.memory.points.length > 0
                    ? `${(resource.memory.points[resource.memory.points.length - 1].value / 1024 / 1024).toFixed(2)} MB`
                    : 'N/A'
                  }
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Task Updates */}
      <section>
        <h2>Task Updates ({taskUpdates.tasks.size} tasks)</h2>
        {taskUpdates.updates.length === 0 ? (
          <div style={{ color: '#6c757d' }}>No task updates</div>
        ) : (
          <div style={{ maxHeight: '200px', overflow: 'auto' }}>
            {taskUpdates.updates.slice(0, 10).map((update, idx) => (
              <div 
                key={idx}
                style={{
                  padding: '8px',
                  marginBottom: '5px',
                  background: '#f8f9fa',
                  border: '1px solid #ddd',
                  borderRadius: '4px',
                  fontSize: '14px',
                }}
              >
                <span style={{ 
                  color: update.updateType === 'created' ? '#28a745' 
                    : update.updateType === 'deleted' ? '#dc3545' 
                    : '#007bff' 
                }}>
                  {update.updateType.toUpperCase()}
                </span>
                {' - '}
                <span style={{ fontFamily: 'monospace' }}>
                  {update.taskId.split('/').pop()}
                </span>
                {update.previousStatus && update.task && (
                  <span style={{ color: '#666' }}>
                    {' '}({update.previousStatus} → {update.task.lastStatus})
                  </span>
                )}
                <span style={{ float: 'right', color: '#999', fontSize: '12px' }}>
                  {new Date(update.timestamp).toLocaleTimeString()}
                </span>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Connection Controls */}
      <div style={{ marginTop: '30px', display: 'flex', gap: '10px' }}>
        <button
          onClick={() => ws.connect()}
          disabled={ws.isConnected}
          style={{
            padding: '8px 16px',
            background: !ws.isConnected ? '#28a745' : '#6c757d',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: !ws.isConnected ? 'pointer' : 'not-allowed',
          }}
        >
          Connect
        </button>
        <button
          onClick={() => ws.disconnect()}
          disabled={!ws.isConnected}
          style={{
            padding: '8px 16px',
            background: ws.isConnected ? '#dc3545' : '#6c757d',
            color: 'white',
            border: 'none',
            borderRadius: '4px',
            cursor: ws.isConnected ? 'pointer' : 'not-allowed',
          }}
        >
          Disconnect
        </button>
      </div>
    </div>
  );
}