import React, { useState } from 'react';
import { useParams, useSearchParams, Link, useNavigate } from 'react-router-dom';
import { useApiData } from '../hooks/useApi';
import { apiClient } from '../services/api';
import { useOperationNotification } from '../hooks/useOperationNotification';
import { TagEditor } from './TagEditor';
import { ServiceDeployments } from './ServiceDeployments';
import { ServiceRevisions } from './ServiceRevisions';
import './DetailPages.css';

export function ServiceDetail() {
  const { serviceName } = useParams<{ serviceName: string }>();
  const [searchParams] = useSearchParams();
  const clusterName = searchParams.get('cluster') || 'default';
  const navigate = useNavigate();
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [activeTab, setActiveTab] = useState<'overview' | 'deployments' | 'revisions'>('overview');
  const { executeWithNotification } = useOperationNotification();
  
  const { data: services, loading: servicesLoading, error: servicesError } = useApiData(
    () => apiClient.describeServices([serviceName || ''], clusterName),
    [serviceName, clusterName]
  );

  const { data: tasks, loading: tasksLoading } = useApiData(
    () => apiClient.listTasks(clusterName),
    [clusterName]
  );

  if (servicesLoading) {
    return (
      <div className="detail-page">
        <div className="loading">Loading service details...</div>
      </div>
    );
  }

  if (servicesError) {
    return (
      <div className="detail-page">
        <div className="error">Error loading service: {servicesError}</div>
        <Link to="/" className="back-link">← Back to Dashboard</Link>
      </div>
    );
  }

  const service = services?.services[0];
  
  if (!service) {
    return (
      <div className="detail-page">
        <div className="error">Service not found: {serviceName}</div>
        <Link to="/" className="back-link">← Back to Dashboard</Link>
      </div>
    );
  }

  const getStatusClass = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active': return 'status-active';
      case 'pending': return 'status-pending';
      case 'draining': return 'status-draining';
      default: return 'status-unknown';
    }
  };

  const handleDeleteService = async () => {
    if (!serviceName) return;
    
    const confirmed = window.confirm(
      `Are you sure you want to delete the service "${serviceName}"? This action cannot be undone.`
    );
    
    if (!confirmed) return;
    
    setDeleteLoading(true);

    const result = await executeWithNotification(
      async () => {
        return await apiClient.deleteService({
          cluster: clusterName,
          service: serviceName,
        });
      },
      {
        inProgressTitle: 'Deleting Service',
        inProgressMessage: `Deleting service "${serviceName}" from cluster "${clusterName}"...`,
        successTitle: 'Service Deleted Successfully',
        successMessage: `Service "${serviceName}" has been deleted from cluster "${clusterName}".`,
        errorTitle: 'Failed to Delete Service',
      }
    );

    setDeleteLoading(false);

    if (result) {
      // Navigate back to services list
      navigate('/services');
    }
  };

  // Extract task definition family and revision
  const taskDefParts = service.taskDefinition.split('/').pop()?.split(':');
  const taskDefFamily = taskDefParts?.[0];
  const taskDefRevision = taskDefParts?.[1];

  return (
    <div className="detail-page">
      <div className="detail-header">
        <Link to="/" className="back-link">← Back to Dashboard</Link>
        <h1>Service: {service.serviceName}</h1>
        <div className="header-actions">
          <Link 
            to={`/services/${serviceName}/update?cluster=${clusterName}`}
            className="btn btn-secondary"
          >
            Update Service
          </Link>
          <button
            onClick={handleDeleteService}
            disabled={deleteLoading}
            className="btn btn-danger"
          >
            {deleteLoading ? 'Deleting...' : 'Delete Service'}
          </button>
          <div className={`status-badge ${getStatusClass(service.status)}`}>
            {service.status}
          </div>
        </div>
      </div>

      <div className="tabs">
        <button
          className={`tab ${activeTab === 'overview' ? 'active' : ''}`}
          onClick={() => setActiveTab('overview')}
        >
          Overview
        </button>
        <button
          className={`tab ${activeTab === 'deployments' ? 'active' : ''}`}
          onClick={() => setActiveTab('deployments')}
        >
          Deployments
        </button>
        <button
          className={`tab ${activeTab === 'revisions' ? 'active' : ''}`}
          onClick={() => setActiveTab('revisions')}
        >
          Revisions
        </button>
      </div>

      {activeTab === 'overview' && (
        <div className="detail-grid">
        <div className="detail-card">
          <h2>Overview</h2>
          <div className="info-grid">
            <div className="info-item">
              <label>Name:</label>
              <span>{service.serviceName}</span>
            </div>
            <div className="info-item">
              <label>ARN:</label>
              <span className="arn">{service.serviceArn}</span>
            </div>
            <div className="info-item">
              <label>Cluster:</label>
              <Link to={`/clusters/${clusterName}`} className="link">
                {clusterName}
              </Link>
            </div>
            <div className="info-item">
              <label>Status:</label>
              <span className={`status ${service.status.toLowerCase()}`}>
                {service.status}
              </span>
            </div>
            <div className="info-item">
              <label>Launch Type:</label>
              <span>{service.launchType}</span>
            </div>
            <div className="info-item">
              <label>Platform Version:</label>
              <span>{service.platformVersion || 'N/A'}</span>
            </div>
            {service.createdAt && (
              <div className="info-item">
                <label>Created:</label>
                <span>{new Date(service.createdAt).toLocaleString()}</span>
              </div>
            )}
          </div>
        </div>

        <div className="detail-card">
          <h2>Task Counts</h2>
          <div className="metrics-grid">
            <div className="metric-item">
              <div className="metric-value">{service.desiredCount}</div>
              <div className="metric-label">Desired</div>
            </div>
            <div className="metric-item">
              <div className="metric-value">{service.runningCount}</div>
              <div className="metric-label">Running</div>
            </div>
            <div className="metric-item">
              <div className="metric-value">{service.pendingCount}</div>
              <div className="metric-label">Pending</div>
            </div>
            <div className="metric-item">
              <div className="metric-value">
                {Math.round((service.runningCount / service.desiredCount) * 100) || 0}%
              </div>
              <div className="metric-label">Health</div>
            </div>
          </div>
        </div>

        <div className="detail-card">
          <TagEditor 
            resourceArn={service.serviceArn} 
            editable={true}
          />
        </div>

        <div className="detail-card">
          <h2>Task Definition</h2>
          <div className="info-grid">
            <div className="info-item">
              <label>ARN:</label>
              <span className="arn">{service.taskDefinition}</span>
            </div>
            <div className="info-item">
              <label>Family:</label>
              <span>{taskDefFamily}</span>
            </div>
            <div className="info-item">
              <label>Revision:</label>
              <span>{taskDefRevision}</span>
            </div>
          </div>
        </div>

        <div className="detail-card">
          <h2>Configuration</h2>
          <div className="info-grid">
            <div className="info-item">
              <label>Scheduling Strategy:</label>
              <span>{service.schedulingStrategy || 'REPLICA'}</span>
            </div>
            <div className="info-item">
              <label>Launch Type:</label>
              <span>{service.launchType}</span>
            </div>
            {service.platformVersion && (
              <div className="info-item">
                <label>Platform Version:</label>
                <span>{service.platformVersion}</span>
              </div>
            )}
          </div>
        </div>

        <div className="detail-card full-width">
          <h2>Related Tasks</h2>
          {tasksLoading ? (
            <div className="loading">Loading tasks...</div>
          ) : tasks?.taskArns.length ? (
            <div className="resource-list">
              {tasks.taskArns.slice(0, 5).map((taskArn) => {
                const taskId = taskArn.split('/').pop();
                return (
                  <div key={taskArn} className="resource-item">
                    <Link 
                      to={`/tasks/${taskId}?cluster=${clusterName}`}
                      className="resource-link"
                    >
                      {taskId}
                    </Link>
                    <span className="resource-arn">{taskArn}</span>
                  </div>
                );
              })}
              {tasks.taskArns.length > 5 && (
                <div className="more-items">
                  and {tasks.taskArns.length - 5} more tasks...
                </div>
              )}
            </div>
          ) : (
            <div className="empty-state">No tasks found for this cluster</div>
          )}
        </div>
      </div>
      )}

      {activeTab === 'deployments' && (
        <ServiceDeployments 
          serviceArn={service.serviceArn} 
          clusterArn={service.clusterArn}
        />
      )}

      {activeTab === 'revisions' && (
        <ServiceRevisions 
          serviceArn={service.serviceArn}
          currentTaskDefinition={service.taskDefinition}
        />
      )}
    </div>
  );
}