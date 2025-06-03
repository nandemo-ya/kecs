import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { apiClient } from '../services/api';
import { CreateServiceRequest, ListClustersResponse, ListTaskDefinitionsResponse } from '../types/api';
import { useOperationNotification } from '../hooks/useOperationNotification';

interface FormData {
  cluster: string;
  serviceName: string;
  taskDefinition: string;
  desiredCount: number;
  launchType: string;
  platformVersion: string;
  schedulingStrategy: string;
}

export function CreateService() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const initialCluster = searchParams.get('cluster') || '';
  const { executeWithNotification } = useOperationNotification();

  const [formData, setFormData] = useState<FormData>({
    cluster: initialCluster,
    serviceName: '',
    taskDefinition: '',
    desiredCount: 1,
    launchType: 'FARGATE',
    platformVersion: 'LATEST',
    schedulingStrategy: 'REPLICA',
  });

  const [clusters, setClusters] = useState<string[]>([]);
  const [taskDefinitions, setTaskDefinitions] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitLoading, setSubmitLoading] = useState(false);

  const loadClusters = useCallback(async () => {
    try {
      const response: ListClustersResponse = await apiClient.listClusters();
      const clusterNames = response.clusterArns.map(arn => {
        const parts = arn.split('/');
        return parts[parts.length - 1];
      });
      setClusters(clusterNames);
    } catch (err) {
      console.error('Failed to load clusters:', err);
    }
  }, []);

  const loadTaskDefinitions = useCallback(async () => {
    try {
      const response: ListTaskDefinitionsResponse = await apiClient.listTaskDefinitions();
      const taskDefNames = response.taskDefinitionArns.map(arn => {
        const parts = arn.split('/');
        return parts[parts.length - 1];
      });
      setTaskDefinitions(taskDefNames);
    } catch (err) {
      console.error('Failed to load task definitions:', err);
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    Promise.all([loadClusters(), loadTaskDefinitions()])
      .finally(() => setLoading(false));
  }, [loadClusters, loadTaskDefinitions]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: name === 'desiredCount' ? parseInt(value) || 0 : value,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitLoading(true);
    setError(null);

    const result = await executeWithNotification(
      async () => {
        const request: CreateServiceRequest = {
          cluster: formData.cluster,
          serviceName: formData.serviceName,
          taskDefinition: formData.taskDefinition,
          desiredCount: formData.desiredCount,
          launchType: formData.launchType,
          platformVersion: formData.platformVersion,
          schedulingStrategy: formData.schedulingStrategy,
        };

        return await apiClient.createService(request);
      },
      {
        inProgressTitle: 'Creating Service',
        inProgressMessage: `Creating service "${formData.serviceName}" in cluster "${formData.cluster}"...`,
        successTitle: 'Service Created Successfully',
        successMessage: `Service "${formData.serviceName}" has been created in cluster "${formData.cluster}".`,
        errorTitle: 'Failed to Create Service',
      }
    );

    setSubmitLoading(false);

    if (result) {
      // Navigate to service detail page
      navigate(`/services/${formData.serviceName}?cluster=${formData.cluster}`);
    }
  };

  const isFormValid = () => {
    return formData.cluster && 
           formData.serviceName && 
           formData.taskDefinition && 
           formData.desiredCount > 0;
  };

  if (loading) {
    return (
      <main className="App-main">
        <div className="loading">Loading form data...</div>
      </main>
    );
  }

  return (
    <main className="App-main">
      <div className="form-page">
        <div className="form-header">
          <h2>Create Service</h2>
          <button 
            type="button" 
            className="back-button"
            onClick={() => navigate('/services')}
          >
            ← Back to Services
          </button>
        </div>

        {error && (
          <div className="error-banner">
            <strong>Error:</strong> {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="service-form">
          <div className="form-section">
            <h3>Basic Configuration</h3>
            
            <div className="form-group">
              <label htmlFor="cluster">Cluster *</label>
              <select
                id="cluster"
                name="cluster"
                value={formData.cluster}
                onChange={handleInputChange}
                required
                className="form-control"
              >
                <option value="">Select a cluster</option>
                {clusters.map(cluster => (
                  <option key={cluster} value={cluster}>
                    {cluster}
                  </option>
                ))}
              </select>
            </div>

            <div className="form-group">
              <label htmlFor="serviceName">Service Name *</label>
              <input
                id="serviceName"
                name="serviceName"
                type="text"
                value={formData.serviceName}
                onChange={handleInputChange}
                required
                className="form-control"
                placeholder="Enter service name"
              />
            </div>

            <div className="form-group">
              <label htmlFor="taskDefinition">Task Definition *</label>
              <select
                id="taskDefinition"
                name="taskDefinition"
                value={formData.taskDefinition}
                onChange={handleInputChange}
                required
                className="form-control"
              >
                <option value="">Select a task definition</option>
                {taskDefinitions.map(taskDef => (
                  <option key={taskDef} value={taskDef}>
                    {taskDef}
                  </option>
                ))}
              </select>
            </div>

            <div className="form-group">
              <label htmlFor="desiredCount">Desired Count *</label>
              <input
                id="desiredCount"
                name="desiredCount"
                type="number"
                min="0"
                value={formData.desiredCount}
                onChange={handleInputChange}
                required
                className="form-control"
              />
            </div>
          </div>

          <div className="form-section">
            <h3>Launch Configuration</h3>
            
            <div className="form-group">
              <label htmlFor="launchType">Launch Type</label>
              <select
                id="launchType"
                name="launchType"
                value={formData.launchType}
                onChange={handleInputChange}
                className="form-control"
              >
                <option value="FARGATE">Fargate</option>
                <option value="EC2">EC2</option>
              </select>
            </div>

            <div className="form-group">
              <label htmlFor="platformVersion">Platform Version</label>
              <input
                id="platformVersion"
                name="platformVersion"
                type="text"
                value={formData.platformVersion}
                onChange={handleInputChange}
                className="form-control"
                placeholder="LATEST"
              />
            </div>

            <div className="form-group">
              <label htmlFor="schedulingStrategy">Scheduling Strategy</label>
              <select
                id="schedulingStrategy"
                name="schedulingStrategy"
                value={formData.schedulingStrategy}
                onChange={handleInputChange}
                className="form-control"
              >
                <option value="REPLICA">Replica</option>
                <option value="DAEMON">Daemon</option>
              </select>
            </div>
          </div>

          <div className="form-actions">
            <button
              type="button"
              className="btn btn-secondary"
              onClick={() => navigate('/services')}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={!isFormValid() || submitLoading}
            >
              {submitLoading ? 'Creating...' : 'Create Service'}
            </button>
          </div>
        </form>
      </div>
    </main>
  );
}