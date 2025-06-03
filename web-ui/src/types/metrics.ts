// Metrics and visualization types

export interface MetricDataPoint {
  timestamp: number; // Unix timestamp
  value: number;
  label?: string;
}

export interface TimeSeriesData {
  name: string;
  data: MetricDataPoint[];
  color?: string;
}

export interface ResourceMetrics {
  clusterName: string;
  servicesCount: number;
  tasksCount: number;
  runningTasks: number;
  pendingTasks: number;
  stoppedTasks: number;
  timestamp: number;
}

export interface ServiceMetrics {
  serviceName: string;
  clusterName: string;
  desiredCount: number;
  runningCount: number;
  pendingCount: number;
  cpuUtilization?: number;
  memoryUtilization?: number;
  timestamp: number;
}

export interface TaskDefinitionMetrics {
  family: string;
  activeRevisions: number;
  totalTasks: number;
  timestamp: number;
}

export interface DashboardMetrics {
  totalClusters: number;
  totalServices: number;
  totalTasks: number;
  totalTaskDefinitions: number;
  healthyServices: number;
  unhealthyServices: number;
  runningTasks: number;
  pendingTasks: number;
  stoppedTasks: number;
  timestamp: number;
}

export interface ChartConfig {
  type: 'line' | 'area' | 'bar' | 'pie' | 'donut';
  title: string;
  description?: string;
  xAxisLabel?: string;
  yAxisLabel?: string;
  showGrid?: boolean;
  showLegend?: boolean;
  height?: number;
  colors?: string[];
}

export interface MetricsHistoryEntry {
  timestamp: number;
  clusters: ResourceMetrics[];
  services: ServiceMetrics[];
  taskDefinitions: TaskDefinitionMetrics[];
  dashboard: DashboardMetrics;
}

// Chart data transformation helpers
export interface PieChartData {
  name: string;
  value: number;
  color?: string;
}

export interface BarChartData {
  name: string;
  value: number;
  fill?: string;
}

export interface LineChartData {
  timestamp: number;
  [key: string]: number;
}

// Time range options for metrics
export type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d';

export interface TimeRangeConfig {
  label: string;
  value: TimeRange;
  hours: number;
  intervalMinutes: number;
}