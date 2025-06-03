// Network dependency visualization types
import { Node, Edge } from 'reactflow';

// Network dependency node types
export interface NetworkNode {
  id: string;
  name: string;
  type: 'service' | 'database' | 'external' | 'load_balancer' | 'gateway' | 'cache' | 'queue';
  cluster?: string;
  namespace?: string;
  ip?: string;
  port?: number;
  protocol?: string;
  status: 'active' | 'inactive' | 'degraded' | 'unknown';
  criticality: 'critical' | 'high' | 'medium' | 'low';
  security: {
    encrypted: boolean;
    authentication: boolean;
    authorization: boolean;
    firewall: boolean;
  };
  metadata?: Record<string, any>;
}

// Network dependency connection
export interface NetworkDependency {
  id: string;
  source: string;
  target: string;
  dependencyType: 'api_call' | 'database_query' | 'file_access' | 'event_stream' | 'cache_access' | 'load_balance' | 'proxy' | 'custom';
  protocol: 'HTTP' | 'HTTPS' | 'TCP' | 'UDP' | 'gRPC' | 'WebSocket' | 'AMQP' | 'MQTT' | 'Redis' | 'SQL' | 'Custom';
  port: number;
  direction: 'incoming' | 'outgoing' | 'bidirectional';
  strength: 'weak' | 'moderate' | 'strong' | 'critical';
  frequency: number; // requests per minute
  latency: number; // milliseconds
  errorRate: number; // percentage
  bandwidth: number; // bytes per second
  security: {
    encrypted: boolean;
    authenticated: boolean;
    authorized: boolean;
  };
  sla?: {
    availability: number; // percentage
    responseTime: number; // milliseconds
    throughput: number; // requests per second
  };
  metadata?: Record<string, any>;
}

// Network flow data for traffic visualization
export interface NetworkFlow {
  id: string;
  sourceId: string;
  targetId: string;
  volume: number; // bytes
  packets: number;
  duration: number; // milliseconds
  timestamp: string;
  protocol: string;
  quality: 'excellent' | 'good' | 'fair' | 'poor';
}

// Dependency path for tracing
export interface DependencyPath {
  id: string;
  path: string[]; // array of node IDs
  type: 'direct' | 'transitive' | 'circular';
  length: number;
  totalLatency: number;
  reliability: number; // percentage
  bottlenecks: string[]; // node IDs that are bottlenecks
}

// Network security analysis
export interface SecurityAnalysis {
  nodeId: string;
  vulnerabilities: SecurityVulnerability[];
  compliance: ComplianceStatus[];
  riskScore: number; // 0-100
  recommendations: string[];
}

export interface SecurityVulnerability {
  id: string;
  type: 'unencrypted_traffic' | 'weak_authentication' | 'open_port' | 'outdated_protocol' | 'missing_firewall' | 'privilege_escalation';
  severity: 'critical' | 'high' | 'medium' | 'low';
  description: string;
  impact: string;
  mitigation: string;
}

export interface ComplianceStatus {
  standard: 'PCI_DSS' | 'HIPAA' | 'SOX' | 'GDPR' | 'SOC2' | 'ISO27001';
  status: 'compliant' | 'non_compliant' | 'partial' | 'unknown';
  requirements: string[];
}

// React Flow specific types
export interface NetworkNodeData extends NetworkNode {
  isHighlighted?: boolean;
  isSelected?: boolean;
  showDetails?: boolean;
  flowMetrics?: {
    inbound: number;
    outbound: number;
    errors: number;
  };
}

export interface NetworkDependencyData extends NetworkDependency {
  isHighlighted?: boolean;
  isAnimated?: boolean;
  flowDirection?: 'forward' | 'reverse' | 'both';
}

export type NetworkFlowNode = Node<NetworkNodeData>;
export type NetworkFlowEdge = Edge<NetworkDependencyData>;

