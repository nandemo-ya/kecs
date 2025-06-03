import React from 'react';
import { Link } from 'react-router-dom';
import { useDashboardStats, useHealthStatus, useApiData } from '../hooks/useApi';
import { apiClient } from '../services/api';
import { useAutoRefresh } from '../hooks/useAutoRefresh';
import { AutoRefreshToggle } from './AutoRefreshToggle';

export function Dashboard() {
  const { stats, loading: statsLoading, error: statsError, refresh: refreshStats } = useDashboardStats();
  const { health, refresh: refreshHealth } = useHealthStatus();
  
  const refreshAll = () => {
    refreshStats();
    refreshHealth();
  };
  
  const { isAutoRefreshEnabled, isRefreshing, toggleRefresh } = useAutoRefresh(refreshAll, {
    interval: 10000, // 10 seconds for dashboard
  });
  
  // Get actual cluster and service data for linking
  const { data: clusters } = useApiData(() => apiClient.listClusters(), []);
  const { data: services } = useApiData(async () => {
    // First get clusters, then get services for the first cluster
    const clusterResponse = await apiClient.listClusters();
    if (clusterResponse.clusterArns && clusterResponse.clusterArns.length > 0) {
      const firstClusterName = clusterResponse.clusterArns[0].split('/').pop();
      return apiClient.listServices(firstClusterName);
    }
    return { serviceArns: [] };
  }, []);

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
    <main className="App-main">
      <section id="dashboard">
        <div className="dashboard-header">
          <h2>Dashboard</h2>
          <div className="header-actions">
            <AutoRefreshToggle
              isEnabled={isAutoRefreshEnabled}
              isRefreshing={isRefreshing}
              onToggle={toggleRefresh}
              interval={10000}
            />
            <button 
              className="refresh-button" 
              onClick={refreshAll}
              disabled={statsLoading}
              title="Refresh dashboard data"
            >
              {statsLoading ? '⟳' : '↻'}
            </button>
          </div>
        </div>
        
        {statsError && (
          <div className="error-banner">
            <strong>Error:</strong> {statsError}
            <button onClick={refreshStats}>Retry</button>
          </div>
        )}
        
        <div className="dashboard-cards">
          <Link to="/clusters" className="card card-link">
            <h3>Clusters</h3>
            <p className="metric">{formatMetric(stats.clusters, statsLoading, statsError)}</p>
            <small>Active clusters</small>
          </Link>
          <Link to="/services" className="card card-link">
            <h3>Services</h3>
            <p className="metric">{formatMetric(stats.services, statsLoading, statsError)}</p>
            <small>Running services</small>
          </Link>
          <Link to="/tasks" className="card card-link">
            <h3>Tasks</h3>
            <p className="metric">{formatMetric(stats.tasks, statsLoading, statsError)}</p>
            <small>Active tasks</small>
          </Link>
          <Link to="/task-definitions" className="card card-link">
            <h3>Task Definitions</h3>
            <p className="metric">{formatMetric(stats.taskDefinitions, statsLoading, statsError)}</p>
            <small>Registered definitions</small>
          </Link>
        </div>
        
        {/* Quick Access Section */}
        <div className="quick-access">
          <h3>Quick Access</h3>
          <div className="quick-links">
            {(clusters?.clusterArns || []).slice(0, 3).map((clusterArn) => {
              const clusterName = clusterArn.split('/').pop();
              return (
                <Link 
                  key={clusterArn} 
                  to={`/clusters/${clusterName}`}
                  className="quick-link"
                >
                  📦 {clusterName}
                </Link>
              );
            })}
            {(services?.serviceArns || []).slice(0, 3).map((serviceArn) => {
              const serviceName = serviceArn.split('/').pop();
              return (
                <Link 
                  key={serviceArn} 
                  to={`/services/${serviceName}`}
                  className="quick-link"
                >
                  🚀 {serviceName}
                </Link>
              );
            })}
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
  );
}