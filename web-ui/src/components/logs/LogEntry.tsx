import React, { memo, useState } from 'react';
import { LogEntry as LogEntryType, getLogLevelColor, formatLogTimestamp } from '../../types/logs';
import { LogHighlight } from './LogHighlight';
import './Logs.css';

interface LogEntryProps {
  entry: LogEntryType;
  showTimestamp?: boolean;
  showLevel?: boolean;
  showSource?: boolean;
  showLineNumber?: boolean;
  lineNumber?: number;
  dateFormat?: string;
  isSelected?: boolean;
  searchTerm?: string;
  highlights?: Array<{
    pattern: string | RegExp;
    className: string;
    color?: string;
  }>;
  onSelect?: (entry: LogEntryType) => void;
  onContextMenu?: (entry: LogEntryType, event: React.MouseEvent) => void;
}

export const LogEntry = memo(({
  entry,
  showTimestamp = true,
  showLevel = true,
  showSource = true,
  showLineNumber = false,
  lineNumber,
  dateFormat = 'HH:mm:ss.SSS',
  isSelected = false,
  searchTerm,
  highlights,
  onSelect,
  onContextMenu,
}: LogEntryProps) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const levelColor = getLogLevelColor(entry.level);

  const handleClick = () => {
    if (onSelect) {
      onSelect(entry);
    }
  };

  const handleContextMenu = (event: React.MouseEvent) => {
    event.preventDefault();
    if (onContextMenu) {
      onContextMenu(entry, event);
    }
  };

  const toggleExpanded = (event: React.MouseEvent) => {
    event.stopPropagation();
    setIsExpanded(!isExpanded);
  };

  const hasMetadata = entry.metadata && Object.keys(entry.metadata).length > 0;
  const hasStackTrace = !!entry.stackTrace;
  const isExpandable = hasMetadata || hasStackTrace;

  return (
    <div 
      className={`log-entry log-level-${entry.level} ${isSelected ? 'selected' : ''}`}
      onClick={handleClick}
      onContextMenu={handleContextMenu}
    >
      <div className="log-entry-main">
        {showLineNumber && lineNumber !== undefined && (
          <span className="log-line-number">{lineNumber}</span>
        )}
        
        {showTimestamp && (
          <span className="log-timestamp">
            {formatLogTimestamp(entry.timestamp, dateFormat)}
          </span>
        )}
        
        {showLevel && (
          <span 
            className="log-level"
            style={{ color: levelColor }}
          >
            [{entry.level.toUpperCase()}]
          </span>
        )}
        
        {showSource && (
          <span className="log-source">
            [{entry.source.name}]
          </span>
        )}
        
        <span className="log-message">
          <LogHighlight 
            text={entry.message} 
            search={searchTerm}
            highlights={highlights}
          />
        </span>

        {isExpandable && (
          <button 
            className={`log-expand-button ${isExpanded ? 'expanded' : ''}`}
            onClick={toggleExpanded}
            aria-label={isExpanded ? 'Collapse' : 'Expand'}
          >
            {isExpanded ? '▼' : '▶'}
          </button>
        )}
      </div>

      {isExpanded && (
        <div className="log-entry-details">
          {hasMetadata && (
            <div className="log-metadata">
              <div className="log-metadata-header">Metadata:</div>
              <div className="log-metadata-content">
                {Object.entries(entry.metadata!).map(([key, value]) => (
                  <div key={key} className="log-metadata-item">
                    <span className="log-metadata-key">{key}:</span>
                    <span className="log-metadata-value">
                      {typeof value === 'object' ? JSON.stringify(value, null, 2) : String(value)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {hasStackTrace && (
            <div className="log-stacktrace">
              <div className="log-stacktrace-header">Stack Trace:</div>
              <pre className="log-stacktrace-content">{entry.stackTrace}</pre>
            </div>
          )}

          {entry.correlationId && (
            <div className="log-correlation">
              <span className="log-correlation-label">Correlation ID:</span>
              <span className="log-correlation-id">{entry.correlationId}</span>
            </div>
          )}

          <div className="log-entry-ids">
            {entry.taskId && (
              <span className="log-id-item">
                <span className="log-id-label">Task:</span>
                <span className="log-id-value">{entry.taskId}</span>
              </span>
            )}
            {entry.containerId && (
              <span className="log-id-item">
                <span className="log-id-label">Container:</span>
                <span className="log-id-value">{entry.containerId}</span>
              </span>
            )}
            {entry.serviceName && (
              <span className="log-id-item">
                <span className="log-id-label">Service:</span>
                <span className="log-id-value">{entry.serviceName}</span>
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
});

LogEntry.displayName = 'LogEntry';