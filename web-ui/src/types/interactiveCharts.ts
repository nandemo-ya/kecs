// Interactive chart types and interfaces

// Common types for interactive charts
export interface InteractiveChartConfig {
  width?: number;
  height?: number;
  margin?: {
    top: number;
    right: number;
    bottom: number;
    left: number;
  };
  animationDuration?: number;
  responsive?: boolean;
  showTooltip?: boolean;
  showLegend?: boolean;
  enableZoom?: boolean;
  enablePan?: boolean;
  enableExport?: boolean;
}

// Drill-down functionality
export interface DrillDownConfig {
  enabled: boolean;
  levels: string[];
  onDrillDown?: (level: string, data: any) => void;
  onDrillUp?: () => void;
}

// Interactive Pie Chart types
export interface InteractivePieData {
  id: string;
  label: string;
  value: number;
  color?: string;
  children?: InteractivePieData[];
  metadata?: Record<string, any>;
}

export interface InteractivePieChartProps {
  data: InteractivePieData[];
  config?: InteractiveChartConfig;
  drillDown?: DrillDownConfig;
  onSegmentClick?: (segment: InteractivePieData) => void;
  onSegmentHover?: (segment: InteractivePieData | null) => void;
  selectedSegmentId?: string;
  innerRadius?: number;
  outerRadius?: number;
  padAngle?: number;
  cornerRadius?: number;
}

// Interactive Bar Chart types
export interface InteractiveBarData {
  id: string;
  category: string;
  value: number;
  subCategories?: InteractiveBarData[];
  color?: string;
  metadata?: Record<string, any>;
}

export interface InteractiveBarChartProps {
  data: InteractiveBarData[];
  config?: InteractiveChartConfig;
  orientation?: 'horizontal' | 'vertical';
  sortBy?: 'value' | 'category' | 'none';
  sortOrder?: 'asc' | 'desc';
  onBarClick?: (bar: InteractiveBarData) => void;
  onBarHover?: (bar: InteractiveBarData | null) => void;
  selectedBarId?: string;
  groupBy?: string;
  stackBy?: string;
  showValues?: boolean;
  enableSorting?: boolean;
  enableFiltering?: boolean;
  filterOptions?: FilterOption[];
}

export interface FilterOption {
  field: string;
  label: string;
  type: 'select' | 'range' | 'search';
  values?: string[] | number[];
  min?: number;
  max?: number;
}

// Sankey Diagram types
export interface SankeyNode {
  id: string;
  name: string;
  value?: number;
  color?: string;
  metadata?: Record<string, any>;
}

export interface SankeyLink {
  source: string;
  target: string;
  value: number;
  color?: string;
  metadata?: Record<string, any>;
}

export interface InteractiveSankeyProps {
  nodes: SankeyNode[];
  links: SankeyLink[];
  config?: InteractiveChartConfig;
  nodeWidth?: number;
  nodePadding?: number;
  nodeAlign?: 'left' | 'right' | 'center' | 'justify';
  onNodeClick?: (node: SankeyNode) => void;
  onLinkClick?: (link: SankeyLink) => void;
  onNodeHover?: (node: SankeyNode | null) => void;
  onLinkHover?: (link: SankeyLink | null) => void;
  highlightConnected?: boolean;
  enableNodeDragging?: boolean;
}

// Treemap types
export interface TreemapNode {
  id: string;
  name: string;
  value?: number;
  children?: TreemapNode[];
  color?: string;
  metadata?: Record<string, any>;
}

export interface InteractiveTreemapProps {
  data: TreemapNode;
  config?: InteractiveChartConfig;
  tileType?: 'squarify' | 'binary' | 'dice' | 'slice' | 'sliceDice' | 'resquarify';
  onNodeClick?: (node: TreemapNode) => void;
  onNodeHover?: (node: TreemapNode | null) => void;
  selectedNodeId?: string;
  colorScale?: any;
  enableZoom?: boolean;
  maxDepth?: number;
  valueFormat?: (value: number) => string;
}

// Radar Chart types
export interface RadarAxis {
  id: string;
  label: string;
  maxValue?: number;
  minValue?: number;
  unit?: string;
  description?: string;
  metadata?: Record<string, any>;
}

export interface RadarDataPoint {
  axis: string;
  value: number;
  label?: string;
  metadata?: Record<string, any>;
}

export interface RadarSeries {
  id: string;
  name: string;
  data: RadarDataPoint[];
  color?: string;
  opacity?: number;
  strokeWidth?: number;
  dotSize?: number;
  visible?: boolean;
  metadata?: Record<string, any>;
}

export interface InteractiveRadarProps {
  axes: RadarAxis[];
  series: RadarSeries[];
  config?: InteractiveChartConfig;
  maxValue?: number;
  levels?: number;
  onSeriesClick?: (series: RadarSeries) => void;
  onSeriesHover?: (series: RadarSeries | null) => void;
  onAxisClick?: (axis: RadarAxis) => void;
  selectedSeriesId?: string;
  showGrid?: boolean;
  showAxis?: boolean;
  showLabels?: boolean;
  showDots?: boolean;
  animateOnLoad?: boolean;
}

