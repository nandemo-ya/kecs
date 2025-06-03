import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { apiClient } from '../services/api';
import { ContainerDefinition, PortMapping, EnvironmentVariable } from '../types/api';

export function RegisterTaskDefinition() {
  const navigate = useNavigate();
  const [submitLoading, setSubmitLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const [formData, setFormData] = useState({
    family: '',
    cpu: '256',
    memory: '512',
    networkMode: 'awsvpc',
    requiresCompatibilities: 'FARGATE',
    containers: [{
      name: '',
      image: '',
      memory: 512,
      cpu: 256,
      essential: true,
      portMappings: [] as PortMapping[],
      environment: [] as EnvironmentVariable[],
    }] as ContainerDefinition[],
  });

  const handleAddContainer = () => {
    setFormData({
      ...formData,
      containers: [...formData.containers, {
        name: '',
        image: '',
        memory: 512,
        cpu: 256,
        essential: true,
        portMappings: [],
        environment: [],
      }],
    });
  };

  const handleRemoveContainer = (index: number) => {
    setFormData({
      ...formData,
      containers: formData.containers.filter((_, i) => i !== index),
    });
  };

  const handleContainerChange = (index: number, field: keyof ContainerDefinition, value: any) => {
    const updatedContainers = [...formData.containers];
    updatedContainers[index] = {
      ...updatedContainers[index],
      [field]: value,
    };
    setFormData({
      ...formData,
      containers: updatedContainers,
    });
  };

  const handleAddPortMapping = (containerIndex: number) => {
    const updatedContainers = [...formData.containers];
    updatedContainers[containerIndex].portMappings = [
      ...(updatedContainers[containerIndex].portMappings || []),
      { containerPort: 80, protocol: 'tcp' },
    ];
    setFormData({
      ...formData,
      containers: updatedContainers,
    });
  };

  const handleRemovePortMapping = (containerIndex: number, portIndex: number) => {
    const updatedContainers = [...formData.containers];
    updatedContainers[containerIndex].portMappings = 
      updatedContainers[containerIndex].portMappings?.filter((_, i) => i !== portIndex) || [];
    setFormData({
      ...formData,
      containers: updatedContainers,
    });
  };

  const handlePortMappingChange = (containerIndex: number, portIndex: number, field: keyof PortMapping, value: any) => {
    const updatedContainers = [...formData.containers];
    const portMappings = [...(updatedContainers[containerIndex].portMappings || [])];
    portMappings[portIndex] = {
      ...portMappings[portIndex],
      [field]: value,
    };
    updatedContainers[containerIndex].portMappings = portMappings;
    setFormData({
      ...formData,
      containers: updatedContainers,
    });
  };

  const handleAddEnvironmentVariable = (containerIndex: number) => {
    const updatedContainers = [...formData.containers];
    updatedContainers[containerIndex].environment = [
      ...(updatedContainers[containerIndex].environment || []),
      { name: '', value: '' },
    ];
    setFormData({
      ...formData,
      containers: updatedContainers,
    });
  };

  const handleRemoveEnvironmentVariable = (containerIndex: number, envIndex: number) => {
    const updatedContainers = [...formData.containers];
    updatedContainers[containerIndex].environment = 
      updatedContainers[containerIndex].environment?.filter((_, i) => i !== envIndex) || [];
    setFormData({
      ...formData,
      containers: updatedContainers,
    });
  };

  const handleEnvironmentVariableChange = (containerIndex: number, envIndex: number, field: keyof EnvironmentVariable, value: string) => {
    const updatedContainers = [...formData.containers];
    const environment = [...(updatedContainers[containerIndex].environment || [])];
    environment[envIndex] = {
      ...environment[envIndex],
      [field]: value,
    };
    updatedContainers[containerIndex].environment = environment;
    setFormData({
      ...formData,
      containers: updatedContainers,
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitLoading(true);
    setError(null);

    try {
      await apiClient.registerTaskDefinition({
        family: formData.family,
        cpu: formData.cpu,
        memory: formData.memory,
        networkMode: formData.networkMode,
        requiresCompatibilities: formData.requiresCompatibilities.split(',').map(s => s.trim()),
        containerDefinitions: formData.containers.map(container => ({
          ...container,
          memory: Number(container.memory),
          cpu: container.cpu ? Number(container.cpu) : undefined,
          portMappings: container.portMappings?.map(pm => ({
            ...pm,
            containerPort: Number(pm.containerPort),
            hostPort: pm.hostPort ? Number(pm.hostPort) : undefined,
          })),
        })),
      });

      navigate('/task-definitions');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to register task definition');
    } finally {
      setSubmitLoading(false);
    }
  };

  return (
    <div className="form-page">
      <div className="form-header">
        <h2>Register Task Definition</h2>
        <Link to="/task-definitions" className="back-button">
          Back to Task Definitions
        </Link>
      </div>

      {error && (
        <div className="error-banner">
          {error}
          <button onClick={() => setError(null)}>×</button>
        </div>
      )}

      <form onSubmit={handleSubmit} className="service-form">
        <div className="form-section">
          <h3>Basic Configuration</h3>
          <div className="form-group">
            <label htmlFor="family">Family Name</label>
            <input
              type="text"
              id="family"
              className="form-control"
              value={formData.family}
              onChange={(e) => setFormData({ ...formData, family: e.target.value })}
              required
            />
            <span className="form-help">
              Unique name for your task definition family
            </span>
          </div>

          <div className="form-group">
            <label htmlFor="networkMode">Network Mode</label>
            <select
              id="networkMode"
              className="form-control"
              value={formData.networkMode}
              onChange={(e) => setFormData({ ...formData, networkMode: e.target.value })}
            >
              <option value="awsvpc">awsvpc</option>
              <option value="bridge">bridge</option>
              <option value="host">host</option>
              <option value="none">none</option>
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="requiresCompatibilities">Launch Type</label>
            <select
              id="requiresCompatibilities"
              className="form-control"
              value={formData.requiresCompatibilities}
              onChange={(e) => setFormData({ ...formData, requiresCompatibilities: e.target.value })}
            >
              <option value="FARGATE">Fargate</option>
              <option value="EC2">EC2</option>
              <option value="FARGATE,EC2">Both</option>
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="cpu">CPU (units)</label>
            <select
              id="cpu"
              className="form-control"
              value={formData.cpu}
              onChange={(e) => setFormData({ ...formData, cpu: e.target.value })}
            >
              <option value="256">256 (.25 vCPU)</option>
              <option value="512">512 (.5 vCPU)</option>
              <option value="1024">1024 (1 vCPU)</option>
              <option value="2048">2048 (2 vCPU)</option>
              <option value="4096">4096 (4 vCPU)</option>
            </select>
          </div>

          <div className="form-group">
            <label htmlFor="memory">Memory (MB)</label>
            <select
              id="memory"
              className="form-control"
              value={formData.memory}
              onChange={(e) => setFormData({ ...formData, memory: e.target.value })}
            >
              <option value="512">512 MB</option>
              <option value="1024">1 GB</option>
              <option value="2048">2 GB</option>
              <option value="4096">4 GB</option>
              <option value="8192">8 GB</option>
              <option value="16384">16 GB</option>
              <option value="32768">32 GB</option>
            </select>
          </div>
        </div>

        {formData.containers.map((container, containerIndex) => (
          <div key={containerIndex} className="form-section">
            <div className="form-header" style={{ marginBottom: '1rem' }}>
              <h3>Container {containerIndex + 1}</h3>
              {formData.containers.length > 1 && (
                <button
                  type="button"
                  className="btn btn-danger"
                  onClick={() => handleRemoveContainer(containerIndex)}
                >
                  Remove Container
                </button>
              )}
            </div>

            <div className="form-group">
              <label>Container Name</label>
              <input
                type="text"
                className="form-control"
                value={container.name}
                onChange={(e) => handleContainerChange(containerIndex, 'name', e.target.value)}
                required
              />
            </div>

            <div className="form-group">
              <label>Image</label>
              <input
                type="text"
                className="form-control"
                value={container.image}
                onChange={(e) => handleContainerChange(containerIndex, 'image', e.target.value)}
                placeholder="nginx:latest"
                required
              />
            </div>

            <div className="form-group">
              <label>Memory (MB)</label>
              <input
                type="number"
                className="form-control"
                value={container.memory}
                onChange={(e) => handleContainerChange(containerIndex, 'memory', e.target.value)}
                min="128"
                required
              />
            </div>

            <div className="form-group">
              <label>CPU (units)</label>
              <input
                type="number"
                className="form-control"
                value={container.cpu || ''}
                onChange={(e) => handleContainerChange(containerIndex, 'cpu', e.target.value)}
                min="128"
              />
              <span className="form-help">Optional. CPU units (1024 = 1 vCPU)</span>
            </div>

            <div className="form-group">
              <label>Essential</label>
              <select
                className="form-control"
                value={container.essential ? 'true' : 'false'}
                onChange={(e) => handleContainerChange(containerIndex, 'essential', e.target.value === 'true')}
              >
                <option value="true">Yes</option>
                <option value="false">No</option>
              </select>
            </div>

            <div className="form-section" style={{ marginTop: '1rem', padding: '1rem', background: '#f9fafb' }}>
              <h4>Port Mappings</h4>
              {container.portMappings?.map((portMapping, portIndex) => (
                <div key={portIndex} style={{ display: 'flex', gap: '1rem', marginBottom: '0.5rem' }}>
                  <input
                    type="number"
                    className="form-control"
                    placeholder="Container Port"
                    value={portMapping.containerPort}
                    onChange={(e) => handlePortMappingChange(containerIndex, portIndex, 'containerPort', e.target.value)}
                    style={{ flex: 1 }}
                  />
                  <input
                    type="number"
                    className="form-control"
                    placeholder="Host Port (optional)"
                    value={portMapping.hostPort || ''}
                    onChange={(e) => handlePortMappingChange(containerIndex, portIndex, 'hostPort', e.target.value)}
                    style={{ flex: 1 }}
                  />
                  <select
                    className="form-control"
                    value={portMapping.protocol || 'tcp'}
                    onChange={(e) => handlePortMappingChange(containerIndex, portIndex, 'protocol', e.target.value)}
                    style={{ flex: 1 }}
                  >
                    <option value="tcp">TCP</option>
                    <option value="udp">UDP</option>
                  </select>
                  <button
                    type="button"
                    className="btn btn-danger"
                    onClick={() => handleRemovePortMapping(containerIndex, portIndex)}
                  >
                    Remove
                  </button>
                </div>
              ))}
              <button
                type="button"
                className="btn btn-secondary"
                onClick={() => handleAddPortMapping(containerIndex)}
                style={{ marginTop: '0.5rem' }}
              >
                Add Port Mapping
              </button>
            </div>

            <div className="form-section" style={{ marginTop: '1rem', padding: '1rem', background: '#f9fafb' }}>
              <h4>Environment Variables</h4>
              {container.environment?.map((envVar, envIndex) => (
                <div key={envIndex} style={{ display: 'flex', gap: '1rem', marginBottom: '0.5rem' }}>
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Name"
                    value={envVar.name}
                    onChange={(e) => handleEnvironmentVariableChange(containerIndex, envIndex, 'name', e.target.value)}
                    style={{ flex: 1 }}
                  />
                  <input
                    type="text"
                    className="form-control"
                    placeholder="Value"
                    value={envVar.value}
                    onChange={(e) => handleEnvironmentVariableChange(containerIndex, envIndex, 'value', e.target.value)}
                    style={{ flex: 1 }}
                  />
                  <button
                    type="button"
                    className="btn btn-danger"
                    onClick={() => handleRemoveEnvironmentVariable(containerIndex, envIndex)}
                  >
                    Remove
                  </button>
                </div>
              ))}
              <button
                type="button"
                className="btn btn-secondary"
                onClick={() => handleAddEnvironmentVariable(containerIndex)}
                style={{ marginTop: '0.5rem' }}
              >
                Add Environment Variable
              </button>
            </div>
          </div>
        ))}

        <div className="form-section">
          <button
            type="button"
            className="btn btn-secondary"
            onClick={handleAddContainer}
          >
            Add Container
          </button>
        </div>

        <div className="form-actions">
          <Link to="/task-definitions" className="btn btn-secondary">
            Cancel
          </Link>
          <button
            type="submit"
            disabled={submitLoading || !formData.family || formData.containers.length === 0}
            className="btn btn-primary"
          >
            {submitLoading ? 'Registering...' : 'Register Task Definition'}
          </button>
        </div>
      </form>
    </div>
  );
}