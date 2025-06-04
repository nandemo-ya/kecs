import { useEffect, useState, useCallback, useRef } from 'react';
import { 
  WebSocketService, 
  WebSocketConfig, 
  WebSocketMessage, 
  WebSocketState,
  buildWebSocketUrl 
} from '../services/websocket';

export interface UseWebSocketOptions extends Omit<WebSocketConfig, 'url'> {
  path: string;
  params?: Record<string, string>;
  autoConnect?: boolean;
  onConnected?: () => void;
  onDisconnected?: (event: CloseEvent) => void;
  onError?: (error: Event) => void;
  onMessage?: (message: WebSocketMessage) => void;
  onStateChange?: (state: WebSocketState) => void;
}

export interface UseWebSocketResult {
  state: WebSocketState;
  isConnected: boolean;
  isConnecting: boolean;
  isReconnecting: boolean;
  error: Error | null;
  connect: () => void;
  disconnect: () => void;
  send: (message: WebSocketMessage) => Promise<void>;
  subscribe: (type: string, handler: (data: any) => void) => () => void;
}

export function useWebSocket(options: UseWebSocketOptions): UseWebSocketResult {
  const {
    path,
    params,
    autoConnect = true,
    onConnected,
    onDisconnected,
    onError,
    onMessage,
    onStateChange,
    ...wsConfig
  } = options;

  const [state, setState] = useState<WebSocketState>(WebSocketState.DISCONNECTED);
  const [error, setError] = useState<Error | null>(null);
  const wsRef = useRef<WebSocketService | null>(null);
  const cleanupRef = useRef<(() => void)[]>([]);

  // Build WebSocket URL
  const url = buildWebSocketUrl(path, params);

  // Initialize WebSocket service
  useEffect(() => {
    const ws = new WebSocketService({ ...wsConfig, url });
    wsRef.current = ws;

    // Set up event listeners
    const handleConnected = () => {
      setError(null);
      onConnected?.();
    };
    const handleDisconnected = (event: CloseEvent) => {
      onDisconnected?.(event);
    };
    const handleError = (err: Event) => {
      setError(new Error('WebSocket error'));
      onError?.(err);
    };
    const handleMessage = (message: WebSocketMessage) => {
      onMessage?.(message);
    };
    const handleStateChange = ({ newState }: { newState: WebSocketState }) => {
      setState(newState);
      onStateChange?.(newState);
    };
    const handleMaxReconnectAttempts = () => {
      setError(new Error('Maximum reconnection attempts reached'));
    };

    ws.on('connected', handleConnected);
    ws.on('disconnected', handleDisconnected);
    ws.on('error', handleError);
    ws.on('message', handleMessage);
    ws.on('stateChange', handleStateChange);
    ws.on('maxReconnectAttemptsReached', handleMaxReconnectAttempts);

    cleanupRef.current = [
      () => ws.removeListener('connected', handleConnected),
      () => ws.removeListener('disconnected', handleDisconnected),
      () => ws.removeListener('error', handleError),
      () => ws.removeListener('message', handleMessage),
      () => ws.removeListener('stateChange', handleStateChange),
      () => ws.removeListener('maxReconnectAttemptsReached', handleMaxReconnectAttempts),
    ];

    // Auto-connect if enabled
    if (autoConnect) {
      ws.connect();
    }

    // Cleanup
    return () => {
      cleanupRef.current.forEach(cleanup => cleanup());
      ws.disconnect();
      wsRef.current = null;
    };
  }, [url]); // Only recreate when URL changes

  // Connect method
  const connect = useCallback(() => {
    wsRef.current?.connect();
  }, []);

  // Disconnect method
  const disconnect = useCallback(() => {
    wsRef.current?.disconnect();
  }, []);

  // Send method
  const send = useCallback((message: WebSocketMessage): Promise<void> => {
    if (!wsRef.current) {
      return Promise.reject(new Error('WebSocket not initialized'));
    }
    return wsRef.current.send(message);
  }, []);

  // Subscribe method
  const subscribe = useCallback((type: string, handler: (data: any) => void): (() => void) => {
    if (!wsRef.current) {
      return () => {};
    }
    return wsRef.current.subscribe(type, handler);
  }, []);

  return {
    state,
    isConnected: state === WebSocketState.CONNECTED,
    isConnecting: state === WebSocketState.CONNECTING,
    isReconnecting: state === WebSocketState.RECONNECTING,
    error,
    connect,
    disconnect,
    send,
    subscribe,
  };
}

// Helper hook for subscribing to specific message types
export function useWebSocketSubscription<T = any>(
  ws: UseWebSocketResult,
  type: string,
  handler: (data: T) => void,
  deps: React.DependencyList = []
): void {
  useEffect(() => {
    if (!ws.isConnected) return;
    
    const unsubscribe = ws.subscribe(type, handler);
    return unsubscribe;
  }, [ws.isConnected, type, ...deps]);
}

