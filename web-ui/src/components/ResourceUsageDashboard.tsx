import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { CpuUsageChart } from './charts/CpuUsageChart';
import { MemoryUsageChart } from './charts/MemoryUsageChart';
import { CombinedUsageChart } from './charts/CombinedUsageChart';
import { GaugeChart } from './charts/GaugeChart';
import { ResourceChartData, ResourceTimeRange } from '../types/resourceUsage';
import './charts/Charts.css';

export function ResourceUsageDashboard() {
  const [timeRange, setTimeRange] = useState<ResourceTimeRange>('1h');
  const [loading, setLoading] = useState(true);
  const [chartData, setChartData] = useState<ResourceChartData[]>([]);
  const [currentCpuUsage, setCurrentCpuUsage] = useState(0);
  const [currentMemoryUsage, setCurrentMemoryUsage] = useState(0);

  useEffect(() => {
    const generateMockData = () => {
      const now = Date.now();
      const dataPoints = timeRange === '1h' ? 12 : timeRange === '6h' ? 36 : timeRange === '24h' ? 48 : 168;
      const interval = timeRange === '1h' ? 5 * 60 * 1000 : timeRange === '6h' ? 10 * 60 * 1000 : timeRange === '24h' ? 30 * 60 * 1000 : 60 * 60 * 1000;
      
      const data: ResourceChartData[] = [];
      
      for (let i = dataPoints - 1; i >= 0; i--) {
        const timestamp = now - (i * interval);
        
        // Generate realistic usage patterns
        const baseTime = new Date(timestamp).getHours();
        const cpuBase = 30 + Math.sin(baseTime / 24 * Math.PI * 2) * 20;
        const memoryBase = 40 + Math.sin((baseTime + 6) / 24 * Math.PI * 2) * 15;
        
        // Add some randomness
        const cpuNoise = (Math.random() - 0.5) * 20;
        const memoryNoise = (Math.random() - 0.5) * 15;
        
        const cpu = Math.max(5, Math.min(95, cpuBase + cpuNoise));
        const memory = Math.max(10, Math.min(90, memoryBase + memoryNoise));
        
        data.push({
          timestamp,
          cpu: Math.round(cpu * 10) / 10,
          memory: Math.round(memory * 10) / 10,
          cpuLimit: 1024, // 1 vCPU
          memoryLimit: 2048, // 2GB
          cpuRequest: 512, // 0.5 vCPU
          memoryRequest: 1024, // 1GB
        });
      }
      
      // Set current usage from the latest data point
      if (data.length > 0) {
        const latest = data[data.length - 1];
        setCurrentCpuUsage(latest.cpu || 0);
        setCurrentMemoryUsage(latest.memory || 0);
      }
      
      return data;
    };

    setLoading(true);
    
    // Simulate API call delay
    const timer = setTimeout(() => {
      setChartData(generateMockData());
      setLoading(false);
    }, 500);

    return () => clearTimeout(timer);
  }, [timeRange]);

  const timeRangeOptions = [
    { value: '1h' as const, label: '1 Hour' },
    { value: '6h' as const, label: '6 Hours' },
    { value: '24h' as const, label: '24 Hours' },
    { value: '7d' as const, label: '7 Days' },
  ];

  return (
    <div className="resource-usage-dashboard">
      <div className="page-header">
        <div className="page-title">
          <h1>Resource Usage Dashboard</h1>
          <p>Monitor CPU and memory utilization across your cluster</p>
        </div>
        
        <div className="page-actions">
          <Link to="/metrics" className="btn btn-secondary">
            ← Back to Metrics
          </Link>
        </div>
      </div>

      {/* Time Range Controls */}
      <div className="chart-controls">
        <div className="chart-controls-left">
          <span>Time Range:</span>
          <div className="time-range-selector">
            {timeRangeOptions.map(option => (
              <button
                key={option.value}
                className={`time-range-button ${timeRange === option.value ? 'active' : ''}`}
                onClick={() => setTimeRange(option.value)}
              >
                {option.label}
              </button>
            ))}
          </div>
        </div>
        
        <div className="chart-controls-right">
          <button 
            className="chart-refresh-button"
            onClick={() => {
              setLoading(true);
              // Force re-fetch by updating a dependency
              setTimeRange(prev => prev);
            }}
            disabled={loading}
          >
            <span>⟳</span>
            Refresh
          </button>
        </div>
      </div>

      {/* Current Usage Gauges */}
      <div className="metric-cards-row">
        <div className="chart-container">
          <div className="chart-header">
            <h3>Current CPU Usage</h3>
            <p>Real-time CPU utilization</p>
          </div>
          <GaugeChart
            value={currentCpuUsage}
            title="CPU"
            subtitle="Current Usage"
            warningThreshold={70}
            criticalThreshold={90}
            loading={loading}
            size="medium"
          />
        </div>
        
        <div className="chart-container">
          <div className="chart-header">
            <h3>Current Memory Usage</h3>
            <p>Real-time memory utilization</p>
          </div>
          <GaugeChart
            value={currentMemoryUsage}
            title="Memory"
            subtitle="Current Usage"
            warningThreshold={80}
            criticalThreshold={95}
            loading={loading}
            size="medium"
          />
        </div>
      </div>

      {/* Usage Charts Grid */}
      <div className="charts-grid">
        {/* Combined Usage Chart - Full Width */}
        <div className="chart-full-width">
          <CombinedUsageChart
            data={chartData}
            timeRange={timeRange}
            loading={loading}
            height={350}
          />
        </div>

        {/* Individual Usage Charts */}
        <CpuUsageChart
          data={chartData}
          timeRange={timeRange}
          loading={loading}
          height={300}
        />
        
        <MemoryUsageChart
          data={chartData}
          timeRange={timeRange}
          loading={loading}
          height={300}
        />
      </div>

      {/* Usage Statistics */}
      <div className="charts-grid charts-grid-3">
        <div className="chart-container">
          <div className="chart-header">
            <h3>CPU Statistics</h3>
            <p>Usage patterns for {timeRange}</p>
          </div>
          <div className="metric-cards-row">
            <div className="metric-card">
              <div className="metric-card-value">
                {chartData.length > 0 ? Math.round(chartData.reduce((sum, d) => sum + (d.cpu || 0), 0) / chartData.length) : 0}%
              </div>
              <div className="metric-card-label">Average</div>
            </div>
            <div className="metric-card">
              <div className="metric-card-value">
                {chartData.length > 0 ? Math.round(Math.max(...chartData.map(d => d.cpu || 0))) : 0}%
              </div>
              <div className="metric-card-label">Peak</div>
            </div>
          </div>
        </div>

        <div className="chart-container">
          <div className="chart-header">
            <h3>Memory Statistics</h3>
            <p>Usage patterns for {timeRange}</p>
          </div>
          <div className="metric-cards-row">
            <div className="metric-card">
              <div className="metric-card-value">
                {chartData.length > 0 ? Math.round(chartData.reduce((sum, d) => sum + (d.memory || 0), 0) / chartData.length) : 0}%
              </div>
              <div className="metric-card-label">Average</div>
            </div>
            <div className="metric-card">
              <div className="metric-card-value">
                {chartData.length > 0 ? Math.round(Math.max(...chartData.map(d => d.memory || 0))) : 0}%
              </div>
              <div className="metric-card-label">Peak</div>
            </div>
          </div>
        </div>

        <div className="chart-container">
          <div className="chart-header">
            <h3>Resource Efficiency</h3>
            <p>Usage vs allocated resources</p>
          </div>
          <div className="metric-cards-row">
            <div className="metric-card">
              <div className="metric-card-value">
                {chartData.length > 0 && chartData[0].cpuRequest && chartData[0].cpuLimit 
                  ? Math.round((chartData[0].cpuRequest / chartData[0].cpuLimit) * 100) 
                  : 50}%
              </div>
              <div className="metric-card-label">CPU Requested</div>
            </div>
            <div className="metric-card">
              <div className="metric-card-value">
                {chartData.length > 0 && chartData[0].memoryRequest && chartData[0].memoryLimit 
                  ? Math.round((chartData[0].memoryRequest / chartData[0].memoryLimit) * 100) 
                  : 50}%
              </div>
              <div className="metric-card-label">Memory Requested</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}