// Chart interaction events
export interface ChartInteractionEvent {
  type: 'click' | 'hover' | 'drag' | 'zoom' | 'pan';
  target: 'node' | 'link' | 'axis' | 'legend' | 'background';
  data: any;
  coordinates: {
    x: number;
    y: number;
    clientX: number;
    clientY: number;
  };
  modifiers: {
    ctrlKey: boolean;
    shiftKey: boolean;
    altKey: boolean;
  };
}

// Chart export options
export interface ExportOptions {
  format: 'png' | 'svg' | 'json' | 'csv';
  filename?: string;
  scale?: number;
  backgroundColor?: string;
  includeData?: boolean;
}

// Tooltip configuration
export interface TooltipConfig {
  enabled: boolean;
  position?: 'auto' | 'top' | 'right' | 'bottom' | 'left';
  offset?: { x: number; y: number };
  formatter?: (data: any) => string | React.ReactNode;
  style?: React.CSSProperties;
  showDelay?: number;
  hideDelay?: number;
}

// Animation configuration
export interface AnimationConfig {
  enabled: boolean;
  duration?: number;
  easing?: 'linear' | 'easeIn' | 'easeOut' | 'easeInOut' | 'bounce' | 'elastic';
  delay?: number;
  stagger?: number;
}

// Legend configuration
export interface LegendConfig {
  enabled: boolean;
  position?: 'top' | 'right' | 'bottom' | 'left';
  orientation?: 'horizontal' | 'vertical';
  itemWidth?: number;
  itemHeight?: number;
  itemSpacing?: number;
  clickable?: boolean;
  formatter?: (label: string, data: any) => string;
}

// Chart theme
export interface ChartTheme {
  backgroundColor?: string;
  textColor?: string;
  gridColor?: string;
  axisColor?: string;
  colors?: string[];
  fontFamily?: string;
  fontSize?: {
    title?: number;
    label?: number;
    tick?: number;
    legend?: number;
  };
}

// Data transformation utilities
export interface DataTransform {
  aggregate?: 'sum' | 'average' | 'min' | 'max' | 'count';
  groupBy?: string | string[];
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  filter?: (item: any) => boolean;
  limit?: number;
}

// Chart state management
export interface ChartState {
  selectedItems: string[];
  hoveredItem: string | null;
  zoomLevel: number;
  panOffset: { x: number; y: number };
  filters: Record<string, any>;
  sortConfig: {
    field: string;
    order: 'asc' | 'desc';
  };
  drillDownPath: string[];
}

// Chart update methods
export interface ChartMethods {
  updateData: (data: any) => void;
  refresh: () => void;
  reset: () => void;
  zoomIn: () => void;
  zoomOut: () => void;
  panTo: (x: number, y: number) => void;
  selectItem: (id: string) => void;
  deselectItem: (id: string) => void;
  clearSelection: () => void;
  exportChart: (options: ExportOptions) => Promise<void>;
}

// Responsive breakpoints
export interface ResponsiveConfig {
  breakpoints: {
    small?: number;
    medium?: number;
    large?: number;
  };
  rules: {
    small?: Partial<InteractiveChartConfig>;
    medium?: Partial<InteractiveChartConfig>;
    large?: Partial<InteractiveChartConfig>;
  };
}

// Default configurations
export const DEFAULT_CHART_CONFIG: InteractiveChartConfig = {
  width: 600,
  height: 400,
  margin: { top: 20, right: 20, bottom: 40, left: 40 },
  animationDuration: 300,
  responsive: true,
  showTooltip: true,
  showLegend: true,
  enableZoom: false,
  enablePan: false,
  enableExport: true,
};

export const DEFAULT_CHART_THEME: ChartTheme = {
  backgroundColor: '#ffffff',
  textColor: '#374151',
  gridColor: '#e5e7eb',
  axisColor: '#9ca3af',
  colors: [
    '#3b82f6', // blue
    '#10b981', // green
    '#f59e0b', // amber
    '#ef4444', // red
    '#8b5cf6', // violet
    '#ec4899', // pink
    '#06b6d4', // cyan
    '#f97316', // orange
  ],
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
  fontSize: {
    title: 18,
    label: 14,
    tick: 12,
    legend: 12,
  },
};

// Utility functions
export function interpolateColors(startColor: string, endColor: string, steps: number): string[] {
  // Simple color interpolation - in real implementation, use d3-interpolate
  const colors: string[] = [];
  for (let i = 0; i < steps; i++) {
    const ratio = i / (steps - 1);
    // Simplified interpolation - replace with proper color interpolation
    colors.push(ratio < 0.5 ? startColor : endColor);
  }
  return colors;
}

export function formatValue(value: number, format?: string): string {
  if (format === 'percentage') {
    return `${(value * 100).toFixed(1)}%`;
  }
  if (format === 'currency') {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(value);
  }
  if (value >= 1000000) {
    return `${(value / 1000000).toFixed(1)}M`;
  }
  if (value >= 1000) {
    return `${(value / 1000).toFixed(1)}K`;
  }
  return value.toFixed(1);
}

export function generateChartId(prefix: string = 'chart'): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}