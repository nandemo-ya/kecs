import React from 'react';
import { 
  getBezierPath, 
  EdgeProps, 
  EdgeLabelRenderer, 
  BaseEdge,
  getSmoothStepPath
} from 'reactflow';
import { 
  NetworkDependencyData, 
  getDependencyTypeColor, 
  getSecurityColor,
  PROTOCOL_ICONS 
} from '../../types/networkDependencies';
import './NetworkEdge.css';

export function NetworkEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
  markerEnd,
  selected,
}: EdgeProps<NetworkDependencyData>) {
  const dependencyColor = getDependencyTypeColor(data?.dependencyType || 'custom');
  const securityColor = getSecurityColor(data?.security?.encrypted || false);
  const protocolIcon = PROTOCOL_ICONS[data?.protocol || 'Custom'] || PROTOCOL_ICONS.Custom;
  
  // Choose path based on dependency type
  const getPath = () => {
    if (data?.dependencyType === 'database_query' || data?.dependencyType === 'cache_access') {
      return getSmoothStepPath({
        sourceX,
        sourceY,
        targetX,
        targetY,
        sourcePosition,
        targetPosition,
        borderRadius: 12,
      });
    }
    return getBezierPath({
      sourceX,
      sourceY,
      targetX,
      targetY,
      sourcePosition,
      targetPosition,
    });
  };

  const [edgePath, labelX, labelY] = getPath();

  // Determine edge styling based on properties
  const getStrokeWidth = () => {
    if (selected) return 4;
    switch (data?.strength) {
      case 'critical': return 4;
      case 'strong': return 3;
      case 'moderate': return 2;
      case 'weak': return 1;
      default: return 2;
    }
  };

  const getStrokeDashArray = () => {
    if (!data?.security?.encrypted) return '8 4';
    switch (data?.dependencyType) {
      case 'event_stream': return '12 6';
      case 'cache_access': return '6 3';
      default: return '0';
    }
  };

  const getOpacity = () => {
    if (data?.isHighlighted) return 1;
    switch (data?.strength) {
      case 'critical': return 1;
      case 'strong': return 0.9;
      case 'moderate': return 0.7;
      case 'weak': return 0.5;
      default: return 0.7;
    }
  };

  // Format metrics for display
  const formatMetrics = () => {
    const metrics = [];
    if (data?.frequency) {
      const unit = data.frequency > 1000 ? 'k/min' : '/min';
      const value = data.frequency > 1000 ? (data.frequency / 1000).toFixed(1) : data.frequency;
      metrics.push(`${value}${unit}`);
    }
    if (data?.latency) {
      metrics.push(`${data.latency}ms`);
    }
    if (data?.errorRate && data.errorRate > 0) {
      metrics.push(`${(data.errorRate * 100).toFixed(1)}% err`);
    }
    if (data?.bandwidth) {
      const bandwidth = formatBandwidth(data.bandwidth);
      metrics.push(bandwidth);
    }
    return metrics;
  };

  const formatBandwidth = (bytes: number): string => {
    if (bytes < 1024) return `${bytes}B/s`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB/s`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)}MB/s`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)}GB/s`;
  };

  const isHighLatency = data?.latency && data.latency > 100;
  const isHighError = data?.errorRate && data.errorRate > 0.05;
  const isHighTraffic = data?.frequency && data.frequency > 1000;
  const isSecure = data?.security?.encrypted && data?.security?.authenticated;
  const showAnimation = data?.isAnimated || isHighTraffic;

  const metrics = formatMetrics();

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        markerEnd={markerEnd}
        style={{
          stroke: selected ? '#3b82f6' : (data?.security?.encrypted ? dependencyColor : securityColor),
          strokeWidth: getStrokeWidth(),
          strokeDasharray: getStrokeDashArray(),
          strokeOpacity: getOpacity(),
        }}
      />
      
      {/* Traffic flow animation */}
      {showAnimation && (
        <path
          d={edgePath}
          className="traffic-animation"
          style={{
            stroke: dependencyColor,
            strokeWidth: 2,
            strokeDasharray: '6 12',
            fill: 'none',
            opacity: 0.6,
            animation: data?.flowDirection === 'reverse' ? 
              'traffic-reverse 2s linear infinite' : 
              'traffic-forward 2s linear infinite',
          }}
        />
      )}

      {/* Security warning overlay */}
      {!data?.security?.encrypted && (
        <path
          d={edgePath}
          className="security-warning"
          style={{
            stroke: '#ef4444',
            strokeWidth: 1,
            strokeDasharray: '4 8',
            fill: 'none',
            opacity: 0.7,
            animation: 'security-pulse 3s ease-in-out infinite',
          }}
        />
      )}

      {/* Edge label with metrics */}
      <EdgeLabelRenderer>
        <div
          className={`network-edge-label ${selected ? 'selected' : ''} ${isHighLatency ? 'high-latency' : ''} ${isHighError ? 'high-error' : ''} ${isSecure ? 'secure' : 'insecure'}`}
          style={{
            position: 'absolute',
            transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
            pointerEvents: 'all',
          }}
        >
          {/* Protocol and dependency type */}
          <div className="edge-header">
            <span className="protocol-icon" title={data?.protocol}>
              {protocolIcon}
            </span>
            <span className="dependency-type" title={data?.dependencyType}>
              {data?.dependencyType?.replace('_', ' ')}
            </span>
            {data?.port && (
              <span className="port-info" title={`Port ${data.port}`}>
                :{data.port}
              </span>
            )}
          </div>
          
          {/* Metrics */}
          {metrics.length > 0 && (
            <div className="edge-metrics">
              {metrics.map((metric, index) => (
                <span key={index} className="metric">{metric}</span>
              ))}
            </div>
          )}
          
          {/* Security indicators */}
          <div className="security-indicators">
            {data?.security?.encrypted && (
              <span className="security-icon secure" title="Encrypted">🔒</span>
            )}
            {!data?.security?.encrypted && (
              <span className="security-icon insecure" title="Unencrypted">🔓</span>
            )}
            {data?.security?.authenticated && (
              <span className="security-icon" title="Authenticated">🔐</span>
            )}
          </div>
          
          {/* Strength indicator */}
          <div className={`strength-indicator ${data?.strength || 'moderate'}`}>
            <div className="strength-bar"></div>
          </div>
        </div>
      </EdgeLabelRenderer>

      {/* Bidirectional indicator */}
      {data?.direction === 'bidirectional' && (
        <EdgeLabelRenderer>
          <div
            className="bidirectional-indicator"
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY + 25}px)`,
            }}
            title="Bidirectional dependency"
          >
            ↔️
          </div>
        </EdgeLabelRenderer>
      )}

      {/* SLA indicator */}
      {data?.sla && (
        <EdgeLabelRenderer>
          <div
            className="sla-indicator"
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX + 30}px, ${labelY - 15}px)`,
            }}
            title={`SLA: ${data.sla.availability}% uptime, ${data.sla.responseTime}ms response`}
          >
            <span className="sla-badge">SLA</span>
          </div>
        </EdgeLabelRenderer>
      )}

      {/* Error rate warning */}
      {isHighError && (
        <EdgeLabelRenderer>
          <div
            className="error-warning"
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX - 30}px, ${labelY - 15}px)`,
            }}
            title={`High error rate: ${(data!.errorRate! * 100).toFixed(1)}%`}
          >
            ⚠️
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  );
}

