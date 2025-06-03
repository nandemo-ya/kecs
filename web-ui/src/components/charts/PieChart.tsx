import React from 'react';
import {
  PieChart as RechartsPieChart,
  Pie,
  Cell,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { PieChartData, ChartConfig } from '../../types/metrics';
import './Charts.css';

interface PieChartProps {
  data: PieChartData[];
  config: ChartConfig;
  loading?: boolean;
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

export function PieChart({ data, config, loading = false }: PieChartProps) {
  const total = React.useMemo(() => {
    return data.reduce((sum, item) => sum + item.value, 0);
  }, [data]);

  const formatTooltip = (value: any, name: string) => {
    const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0';
    return [`${value.toLocaleString()} (${percentage}%)`, name];
  };

  const renderCustomLabel = (entry: any) => {
    const percentage = total > 0 ? ((entry.value / total) * 100).toFixed(1) : '0.0';
    return `${percentage}%`;
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

  const isDonut = config.type === 'donut';

  return (
    <div className="chart-container">
      <div className="chart-header">
        <h3>{config.title}</h3>
        {config.description && <p>{config.description}</p>}
      </div>
      <div className="chart-content">
        <ResponsiveContainer width="100%" height={config.height || 300}>
          <RechartsPieChart>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              labelLine={false}
              label={renderCustomLabel}
              outerRadius={isDonut ? 100 : 120}
              innerRadius={isDonut ? 60 : 0}
              fill="#8884d8"
              dataKey="value"
            >
              {data.map((entry, index) => (
                <Cell 
                  key={`cell-${index}`} 
                  fill={entry.color || config.colors?.[index] || DEFAULT_COLORS[index % DEFAULT_COLORS.length]} 
                />
              ))}
            </Pie>
            <Tooltip 
              formatter={formatTooltip}
              contentStyle={{
                backgroundColor: 'white',
                border: '1px solid #d1d5db',
                borderRadius: '0.375rem',
                boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
              }}
            />
            {config.showLegend !== false && (
              <Legend 
                verticalAlign="bottom" 
                height={36}
                iconType="circle"
              />
            )}
          </RechartsPieChart>
        </ResponsiveContainer>
        
        {/* Display total in center for donut charts */}
        {isDonut && (
          <div className="donut-center">
            <div className="donut-total">
              <span className="donut-total-value">{total.toLocaleString()}</span>
              <span className="donut-total-label">Total</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}