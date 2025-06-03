import React, { useState, useMemo } from 'react';
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine,
  Brush,
  ReferenceArea,
} from 'recharts';
import {
  TimeSeriesData,
  TimeSeriesChartConfig,
  TimeSeriesAnnotation,
  TimeSeriesThreshold,
  YAxisConfig,
} from '../../types/timeseries';
import './Charts.css';

interface TimeSeriesChartProps {
  data: TimeSeriesData[];
  config: TimeSeriesChartConfig;
  loading?: boolean;
  onZoom?: (domain: [number, number]) => void;
  onBrush?: (domain: [number, number]) => void;
}

export function TimeSeriesChart({
  data,
  config,
  loading = false,
  onZoom,
  onBrush,
}: TimeSeriesChartProps) {
  const [zoomDomain, setZoomDomain] = useState<[number, number] | null>(null);
  const [brushDomain, setBrushDomain] = useState<[number, number] | null>(null);

  // Transform data for Recharts
  const chartData = useMemo(() => {
    if (!data || data.length === 0) return [];

    // Collect all unique timestamps
    const timestampSet = new Set<number>();
    data.forEach(series => {
      if (series.visible !== false) {
        series.data.forEach(point => timestampSet.add(point.timestamp));
      }
    });

    const timestamps = Array.from(timestampSet).sort((a, b) => a - b);

    // Create combined data points
    return timestamps.map(timestamp => {
      const point: any = { timestamp };
      
      data.forEach(series => {
        if (series.visible !== false) {
          const dataPoint = series.data.find(p => p.timestamp === timestamp);
          point[series.id] = dataPoint?.value ?? null;
        }
      });

      return point;
    });
  }, [data]);

  // Format functions
  const formatXAxis = (tickItem: number) => {
    const date = new Date(tickItem);
    const now = new Date();
    const diffHours = (now.getTime() - date.getTime()) / (1000 * 60 * 60);

    if (diffHours < 24) {
      return date.toLocaleTimeString('en-US', { 
        hour: '2-digit', 
        minute: '2-digit' 
      });
    } else if (diffHours < 24 * 7) {
      return date.toLocaleDateString('en-US', { 
        month: 'short', 
        day: 'numeric',
        hour: '2-digit'
      });
    } else {
      return date.toLocaleDateString('en-US', { 
        month: 'short', 
        day: 'numeric' 
      });
    }
  };

  const formatTooltip = (value: any, name: string) => {
    if (value === null || value === undefined) return ['N/A', name];
    
    const series = data.find(s => s.id === name);
    const unit = series?.unit || '';
    
    if (unit === '%') {
      return [`${value.toFixed(2)}%`, series?.name || name];
    } else if (unit === 'bytes') {
      return [formatBytes(value), series?.name || name];
    } else if (unit === 'ms' || unit === 'seconds') {
      return [formatDuration(value, unit), series?.name || name];
    } else {
      return [`${value.toLocaleString()}${unit ? ` ${unit}` : ''}`, series?.name || name];
    }
  };

  const formatTooltipLabel = (label: number) => {
    const date = new Date(label);
    return date.toLocaleString();
  };

  const formatBytes = (bytes: number): string => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    
    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  const formatDuration = (value: number, unit: string): string => {
    if (unit === 'ms') {
      if (value >= 1000) {
        return `${(value / 1000).toFixed(2)}s`;
      }
      return `${value.toFixed(2)}ms`;
    } else if (unit === 'seconds') {
      if (value >= 60) {
        const minutes = Math.floor(value / 60);
        const seconds = value % 60;
        return `${minutes}m ${seconds.toFixed(1)}s`;
      }
      return `${value.toFixed(2)}s`;
    }
    return `${value} ${unit}`;
  };

  // Event handlers
  const handleZoom = (e: any) => {
    if (e && e.startIndex !== undefined && e.endIndex !== undefined) {
      const startTime = chartData[e.startIndex]?.timestamp;
      const endTime = chartData[e.endIndex]?.timestamp;
      if (startTime && endTime) {
        const domain: [number, number] = [startTime, endTime];
        setZoomDomain(domain);
        onZoom?.(domain);
      }
    }
  };

  const handleBrushChange = (e: any) => {
    if (e && e.startIndex !== undefined && e.endIndex !== undefined) {
      const startTime = chartData[e.startIndex]?.timestamp;
      const endTime = chartData[e.endIndex]?.timestamp;
      if (startTime && endTime) {
        const domain: [number, number] = [startTime, endTime];
        setBrushDomain(domain);
        onBrush?.(domain);
      }
    }
  };

  // Loading state
  if (loading) {
    return (
      <div className="chart-container">
        <div className="chart-header">
          <h3>{config.title}</h3>
          {config.description && <p>{config.description}</p>}
        </div>
        <div className="chart-loading">
          <div className="loading-spinner">⟳</div>
          <p>Loading time series data...</p>
        </div>
      </div>
    );
  }

  // Empty state
  if (!chartData || chartData.length === 0) {
    return (
      <div className="chart-container">
        <div className="chart-header">
          <h3>{config.title}</h3>
          {config.description && <p>{config.description}</p>}
        </div>
        <div className="chart-empty">
          <p>No time series data available</p>
        </div>
      </div>
    );
  }

  // Determine chart type based on data
  const hasAreaSeries = data.some(series => series.type === 'area');
  const ChartComponent = hasAreaSeries ? AreaChart : LineChart;

  return (
    <div className="chart-container">
      <div className="chart-header">
        <h3>{config.title}</h3>
        {config.description && <p>{config.description}</p>}
        
        {/* Series indicators */}
        <div className="timeseries-indicators">
          {data.filter(series => series.visible !== false).map(series => (
            <div key={series.id} className="timeseries-indicator">
              <span 
                className="indicator-dot" 
                style={{ backgroundColor: series.color }}
              ></span>
              <span>{series.name}</span>
              {series.unit && <span className="unit">({series.unit})</span>}
            </div>
          ))}
        </div>
      </div>
      
      <div className="chart-content">
        <ResponsiveContainer width="100%" height={config.height || 400}>
          <ChartComponent
            data={chartData}
            margin={{
              top: 10,
              right: 30,
              left: 20,
              bottom: config.enableBrush ? 60 : 5,
            }}
            onMouseDown={config.enableZoom ? handleZoom : undefined}
          >
            {config.showGrid && (
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            )}
            
            <XAxis 
              dataKey="timestamp"
              type="number"
              scale="time"
              domain={zoomDomain || ['dataMin', 'dataMax']}
              tickFormatter={formatXAxis}
              stroke="#6b7280"
              fontSize={12}
            />
            
            {/* Y-Axes */}
            {config.yAxes ? (
              config.yAxes.map(yAxis => (
                <YAxis
                  key={yAxis.id}
                  yAxisId={yAxis.id}
                  orientation={yAxis.orientation}
                  domain={yAxis.domain || ['auto', 'auto']}
                  tickFormatter={yAxis.tickFormatter}
                  stroke={yAxis.color || "#6b7280"}
                  fontSize={12}
                  label={yAxis.label ? { value: yAxis.label, angle: -90, position: 'insideLeft' } : undefined}
                />
              ))
            ) : (
              <YAxis 
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
            
            {config.showLegend && <Legend />}
            
            {/* Thresholds */}
            {config.thresholds?.map(threshold => (
              <ReferenceLine
                key={threshold.id}
                y={threshold.value}
                yAxisId={threshold.yAxisId}
                stroke={threshold.color}
                strokeDasharray={threshold.strokeStyle === 'dashed' ? '5 5' : undefined}
                label={threshold.label}
              />
            ))}
            
            {/* Annotations */}
            {config.annotations?.map(annotation => {
              if (annotation.type === 'vertical-line' && annotation.timestamp) {
                return (
                  <ReferenceLine
                    key={annotation.id}
                    x={annotation.timestamp}
                    stroke={annotation.color || '#666'}
                    strokeDasharray={annotation.strokeStyle === 'dashed' ? '5 5' : undefined}
                    label={annotation.label}
                  />
                );
              } else if (annotation.type === 'horizontal-line' && annotation.value !== undefined) {
                return (
                  <ReferenceLine
                    key={annotation.id}
                    y={annotation.value}
                    stroke={annotation.color || '#666'}
                    strokeDasharray={annotation.strokeStyle === 'dashed' ? '5 5' : undefined}
                    label={annotation.label}
                  />
                );
              }
              return null;
            })}
            
            {/* Data Series */}
            {data.filter(series => series.visible !== false).map(series => {
              if (series.type === 'area') {
                return (
                  <Area
                    key={series.id}
                    type={config.interpolation || 'monotone'}
                    dataKey={series.id}
                    stroke={series.color}
                    fill={series.color}
                    fillOpacity={0.2}
                    strokeWidth={2}
                    name={series.name}
                    yAxisId={series.yAxisId}
                    connectNulls={false}
                  />
                );
              } else {
                return (
                  <Line
                    key={series.id}
                    type={config.interpolation || 'monotone'}
                    dataKey={series.id}
                    stroke={series.color}
                    strokeWidth={2}
                    dot={false}
                    name={series.name}
                    yAxisId={series.yAxisId}
                    connectNulls={false}
                  />
                );
              }
            })}
            
            {/* Brush for time range selection */}
            {config.enableBrush && (
              <Brush
                dataKey="timestamp"
                height={30}
                stroke="#8884d8"
                onChange={handleBrushChange}
                tickFormatter={formatXAxis}
              />
            )}
          </ChartComponent>
        </ResponsiveContainer>
      </div>
    </div>
  );
}