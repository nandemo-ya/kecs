// Resource usage and monitoring types

export interface ResourceUsagePoint {
  timestamp: number;
  cpuUsage: number; // Percentage (0-100)
  memoryUsage: number; // Percentage (0-100)
  cpuLimit?: number; // CPU units (1024 = 1 vCPU)
  memoryLimit?: number; // Memory in MB
  cpuRequest?: number; // CPU units requested
  memoryRequest?: number; // Memory in MB requested
}

export interface ClusterResourceUsage {
  clusterName: string;
  totalCpuCapacity: number; // Total CPU units available
  totalMemoryCapacity: number; // Total memory in MB available
  usedCpu: number; // Used CPU units
  usedMemory: number; // Used memory in MB
  cpuUtilization: number; // Percentage (0-100)
  memoryUtilization: number; // Percentage (0-100)
  nodeCount: number;
  podCount: number;
  timestamp: number;
  history: ResourceUsagePoint[];
}

export interface ServiceResourceUsage {
  serviceName: string;
  clusterName: string;
  taskCount: number;
  averageCpuUsage: number; // Percentage across all tasks
  averageMemoryUsage: number; // Percentage across all tasks
  peakCpuUsage: number;
  peakMemoryUsage: number;
  cpuLimit: number; // Per task
  memoryLimit: number; // Per task
  cpuRequest: number; // Per task
  memoryRequest: number; // Per task
  timestamp: number;
  history: ResourceUsagePoint[];
}

export interface TaskResourceUsage {
  taskArn: string;
  taskId: string;
  serviceName?: string;
  clusterName: string;
  cpuUsage: number; // Percentage
  memoryUsage: number; // Percentage
  cpuLimit: number; // CPU units
  memoryLimit: number; // Memory in MB
  cpuRequest: number; // CPU units
  memoryRequest: number; // Memory in MB
  networkRx?: number; // Bytes received
  networkTx?: number; // Bytes transmitted
  diskRead?: number; // Bytes read
  diskWrite?: number; // Bytes written
  timestamp: number;
  history: ResourceUsagePoint[];
}

export interface NodeResourceUsage {
  nodeName: string;
  clusterName: string;
  cpuCapacity: number; // Total CPU units
  memoryCapacity: number; // Total memory in MB
  cpuAllocatable: number; // Allocatable CPU units
  memoryAllocatable: number; // Allocatable memory in MB
  cpuUsage: number; // Current usage percentage
  memoryUsage: number; // Current usage percentage
  podCount: number;
  podCapacity: number;
  conditions: NodeCondition[];
  timestamp: number;
  history: ResourceUsagePoint[];
}

export interface NodeCondition {
  type: string; // Ready, OutOfDisk, MemoryPressure, etc.
  status: string; // True, False, Unknown
  reason?: string;
  message?: string;
  lastTransitionTime: string;
}

// Resource alert definitions
export interface ResourceAlert {
  id: string;
  type: 'cpu' | 'memory' | 'disk' | 'network';
  level: 'warning' | 'critical';
  threshold: number; // Percentage
  resourceName: string; // cluster, service, or task name
  resourceType: 'cluster' | 'service' | 'task' | 'node';
  currentValue: number;
  message: string;
  timestamp: number;
  acknowledged: boolean;
}

// Chart configuration for resource usage
export interface ResourceChartConfig {
  type: 'line' | 'area' | 'gauge' | 'heatmap';
  title: string;
  description?: string;
  metric: 'cpu' | 'memory' | 'both';
  timeRange: '1h' | '6h' | '24h' | '7d' | '30d';
  showThresholds?: boolean;
  warningThreshold?: number; // Percentage
  criticalThreshold?: number; // Percentage
  aggregationType?: 'avg' | 'max' | 'min' | 'p95' | 'p99';
  height?: number;
  refreshInterval?: number; // milliseconds
}

// Resource usage aggregations
export interface ResourceUsageSummary {
  resourceName: string;
  resourceType: 'cluster' | 'service' | 'task' | 'node';
  cpuStats: UsageStats;
  memoryStats: UsageStats;
  period: {
    start: number;
    end: number;
    duration: number; // milliseconds
  };
}

export interface UsageStats {
  current: number; // Current percentage
  average: number; // Average over period
  peak: number; // Peak usage
  minimum: number; // Minimum usage
  p95: number; // 95th percentile
  p99: number; // 99th percentile
  trend: 'increasing' | 'decreasing' | 'stable';
  efficiency: number; // Usage vs requests ratio
}

// Resource recommendations
export interface ResourceRecommendation {
  resourceName: string;
  resourceType: 'service' | 'task';
  type: 'rightsizing' | 'scaling' | 'optimization';
  severity: 'low' | 'medium' | 'high';
  title: string;
  description: string;
  currentConfig: {
    cpuRequest?: number;
    memoryRequest?: number;
    cpuLimit?: number;
    memoryLimit?: number;
    replicas?: number;
  };
  recommendedConfig: {
    cpuRequest?: number;
    memoryRequest?: number;
    cpuLimit?: number;
    memoryLimit?: number;
    replicas?: number;
  };
  estimatedSavings?: {
    cpu: number; // Percentage
    memory: number; // Percentage
    cost?: number; // If cost data available
  };
  confidence: number; // 0-100 percentage
  timestamp: number;
}

// Exported utility types
export type ResourceMetric = 'cpu' | 'memory' | 'network' | 'disk';
export type ResourceTimeRange = '1h' | '6h' | '24h' | '7d' | '30d';
export type AggregationType = 'avg' | 'max' | 'min' | 'p95' | 'p99';

// For chart data transformation
export interface ResourceChartData {
  timestamp: number;
  cpu?: number;
  memory?: number;
  cpuLimit?: number;
  memoryLimit?: number;
  cpuRequest?: number;
  memoryRequest?: number;
  [key: string]: number | undefined;
}