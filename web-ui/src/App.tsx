import React from 'react';
import './App.css';
import { useDashboardStats, useHealthStatus } from './hooks/useApi';

function App() {
  const { stats, loading: statsLoading, error: statsError, refresh: refreshStats } = useDashboardStats();
  const { health, refresh: refreshHealth } = useHealthStatus();

  const getStatusIndicatorClass = (status: string) => {
    switch (status) {
      case 'connected':
        return 'status-indicator connected';
      case 'error':
        return 'status-indicator error';
      default:
        return 'status-indicator';
    }
  };

  const formatMetric = (value: number, loading: boolean, error: string | null) => {
    if (loading) return '...';
    if (error) return '!';
    return value.toString();
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>KECS Web UI</h1>
        <p>Kubernetes-based ECS Compatible Service</p>
        <nav>
          <ul>
            <li><a href="#clusters">Clusters</a></li>
            <li><a href="#services">Services</a></li>
            <li><a href="#tasks">Tasks</a></li>
            <li><a href="#task-definitions">Task Definitions</a></li>
          </ul>
        </nav>
      </header>
      
      <main className="App-main">
        <section id="dashboard">
          <div className="dashboard-header">
            <h2>Dashboard</h2>
            <button 
              className="refresh-button" 
              onClick={refreshStats}
              disabled={statsLoading}
              title="Refresh dashboard data"
            >
              {statsLoading ? '⟳' : '↻'}
            </button>
          </div>
          
          {statsError && (
            <div className="error-banner">
              <strong>Error:</strong> {statsError}
              <button onClick={refreshStats}>Retry</button>
            </div>
          )}
          
          <div className="dashboard-cards">
            <div className="card">
              <h3>Clusters</h3>
              <p className="metric">{formatMetric(stats.clusters, statsLoading, statsError)}</p>
              <small>Active clusters</small>
            </div>
            <div className="card">
              <h3>Services</h3>
              <p className="metric">{formatMetric(stats.services, statsLoading, statsError)}</p>
              <small>Running services</small>
            </div>
            <div className="card">
              <h3>Tasks</h3>
              <p className="metric">{formatMetric(stats.tasks, statsLoading, statsError)}</p>
              <small>Active tasks</small>
            </div>
            <div className="card">
              <h3>Task Definitions</h3>
              <p className="metric">{formatMetric(stats.taskDefinitions, statsLoading, statsError)}</p>
              <small>Registered definitions</small>
            </div>
          </div>
        </section>
        
        <section id="status">
          <div className="status-header">
            <h2>System Status</h2>
            <button 
              className="refresh-button" 
              onClick={refreshHealth}
              title="Check connection status"
            >
              ↻
            </button>
          </div>
          <div className="status-info">
            <p>
              <strong>KECS Control Plane:</strong> 
              <span className={getStatusIndicatorClass(health.status)}>
                {health.message}
              </span>
            </p>
            <p><strong>API Endpoint:</strong> http://localhost:8080</p>
            <p><strong>Last Updated:</strong> {new Date(health.timestamp).toLocaleTimeString()}</p>
            <p><strong>Version:</strong> 0.1.0</p>
          </div>
        </section>
      </main>
      
      <footer className="App-footer">
        <p>&copy; 2025 KECS - Kubernetes-based ECS Compatible Service</p>
      </footer>
    </div>
  );
}

export default App;