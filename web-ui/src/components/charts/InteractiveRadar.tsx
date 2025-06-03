import React, { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import * as d3 from 'd3';
import {
  RadarDataPoint,
  RadarSeries,
  InteractiveRadarProps,
  DEFAULT_CHART_CONFIG,
  DEFAULT_CHART_THEME,
  formatValue,
} from '../../types/interactiveCharts';
import './InteractiveCharts.css';

export function InteractiveRadar({
  axes,
  series,
  config = DEFAULT_CHART_CONFIG,
  maxValue,
  levels = 5,
  onSeriesClick,
  onSeriesHover,
  onAxisClick,
  selectedSeriesId,
  showGrid = true,
  showAxis = true,
  showLabels = true,
  showDots = true,
  animateOnLoad = true,
}: InteractiveRadarProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [hoveredSeries, setHoveredSeries] = useState<string | null>(null);
  const [hoveredAxis, setHoveredAxis] = useState<string | null>(null);
  const [visibleSeries, setVisibleSeries] = useState<Set<string>>(
    new Set(series.map(s => s.id))
  );

  const theme = DEFAULT_CHART_THEME;
  const margin = config.margin || DEFAULT_CHART_CONFIG.margin!;
  const width = (config.width || 600) - margin.left - margin.right;
  const height = (config.height || 600) - margin.top - margin.bottom;
  const radius = Math.min(width, height) / 2;
  const angleSlice = (Math.PI * 2) / axes.length;

  // Calculate max value if not provided
  const calculatedMaxValue = useMemo(() => {
    if (maxValue !== undefined) return maxValue;
    
    let max = 0;
    series.forEach(s => {
      s.data.forEach(d => {
        if (d.value > max) max = d.value;
      });
    });
    return max * 1.1; // Add 10% padding
  }, [series, maxValue]);

  // Create scales
  const rScale = d3.scaleLinear()
    .domain([0, calculatedMaxValue])
    .range([0, radius]);

  // Render chart
  useEffect(() => {
    if (!svgRef.current) return;

    // Clear previous content
    d3.select(svgRef.current).selectAll('*').remove();

    // Create SVG
    const svg = d3.select(svgRef.current)
      .attr('width', width + margin.left + margin.right)
      .attr('height', height + margin.top + margin.bottom);

    const g = svg.append('g')
      .attr('transform', `translate(${width / 2 + margin.left},${height / 2 + margin.top})`);

    // Draw grid circles
    if (showGrid) {
      const gridG = g.append('g').attr('class', 'grid');
      
      for (let level = 0; level < levels; level++) {
        const levelRadius = (radius / levels) * (level + 1);
        
        gridG.append('circle')
          .attr('r', levelRadius)
          .attr('class', 'radar-grid')
          .style('fill', 'none')
          .style('stroke', '#e5e7eb')
          .style('stroke-dasharray', '2,2');
        
        // Add level labels
        if (showLabels) {
          gridG.append('text')
            .attr('x', 5)
            .attr('y', -levelRadius)
            .attr('dy', '0.4em')
            .style('font-size', '10px')
            .style('fill', '#9ca3af')
            .text(formatValue((calculatedMaxValue / levels) * (level + 1)));
        }
      }
    }

    // Draw axes
    if (showAxis) {
      const axisG = g.append('g').attr('class', 'axes');
      
      axes.forEach((axis, i) => {
        const angle = angleSlice * i - Math.PI / 2;
        const x = Math.cos(angle) * radius;
        const y = Math.sin(angle) * radius;
        
        // Draw axis line
        axisG.append('line')
          .attr('x1', 0)
          .attr('y1', 0)
          .attr('x2', x)
          .attr('y2', y)
          .attr('class', 'radar-axis')
          .style('stroke', '#e5e7eb')
          .style('stroke-width', 1)
          .style('cursor', onAxisClick ? 'pointer' : 'default')
          .on('mouseenter', function() {
            setHoveredAxis(axis.id);
            d3.select(this).style('stroke', '#3b82f6').style('stroke-width', 2);
          })
          .on('mouseleave', function() {
            setHoveredAxis(null);
            d3.select(this).style('stroke', '#e5e7eb').style('stroke-width', 1);
          })
          .on('click', function() {
            if (onAxisClick) {
              onAxisClick(axis);
            }
          });
        
        // Draw axis labels
        if (showLabels) {
          const labelX = Math.cos(angle) * (radius + 20);
          const labelY = Math.sin(angle) * (radius + 20);
          
          axisG.append('text')
            .attr('x', labelX)
            .attr('y', labelY)
            .attr('text-anchor', 'middle')
            .attr('dy', '0.35em')
            .style('font-size', '12px')
            .style('font-weight', '500')
            .style('fill', hoveredAxis === axis.id ? '#3b82f6' : '#374151')
            .style('cursor', onAxisClick ? 'pointer' : 'default')
            .text(axis.label)
            .on('mouseenter', function() {
              setHoveredAxis(axis.id);
              d3.select(this).style('fill', '#3b82f6');
            })
            .on('mouseleave', function() {
              setHoveredAxis(null);
              d3.select(this).style('fill', '#374151');
            })
            .on('click', function() {
              if (onAxisClick) {
                onAxisClick(axis);
              }
            });
        }
      });
    }

    // Line generator
    const radarLine = d3.lineRadial<RadarDataPoint>()
      .radius(d => rScale(d.value))
      .angle((d, i) => angleSlice * i)
      .curve(d3.curveLinearClosed);

    // Draw series
    const seriesG = g.append('g').attr('class', 'series');
    
    series.forEach((seriesData, seriesIndex) => {
      if (!visibleSeries.has(seriesData.id)) return;
      
      const seriesGroup = seriesG.append('g')
        .attr('class', `series-${seriesData.id}`);
      
      // Ensure data is aligned with axes
      const alignedData = axes.map(axis => {
        const dataPoint = seriesData.data.find(d => d.axis === axis.id);
        return dataPoint || { axis: axis.id, value: 0 };
      });
      
      // Draw polygon
      const polygon = seriesGroup.append('path')
        .datum(alignedData)
        .attr('d', radarLine as any)
        .attr('class', 'radar-polygon')
        .style('fill', seriesData.color || theme.colors?.[seriesIndex % theme.colors.length] || '#3b82f6')
        .style('fill-opacity', hoveredSeries === seriesData.id ? 0.5 : 0.3)
        .style('stroke', seriesData.color || theme.colors?.[seriesIndex % theme.colors.length] || '#3b82f6')
        .style('stroke-width', selectedSeriesId === seriesData.id ? 3 : 2)
        .style('cursor', 'pointer')
        .on('mouseenter', function() {
          setHoveredSeries(seriesData.id);
          d3.select(this).style('fill-opacity', 0.5);
          if (onSeriesHover) {
            onSeriesHover(seriesData);
          }
        })
        .on('mouseleave', function() {
          setHoveredSeries(null);
          d3.select(this).style('fill-opacity', 0.3);
          if (onSeriesHover) {
            onSeriesHover(null);
          }
        })
        .on('click', function() {
          if (onSeriesClick) {
            onSeriesClick(seriesData);
          }
        });
      
      // Animate on load
      if (animateOnLoad) {
        polygon
          .style('fill-opacity', 0)
          .style('stroke-opacity', 0)
          .transition()
          .duration(config.animationDuration || 300)
          .delay(seriesIndex * 100)
          .style('fill-opacity', 0.3)
          .style('stroke-opacity', 1);
      }
      
      // Draw dots
      if (showDots) {
        alignedData.forEach((d, i) => {
          const angle = angleSlice * i - Math.PI / 2;
          const x = Math.cos(angle) * rScale(d.value);
          const y = Math.sin(angle) * rScale(d.value);
          
          seriesGroup.append('circle')
            .attr('cx', x)
            .attr('cy', y)
            .attr('r', 4)
            .attr('class', 'radar-dot')
            .style('fill', 'white')
            .style('stroke', seriesData.color || theme.colors?.[seriesIndex % theme.colors.length] || '#3b82f6')
            .style('stroke-width', 2)
            .style('cursor', 'pointer')
            .on('mouseenter', function() {
              d3.select(this).attr('r', 6);
            })
            .on('mouseleave', function() {
              d3.select(this).attr('r', 4);
            })
            .append('title')
            .text(`${axes[i].label}: ${formatValue(d.value)}`);
        });
      }
    });
    
  }, [axes, series, visibleSeries, width, height, radius, calculatedMaxValue, levels, 
      showGrid, showAxis, showLabels, showDots, hoveredSeries, hoveredAxis, 
      selectedSeriesId, animateOnLoad, config.animationDuration]);

  // Toggle series visibility
  const toggleSeries = useCallback((seriesId: string) => {
    setVisibleSeries(prev => {
      const newSet = new Set(prev);
      if (newSet.has(seriesId)) {
        newSet.delete(seriesId);
      } else {
        newSet.add(seriesId);
      }
      return newSet;
    });
  }, []);

  // Export functionality
  const exportChart = useCallback((format: 'png' | 'svg' | 'json') => {
    if (format === 'json') {
      const data = { axes, series };
      const dataStr = JSON.stringify(data, null, 2);
      const blob = new Blob([dataStr], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `radar-chart-${Date.now()}.json`;
      link.click();
      URL.revokeObjectURL(url);
    } else if (format === 'svg' && svgRef.current) {
      const svgData = new XMLSerializer().serializeToString(svgRef.current);
      const blob = new Blob([svgData], { type: 'image/svg+xml' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `radar-chart-${Date.now()}.svg`;
      link.click();
      URL.revokeObjectURL(url);
    }
  }, [axes, series]);

  // Calculate statistics
  const stats = useMemo(() => {
    const visibleSeriesData = series.filter(s => visibleSeries.has(s.id));
    const allValues = visibleSeriesData.flatMap(s => s.data.map(d => d.value));
    
    return {
      seriesCount: visibleSeriesData.length,
      axisCount: axes.length,
      maxValue: Math.max(...allValues),
      minValue: Math.min(...allValues),
      avgValue: allValues.reduce((sum, val) => sum + val, 0) / allValues.length,
    };
  }, [series, axes, visibleSeries]);

  return (
    <div className="interactive-radar interactive-chart">
      <div className="chart-header">
        <div className="chart-info">
          <span className="total-value">{stats.seriesCount} series</span>
          <span className="segment-count">{stats.axisCount} dimensions</span>
        </div>
        <div className="chart-actions">
          <button 
            className="export-button"
            onClick={() => exportChart('svg')}
            title="Export as SVG"
          >
            🖼️
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

      <div className="radar-container">
        <svg ref={svgRef}></svg>
      </div>

      {/* Legend */}
      <div className="radar-legend">
        {series.map((s, i) => (
          <div 
            key={s.id}
            className={`legend-item ${!visibleSeries.has(s.id) ? 'disabled' : ''}`}
            onClick={() => toggleSeries(s.id)}
            style={{ cursor: 'pointer' }}
          >
            <span 
              className="legend-color"
              style={{
                backgroundColor: s.color || theme.colors?.[i % theme.colors.length] || '#3b82f6',
                display: 'inline-block',
                width: '12px',
                height: '12px',
                borderRadius: '2px',
                marginRight: '8px',
                opacity: visibleSeries.has(s.id) ? 1 : 0.3,
              }}
            />
            <span style={{ opacity: visibleSeries.has(s.id) ? 1 : 0.5 }}>
              {s.name}
            </span>
          </div>
        ))}
      </div>

      {/* Statistics */}
      <div className="radar-statistics">
        <div className="stat-item">
          <span className="stat-label">Max:</span>
          <span className="stat-value">{formatValue(stats.maxValue)}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Min:</span>
          <span className="stat-value">{formatValue(stats.minValue)}</span>
        </div>
        <div className="stat-item">
          <span className="stat-label">Average:</span>
          <span className="stat-value">{formatValue(stats.avgValue)}</span>
        </div>
      </div>
    </div>
  );
}