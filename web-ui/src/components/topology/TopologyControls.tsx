import React, { useState } from 'react';
import { TopologyViewOptions, LayoutAlgorithm } from '../../types/topology';
import './TopologyControls.css';

interface TopologyControlsProps {
  viewOptions: TopologyViewOptions;
  onViewOptionsChange: (options: Partial<TopologyViewOptions>) => void;
  onLayoutChange: (layout: LayoutAlgorithm) => void;
  onRefresh: () => void;
  onExport: () => void;
}

export function TopologyControls({
  viewOptions,
  onViewOptionsChange,
  onLayoutChange,
  onRefresh,
  onExport,
}: TopologyControlsProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const layoutOptions: { value: LayoutAlgorithm; label: string; icon: string }[] = [
    { value: 'hierarchical', label: 'Hierarchical', icon: '🌳' },
    { value: 'force', label: 'Force', icon: '⚡' },
    { value: 'circular', label: 'Circular', icon: '⭕' },
    { value: 'grid', label: 'Grid', icon: '⚏' },
    { value: 'manual', label: 'Manual', icon: '✋' },
  ];

  const handleToggle = (key: keyof TopologyViewOptions) => {
    onViewOptionsChange({ [key]: !viewOptions[key] });
  };

  return (
    <div className={`topology-controls ${isExpanded ? 'expanded' : ''}`}>
      <div className="controls-header">
        <button
          className="controls-toggle"
          onClick={() => setIsExpanded(!isExpanded)}
          title={isExpanded ? 'Collapse controls' : 'Expand controls'}
        >
          <span className="toggle-icon">{isExpanded ? '◀' : '▶'}</span>
          <span className="toggle-label">Controls</span>
        </button>

        <div className="controls-quick-actions">
          <button
            className="control-button"
            onClick={onRefresh}
            title="Refresh topology"
          >
            <span className="button-icon">🔄</span>
          </button>
          <button
            className="control-button"
            onClick={onExport}
            title="Export topology"
          >
            <span className="button-icon">📥</span>
          </button>
        </div>
      </div>

      {isExpanded && (
        <div className="controls-body">
          {/* Layout Selection */}
          <div className="control-section">
            <h4 className="section-title">Layout</h4>
            <div className="layout-options">
              {layoutOptions.map(option => (
                <button
                  key={option.value}
                  className={`layout-button ${viewOptions.layout === option.value ? 'active' : ''}`}
                  onClick={() => onLayoutChange(option.value)}
                  title={option.label}
                >
                  <span className="layout-icon">{option.icon}</span>
                  <span className="layout-label">{option.label}</span>
                </button>
              ))}
            </div>
          </div>

          {/* Display Options */}
          <div className="control-section">
            <h4 className="section-title">Display Options</h4>
            <div className="control-options">
              <label className="control-checkbox">
                <input
                  type="checkbox"
                  checked={viewOptions.showHealthStatus}
                  onChange={() => handleToggle('showHealthStatus')}
                />
                <span>Health Status</span>
              </label>
              <label className="control-checkbox">
                <input
                  type="checkbox"
                  checked={viewOptions.showTaskCounts}
                  onChange={() => handleToggle('showTaskCounts')}
                />
                <span>Task Counts</span>
              </label>
              <label className="control-checkbox">
                <input
                  type="checkbox"
                  checked={viewOptions.showConnections}
                  onChange={() => handleToggle('showConnections')}
                />
                <span>Connections</span>
              </label>
              <label className="control-checkbox">
                <input
                  type="checkbox"
                  checked={viewOptions.showTrafficFlow}
                  onChange={() => handleToggle('showTrafficFlow')}
                />
                <span>Traffic Flow</span>
              </label>
              <label className="control-checkbox">
                <input
                  type="checkbox"
                  checked={viewOptions.showLatency}
                  onChange={() => handleToggle('showLatency')}
                />
                <span>Latency</span>
              </label>
            </div>
          </div>

          {/* Auto Refresh */}
          <div className="control-section">
            <h4 className="section-title">Auto Refresh</h4>
            <label className="control-checkbox">
              <input
                type="checkbox"
                checked={viewOptions.autoRefresh}
                onChange={() => handleToggle('autoRefresh')}
              />
              <span>Enable Auto Refresh</span>
            </label>
            {viewOptions.autoRefresh && (
              <div className="refresh-interval">
                <label>
                  <span>Interval:</span>
                  <select
                    value={viewOptions.refreshInterval}
                    onChange={(e) => onViewOptionsChange({ refreshInterval: Number(e.target.value) })}
                  >
                    <option value={10000}>10 seconds</option>
                    <option value={30000}>30 seconds</option>
                    <option value={60000}>1 minute</option>
                    <option value={300000}>5 minutes</option>
                  </select>
                </label>
              </div>
            )}
          </div>

          {/* Filters */}
          <div className="control-section">
            <h4 className="section-title">Filters</h4>
            <div className="filter-group">
              <label>
                <span>Cluster:</span>
                <select
                  value={viewOptions.filterByCluster || ''}
                  onChange={(e) => onViewOptionsChange({ 
                    filterByCluster: e.target.value || undefined 
                  })}
                >
                  <option value="">All Clusters</option>
                  <option value="production">Production</option>
                  <option value="staging">Staging</option>
                  <option value="development">Development</option>
                </select>
              </label>
            </div>

            <div className="filter-group">
              <span>Service Type:</span>
              <div className="filter-chips">
                {['web', 'api', 'database', 'cache', 'queue', 'storage'].map(type => (
                  <button
                    key={type}
                    className={`filter-chip ${
                      viewOptions.filterByServiceType?.includes(type) ? 'active' : ''
                    }`}
                    onClick={() => {
                      const current = viewOptions.filterByServiceType || [];
                      const newTypes = current.includes(type)
                        ? current.filter(t => t !== type)
                        : [...current, type];
                      onViewOptionsChange({ 
                        filterByServiceType: newTypes.length > 0 ? newTypes : undefined 
                      });
                    }}
                  >
                    {type}
                  </button>
                ))}
              </div>
            </div>

            <div className="filter-group">
              <span>Health Status:</span>
              <div className="filter-chips">
                {['healthy', 'unhealthy', 'degraded', 'unknown'].map(status => (
                  <button
                    key={status}
                    className={`filter-chip ${
                      viewOptions.filterByHealth?.includes(status) ? 'active' : ''
                    }`}
                    onClick={() => {
                      const current = viewOptions.filterByHealth || [];
                      const newStatuses = current.includes(status)
                        ? current.filter(s => s !== status)
                        : [...current, status];
                      onViewOptionsChange({ 
                        filterByHealth: newStatuses.length > 0 ? newStatuses : undefined 
                      });
                    }}
                  >
                    {status}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}