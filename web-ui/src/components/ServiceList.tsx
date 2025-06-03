import React, { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { apiClient } from '../services/api';
import { ListClustersResponse, ListServicesResponse, DescribeServicesResponse } from '../types/api';

interface ServiceListItem {
  serviceName: string;
  serviceArn: string;
  clusterName: string;
  status: string;
  desiredCount: number;
  runningCount: number;
  pendingCount: number;
  taskDefinition: string;
  launchType: string;
  createdAt?: string;
}

export function ServiceList() {
  const [services, setServices] = useState<ServiceListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadServices = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // First get all clusters
      const clustersResponse: ListClustersResponse = await apiClient.listClusters();
      
      if (clustersResponse.clusterArns.length === 0) {
        setServices([]);
        return;
      }

      // Extract cluster names from ARNs
      const clusterNames = clustersResponse.clusterArns.map(arn => {
        const parts = arn.split('/');
        return parts[parts.length - 1];
      });

      // Get services for each cluster
      const allServices: ServiceListItem[] = [];
      
      for (const clusterName of clusterNames) {
        try {
          const servicesResponse: ListServicesResponse = await apiClient.listServices(clusterName);
          
          if (servicesResponse.serviceArns && servicesResponse.serviceArns.length > 0) {
            // Get service names from ARNs
            const serviceNames = servicesResponse.serviceArns.map(arn => {
              const parts = arn.split('/');
              return parts[parts.length - 1];
            });

            // Get detailed information for services in this cluster
            const detailResponse: DescribeServicesResponse = await apiClient.describeServices(serviceNames, clusterName);
            
            const clusterServices: ServiceListItem[] = detailResponse.services.map(service => ({
              serviceName: service.serviceName,
              serviceArn: service.serviceArn,
              clusterName: clusterName,
              status: service.status,
              desiredCount: service.desiredCount || 0,
              runningCount: service.runningCount || 0,
              pendingCount: service.pendingCount || 0,
              taskDefinition: service.taskDefinition,
              launchType: service.launchType,
              createdAt: service.createdAt,
            }));

            allServices.push(...clusterServices);
          }
        } catch (err) {
          console.warn(`Failed to load services for cluster ${clusterName}:`, err);
        }
      }

      setServices(allServices);
    } catch (err) {
      console.error('Failed to load services:', err);
      setError('Failed to load services');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadServices();
  }, [loadServices]);

  if (loading) {
    return (
      <main className="App-main">
        <div className="loading">Loading services...</div>
      </main>
    );
  }

  if (error) {
    return (
      <main className="App-main">
        <div className="error">
          {error}
          <button onClick={loadServices}>Retry</button>
        </div>
      </main>
    );
  }

  const getStatusClass = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active': return 'status-active';
      case 'pending': return 'status-pending';
      case 'draining': return 'status-draining';
      default: return 'status-unknown';
    }
  };

  return (
    <main className="App-main">
      <div className="dashboard-header">
        <h2>Services</h2>
        <div className="header-actions">
          <Link to="/services/create" className="btn btn-primary">
            + Create Service
          </Link>
          <button className="refresh-button" onClick={loadServices}>
            Refresh
          </button>
        </div>
      </div>

      {services.length === 0 ? (
        <div className="empty-state">
          No services found. Create a service to get started.
        </div>
      ) : (
        <div className="resource-list-grid">
          {services.map((service) => {
            const taskDefParts = service.taskDefinition.split('/').pop()?.split(':');
            const taskDefFamily = taskDefParts?.[0];
            const taskDefRevision = taskDefParts?.[1];
            
            return (
              <div key={service.serviceArn} className="resource-item-card card">
                <div className="resource-header">
                  <h3>
                    <Link 
                      to={`/services/${service.serviceName}?cluster=${service.clusterName}`} 
                      className="resource-link"
                    >
                      {service.serviceName}
                    </Link>
                  </h3>
                  <span className={`status-badge ${getStatusClass(service.status)}`}>
                    {service.status}
                  </span>
                </div>
                
                <div className="resource-arn">
                  {service.serviceArn}
                </div>

                <div className="resource-info">
                  <div className="info-row">
                    <label>Cluster:</label>
                    <Link to={`/clusters/${service.clusterName}`} className="link">
                      {service.clusterName}
                    </Link>
                  </div>
                  <div className="info-row">
                    <label>Task Definition:</label>
                    <span>{taskDefFamily}:{taskDefRevision}</span>
                  </div>
                  <div className="info-row">
                    <label>Launch Type:</label>
                    <span>{service.launchType}</span>
                  </div>
                  {service.createdAt && (
                    <div className="info-row">
                      <label>Created:</label>
                      <span>{new Date(service.createdAt).toLocaleDateString()}</span>
                    </div>
                  )}
                </div>

                <div className="resource-metrics">
                  <div className="metric-item">
                    <div className="metric-value">{service.desiredCount}</div>
                    <div className="metric-label">Desired</div>
                  </div>
                  <div className="metric-item">
                    <div className="metric-value">{service.runningCount}</div>
                    <div className="metric-label">Running</div>
                  </div>
                  <div className="metric-item">
                    <div className="metric-value">{service.pendingCount}</div>
                    <div className="metric-label">Pending</div>
                  </div>
                  <div className="metric-item">
                    <div className="metric-value">
                      {Math.round((service.runningCount / service.desiredCount) * 100) || 0}%
                    </div>
                    <div className="metric-label">Health</div>
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