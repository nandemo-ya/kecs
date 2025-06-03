import React, { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Tooltip,
  Legend,
  Sector,
} from 'recharts';
import {
  InteractivePieData,
  InteractivePieChartProps,
  DEFAULT_CHART_CONFIG,
  DEFAULT_CHART_THEME,
  formatValue,
} from '../../types/interactiveCharts';
import './InteractiveCharts.css';

// Custom active shape for interactive hover effect
const renderActiveShape = (props: any) => {
  const RADIAN = Math.PI / 180;
  const {
    cx, cy, midAngle, innerRadius, outerRadius, startAngle, endAngle,
    fill, payload, percent, value,
  } = props;
  const sin = Math.sin(-RADIAN * midAngle);
  const cos = Math.cos(-RADIAN * midAngle);
  const sx = cx + (outerRadius + 10) * cos;
  const sy = cy + (outerRadius + 10) * sin;
  const mx = cx + (outerRadius + 25) * cos;
  const my = cy + (outerRadius + 25) * sin;
  const ex = mx + (cos >= 0 ? 1 : -1) * 20;
  const ey = my;
  const textAnchor = cos >= 0 ? 'start' : 'end';

  return (
    <g>
      <text x={cx} y={cy} dy={-8} textAnchor="middle" fill={fill} className="pie-center-text">
        {payload.label}
      </text>
      <text x={cx} y={cy} dy={12} textAnchor="middle" fill="#666" fontSize="14">
        {`${formatValue(value)}`}
      </text>
      <Sector
        cx={cx}
        cy={cy}
        innerRadius={innerRadius}
        outerRadius={outerRadius}
        startAngle={startAngle}
        endAngle={endAngle}
        fill={fill}
      />
      <Sector
        cx={cx}
        cy={cy}
        startAngle={startAngle}
        endAngle={endAngle}
        innerRadius={outerRadius + 6}
        outerRadius={outerRadius + 10}
        fill={fill}
      />
      <path d={`M${sx},${sy}L${mx},${my}L${ex},${ey}`} stroke={fill} fill="none"/>
      <circle cx={ex} cy={ey} r={2} fill={fill} stroke="none"/>
      <text x={ex + (cos >= 0 ? 1 : -1) * 12} y={ey} textAnchor={textAnchor} fill="#333" className="pie-label-text">
        {payload.label}
      </text>
      <text x={ex + (cos >= 0 ? 1 : -1) * 12} y={ey} dy={18} textAnchor={textAnchor} fill="#999" className="pie-value-text">
        {`${formatValue(value)} (${(percent * 100).toFixed(1)}%)`}
      </text>
    </g>
  );
};

