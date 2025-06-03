import React, { useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useApiData } from '../hooks/useApi';
import { apiClient } from '../services/api';
import './DetailPages.css';

export function TaskDefinitionDetail() {
  const { family, revision } = useParams<{ family: string; revision: string }>();
  const navigate = useNavigate();
  const [deregisterLoading, setDeregisterLoading] = useState(false);
  
  const taskDefinitionArn = `${family}:${revision}`;
  
  const { data, loading, error } = useApiData(
    () => apiClient.describeTaskDefinition(taskDefinitionArn),
    [taskDefinitionArn]
  );

  if (loading) {
    return (
      <div className="detail-page">
        <div className="loading">Loading task definition details...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="detail-page">
        <div className="error">Error loading task definition: {error}</div>
        <Link to="/task-definitions" className="back-link">← Back to Task Definitions</Link>
      </div>
    );
  }

  const taskDefinition = data?.taskDefinition;
  
  if (!taskDefinition) {
    return (
      <div className="detail-page">
        <div className="error">Task definition not found: {taskDefinitionArn}</div>
        <Link to="/task-definitions" className="back-link">← Back to Task Definitions</Link>
      </div>
    );
  }

  const getStatusClass = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active': return 'status-active';
      case 'inactive': return 'status-stopped';
      default: return 'status-unknown';
    }
  };

  const handleDeregister = async () => {
    const confirmed = window.confirm(
      `Are you sure you want to deregister the task definition "${family}:${revision}"? This action cannot be undone.`
    );
    
    if (!confirmed) return;
    
    setDeregisterLoading(true);
    try {
      await apiClient.deregisterTaskDefinition({
        taskDefinition: taskDefinitionArn,
      });
      
      // Navigate back to task definitions list
      navigate('/task-definitions');
    } catch (err) {
      alert('Failed to deregister task definition: ' + (err instanceof Error ? err.message : 'Unknown error'));
    } finally {
      setDeregisterLoading(false);
    }
  };

  return (
    <div className="detail-page">
      <div className="detail-header">
        <Link to="/task-definitions" className="back-link">← Back to Task Definitions</Link>
        <h1>Task Definition: {taskDefinition.family}:{taskDefinition.revision}</h1>
        <div className="header-actions">
          <button
            onClick={handleDeregister}
            disabled={deregisterLoading || taskDefinition.status !== 'ACTIVE'}
            className="btn btn-danger"
          >
            {deregisterLoading ? 'Deregistering...' : 'Deregister'}
          </button>
          <div className={`status-badge ${getStatusClass(taskDefinition.status)}`}>
            {taskDefinition.status}
          </div>
        </div>
      </div>

      <div className="detail-grid">
        <div className="detail-card">
          <h2>Overview</h2>
          <div className="info-grid">
            <div className="info-item">
              <label>Family:</label>
              <span>{taskDefinition.family}</span>
            </div>
            <div className="info-item">
              <label>Revision:</label>
              <span>{taskDefinition.revision}</span>
            </div>
            <div className="info-item">
              <label>ARN:</label>
              <span className="arn">{taskDefinition.taskDefinitionArn}</span>
            </div>
            <div className="info-item">
              <label>Status:</label>
              <span className={`status ${taskDefinition.status.toLowerCase()}`}>
                {taskDefinition.status}
              </span>
            </div>
            {taskDefinition.registeredAt && (
              <div className="info-item">
                <label>Registered:</label>
                <span>{new Date(taskDefinition.registeredAt).toLocaleString()}</span>
              </div>
            )}
          </div>
        </div>

        <div className="detail-card">
          <h2>Resource Configuration</h2>
          <div className="info-grid">
            {taskDefinition.cpu && (
              <div className="info-item">
                <label>CPU:</label>
                <span>{taskDefinition.cpu} units</span>
              </div>
            )}
            {taskDefinition.memory && (
              <div className="info-item">
                <label>Memory:</label>
                <span>{taskDefinition.memory} MB</span>
              </div>
            )}
            {taskDefinition.networkMode && (
              <div className="info-item">
                <label>Network Mode:</label>
                <span>{taskDefinition.networkMode}</span>
              </div>
            )}
            {taskDefinition.requiresCompatibilities && taskDefinition.requiresCompatibilities.length > 0 && (
              <div className="info-item">
                <label>Launch Types:</label>
                <span>{taskDefinition.requiresCompatibilities.join(', ')}</span>
              </div>
            )}
          </div>
        </div>

        {taskDefinition.containerDefinitions && taskDefinition.containerDefinitions.length > 0 && (
          <div className="detail-card full-width">
            <h2>Container Definitions</h2>
            <div className="container-list">
              {taskDefinition.containerDefinitions.map((container, index) => (
                <div key={index} className="container-item">
                  <h3>{container.name}</h3>
                  <div className="info-grid">
                    <div className="info-item">
                      <label>Image:</label>
                      <span>{container.image}</span>
                    </div>
                    {container.memory && (
                      <div className="info-item">
                        <label>Memory:</label>
                        <span>{container.memory} MB</span>
                      </div>
                    )}
                    {container.cpu && (
                      <div className="info-item">
                        <label>CPU:</label>
                        <span>{container.cpu} units</span>
                      </div>
                    )}
                    <div className="info-item">
                      <label>Essential:</label>
                      <span>{container.essential ? 'Yes' : 'No'}</span>
                    </div>
                  </div>

                  {container.portMappings && container.portMappings.length > 0 && (
                    <div className="sub-section">
                      <h4>Port Mappings</h4>
                      <div className="port-mappings">
                        {container.portMappings.map((port, portIndex) => (
                          <div key={portIndex} className="port-mapping">
                            Container: {port.containerPort} 
                            {port.hostPort && ` → Host: ${port.hostPort}`}
                            {port.protocol && ` (${port.protocol})`}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {container.environment && container.environment.length > 0 && (
                    <div className="sub-section">
                      <h4>Environment Variables</h4>
                      <div className="env-vars">
                        {container.environment.map((env, envIndex) => (
                          <div key={envIndex} className="env-var">
                            <span className="env-name">{env.name}:</span>
                            <span className="env-value">{env.value}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

    </div>
  );
}