import React, { useState } from 'react';
import { ServiceNodeData, ServiceDetails, ServiceEvent } from '../../types/topology';
import './ServiceDetailsPanel.css';

interface ServiceDetailsPanelProps {
  service: ServiceNodeData;
  serviceDetails?: ServiceDetails | null;
  onClose: () => void;
  onRefresh: () => void;
}

export function ServiceDetailsPanel({
  service,
  serviceDetails,
  onClose,
  onRefresh,
}: ServiceDetailsPanelProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'tasks' | 'connections' | 'events'>('overview');

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const getHealthIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return '✅';
      case 'unhealthy':
        return '❌';
      case 'degraded':
        return '⚠️';
      default:
        return '❓';
    }
  };

  const getEventIcon = (type: ServiceEvent['type']) => {
    switch (type) {
      case 'deployment':
        return '🚀';
      case 'scale':
        return '📈';
      case 'health':
        return '🏥';
      case 'error':
        return '🚨';
      case 'config':
        return '⚙️';
      default:
        return '📋';
    }
  };

  return (
    <div className="service-details-panel">
      <div className="details-header">
        <div className="details-title">
          <h3>{service.serviceName}</h3>
          <span className="cluster-badge">{service.clusterName}</span>
        </div>
        <div className="details-actions">
          <button className="action-button" onClick={onRefresh} title="Refresh">
            🔄
          </button>
          <button className="action-button" onClick={onClose} title="Close">
            ✕
          </button>
        </div>
      </div>

      <div className="details-tabs">
        <button
          className={`tab ${activeTab === 'overview' ? 'active' : ''}`}
          onClick={() => setActiveTab('overview')}
        >
          Overview
        </button>
        <button
          className={`tab ${activeTab === 'tasks' ? 'active' : ''}`}
          onClick={() => setActiveTab('tasks')}
        >
          Tasks ({service.runningCount})
        </button>
        <button
          className={`tab ${activeTab === 'connections' ? 'active' : ''}`}
          onClick={() => setActiveTab('connections')}
        >
          Connections
        </button>
        <button
          className={`tab ${activeTab === 'events' ? 'active' : ''}`}
          onClick={() => setActiveTab('events')}
        >
          Events
        </button>
      </div>

      <div className="details-content">
        {activeTab === 'overview' && (
          <div className="tab-content">
            <div className="info-section">
              <h4>Service Information</h4>
              <div className="info-grid">
                <div className="info-item">
                  <span className="label">Type:</span>
                  <span className="value">{service.serviceType}</span>
                </div>
                <div className="info-item">
                  <span className="label">Health:</span>
                  <span className={`value health-${service.healthStatus}`}>
                    {getHealthIcon(service.healthStatus)} {service.healthStatus}
                  </span>
                </div>
                <div className="info-item">
                  <span className="label">Launch Type:</span>
                  <span className="value">{service.launchType || 'N/A'}</span>
                </div>
                <div className="info-item">
                  <span className="label">Created:</span>
                  <span className="value">{formatDate(service.createdAt)}</span>
                </div>
                <div className="info-item">
                  <span className="label">Task Definition:</span>
                  <span className="value">{service.taskDefinition || 'N/A'}</span>
                </div>
                <div className="info-item">
                  <span className="label">Status:</span>
                  <span className="value">{service.deploymentStatus}</span>
                </div>
              </div>
            </div>

            <div className="info-section">
              <h4>Resource Allocation</h4>
              <div className="resource-bars">
                <div className="resource-item">
                  <div className="resource-header">
                    <span>Tasks</span>
                    <span>{service.runningCount} / {service.desiredCount}</span>
                  </div>
                  <div className="resource-bar">
                    <div 
                      className="resource-fill tasks"
                      style={{ width: `${(service.runningCount / service.desiredCount) * 100}%` }}
                    />
                  </div>
                  {service.pendingCount > 0 && (
                    <div className="resource-note">+{service.pendingCount} pending</div>
                  )}
                </div>
              </div>
            </div>

            {serviceDetails?.metrics && (
              <div className="info-section">
                <h4>Performance Metrics</h4>
                <div className="metrics-grid">
                  <div className="metric-item">
                    <span className="metric-value">{serviceDetails.metrics.cpu}%</span>
                    <span className="metric-label">CPU Usage</span>
                  </div>
                  <div className="metric-item">
                    <span className="metric-value">{serviceDetails.metrics.memory}%</span>
                    <span className="metric-label">Memory Usage</span>
                  </div>
                  <div className="metric-item">
                    <span className="metric-value">{serviceDetails.metrics.requestsPerMinute}</span>
                    <span className="metric-label">Requests/min</span>
                  </div>
                  <div className="metric-item">
                    <span className="metric-value">{serviceDetails.metrics.avgResponseTime}ms</span>
                    <span className="metric-label">Avg Response</span>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'tasks' && (
          <div className="tab-content">
            {serviceDetails?.taskDetails && serviceDetails.taskDetails.length > 0 ? (
              <div className="tasks-list">
                {serviceDetails.taskDetails.map((task) => (
                  <div key={task.taskId} className="task-item">
                    <div className="task-header">
                      <span className="task-id">{task.taskId}</span>
                      <span className={`task-status ${task.status.toLowerCase()}`}>
                        {task.status}
                      </span>
                    </div>
                    <div className="task-details">
                      <div className="task-info">
                        <span className="label">CPU:</span> {task.cpu}
                        <span className="separator">|</span>
                        <span className="label">Memory:</span> {task.memory}
                      </div>
                      {task.startedAt && (
                        <div className="task-info">
                          <span className="label">Started:</span> {formatDate(task.startedAt)}
                        </div>
                      )}
                    </div>
                    <div className="container-list">
                      {task.containerInfo.map((container, index) => (
                        <div key={index} className="container-item">
                          <span className="container-name">{container.name}</span>
                          <span className={`container-status ${container.status.toLowerCase()}`}>
                            {container.status}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="empty-state">
                <p>No task details available</p>
              </div>
            )}
          </div>
        )}

        {activeTab === 'connections' && (
          <div className="tab-content">
            {serviceDetails?.connections && (
              <>
                <div className="connections-section">
                  <h4>Incoming Connections ({serviceDetails.connections.incoming.length})</h4>
                  {serviceDetails.connections.incoming.length > 0 ? (
                    <div className="connections-list">
                      {serviceDetails.connections.incoming.map((conn) => (
                        <div key={conn.id} className="connection-item">
                          <span className="connection-source">{conn.source}</span>
                          <span className="connection-type">{conn.connectionType}</span>
                          {conn.requestsPerMinute && (
                            <span className="connection-metric">{conn.requestsPerMinute} req/min</span>
                          )}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="no-connections">No incoming connections</p>
                  )}
                </div>

                <div className="connections-section">
                  <h4>Outgoing Connections ({serviceDetails.connections.outgoing.length})</h4>
                  {serviceDetails.connections.outgoing.length > 0 ? (
                    <div className="connections-list">
                      {serviceDetails.connections.outgoing.map((conn) => (
                        <div key={conn.id} className="connection-item">
                          <span className="connection-target">{conn.target}</span>
                          <span className="connection-type">{conn.connectionType}</span>
                          {conn.latencyMs && (
                            <span className="connection-metric">{conn.latencyMs}ms</span>
                          )}
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="no-connections">No outgoing connections</p>
                  )}
                </div>
              </>
            )}
          </div>
        )}

        {activeTab === 'events' && (
          <div className="tab-content">
            {serviceDetails?.recentEvents && serviceDetails.recentEvents.length > 0 ? (
              <div className="events-list">
                {serviceDetails.recentEvents.map((event) => (
                  <div key={event.id} className={`event-item severity-${event.severity}`}>
                    <div className="event-header">
                      <span className="event-icon">{getEventIcon(event.type)}</span>
                      <span className="event-time">{formatDate(event.timestamp)}</span>
                    </div>
                    <div className="event-message">{event.message}</div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="empty-state">
                <p>No recent events</p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}