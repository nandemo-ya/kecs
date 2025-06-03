import React, { useState, useCallback, useMemo, useRef } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  Cell,
  ReferenceLine,
  Brush,
} from 'recharts';
import {
  InteractiveBarData,
  InteractiveBarChartProps,
  DEFAULT_CHART_CONFIG,
  DEFAULT_CHART_THEME,
  formatValue,
} from '../../types/interactiveCharts';
import './InteractiveCharts.css';

// Custom tooltip
const CustomTooltip = ({ active, payload, label }: any) => {
  if (active && payload && payload.length) {
    const data = payload[0];
    return (
      <div className="interactive-chart-tooltip">
        <p className="tooltip-label">{label}</p>
        <p className="tooltip-value">
          Value: <span>{formatValue(data.value)}</span>
        </p>
        {data.payload.metadata && (
          <div className="tooltip-metadata">
            {Object.entries(data.payload.metadata).map(([key, value]) => (
              <p key={key} className="tooltip-meta-item">
                {key}: <span>{String(value)}</span>
              </p>
            ))}
          </div>
        )}
      </div>
    );
  }
  return null;
};

// Custom bar shape with animation
const CustomBar = (props: any) => {
  const { fill, x, y, width, height } = props;
  return (
    <rect
      x={x}
      y={y}
      width={width}
      height={height}
      fill={fill}
      className="bar-segment"
      rx={4}
      ry={4}
    />
  );
};

