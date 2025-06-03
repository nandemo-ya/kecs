import React from 'react';
import {
  BarChart as RechartsBarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { BarChartData, ChartConfig } from '../../types/metrics';
import './Charts.css';

interface BarChartProps {
  data: BarChartData[];
  config: ChartConfig;
  loading?: boolean;
  horizontal?: boolean;
}

const DEFAULT_COLORS = [
  '#3b82f6', // blue
  '#10b981', // green
  '#f59e0b', // yellow
  '#ef4444', // red
  '#8b5cf6', // purple
  '#06b6d4', // cyan
  '#f97316', // orange
  '#84cc16', // lime
];

export function BarChart({ data, config, loading = false, horizontal = false }: BarChartProps) {
  const formatTooltip = (value: any, name: string) => {
    return [value.toLocaleString(), name];
  };

  if (loading) {
    return (
      <div className="chart-container">
        <div className="chart-header">
          <h3>{config.title}</h3>
          {config.description && <p>{config.description}</p>}
        </div>
        <div className="chart-loading">
          <div className="loading-spinner">⟳</div>
          <p>Loading chart data...</p>
        </div>
      </div>
    );
  }

  if (!data || data.length === 0) {
    return (
      <div className="chart-container">
        <div className="chart-header">
          <h3>{config.title}</h3>
          {config.description && <p>{config.description}</p>}
        </div>
        <div className="chart-empty">
          <p>No data available</p>
        </div>
      </div>
    );
  }

  // Add colors to data if not provided
  const dataWithColors = data.map((item, index) => ({
    ...item,
    fill: item.fill || config.colors?.[index] || DEFAULT_COLORS[index % DEFAULT_COLORS.length],
  }));

  return (
    <div className="chart-container">
      <div className="chart-header">
        <h3>{config.title}</h3>
        {config.description && <p>{config.description}</p>}
      </div>
      <div className="chart-content">
        <ResponsiveContainer width="100%" height={config.height || 300}>
          <RechartsBarChart
            layout={horizontal ? 'horizontal' : 'vertical'}
            data={dataWithColors}
            margin={{
              top: 5,
              right: 30,
              left: horizontal ? 60 : 20,
              bottom: 5,
            }}
          >
            {config.showGrid !== false && (
              <CartesianGrid 
                strokeDasharray="3 3" 
                stroke="#e5e7eb"
                horizontal={!horizontal}
                vertical={horizontal}
              />
            )}
            
            {horizontal ? (
              <>
                <XAxis 
                  type="number"
                  stroke="#6b7280"
                  fontSize={12}
                  label={config.xAxisLabel ? { 
                    value: config.xAxisLabel, 
                    position: 'insideBottom',
                    offset: -5 
                  } : undefined}
                />
                <YAxis 
                  type="category"
                  dataKey="name"
                  stroke="#6b7280"
                  fontSize={12}
                  width={60}
                />
              </>
            ) : (
              <>
                <XAxis 
                  dataKey="name"
                  stroke="#6b7280"
                  fontSize={12}
                  label={config.xAxisLabel ? { 
                    value: config.xAxisLabel, 
                    position: 'insideBottom',
                    offset: -5 
                  } : undefined}
                />
                <YAxis 
                  stroke="#6b7280"
                  fontSize={12}
                  label={config.yAxisLabel ? { 
                    value: config.yAxisLabel, 
                    angle: -90, 
                    position: 'insideLeft' 
                  } : undefined}
                />
              </>
            )}
            
            <Tooltip 
              formatter={formatTooltip}
              contentStyle={{
                backgroundColor: 'white',
                border: '1px solid #d1d5db',
                borderRadius: '0.375rem',
                boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
              }}
            />
            
            {config.showLegend !== false && <Legend />}
            
            <Bar 
              dataKey="value" 
              radius={horizontal ? [0, 4, 4, 0] : [4, 4, 0, 0]}
            />
          </RechartsBarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}