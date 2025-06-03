import React from 'react';
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
} from 'recharts';
import './Charts.css';

interface GaugeChartProps {
  value: number; // 0-100 percentage
  title: string;
  subtitle?: string;
  warningThreshold?: number;
  criticalThreshold?: number;
  loading?: boolean;
  size?: 'small' | 'medium' | 'large';
  unit?: string;
}

export function GaugeChart({ 
  value, 
  title, 
  subtitle,
  warningThreshold = 70,
  criticalThreshold = 90,
  loading = false,
  size = 'medium',
  unit = '%'
}: GaugeChartProps) {
  
  const normalizedValue = Math.max(0, Math.min(100, value));
  
  // Determine color based on thresholds
  const getColor = () => {
    if (normalizedValue >= criticalThreshold) {
      return '#ef4444'; // Red
    } else if (normalizedValue >= warningThreshold) {
      return '#f59e0b'; // Orange
    } else {
      return '#10b981'; // Green
    }
  };

  const getSize = () => {
    switch (size) {
      case 'small':
        return { width: 120, height: 80, outerRadius: 45, innerRadius: 30 };
      case 'large':
        return { width: 200, height: 130, outerRadius: 75, innerRadius: 50 };
      default: // medium
        return { width: 160, height: 100, outerRadius: 60, innerRadius: 40 };
    }
  };

  const dimensions = getSize();
  
  // Create gauge data (semicircle)
  const gaugeData = [
    { name: 'used', value: normalizedValue, fill: getColor() },
    { name: 'unused', value: 100 - normalizedValue, fill: '#f3f4f6' }
  ];

  const formatValue = (val: number) => {
    if (unit === '%') {
      return `${val.toFixed(1)}${unit}`;
    }
    return `${val.toLocaleString()}${unit}`;
  };

  if (loading) {
    return (
      <div className="gauge-chart-container" style={{ width: dimensions.width, height: dimensions.height + 40 }}>
        <div className="gauge-loading">
          <div className="loading-spinner">⟳</div>
        </div>
        <div className="gauge-labels">
          <div className="gauge-title">{title}</div>
          <div className="gauge-value">Loading...</div>
        </div>
      </div>
    );
  }

  return (
    <div className="gauge-chart-container" style={{ width: dimensions.width, height: dimensions.height + 40 }}>
      <div className="gauge-chart" style={{ height: dimensions.height }}>
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={gaugeData}
              cx="50%"
              cy="80%"
              startAngle={180}
              endAngle={0}
              innerRadius={dimensions.innerRadius}
              outerRadius={dimensions.outerRadius}
              paddingAngle={2}
              dataKey="value"
            >
              {gaugeData.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={entry.fill} />
              ))}
            </Pie>
          </PieChart>
        </ResponsiveContainer>
        
        {/* Center value display */}
        <div className="gauge-center">
          <div className="gauge-value" style={{ color: getColor() }}>
            {formatValue(normalizedValue)}
          </div>
          {subtitle && <div className="gauge-subtitle">{subtitle}</div>}
        </div>
        
        {/* Threshold indicators */}
        <div className="gauge-thresholds">
          <div 
            className="threshold-marker warning" 
            style={{ 
              left: `${warningThreshold}%`,
              opacity: normalizedValue >= warningThreshold ? 1 : 0.3 
            }}
          >
            ⚠
          </div>
          <div 
            className="threshold-marker critical" 
            style={{ 
              left: `${criticalThreshold}%`,
              opacity: normalizedValue >= criticalThreshold ? 1 : 0.3 
            }}
          >
            🚨
          </div>
        </div>
      </div>
      
      <div className="gauge-labels">
        <div className="gauge-title">{title}</div>
        {subtitle && <div className="gauge-subtitle">{subtitle}</div>}
      </div>
    </div>
  );
}