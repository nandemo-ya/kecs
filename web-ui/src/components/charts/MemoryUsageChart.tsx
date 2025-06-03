import React from 'react';
import { ResourceUsageChart } from './ResourceUsageChart';
import { ResourceChartData, ResourceChartConfig } from '../../types/resourceUsage';

interface MemoryUsageChartProps {
  data: ResourceChartData[];
  title?: string;
  description?: string;
  height?: number;
  timeRange?: '1h' | '6h' | '24h' | '7d' | '30d';
  showThresholds?: boolean;
  warningThreshold?: number;
  criticalThreshold?: number;
  loading?: boolean;
}

export function MemoryUsageChart({ 
  data, 
  title = 'Memory Usage',
  description = 'Memory utilization over time',
  height = 300,
  timeRange = '1h',
  showThresholds = true,
  warningThreshold = 80,
  criticalThreshold = 95,
  loading = false
}: MemoryUsageChartProps) {
  
  const chartConfig: ResourceChartConfig = {
    type: 'area',
    title,
    description,
    metric: 'memory',
    timeRange,
    showThresholds,
    warningThreshold,
    criticalThreshold,
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