import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { apiClient } from '../services/api';
import { ListClustersResponse, ListTasksResponse, DescribeTasksResponse } from '../types/api';

interface TaskListItem {
  taskId: string;
  taskArn: string;
  clusterName: string;
  lastStatus: string;
  desiredStatus: string;
  taskDefinitionArn: string;
  launchType: string;
  startedAt?: string;
  startedBy?: string;
  group?: string;
  cpu?: string;
  memory?: string;
  healthStatus?: string;
}

export function TaskList() {
  const [tasks, setTasks] = useState<TaskListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadTasks = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // First get all clusters
      const clustersResponse: ListClustersResponse = await apiClient.listClusters();
      
      if (clustersResponse.clusterArns.length === 0) {
        setTasks([]);
        return;
      }

      // Extract cluster names from ARNs
      const clusterNames = clustersResponse.clusterArns.map(arn => {
        const parts = arn.split('/');
        return parts[parts.length - 1];
      });

      // Get tasks for each cluster
      const allTasks: TaskListItem[] = [];
      
      for (const clusterName of clusterNames) {
        try {
          const tasksResponse: ListTasksResponse = await apiClient.listTasks(clusterName);
          
          if (tasksResponse.taskArns && tasksResponse.taskArns.length > 0) {
            // Get detailed information for tasks in this cluster
            const detailResponse: DescribeTasksResponse = await apiClient.describeTasks(tasksResponse.taskArns, clusterName);
            
            const clusterTasks: TaskListItem[] = detailResponse.tasks.map(task => {
              const taskId = task.taskArn.split('/').pop() || 'unknown';
              
              return {
                taskId,
                taskArn: task.taskArn,
                clusterName: clusterName,
                lastStatus: task.lastStatus,
                desiredStatus: task.desiredStatus,
                taskDefinitionArn: task.taskDefinitionArn,
                launchType: task.launchType,
                startedAt: task.startedAt,
                startedBy: task.startedBy,
                group: task.group,
                cpu: task.cpu,
                memory: task.memory,
                healthStatus: task.healthStatus,
              };
            });

            allTasks.push(...clusterTasks);
          }
        } catch (err) {
          console.warn(`Failed to load tasks for cluster ${clusterName}:`, err);
        }
      }

      setTasks(allTasks);
    } catch (err) {
      console.error('Failed to load tasks:', err);
      setError('Failed to load tasks');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadTasks();
  }, [loadTasks]);

  if (loading) {
    return (
      <main className="App-main">
        <div className="loading">Loading tasks...</div>
      </main>
    );
  }

  if (error) {
    return (
      <main className="App-main">
        <div className="error">
          {error}
          <button onClick={loadTasks}>Retry</button>
        </div>
      </main>
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

  return (
    <main className="App-main">
      <div className="dashboard-header">
        <h2>Tasks</h2>
        <button className="refresh-button" onClick={loadTasks}>
          Refresh
        </button>
      </div>

      {tasks.length === 0 ? (
        <div className="empty-state">
          No tasks found. Run a task to get started.
        </div>
      ) : (
        <div className="resource-list-grid">
          {tasks.map((task) => {
            const taskDefParts = task.taskDefinitionArn.split('/').pop()?.split(':');
            const taskDefFamily = taskDefParts?.[0];
            const taskDefRevision = taskDefParts?.[1];
            
            return (
              <div key={task.taskArn} className="resource-item-card card">
                <div className="resource-header">
                  <h3>
                    <Link 
                      to={`/tasks/${task.taskId}?cluster=${task.clusterName}`} 
                      className="resource-link"
                    >
                      {task.taskId}
                    </Link>
                  </h3>
                  <span className={`status-badge ${getStatusClass(task.lastStatus)}`}>
                    {task.lastStatus}
                  </span>
                </div>
                
                <div className="resource-arn">
                  {task.taskArn}
                </div>

                <div className="resource-info">
                  <div className="info-row">
                    <label>Cluster:</label>
                    <Link to={`/clusters/${task.clusterName}`} className="link">
                      {task.clusterName}
                    </Link>
                  </div>
                  <div className="info-row">
                    <label>Task Definition:</label>
                    <span>{taskDefFamily}:{taskDefRevision}</span>
                  </div>
                  <div className="info-row">
                    <label>Desired Status:</label>
                    <span className={`status ${task.desiredStatus.toLowerCase()}`}>
                      {task.desiredStatus}
                    </span>
                  </div>
                  <div className="info-row">
                    <label>Launch Type:</label>
                    <span>{task.launchType}</span>
                  </div>
                  {task.startedBy && (
                    <div className="info-row">
                      <label>Started By:</label>
                      {task.startedBy.startsWith('service:') ? (
                        <Link 
                          to={`/services/${task.startedBy.replace('service:', '')}?cluster=${task.clusterName}`}
                          className="link"
                        >
                          {task.startedBy.replace('service:', '')}
                        </Link>
                      ) : (
                        <span>{task.startedBy}</span>
                      )}
                    </div>
                  )}
                  {task.startedAt && (
                    <div className="info-row">
                      <label>Started:</label>
                      <span>{new Date(task.startedAt).toLocaleString()}</span>
                    </div>
                  )}
                </div>

                <div className="resource-metrics">
                  {task.cpu && (
                    <div className="metric-item">
                      <div className="metric-value">{task.cpu}</div>
                      <div className="metric-label">CPU</div>
                    </div>
                  )}
                  {task.memory && (
                    <div className="metric-item">
                      <div className="metric-value">{task.memory}</div>
                      <div className="metric-label">Memory</div>
                    </div>
                  )}
                  <div className="metric-item">
                    <div className="metric-value">{task.launchType}</div>
                    <div className="metric-label">Launch Type</div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </main>
  );
}