import React from 'react';
import { useNotifications, NotificationType } from '../contexts/NotificationContext';
import './NotificationContainer.css';

export function NotificationContainer() {
  const { notifications, removeNotification } = useNotifications();

  const getIconForType = (type: NotificationType) => {
    switch (type) {
      case 'success':
        return '✓';
      case 'error':
        return '✕';
      case 'warning':
        return '⚠';
      case 'info':
        return 'ℹ';
    }
  };

  const getClassForType = (type: NotificationType) => {
    return `notification notification-${type}`;
  };

  if (notifications.length === 0) {
    return null;
  }

  return (
    <div className="notification-container">
      {notifications.map(notification => (
        <div
          key={notification.id}
          className={getClassForType(notification.type)}
          onClick={() => removeNotification(notification.id)}
        >
          <div className="notification-icon">
            {getIconForType(notification.type)}
          </div>
          <div className="notification-content">
            <div className="notification-title">{notification.title}</div>
            {notification.message && (
              <div className="notification-message">{notification.message}</div>
            )}
          </div>
          <button
            className="notification-close"
            onClick={(e) => {
              e.stopPropagation();
              removeNotification(notification.id);
            }}
            aria-label="Close notification"
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
}