import React, { useEffect, useRef, useState, useCallback } from 'react';
import * as d3 from 'd3';
import { sankey, sankeyLinkHorizontal, sankeyLeft, sankeyRight, sankeyCenter, sankeyJustify, SankeyNode as D3SankeyNode, SankeyLink as D3SankeyLink } from 'd3-sankey';
import {
  SankeyNode,
  SankeyLink,
  InteractiveSankeyProps,
  DEFAULT_CHART_CONFIG,
  DEFAULT_CHART_THEME,
  formatValue,
} from '../../types/interactiveCharts';
import './InteractiveCharts.css';

type D3Node = D3SankeyNode<SankeyNode, SankeyLink>;
type D3Link = D3SankeyLink<SankeyNode, SankeyLink>;

export function InteractiveSankey({
  nodes,
  links,
  config = DEFAULT_CHART_CONFIG,
  nodeWidth = 15,
  nodePadding = 10,
  nodeAlign = 'justify',
  onNodeClick,
  onLinkClick,
  onNodeHover,
  onLinkHover,
  highlightConnected = true,
  enableNodeDragging = true,
}: InteractiveSankeyProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [hoveredNode, setHoveredNode] = useState<string | null>(null);
  const [hoveredLink, setHoveredLink] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<string | null>(null);

  const theme = DEFAULT_CHART_THEME;
  const margin = config.margin || DEFAULT_CHART_CONFIG.margin!;
  const width = (config.width || 800) - margin.left - margin.right;
  const height = (config.height || 600) - margin.top - margin.bottom;

  useEffect(() => {
    if (!svgRef.current) return;

    // Clear previous content
    d3.select(svgRef.current).selectAll('*').remove();

    // Create main SVG group
    const svg = d3.select(svgRef.current)
      .attr('width', width + margin.left + margin.right)
      .attr('height', height + margin.top + margin.bottom);

    const g = svg.append('g')
      .attr('transform', `translate(${margin.left},${margin.top})`);

    // Create sankey generator
    const sankeyGenerator = sankey<SankeyNode, SankeyLink>()
      .nodeId((d: any) => d.id)
      .nodeWidth(nodeWidth)
      .nodePadding(nodePadding)
      .extent([[0, 0], [width, height]]);

    // Set node alignment
    switch (nodeAlign) {
      case 'left':
        sankeyGenerator.nodeAlign(sankeyLeft);
        break;
      case 'right':
        sankeyGenerator.nodeAlign(sankeyRight);
        break;
      case 'center':
        sankeyGenerator.nodeAlign(sankeyCenter);
        break;
      default:
        sankeyGenerator.nodeAlign(sankeyJustify);
    }

    // Generate layout
    const graph = sankeyGenerator({
      nodes: nodes.map(d => ({ ...d })),
      links: links.map(d => ({ ...d }))
    });

    // Define gradients for links
    const defs = svg.append('defs');
    graph.links.forEach((link: any, i: number) => {
      const gradient = defs.append('linearGradient')
        .attr('id', `gradient-${i}`)
        .attr('gradientUnits', 'userSpaceOnUse')
        .attr('x1', link.source.x1)
        .attr('y1', link.source.y0)
        .attr('x2', link.target.x0)
        .attr('y2', link.target.y0);

      gradient.append('stop')
        .attr('offset', '0%')
        .attr('stop-color', link.source.color || theme.colors?.[0] || '#3b82f6')
        .attr('stop-opacity', 0.5);

      gradient.append('stop')
        .attr('offset', '100%')
        .attr('stop-color', link.target.color || theme.colors?.[1] || '#10b981')
        .attr('stop-opacity', 0.5);
    });

    // Draw links
    const linkG = g.append('g')
      .attr('class', 'links')
      .attr('fill', 'none');

    const linkPaths = linkG.selectAll('path')
      .data(graph.links)
      .enter().append('path')
      .attr('d', sankeyLinkHorizontal())
      .attr('stroke', (d: any, i: number) => `url(#gradient-${i})`)
      .attr('stroke-width', (d: any) => Math.max(1, d.width))
      .attr('class', 'sankey-link')
      .style('cursor', 'pointer')
      .on('mouseenter', function(event: any, d: any) {
        handleLinkHover(d.source.id, d.target.id, true);
      })
      .on('mouseleave', function() {
        handleLinkHover(null, null, false);
      })
      .on('click', function(event: any, d: any) {
        if (onLinkClick) {
          const link = links.find(l => l.source === d.source.id && l.target === d.target.id);
          if (link) onLinkClick(link);
        }
      });

    // Add link titles
    linkPaths.append('title')
      .text((d: any) => `${d.source.name} → ${d.target.name}\n${formatValue(d.value)}`);

    // Draw nodes
    const nodeG = g.append('g')
      .attr('class', 'nodes');

    const nodeRects = nodeG.selectAll('rect')
      .data(graph.nodes)
      .enter().append('g')
      .attr('class', 'sankey-node-group');

    const rects = nodeRects.append('rect')
      .attr('x', (d: any) => d.x0)
      .attr('y', (d: any) => d.y0)
      .attr('height', (d: any) => d.y1 - d.y0)
      .attr('width', (d: any) => d.x1 - d.x0)
      .attr('fill', (d: any) => d.color || theme.colors?.[d.index % theme.colors.length] || '#3b82f6')
      .attr('class', 'sankey-node')
      .style('cursor', enableNodeDragging ? 'move' : 'pointer')
      .on('mouseenter', function(event: any, d: any) {
        handleNodeHover(d.id, true);
      })
      .on('mouseleave', function() {
        handleNodeHover(null, false);
      })
      .on('click', function(event: any, d: any) {
        setSelectedNode(d.id);
        if (onNodeClick) {
          const node = nodes.find(n => n.id === d.id);
          if (node) onNodeClick(node);
        }
      });

    // Add node labels
    const labels = nodeRects.append('text')
      .attr('x', (d: any) => d.x0 < width / 2 ? d.x1 + 6 : d.x0 - 6)
      .attr('y', (d: any) => (d.y1 + d.y0) / 2)
      .attr('dy', '0.35em')
      .attr('text-anchor', (d: any) => d.x0 < width / 2 ? 'start' : 'end')
      .text((d: any) => d.name)
      .style('font-size', '12px')
      .style('font-weight', '500')
      .style('fill', theme.textColor || '#374151');

    // Add node values
    nodeRects.append('text')
      .attr('x', (d: any) => d.x0 < width / 2 ? d.x1 + 6 : d.x0 - 6)
      .attr('y', (d: any) => (d.y1 + d.y0) / 2)
      .attr('dy', '1.5em')
      .attr('text-anchor', (d: any) => d.x0 < width / 2 ? 'start' : 'end')
      .text((d: any) => formatValue(d.value))
      .style('font-size', '10px')
      .style('fill', '#6b7280');

    // Enable node dragging
    if (enableNodeDragging) {
      const drag = d3.drag<any, any>()
        .on('start', function(event: any, d: any) {
          d3.select(this).raise();
        })
        .on('drag', function(event: any, d: any) {
          const rectY = event.y;
          d.y0 = rectY;
          d.y1 = rectY + (d.y1 - d.y0);
          
          // Update node position
          d3.select(this).select('rect')
            .attr('y', d.y0);
          
          // Update links
          sankeyGenerator.update(graph);
          linkPaths.attr('d', sankeyLinkHorizontal());
        });

      nodeRects.call(drag);
    }

    // Handle highlighting
    const handleNodeHover = (nodeId: string | null, isHover: boolean) => {
      setHoveredNode(isHover ? nodeId : null);
      if (onNodeHover) {
        const node = isHover ? nodes.find(n => n.id === nodeId) : null;
        if (node || !isHover) onNodeHover(node || null);
      }

      if (highlightConnected && nodeId) {
        // Highlight connected links
        linkPaths.style('opacity', (d: any) => 
          isHover && (d.source.id === nodeId || d.target.id === nodeId) ? 1 : isHover ? 0.2 : 1
        );
        
        // Highlight connected nodes
        rects.style('opacity', (d: any) => {
          if (!isHover) return 1;
          if (d.id === nodeId) return 1;
          const connected = graph.links.some((l: any) => 
            (l.source.id === nodeId && l.target.id === d.id) ||
            (l.target.id === nodeId && l.source.id === d.id)
          );
          return connected ? 1 : 0.2;
        });
      }
    };

    const handleLinkHover = (sourceId: string | null, targetId: string | null, isHover: boolean) => {
      const linkKey = sourceId && targetId ? `${sourceId}-${targetId}` : null;
      setHoveredLink(isHover ? linkKey : null);
      
      if (onLinkHover) {
        const link = isHover && sourceId && targetId 
          ? links.find(l => l.source === sourceId && l.target === targetId)
          : null;
        if (link || !isHover) onLinkHover(link || null);
      }

      if (highlightConnected && linkKey) {
        linkPaths.style('opacity', (d: any) => 
          isHover && d.source.id === sourceId && d.target.id === targetId ? 1 : isHover ? 0.2 : 1
        );
        
        rects.style('opacity', (d: any) => 
          isHover && (d.id === sourceId || d.id === targetId) ? 1 : isHover ? 0.2 : 1
        );
      }
    };

  }, [nodes, links, width, height, nodeWidth, nodePadding, nodeAlign, highlightConnected, enableNodeDragging]);

  const exportChart = useCallback((format: 'png' | 'svg' | 'json') => {
    if (format === 'json') {
      const data = { nodes, links };
      const dataStr = JSON.stringify(data, null, 2);
      const blob = new Blob([dataStr], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `sankey-diagram-${Date.now()}.json`;
      link.click();
      URL.revokeObjectURL(url);
    } else if (format === 'svg' && svgRef.current) {
      const svgData = new XMLSerializer().serializeToString(svgRef.current);
      const blob = new Blob([svgData], { type: 'image/svg+xml' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `sankey-diagram-${Date.now()}.svg`;
      link.click();
      URL.revokeObjectURL(url);
    }
  }, [nodes, links]);

  const totalFlow = links.reduce((sum, link) => sum + link.value, 0);

  return (
    <div className="interactive-sankey interactive-chart">
      <div className="chart-header">
        <div className="chart-info">
          <span className="total-value">Total Flow: {formatValue(totalFlow)}</span>
          <span className="segment-count">{nodes.length} nodes, {links.length} flows</span>
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

      <div className="sankey-container">
        <svg ref={svgRef}></svg>
      </div>

      {hoveredNode && (
        <div className="sankey-tooltip">
          <p>Node: {nodes.find(n => n.id === hoveredNode)?.name}</p>
        </div>
      )}
    </div>
  );
}