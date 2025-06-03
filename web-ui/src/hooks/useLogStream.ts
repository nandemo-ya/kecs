import { useState, useEffect, useRef, useCallback } from 'react';
import { LogEntry, LogStreamConfig, LogStreamMessage } from '../types/logs';

interface UseLogStreamOptions extends Partial<LogStreamConfig> {
  url?: string;
  onError?: (error: Error) => void;
  onConnect?: () => void;
  onDisconnect?: () => void;
}

interface UseLogStreamResult {
  logs: LogEntry[];
  isConnected: boolean;
  isReconnecting: boolean;
  error: Error | null;
  connect: () => void;
  disconnect: () => void;
  clear: () => void;
  pause: () => void;
  resume: () => void;
  isPaused: boolean;
}

export function useLogStream(options: UseLogStreamOptions = {}): UseLogStreamResult {
  const {
    enabled = true,
    maxBufferSize = 1000,
    reconnectInterval = 5000,
    heartbeatInterval = 30000,
    compression = false,
    url = '/api/logs/stream',
    onError,
    onConnect,
    onDisconnect,
  } = options;

  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [isReconnecting, setIsReconnecting] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [isPaused, setIsPaused] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const heartbeatIntervalRef = useRef<number | null>(null);
  const logsBufferRef = useRef<LogEntry[]>([]);

  // Clear function
  const clear = useCallback(() => {
    setLogs([]);
    logsBufferRef.current = [];
  }, []);

  // Pause/Resume functions
  const pause = useCallback(() => {
    setIsPaused(true);
  }, []);

  const resume = useCallback(() => {
    setIsPaused(false);
    // Flush buffered logs
    if (logsBufferRef.current.length > 0) {
      setLogs(prev => {
        const combined = [...prev, ...logsBufferRef.current];
        // Trim to max buffer size
        if (combined.length > maxBufferSize) {
          return combined.slice(-maxBufferSize);
        }
        return combined;
      });
      logsBufferRef.current = [];
    }
  }, [maxBufferSize]);

  // Add log entry
  const addLogEntry = useCallback((entry: LogEntry) => {
    if (isPaused) {
      // Buffer logs when paused
      logsBufferRef.current.push(entry);
      if (logsBufferRef.current.length > maxBufferSize) {
        logsBufferRef.current = logsBufferRef.current.slice(-maxBufferSize);
      }
    } else {
      setLogs(prev => {
        const newLogs = [...prev, entry];
        // Trim to max buffer size
        if (newLogs.length > maxBufferSize) {
          return newLogs.slice(-maxBufferSize);
        }
        return newLogs;
      });
    }
  }, [isPaused, maxBufferSize]);

  // WebSocket message handler
  const handleMessage = useCallback((event: MessageEvent) => {
    try {
      const message: LogStreamMessage = JSON.parse(event.data);
      
      switch (message.type) {
        case 'log':
          if (message.data) {
            const logEntry: LogEntry = {
              ...message.data,
              timestamp: new Date(message.data.timestamp),
            };
            addLogEntry(logEntry);
          }
          break;
        
        case 'heartbeat':
          // Heartbeat received, connection is alive
          break;
        
        case 'error':
          setError(new Error(message.data?.message || 'Unknown error'));
          if (onError) {
            onError(new Error(message.data?.message || 'Unknown error'));
          }
          break;
        
        case 'config':
          // Configuration update from server
          console.log('Received config update:', message.data);
          break;
      }
    } catch (err) {
      console.error('Failed to parse log message:', err);
      setError(err as Error);
      if (onError) {
        onError(err as Error);
      }
    }
  }, [addLogEntry, onError]);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      // Construct WebSocket URL
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}${url}`;
      
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        setIsConnected(true);
        setIsReconnecting(false);
        setError(null);
        
        // Start heartbeat
        heartbeatIntervalRef.current = window.setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'heartbeat', timestamp: Date.now() }));
          }
        }, heartbeatInterval);

        if (onConnect) {
          onConnect();
        }
      };

      ws.onmessage = handleMessage;

      ws.onerror = (event) => {
        const error = new Error('WebSocket error');
        setError(error);
        if (onError) {
          onError(error);
        }
      };

      ws.onclose = () => {
        setIsConnected(false);
        wsRef.current = null;

        // Clear heartbeat
        if (heartbeatIntervalRef.current) {
          clearInterval(heartbeatIntervalRef.current);
          heartbeatIntervalRef.current = null;
        }

        if (onDisconnect) {
          onDisconnect();
        }

        // Attempt reconnection if enabled
        if (enabled && !reconnectTimeoutRef.current) {
          setIsReconnecting(true);
          reconnectTimeoutRef.current = window.setTimeout(() => {
            reconnectTimeoutRef.current = null;
            connect();
          }, reconnectInterval);
        }
      };
    } catch (err) {
      setError(err as Error);
      if (onError) {
        onError(err as Error);
      }
    }
  }, [url, enabled, reconnectInterval, heartbeatInterval, handleMessage, onConnect, onDisconnect, onError]);

  // Disconnect from WebSocket
  const disconnect = useCallback(() => {
    // Clear reconnection timeout
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    // Clear heartbeat interval
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
      heartbeatIntervalRef.current = null;
    }

    // Close WebSocket
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setIsConnected(false);
    setIsReconnecting(false);
  }, []);

  // Effect to manage connection
  useEffect(() => {
    if (enabled) {
      connect();
    } else {
      disconnect();
    }

    return () => {
      disconnect();
    };
  }, [enabled, connect, disconnect]);

  return {
    logs,
    isConnected,
    isReconnecting,
    error,
    connect,
    disconnect,
    clear,
    pause,
    resume,
    isPaused,
  };
}

// Mock log stream for development
export function useMockLogStream(options: UseLogStreamOptions = {}): UseLogStreamResult {
  const {
    maxBufferSize = 1000,
    onConnect,
    onDisconnect,
  } = options;

  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [isPaused, setIsPaused] = useState(false);
  const intervalRef = useRef<number | null>(null);

  const logTemplates = [
    { level: 'info', message: 'Container started successfully' },
    { level: 'debug', message: 'Health check passed' },
    { level: 'info', message: 'Processing request from client' },
    { level: 'warn', message: 'High memory usage detected' },
    { level: 'error', message: 'Failed to connect to database' },
    { level: 'info', message: 'Request completed successfully' },
    { level: 'debug', message: 'Cache hit for key: user_123' },
    { level: 'info', message: 'Scaling up to handle increased load' },
    { level: 'warn', message: 'Slow query detected: SELECT * FROM large_table' },
    { level: 'info', message: 'Deployment completed' },
  ];

  const sources = ['web-service', 'api-gateway', 'worker-service', 'database-proxy', 'cache-service'];

  const generateMockLog = (): LogEntry => {
    const template = logTemplates[Math.floor(Math.random() * logTemplates.length)];
    const source = sources[Math.floor(Math.random() * sources.length)];
    
    return {
      id: `log-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      timestamp: new Date(),
      level: template.level as any,
      source: {
        type: 'container',
        name: source,
        identifier: `${source}-${Math.floor(Math.random() * 10)}`,
      },
      message: template.message,
      taskId: `task-${Math.floor(Math.random() * 100)}`,
      serviceName: source,
      containerId: `container-${Math.random().toString(36).substr(2, 9)}`,
      metadata: {
        region: 'us-east-1',
        cluster: 'production',
        version: '1.0.0',
      },
    };
  };

  const connect = useCallback(() => {
    setIsConnected(true);
    if (onConnect) {
      onConnect();
    }

    // Start generating mock logs
    intervalRef.current = window.setInterval(() => {
      if (!isPaused) {
        const newLog = generateMockLog();
        setLogs(prev => {
          const updated = [...prev, newLog];
          if (updated.length > maxBufferSize) {
            return updated.slice(-maxBufferSize);
          }
          return updated;
        });
      }
    }, Math.random() * 2000 + 500); // Random interval between 0.5s and 2.5s
  }, [isPaused, maxBufferSize, onConnect]);

  const disconnect = useCallback(() => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
    setIsConnected(false);
    if (onDisconnect) {
      onDisconnect();
    }
  }, [onDisconnect]);

  const clear = useCallback(() => {
    setLogs([]);
  }, []);

  const pause = useCallback(() => {
    setIsPaused(true);
  }, []);

  const resume = useCallback(() => {
    setIsPaused(false);
  }, []);

  useEffect(() => {
    connect();
    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  return {
    logs,
    isConnected,
    isReconnecting: false,
    error: null,
    connect,
    disconnect,
    clear,
    pause,
    resume,
    isPaused,
  };
}