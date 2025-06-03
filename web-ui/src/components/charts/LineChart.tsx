import React from 'react';
import {
  LineChart as RechartsLineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { TimeSeriesData, ChartConfig } from '../../types/metrics';
import './Charts.css';

interface LineChartProps {
  data: TimeSeriesData[];
  config: ChartConfig;
  loading?: boolean;
}

export function LineChart({ data, config, loading = false }: LineChartProps) {
  // Transform data for Recharts
  const chartData = React.useMemo(() => {
    if (!data || data.length === 0) return [];

    // Get all unique timestamps
    const allTimestamps = new Set<number>();
    data.forEach(series => {
      series.data.forEach(point => allTimestamps.add(point.timestamp));
    });

    const timestamps = Array.from(allTimestamps).sort();

    // Create chart data points
    return timestamps.map(timestamp => {
      const point: any = { timestamp };
      
      data.forEach(series => {
        const dataPoint = series.data.find(p => p.timestamp === timestamp);
        point[series.name] = dataPoint ? dataPoint.value : null;
      });

      return point;
    });
  }, [data]);

  const formatXAxis = (tickItem: any) => {
    const date = new Date(tickItem);
    return date.toLocaleTimeString('en-US', { 
      hour: '2-digit', 
      minute: '2-digit' 
    });
  };

  const formatTooltip = (value: any, name: string) => {
    if (value === null || value === undefined) return ['N/A', name];
    return [value.toLocaleString(), name];
  };

  const formatTooltipLabel = (label: any) => {
    const date = new Date(label);
    return date.toLocaleString();
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

  if (!data || data.length === 0 || chartData.length === 0) {
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

  return (
    <div className="chart-container">
      <div className="chart-header">
        <h3>{config.title}</h3>
        {config.description && <p>{config.description}</p>}
      </div>
      <div className="chart-content">
        <ResponsiveContainer width="100%" height={config.height || 300}>
          <RechartsLineChart
            data={chartData}
            margin={{
              top: 5,
              right: 30,
              left: 20,
              bottom: 5,
            }}
          >
            {config.showGrid !== false && (
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            )}
            <XAxis 
              dataKey="timestamp"
              tickFormatter={formatXAxis}
              stroke="#6b7280"
              fontSize={12}
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
            <Tooltip 
              formatter={formatTooltip}
              labelFormatter={formatTooltipLabel}
              contentStyle={{
                backgroundColor: 'white',
                border: '1px solid #d1d5db',
                borderRadius: '0.375rem',
                boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
              }}
            />
            {config.showLegend !== false && <Legend />}
            {data.map((series, index) => (
              <Line
                key={series.name}
                type="monotone"
                dataKey={series.name}
                stroke={series.color || config.colors?.[index] || `hsl(${index * 45}, 70%, 50%)`}
                strokeWidth={2}
                dot={{ r: 3 }}
                activeDot={{ r: 5 }}
                connectNulls={false}
              />
            ))}
          </RechartsLineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}