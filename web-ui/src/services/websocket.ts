import { EventEmitter } from '../utils/EventEmitter';

export interface WebSocketConfig {
  url: string;
  protocols?: string | string[];
  reconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  heartbeatInterval?: number;
  messageTimeout?: number;
}

export interface WebSocketMessage {
  type: string;
  payload?: any;
  id?: string;
  timestamp?: Date;
}

export enum WebSocketState {
  CONNECTING = 'CONNECTING',
  CONNECTED = 'CONNECTED',
  DISCONNECTED = 'DISCONNECTED',
  RECONNECTING = 'RECONNECTING',
  ERROR = 'ERROR',
}

interface InternalWebSocketConfig {
  url: string;
  protocols?: string | string[];
  reconnect: boolean;
  reconnectInterval: number;
  maxReconnectAttempts: number;
  heartbeatInterval: number;
  messageTimeout: number;
}

export class WebSocketService extends EventEmitter {
  private ws: WebSocket | null = null;
  private config: InternalWebSocketConfig;
  private reconnectAttempts = 0;
  private reconnectTimer: number | null = null;
  private heartbeatTimer: number | null = null;
  private messageHandlers = new Map<string, Set<(data: any) => void>>();
  private pendingMessages: WebSocketMessage[] = [];
  private state: WebSocketState = WebSocketState.DISCONNECTED;
  private lastPongTime: number = Date.now();

  constructor(config: WebSocketConfig) {
    super();
    this.config = {
      reconnect: true,
      reconnectInterval: 5000,
      maxReconnectAttempts: 10,
      heartbeatInterval: 30000,
      messageTimeout: 10000,
      ...config,
    };
  }

  connect(): void {
    if (this.state === WebSocketState.CONNECTED || this.state === WebSocketState.CONNECTING) {
      return;
    }

    this.setState(WebSocketState.CONNECTING);

    try {
      this.ws = new WebSocket(this.config.url, this.config.protocols);
      this.setupEventHandlers();
    } catch (error) {
      console.error('Failed to create WebSocket:', error);
      this.setState(WebSocketState.ERROR);
      this.handleReconnect();
    }
  }

  disconnect(): void {
    this.clearTimers();
    this.reconnectAttempts = 0;
    
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect');
      this.ws = null;
    }
    
    this.setState(WebSocketState.DISCONNECTED);
  }

  send(message: WebSocketMessage): Promise<void> {
    return new Promise((resolve, reject) => {
      if (!message.id) {
        message.id = this.generateMessageId();
      }
      
      if (!message.timestamp) {
        message.timestamp = new Date();
      }

      if (this.state !== WebSocketState.CONNECTED || !this.ws || this.ws.readyState !== WebSocket.OPEN) {
        // Queue message for later
        this.pendingMessages.push(message);
        resolve();
        return;
      }

      try {
        this.ws.send(JSON.stringify(message));
        resolve();
      } catch (error) {
        reject(error);
      }
    });
  }

  subscribe(type: string, handler: (data: any) => void): () => void {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, new Set());
    }
    
    this.messageHandlers.get(type)!.add(handler);
    
    // Return unsubscribe function
    return () => {
      const handlers = this.messageHandlers.get(type);
      if (handlers) {
        handlers.delete(handler);
        if (handlers.size === 0) {
          this.messageHandlers.delete(type);
        }
      }
    };
  }

  getState(): WebSocketState {
    return this.state;
  }

  isConnected(): boolean {
    return this.state === WebSocketState.CONNECTED;
  }

  private setupEventHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      console.log('WebSocket connected');
      this.setState(WebSocketState.CONNECTED);
      this.reconnectAttempts = 0;
      this.startHeartbeat();
      this.flushPendingMessages();
      this.emit('connected');
    };

    this.ws.onclose = (event) => {
      console.log('WebSocket closed:', event.code, event.reason);
      this.setState(WebSocketState.DISCONNECTED);
      this.clearTimers();
      this.emit('disconnected', event);
      
      if (this.config.reconnect && !event.wasClean) {
        this.handleReconnect();
      }
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      this.emit('error', error);
    };

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as WebSocketMessage;
        this.handleMessage(message);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };
  }

  private handleMessage(message: WebSocketMessage): void {
    // Handle heartbeat
    if (message.type === 'pong') {
      this.lastPongTime = Date.now();
      return;
    }

    // Emit raw message event
    this.emit('message', message);

    // Call type-specific handlers
    const handlers = this.messageHandlers.get(message.type);
    if (handlers) {
      handlers.forEach(handler => {
        try {
          handler(message.payload);
        } catch (error) {
          console.error(`Error in message handler for type ${message.type}:`, error);
        }
      });
    }
  }

  private startHeartbeat(): void {
    this.clearHeartbeat();
    
    this.heartbeatTimer = window.setInterval(() => {
      if (this.state !== WebSocketState.CONNECTED) {
        this.clearHeartbeat();
        return;
      }

      // Check if we've received a pong recently
      const timeSinceLastPong = Date.now() - this.lastPongTime;
      if (timeSinceLastPong > this.config.heartbeatInterval * 2) {
        console.warn('No pong received, reconnecting...');
        this.ws?.close();
        return;
      }

      // Send ping
      this.send({ type: 'ping' }).catch(console.error);
    }, this.config.heartbeatInterval);
  }

  private clearHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
  }

  private handleReconnect(): void {
    if (!this.config.reconnect || this.reconnectAttempts >= this.config.maxReconnectAttempts) {
      this.setState(WebSocketState.ERROR);
      this.emit('maxReconnectAttemptsReached');
      return;
    }

    this.reconnectAttempts++;
    this.setState(WebSocketState.RECONNECTING);
    
    const delay = Math.min(
      this.config.reconnectInterval * Math.pow(1.5, this.reconnectAttempts - 1),
      30000
    );

    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.config.maxReconnectAttempts})`);
    
    this.reconnectTimer = window.setTimeout(() => {
      this.connect();
    }, delay);
  }

  private flushPendingMessages(): void {
    while (this.pendingMessages.length > 0) {
      const message = this.pendingMessages.shift();
      if (message) {
        this.send(message).catch(console.error);
      }
    }
  }

  private clearTimers(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    
    this.clearHeartbeat();
  }

  private setState(state: WebSocketState): void {
    if (this.state !== state) {
      const oldState = this.state;
      this.state = state;
      this.emit('stateChange', { oldState, newState: state });
    }
  }

  private generateMessageId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }
}

// Singleton instance management
let defaultInstance: WebSocketService | null = null;

export function initializeWebSocket(config: WebSocketConfig): WebSocketService {
  if (defaultInstance) {
    defaultInstance.disconnect();
  }
  
  defaultInstance = new WebSocketService(config);
  return defaultInstance;
}

export function getWebSocketInstance(): WebSocketService | null {
  return defaultInstance;
}

// WebSocket URL builder
export function buildWebSocketUrl(path: string, params?: Record<string, string>): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const host = process.env.REACT_APP_WS_HOST || window.location.host;
  
  let url = `${protocol}//${host}${path}`;
  
  if (params) {
    const queryString = new URLSearchParams(params).toString();
    if (queryString) {
      url += `?${queryString}`;
    }
  }
  
  return url;
}