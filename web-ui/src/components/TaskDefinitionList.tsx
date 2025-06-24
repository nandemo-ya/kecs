import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { apiClient } from '../services/api';
import { ListTaskDefinitionsResponse, DescribeTaskDefinitionResponse } from '../types/api';
import { useOperationNotification } from '../hooks/useOperationNotification';

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
  const [selectedTaskDefs, setSelectedTaskDefs] = useState<Set<string>>(new Set());
  const [deleteLoading, setDeleteLoading] = useState(false);
  const { executeWithNotification } = useOperationNotification();

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

  const handleSelectTaskDef = (taskDefArn: string) => {
    const newSelected = new Set(selectedTaskDefs);
    if (newSelected.has(taskDefArn)) {
      newSelected.delete(taskDefArn);
    } else {
      newSelected.add(taskDefArn);
    }
    setSelectedTaskDefs(newSelected);
  };

  const handleSelectAll = () => {
    if (selectedTaskDefs.size === taskDefinitions.length) {
      setSelectedTaskDefs(new Set());
    } else {
      setSelectedTaskDefs(new Set(taskDefinitions.map(td => td.taskDefinitionArn)));
    }
  };

  const handleDeleteSelected = async () => {
    if (selectedTaskDefs.size === 0) return;

    const confirmed = window.confirm(
      `Are you sure you want to delete ${selectedTaskDefs.size} task definition(s)? This action cannot be undone.`
    );

    if (!confirmed) return;

    setDeleteLoading(true);

    const result = await executeWithNotification(
      async () => {
        const response = await apiClient.deleteTaskDefinitions({
          taskDefinitions: Array.from(selectedTaskDefs),
        });
        
        // Filter out successfully deleted task definitions
        if (response.taskDefinitions) {
          const deletedArns = new Set(response.taskDefinitions.map(td => td.taskDefinitionArn));
          setTaskDefinitions(prevDefs => prevDefs.filter(td => !deletedArns.has(td.taskDefinitionArn)));
          setSelectedTaskDefs(new Set());
        }
        
        return response;
      },
      {
        inProgressTitle: 'Deleting Task Definitions',
        inProgressMessage: `Deleting ${selectedTaskDefs.size} task definition(s)...`,
        successTitle: 'Task Definitions Deleted',
        successMessage: `Successfully deleted ${selectedTaskDefs.size} task definition(s).`,
        errorTitle: 'Failed to Delete Task Definitions',
      }
    );

    setDeleteLoading(false);
  };

  return (
    <main className="App-main">
      <div className="dashboard-header">
        <h2>Task Definitions</h2>
        <div className="header-actions">
          {selectedTaskDefs.size > 0 && (
            <button 
              className="btn btn-danger" 
              onClick={handleDeleteSelected}
              disabled={deleteLoading}
            >
              {deleteLoading ? 'Deleting...' : `Delete Selected (${selectedTaskDefs.size})`}
            </button>
          )}
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
        <>
          {taskDefinitions.length > 0 && (
            <div className="batch-select-all">
              <label>
                <input
                  type="checkbox"
                  checked={selectedTaskDefs.size === taskDefinitions.length}
                  onChange={handleSelectAll}
                />
                Select All
              </label>
            </div>
          )}
          <div className="resource-list-grid">
            {taskDefinitions.map((taskDef) => (
              <div key={taskDef.taskDefinitionArn} className="resource-item-card card">
                <div className="resource-header">
                  <div className="resource-select">
                    <input
                      type="checkbox"
                      checked={selectedTaskDefs.has(taskDef.taskDefinitionArn)}
                      onChange={() => handleSelectTaskDef(taskDef.taskDefinitionArn)}
                    />
                  </div>
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
        </>
      )}
    </main>
  );
}