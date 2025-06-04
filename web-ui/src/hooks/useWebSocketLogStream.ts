import { useState, useCallback, useEffect, useRef } from 'react';
import { useWebSocket, useWebSocketSubscription } from './useWebSocket';
import { LogEntry, LogStreamConfig } from '../types/logs';

interface UseWebSocketLogStreamOptions extends Partial<LogStreamConfig> {
  taskId?: string;
  serviceName?: string;
  containerId?: string;
  follow?: boolean;
  batchInterval?: number;
}

interface UseWebSocketLogStreamResult {
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
  setFilter: (filter: LogStreamFilter) => void;
}

interface LogStreamFilter {
  taskIds?: string[];
  serviceNames?: string[];
  containerIds?: string[];
  levels?: string[];
  search?: string;
}

interface LogStreamMessage {
  type: 'log_entry' | 'log_batch' | 'log_history' | 'filter_applied' | 'stream_error';
  payload?: any;
}

export function useWebSocketLogStream(options: UseWebSocketLogStreamOptions = {}): UseWebSocketLogStreamResult {
  const {
    taskId,
    serviceName,
    containerId,
    follow = true,
    maxBufferSize = 10000,
    batchInterval = 100,
    ...streamConfig
  } = options;

  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [isPaused, setIsPaused] = useState(false);
  const [filter, setFilter] = useState<LogStreamFilter>({
    taskIds: taskId ? [taskId] : [],
    serviceNames: serviceName ? [serviceName] : [],
    containerIds: containerId ? [containerId] : [],
  });

  const bufferRef = useRef<LogEntry[]>([]);
  const batchTimerRef = useRef<number | null>(null);

  // Build WebSocket parameters
  const wsParams: Record<string, string> = {
    follow: follow.toString(),
  };
  
  if (taskId) wsParams.taskId = taskId;
  if (serviceName) wsParams.serviceName = serviceName;
  if (containerId) wsParams.containerId = containerId;

  // Initialize WebSocket connection
  const ws = useWebSocket({
    path: '/ws/logs',
    params: wsParams,
    reconnect: true,
    reconnectInterval: 5000,
    maxReconnectAttempts: 10,
    onConnected: () => {
      console.log('Log stream WebSocket connected');
      // Request log history when connected
      ws.send({
        type: 'request_history',
        payload: {
          limit: 1000,
          filter,
        },
      });
    },
    onDisconnected: () => {
      console.log('Log stream WebSocket disconnected');
    },
    onError: (error) => {
      console.error('Log stream WebSocket error:', error);
    },
  });

  // Handle incoming log entries
  const handleLogEntry = useCallback((entry: LogEntry) => {
    if (isPaused) return;

    // Add to buffer
    bufferRef.current.push(entry);

    // Start batch timer if not already running
    if (!batchTimerRef.current) {
      batchTimerRef.current = window.setTimeout(() => {
        flushBuffer();
        batchTimerRef.current = null;
      }, batchInterval);
    }
  }, [isPaused, batchInterval]);

  // Handle batch of log entries
  const handleLogBatch = useCallback((entries: LogEntry[]) => {
    if (isPaused) return;

    setLogs(prevLogs => {
      const newLogs = [...prevLogs, ...entries];
      
      // Limit buffer size
      if (newLogs.length > maxBufferSize) {
        return newLogs.slice(-maxBufferSize);
      }
      
      return newLogs;
    });
  }, [isPaused, maxBufferSize]);

  // Handle log history
  const handleLogHistory = useCallback((entries: LogEntry[]) => {
    // Replace logs with historical entries
    setLogs(entries);
  }, []);

  // Handle stream errors
  const handleStreamError = useCallback((error: any) => {
    console.error('Log stream error:', error);
  }, []);

  // Flush buffer
  const flushBuffer = useCallback(() => {
    if (bufferRef.current.length === 0) return;

    const entries = [...bufferRef.current];
    bufferRef.current = [];

    setLogs(prevLogs => {
      const newLogs = [...prevLogs, ...entries];
      
      // Limit buffer size
      if (newLogs.length > maxBufferSize) {
        return newLogs.slice(-maxBufferSize);
      }
      
      return newLogs;
    });
  }, [maxBufferSize]);

  // Subscribe to WebSocket messages
  useWebSocketSubscription(ws, 'log_entry', handleLogEntry, [handleLogEntry]);
  useWebSocketSubscription(ws, 'log_batch', handleLogBatch, [handleLogBatch]);
  useWebSocketSubscription(ws, 'log_history', handleLogHistory, [handleLogHistory]);
  useWebSocketSubscription(ws, 'stream_error', handleStreamError, [handleStreamError]);

  // Clear logs
  const clear = useCallback(() => {
    setLogs([]);
    bufferRef.current = [];
  }, []);

  // Pause streaming
  const pause = useCallback(() => {
    setIsPaused(true);
    ws.send({
      type: 'pause_stream',
    });
  }, [ws]);

  // Resume streaming
  const resume = useCallback(() => {
    setIsPaused(false);
    ws.send({
      type: 'resume_stream',
    });
  }, [ws]);

  // Update filter
  const updateFilter = useCallback((newFilter: LogStreamFilter) => {
    setFilter(newFilter);
    
    // Send filter update to server
    ws.send({
      type: 'update_filter',
      payload: newFilter,
    });
  }, [ws]);

  // Cleanup batch timer on unmount
  useEffect(() => {
    return () => {
      if (batchTimerRef.current) {
        clearTimeout(batchTimerRef.current);
      }
    };
  }, []);

  return {
    logs,
    isConnected: ws.isConnected,
    isReconnecting: ws.isReconnecting,
    error: ws.error,
    connect: ws.connect,
    disconnect: ws.disconnect,
    clear,
    pause,
    resume,
    isPaused,
    setFilter: updateFilter,
  };
}

// Export type for external use
export type { UseWebSocketLogStreamResult, LogStreamFilter };