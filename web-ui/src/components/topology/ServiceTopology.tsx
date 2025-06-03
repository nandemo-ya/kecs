import React, { useState, useCallback, useEffect, useMemo } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  addEdge,
  Connection,
  ConnectionMode,
  Panel,
  ReactFlowProvider,
  useReactFlow,
  BackgroundVariant,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { ServiceNode } from './ServiceNode';
import { ServiceEdge, EdgeMarkers } from './ServiceEdge';
import { TopologyControls } from './TopologyControls';
import { ServiceDetailsPanel } from './ServiceDetailsPanel';
import { useTopologyLayout } from '../../hooks/useTopologyLayout';
import {
  ServiceFlowNode,
  ServiceFlowEdge,
  ServiceEdgeData,
  TopologyViewOptions,
  ServiceDetails,
  LayoutAlgorithm,
} from '../../types/topology';
import './ServiceTopology.css';

// Define custom node and edge types
const nodeTypes = {
  service: ServiceNode,
};

const edgeTypes = {
  service: ServiceEdge,
};

// Default view options
const defaultViewOptions: TopologyViewOptions = {
  layout: 'hierarchical',
  showHealthStatus: true,
  showTaskCounts: true,
  showConnections: true,
  showTrafficFlow: true,
  showLatency: true,
  autoRefresh: false,
  refreshInterval: 30000,
  filterByCluster: undefined,
  filterByServiceType: undefined,
  filterByHealth: undefined,
};

interface ServiceTopologyProps {
  initialNodes?: ServiceFlowNode[];
  initialEdges?: ServiceFlowEdge[];
  onNodeClick?: (node: ServiceFlowNode) => void;
  onEdgeClick?: (edge: ServiceFlowEdge) => void;
}

