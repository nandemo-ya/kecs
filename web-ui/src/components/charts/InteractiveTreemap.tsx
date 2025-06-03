import React, { useState, useCallback, useMemo, useRef, useEffect } from 'react';
import * as d3 from 'd3';
import {
  TreemapNode,
  InteractiveTreemapProps,
  DEFAULT_CHART_CONFIG,
  DEFAULT_CHART_THEME,
  formatValue,
} from '../../types/interactiveCharts';
import './InteractiveCharts.css';

interface D3Node extends d3.HierarchyRectangularNode<TreemapNode> {
  x0: number;
  y0: number;
  x1: number;
  y1: number;
}

export function InteractiveTreemap({
  data,
  config = DEFAULT_CHART_CONFIG,
  tileType = 'squarify',
  onNodeClick,
  onNodeHover,
  selectedNodeId,
  colorScale,
  enableZoom = true,
  maxDepth = Infinity,
  valueFormat,
}: InteractiveTreemapProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [currentRoot, setCurrentRoot] = useState<TreemapNode>(data);
  const [breadcrumb, setBreadcrumb] = useState<TreemapNode[]>([]);

  const theme = DEFAULT_CHART_THEME;
  const margin = config.margin || DEFAULT_CHART_CONFIG.margin!;
  const width = (config.width || 800) - margin.left - margin.right;
  const height = (config.height || 600) - margin.top - margin.bottom;

  // Create hierarchy and layout
  useEffect(() => {
    if (!svgRef.current) return;

    // Clear previous content
    d3.select(svgRef.current).selectAll('*').remove();

    // Create SVG
    const svg = d3.select(svgRef.current)
      .attr('width', width + margin.left + margin.right)
      .attr('height', height + margin.top + margin.bottom);

    const g = svg.append('g')
      .attr('transform', `translate(${margin.left},${margin.top})`);

    // Create hierarchy
    const root = d3.hierarchy(currentRoot)
      .sum(d => d.value || 0)
      .sort((a, b) => (b.value || 0) - (a.value || 0));

    // Create treemap layout
    const treemap = d3.treemap<TreemapNode>()
      .size([width, height])
      .padding(2)
      .round(true);

    // Set tile method
    switch (tileType) {
      case 'binary':
        treemap.tile(d3.treemapBinary);
        break;
      case 'dice':
        treemap.tile(d3.treemapDice);
        break;
      case 'slice':
        treemap.tile(d3.treemapSlice);
        break;
      case 'sliceDice':
        treemap.tile(d3.treemapSliceDice);
        break;
      case 'resquarify':
        treemap.tile(d3.treemapResquarify);
        break;
      default:
        treemap.tile(d3.treemapSquarify);
    }

    // Generate layout
    treemap(root);

    // Color scale
    const color = colorScale || d3.scaleOrdinal(theme.colors || d3.schemeCategory10);

    // Filter nodes by max depth
    const nodes = root.descendants().filter(d => d.depth <= maxDepth);

    // Create groups for each node
    const node = g.selectAll('g')
      .data(nodes)
      .enter().append('g')
      .attr('transform', (d: any) => `translate(${d.x0},${d.y0})`);

    // Add rectangles
    node.append('rect')
      .attr('id', (d: any) => d.data.id)
      .attr('width', (d: any) => d.x1 - d.x0)
      .attr('height', (d: any) => d.y1 - d.y0)
      .attr('fill', (d: any) => {
        if (d.data.color) return d.data.color;
        // Use parent's name for consistent coloring
        let parent = d;
        while (parent.depth > 1) parent = parent.parent!;
        return color(parent.data.name);
      })
      .attr('stroke', '#fff')
      .attr('stroke-width', 1)
      .attr('class', 'treemap-node')
      .style('cursor', (d: any) => d.children && enableZoom ? 'pointer' : 'default')
      .on('click', function(event: any, d: any) {
        event.stopPropagation();
        if (enableZoom && d.children) {
          // Update breadcrumb
          const path: TreemapNode[] = [];
          let current = d;
          while (current) {
            path.unshift(current.data);
            current = current.parent;
          }
          setBreadcrumb(path);
          setCurrentRoot(d.data);
        }
        if (onNodeClick) {
          onNodeClick(d.data);
        }
      })
      .on('mouseenter', function(event: any, d: any) {
        setHoveredNode(d.data.id);
        if (onNodeHover) {
          onNodeHover(d.data);
        }
        // Highlight effect
        d3.select(this).style('opacity', 0.8);
      })
      .on('mouseleave', function(event: any, d: any) {
        setHoveredNode(null);
        if (onNodeHover) {
          onNodeHover(null);
        }
        // Remove highlight
        d3.select(this).style('opacity', 1);
      });

    // Add selected state
    if (selectedNodeId) {
      node.selectAll('rect')
        .filter((d: any) => d.data.id === selectedNodeId)
        .style('stroke', '#3b82f6')
        .style('stroke-width', 3);
    }

    // Add labels
    node.append('text')
      .attr('x', 4)
      .attr('y', 16)
      .text((d: any) => {
        const width = d.x1 - d.x0;
        const height = d.y1 - d.y0;
        // Only show text if node is large enough
        if (width > 50 && height > 20) {
          return d.data.name;
        }
        return '';
      })
      .attr('font-size', '12px')
      .attr('fill', 'white')
      .attr('font-weight', '500')
      .style('text-shadow', '0 1px 2px rgba(0,0,0,0.5)')
      .style('pointer-events', 'none');

    // Add value labels
    node.append('text')
      .attr('x', 4)
      .attr('y', 32)
      .text((d: any) => {
        const width = d.x1 - d.x0;
        const height = d.y1 - d.y0;
        // Only show text if node is large enough
        if (width > 60 && height > 35) {
          return valueFormat ? valueFormat(d.value) : formatValue(d.value);
        }
        return '';
      })
      .attr('font-size', '10px')
      .attr('fill', 'white')
      .attr('opacity', 0.8)
      .style('pointer-events', 'none');

    // Add tooltips
    node.append('title')
      .text((d: any) => {
        const value = valueFormat ? valueFormat(d.value) : formatValue(d.value);
        const path = d.ancestors().reverse().map((n: any) => n.data.name).join(' / ');
        return `${path}\n${value}`;
      });

  }, [currentRoot, width, height, tileType, selectedNodeId, maxDepth, colorScale, enableZoom, valueFormat]);

  // Calculate statistics
  const stats = useMemo(() => {
    const allNodes: TreemapNode[] = [];
    const collectNodes = (node: TreemapNode) => {
      allNodes.push(node);
      if (node.children) {
        node.children.forEach(collectNodes);
      }
    };
    collectNodes(currentRoot);
    
    const leafNodes = allNodes.filter(n => !n.children || n.children.length === 0);
    const totalValue = leafNodes.reduce((sum, node) => sum + (node.value || 0), 0);
    
    return {
      totalValue,
      nodeCount: allNodes.length,
      leafCount: leafNodes.length,
    };
  }, [currentRoot]);

  // Handle breadcrumb navigation
  const handleBreadcrumbClick = useCallback((index: number) => {
    if (index === 0) {
      setCurrentRoot(data);
      setBreadcrumb([]);
    } else {
      const newBreadcrumb = breadcrumb.slice(0, index + 1);
      setBreadcrumb(newBreadcrumb);
      setCurrentRoot(newBreadcrumb[newBreadcrumb.length - 1]);
    }
  }, [data, breadcrumb]);

  // Export functionality
  const exportChart = useCallback((format: 'png' | 'svg' | 'json') => {
    if (format === 'json') {
      const dataStr = JSON.stringify(currentRoot, null, 2);
      const blob = new Blob([dataStr], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `treemap-${Date.now()}.json`;
      link.click();
      URL.revokeObjectURL(url);
    } else if (format === 'svg' && svgRef.current) {
      const svgData = new XMLSerializer().serializeToString(svgRef.current);
      const blob = new Blob([svgData], { type: 'image/svg+xml' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `treemap-${Date.now()}.svg`;
      link.click();
      URL.revokeObjectURL(url);
    }
  }, [currentRoot]);

  return (
    <div className="interactive-treemap interactive-chart">
      <div className="chart-header">
        <div className="chart-info">
          <span className="total-value">Total: {formatValue(stats.totalValue)}</span>
          <span className="segment-count">{stats.leafCount} items</span>
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

      {/* Breadcrumb navigation */}
      {breadcrumb.length > 0 && (
        <div className="treemap-breadcrumb">
          <span 
            className="breadcrumb-item"
            onClick={() => handleBreadcrumbClick(0)}
          >
            Root
          </span>
          {breadcrumb.map((node, index) => (
            <React.Fragment key={node.id}>
              <span className="breadcrumb-separator">›</span>
              <span 
                className="breadcrumb-item"
                onClick={() => handleBreadcrumbClick(index)}
              >
                {node.name}
              </span>
            </React.Fragment>
          ))}
        </div>
      )}

      <div className="treemap-container">
        <svg ref={svgRef}></svg>
      </div>

      {hoveredNode && (
        <div className="treemap-tooltip">
          {/* Tooltip content handled by title element */}
        </div>
      )}
    </div>
  );
}