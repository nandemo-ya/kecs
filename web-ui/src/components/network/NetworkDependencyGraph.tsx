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
import { NetworkNode } from './NetworkNode';
import { NetworkEdge, NetworkEdgeMarkers } from './NetworkEdge';
import { NetworkAnalysisPanel } from './NetworkAnalysisPanel';
import { NetworkSecurityPanel } from './NetworkSecurityPanel';
import { DependencyPathTracer } from './DependencyPathTracer';
import { useNetworkAnalysis } from '../../hooks/useNetworkAnalysis';
import { useTopologyLayout } from '../../hooks/useTopologyLayout';
import {
  NetworkFlowNode,
  NetworkFlowEdge,
  NetworkVisualizationOptions,
  DependencyAnalysisOptions,
  DependencyPath,
  ImpactAnalysis,
  SecurityAnalysis,
  NetworkNode as NetworkNodeType,
  NetworkDependency,
} from '../../types/networkDependencies';
import './NetworkDependencyGraph.css';

// Define custom node and edge types
const nodeTypes = {
  network: NetworkNode,
};

const edgeTypes = {
  network: NetworkEdge,
};

// Default visualization options
const defaultVisualizationOptions: NetworkVisualizationOptions = {
  layout: 'hierarchical',
  showTrafficFlow: true,
  showLatency: true,
  showSecurity: true,
  showCriticality: true,
  animateTraffic: true,
  highlightPaths: true,
  groupingSetting: 'none',
  autoRefresh: false,
  refreshInterval: 30000,
};

// Default analysis options
const defaultAnalysisOptions: DependencyAnalysisOptions = {
  includeTransitive: true,
  maxDepth: 5,
  includeExternal: true,
  filterByCriticality: [],
  filterByProtocol: [],
  showSecurityVulnerabilities: true,
  showPerformanceMetrics: true,
  groupByCluster: true,
  groupByNamespace: false,
};

interface NetworkDependencyGraphProps {
  initialNodes?: NetworkNodeType[];
  initialDependencies?: NetworkDependency[];
  onNodeClick?: (node: NetworkFlowNode) => void;
  onEdgeClick?: (edge: NetworkFlowEdge) => void;
}

