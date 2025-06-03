import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { apiClient } from '../services/api';
import { ListTaskDefinitionsResponse, DescribeTaskDefinitionResponse } from '../types/api';

interface TaskDefinitionListItem {
  family: string;
  revision: number;
  taskDefinitionArn: string;
  status: string;
  cpu?: string;
  memory?: string;
  networkMode?: string;
  requiresCompatibilities?: string[];
  registeredAt?: string;
}

export function TaskDefinitionList() {
  const [taskDefinitions, setTaskDefinitions] = useState<TaskDefinitionListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadTaskDefinitions = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // Get list of task definition ARNs
      const listResponse: ListTaskDefinitionsResponse = await apiClient.listTaskDefinitions();
      
      if (listResponse.taskDefinitionArns.length === 0) {
        setTaskDefinitions([]);
        return;
      }

      // Get detailed information for each task definition
      const allTaskDefinitions: TaskDefinitionListItem[] = [];
      
      for (const taskDefArn of listResponse.taskDefinitionArns) {
        try {
          const detailResponse: DescribeTaskDefinitionResponse = await apiClient.describeTaskDefinition(taskDefArn);
          
          if (detailResponse.taskDefinition) {
            const taskDef = detailResponse.taskDefinition;
            
            allTaskDefinitions.push({
              family: taskDef.family,
              revision: taskDef.revision,
              taskDefinitionArn: taskDef.taskDefinitionArn,
              status: taskDef.status,
              cpu: taskDef.cpu,
              memory: taskDef.memory,
              networkMode: taskDef.networkMode,
              requiresCompatibilities: taskDef.requiresCompatibilities,
              registeredAt: taskDef.registeredAt,
            });
          }
        } catch (err) {
          console.warn(`Failed to load task definition ${taskDefArn}:`, err);
        }
      }

      setTaskDefinitions(allTaskDefinitions);
    } catch (err) {
      console.error('Failed to load task definitions:', err);
      setError('Failed to load task definitions');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTaskDefinitions();
  }, [loadTaskDefinitions]);

  if (loading) {
    return (
      <main className="App-main">
        <div className="loading">Loading task definitions...</div>
      </main>
    );
  }

  if (error) {
    return (
      <main className="App-main">
        <div className="error">
          {error}
          <button onClick={loadTaskDefinitions}>Retry</button>
        </div>
      </main>
    );
  }

  const getStatusClass = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active': return 'status-active';
      case 'inactive': return 'status-stopped';
      default: return 'status-unknown';
    }
  };

  return (
    <main className="App-main">
      <div className="dashboard-header">
        <h2>Task Definitions</h2>
        <div className="header-actions">
          <Link to="/task-definitions/register" className="btn btn-primary">
            Register Task Definition
          </Link>
          <button className="refresh-button" onClick={loadTaskDefinitions}>
            Refresh
          </button>
        </div>
      </div>

      {taskDefinitions.length === 0 ? (
        <div className="empty-state">
          No task definitions found. Register a task definition to get started.
        </div>
      ) : (
        <div className="resource-list-grid">
          {taskDefinitions.map((taskDef) => (
            <div key={taskDef.taskDefinitionArn} className="resource-item-card card">
              <div className="resource-header">
                <h3>
                  <Link 
                    to={`/task-definitions/${taskDef.family}/${taskDef.revision}`} 
                    className="resource-link"
                  >
                    {taskDef.family}:{taskDef.revision}
                  </Link>
                </h3>
                <span className={`status-badge ${getStatusClass(taskDef.status)}`}>
                  {taskDef.status}
                </span>
              </div>
              
              <div className="resource-arn">
                {taskDef.taskDefinitionArn}
              </div>

              <div className="resource-info">
                <div className="info-row">
                  <label>Family:</label>
                  <span>{taskDef.family}</span>
                </div>
                <div className="info-row">
                  <label>Revision:</label>
                  <span>{taskDef.revision}</span>
                </div>
                {taskDef.networkMode && (
                  <div className="info-row">
                    <label>Network Mode:</label>
                    <span>{taskDef.networkMode}</span>
                  </div>
                )}
                {taskDef.requiresCompatibilities && taskDef.requiresCompatibilities.length > 0 && (
                  <div className="info-row">
                    <label>Compatibility:</label>
                    <span>{taskDef.requiresCompatibilities.join(', ')}</span>
                  </div>
                )}
                {taskDef.registeredAt && (
                  <div className="info-row">
                    <label>Registered:</label>
                    <span>{new Date(taskDef.registeredAt).toLocaleDateString()}</span>
                  </div>
                )}
              </div>

              <div className="resource-metrics">
                {taskDef.cpu && (
                  <div className="metric-item">
                    <div className="metric-value">{taskDef.cpu}</div>
                    <div className="metric-label">CPU</div>
                  </div>
                )}
                {taskDef.memory && (
                  <div className="metric-item">
                    <div className="metric-value">{taskDef.memory}</div>
                    <div className="metric-label">Memory</div>
                  </div>
                )}
                <div className="metric-item">
                  <div className="metric-value">{taskDef.revision}</div>
                  <div className="metric-label">Revision</div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </main>
  );
}