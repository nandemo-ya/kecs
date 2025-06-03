import { useCallback } from 'react';
import { ServiceFlowNode, ServiceFlowEdge, LayoutAlgorithm } from '../types/topology';

interface LayoutPosition {
  x: number;
  y: number;
}

export function useTopologyLayout() {
  const applyLayout = useCallback((
    nodes: ServiceFlowNode[],
    edges: ServiceFlowEdge[],
    algorithm: LayoutAlgorithm
  ): ServiceFlowNode[] => {
    if (nodes.length === 0) return nodes;

    const positions = calculateLayout(nodes, edges, algorithm);
    
    return nodes.map((node, index) => ({
      ...node,
      position: positions[index] || { x: 0, y: 0 },
    }));
  }, []);

  return { applyLayout };
}

function calculateLayout(
  nodes: ServiceFlowNode[],
  edges: ServiceFlowEdge[],
  algorithm: LayoutAlgorithm
): LayoutPosition[] {
  switch (algorithm) {
    case 'hierarchical':
      return calculateHierarchicalLayout(nodes, edges);
    case 'force':
      return calculateForceLayout(nodes, edges);
    case 'circular':
      return calculateCircularLayout(nodes);
    case 'grid':
      return calculateGridLayout(nodes);
    case 'manual':
      return nodes.map(node => node.position);
    default:
      return calculateHierarchicalLayout(nodes, edges);
  }
}

function calculateHierarchicalLayout(
  nodes: ServiceFlowNode[],
  edges: ServiceFlowEdge[]
): LayoutPosition[] {
  // Build adjacency list to determine hierarchy
  const adjacencyList = new Map<string, string[]>();
  const incomingCount = new Map<string, number>();
  
  // Initialize
  nodes.forEach(node => {
    adjacencyList.set(node.id, []);
    incomingCount.set(node.id, 0);
  });

  // Build graph
  edges.forEach(edge => {
    const sources = adjacencyList.get(edge.source) || [];
    sources.push(edge.target);
    adjacencyList.set(edge.source, sources);
    
    const incoming = incomingCount.get(edge.target) || 0;
    incomingCount.set(edge.target, incoming + 1);
  });

  // Find root nodes (no incoming edges)
  const rootNodes = nodes.filter(node => (incomingCount.get(node.id) || 0) === 0);
  
  // If no root nodes, use first node as root
  if (rootNodes.length === 0 && nodes.length > 0) {
    rootNodes.push(nodes[0]);
  }

  // Assign levels using BFS
  const levels = new Map<string, number>();
  const queue: Array<{ nodeId: string; level: number }> = [];
  
  rootNodes.forEach(node => {
    levels.set(node.id, 0);
    queue.push({ nodeId: node.id, level: 0 });
  });

  let maxLevel = 0;
  while (queue.length > 0) {
    const { nodeId, level } = queue.shift()!;
    const children = adjacencyList.get(nodeId) || [];
    
    children.forEach(childId => {
      const currentLevel = levels.get(childId);
      const newLevel = level + 1;
      
      if (currentLevel === undefined || newLevel > currentLevel) {
        levels.set(childId, newLevel);
        maxLevel = Math.max(maxLevel, newLevel);
        queue.push({ nodeId: childId, level: newLevel });
      }
    });
  }

  // Assign nodes to levels
  const levelNodes = new Map<number, string[]>();
  for (let i = 0; i <= maxLevel; i++) {
    levelNodes.set(i, []);
  }

  nodes.forEach(node => {
    const level = levels.get(node.id) || 0;
    const nodesAtLevel = levelNodes.get(level) || [];
    nodesAtLevel.push(node.id);
    levelNodes.set(level, nodesAtLevel);
  });

  // Calculate positions
  const positions: LayoutPosition[] = [];
  const nodeSpacing = 200;
  const levelSpacing = 150;
  
  nodes.forEach(node => {
    const level = levels.get(node.id) || 0;
    const nodesAtLevel = levelNodes.get(level) || [];
    const indexAtLevel = nodesAtLevel.indexOf(node.id);
    
    const x = indexAtLevel * nodeSpacing - (nodesAtLevel.length - 1) * nodeSpacing / 2;
    const y = level * levelSpacing;
    
    positions.push({ x, y });
  });

  return positions;
}

function calculateForceLayout(
  nodes: ServiceFlowNode[],
  edges: ServiceFlowEdge[]
): LayoutPosition[] {
  // Simple force-directed layout simulation
  const positions = nodes.map((_, index) => ({
    x: Math.random() * 800 - 400,
    y: Math.random() * 600 - 300,
  }));

  const iterations = 100;
  const k = Math.sqrt((800 * 600) / nodes.length); // Ideal distance
  const dt = 0.1;
  
  for (let iter = 0; iter < iterations; iter++) {
    const forces = positions.map(() => ({ x: 0, y: 0 }));
    
    // Repulsive forces between all nodes
    for (let i = 0; i < nodes.length; i++) {
      for (let j = i + 1; j < nodes.length; j++) {
        const dx = positions[i].x - positions[j].x;
        const dy = positions[i].y - positions[j].y;
        const distance = Math.sqrt(dx * dx + dy * dy) || 1;
        
        const force = (k * k) / distance;
        const fx = (dx / distance) * force;
        const fy = (dy / distance) * force;
        
        forces[i].x += fx;
        forces[i].y += fy;
        forces[j].x -= fx;
        forces[j].y -= fy;
      }
    }
    
    // Attractive forces for connected nodes
    edges.forEach(edge => {
      const sourceIndex = nodes.findIndex(n => n.id === edge.source);
      const targetIndex = nodes.findIndex(n => n.id === edge.target);
      
      if (sourceIndex >= 0 && targetIndex >= 0) {
        const dx = positions[targetIndex].x - positions[sourceIndex].x;
        const dy = positions[targetIndex].y - positions[sourceIndex].y;
        const distance = Math.sqrt(dx * dx + dy * dy) || 1;
        
        const force = (distance * distance) / k;
        const fx = (dx / distance) * force;
        const fy = (dy / distance) * force;
        
        forces[sourceIndex].x += fx;
        forces[sourceIndex].y += fy;
        forces[targetIndex].x -= fx;
        forces[targetIndex].y -= fy;
      }
    });
    
    // Apply forces
    positions.forEach((pos, i) => {
      pos.x += forces[i].x * dt;
      pos.y += forces[i].y * dt;
    });
  }
  
  return positions;
}

function calculateCircularLayout(nodes: ServiceFlowNode[]): LayoutPosition[] {
  const centerX = 0;
  const centerY = 0;
  const radius = Math.max(200, nodes.length * 30);
  
  return nodes.map((_, index) => {
    const angle = (2 * Math.PI * index) / nodes.length;
    return {
      x: centerX + radius * Math.cos(angle),
      y: centerY + radius * Math.sin(angle),
    };
  });
}

function calculateGridLayout(nodes: ServiceFlowNode[]): LayoutPosition[] {
  const cols = Math.ceil(Math.sqrt(nodes.length));
  const nodeSpacing = 200;
  
  return nodes.map((_, index) => {
    const row = Math.floor(index / cols);
    const col = index % cols;
    
    return {
      x: col * nodeSpacing - (cols - 1) * nodeSpacing / 2,
      y: row * nodeSpacing,
    };
  });
}