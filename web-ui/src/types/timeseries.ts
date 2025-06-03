// Time series data visualization types

export interface TimeSeriesDataPoint {
  timestamp: number;
  value: number;
  label?: string;
  metadata?: Record<string, any>;
}

export interface TimeSeriesData {
  id: string;
  name: string;
  color: string;
  data: TimeSeriesDataPoint[];
  unit?: string;
  type?: 'line' | 'area' | 'bar' | 'scatter';
  visible?: boolean;
  yAxisId?: string;
}

export interface TimeSeriesConfig {
  title: string;
  description?: string;
  height?: number;
  showLegend?: boolean;
  showGrid?: boolean;
  enableZoom?: boolean;
  enableBrush?: boolean;
  interpolation?: 'linear' | 'monotone' | 'step' | 'stepBefore' | 'stepAfter';
  aggregation?: TimeSeriesAggregation;
  timeRange?: TimeRange;
  refreshInterval?: number;
}

export interface TimeSeriesAggregation {
  type: 'none' | 'average' | 'sum' | 'min' | 'max' | 'count';
  interval: 'minute' | 'hour' | 'day' | 'week';
  smoothing?: 'none' | 'moving-average' | 'exponential';
  windowSize?: number;
}

export interface TimeRange {
  start: number;
  end: number;
  label: string;
  preset?: 'last-hour' | 'last-day' | 'last-week' | 'last-month' | 'custom';
}

// Heatmap specific types
export interface HeatmapDataPoint {
  x: number; // timestamp or category
  y: number; // category or metric
  value: number;
  label?: string;
}

export interface HeatmapConfig {
  title: string;
  description?: string;
  height?: number;
  width?: number;
  xAxisLabel?: string;
  yAxisLabel?: string;
  colorScale?: 'blues' | 'reds' | 'greens' | 'viridis' | 'plasma';
  showValues?: boolean;
  cellSize?: number;
}

// Multi-axis chart types
export interface YAxisConfig {
  id: string;
  orientation: 'left' | 'right';
  domain?: [number, number] | ['auto', 'auto'] | ['dataMin', 'dataMax'];
  label?: string;
  unit?: string;
  color?: string;
  tickFormatter?: (value: number) => string;
}

export interface TimeSeriesChartConfig extends TimeSeriesConfig {
  yAxes?: YAxisConfig[];
  annotations?: TimeSeriesAnnotation[];
  thresholds?: TimeSeriesThreshold[];
}

export interface TimeSeriesAnnotation {
  id: string;
  type: 'vertical-line' | 'horizontal-line' | 'range' | 'point';
  timestamp?: number;
  timestampEnd?: number;
  value?: number;
  valueEnd?: number;
  label: string;
  color?: string;
  strokeStyle?: 'solid' | 'dashed' | 'dotted';
}

export interface TimeSeriesThreshold {
  id: string;
  value: number;
  label: string;
  color: string;
  strokeStyle?: 'solid' | 'dashed' | 'dotted';
  yAxisId?: string;
}

// Data transformation utilities
export interface TimeSeriesTransform {
  type: 'derivative' | 'rate' | 'cumulative' | 'delta' | 'percent-change';
  window?: number;
  unit?: string;
}

// Real-time streaming types
export interface TimeSeriesStream {
  id: string;
  endpoint: string;
  interval: number;
  bufferSize: number;
  autoScale?: boolean;
}

// Dashboard layout types
export interface TimeSeriesWidget {
  id: string;
  type: 'timeseries' | 'heatmap' | 'gauge' | 'counter';
  title: string;
  gridPosition: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
  config: TimeSeriesChartConfig | HeatmapConfig;
  dataSource: string | TimeSeriesData[];
}

export interface TimeSeriesDashboardConfig {
  title: string;
  description?: string;
  layout: 'grid' | 'masonry' | 'flex';
  widgets: TimeSeriesWidget[];
  globalTimeRange?: TimeRange;
  autoRefresh?: boolean;
  refreshInterval?: number;
}

// Event and anomaly detection
export interface TimeSeriesEvent {
  id: string;
  timestamp: number;
  type: 'spike' | 'drop' | 'anomaly' | 'threshold-breach' | 'custom';
  severity: 'low' | 'medium' | 'high' | 'critical';
  message: string;
  value?: number;
  seriesId?: string;
  metadata?: Record<string, any>;
}

// Export utility types
export type TimeSeriesValueType = 'percentage' | 'bytes' | 'count' | 'duration' | 'rate' | 'custom';
export type ChartInteraction = 'zoom' | 'pan' | 'brush' | 'tooltip' | 'crosshair';
export type DataFrequency = 'realtime' | 'high' | 'medium' | 'low' | 'batch';

// Commonly used chart presets
export const TIME_SERIES_PRESETS = {
  SYSTEM_METRICS: {
    colors: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444'],
    interpolation: 'monotone' as const,
    showGrid: true,
    enableZoom: true,
  },
  BUSINESS_METRICS: {
    colors: ['#6366f1', '#8b5cf6', '#ec4899', '#f97316'],
    interpolation: 'linear' as const,
    showGrid: false,
    enableBrush: true,
  },
  PERFORMANCE_METRICS: {
    colors: ['#06b6d4', '#84cc16', '#fbbf24', '#f87171'],
    interpolation: 'step' as const,
    showGrid: true,
    enableZoom: true,
  },
} as const;