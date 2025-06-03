import React, { useState } from 'react';
import {
  NetworkNode,
  NetworkDependency,
  DependencyPath,
  ImpactAnalysis,
  DependencyAnalysisOptions,
} from '../../types/networkDependencies';
import './NetworkAnalysisPanel.css';

interface NetworkAnalysisPanelProps {
  nodes: NetworkNode[];
  dependencies: NetworkDependency[];
  criticalPaths: DependencyPath[];
  impactAnalysis: ImpactAnalysis | null;
  analysisOptions: DependencyAnalysisOptions;
  onAnalysisOptionsChange: (options: Partial<DependencyAnalysisOptions>) => void;
  onClose: () => void;
  onPathHighlight: (paths: DependencyPath[]) => void;
}

export function NetworkAnalysisPanel({
  nodes,
  dependencies,
  criticalPaths,
  impactAnalysis,
  analysisOptions,
  onAnalysisOptionsChange,
  onClose,
  onPathHighlight,
}: NetworkAnalysisPanelProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'paths' | 'impact' | 'settings'>('overview');

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
    }).format(amount);
  };

  return (
    <div className="network-analysis-panel">
      <div className="analysis-header">
        <h3>Network Analysis</h3>
        <button className="close-button" onClick={onClose} title="Close">
          ✕
        </button>
      </div>

      <div className="analysis-tabs">
        <button
          className={`tab ${activeTab === 'overview' ? 'active' : ''}`}
          onClick={() => setActiveTab('overview')}
        >
          Overview
        </button>
        <button
          className={`tab ${activeTab === 'paths' ? 'active' : ''}`}
          onClick={() => setActiveTab('paths')}
        >
          Critical Paths
        </button>
        <button
          className={`tab ${activeTab === 'impact' ? 'active' : ''}`}
          onClick={() => setActiveTab('impact')}
        >
          Impact Analysis
        </button>
        <button
          className={`tab ${activeTab === 'settings' ? 'active' : ''}`}
          onClick={() => setActiveTab('settings')}
        >
          Settings
        </button>
      </div>

      <div className="analysis-content">
        {activeTab === 'overview' && (
          <div className="tab-content">
            <div className="overview-metrics">
              <div className="metric-card">
                <div className="metric-value">{nodes.length}</div>
                <div className="metric-label">Total Nodes</div>
              </div>
              <div className="metric-card">
                <div className="metric-value">{dependencies.length}</div>
                <div className="metric-label">Dependencies</div>
              </div>
              <div className="metric-card">
                <div className="metric-value">{criticalPaths.length}</div>
                <div className="metric-label">Critical Paths</div>
              </div>
              <div className="metric-card">
                <div className="metric-value">
                  {nodes.filter(n => n.criticality === 'critical').length}
                </div>
                <div className="metric-label">Critical Nodes</div>
              </div>
            </div>

            <div className="network-health">
              <h4>Network Health</h4>
              <div className="health-indicators">
                <div className="health-item">
                  <span className="health-label">Active Nodes:</span>
                  <span className="health-value">
                    {nodes.filter(n => n.status === 'active').length} / {nodes.length}
                  </span>
                  <div className="health-bar">
                    <div 
                      className="health-fill active"
                      style={{ width: `${(nodes.filter(n => n.status === 'active').length / nodes.length) * 100}%` }}
                    />
                  </div>
                </div>
                <div className="health-item">
                  <span className="health-label">Secure Connections:</span>
                  <span className="health-value">
                    {dependencies.filter(d => d.security.encrypted).length} / {dependencies.length}
                  </span>
                  <div className="health-bar">
                    <div 
                      className="health-fill secure"
                      style={{ width: `${(dependencies.filter(d => d.security.encrypted).length / dependencies.length) * 100}%` }}
                    />
                  </div>
                </div>
                <div className="health-item">
                  <span className="health-label">Avg Latency:</span>
                  <span className="health-value">
                    {Math.round(dependencies.reduce((sum, d) => sum + d.latency, 0) / dependencies.length || 0)}ms
                  </span>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'paths' && (
          <div className="tab-content">
            <div className="paths-header">
              <h4>Critical Paths ({criticalPaths.length})</h4>
              <button
                className="highlight-all-button"
                onClick={() => onPathHighlight(criticalPaths)}
              >
                Highlight All
              </button>
            </div>
            <div className="critical-paths-list">
              {criticalPaths.map((path, index) => (
                <div key={path.id} className={`path-item ${path.type}`}>
                  <div className="path-header">
                    <span className="path-index">#{index + 1}</span>
                    <span className={`path-type ${path.type}`}>
                      {path.type.replace('_', ' ')}
                    </span>
                    <button
                      className="highlight-path-button"
                      onClick={() => onPathHighlight([path])}
                      title="Highlight this path"
                    >
                      🔍
                    </button>
                  </div>
                  <div className="path-details">
                    <div className="path-nodes">
                      {path.path.map((nodeId, nodeIndex) => (
                        <React.Fragment key={nodeId}>
                          <span className="path-node">
                            {nodes.find(n => n.id === nodeId)?.name || nodeId}
                          </span>
                          {nodeIndex < path.path.length - 1 && (
                            <span className="path-arrow">→</span>
                          )}
                        </React.Fragment>
                      ))}
                    </div>
                    <div className="path-metrics">
                      <span className="metric">
                        Length: {path.length}
                      </span>
                      <span className="metric">
                        Latency: {path.totalLatency}ms
                      </span>
                      <span className="metric">
                        Reliability: {path.reliability.toFixed(1)}%
                      </span>
                    </div>
                    {path.bottlenecks.length > 0 && (
                      <div className="path-bottlenecks">
                        <span className="bottleneck-label">Bottlenecks:</span>
                        {path.bottlenecks.map(bottleneck => (
                          <span key={bottleneck} className="bottleneck">
                            {nodes.find(n => n.id === bottleneck)?.name || bottleneck}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {activeTab === 'impact' && (
          <div className="tab-content">
            {impactAnalysis ? (
              <div className="impact-analysis">
                <div className="impact-header">
                  <h4>Impact Analysis</h4>
                  <span className="analyzed-node">
                    {nodes.find(n => n.id === impactAnalysis.nodeId)?.name || impactAnalysis.nodeId}
                  </span>
                </div>
                <div className="impact-summary">
                  <div className="impact-metric">
                    <span className="impact-label">Business Criticality:</span>
                    <span className={`impact-value ${impactAnalysis.businessCriticality}`}>
                      {impactAnalysis.businessCriticality.toUpperCase()}
                    </span>
                  </div>
                  <div className="impact-metric">
                    <span className="impact-label">Impact Radius:</span>
                    <span className="impact-value">
                      {impactAnalysis.impactRadius.toFixed(1)}
                    </span>
                  </div>
                  <div className="impact-metric">
                    <span className="impact-label">Estimated Downtime:</span>
                    <span className="impact-value">
                      {impactAnalysis.estimatedDowntime} minutes
                    </span>
                  </div>
                  <div className="impact-metric">
                    <span className="impact-label">Estimated Cost:</span>
                    <span className="impact-value cost">
                      {formatCurrency(impactAnalysis.estimatedCost)}
                    </span>
                  </div>
                </div>
                <div className="impact-dependencies">
                  <div className="dependency-section">
                    <h5>Direct Dependents ({impactAnalysis.directDependents.length})</h5>
                    <div className="dependency-list">
                      {impactAnalysis.directDependents.map(depId => (
                        <span key={depId} className="dependent">
                          {nodes.find(n => n.id === depId)?.name || depId}
                        </span>
                      ))}
                    </div>
                  </div>
                  <div className="dependency-section">
                    <h5>Transitive Dependents ({impactAnalysis.transitiveDependents.length})</h5>
                    <div className="dependency-list">
                      {impactAnalysis.transitiveDependents.slice(0, 10).map(depId => (
                        <span key={depId} className="dependent transitive">
                          {nodes.find(n => n.id === depId)?.name || depId}
                        </span>
                      ))}
                      {impactAnalysis.transitiveDependents.length > 10 && (
                        <span className="dependent more">
                          +{impactAnalysis.transitiveDependents.length - 10} more
                        </span>
                      )}
                    </div>
                  </div>
                </div>
                <div className="mitigation-strategies">
                  <h5>Mitigation Strategies</h5>
                  <ul className="strategies-list">
                    {impactAnalysis.mitigationStrategies.map((strategy, index) => (
                      <li key={index} className="strategy">
                        {strategy}
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            ) : (
              <div className="no-analysis">
                <p>Select a node to view impact analysis</p>
              </div>
            )}
          </div>
        )}

        {activeTab === 'settings' && (
          <div className="tab-content">
            <div className="analysis-settings">
              <h4>Analysis Options</h4>
              <div className="setting-group">
                <label className="setting-checkbox">
                  <input
                    type="checkbox"
                    checked={analysisOptions.includeTransitive}
                    onChange={(e) => onAnalysisOptionsChange({ includeTransitive: e.target.checked })}
                  />
                  Include Transitive Dependencies
                </label>
                <label className="setting-checkbox">
                  <input
                    type="checkbox"
                    checked={analysisOptions.includeExternal}
                    onChange={(e) => onAnalysisOptionsChange({ includeExternal: e.target.checked })}
                  />
                  Include External Dependencies
                </label>
                <label className="setting-checkbox">
                  <input
                    type="checkbox"
                    checked={analysisOptions.showSecurityVulnerabilities}
                    onChange={(e) => onAnalysisOptionsChange({ showSecurityVulnerabilities: e.target.checked })}
                  />
                  Show Security Vulnerabilities
                </label>
                <label className="setting-checkbox">
                  <input
                    type="checkbox"
                    checked={analysisOptions.showPerformanceMetrics}
                    onChange={(e) => onAnalysisOptionsChange({ showPerformanceMetrics: e.target.checked })}
                  />
                  Show Performance Metrics
                </label>
              </div>
              <div className="setting-group">
                <label className="setting-range">
                  <span>Max Depth: {analysisOptions.maxDepth}</span>
                  <input
                    type="range"
                    min="1"
                    max="10"
                    value={analysisOptions.maxDepth}
                    onChange={(e) => onAnalysisOptionsChange({ maxDepth: parseInt(e.target.value) })}
                  />
                </label>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}