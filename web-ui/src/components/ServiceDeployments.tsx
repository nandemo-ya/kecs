import React, { useState, useEffect } from 'react';
import { apiClient } from '../services/api';
import { ServiceDeployment, ListServiceDeploymentsResponse } from '../types/api';
import './ServiceDeployments.css';

interface ServiceDeploymentsProps {
  serviceArn: string;
  clusterArn: string;
}

export function ServiceDeployments({ serviceArn, clusterArn }: ServiceDeploymentsProps) {
  const [deployments, setDeployments] = useState<ServiceDeployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadDeployments();
  }, [serviceArn, clusterArn]);

  const loadDeployments = async () => {
    try {
      setLoading(true);
      setError(null);
      
      // Extract cluster name from ARN
      const clusterName = clusterArn.split('/').pop() || 'default';
      
      const response: ListServiceDeploymentsResponse = await apiClient.listServiceDeployments({
        service: serviceArn,
        cluster: clusterName,
      });
      
      setDeployments(response.serviceDeployments || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load deployments');
    } finally {
      setLoading(false);
    }
  };

  const getStatusClass = (status?: string) => {
    if (!status) return 'status-unknown';
    switch (status.toUpperCase()) {
      case 'PRIMARY':
      case 'ACTIVE':
        return 'status-active';
      case 'COMPLETED':
        return 'status-completed';
      case 'STOPPED':
      case 'FAILED':
        return 'status-stopped';
      case 'IN_PROGRESS':
      case 'PENDING':
        return 'status-pending';
      default:
        return 'status-unknown';
    }
  };

  const getRolloutStateClass = (state?: string) => {
    if (!state) return '';
    switch (state.toUpperCase()) {
      case 'COMPLETED':
        return 'rollout-completed';
      case 'FAILED':
        return 'rollout-failed';
      case 'IN_PROGRESS':
        return 'rollout-in-progress';
      default:
        return '';
    }
  };

  if (loading) {
    return <div className="deployments-loading">Loading deployments...</div>;
  }

  if (error) {
    return <div className="deployments-error">Error: {error}</div>;
  }

  if (deployments.length === 0) {
    return <div className="deployments-empty">No deployments found for this service.</div>;
  }

  return (
    <div className="service-deployments">
      <div className="deployments-list">
        {deployments.map((deployment, index) => (
          <div key={deployment.id || index} className="deployment-card">
            <div className="deployment-header">
              <div className="deployment-id">
                {deployment.id || 'Unknown ID'}
              </div>
              <div className={`deployment-status ${getStatusClass(deployment.status)}`}>
                {deployment.status || 'Unknown'}
              </div>
            </div>

            <div className="deployment-details">
              <div className="detail-row">
                <span className="detail-label">Task Definition:</span>
                <span className="detail-value">{deployment.taskDefinition || 'N/A'}</span>
              </div>

              <div className="detail-row">
                <span className="detail-label">Desired Count:</span>
                <span className="detail-value">{deployment.desiredCount ?? 'N/A'}</span>
              </div>

              <div className="detail-row">
                <span className="detail-label">Running Count:</span>
                <span className="detail-value">{deployment.runningCount ?? 'N/A'}</span>
              </div>

              <div className="detail-row">
                <span className="detail-label">Pending Count:</span>
                <span className="detail-value">{deployment.pendingCount ?? 'N/A'}</span>
              </div>

              {deployment.failedTasks !== undefined && deployment.failedTasks > 0 && (
                <div className="detail-row">
                  <span className="detail-label">Failed Tasks:</span>
                  <span className="detail-value error">{deployment.failedTasks}</span>
                </div>
              )}

              {deployment.rolloutState && (
                <div className="detail-row">
                  <span className="detail-label">Rollout State:</span>
                  <span className={`detail-value ${getRolloutStateClass(deployment.rolloutState)}`}>
                    {deployment.rolloutState}
                  </span>
                </div>
              )}

              {deployment.rolloutStateReason && (
                <div className="detail-row">
                  <span className="detail-label">Rollout Reason:</span>
                  <span className="detail-value">{deployment.rolloutStateReason}</span>
                </div>
              )}

              <div className="detail-row">
                <span className="detail-label">Launch Type:</span>
                <span className="detail-value">{deployment.launchType || 'N/A'}</span>
              </div>

              {deployment.platformVersion && (
                <div className="detail-row">
                  <span className="detail-label">Platform Version:</span>
                  <span className="detail-value">{deployment.platformVersion}</span>
                </div>
              )}

              {deployment.createdAt && (
                <div className="detail-row">
                  <span className="detail-label">Created:</span>
                  <span className="detail-value">
                    {new Date(deployment.createdAt).toLocaleString()}
                  </span>
                </div>
              )}

              {deployment.updatedAt && (
                <div className="detail-row">
                  <span className="detail-label">Updated:</span>
                  <span className="detail-value">
                    {new Date(deployment.updatedAt).toLocaleString()}
                  </span>
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}