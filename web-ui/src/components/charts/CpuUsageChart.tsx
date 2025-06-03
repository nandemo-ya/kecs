import React from 'react';
import { ResourceUsageChart } from './ResourceUsageChart';
import { ResourceChartData, ResourceChartConfig } from '../../types/resourceUsage';

interface CpuUsageChartProps {
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

export function CpuUsageChart({ 
  data, 
  title = 'CPU Usage',
  description = 'CPU utilization over time',
  height = 300,
  timeRange = '1h',
  showThresholds = true,
  warningThreshold = 70,
  criticalThreshold = 90,
  loading = false
}: CpuUsageChartProps) {
  
  const chartConfig: ResourceChartConfig = {
    type: 'area',
    title,
    description,
    metric: 'cpu',
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