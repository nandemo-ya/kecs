import React, { useState, useCallback } from 'react';
import { LineChart } from './charts/LineChart';
import { PieChart } from './charts/PieChart';
import { BarChart } from './charts/BarChart';
import { AutoRefreshToggle } from './AutoRefreshToggle';
import { useAutoRefresh } from '../hooks/useAutoRefresh';
import { useDashboardStats } from '../hooks/useApi';
import { 
  TimeSeriesData, 
  PieChartData, 
  BarChartData, 
  ChartConfig,
  TimeRange,
  TimeRangeConfig 
} from '../types/metrics';
import './charts/Charts.css';

const TIME_RANGES: TimeRangeConfig[] = [
  { label: '1 Hour', value: '1h', hours: 1, intervalMinutes: 5 },
  { label: '6 Hours', value: '6h', hours: 6, intervalMinutes: 15 },
  { label: '24 Hours', value: '24h', hours: 24, intervalMinutes: 60 },
  { label: '7 Days', value: '7d', hours: 168, intervalMinutes: 360 },
  { label: '30 Days', value: '30d', hours: 720, intervalMinutes: 1440 },
];

export function MetricsDashboard() {
  const [selectedTimeRange, setSelectedTimeRange] = useState<TimeRange>('24h');
  const [loading, setLoading] = useState(false);
  
  const { stats, loading: statsLoading, refresh: refreshStats } = useDashboardStats();

  // Generate mock time-series data for demonstration
  const generateMockTimeSeriesData = useCallback((timeRange: TimeRange): TimeSeriesData[] => {
    const config = TIME_RANGES.find(t => t.value === timeRange)!;
    const now = Date.now();
    const dataPoints: any[] = [];
    
    // Generate data points based on time range
    const totalPoints = Math.min(50, config.hours * 60 / config.intervalMinutes);
    
    for (let i = totalPoints; i >= 0; i--) {
      const timestamp = now - (i * config.intervalMinutes * 60 * 1000);
      dataPoints.push({
        timestamp,
        services: Math.floor(Math.random() * 10) + (stats?.services || 0),
        tasks: Math.floor(Math.random() * 20) + (stats?.tasks || 0),
        clusters: Math.floor(Math.random() * 3) + (stats?.clusters || 0),
      });
    }

    return [
      {
        name: 'Services',
        data: dataPoints.map(p => ({ timestamp: p.timestamp, value: p.services })),
        color: '#3b82f6',
      },
      {
        name: 'Tasks',
        data: dataPoints.map(p => ({ timestamp: p.timestamp, value: p.tasks })),
        color: '#10b981',
      },
      {
        name: 'Clusters',
        data: dataPoints.map(p => ({ timestamp: p.timestamp, value: p.clusters })),
        color: '#f59e0b',
      },
    ];
  }, [stats]);

  // Generate mock pie chart data
  const generateTaskStatusData = useCallback((): PieChartData[] => {
    const total = stats?.tasks || 0;
    const running = Math.floor(total * 0.7);
    const pending = Math.floor(total * 0.2);
    const stopped = total - running - pending;

    return [
      { name: 'Running', value: running, color: '#10b981' },
      { name: 'Pending', value: pending, color: '#f59e0b' },
      { name: 'Stopped', value: stopped, color: '#ef4444' },
    ];
  }, [stats]);

  // Generate mock bar chart data
  const generateClusterResourcesData = useCallback((): BarChartData[] => {
    const clusterNames = ['production', 'staging', 'development'];
    return clusterNames.map((name, index) => ({
      name,
      value: Math.floor(Math.random() * 50) + 10,
      fill: `hsl(${index * 120}, 70%, 50%)`,
    }));
  }, []);

  const refreshData = useCallback(() => {
    setLoading(true);
    refreshStats();
    // Simulate API call delay
    setTimeout(() => {
      setLoading(false);
    }, 500);
  }, [refreshStats]);

  const { isAutoRefreshEnabled, isRefreshing, toggleRefresh } = useAutoRefresh(refreshData, {
    interval: 30000, // 30 seconds for metrics
  });

  // Chart configurations
  const timeSeriesConfig: ChartConfig = {
    type: 'line',
    title: 'Resource Trends',
    description: 'Number of resources over time',
    yAxisLabel: 'Count',
    height: 400,
    showGrid: true,
    showLegend: true,
  };

  const taskStatusConfig: ChartConfig = {
    type: 'donut',
    title: 'Task Status Distribution',
    description: 'Current status of all tasks',
    height: 300,
    showLegend: true,
  };

  const clusterResourcesConfig: ChartConfig = {
    type: 'bar',
    title: 'Resources by Cluster',
    description: 'Total resources in each cluster',
    xAxisLabel: 'Cluster',
    yAxisLabel: 'Resources',
    height: 300,
    showGrid: true,
  };

  const timeSeriesData = generateMockTimeSeriesData(selectedTimeRange);
  const taskStatusData = generateTaskStatusData();
  const clusterResourcesData = generateClusterResourcesData();

  return (
    <main className="App-main">
      <div className="dashboard-header">
        <h2>📊 Metrics Dashboard</h2>
        <div className="header-actions">
          <AutoRefreshToggle
            isEnabled={isAutoRefreshEnabled}
            isRefreshing={isRefreshing}
            onToggle={toggleRefresh}
            interval={30000}
          />
          <button 
            className="refresh-button" 
            onClick={refreshData}
            disabled={loading}
            title="Refresh metrics data"
          >
            {loading ? '⟳' : '↻'}
          </button>
        </div>
      </div>

      {/* Summary Metrics Cards */}
      <div className="metric-cards-row">
        <div className="metric-card">
          <div className="metric-card-value">
            {statsLoading ? '...' : (stats?.clusters || 0)}
          </div>
          <div className="metric-card-label">Total Clusters</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">
            {statsLoading ? '...' : (stats?.services || 0)}
          </div>
          <div className="metric-card-label">Total Services</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">
            {statsLoading ? '...' : (stats?.tasks || 0)}
          </div>
          <div className="metric-card-label">Total Tasks</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">
            {statsLoading ? '...' : (stats?.taskDefinitions || 0)}
          </div>
          <div className="metric-card-label">Task Definitions</div>
        </div>
      </div>

      {/* Time Range Selector */}
      <div className="chart-controls">
        <div className="chart-controls-left">
          <span style={{ fontWeight: 600, color: '#374151' }}>Time Range:</span>
          <div className="time-range-selector">
            {TIME_RANGES.map((range) => (
              <button
                key={range.value}
                className={`time-range-button ${selectedTimeRange === range.value ? 'active' : ''}`}
                onClick={() => setSelectedTimeRange(range.value)}
              >
                {range.label}
              </button>
            ))}
          </div>
        </div>
        <div className="chart-controls-right">
          <span style={{ fontSize: '0.875rem', color: '#6b7280' }}>
            Last updated: {new Date().toLocaleTimeString()}
          </span>
        </div>
      </div>

      {/* Charts Grid */}
      <div className="charts-grid">
        {/* Time Series Chart - Full Width */}
        <div className="chart-full-width">
          <LineChart 
            data={timeSeriesData}
            config={timeSeriesConfig}
            loading={loading}
          />
        </div>

        {/* Pie Chart */}
        <PieChart 
          data={taskStatusData}
          config={taskStatusConfig}
          loading={loading}
        />

        {/* Bar Chart */}
        <BarChart 
          data={clusterResourcesData}
          config={clusterResourcesConfig}
          loading={loading}
        />
      </div>

      {/* Additional Information */}
      <div className="chart-container">
        <div className="chart-header">
          <h3>📈 Metrics Information</h3>
          <p>Real-time monitoring of KECS resources and performance</p>
        </div>
        <div style={{ padding: '1rem 0' }}>
          <h4>Available Metrics:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>Resource count trends over time</li>
            <li>Task status distribution</li>
            <li>Cluster resource allocation</li>
            <li>Service health and availability</li>
            <li>Performance monitoring (coming soon)</li>
          </ul>
          
          <h4 style={{ marginTop: '1.5rem' }}>Features:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>Auto-refresh with configurable intervals</li>
            <li>Multiple time range views</li>
            <li>Interactive charts with tooltips</li>
            <li>Real-time data updates</li>
            <li>Responsive design for all devices</li>
          </ul>
        </div>
      </div>
    </main>
  );
}