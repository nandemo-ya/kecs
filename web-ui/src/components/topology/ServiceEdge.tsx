import React from 'react';
import { 
  getBezierPath, 
  EdgeProps, 
  EdgeLabelRenderer, 
  BaseEdge,
  getStraightPath,
  getSmoothStepPath
} from 'reactflow';
import { ServiceEdgeData, getConnectionColor, CONNECTION_STYLES } from '../../types/topology';
import './ServiceEdge.css';

export function ServiceEdge({
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
}: EdgeProps<ServiceEdgeData>) {
  const connectionStyle = CONNECTION_STYLES[data?.connectionType || 'custom'];
  const color = getConnectionColor(data?.connectionType || 'custom');
  
  // Choose path based on connection type
  const getPath = () => {
    if (data?.connectionType === 'database' || data?.connectionType === 'cache') {
      return getSmoothStepPath({
        sourceX,
        sourceY,
        targetX,
        targetY,
        sourcePosition,
        targetPosition,
        borderRadius: 8,
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

  // Format metrics for display
  const formatMetrics = () => {
    const metrics = [];
    if (data?.requestsPerMinute) {
      metrics.push(`${data.requestsPerMinute} req/min`);
    }
    if (data?.latencyMs) {
      metrics.push(`${data.latencyMs}ms`);
    }
    if (data?.errorRate) {
      metrics.push(`${(data.errorRate * 100).toFixed(1)}% errors`);
    }
    return metrics;
  };

  const metrics = formatMetrics();
  const isHighLatency = data?.latencyMs && data.latencyMs > 100;
  const isHighError = data?.errorRate && data.errorRate > 0.05;
  const showTraffic = data?.animated || (data?.requestsPerMinute && data.requestsPerMinute > 0);

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        markerEnd={markerEnd || `url(#${connectionStyle.markerEnd}-${data?.connectionType || 'custom'})`}
        style={{
          stroke: selected ? '#3b82f6' : color,
          strokeWidth: selected ? 3 : 2,
          strokeDasharray: connectionStyle.strokeDasharray,
          strokeOpacity: data?.isHighlighted ? 1 : 0.7,
        }}
      />
      
      {showTraffic && (
        <path
          d={edgePath}
          className="traffic-flow"
          style={{
            stroke: color,
            strokeWidth: 3,
            strokeDasharray: '10 20',
            fill: 'none',
            opacity: 0.3,
          }}
        />
      )}

      <EdgeLabelRenderer>
        <div
          className={`edge-label ${selected ? 'selected' : ''} ${isHighLatency ? 'high-latency' : ''} ${isHighError ? 'high-error' : ''}`}
          style={{
            position: 'absolute',
            transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
            pointerEvents: 'all',
          }}
        >
          {data?.label && (
            <div className="edge-label-text">{data.label}</div>
          )}
          {metrics.length > 0 && (
            <div className="edge-metrics">
              {metrics.map((metric, index) => (
                <span key={index} className="metric">{metric}</span>
              ))}
            </div>
          )}
          {data?.protocol && (
            <div className="edge-protocol">{data.protocol}</div>
          )}
        </div>
      </EdgeLabelRenderer>

      {/* Connection type icon */}
      <EdgeLabelRenderer>
        <div
          className="edge-type-icon"
          style={{
            position: 'absolute',
            transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY - 20}px)`,
          }}
        >
          <span className="connection-icon" title={data?.connectionType}>
            {getConnectionIcon(data?.connectionType || 'custom')}
          </span>
        </div>
      </EdgeLabelRenderer>

      {/* Bidirectional indicator */}
      {data?.trafficFlow === 'bidirectional' && (
        <EdgeLabelRenderer>
          <div
            className="bidirectional-indicator"
            style={{
              position: 'absolute',
              transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY + 20}px)`,
            }}
            title="Bidirectional traffic"
          >
            ↔️
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  );
}

// Helper function to get connection type icon
function getConnectionIcon(type: string): string {
  const icons: Record<string, string> = {
    http: '🌐',
    grpc: '⚡',
    tcp: '🔌',
    database: '🗄️',
    cache: '💾',
    queue: '📬',
    custom: '🔗',
  };
  return icons[type] || icons.custom;
}

// Edge markers definition component
export function EdgeMarkers() {
  const markerTypes = ['http', 'grpc', 'tcp', 'database', 'cache', 'queue', 'custom'];
  
  return (
    <defs>
      {markerTypes.map(type => {
        const color = getConnectionColor(type);
        return (
          <React.Fragment key={type}>
            <marker
              id={`arrow-${type}`}
              viewBox="0 -5 10 10"
              refX="10"
              refY="0"
              markerWidth="5"
              markerHeight="5"
              orient="auto"
            >
              <path
                d="M 0,-5 L 10,0 L 0,5"
                fill={color}
                stroke={color}
              />
            </marker>
            <marker
              id={`arrowclosed-${type}`}
              viewBox="0 -5 10 10"
              refX="10"
              refY="0"
              markerWidth="5"
              markerHeight="5"
              orient="auto"
            >
              <path
                d="M 0,-5 L 10,0 L 0,5 z"
                fill={color}
                stroke={color}
                strokeWidth="1"
              />
            </marker>
          </React.Fragment>
        );
      })}
    </defs>
  );
}