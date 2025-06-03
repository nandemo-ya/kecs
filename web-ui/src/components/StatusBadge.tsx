import React from 'react';
import './StatusBadge.css';

interface StatusBadgeProps {
  status: string;
  isLoading?: boolean;
  lastUpdated?: Date;
}

export function StatusBadge({ status, isLoading = false, lastUpdated }: StatusBadgeProps) {
  const getStatusClass = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active':
      case 'running':
      case 'connected':
        return 'status-active';
      case 'pending':
      case 'starting':
        return 'status-pending';
      case 'stopped':
      case 'inactive':
      case 'draining':
        return 'status-stopped';
      case 'error':
      case 'failed':
        return 'status-error';
      default:
        return 'status-unknown';
    }
  };

  const formatLastUpdated = (date: Date) => {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const seconds = Math.floor(diff / 1000);
    
    if (seconds < 60) {
      return `${seconds}s ago`;
    } else if (seconds < 3600) {
      return `${Math.floor(seconds / 60)}m ago`;
    } else {
      return `${Math.floor(seconds / 3600)}h ago`;
    }
  };

  return (
    <div className={`status-badge ${getStatusClass(status)} ${isLoading ? 'loading' : ''}`}>
      <span className="status-text">{status}</span>
      {isLoading && <span className="loading-indicator">⟳</span>}
      {lastUpdated && !isLoading && (
        <span className="last-updated" title={lastUpdated.toLocaleString()}>
          {formatLastUpdated(lastUpdated)}
        </span>
      )}
    </div>
  );
}