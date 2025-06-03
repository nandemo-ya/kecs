import React from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';
import { ResourceChartData, ResourceChartConfig } from '../../types/resourceUsage';
import './Charts.css';

interface ResourceUsageChartProps {
  data: ResourceChartData[];
  config: ResourceChartConfig;
  loading?: boolean;
}

export function ResourceUsageChart({ data, config, loading = false }: ResourceUsageChartProps) {
  const formatXAxis = (tickItem: any) => {
    const date = new Date(tickItem);
    return date.toLocaleTimeString('en-US', { 
      hour: '2-digit', 
      minute: '2-digit' 
    });
  };

  const formatTooltip = (value: any, name: string) => {
    if (value === null || value === undefined) return ['N/A', name];
    
    if (name.includes('Usage') || name.includes('Utilization')) {
      return [`${value.toFixed(1)}%`, name];
    }
    
    if (name.includes('CPU')) {
      return [`${value} units`, name];
    }
    
    if (name.includes('Memory')) {
      return [`${value} MB`, name];
    }
    
    return [value.toLocaleString(), name];
  };

  const formatTooltipLabel = (label: any) => {
    const date = new Date(label);
    return date.toLocaleString();
  };

  const formatYAxis = (value: any) => {
    if (config.metric === 'cpu' && !config.title.includes('Usage')) {
      return `${value}`;
    }
    return `${value}%`;
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
          <p>Loading resource usage data...</p>
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
          <p>No resource usage data available</p>
        </div>
      </div>
    );
  }

  const isPercentageChart = config.metric === 'memory' || config.title.includes('Usage');

  return (
    <div className="chart-container">
      <div className="chart-header">
        <h3>{config.title}</h3>
        {config.description && <p>{config.description}</p>}
        
        {/* Resource usage indicators */}
        <div className="resource-indicators">
          {config.metric === 'cpu' || config.metric === 'both' ? (
            <div className="resource-indicator cpu">
              <span className="indicator-dot cpu"></span>
              <span>CPU Usage</span>
            </div>
          ) : null}
          
          {config.metric === 'memory' || config.metric === 'both' ? (
            <div className="resource-indicator memory">
              <span className="indicator-dot memory"></span>
              <span>Memory Usage</span>
            </div>
          ) : null}
          
          {config.showThresholds && (
            <>
              {config.warningThreshold && (
                <div className="resource-indicator warning">
                  <span className="indicator-dot warning"></span>
                  <span>Warning ({config.warningThreshold}%)</span>
                </div>
              )}
              {config.criticalThreshold && (
                <div className="resource-indicator critical">
                  <span className="indicator-dot critical"></span>
                  <span>Critical ({config.criticalThreshold}%)</span>
                </div>
              )}
            </>
          )}
        </div>
      </div>
      
      <div className="chart-content">
        <ResponsiveContainer width="100%" height={config.height || 400}>
          <AreaChart
            data={data}
            margin={{
              top: 10,
              right: 30,
              left: 20,
              bottom: 5,
            }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis 
              dataKey="timestamp"
              tickFormatter={formatXAxis}
              stroke="#6b7280"
              fontSize={12}
            />
            {isPercentageChart ? (
              <YAxis 
                domain={[0, 100]}
                tickFormatter={formatYAxis}
                stroke="#6b7280"
                fontSize={12}
              />
            ) : (
              <YAxis 
                domain={[0, 'dataMax']}
                tickFormatter={formatYAxis}
                stroke="#6b7280"
                fontSize={12}
              />
            )}
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
            <Legend />
            
            {/* Threshold lines */}
            {config.showThresholds && config.warningThreshold && (
              <ReferenceLine 
                y={config.warningThreshold} 
                stroke="#f59e0b" 
                strokeDasharray="5 5"
                label="Warning"
              />
            )}
            {config.showThresholds && config.criticalThreshold && (
              <ReferenceLine 
                y={config.criticalThreshold} 
                stroke="#ef4444" 
                strokeDasharray="5 5"
                label="Critical"
              />
            )}
            
            {/* CPU Usage Area */}
            {(config.metric === 'cpu' || config.metric === 'both') && (
              <Area
                type="monotone"
                dataKey="cpu"
                stackId="1"
                stroke="#3b82f6"
                fill="url(#cpuGradient)"
                strokeWidth={2}
                name="CPU Usage"
                connectNulls={false}
              />
            )}
            
            {/* Memory Usage Area */}
            {(config.metric === 'memory' || config.metric === 'both') && (
              <Area
                type="monotone"
                dataKey="memory"
                stackId={config.metric === 'both' ? "2" : "1"}
                stroke="#10b981"
                fill="url(#memoryGradient)"
                strokeWidth={2}
                name="Memory Usage"
                connectNulls={false}
              />
            )}
            
            {/* Gradients definition */}
            <defs>
              <linearGradient id="cpuGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                <stop offset="95%" stopColor="#3b82f6" stopOpacity={0.1}/>
              </linearGradient>
              <linearGradient id="memoryGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#10b981" stopOpacity={0.3}/>
                <stop offset="95%" stopColor="#10b981" stopOpacity={0.1}/>
              </linearGradient>
            </defs>
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}