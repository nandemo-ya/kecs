import React, { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { LogViewer } from './logs/LogViewer';
import './logs/Logs.css';

export function LogViewerDashboard() {
  const { taskId, serviceName, containerId } = useParams<{
    taskId?: string;
    serviceName?: string;
    containerId?: string;
  }>();
  
  const [viewMode, setViewMode] = useState<'all' | 'task' | 'service' | 'container'>('all');

  // Determine view mode based on params
  React.useEffect(() => {
    if (taskId) {
      setViewMode('task');
    } else if (serviceName) {
      setViewMode('service');
    } else if (containerId) {
      setViewMode('container');
    } else {
      setViewMode('all');
    }
  }, [taskId, serviceName, containerId]);

  return (
    <main className="App-main">
      <div className="dashboard-header">
        <div className="header-content">
          <h2>📋 Log Viewer</h2>
          <div className="header-breadcrumb">
            {taskId && (
              <>
                <Link to="/tasks">Tasks</Link>
                <span className="breadcrumb-separator">›</span>
                <span>Task {taskId} Logs</span>
              </>
            )}
            {serviceName && (
              <>
                <Link to="/services">Services</Link>
                <span className="breadcrumb-separator">›</span>
                <span>Service {serviceName} Logs</span>
              </>
            )}
            {containerId && (
              <>
                <span>Container {containerId} Logs</span>
              </>
            )}
            {viewMode === 'all' && (
              <span>All Logs</span>
            )}
          </div>
        </div>
        
        <div className="header-actions">
          <div className="view-mode-selector">
            <button
              className={`view-mode-button ${viewMode === 'all' ? 'active' : ''}`}
              onClick={() => setViewMode('all')}
            >
              All Logs
            </button>
            <button
              className={`view-mode-button ${viewMode === 'task' ? 'active' : ''}`}
              onClick={() => setViewMode('task')}
              disabled={!taskId}
            >
              Task Logs
            </button>
            <button
              className={`view-mode-button ${viewMode === 'service' ? 'active' : ''}`}
              onClick={() => setViewMode('service')}
              disabled={!serviceName}
            >
              Service Logs
            </button>
            <button
              className={`view-mode-button ${viewMode === 'container' ? 'active' : ''}`}
              onClick={() => setViewMode('container')}
              disabled={!containerId}
            >
              Container Logs
            </button>
          </div>
        </div>
      </div>

      <div className="log-viewer-container">
        <LogViewer
          taskId={viewMode === 'task' ? taskId : undefined}
          serviceName={viewMode === 'service' ? serviceName : undefined}
          containerId={viewMode === 'container' ? containerId : undefined}
          enableStreaming={true}
          maxDisplayEntries={1000}
        />
      </div>

      <div className="log-viewer-info">
        <div className="info-card">
          <h3>📊 Log Streaming</h3>
          <p>Real-time log streaming with WebSocket connection</p>
          <ul>
            <li>Automatic reconnection on disconnection</li>
            <li>Buffer management for performance</li>
            <li>Pause/resume functionality</li>
            <li>Export logs in multiple formats</li>
          </ul>
        </div>

        <div className="info-card">
          <h3>🔍 Filtering</h3>
          <p>Advanced filtering capabilities</p>
          <ul>
            <li>Filter by log level (trace, debug, info, warn, error, fatal)</li>
            <li>Filter by source/service</li>
            <li>Text search across log messages</li>
            <li>Regular expression support</li>
            <li>Time range filtering</li>
          </ul>
        </div>

        <div className="info-card">
          <h3>📈 Statistics</h3>
          <p>Real-time log statistics</p>
          <ul>
            <li>Log level distribution</li>
            <li>Error and warning rates</li>
            <li>Top log sources</li>
            <li>Logs per minute average</li>
          </ul>
        </div>

        <div className="info-card">
          <h3>⚡ Features</h3>
          <p>Enhanced viewing experience</p>
          <ul>
            <li>Expandable log entries with metadata</li>
            <li>Stack trace visualization</li>
            <li>Correlation ID tracking</li>
            <li>Auto-scroll with manual override</li>
            <li>Line numbers and timestamps</li>
            <li>Syntax highlighting by log level</li>
          </ul>
        </div>
      </div>

    </main>
  );
}