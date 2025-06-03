// Service topology and relationship types
import { Node, Edge, Position, MarkerType } from 'reactflow';

// Service node types
export interface ServiceNode {
  id: string;
  serviceName: string;
  clusterName: string;
  taskCount: number;
  desiredCount: number;
  runningCount: number;
  pendingCount: number;
  serviceType: 'web' | 'api' | 'database' | 'cache' | 'queue' | 'storage' | 'custom';
  healthStatus: 'healthy' | 'unhealthy' | 'degraded' | 'unknown';
  deploymentStatus: 'active' | 'updating' | 'inactive';
  launchType?: 'EC2' | 'FARGATE' | 'EXTERNAL';
  taskDefinition?: string;
  createdAt: string;
  metadata?: Record<string, any>;
}

// Connection types between services
export interface ServiceConnection {
  id: string;
  source: string; // Service ID
  target: string; // Service ID
  connectionType: 'http' | 'grpc' | 'tcp' | 'database' | 'cache' | 'queue' | 'custom';
  protocol?: string;
  port?: number;
  isSecure?: boolean;
  trafficFlow?: 'unidirectional' | 'bidirectional';
  requestsPerMinute?: number;
  latencyMs?: number;
  errorRate?: number;
  metadata?: Record<string, any>;
}

// React Flow node data
export interface ServiceNodeData extends ServiceNode {
  isSelected?: boolean;
  isHighlighted?: boolean;
  showDetails?: boolean;
}

// React Flow edge data
export interface ServiceEdgeData extends ServiceConnection {
  animated?: boolean;
  label?: string;
  isHighlighted?: boolean;
}

// Extended React Flow types
export type ServiceFlowNode = Node<ServiceNodeData>;
export type ServiceFlowEdge = Edge<ServiceEdgeData>;

// Layout algorithms
export type LayoutAlgorithm = 'hierarchical' | 'force' | 'circular' | 'grid' | 'manual';

// Topology view options
export interface TopologyViewOptions {
  layout: LayoutAlgorithm;
  showHealthStatus: boolean;
  showTaskCounts: boolean;
  showConnections: boolean;
  showTrafficFlow: boolean;
  showLatency: boolean;
  autoRefresh: boolean;
  refreshInterval: number;
  filterByCluster?: string;
  filterByServiceType?: string[];
  filterByHealth?: string[];
}

// Service group for clustering
export interface ServiceGroup {
  id: string;
  name: string;
  services: string[]; // Service IDs
  color?: string;
  collapsed?: boolean;
}

// Topology metrics
export interface TopologyMetrics {
  totalServices: number;
  healthyServices: number;
  unhealthyServices: number;
  totalConnections: number;
  avgLatency: number;
  totalRequestsPerMinute: number;
  clusters: string[];
  serviceTypes: Record<string, number>;
}

// Node style configurations
export interface NodeStyleConfig {
  width: number;
  height: number;
  borderRadius: number;
  fontSize: number;
  iconSize: number;
  colors: {
    healthy: string;
    unhealthy: string;
    degraded: string;
    unknown: string;
    selected: string;
    highlighted: string;
  };
}

// Edge style configurations
export interface EdgeStyleConfig {
  strokeWidth: number;
  arrowSize: number;
  colors: {
    http: string;
    grpc: string;
    tcp: string;
    database: string;
    cache: string;
    queue: string;
    custom: string;
  };
  animationSpeed: number;
}

// Service details panel
export interface ServiceDetails {
  service: ServiceNode;
  connections: {
    incoming: ServiceConnection[];
    outgoing: ServiceConnection[];
  };
  metrics: {
    cpu: number;
    memory: number;
    requestsPerMinute: number;
    errorRate: number;
    avgResponseTime: number;
  };
  recentEvents: ServiceEvent[];
  taskDetails: TaskInfo[];
}

// Service events
export interface ServiceEvent {
  id: string;
  timestamp: string;
  type: 'deployment' | 'scale' | 'health' | 'error' | 'config';
  severity: 'info' | 'warning' | 'error' | 'critical';
  message: string;
  metadata?: Record<string, any>;
}

// Task information
export interface TaskInfo {
  taskArn: string;
  taskId: string;
  status: string;
  lastStatus: string;
  cpu: string;
  memory: string;
  startedAt?: string;
  containerInfo: {
    name: string;
    image: string;
    status: string;
    ports?: number[];
  }[];
}

// Layout position calculation
export interface LayoutPosition {
  x: number;
  y: number;
  level?: number;
  cluster?: string;
}

// Service type icons
export const SERVICE_TYPE_ICONS: Record<string, string> = {
  web: '🌐',
  api: '⚡',
  database: '🗄️',
  cache: '💾',
  queue: '📬',
  storage: '📦',
  custom: '⚙️',
};

// Connection type styles
export const CONNECTION_STYLES = {
  http: { strokeDasharray: '0', markerEnd: MarkerType.ArrowClosed },
  grpc: { strokeDasharray: '5 5', markerEnd: MarkerType.ArrowClosed },
  tcp: { strokeDasharray: '0', markerEnd: MarkerType.Arrow },
  database: { strokeDasharray: '0', markerEnd: MarkerType.ArrowClosed },
  cache: { strokeDasharray: '3 3', markerEnd: MarkerType.Arrow },
  queue: { strokeDasharray: '10 5', markerEnd: MarkerType.ArrowClosed },
  custom: { strokeDasharray: '0', markerEnd: MarkerType.Arrow },
} as const;

// Default configurations
export const DEFAULT_NODE_STYLE: NodeStyleConfig = {
  width: 180,
  height: 80,
  borderRadius: 8,
  fontSize: 14,
  iconSize: 24,
  colors: {
    healthy: '#10b981',
    unhealthy: '#ef4444',
    degraded: '#f59e0b',
    unknown: '#6b7280',
    selected: '#3b82f6',
    highlighted: '#8b5cf6',
  },
};

export const DEFAULT_EDGE_STYLE: EdgeStyleConfig = {
  strokeWidth: 2,
  arrowSize: 20,
  colors: {
    http: '#3b82f6',
    grpc: '#8b5cf6',
    tcp: '#6b7280',
    database: '#059669',
    cache: '#f59e0b',
    queue: '#ec4899',
    custom: '#6366f1',
  },
  animationSpeed: 1,
};

// Utility function to get service health color
export function getHealthColor(status: string): string {
  switch (status) {
    case 'healthy':
      return DEFAULT_NODE_STYLE.colors.healthy;
    case 'unhealthy':
      return DEFAULT_NODE_STYLE.colors.unhealthy;
    case 'degraded':
      return DEFAULT_NODE_STYLE.colors.degraded;
    default:
      return DEFAULT_NODE_STYLE.colors.unknown;
  }
}

// Utility function to get connection color
export function getConnectionColor(type: string): string {
  return DEFAULT_EDGE_STYLE.colors[type as keyof typeof DEFAULT_EDGE_STYLE.colors] || DEFAULT_EDGE_STYLE.colors.custom;
}