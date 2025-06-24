import React, { useCallback, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useApiData } from '../hooks/useApi';
import { apiClient } from '../services/api';
import { useOperationNotification } from '../hooks/useOperationNotification';
import { RunTask } from './RunTask';
import { TagEditor } from './TagEditor';
import './DetailPages.css';

export function ClusterDetail() {
  const { clusterName } = useParams<{ clusterName: string }>();
  const navigate = useNavigate();
  const [deleting, setDeleting] = useState(false);
  const [showRunTaskModal, setShowRunTaskModal] = useState(false);
  const { notifySuccess, notifyError } = useOperationNotification();
  
  const describeClusters = useCallback(
    () => apiClient.describeClusters([clusterName || '']),
    [clusterName]
  );
  
  const listServices = useCallback(
    () => apiClient.listServices(clusterName),
    [clusterName]
  );
  
  const listTasks = useCallback(
    () => apiClient.listTasks(clusterName),
    [clusterName]
  );
  
  const { data: clusters, loading: clustersLoading, error: clustersError } = useApiData(
    describeClusters,
    [clusterName]
  );
  
  const { data: services, loading: servicesLoading } = useApiData(
    listServices,
    [clusterName]
  );
  
  const { data: tasks, loading: tasksLoading, refresh: refetchTasks } = useApiData(
    listTasks,
    [clusterName]
  );

  const handleDelete = async () => {
    if (!clusterName) return;
    
    const confirmed = window.confirm(`Are you sure you want to delete cluster "${clusterName}"? This action cannot be undone.`);
    if (!confirmed) return;

    setDeleting(true);
    try {
      await apiClient.deleteCluster({ cluster: clusterName });
      notifySuccess(`Cluster "${clusterName}" deleted successfully`);
      navigate('/clusters');
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to delete cluster';
      notifyError(errorMessage);
    } finally {
      setDeleting(false);
    }
  };

  if (clustersLoading) {
    return (
      <div className="detail-page">
        <div className="loading">Loading cluster details...</div>
      </div>
    );
  }

  if (clustersError) {
    return (
      <div className="detail-page">
        <div className="error">Error loading cluster: {clustersError}</div>
        <Link to="/" className="back-link">← Back to Dashboard</Link>
      </div>
    );
  }

  const cluster = clusters?.clusters[0];
  
  if (!cluster) {
    return (
      <div className="detail-page">
        <div className="error">Cluster not found: {clusterName}</div>
        <Link to="/" className="back-link">← Back to Dashboard</Link>
      </div>
    );
  }

  return (
    <div className="detail-page">
      <div className="detail-header">
        <Link to="/" className="back-link">← Back to Dashboard</Link>
        <h1>Cluster: {cluster.clusterName}</h1>
        <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
          <div className="status-badge status-active">{cluster.status}</div>
          <button 
            className="button button-danger" 
            onClick={handleDelete}
            disabled={deleting || cluster.activeServicesCount > 0 || cluster.runningTasksCount > 0}
            title={cluster.activeServicesCount > 0 || cluster.runningTasksCount > 0 ? 'Cannot delete cluster with active services or running tasks' : ''}
          >
            {deleting ? 'Deleting...' : 'Delete Cluster'}
          </button>
        </div>
      </div>

      <div className="detail-grid">
        <div className="detail-card">
          <h2>Overview</h2>
          <div className="info-grid">
            <div className="info-item">
              <label>Name:</label>
              <span>{cluster.clusterName}</span>
            </div>
            <div className="info-item">
              <label>ARN:</label>
              <span className="arn">{cluster.clusterArn}</span>
            </div>
            <div className="info-item">
              <label>Status:</label>
              <span className={`status ${cluster.status.toLowerCase()}`}>
                {cluster.status}
              </span>
            </div>
          </div>
        </div>

        <div className="detail-card">
          <h2>Resource Counts</h2>
          <div className="metrics-grid">
            <div className="metric-item">
              <div className="metric-value">{cluster.activeServicesCount}</div>
              <div className="metric-label">Active Services</div>
            </div>
            <div className="metric-item">
              <div className="metric-value">{cluster.runningTasksCount}</div>
              <div className="metric-label">Running Tasks</div>
            </div>
            <div className="metric-item">
              <div className="metric-value">{cluster.pendingTasksCount}</div>
              <div className="metric-label">Pending Tasks</div>
            </div>
            <div className="metric-item">
              <div className="metric-value">{cluster.registeredContainerInstancesCount}</div>
              <div className="metric-label">Container Instances</div>
            </div>
          </div>
        </div>

        <div className="detail-card">
          <TagEditor 
            resourceArn={cluster.clusterArn} 
            editable={true}
          />
        </div>

        <div className="detail-card">
          <h2>Services</h2>
          {servicesLoading ? (
            <div className="loading">Loading services...</div>
          ) : services?.serviceArns && services.serviceArns.length > 0 ? (
            <div className="resource-list">
              {services.serviceArns.map((serviceArn) => {
                const serviceName = serviceArn.split('/').pop();
                return (
                  <div key={serviceArn} className="resource-item">
                    <Link 
                      to={`/services/${serviceName}?cluster=${clusterName}`}
                      className="resource-link"
                    >
                      {serviceName}
                    </Link>
                    <span className="resource-arn">{serviceArn}</span>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="empty-state">No services found</div>
          )}
        </div>

        <div className="detail-card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
            <h2>Tasks</h2>
            <button 
              className="button button-primary" 
              onClick={() => setShowRunTaskModal(true)}
            >
              Run Task
            </button>
          </div>
          {tasksLoading ? (
            <div className="loading">Loading tasks...</div>
          ) : tasks?.taskArns && tasks.taskArns.length > 0 ? (
            <div className="resource-list">
              {tasks.taskArns.slice(0, 10).map((taskArn) => {
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
              {tasks.taskArns.length > 10 && (
                <div className="more-items">
                  and {tasks.taskArns.length - 10} more tasks...
                </div>
              )}
            </div>
          ) : (
            <div className="empty-state">No tasks found</div>
          )}
        </div>
      </div>

      <RunTask
        isOpen={showRunTaskModal}
        onClose={() => setShowRunTaskModal(false)}
        onSuccess={refetchTasks}
        clusterName={clusterName}
      />
    </div>
  );
}