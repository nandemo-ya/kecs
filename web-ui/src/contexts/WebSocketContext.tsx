import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { WebSocketService, WebSocketState, initializeWebSocket, buildWebSocketUrl } from '../services/websocket';

interface WebSocketContextValue {
  wsService: WebSocketService | null;
  state: WebSocketState;
  isConnected: boolean;
  error: Error | null;
}

const WebSocketContext = createContext<WebSocketContextValue>({
  wsService: null,
  state: WebSocketState.DISCONNECTED,
  isConnected: false,
  error: null,
});

interface WebSocketProviderProps {
  children: ReactNode;
  autoConnect?: boolean;
}

export function WebSocketProvider({ children, autoConnect = true }: WebSocketProviderProps) {
  const [wsService, setWsService] = useState<WebSocketService | null>(null);
  const [state, setState] = useState<WebSocketState>(WebSocketState.DISCONNECTED);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    // Initialize WebSocket service
    const url = buildWebSocketUrl('/ws');
    const service = initializeWebSocket({
      url,
      reconnect: true,
      reconnectInterval: 5000,
      maxReconnectAttempts: 10,
      heartbeatInterval: 30000,
    });

    setWsService(service);

    // Listen to state changes
    const handleStateChange = ({ newState }: { newState: WebSocketState }) => {
      setState(newState);
    };

    const handleError = (err: any) => {
      setError(new Error(err.message || 'WebSocket error'));
    };

    const handleConnected = () => {
      setError(null);
    };

    service.on('stateChange', handleStateChange);
    service.on('error', handleError);
    service.on('connected', handleConnected);

    // Auto-connect if enabled
    if (autoConnect) {
      service.connect();
    }

    // Cleanup
    return () => {
      service.removeListener('stateChange', handleStateChange);
      service.removeListener('error', handleError);
      service.removeListener('connected', handleConnected);
      service.disconnect();
    };
  }, [autoConnect]);

  const value: WebSocketContextValue = {
    wsService,
    state,
    isConnected: state === WebSocketState.CONNECTED,
    error,
  };

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocketContext() {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocketContext must be used within a WebSocketProvider');
  }
  return context;
}