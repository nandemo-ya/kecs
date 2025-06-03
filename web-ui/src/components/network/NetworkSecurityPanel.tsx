import React from 'react';
import {
  NetworkNode,
  NetworkDependency,
  SecurityAnalysis,
} from '../../types/networkDependencies';

interface NetworkSecurityPanelProps {
  nodes: NetworkNode[];
  dependencies: NetworkDependency[];
  securityAnalysis: SecurityAnalysis | null;
  onClose: () => void;
  onNodeSecurityAnalysis: (nodeId: string) => void;
}

export function NetworkSecurityPanel({
  nodes,
  dependencies,
  securityAnalysis,
  onClose,
  onNodeSecurityAnalysis,
}: NetworkSecurityPanelProps) {
  return (
    <div className="network-security-panel">
      <div className="security-header">
        <h3>Security Analysis</h3>
        <button className="close-button" onClick={onClose}>
          ✕
        </button>
      </div>
      <div className="security-content">
        <p>Security analysis panel - implementation coming soon</p>
        {securityAnalysis && (
          <div>
            <h4>Analysis for: {securityAnalysis.nodeId}</h4>
            <p>Risk Score: {securityAnalysis.riskScore}</p>
            <p>Vulnerabilities: {securityAnalysis.vulnerabilities.length}</p>
          </div>
        )}
      </div>
    </div>
  );
}