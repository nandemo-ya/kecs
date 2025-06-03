import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { apiClient } from '../services/api';
import { UpdateServiceRequest, DescribeServicesResponse, ListTaskDefinitionsResponse } from '../types/api';

interface FormData {
  desiredCount: number;
  taskDefinition: string;
  platformVersion: string;
}

export function UpdateService() {
  const navigate = useNavigate();
  const { serviceName } = useParams<{ serviceName: string }>();
  const [searchParams] = useSearchParams();
  const clusterName = searchParams.get('cluster') || 'default';

  const [formData, setFormData] = useState<FormData>({
    desiredCount: 1,
    taskDefinition: '',
    platformVersion: '',
  });

  const [currentService, setCurrentService] = useState<any>(null);
  const [taskDefinitions, setTaskDefinitions] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [submitLoading, setSubmitLoading] = useState(false);

  const loadServiceData = useCallback(async () => {
    try {
      const response: DescribeServicesResponse = await apiClient.describeServices([serviceName || ''], clusterName);
      const service = response.services[0];
      
      if (service) {
        setCurrentService(service);
        setFormData({
          desiredCount: service.desiredCount || 1,
          taskDefinition: service.taskDefinition,
          platformVersion: service.platformVersion || '',
        });
      } else {
        setError('Service not found');
      }
    } catch (err) {
      setError('Failed to load service data');
      console.error('Failed to load service:', err);
    }
  }, [serviceName, clusterName]);

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
    Promise.all([loadServiceData(), loadTaskDefinitions()])
      .finally(() => setLoading(false));
  }, [loadServiceData, loadTaskDefinitions]);

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

    try {
      const request: UpdateServiceRequest = {
        cluster: clusterName,
        service: serviceName || '',
        desiredCount: formData.desiredCount,
        taskDefinition: formData.taskDefinition,
        platformVersion: formData.platformVersion || undefined,
      };

      await apiClient.updateService(request);
      
      // Navigate back to service detail page
      navigate(`/services/${serviceName}?cluster=${clusterName}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update service');
    } finally {
      setSubmitLoading(false);
    }
  };

  const isFormValid = () => {
    return formData.taskDefinition && formData.desiredCount >= 0;
  };

  const hasChanges = () => {
    if (!currentService) return false;
    
    return formData.desiredCount !== currentService.desiredCount ||
           formData.taskDefinition !== currentService.taskDefinition ||
           (formData.platformVersion !== (currentService.platformVersion || ''));
  };

  if (loading) {
    return (
      <main className="App-main">
        <div className="loading">Loading service data...</div>
      </main>
    );
  }

  if (error && !currentService) {
    return (
      <main className="App-main">
        <div className="error">
          {error}
          <button onClick={() => navigate('/services')}>← Back to Services</button>
        </div>
      </main>
    );
  }

  return (
    <main className="App-main">
      <div className="form-page">
        <div className="form-header">
          <h2>Update Service: {serviceName}</h2>
          <button 
            type="button" 
            className="back-button"
            onClick={() => navigate(`/services/${serviceName}?cluster=${clusterName}`)}
          >
            ← Back to Service Details
          </button>
        </div>

        {error && (
          <div className="error-banner">
            <strong>Error:</strong> {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="service-form">
          <div className="form-section">
            <h3>Service Information</h3>
            <div className="form-info">
              <div className="info-item">
                <label>Service Name:</label>
                <span>{serviceName}</span>
              </div>
              <div className="info-item">
                <label>Cluster:</label>
                <span>{clusterName}</span>
              </div>
              <div className="info-item">
                <label>Status:</label>
                <span className={`status ${currentService?.status?.toLowerCase()}`}>
                  {currentService?.status}
                </span>
              </div>
            </div>
          </div>

          <div className="form-section">
            <h3>Update Configuration</h3>
            
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
              <small className="form-help">
                Current: {currentService?.desiredCount}
              </small>
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
              <small className="form-help">
                Current: {currentService?.taskDefinition}
              </small>
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
              <small className="form-help">
                Current: {currentService?.platformVersion || 'Not set'}
              </small>
            </div>
          </div>

          <div className="form-actions">
            <button
              type="button"
              className="btn btn-secondary"
              onClick={() => navigate(`/services/${serviceName}?cluster=${clusterName}`)}
            >
              Cancel
            </button>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={!isFormValid() || !hasChanges() || submitLoading}
            >
              {submitLoading ? 'Updating...' : 'Update Service'}
            </button>
          </div>
        </form>
      </div>
    </main>
  );
}