// Custom tooltip
const CustomTooltip = ({ active, payload }: any) => {
  if (active && payload && payload.length) {
    const data = payload[0];
    return (
      <div className="interactive-chart-tooltip">
        <p className="tooltip-label">{data.name}</p>
        <p className="tooltip-value">
          Value: <span>{formatValue(data.value)}</span>
        </p>
        <p className="tooltip-percent">
          Percentage: <span>{(data.percent * 100).toFixed(1)}%</span>
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

export function InteractivePieChart({
  data,
  config = DEFAULT_CHART_CONFIG,
  drillDown,
  onSegmentClick,
  onSegmentHover,
  selectedSegmentId,
  innerRadius = 0,
  outerRadius = 80,
  padAngle = 1,
  cornerRadius = 4,
}: InteractivePieChartProps) {
  const [activeIndex, setActiveIndex] = useState<number>(-1);
  const [drillDownPath, setDrillDownPath] = useState<string[]>([]);
  const [currentData, setCurrentData] = useState<InteractivePieData[]>(data);
  const chartRef = useRef<HTMLDivElement>(null);

  // Get the effective data based on drill-down state
  useEffect(() => {
    let effectiveData = data;
    
    // Navigate through drill-down path
    for (const pathId of drillDownPath) {
      const parent = effectiveData.find(d => d.id === pathId);
      if (parent?.children) {
        effectiveData = parent.children;
      }
    }
    
    setCurrentData(effectiveData);
  }, [data, drillDownPath]);

  // Calculate total value
  const totalValue = useMemo(() => {
    return currentData.reduce((sum, item) => sum + item.value, 0);
  }, [currentData]);

  // Handle pie enter (hover)
  const onPieEnter = useCallback((data: any, index: number) => {
    setActiveIndex(index);
    if (onSegmentHover) {
      onSegmentHover(currentData[index]);
    }
  }, [currentData, onSegmentHover]);

  // Handle pie leave
  const onPieLeave = useCallback(() => {
    setActiveIndex(-1);
    if (onSegmentHover) {
      onSegmentHover(null);
    }
  }, [onSegmentHover]);

  // Handle segment click
  const handleSegmentClick = useCallback((data: any, index: number) => {
    const segment = currentData[index];
    
    // Handle drill-down
    if (drillDown?.enabled && segment.children && segment.children.length > 0) {
      setDrillDownPath([...drillDownPath, segment.id]);
      if (drillDown.onDrillDown) {
        drillDown.onDrillDown(segment.id, segment);
      }
    }
    
    // Call click handler
    if (onSegmentClick) {
      onSegmentClick(segment);
    }
  }, [currentData, drillDown, drillDownPath, onSegmentClick]);

  // Handle drill-up
  const handleDrillUp = useCallback(() => {
    if (drillDownPath.length > 0) {
      const newPath = [...drillDownPath];
      newPath.pop();
      setDrillDownPath(newPath);
      if (drillDown?.onDrillUp) {
        drillDown.onDrillUp();
      }
    }
  }, [drillDownPath, drillDown]);

  // Export chart
  const exportChart = useCallback((format: 'png' | 'svg' | 'json') => {
    if (format === 'json') {
      const dataStr = JSON.stringify(currentData, null, 2);
      const blob = new Blob([dataStr], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `pie-chart-${Date.now()}.json`;
      link.click();
      URL.revokeObjectURL(url);
    } else {
      // For PNG/SVG export, we would need to implement SVG serialization
      console.log('Export format not yet implemented:', format);
    }
  }, [currentData]);

  const chartMargin = config.margin || DEFAULT_CHART_CONFIG.margin!;
  const theme = DEFAULT_CHART_THEME;

  return (
    <div className="interactive-pie-chart interactive-chart" ref={chartRef}>
      {/* Chart Header */}
      <div className="chart-header">
        {drillDownPath.length > 0 && (
          <button className="drill-up-button" onClick={handleDrillUp}>
            ← Back
          </button>
        )}
        <div className="chart-info">
          <span className="total-value">Total: {formatValue(totalValue)}</span>
          <span className="segment-count">{currentData.length} segments</span>
        </div>
        <div className="chart-actions">
          <button 
            className="export-button"
            onClick={() => exportChart('json')}
            title="Export as JSON"
          >
            📥
          </button>
        </div>
      </div>

      {/* Main Chart */}
      <ResponsiveContainer width="100%" height={config.height || 400}>
        <PieChart margin={chartMargin}>
          <Pie
            activeIndex={activeIndex}
            activeShape={renderActiveShape}
            data={currentData}
            cx="50%"
            cy="50%"
            innerRadius={innerRadius}
            outerRadius={outerRadius}
            fill={theme.colors?.[0] || '#3b82f6'}
            dataKey="value"
            nameKey="label"
            paddingAngle={padAngle}
            animationBegin={0}
            animationDuration={config.animationDuration || 300}
            onMouseEnter={onPieEnter}
            onMouseLeave={onPieLeave}
            onClick={handleSegmentClick}
          >
            {currentData.map((entry, index) => (
              <Cell 
                key={`cell-${entry.id}`} 
                fill={entry.color || theme.colors?.[index % theme.colors.length] || '#3b82f6'}
                stroke={selectedSegmentId === entry.id ? '#3b82f6' : 'none'}
                strokeWidth={selectedSegmentId === entry.id ? 2 : 0}
                className={`pie-segment ${selectedSegmentId === entry.id ? 'selected' : ''} ${entry.children ? 'has-children' : ''}`}
                style={{
                  filter: activeIndex === index ? 'brightness(1.1)' : 'none',
                  cursor: entry.children ? 'pointer' : 'default',
                }}
              />
            ))}
          </Pie>
          {config.showTooltip !== false && (
            <Tooltip content={<CustomTooltip />} />
          )}
          {config.showLegend !== false && (
            <Legend 
              verticalAlign="bottom" 
              height={50}
              wrapperStyle={{ paddingTop: '20px' }}
              formatter={(value, entry) => (
                <span style={{ color: entry.color, fontSize: '14px' }}>
                  {value} ({((entry.payload?.value || 0) / totalValue * 100).toFixed(1)}%)
                </span>
              )}
            />
          )}
        </PieChart>
      </ResponsiveContainer>

      {/* Drill-down breadcrumb */}
      {drillDownPath.length > 0 && (
        <div className="drill-down-breadcrumb">
          <span className="breadcrumb-item" onClick={() => setDrillDownPath([])}>
            Root
          </span>
          {drillDownPath.map((pathId, index) => {
            const pathData = data.find(d => d.id === pathId);
            return (
              <React.Fragment key={pathId}>
                <span className="breadcrumb-separator">›</span>
                <span 
                  className="breadcrumb-item"
                  onClick={() => setDrillDownPath(drillDownPath.slice(0, index + 1))}
                >
                  {pathData?.label || pathId}
                </span>
              </React.Fragment>
            );
          })}
        </div>
      )}

      {/* Segment Details */}
      {activeIndex >= 0 && currentData[activeIndex] && (
        <div className="segment-details">
          <h4>{currentData[activeIndex].label}</h4>
          <div className="detail-stats">
            <div className="stat-item">
              <span className="stat-label">Value:</span>
              <span className="stat-value">{formatValue(currentData[activeIndex].value)}</span>
            </div>
            <div className="stat-item">
              <span className="stat-label">Percentage:</span>
              <span className="stat-value">
                {((currentData[activeIndex].value / totalValue) * 100).toFixed(1)}%
              </span>
            </div>
            {currentData[activeIndex].children && (
              <div className="stat-item">
                <span className="stat-label">Sub-segments:</span>
                <span className="stat-value">{currentData[activeIndex].children!.length}</span>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}