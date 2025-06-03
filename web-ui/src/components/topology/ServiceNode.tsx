import React, { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { ServiceNodeData, SERVICE_TYPE_ICONS, getHealthColor } from '../../types/topology';
import './ServiceNode.css';

export const ServiceNode = memo(({ data, selected }: NodeProps<ServiceNodeData>) => {
  const healthColor = getHealthColor(data.healthStatus);
  const icon = SERVICE_TYPE_ICONS[data.serviceType] || SERVICE_TYPE_ICONS.custom;
  
  const taskRatio = data.desiredCount > 0 ? data.runningCount / data.desiredCount : 0;
  const isFullyDeployed = taskRatio === 1;
  const isPartiallyDeployed = taskRatio > 0 && taskRatio < 1;
  
  return (
    <div 
      className={`service-node ${selected ? 'selected' : ''} ${data.isHighlighted ? 'highlighted' : ''}`}
      style={{
        borderColor: selected ? '#3b82f6' : healthColor,
        boxShadow: selected ? '0 0 0 2px #3b82f6' : 'none',
      }}
    >
      <Handle
        type="target"
        position={Position.Left}
        style={{ background: '#6b7280' }}
      />
      
      <div className="service-node-header">
        <span className="service-icon" title={data.serviceType}>
          {icon}
        </span>
        <div className="service-name">
          <div className="name" title={data.serviceName}>
            {data.serviceName}
          </div>
          <div className="cluster" title={`Cluster: ${data.clusterName}`}>
            {data.clusterName}
          </div>
        </div>
        <div 
          className="health-indicator"
          style={{ backgroundColor: healthColor }}
          title={`Status: ${data.healthStatus}`}
        />
      </div>
      
      <div className="service-node-body">
        <div className="task-info">
          <div className="task-count">
            <span className="running">{data.runningCount}</span>
            <span className="separator">/</span>
            <span className="desired">{data.desiredCount}</span>
            <span className="label">tasks</span>
          </div>
          {data.pendingCount > 0 && (
            <div className="pending-count" title="Pending tasks">
              +{data.pendingCount} pending
            </div>
          )}
        </div>
        
        <div className="deployment-status">
          <div 
            className="status-bar"
            title={`${Math.round(taskRatio * 100)}% deployed`}
          >
            <div 
              className="status-fill"
              style={{
                width: `${taskRatio * 100}%`,
                backgroundColor: isFullyDeployed ? '#10b981' : isPartiallyDeployed ? '#f59e0b' : '#ef4444',
              }}
            />
          </div>
        </div>
        
        {data.launchType && (
          <div className="launch-type" title={`Launch type: ${data.launchType}`}>
            {data.launchType}
          </div>
        )}
      </div>
      
      {data.showDetails && (
        <div className="service-node-details">
          <div className="detail-item">
            <span className="label">Task Definition:</span>
            <span className="value">{data.taskDefinition || 'N/A'}</span>
          </div>
          <div className="detail-item">
            <span className="label">Created:</span>
            <span className="value">
              {new Date(data.createdAt).toLocaleDateString()}
            </span>
          </div>
        </div>
      )}
      
      <Handle
        type="source"
        position={Position.Right}
        style={{ background: '#6b7280' }}
      />
    </div>
  );
});

ServiceNode.displayName = 'ServiceNode';