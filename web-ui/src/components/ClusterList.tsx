import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { apiClient } from '../services/api';
import { ListClustersResponse, DescribeClustersResponse } from '../types/api';

interface ClusterListItem {
  name: string;
  arn: string;
  status: string;
  runningTasksCount: number;
  activeServicesCount: number;
  registeredContainerInstancesCount: number;
  pendingTasksCount: number;
}

export function ClusterList() {
  const [clusters, setClusters] = useState<ClusterListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadClusters();
  }, []);

  const loadClusters = async () => {
    try {
      setLoading(true);
      setError(null);

      // First get the list of cluster ARNs
      const listResponse: ListClustersResponse = await apiClient.listClusters();
      
      if (listResponse.clusterArns.length === 0) {
        setClusters([]);
        return;
      }

      // Extract cluster names from ARNs
      const clusterNames = listResponse.clusterArns.map(arn => {
        const parts = arn.split('/');
        return parts[parts.length - 1];
      });

      // Get detailed information for all clusters
      const describeResponse: DescribeClustersResponse = await apiClient.describeClusters(clusterNames);
      
      const clusterList: ClusterListItem[] = describeResponse.clusters.map(cluster => ({
        name: cluster.clusterName,
        arn: cluster.clusterArn,
        status: cluster.status,
        runningTasksCount: cluster.runningTasksCount || 0,
        activeServicesCount: cluster.activeServicesCount || 0,
        registeredContainerInstancesCount: cluster.registeredContainerInstancesCount || 0,
        pendingTasksCount: cluster.pendingTasksCount || 0,
      }));

      setClusters(clusterList);
    } catch (err) {
      console.error('Failed to load clusters:', err);
      setError('Failed to load clusters');
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <main className="App-main">
        <div className="loading">Loading clusters...</div>
      </main>
    );
  }

  if (error) {
    return (
      <main className="App-main">
        <div className="error">
          {error}
          <button onClick={loadClusters}>Retry</button>
        </div>
      </main>
    );
  }

  return (
    <main className="App-main">
      <div className="dashboard-header">
        <h2>Clusters</h2>
        <button className="refresh-button" onClick={loadClusters}>
          Refresh
        </button>
      </div>

      {clusters.length === 0 ? (
        <div className="empty-state">
          No clusters found. Create a cluster to get started.
        </div>
      ) : (
        <div className="cluster-list">
          {clusters.map((cluster) => (
            <div key={cluster.arn} className="cluster-item card">
              <div className="cluster-header">
                <h3>
                  <Link to={`/clusters/${cluster.name}`} className="cluster-link">
                    {cluster.name}
                  </Link>
                </h3>
                <span className={`status-badge status-${cluster.status.toLowerCase()}`}>
                  {cluster.status}
                </span>
              </div>
              
              <div className="cluster-arn">
                {cluster.arn}
              </div>

              <div className="cluster-metrics">
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
          ))}
        </div>
      )}
    </main>
  );
}