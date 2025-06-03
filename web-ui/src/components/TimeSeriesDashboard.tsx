import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { TimeSeriesChart } from './charts/TimeSeriesChart';
import { HeatmapChart } from './charts/HeatmapChart';
import {
  TimeSeriesData,
  TimeSeriesChartConfig,
  HeatmapConfig,
  HeatmapDataPoint,
  TimeRange,
  TIME_SERIES_PRESETS,
} from '../types/timeseries';
import './charts/Charts.css';

export function TimeSeriesDashboard() {
  const [selectedTimeRange, setSelectedTimeRange] = useState<string>('last-day');
  const [loading, setLoading] = useState(true);
  const [realTimeEnabled, setRealTimeEnabled] = useState(false);
  const [timeSeriesData, setTimeSeriesData] = useState<TimeSeriesData[]>([]);
  const [heatmapData, setHeatmapData] = useState<HeatmapDataPoint[]>([]);

  // Time range presets
  const timeRangePresets = [
    { value: 'last-hour', label: '1 Hour', hours: 1 },
    { value: 'last-day', label: '24 Hours', hours: 24 },
    { value: 'last-week', label: '7 Days', hours: 168 },
    { value: 'last-month', label: '30 Days', hours: 720 },
  ];

  // Generate mock time series data
  const generateTimeSeriesData = useCallback((hours: number): TimeSeriesData[] => {
    const now = Date.now();
    const interval = hours <= 1 ? 60000 : hours <= 24 ? 300000 : hours <= 168 ? 3600000 : 86400000; // 1min, 5min, 1hour, 1day
    const points = Math.min(200, (hours * 3600000) / interval);

    // CPU Usage Series
    const cpuData = Array.from({ length: points }, (_, i) => {
      const timestamp = now - (points - 1 - i) * interval;
      const baseValue = 30 + Math.sin((i / points) * Math.PI * 4) * 20;
      const noise = (Math.random() - 0.5) * 10;
      return {
        timestamp,
        value: Math.max(5, Math.min(95, baseValue + noise)),
        label: `CPU at ${new Date(timestamp).toLocaleTimeString()}`,
      };
    });

    // Memory Usage Series
    const memoryData = Array.from({ length: points }, (_, i) => {
      const timestamp = now - (points - 1 - i) * interval;
      const baseValue = 45 + Math.sin((i / points) * Math.PI * 3 + 1) * 15;
      const noise = (Math.random() - 0.5) * 8;
      return {
        timestamp,
        value: Math.max(10, Math.min(90, baseValue + noise)),
        label: `Memory at ${new Date(timestamp).toLocaleTimeString()}`,
      };
    });

    // Network I/O Series
    const networkData = Array.from({ length: points }, (_, i) => {
      const timestamp = now - (points - 1 - i) * interval;
      const spikes = Math.random() > 0.9 ? Math.random() * 500 : 0;
      const baseValue = 50 + Math.sin((i / points) * Math.PI * 6) * 30 + spikes;
      return {
        timestamp,
        value: Math.max(0, baseValue),
        label: `Network I/O at ${new Date(timestamp).toLocaleTimeString()}`,
      };
    });

    // Response Time Series
    const responseTimeData = Array.from({ length: points }, (_, i) => {
      const timestamp = now - (points - 1 - i) * interval;
      const baseValue = 100 + Math.sin((i / points) * Math.PI * 2) * 50;
      const spikes = Math.random() > 0.95 ? Math.random() * 500 : 0;
      const noise = (Math.random() - 0.5) * 20;
      return {
        timestamp,
        value: Math.max(10, baseValue + spikes + noise),
        label: `Response Time at ${new Date(timestamp).toLocaleTimeString()}`,
      };
    });

    return [
      {
        id: 'cpu',
        name: 'CPU Usage',
        color: '#3b82f6',
        data: cpuData,
        unit: '%',
        type: 'line',
        yAxisId: 'percentage',
      },
      {
        id: 'memory',
        name: 'Memory Usage',
        color: '#10b981',
        data: memoryData,
        unit: '%',
        type: 'area',
        yAxisId: 'percentage',
      },
      {
        id: 'network',
        name: 'Network I/O',
        color: '#f59e0b',
        data: networkData,
        unit: 'MB/s',
        type: 'line',
        yAxisId: 'throughput',
      },
      {
        id: 'responseTime',
        name: 'Response Time',
        color: '#ef4444',
        data: responseTimeData,
        unit: 'ms',
        type: 'line',
        yAxisId: 'latency',
      },
    ];
  }, []);

  // Generate mock heatmap data (hourly patterns over a week)
  const generateHeatmapData = useCallback((): HeatmapDataPoint[] => {
    const data: HeatmapDataPoint[] = [];
    const daysOfWeek = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
    
    for (let day = 0; day < 7; day++) {
      for (let hour = 0; hour < 24; hour++) {
        // Simulate realistic usage patterns
        let baseValue = 20;
        
        // Business hours (9-17) have higher usage
        if (hour >= 9 && hour <= 17) {
          baseValue += 40;
        }
        
        // Weekends have lower usage
        if (day >= 5) {
          baseValue *= 0.6;
        }
        
        // Lunch time dip
        if (hour >= 12 && hour <= 13) {
          baseValue *= 0.8;
        }
        
        // Add some randomness
        const noise = (Math.random() - 0.5) * 20;
        const value = Math.max(0, Math.min(100, baseValue + noise));
        
        data.push({
          x: hour,
          y: day,
          value,
          label: `${daysOfWeek[day]} ${hour}:00 - ${value.toFixed(1)}%`,
        });
      }
    }
    
    return data;
  }, []);

  // Load data based on selected time range
  useEffect(() => {
    setLoading(true);
    
    const loadData = () => {
      const preset = timeRangePresets.find(p => p.value === selectedTimeRange);
      const hours = preset?.hours || 24;
      
      setTimeSeriesData(generateTimeSeriesData(hours));
      setHeatmapData(generateHeatmapData());
      setLoading(false);
    };

    // Simulate loading delay
    const timer = setTimeout(loadData, 300);
    return () => clearTimeout(timer);
  }, [selectedTimeRange, generateTimeSeriesData, generateHeatmapData]);

  // Real-time updates
  useEffect(() => {
    if (!realTimeEnabled) return;

    const interval = setInterval(() => {
      setTimeSeriesData(prevData => {
        const preset = timeRangePresets.find(p => p.value === selectedTimeRange);
        const hours = preset?.hours || 24;
        return generateTimeSeriesData(hours);
      });
    }, 5000); // Update every 5 seconds

    return () => clearInterval(interval);
  }, [realTimeEnabled, selectedTimeRange, generateTimeSeriesData]);

  // Chart configurations
  const systemMetricsConfig: TimeSeriesChartConfig = {
    title: 'System Resource Usage',
    description: 'CPU and Memory utilization over time',
    height: 350,
    showLegend: true,
    showGrid: true,
    enableZoom: true,
    enableBrush: true,
    interpolation: 'monotone',
    yAxes: [
      {
        id: 'percentage',
        orientation: 'left',
        domain: [0, 100],
        label: 'Usage (%)',
        unit: '%',
        color: '#6b7280',
      },
    ],
    thresholds: [
      {
        id: 'cpu-warning',
        value: 70,
        label: 'CPU Warning',
        color: '#f59e0b',
        strokeStyle: 'dashed',
        yAxisId: 'percentage',
      },
      {
        id: 'memory-critical',
        value: 90,
        label: 'Memory Critical',
        color: '#ef4444',
        strokeStyle: 'dashed',
        yAxisId: 'percentage',
      },
    ],
  };

  const networkMetricsConfig: TimeSeriesChartConfig = {
    title: 'Network & Performance Metrics',
    description: 'Network throughput and response times',
    height: 350,
    showLegend: true,
    showGrid: true,
    enableZoom: true,
    interpolation: 'linear',
    yAxes: [
      {
        id: 'throughput',
        orientation: 'left',
        domain: ['dataMin', 'dataMax'],
        label: 'Throughput (MB/s)',
        unit: 'MB/s',
        color: '#f59e0b',
      },
      {
        id: 'latency',
        orientation: 'right',
        domain: ['dataMin', 'dataMax'],
        label: 'Latency (ms)',
        unit: 'ms',
        color: '#ef4444',
      },
    ],
  };

  const heatmapConfig: HeatmapConfig = {
    title: 'Weekly Usage Patterns',
    description: 'Resource usage heatmap by day and hour',
    height: 300,
    xAxisLabel: 'Hour of Day',
    yAxisLabel: 'Day of Week',
    colorScale: 'blues',
    showValues: false,
    cellSize: 25,
  };

  // Filter data for each chart
  const systemData = useMemo(() => 
    timeSeriesData.filter(series => ['cpu', 'memory'].includes(series.id)),
    [timeSeriesData]
  );

  const networkData = useMemo(() => 
    timeSeriesData.filter(series => ['network', 'responseTime'].includes(series.id)),
    [timeSeriesData]
  );

  return (
    <div className="timeseries-dashboard">
      <div className="page-header">
        <div className="page-title">
          <h1>Time Series Analytics</h1>
          <p>Advanced temporal data visualization and analysis</p>
        </div>
        
        <div className="page-actions">
          <Link to="/metrics" className="btn btn-secondary">
            ← Back to Metrics
          </Link>
        </div>
      </div>

      {/* Time Range and Controls */}
      <div className="timeseries-time-controls">
        <div className="timeseries-time-controls-left">
          <label>Time Range:</label>
          <div className="timeseries-preset-buttons">
            {timeRangePresets.map(preset => (
              <button
                key={preset.value}
                className={`timeseries-preset-button ${selectedTimeRange === preset.value ? 'active' : ''}`}
                onClick={() => setSelectedTimeRange(preset.value)}
              >
                {preset.label}
              </button>
            ))}
          </div>
        </div>
        
        <div className="timeseries-time-controls-right">
          <label>
            <input
              type="checkbox"
              checked={realTimeEnabled}
              onChange={(e) => setRealTimeEnabled(e.target.checked)}
            />
            Real-time Updates
          </label>
          
          {realTimeEnabled && (
            <div className="timeseries-realtime-indicator">
              <div className="timeseries-realtime-dot"></div>
              <span>Live</span>
            </div>
          )}
        </div>
      </div>

      {/* Charts Grid */}
      <div className="charts-grid">
        {/* System Metrics */}
        <div className="chart-full-width">
          <TimeSeriesChart
            data={systemData}
            config={systemMetricsConfig}
            loading={loading}
          />
        </div>

        {/* Network Metrics */}
        <div className="chart-full-width">
          <TimeSeriesChart
            data={networkData}
            config={networkMetricsConfig}
            loading={loading}
          />
        </div>

        {/* Heatmap */}
        <div className="chart-full-width">
          <HeatmapChart
            data={heatmapData}
            config={heatmapConfig}
            loading={loading}
          />
        </div>
      </div>

      {/* Summary Statistics */}
      <div className="charts-grid charts-grid-2">
        <div className="chart-container">
          <div className="chart-header">
            <h3>📊 Data Summary</h3>
            <p>Key metrics from the selected time range</p>
          </div>
          <div className="metric-cards-row">
            {timeSeriesData.slice(0, 2).map(series => {
              const values = series.data.map(d => d.value);
              const avg = values.reduce((a, b) => a + b, 0) / values.length;
              const max = Math.max(...values);
              
              return (
                <div key={series.id} className="metric-card">
                  <div className="metric-card-value" style={{ color: series.color }}>
                    {avg.toFixed(1)}{series.unit}
                  </div>
                  <div className="metric-card-label">{series.name} Avg</div>
                  <div className="metric-card-change neutral">
                    Peak: {max.toFixed(1)}{series.unit}
                  </div>
                </div>
              );
            })}
          </div>
        </div>

        <div className="chart-container">
          <div className="chart-header">
            <h3>⚡ Real-time Status</h3>
            <p>Current system performance indicators</p>
          </div>
          <div className="metric-cards-row">
            {timeSeriesData.slice(2, 4).map(series => {
              const latestValue = series.data[series.data.length - 1]?.value || 0;
              const previousValue = series.data[series.data.length - 2]?.value || 0;
              const change = latestValue - previousValue;
              const changeClass = change > 0 ? 'positive' : change < 0 ? 'negative' : 'neutral';
              
              return (
                <div key={series.id} className="metric-card">
                  <div className="metric-card-value" style={{ color: series.color }}>
                    {latestValue.toFixed(1)}{series.unit}
                  </div>
                  <div className="metric-card-label">{series.name}</div>
                  <div className={`metric-card-change ${changeClass}`}>
                    {change > 0 ? '+' : ''}{change.toFixed(1)}{series.unit}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Feature Information */}
      <div className="chart-container">
        <div className="chart-header">
          <h3>📈 Time Series Features</h3>
          <p>Advanced temporal data analysis capabilities</p>
        </div>
        <div style={{ padding: '1rem 0' }}>
          <div className="charts-grid charts-grid-3">
            <div>
              <h4>📊 Multi-Series Charts</h4>
              <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280', fontSize: '0.875rem' }}>
                <li>Multiple data series with different Y-axes</li>
                <li>Line, area, and scatter plot support</li>
                <li>Configurable interpolation methods</li>
                <li>Interactive tooltips and legends</li>
              </ul>
            </div>
            
            <div>
              <h4>🔥 Heatmap Visualization</h4>
              <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280', fontSize: '0.875rem' }}>
                <li>Temporal pattern analysis</li>
                <li>Multiple color scales (blues, reds, viridis)</li>
                <li>Interactive cell selection</li>
                <li>Weekly/daily usage patterns</li>
              </ul>
            </div>
            
            <div>
              <h4>⚙️ Interactive Controls</h4>
              <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280', fontSize: '0.875rem' }}>
                <li>Zoom and pan functionality</li>
                <li>Time range brush selection</li>
                <li>Real-time data streaming</li>
                <li>Threshold and annotation lines</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}