function ServiceTopologyContent({ 
  initialNodes = [], 
  initialEdges = [],
  onNodeClick,
  onEdgeClick,
}: ServiceTopologyProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [viewOptions, setViewOptions] = useState<TopologyViewOptions>(defaultViewOptions);
  const [selectedNode, setSelectedNode] = useState<ServiceFlowNode | null>(null);
  const [selectedEdge, setSelectedEdge] = useState<ServiceFlowEdge | null>(null);
  const [serviceDetails, setServiceDetails] = useState<ServiceDetails | null>(null);
  const [isDetailsPanelOpen, setIsDetailsPanelOpen] = useState(false);

  const { fitView, zoomIn, zoomOut, setViewport } = useReactFlow();
  const { applyLayout } = useTopologyLayout();

  // Apply layout when nodes or layout algorithm changes
  useEffect(() => {
    if (nodes.length > 0) {
      const layoutedNodes = applyLayout(nodes, edges, viewOptions.layout);
      setNodes(layoutedNodes);
      // Fit view after layout with padding
      setTimeout(() => fitView({ padding: 0.2 }), 100);
    }
  }, [viewOptions.layout]);

  // Filter nodes based on view options
  const filteredNodes = useMemo(() => {
    return nodes.filter(node => {
      if (viewOptions.filterByCluster && node.data.clusterName !== viewOptions.filterByCluster) {
        return false;
      }
      if (viewOptions.filterByServiceType && !viewOptions.filterByServiceType.includes(node.data.serviceType)) {
        return false;
      }
      if (viewOptions.filterByHealth && !viewOptions.filterByHealth.includes(node.data.healthStatus)) {
        return false;
      }
      return true;
    });
  }, [nodes, viewOptions]);

  // Filter edges based on filtered nodes
  const filteredEdges = useMemo(() => {
    const nodeIds = new Set(filteredNodes.map(n => n.id));
    return edges.filter(edge => 
      nodeIds.has(edge.source) && nodeIds.has(edge.target)
    );
  }, [edges, filteredNodes]);

  // Handle node click
  const handleNodeClick = useCallback((event: React.MouseEvent, node: Node) => {
    setSelectedNode(node as ServiceFlowNode);
    setSelectedEdge(null);
    setIsDetailsPanelOpen(true);
    
    // Highlight connected nodes and edges
    const connectedEdges = edges.filter(e => e.source === node.id || e.target === node.id);
    const connectedNodeIds = new Set<string>();
    connectedEdges.forEach(edge => {
      connectedNodeIds.add(edge.source);
      connectedNodeIds.add(edge.target);
    });

    // Update node highlighting
    setNodes(nodes => nodes.map(n => ({
      ...n,
      data: {
        ...n.data,
        isHighlighted: connectedNodeIds.has(n.id) && n.id !== node.id,
        isSelected: n.id === node.id,
      },
    })));

    // Update edge highlighting
    setEdges(edges => edges.map(e => ({
      ...e,
      data: {
        ...e.data,
        isHighlighted: connectedEdges.some(ce => ce.id === e.id),
      } as ServiceEdgeData,
    } as ServiceFlowEdge)));

    onNodeClick?.(node as ServiceFlowNode);
  }, [edges, onNodeClick]);

  // Handle edge click
  const handleEdgeClick = useCallback((event: React.MouseEvent, edge: Edge) => {
    setSelectedEdge(edge as ServiceFlowEdge);
    setSelectedNode(null);
    onEdgeClick?.(edge as ServiceFlowEdge);
  }, [onEdgeClick]);

  // Handle connection creation
  const onConnect = useCallback((params: Connection) => {
    const newEdge = {
      ...params,
      type: 'service',
      data: {
        connectionType: 'custom',
        trafficFlow: 'unidirectional',
      },
    };
    setEdges(eds => addEdge(newEdge, eds));
  }, [setEdges]);

  // Handle background click to deselect
  const handlePaneClick = useCallback(() => {
    setSelectedNode(null);
    setSelectedEdge(null);
    setIsDetailsPanelOpen(false);
    
    // Clear highlighting
    setNodes(nodes => nodes.map(n => ({
      ...n,
      data: {
        ...n.data,
        isHighlighted: false,
        isSelected: false,
      },
    })));
    
    setEdges(edges => edges.map(e => ({
      ...e,
      data: {
        ...e.data,
        isHighlighted: false,
      } as ServiceEdgeData,
    } as ServiceFlowEdge)));
  }, []);

  // Handle view option changes
  const handleViewOptionsChange = useCallback((newOptions: Partial<TopologyViewOptions>) => {
    setViewOptions(prev => ({ ...prev, ...newOptions }));
  }, []);

  // Handle layout change
  const handleLayoutChange = useCallback((layout: LayoutAlgorithm) => {
    handleViewOptionsChange({ layout });
  }, [handleViewOptionsChange]);

  // Auto-refresh functionality
  useEffect(() => {
    if (viewOptions.autoRefresh) {
      const interval = setInterval(() => {
        // Refresh data here
        console.log('Refreshing topology data...');
      }, viewOptions.refreshInterval);
      return () => clearInterval(interval);
    }
  }, [viewOptions.autoRefresh, viewOptions.refreshInterval]);

  return (
    <div className="service-topology-container">
      <ReactFlow
        nodes={filteredNodes}
        edges={filteredEdges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={handleNodeClick}
        onEdgeClick={handleEdgeClick}
        onPaneClick={handlePaneClick}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        connectionMode={ConnectionMode.Loose}
        fitView
        className="service-topology-flow"
      >
        <EdgeMarkers />
        
        <Background 
          variant={BackgroundVariant.Dots} 
          gap={20} 
          size={1} 
          color="#e5e7eb"
        />
        
        <Controls 
          showZoom
          showFitView
          showInteractive
          position="bottom-left"
        />
        
        <MiniMap 
          nodeColor={(node) => {
            const serviceNode = node as ServiceFlowNode;
            switch (serviceNode.data.healthStatus) {
              case 'healthy':
                return '#10b981';
              case 'unhealthy':
                return '#ef4444';
              case 'degraded':
                return '#f59e0b';
              default:
                return '#6b7280';
            }
          }}
          maskColor="rgba(0, 0, 0, 0.1)"
          position="bottom-right"
        />

        <Panel position="top-left" className="topology-panel">
          <TopologyControls
            viewOptions={viewOptions}
            onViewOptionsChange={handleViewOptionsChange}
            onLayoutChange={handleLayoutChange}
            onRefresh={() => console.log('Refresh topology')}
            onExport={() => console.log('Export topology')}
          />
        </Panel>

        {/* Metrics Summary */}
        <Panel position="top-right" className="metrics-panel">
          <div className="topology-metrics">
            <div className="metric">
              <span className="metric-value">{filteredNodes.length}</span>
              <span className="metric-label">Services</span>
            </div>
            <div className="metric">
              <span className="metric-value">{filteredEdges.length}</span>
              <span className="metric-label">Connections</span>
            </div>
            <div className="metric">
              <span className="metric-value healthy">
                {filteredNodes.filter(n => n.data.healthStatus === 'healthy').length}
              </span>
              <span className="metric-label">Healthy</span>
            </div>
            <div className="metric">
              <span className="metric-value unhealthy">
                {filteredNodes.filter(n => n.data.healthStatus === 'unhealthy').length}
              </span>
              <span className="metric-label">Unhealthy</span>
            </div>
          </div>
        </Panel>
      </ReactFlow>

      {/* Service Details Panel */}
      {isDetailsPanelOpen && selectedNode && (
        <ServiceDetailsPanel
          service={selectedNode.data}
          serviceDetails={serviceDetails}
          onClose={() => setIsDetailsPanelOpen(false)}
          onRefresh={() => console.log('Refresh service details')}
        />
      )}
    </div>
  );
}

// Main component with ReactFlowProvider
export function ServiceTopology(props: ServiceTopologyProps) {
  return (
    <ReactFlowProvider>
      <ServiceTopologyContent {...props} />
    </ReactFlowProvider>
  );
}