import React, { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { LogEntry } from './LogEntry';
import { LogFilter } from './LogFilter';
import { LogStats } from './LogStats';
import { AdvancedSearch } from './AdvancedSearch';
import { useWebSocketLogStream } from '../../hooks/useWebSocketLogStream';
import { 
  LogEntry as LogEntryType, 
  LogFilter as LogFilterType,
  filterLogs,
  LogExportOptions,
  formatLogTimestamp,
  getLogLevelColor,
} from '../../types/logs';
import './Logs.css';

interface LogViewerProps {
  taskId?: string;
  serviceName?: string;
  containerId?: string;
  enableStreaming?: boolean;
  maxDisplayEntries?: number;
}

export function LogViewer({
  taskId,
  serviceName,
  containerId,
  enableStreaming = true,
  maxDisplayEntries = 500,
}: LogViewerProps) {
  const [filter, setFilter] = useState<LogFilterType>({
    levels: [],
    sources: [],
    search: '',
    taskIds: taskId ? [taskId] : [],
    serviceNames: serviceName ? [serviceName] : [],
    containerIds: containerId ? [containerId] : [],
  });
  const [selectedEntry, setSelectedEntry] = useState<LogEntryType | null>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const [showStats, setShowStats] = useState(false);
  const [showAdvancedSearch, setShowAdvancedSearch] = useState(false);
  const [highlights, setHighlights] = useState<Array<{
    pattern: string | RegExp;
    className: string;
    color?: string;
  }>>([]);
  
  const containerRef = useRef<HTMLDivElement>(null);

  // Use WebSocket log stream
  const {
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
    setFilter: setStreamFilter,
  } = useWebSocketLogStream({
    taskId,
    serviceName,
    containerId,
    follow: enableStreaming,
    maxBufferSize: 1000,
  });

  // Filter logs
  const filteredLogs = useMemo(() => {
    const filtered = filterLogs(logs, filter);
    // Limit display entries
    if (filtered.length > maxDisplayEntries) {
      return filtered.slice(-maxDisplayEntries);
    }
    return filtered;
  }, [logs, filter, maxDisplayEntries]);

  // Get unique sources for filter
  const availableSources = useMemo(() => {
    const sources = new Set<string>();
    logs.forEach(log => sources.add(log.source.name));
    return Array.from(sources).sort();
  }, [logs]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (autoScroll && containerRef.current && !isPaused) {
      // Use requestAnimationFrame for smooth scrolling without affecting the whole page
      requestAnimationFrame(() => {
        if (containerRef.current) {
          containerRef.current.scrollTop = containerRef.current.scrollHeight;
        }
      });
    }
  }, [filteredLogs, autoScroll, isPaused]);

  // Handle filter change
  const handleFilterChange = useCallback((newFilter: LogFilterType) => {
    setFilter(newFilter);
    
    // Update WebSocket stream filter
    setStreamFilter({
      taskIds: newFilter.taskIds,
      serviceNames: newFilter.serviceNames,
      containerIds: newFilter.containerIds,
      levels: newFilter.levels,
      search: newFilter.search,
    });
  }, []);

  // Handle entry selection
  const handleEntrySelect = useCallback((entry: LogEntryType) => {
    setSelectedEntry(entry);
  }, []);

  // Handle context menu
  const handleEntryContextMenu = useCallback((entry: LogEntryType, event: React.MouseEvent) => {
    // Implement context menu actions
    console.log('Context menu for entry:', entry);
  }, []);

  // Export logs
  const exportLogs = useCallback((options: LogExportOptions) => {
    const logsToExport = filteredLogs;
    let content = '';
    let mimeType = '';
    let extension = '';

    switch (options.format) {
      case 'json':
        content = JSON.stringify(logsToExport, null, 2);
        mimeType = 'application/json';
        extension = 'json';
        break;
      
      case 'csv':
        const headers = ['Timestamp', 'Level', 'Source', 'Message'];
        if (options.includeMetadata) {
          headers.push('Metadata');
        }
        const rows = logsToExport.map(log => {
          const row = [
            formatLogTimestamp(log.timestamp, options.dateFormat),
            log.level.toUpperCase(),
            log.source.name,
            `"${log.message.replace(/"/g, '""')}"`,
          ];
          if (options.includeMetadata && log.metadata) {
            row.push(`"${JSON.stringify(log.metadata).replace(/"/g, '""')}"`);
          }
          return row.join(options.delimiter || ',');
        });
        content = [headers.join(options.delimiter || ','), ...rows].join('\n');
        mimeType = 'text/csv';
        extension = 'csv';
        break;
      
      case 'text':
        content = logsToExport.map(log => 
          `${formatLogTimestamp(log.timestamp, options.dateFormat)} [${log.level.toUpperCase()}] [${log.source.name}] ${log.message}`
        ).join('\n');
        mimeType = 'text/plain';
        extension = 'txt';
        break;
    }

    // Create and download file
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = options.filename || `logs-${Date.now()}.${extension}`;
    link.click();
    URL.revokeObjectURL(url);
  }, [filteredLogs]);

  // Connection status component
  const ConnectionStatus = () => (
    <div className={`connection-status ${isConnected ? 'connected' : 'disconnected'}`}>
      <span className="connection-indicator"></span>
      <span className="connection-text">
        {isConnected ? 'Connected' : isReconnecting ? 'Reconnecting...' : 'Disconnected'}
      </span>
      {error && <span className="connection-error" title={error.message}>⚠️</span>}
    </div>
  );

  return (
    <div className="log-viewer">
      <div className="log-viewer-header">
        <h3>Log Viewer</h3>
        <div className="log-viewer-controls">
          <ConnectionStatus />
          
          <div className="control-group">
            <button
              className={`control-button ${isPaused ? 'paused' : ''}`}
              onClick={isPaused ? resume : pause}
              title={isPaused ? 'Resume' : 'Pause'}
            >
              {isPaused ? '▶️' : '⏸️'}
            </button>
            
            <button
              className="control-button"
              onClick={clear}
              title="Clear logs"
            >
              🗑️
            </button>
            
            <button
              className={`control-button ${autoScroll ? 'active' : ''}`}
              onClick={() => setAutoScroll(!autoScroll)}
              title="Auto-scroll"
            >
              ⬇️
            </button>
            
            <button
              className={`control-button ${showStats ? 'active' : ''}`}
              onClick={() => setShowStats(!showStats)}
              title="Toggle statistics"
            >
              📊
            </button>
            
            <button
              className="control-button"
              onClick={() => setShowAdvancedSearch(true)}
              title="Advanced search"
            >
              🔍
            </button>
            
            <div className="export-dropdown">
              <button className="control-button" title="Export logs">
                💾
              </button>
              <div className="export-menu">
                <button onClick={() => exportLogs({ format: 'json' })}>
                  Export as JSON
                </button>
                <button onClick={() => exportLogs({ format: 'csv' })}>
                  Export as CSV
                </button>
                <button onClick={() => exportLogs({ format: 'text' })}>
                  Export as Text
                </button>
              </div>
            </div>
          </div>

          <div className="log-count">
            {filteredLogs.length} / {logs.length} logs
          </div>
        </div>
      </div>

      <LogFilter
        filter={filter}
        onChange={handleFilterChange}
        availableSources={availableSources}
      />

      {showStats && (
        <LogStats logs={filteredLogs} />
      )}

      <div className="log-viewer-content" ref={containerRef}>
        {filteredLogs.length === 0 ? (
          <div className="log-empty-state">
            {logs.length === 0 ? (
              <>
                <p>No logs available</p>
                {!isConnected && (
                  <button onClick={connect} className="connect-button">
                    Connect to log stream
                  </button>
                )}
              </>
            ) : (
              <p>No logs match the current filter</p>
            )}
          </div>
        ) : (
          <>
            {filteredLogs.map((entry, index) => (
              <LogEntry
                key={entry.id}
                entry={entry}
                lineNumber={index + 1}
                showLineNumber={true}
                isSelected={selectedEntry?.id === entry.id}
                searchTerm={filter.search}
                highlights={highlights}
                onSelect={handleEntrySelect}
                onContextMenu={handleEntryContextMenu}
              />
            ))}
          </>
        )}
      </div>

      {selectedEntry && (
        <div className="log-details-panel">
          <div className="log-details-header">
            <h4>Log Details</h4>
            <button 
              className="close-button"
              onClick={() => setSelectedEntry(null)}
            >
              ✕
            </button>
          </div>
          <div className="log-details-content">
            <div className="detail-row">
              <span className="detail-label">ID:</span>
              <span className="detail-value">{selectedEntry.id}</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Timestamp:</span>
              <span className="detail-value">
                {formatLogTimestamp(selectedEntry.timestamp, 'YYYY-MM-DD HH:mm:ss.SSS')}
              </span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Level:</span>
              <span className="detail-value" style={{ color: getLogLevelColor(selectedEntry.level) }}>
                {selectedEntry.level.toUpperCase()}
              </span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Source:</span>
              <span className="detail-value">{selectedEntry.source.name} ({selectedEntry.source.type})</span>
            </div>
            <div className="detail-row">
              <span className="detail-label">Message:</span>
              <span className="detail-value">{selectedEntry.message}</span>
            </div>
            {selectedEntry.metadata && (
              <div className="detail-row">
                <span className="detail-label">Metadata:</span>
                <pre className="detail-value">
                  {JSON.stringify(selectedEntry.metadata, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      )}

      {showAdvancedSearch && (
        <AdvancedSearch
          filter={filter}
          onChange={handleFilterChange}
          onClose={() => setShowAdvancedSearch(false)}
          availableTasks={logs.map(l => l.taskId).filter(Boolean) as string[]}
          availableServices={logs.map(l => l.serviceName).filter(Boolean) as string[]}
          availableContainers={logs.map(l => l.containerId).filter(Boolean) as string[]}
        />
      )}
    </div>
  );
}