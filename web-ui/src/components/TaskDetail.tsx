import React from 'react';
import { useParams, useSearchParams, Link } from 'react-router-dom';
import { useApiData } from '../hooks/useApi';
import { apiClient } from '../services/api';
import './DetailPages.css';

export function TaskDetail() {
  const { taskId } = useParams<{ taskId: string }>();
  const [searchParams] = useSearchParams();
  const clusterName = searchParams.get('cluster') || 'default';
  
  // We need to construct the full task ARN for the API call
  const taskArn = `arn:aws:ecs:ap-northeast-1:123456789012:task/${clusterName}/${taskId}`;
  
  const { data: tasks, loading: tasksLoading, error: tasksError } = useApiData(
    () => apiClient.describeTasks([taskArn], clusterName),
    [taskArn, clusterName]
  );

  if (tasksLoading) {
    return (
      <div className="detail-page">
        <div className="loading">Loading task details...</div>
      </div>
    );
  }

  if (tasksError) {
    return (
      <div className="detail-page">
        <div className="error">Error loading task: {tasksError}</div>
        <Link to="/" className="back-link">← Back to Dashboard</Link>
      </div>
    );
  }

  const task = tasks?.tasks[0];
  
  if (!task) {
    return (
      <div className="detail-page">
        <div className="error">Task not found: {taskId}</div>
        <Link to="/" className="back-link">← Back to Dashboard</Link>
      </div>
    );
  }

  const getStatusClass = (status: string) => {
    switch (status.toLowerCase()) {
      case 'running': return 'status-active';
      case 'pending': return 'status-pending';
      case 'stopped': return 'status-stopped';
      case 'stopping': return 'status-draining';
      default: return 'status-unknown';
    }
  };

  // Extract task definition family and revision
  const taskDefParts = task.taskDefinitionArn.split('/').pop()?.split(':');
  const taskDefFamily = taskDefParts?.[0];
  const taskDefRevision = taskDefParts?.[1];

  return (
    <div className="detail-page">
      <div className="detail-header">
        <Link to="/" className="back-link">← Back to Dashboard</Link>
        <h1>Task: {taskId}</h1>
        <div className={`status-badge ${getStatusClass(task.lastStatus)}`}>
          {task.lastStatus}
        </div>
      </div>

      <div className="detail-grid">
        <div className="detail-card">
          <h2>Overview</h2>
          <div className="info-grid">
            <div className="info-item">
              <label>Task ID:</label>
              <span>{taskId}</span>
            </div>
            <div className="info-item">
              <label>ARN:</label>
              <span className="arn">{task.taskArn}</span>
            </div>
            <div className="info-item">
              <label>Cluster:</label>
              <Link to={`/clusters/${clusterName}`} className="link">
                {clusterName}
              </Link>
            </div>
            <div className="info-item">
              <label>Last Status:</label>
              <span className={`status ${task.lastStatus.toLowerCase()}`}>
                {task.lastStatus}
              </span>
            </div>
            <div className="info-item">
              <label>Desired Status:</label>
              <span className={`status ${task.desiredStatus.toLowerCase()}`}>
                {task.desiredStatus}
              </span>
            </div>
            <div className="info-item">
              <label>Launch Type:</label>
              <span>{task.launchType}</span>
            </div>
          </div>
        </div>

        <div className="detail-card">
          <h2>Resources</h2>
          <div className="info-grid">
            <div className="info-item">
              <label>CPU:</label>
              <span>{task.cpu || 'N/A'}</span>
            </div>
            <div className="info-item">
              <label>Memory:</label>
              <span>{task.memory || 'N/A'}</span>
            </div>
            <div className="info-item">
              <label>Health Status:</label>
              <span className={`status ${(task.healthStatus || 'unknown').toLowerCase()}`}>
                {task.healthStatus || 'Unknown'}
              </span>
            </div>
          </div>
        </div>

        <div className="detail-card">
          <h2>Task Definition</h2>
          <div className="info-grid">
            <div className="info-item">
              <label>ARN:</label>
              <span className="arn">{task.taskDefinitionArn}</span>
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
          <h2>Execution Details</h2>
          <div className="info-grid">
            {task.startedAt && (
              <div className="info-item">
                <label>Started At:</label>
                <span>{new Date(task.startedAt).toLocaleString()}</span>
              </div>
            )}
            {task.startedBy && (
              <div className="info-item">
                <label>Started By:</label>
                <span>{task.startedBy}</span>
              </div>
            )}
            {task.group && (
              <div className="info-item">
                <label>Group:</label>
                <span>{task.group}</span>
              </div>
            )}
          </div>
        </div>

        {task.startedBy && task.startedBy.startsWith('service:') && (
          <div className="detail-card full-width">
            <h2>Related Service</h2>
            <div className="service-info">
              <div className="info-item">
                <label>Service:</label>
                <Link 
                  to={`/services/${task.startedBy.replace('service:', '')}?cluster=${clusterName}`}
                  className="link"
                >
                  {task.startedBy.replace('service:', '')}
                </Link>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}