// Dependency analysis options
export interface DependencyAnalysisOptions {
  includeTransitive: boolean;
  maxDepth: number;
  includeExternal: boolean;
  filterByCriticality: string[];
  filterByProtocol: string[];
  showSecurityVulnerabilities: boolean;
  showPerformanceMetrics: boolean;
  groupByCluster: boolean;
  groupByNamespace: boolean;
}

// Network visualization options
export interface NetworkVisualizationOptions {
  layout: 'hierarchical' | 'force' | 'circular' | 'grid' | 'manual';
  showTrafficFlow: boolean;
  showLatency: boolean;
  showSecurity: boolean;
  showCriticality: boolean;
  animateTraffic: boolean;
  highlightPaths: boolean;
  groupingSetting: 'none' | 'cluster' | 'namespace' | 'security_zone' | 'protocol';
  autoRefresh: boolean;
  refreshInterval: number;
}

// Network metrics for dashboard
export interface NetworkMetrics {
  totalNodes: number;
  totalDependencies: number;
  criticalPaths: number;
  securityVulnerabilities: number;
  averageLatency: number;
  totalTrafficVolume: number;
  uptime: number;
  errorRate: number;
  topBottlenecks: string[];
  complianceScore: number;
}

// Dependency impact analysis
export interface ImpactAnalysis {
  nodeId: string;
  directDependents: string[];
  transitiveDependents: string[];
  impactRadius: number;
  businessCriticality: 'critical' | 'high' | 'medium' | 'low';
  estimatedDowntime: number; // minutes
  estimatedCost: number; // USD
  mitigationStrategies: string[];
}

// Network topology discovery
export interface TopologyDiscovery {
  discoveryMethod: 'passive' | 'active' | 'hybrid';
  lastUpdate: string;
  coverage: number; // percentage
  confidence: number; // percentage
  sources: string[];
  limitations: string[];
}

// Performance analytics
export interface PerformanceAnalytics {
  nodeId: string;
  metrics: {
    cpu: number;
    memory: number;
    network: number;
    storage: number;
  };
  trends: {
    direction: 'improving' | 'stable' | 'degrading';
    confidence: number;
  };
  predictions: {
    nextBottleneck: string;
    timeToBottleneck: number; // hours
    recommendedActions: string[];
  };
}

// Network node styling configurations
export interface NetworkNodeStyle {
  width: number;
  height: number;
  borderRadius: number;
  fontSize: number;
  iconSize: number;
  colors: {
    service: string;
    database: string;
    external: string;
    load_balancer: string;
    gateway: string;
    cache: string;
    queue: string;
    critical: string;
    high: string;
    medium: string;
    low: string;
    active: string;
    inactive: string;
    degraded: string;
    unknown: string;
  };
}

// Network edge styling configurations
export interface NetworkEdgeStyle {
  strokeWidth: number;
  arrowSize: number;
  colors: {
    api_call: string;
    database_query: string;
    file_access: string;
    event_stream: string;
    cache_access: string;
    load_balance: string;
    proxy: string;
    custom: string;
    encrypted: string;
    unencrypted: string;
    critical: string;
    high: string;
    moderate: string;
    weak: string;
  };
  animations: {
    traffic: string;
    security: string;
    error: string;
  };
}

// Default configurations
export const DEFAULT_NETWORK_NODE_STYLE: NetworkNodeStyle = {
  width: 200,
  height: 100,
  borderRadius: 8,
  fontSize: 14,
  iconSize: 28,
  colors: {
    service: '#3b82f6',
    database: '#059669',
    external: '#dc2626',
    load_balancer: '#7c3aed',
    gateway: '#ea580c',
    cache: '#0891b2',
    queue: '#be185d',
    critical: '#dc2626',
    high: '#ea580c',
    medium: '#ca8a04',
    low: '#65a30d',
    active: '#10b981',
    inactive: '#6b7280',
    degraded: '#f59e0b',
    unknown: '#9ca3af',
  },
};

