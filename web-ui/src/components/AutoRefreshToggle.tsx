import React from 'react';
import './AutoRefreshToggle.css';

interface AutoRefreshToggleProps {
  isEnabled: boolean;
  isRefreshing: boolean;
  onToggle: () => void;
  interval?: number;
}

export function AutoRefreshToggle({ 
  isEnabled, 
  isRefreshing, 
  onToggle, 
  interval = 5000 
}: AutoRefreshToggleProps) {
  const intervalInSeconds = interval / 1000;

  return (
    <button
      className={`auto-refresh-toggle ${isEnabled ? 'enabled' : 'disabled'} ${isRefreshing ? 'refreshing' : ''}`}
      onClick={onToggle}
      title={`Auto-refresh ${isEnabled ? 'enabled' : 'disabled'} (${intervalInSeconds}s)`}
    >
      <span className="refresh-icon">🔄</span>
      <span className="refresh-text">
        Auto ({intervalInSeconds}s)
      </span>
      {isRefreshing && <span className="refresh-indicator">⏳</span>}
    </button>
  );
}