function NetworkDependencyGraphContent({ 
  initialNodes = [], 
  initialDependencies = [],
  onNodeClick,
  onEdgeClick,
}: NetworkDependencyGraphProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [visualizationOptions, setVisualizationOptions] = useState<NetworkVisualizationOptions>(defaultVisualizationOptions);
  const [analysisOptions, setAnalysisOptions] = useState<DependencyAnalysisOptions>(defaultAnalysisOptions);
  const [selectedNode, setSelectedNode] = useState<NetworkFlowNode | null>(null);
  const [selectedEdge, setSelectedEdge] = useState<NetworkFlowEdge | null>(null);
  const [criticalPaths, setCriticalPaths] = useState<DependencyPath[]>([]);
  const [highlightedPaths, setHighlightedPaths] = useState<DependencyPath[]>([]);
  const [showAnalysisPanel, setShowAnalysisPanel] = useState(false);
  const [showSecurityPanel, setShowSecurityPanel] = useState(false);
  const [showPathTracer, setShowPathTracer] = useState(false);
  const [impactAnalysis, setImpactAnalysis] = useState<ImpactAnalysis | null>(null);
  const [securityAnalysis, setSecurityAnalysis] = useState<SecurityAnalysis | null>(null);

  const { fitView, zoomIn, zoomOut, setViewport } = useReactFlow();
  const { applyLayout } = useTopologyLayout();
  
  const {
    findDependencyPaths,
    analyzeImpact,
    analyzeSecurityVulnerabilities,
    analyzePerformance,
    findCriticalPaths,
    convertToFlowData,
  } = useNetworkAnalysis(initialNodes, initialDependencies);

  // Convert initial data to React Flow format
  useEffect(() => {
    if (initialNodes.length > 0 || initialDependencies.length > 0) {
      const { nodes: flowNodes, edges: flowEdges } = convertToFlowData(highlightedPaths);
      setNodes(flowNodes);
      setEdges(flowEdges);
    }
  }, [initialNodes, initialDependencies, convertToFlowData, highlightedPaths]);

  // Apply layout when nodes or layout algorithm changes
  useEffect(() => {
    if (nodes.length > 0) {
      const layoutedNodes = applyLayout(nodes, edges, visualizationOptions.layout);
      setNodes(layoutedNodes);
      // Fit view after layout with padding
      setTimeout(() => fitView({ padding: 0.2 }), 100);
    }
  }, [visualizationOptions.layout, applyLayout, fitView]);

  // Find critical paths on initialization and when analysis options change
  useEffect(() => {
    if (initialNodes.length > 0) {
      const paths = findCriticalPaths(analysisOptions);
      setCriticalPaths(paths);
    }
  }, [initialNodes, initialDependencies, analysisOptions, findCriticalPaths]);

  // Filter nodes based on visualization options
  const filteredNodes = useMemo(() => {
    return nodes.filter(node => {
      if (analysisOptions.filterByCriticality.length > 0 && 
          !analysisOptions.filterByCriticality.includes(node.data.criticality)) {
        return false;
      }
      return true;
    });
  }, [nodes, analysisOptions]);

  // Filter edges based on filtered nodes and options
  const filteredEdges = useMemo(() => {
    const nodeIds = new Set(filteredNodes.map(n => n.id));
    return edges.filter(edge => {
      if (!nodeIds.has(edge.source) || !nodeIds.has(edge.target)) {
        return false;
      }
      if (analysisOptions.filterByProtocol.length > 0 && 
          !analysisOptions.filterByProtocol.includes(edge.data?.protocol || '')) {
        return false;
      }
      return true;
    });
  }, [edges, filteredNodes, analysisOptions]);

  // Handle node click
  const handleNodeClick = useCallback((event: React.MouseEvent, node: Node) => {
    const networkNode = node as NetworkFlowNode;
    setSelectedNode(networkNode);
    setSelectedEdge(null);
    
    // Perform impact analysis
    const impact = analyzeImpact(networkNode.id);
    setImpactAnalysis(impact);
    
    // Perform security analysis
    const security = analyzeSecurityVulnerabilities(networkNode.id);
    setSecurityAnalysis(security);
    
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
      },
    })));

    onNodeClick?.(networkNode);
  }, [edges, analyzeImpact, analyzeSecurityVulnerabilities, onNodeClick]);

  // Handle edge click
  const handleEdgeClick = useCallback((event: React.MouseEvent, edge: Edge) => {
    const networkEdge = edge as NetworkFlowEdge;
    setSelectedEdge(networkEdge);
    setSelectedNode(null);
    onEdgeClick?.(networkEdge);
  }, [onEdgeClick]);

  // Handle connection creation
  const onConnect = useCallback((params: Connection) => {
    const newEdge = {
      ...params,
      type: 'network',
      data: {
        id: `edge-${Date.now()}`,
        source: params.source!,
        target: params.target!,
        dependencyType: 'custom',
        protocol: 'Custom',
        port: 8080,
        direction: 'outgoing',
        strength: 'moderate',
        frequency: 100,
        latency: 50,
        errorRate: 0.01,
        bandwidth: 1024,
        security: {
          encrypted: false,
          authenticated: false,
          authorized: false,
        },
      },
    };
    setEdges(eds => addEdge(newEdge, eds));
  }, [setEdges]);

  // Handle background click to deselect
  const handlePaneClick = useCallback(() => {
    setSelectedNode(null);
    setSelectedEdge(null);
    setImpactAnalysis(null);
    setSecurityAnalysis(null);
    
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
      },
    })));
  }, []);

  // Handle path highlighting
  const handlePathHighlight = useCallback((paths: DependencyPath[]) => {
    setHighlightedPaths(paths);
  }, []);

  // Handle visualization option changes
  const handleVisualizationOptionsChange = useCallback((newOptions: Partial<NetworkVisualizationOptions>) => {
    setVisualizationOptions(prev => ({ ...prev, ...newOptions }));
  }, []);

  // Handle analysis option changes
  const handleAnalysisOptionsChange = useCallback((newOptions: Partial<DependencyAnalysisOptions>) => {
    setAnalysisOptions(prev => ({ ...prev, ...newOptions }));
  }, []);

  // Auto-refresh functionality
  useEffect(() => {
    if (visualizationOptions.autoRefresh) {
      const interval = setInterval(() => {
        // Refresh network data here
        console.log('Refreshing network dependency data...');
      }, visualizationOptions.refreshInterval);
      return () => clearInterval(interval);
    }
  }, [visualizationOptions.autoRefresh, visualizationOptions.refreshInterval]);

  return (
    <div className="network-dependency-graph-container">
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
        className="network-dependency-flow"
      >
        <NetworkEdgeMarkers />
        
        <Background 
          variant={BackgroundVariant.Dots} 
          gap={20} 
          size={1.5} 
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
            const networkNode = node as NetworkFlowNode;
            switch (networkNode.data.criticality) {
              case 'critical':
                return '#dc2626';
              case 'high':
                return '#ea580c';
              case 'medium':
                return '#ca8a04';
              case 'low':
                return '#65a30d';
              default:
                return '#6b7280';
            }
          }}
          maskColor="rgba(0, 0, 0, 0.1)"
          position="bottom-right"
        />

        {/* Network Metrics Panel */}
        <Panel position="top-right" className="network-metrics-panel">
          <div className="network-metrics">
            <div className="metric">
              <span className="metric-value">{filteredNodes.length}</span>
              <span className="metric-label">Nodes</span>
            </div>
            <div className="metric">
              <span className="metric-value">{filteredEdges.length}</span>
              <span className="metric-label">Dependencies</span>
            </div>
            <div className="metric">
              <span className="metric-value">{criticalPaths.length}</span>
              <span className="metric-label">Critical Paths</span>
            </div>
            <div className="metric">
              <span className="metric-value">
                {filteredNodes.filter(n => n.data.status === 'active').length}
              </span>
              <span className="metric-label">Active</span>
            </div>
          </div>
        </Panel>

        {/* Action Buttons Panel */}
        <Panel position="top-left" className="network-actions-panel">
          <div className="action-buttons">
            <button
              className={`action-button ${showAnalysisPanel ? 'active' : ''}`}
              onClick={() => setShowAnalysisPanel(!showAnalysisPanel)}
              title="Network Analysis"
            >
              📊
            </button>
            <button
              className={`action-button ${showSecurityPanel ? 'active' : ''}`}
              onClick={() => setShowSecurityPanel(!showSecurityPanel)}
              title="Security Analysis"
            >
              🔒
            </button>
            <button
              className={`action-button ${showPathTracer ? 'active' : ''}`}
              onClick={() => setShowPathTracer(!showPathTracer)}
              title="Path Tracing"
            >
              🔍
            </button>
          </div>
        </Panel>
      </ReactFlow>

      {/* Analysis Panels */}
      {showAnalysisPanel && (
        <NetworkAnalysisPanel
          nodes={initialNodes}
          dependencies={initialDependencies}
          criticalPaths={criticalPaths}
          impactAnalysis={impactAnalysis}
          analysisOptions={analysisOptions}
          onAnalysisOptionsChange={handleAnalysisOptionsChange}
          onClose={() => setShowAnalysisPanel(false)}
          onPathHighlight={handlePathHighlight}
        />
      )}

      {showSecurityPanel && (
        <NetworkSecurityPanel
          nodes={initialNodes}
          dependencies={initialDependencies}
          securityAnalysis={securityAnalysis}
          onClose={() => setShowSecurityPanel(false)}
          onNodeSecurityAnalysis={(nodeId) => {
            const analysis = analyzeSecurityVulnerabilities(nodeId);
            setSecurityAnalysis(analysis);
          }}
        />
      )}

      {showPathTracer && (
        <DependencyPathTracer
          nodes={initialNodes}
          dependencies={initialDependencies}
          onPathTrace={(sourceId, targetId) => {
            const paths = findDependencyPaths(sourceId, targetId, analysisOptions);
            handlePathHighlight(paths);
          }}
          onClose={() => setShowPathTracer(false)}
        />
      )}
    </div>
  );
}

// Main component with ReactFlowProvider
export function NetworkDependencyGraph(props: NetworkDependencyGraphProps) {
  return (
    <ReactFlowProvider>
      <NetworkDependencyGraphContent {...props} />
    </ReactFlowProvider>
  );
}