export const DEFAULT_NETWORK_EDGE_STYLE: NetworkEdgeStyle = {
  strokeWidth: 2,
  arrowSize: 20,
  colors: {
    api_call: '#3b82f6',
    database_query: '#059669',
    file_access: '#7c3aed',
    event_stream: '#ea580c',
    cache_access: '#0891b2',
    load_balance: '#be185d',
    proxy: '#6366f1',
    custom: '#6b7280',
    encrypted: '#10b981',
    unencrypted: '#ef4444',
    critical: '#dc2626',
    high: '#ea580c',
    moderate: '#ca8a04',
    weak: '#9ca3af',
  },
  animations: {
    traffic: 'flow 2s linear infinite',
    security: 'secure-pulse 3s ease-in-out infinite',
    error: 'error-flash 1s ease-in-out infinite',
  },
};

// Node type icons
export const NETWORK_NODE_ICONS: Record<string, string> = {
  service: '⚙️',
  database: '🗄️',
  external: '🌐',
  load_balancer: '⚖️',
  gateway: '🚪',
  cache: '💾',
  queue: '📬',
};

// Protocol icons
export const PROTOCOL_ICONS: Record<string, string> = {
  HTTP: '🌐',
  HTTPS: '🔒',
  TCP: '🔌',
  UDP: '📡',
  gRPC: '⚡',
  WebSocket: '🔄',
  AMQP: '📬',
  MQTT: '📡',
  Redis: '💾',
  SQL: '🗄️',
  Custom: '🔗',
};

// Security status icons
export const SECURITY_ICONS: Record<string, string> = {
  secure: '🔒',
  warning: '⚠️',
  vulnerable: '🚨',
  unknown: '❓',
};

// Utility functions
export function getNodeTypeColor(type: string): string {
  return DEFAULT_NETWORK_NODE_STYLE.colors[type as keyof typeof DEFAULT_NETWORK_NODE_STYLE.colors] || DEFAULT_NETWORK_NODE_STYLE.colors.service;
}

export function getCriticalityColor(criticality: string): string {
  switch (criticality) {
    case 'critical':
      return DEFAULT_NETWORK_NODE_STYLE.colors.critical;
    case 'high':
      return DEFAULT_NETWORK_NODE_STYLE.colors.high;
    case 'medium':
      return DEFAULT_NETWORK_NODE_STYLE.colors.medium;
    case 'low':
      return DEFAULT_NETWORK_NODE_STYLE.colors.low;
    default:
      return DEFAULT_NETWORK_NODE_STYLE.colors.unknown;
  }
}

export function getDependencyTypeColor(type: string): string {
  return DEFAULT_NETWORK_EDGE_STYLE.colors[type as keyof typeof DEFAULT_NETWORK_EDGE_STYLE.colors] || DEFAULT_NETWORK_EDGE_STYLE.colors.custom;
}

export function getSecurityColor(security: boolean): string {
  return security ? DEFAULT_NETWORK_EDGE_STYLE.colors.encrypted : DEFAULT_NETWORK_EDGE_STYLE.colors.unencrypted;
}

export function calculateRiskScore(node: NetworkNode, dependencies: NetworkDependency[]): number {
  let score = 0;
  
  // Base score based on criticality
  switch (node.criticality) {
    case 'critical': score += 40; break;
    case 'high': score += 30; break;
    case 'medium': score += 20; break;
    case 'low': score += 10; break;
  }
  
  // Security factors
  if (!node.security.encrypted) score += 15;
  if (!node.security.authentication) score += 10;
  if (!node.security.authorization) score += 10;
  if (!node.security.firewall) score += 5;
  
  // Dependency factors
  const nodeDependencies = dependencies.filter(d => d.source === node.id || d.target === node.id);
  const unencryptedDeps = nodeDependencies.filter(d => !d.security.encrypted).length;
  score += unencryptedDeps * 5;
  
  // External dependencies increase risk
  const externalDeps = nodeDependencies.filter(d => 
    (d.source === node.id && d.target.startsWith('external-')) ||
    (d.target === node.id && d.source.startsWith('external-'))
  ).length;
  score += externalDeps * 10;
  
  return Math.min(100, score);
}