export function InteractiveBarChart({
  data,
  config = DEFAULT_CHART_CONFIG,
  orientation = 'vertical',
  sortBy = 'none',
  sortOrder = 'desc',
  onBarClick,
  onBarHover,
  selectedBarId,
  groupBy,
  stackBy,
  showValues = false,
  enableSorting = true,
  enableFiltering = false,
  filterOptions = [],
}: InteractiveBarChartProps) {
  const [activeIndex, setActiveIndex] = useState<number>(-1);
  const [currentSortBy, setCurrentSortBy] = useState(sortBy);
  const [currentSortOrder, setCurrentSortOrder] = useState(sortOrder);
  const [filters, setFilters] = useState<Record<string, any>>({});
  const [brushIndexes, setBrushIndexes] = useState<{ startIndex?: number; endIndex?: number }>({});
  const chartRef = useRef<HTMLDivElement>(null);

  // Sort data
  const sortedData = useMemo(() => {
    if (currentSortBy === 'none') return data;
    
    const sorted = [...data].sort((a, b) => {
      if (currentSortBy === 'value') {
        return currentSortOrder === 'asc' ? a.value - b.value : b.value - a.value;
      } else if (currentSortBy === 'category') {
        return currentSortOrder === 'asc' 
          ? a.category.localeCompare(b.category)
          : b.category.localeCompare(a.category);
      }
      return 0;
    });
    
    return sorted;
  }, [data, currentSortBy, currentSortOrder]);

  // Filter data
  const filteredData = useMemo(() => {
    if (!enableFiltering || Object.keys(filters).length === 0) return sortedData;
    
    return sortedData.filter(item => {
      for (const [field, value] of Object.entries(filters)) {
        if (value && item.metadata && item.metadata[field] !== value) {
          return false;
        }
      }
      return true;
    });
  }, [sortedData, filters, enableFiltering]);

  // Get visible data based on brush
  const visibleData = useMemo(() => {
    if (brushIndexes.startIndex !== undefined && brushIndexes.endIndex !== undefined) {
      return filteredData.slice(brushIndexes.startIndex, brushIndexes.endIndex + 1);
    }
    return filteredData;
  }, [filteredData, brushIndexes]);

  // Calculate statistics
  const stats = useMemo(() => {
    const values = visibleData.map(d => d.value);
    return {
      total: values.reduce((sum, val) => sum + val, 0),
      average: values.reduce((sum, val) => sum + val, 0) / values.length,
      max: Math.max(...values),
      min: Math.min(...values),
    };
  }, [visibleData]);

  // Handle bar click
  const handleBarClick = useCallback((data: any, index: number) => {
    if (onBarClick) {
      onBarClick(filteredData[index]);
    }
  }, [filteredData, onBarClick]);

  // Handle bar hover
  const handleBarMouseEnter = useCallback((data: any, index: number) => {
    setActiveIndex(index);
    if (onBarHover) {
      onBarHover(filteredData[index]);
    }
  }, [filteredData, onBarHover]);

  const handleBarMouseLeave = useCallback(() => {
    setActiveIndex(-1);
    if (onBarHover) {
      onBarHover(null);
    }
  }, [onBarHover]);

  // Handle sorting
  const handleSort = useCallback((newSortBy: typeof sortBy) => {
    if (newSortBy === currentSortBy) {
      setCurrentSortOrder(currentSortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setCurrentSortBy(newSortBy);
      setCurrentSortOrder('desc');
    }
  }, [currentSortBy, currentSortOrder]);

  // Handle filtering
  const handleFilterChange = useCallback((field: string, value: any) => {
    setFilters(prev => ({
      ...prev,
      [field]: value || undefined,
    }));
  }, []);

  // Export chart
  const exportChart = useCallback((format: 'png' | 'svg' | 'json' | 'csv') => {
    if (format === 'json') {
      const dataStr = JSON.stringify(visibleData, null, 2);
      const blob = new Blob([dataStr], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `bar-chart-${Date.now()}.json`;
      link.click();
      URL.revokeObjectURL(url);
    } else if (format === 'csv') {
      const headers = ['Category', 'Value'];
      const rows = visibleData.map(d => [d.category, d.value]);
      const csv = [headers, ...rows].map(row => row.join(',')).join('\n');
      const blob = new Blob([csv], { type: 'text/csv' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `bar-chart-${Date.now()}.csv`;
      link.click();
      URL.revokeObjectURL(url);
    }
  }, [visibleData]);

  const chartMargin = config.margin || DEFAULT_CHART_CONFIG.margin!;
  const theme = DEFAULT_CHART_THEME;
  const isHorizontal = orientation === 'horizontal';

  return (
    <div className="interactive-bar-chart interactive-chart" ref={chartRef}>
      {/* Chart Header */}
      <div className="chart-header">
        <div className="chart-info">
          <span className="total-value">Total: {formatValue(stats.total)}</span>
          <span className="segment-count">{visibleData.length} items</span>
        </div>
        <div className="chart-actions">
          <button 
            className="export-button"
            onClick={() => exportChart('csv')}
            title="Export as CSV"
          >
            📊
          </button>
          <button 
            className="export-button"
            onClick={() => exportChart('json')}
            title="Export as JSON"
          >
            📥
          </button>
        </div>
      </div>

      {/* Controls */}
      <div className="bar-controls">
        {enableSorting && (
          <div className="sort-controls">
            <span>Sort by:</span>
            <button
              className={`sort-button ${currentSortBy === 'value' ? 'active' : ''}`}
              onClick={() => handleSort('value')}
            >
              Value {currentSortBy === 'value' && (currentSortOrder === 'asc' ? '↑' : '↓')}
            </button>
            <button
              className={`sort-button ${currentSortBy === 'category' ? 'active' : ''}`}
              onClick={() => handleSort('category')}
            >
              Category {currentSortBy === 'category' && (currentSortOrder === 'asc' ? '↑' : '↓')}
            </button>
            <button
              className={`sort-button ${currentSortBy === 'none' ? 'active' : ''}`}
              onClick={() => handleSort('none')}
            >
              Original
            </button>
          </div>
        )}
        
        {enableFiltering && filterOptions.length > 0 && (
          <div className="filter-controls">
            {filterOptions.map(option => (
              <select
                key={option.field}
                className="filter-select"
                value={filters[option.field] || ''}
                onChange={(e) => handleFilterChange(option.field, e.target.value)}
              >
                <option value="">{option.label}</option>
                {option.values?.map(value => (
                  <option key={String(value)} value={String(value)}>
                    {String(value)}
                  </option>
                ))}
              </select>
            ))}
          </div>
        )}
      </div>

      {/* Main Chart */}
      <ResponsiveContainer width="100%" height={config.height || 400}>
        <BarChart
          data={filteredData}
          margin={chartMargin}
          layout={isHorizontal ? 'horizontal' : 'vertical'}
        >
          <CartesianGrid strokeDasharray="3 3" stroke={theme.gridColor} />
          
          {isHorizontal ? (
            <>
              <XAxis type="number" stroke={theme.axisColor} />
              <YAxis 
                dataKey="category" 
                type="category" 
                stroke={theme.axisColor}
                width={100}
              />
            </>
          ) : (
            <>
              <XAxis 
                dataKey="category" 
                stroke={theme.axisColor}
                angle={-45}
                textAnchor="end"
                height={80}
              />
              <YAxis type="number" stroke={theme.axisColor} />
            </>
          )}
          
          {config.showTooltip !== false && (
            <Tooltip content={<CustomTooltip />} />
          )}
          
          {config.showLegend !== false && (
            <Legend />
          )}
          
          <ReferenceLine 
            {...(isHorizontal ? { x: stats.average } : { y: stats.average })}
            stroke="#6b7280"
            strokeDasharray="8 4"
            label={{ value: `Avg: ${formatValue(stats.average)}`, position: 'top' }}
          />
          
          <Bar 
            dataKey="value" 
            fill={theme.colors?.[0] || '#3b82f6'}
            shape={<CustomBar />}
            onClick={handleBarClick}
            onMouseEnter={handleBarMouseEnter}
            onMouseLeave={handleBarMouseLeave}
          >
            {filteredData.map((entry, index) => (
              <Cell 
                key={`cell-${entry.id}`}
                fill={entry.color || theme.colors?.[index % theme.colors.length] || '#3b82f6'}
                stroke={selectedBarId === entry.id ? '#1f2937' : 'none'}
                strokeWidth={selectedBarId === entry.id ? 2 : 0}
                style={{
                  filter: activeIndex === index ? 'brightness(1.2)' : 'none',
                  cursor: 'pointer',
                }}
              />
            ))}
          </Bar>
          
          {filteredData.length > 20 && (
            <Brush
              dataKey="category"
              height={30}
              stroke={theme.colors?.[0] || '#3b82f6'}
              onChange={(indexes: any) => setBrushIndexes(indexes)}
            />
          )}
        </BarChart>
      </ResponsiveContainer>

      {/* Statistics */}
      <div className="bar-statistics">
        <div className="stat-item">
          <span className="stat-label">Average:</span>
          <span className="stat-value">{formatValue(stats.average)}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Max:</span>
          <span className="stat-value">{formatValue(stats.max)}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Min:</span>
          <span className="stat-value">{formatValue(stats.min)}</span>
        </div>
      </div>
    </div>
  );
}