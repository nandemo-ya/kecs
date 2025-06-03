import React, { useMemo } from 'react';
import { HeatmapDataPoint, HeatmapConfig } from '../../types/timeseries';
import './Charts.css';

interface HeatmapChartProps {
  data: HeatmapDataPoint[];
  config: HeatmapConfig;
  loading?: boolean;
  onCellClick?: (dataPoint: HeatmapDataPoint) => void;
}

export function HeatmapChart({
  data,
  config,
  loading = false,
  onCellClick,
}: HeatmapChartProps) {
  
  // Process data into a 2D grid
  const { gridData, xLabels, yLabels, minValue, maxValue } = useMemo(() => {
    if (!data || data.length === 0) {
      return { gridData: [], xLabels: [], yLabels: [], minValue: 0, maxValue: 0 };
    }

    // Extract unique x and y values
    const xValues = Array.from(new Set(data.map(d => d.x))).sort((a, b) => a - b);
    const yValues = Array.from(new Set(data.map(d => d.y))).sort((a, b) => a - b);
    
    // Create labels
    const xLabels = xValues.map(x => {
      if (typeof x === 'number' && x > 1000000000) {
        // Treat as timestamp
        return new Date(x).toLocaleTimeString('en-US', { 
          hour: '2-digit', 
          minute: '2-digit' 
        });
      }
      return x.toString();
    });
    
    const yLabels = yValues.map(y => y.toString());
    
    // Find min and max values for color scaling
    const values = data.map(d => d.value);
    const minValue = Math.min(...values);
    const maxValue = Math.max(...values);
    
    // Create 2D grid
    const gridData: (HeatmapDataPoint | null)[][] = [];
    for (let i = 0; i < yValues.length; i++) {
      gridData[i] = [];
      for (let j = 0; j < xValues.length; j++) {
        const dataPoint = data.find(d => d.x === xValues[j] && d.y === yValues[i]);
        gridData[i][j] = dataPoint || null;
      }
    }
    
    return { gridData, xLabels, yLabels, minValue, maxValue };
  }, [data]);

  // Color scale functions
  const getColorScale = (value: number): string => {
    if (minValue === maxValue) return getColorFromScale(0.5, config.colorScale || 'blues');
    
    const normalized = (value - minValue) / (maxValue - minValue);
    return getColorFromScale(normalized, config.colorScale || 'blues');
  };

  const getColorFromScale = (value: number, scale: string): string => {
    // Clamp value between 0 and 1
    const v = Math.max(0, Math.min(1, value));
    
    switch (scale) {
      case 'blues':
        return `hsl(210, ${60 + v * 40}%, ${90 - v * 60}%)`;
      case 'reds':
        return `hsl(0, ${60 + v * 40}%, ${90 - v * 60}%)`;
      case 'greens':
        return `hsl(120, ${60 + v * 40}%, ${90 - v * 60}%)`;
      case 'viridis':
        // Simplified viridis color scale
        const r = Math.round(68 + v * (253 - 68));
        const g = Math.round(1 + v * (231 - 1));
        const b = Math.round(84 + v * (37 - 84));
        return `rgb(${r}, ${g}, ${b})`;
      case 'plasma':
        // Simplified plasma color scale
        const pr = Math.round(13 + v * (240 - 13));
        const pg = Math.round(8 + v * (249 - 8));
        const pb = Math.round(135 + v * (33 - 135));
        return `rgb(${pr}, ${pg}, ${pb})`;
      default:
        return `hsl(210, ${60 + v * 40}%, ${90 - v * 60}%)`;
    }
  };

  const formatValue = (value: number): string => {
    if (value >= 1000000) {
      return `${(value / 1000000).toFixed(1)}M`;
    } else if (value >= 1000) {
      return `${(value / 1000).toFixed(1)}K`;
    }
    return value.toFixed(2);
  };

  const cellSize = config.cellSize || 40;
  const showValues = config.showValues !== false;

  if (loading) {
    return (
      <div className="chart-container">
        <div className="chart-header">
          <h3>{config.title}</h3>
          {config.description && <p>{config.description}</p>}
        </div>
        <div className="chart-loading">
          <div className="loading-spinner">⟳</div>
          <p>Loading heatmap data...</p>
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
          <p>No heatmap data available</p>
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
        <div className="heatmap-container">
          {/* Y-axis labels */}
          {config.yAxisLabel && (
            <div className="heatmap-y-label">
              <span>{config.yAxisLabel}</span>
            </div>
          )}
          
          <div className="heatmap-content">
            {/* Y-axis */}
            <div className="heatmap-y-axis">
              {yLabels.map((label, index) => (
                <div
                  key={index}
                  className="heatmap-y-tick"
                  style={{ height: cellSize }}
                >
                  {label}
                </div>
              ))}
            </div>
            
            {/* Main heatmap grid */}
            <div className="heatmap-grid">
              {/* X-axis */}
              <div className="heatmap-x-axis">
                {xLabels.map((label, index) => (
                  <div
                    key={index}
                    className="heatmap-x-tick"
                    style={{ width: cellSize }}
                  >
                    {label}
                  </div>
                ))}
              </div>
              
              {/* Grid cells */}
              <div className="heatmap-cells">
                {gridData.map((row, rowIndex) => (
                  <div key={rowIndex} className="heatmap-row">
                    {row.map((cell, colIndex) => (
                      <div
                        key={colIndex}
                        className={`heatmap-cell ${cell ? 'has-data' : 'no-data'}`}
                        style={{
                          width: cellSize,
                          height: cellSize,
                          backgroundColor: cell ? getColorScale(cell.value) : '#f3f4f6',
                          cursor: cell && onCellClick ? 'pointer' : 'default',
                        }}
                        onClick={() => cell && onCellClick?.(cell)}
                        title={cell ? `${cell.label || ''} Value: ${formatValue(cell.value)}` : 'No data'}
                      >
                        {cell && showValues && (
                          <span className="heatmap-cell-value">
                            {formatValue(cell.value)}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          </div>
          
          {/* X-axis label */}
          {config.xAxisLabel && (
            <div className="heatmap-x-label">
              <span>{config.xAxisLabel}</span>
            </div>
          )}
        </div>
        
        {/* Color scale legend */}
        <div className="heatmap-legend">
          <div className="legend-title">Scale</div>
          <div className="legend-scale">
            <div className="legend-gradient">
              {Array.from({ length: 10 }, (_, i) => (
                <div
                  key={i}
                  className="legend-step"
                  style={{
                    backgroundColor: getColorFromScale(i / 9, config.colorScale || 'blues'),
                  }}
                />
              ))}
            </div>
            <div className="legend-labels">
              <span>{formatValue(minValue)}</span>
              <span>{formatValue(maxValue)}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}