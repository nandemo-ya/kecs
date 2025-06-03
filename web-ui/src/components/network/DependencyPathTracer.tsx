import React, { useState } from 'react';
import {
  NetworkNode,
  NetworkDependency,
} from '../../types/networkDependencies';

interface DependencyPathTracerProps {
  nodes: NetworkNode[];
  dependencies: NetworkDependency[];
  onPathTrace: (sourceId: string, targetId: string) => void;
  onClose: () => void;
}

export function DependencyPathTracer({
  nodes,
  dependencies,
  onPathTrace,
  onClose,
}: DependencyPathTracerProps) {
  const [sourceId, setSourceId] = useState('');
  const [targetId, setTargetId] = useState('');

  const handleTrace = () => {
    if (sourceId && targetId) {
      onPathTrace(sourceId, targetId);
    }
  };

  return (
    <div className="dependency-path-tracer">
      <div className="tracer-header">
        <h3>Path Tracer</h3>
        <button className="close-button" onClick={onClose}>
          ✕
        </button>
      </div>
      <div className="tracer-content">
        <div className="path-selection">
          <div className="selection-group">
            <label>Source Node:</label>
            <select value={sourceId} onChange={(e) => setSourceId(e.target.value)}>
              <option value="">Select source...</option>
              {nodes.map(node => (
                <option key={node.id} value={node.id}>
                  {node.name}
                </option>
              ))}
            </select>
          </div>
          <div className="selection-group">
            <label>Target Node:</label>
            <select value={targetId} onChange={(e) => setTargetId(e.target.value)}>
              <option value="">Select target...</option>
              {nodes.map(node => (
                <option key={node.id} value={node.id}>
                  {node.name}
                </option>
              ))}
            </select>
          </div>
          <button 
            className="trace-button" 
            onClick={handleTrace}
            disabled={!sourceId || !targetId}
          >
            Trace Path
          </button>
        </div>
      </div>
    </div>
  );
}