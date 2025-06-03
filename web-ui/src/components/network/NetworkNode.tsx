import React, { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { 
  NetworkNodeData, 
  NETWORK_NODE_ICONS, 
  getNodeTypeColor, 
  getCriticalityColor,
  SECURITY_ICONS 
} from '../../types/networkDependencies';
import './NetworkNode.css';

export const NetworkNode = memo(({ data, selected }: NodeProps<NetworkNodeData>) => {
  const typeColor = getNodeTypeColor(data.type);
  const criticalityColor = getCriticalityColor(data.criticality);
  const nodeIcon = NETWORK_NODE_ICONS[data.type] || NETWORK_NODE_ICONS.service;
  
  const getStatusIcon = () => {
    switch (data.status) {
      case 'active':
        return '🟢';
      case 'inactive':
        return '🔴';
      case 'degraded':
        return '🟡';
      default:
        return '⚪';
    }
  };
  
  const getSecurityIcon = () => {
    const { security } = data;
    if (security.encrypted && security.authentication && security.authorization && security.firewall) {
      return SECURITY_ICONS.secure;
    } else if (!security.encrypted || !security.authentication) {
      return SECURITY_ICONS.vulnerable;
    } else {
      return SECURITY_ICONS.warning;
    }
  };
  
  const getSecurityLevel = () => {
    const { security } = data;
    const secureFeatures = [
      security.encrypted,
      security.authentication,
      security.authorization,
      security.firewall
    ].filter(Boolean).length;
    
    if (secureFeatures === 4) return 'secure';
    if (secureFeatures >= 2) return 'warning';
    return 'vulnerable';
  };

  return (
    <div 
      className={`network-node ${selected ? 'selected' : ''} ${data.isHighlighted ? 'highlighted' : ''}`}
      data-node-type={data.type}
      data-criticality={data.criticality}
      data-status={data.status}
      style={{
        borderColor: selected ? '#3b82f6' : typeColor,
        boxShadow: selected ? '0 0 0 2px #3b82f6' : 'none',
      }}
    >
      <Handle
        type="target"
        position={Position.Left}
        style={{ background: '#6b7280' }}
      />
      
      {/* Node Header */}
      <div className="network-node-header">
        <div className="node-type-info">
          <span className="node-icon" style={{ color: typeColor }} title={data.type}>
            {nodeIcon}
          </span>
          <div className="node-name-info">
            <div className="node-name" title={data.name}>
              {data.name}
            </div>
            {data.cluster && (
              <div className="node-cluster" title={`Cluster: ${data.cluster}`}>
                {data.cluster}
              </div>
            )}
          </div>
        </div>
        <div className="node-status-indicators">
          <span className="status-indicator" title={`Status: ${data.status}`}>
            {getStatusIcon()}
          </span>
          <span 
            className={`security-indicator ${getSecurityLevel()}`}
            title={`Security: ${getSecurityLevel()}`}
          >
            {getSecurityIcon()}
          </span>
        </div>
      </div>
      
      {/* Node Body */}
      <div className="network-node-body">
        {/* Connection Info */}
        <div className="connection-info">
          {data.ip && (
            <div className="connection-detail">
              <span className="label">IP:</span>
              <span className="value">{data.ip}</span>
            </div>
          )}
          {data.port && (
            <div className="connection-detail">
              <span className="label">Port:</span>
              <span className="value">{data.port}</span>
            </div>
          )}
          {data.protocol && (
            <div className="connection-detail">
              <span className="label">Protocol:</span>
              <span className="value">{data.protocol}</span>
            </div>
          )}
        </div>
        
        {/* Criticality Badge */}
        <div className="criticality-section">
          <div 
            className={`criticality-badge ${data.criticality}`}
            style={{ backgroundColor: criticalityColor }}
            title={`Criticality: ${data.criticality}`}
          >
            {data.criticality.toUpperCase()}
          </div>
        </div>
        
        {/* Flow Metrics */}
        {data.flowMetrics && (
          <div className="flow-metrics">
            <div className="flow-metric">
              <span className="metric-icon">⬇️</span>
              <span className="metric-value">{data.flowMetrics.inbound}</span>
            </div>
            <div className="flow-metric">
              <span className="metric-icon">⬆️</span>
              <span className="metric-value">{data.flowMetrics.outbound}</span>
            </div>
            {data.flowMetrics.errors > 0 && (
              <div className="flow-metric error">
                <span className="metric-icon">❌</span>
                <span className="metric-value">{data.flowMetrics.errors}</span>
              </div>
            )}
          </div>
        )}
        
        {/* Namespace Badge */}
        {data.namespace && (
          <div className="namespace-badge" title={`Namespace: ${data.namespace}`}>
            {data.namespace}
          </div>
        )}
      </div>
      
      {/* Security Details (when expanded) */}
      {data.showDetails && (
        <div className="network-node-details">
          <div className="security-details">
            <h5>Security Features</h5>
            <div className="security-features">
              <div className={`security-feature ${data.security.encrypted ? 'enabled' : 'disabled'}`}>
                <span className="feature-icon">{data.security.encrypted ? '🔒' : '🔓'}</span>
                <span>Encryption</span>
              </div>
              <div className={`security-feature ${data.security.authentication ? 'enabled' : 'disabled'}`}>
                <span className="feature-icon">{data.security.authentication ? '👤' : '👤'}</span>
                <span>Auth</span>
              </div>
              <div className={`security-feature ${data.security.authorization ? 'enabled' : 'disabled'}`}>
                <span className="feature-icon">{data.security.authorization ? '🛡️' : '🛡️'}</span>
                <span>AuthZ</span>
              </div>
              <div className={`security-feature ${data.security.firewall ? 'enabled' : 'disabled'}`}>
                <span className="feature-icon">{data.security.firewall ? '🔥' : '🔥'}</span>
                <span>Firewall</span>
              </div>
            </div>
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

NetworkNode.displayName = 'NetworkNode';