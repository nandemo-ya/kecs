import { useState, useCallback, useEffect } from 'react';
import { useWebSocket, useWebSocketSubscription } from './useWebSocket';

export type NotificationType = 'info' | 'success' | 'warning' | 'error';
export type NotificationPriority = 'low' | 'normal' | 'high' | 'critical';

export interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  message: string;
  timestamp: Date;
  priority?: NotificationPriority;
  source?: string;
  metadata?: Record<string, any>;
  actions?: NotificationAction[];
  read?: boolean;
  expires?: Date;
}

export interface NotificationAction {
  label: string;
  action: string;
  primary?: boolean;
  destructive?: boolean;
}

interface UseWebSocketNotificationsOptions {
  autoMarkAsRead?: boolean;
  maxNotifications?: number;
  persistUnread?: boolean;
  filter?: {
    types?: NotificationType[];
    priorities?: NotificationPriority[];
    sources?: string[];
  };
}

interface UseWebSocketNotificationsResult {
  notifications: Notification[];
  unreadCount: number;
  isConnected: boolean;
  error: Error | null;
  markAsRead: (notificationId: string) => void;
  markAllAsRead: () => void;
  dismiss: (notificationId: string) => void;
  dismissAll: () => void;
  executeAction: (notificationId: string, action: string) => void;
}

export function useWebSocketNotifications(
  options: UseWebSocketNotificationsOptions = {}
): UseWebSocketNotificationsResult {
  const {
    autoMarkAsRead = false,
    maxNotifications = 100,
    persistUnread = true,
    filter = {},
  } = options;

  const [notifications, setNotifications] = useState<Notification[]>([]);

  // Initialize WebSocket connection
  const ws = useWebSocket({
    path: '/ws/notifications',
    params: {
      autoMarkAsRead: autoMarkAsRead.toString(),
      persistUnread: persistUnread.toString(),
    },
    reconnect: true,
    reconnectInterval: 5000,
    onConnected: () => {
      console.log('Notifications WebSocket connected');
      // Request notification history
      ws.send({
        type: 'request_history',
        payload: {
          limit: maxNotifications,
          filter,
        },
      });
    },
  });

  // Handle new notification
  const handleNotification = useCallback((notification: Notification) => {
    // Apply client-side filter
    if (filter.types && !filter.types.includes(notification.type)) return;
    if (filter.priorities && notification.priority && !filter.priorities.includes(notification.priority)) return;
    if (filter.sources && notification.source && !filter.sources.includes(notification.source)) return;

    setNotifications(prev => {
      const updated = [notification, ...prev];
      
      // Limit notifications
      if (updated.length > maxNotifications) {
        return updated.slice(0, maxNotifications);
      }
      
      return updated;
    });

    // Show browser notification if permission granted
    if ('Notification' in window && Notification.permission === 'granted') {
      new Notification(notification.title, {
        body: notification.message,
        icon: '/favicon.ico',
        tag: notification.id,
      });
    }
  }, [filter, maxNotifications]);

  // Handle notification history
  const handleHistory = useCallback((history: Notification[]) => {
    setNotifications(history);
  }, []);

  // Handle notification update
  const handleUpdate = useCallback((update: { id: string; changes: Partial<Notification> }) => {
    setNotifications(prev => 
      prev.map(n => n.id === update.id ? { ...n, ...update.changes } : n)
    );
  }, []);

  // Subscribe to WebSocket messages
  useWebSocketSubscription(ws, 'notification', handleNotification, [handleNotification]);
  useWebSocketSubscription(ws, 'notification_history', handleHistory, [handleHistory]);
  useWebSocketSubscription(ws, 'notification_update', handleUpdate, [handleUpdate]);

  // Mark notification as read
  const markAsRead = useCallback((notificationId: string) => {
    setNotifications(prev => 
      prev.map(n => n.id === notificationId ? { ...n, read: true } : n)
    );

    ws.send({
      type: 'mark_as_read',
      payload: { notificationId },
    });
  }, [ws]);

  // Mark all notifications as read
  const markAllAsRead = useCallback(() => {
    setNotifications(prev => 
      prev.map(n => ({ ...n, read: true }))
    );

    ws.send({
      type: 'mark_all_as_read',
    });
  }, [ws]);

  // Dismiss notification
  const dismiss = useCallback((notificationId: string) => {
    setNotifications(prev => 
      prev.filter(n => n.id !== notificationId)
    );

    ws.send({
      type: 'dismiss',
      payload: { notificationId },
    });
  }, [ws]);

  // Dismiss all notifications
  const dismissAll = useCallback(() => {
    setNotifications([]);

    ws.send({
      type: 'dismiss_all',
    });
  }, [ws]);

  // Execute notification action
  const executeAction = useCallback((notificationId: string, action: string) => {
    ws.send({
      type: 'execute_action',
      payload: { notificationId, action },
    });

    // Mark as read when action is executed
    markAsRead(notificationId);
  }, [ws, markAsRead]);

  // Request browser notification permission
  useEffect(() => {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission();
    }
  }, []);

  // Remove expired notifications
  useEffect(() => {
    const interval = setInterval(() => {
      const now = new Date();
      setNotifications(prev => 
        prev.filter(n => !n.expires || n.expires > now)
      );
    }, 60000); // Check every minute

    return () => clearInterval(interval);
  }, []);

  const unreadCount = notifications.filter(n => !n.read).length;

  return {
    notifications,
    unreadCount,
    isConnected: ws.isConnected,
    error: ws.error,
    markAsRead,
    markAllAsRead,
    dismiss,
    dismissAll,
    executeAction,
  };
}