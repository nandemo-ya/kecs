import React from 'react';
import { ResourceUsageChart } from './ResourceUsageChart';
import { ResourceChartData, ResourceChartConfig } from '../../types/resourceUsage';

interface CombinedUsageChartProps {
  data: ResourceChartData[];
  title?: string;
  description?: string;
  height?: number;
  timeRange?: '1h' | '6h' | '24h' | '7d' | '30d';
  showThresholds?: boolean;
  cpuWarningThreshold?: number;
  cpuCriticalThreshold?: number;
  memoryWarningThreshold?: number;
  memoryCriticalThreshold?: number;
  loading?: boolean;
}

export function CombinedUsageChart({ 
  data, 
  title = 'CPU & Memory Usage',
  description = 'Combined CPU and memory utilization over time',
  height = 400,
  timeRange = '1h',
  showThresholds = false, // Disable thresholds for combined chart as it gets confusing
  cpuWarningThreshold = 70,
  cpuCriticalThreshold = 90,
  memoryWarningThreshold = 80,
  memoryCriticalThreshold = 95,
  loading = false
}: CombinedUsageChartProps) {
  
  const chartConfig: ResourceChartConfig = {
    type: 'area',
    title,
    description,
    metric: 'both',
    timeRange,
    showThresholds,
    warningThreshold: cpuWarningThreshold, // Use CPU thresholds as primary
    criticalThreshold: cpuCriticalThreshold,
    height,
  };

  return (
    <ResourceUsageChart 
      data={data}
      config={chartConfig}
      loading={loading}
    />
  );
}