// Edge markers for different dependency types
export function NetworkEdgeMarkers() {
  const dependencyTypes = ['api_call', 'database_query', 'file_access', 'event_stream', 'cache_access', 'load_balance', 'proxy', 'custom'];
  
  return (
    <defs>
      {dependencyTypes.map(type => {
        const color = getDependencyTypeColor(type);
        return (
          <React.Fragment key={type}>
            <marker
              id={`network-arrow-${type}`}
              viewBox="0 -5 10 10"
              refX="10"
              refY="0"
              markerWidth="6"
              markerHeight="6"
              orient="auto"
            >
              <path
                d="M 0,-5 L 10,0 L 0,5"
                fill={color}
                stroke={color}
                strokeWidth="1"
              />
            </marker>
            <marker
              id={`network-arrow-secure-${type}`}
              viewBox="0 -5 10 10"
              refX="10"
              refY="0"
              markerWidth="6"
              markerHeight="6"
              orient="auto"
            >
              <path
                d="M 0,-5 L 10,0 L 0,5 z"
                fill={color}
                stroke="#10b981"
                strokeWidth="1"
              />
            </marker>
          </React.Fragment>
        );
      })}
      
      {/* Special markers for security states */}
      <marker
        id="network-arrow-insecure"
        viewBox="0 -5 10 10"
        refX="10"
        refY="0"
        markerWidth="6"
        markerHeight="6"
        orient="auto"
      >
        <path
          d="M 0,-5 L 10,0 L 0,5"
          fill="#ef4444"
          stroke="#ef4444"
          strokeWidth="1"
        />
      </marker>
    </defs>
  );
}