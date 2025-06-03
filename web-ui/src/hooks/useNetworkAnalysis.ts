import { useCallback, useMemo } from 'react';
import {
  NetworkNode,
  NetworkDependency,
  DependencyPath,
  ImpactAnalysis,
  SecurityAnalysis,
  PerformanceAnalytics,
  NetworkFlowNode,
  NetworkFlowEdge,
  DependencyAnalysisOptions,
  calculateRiskScore,
} from '../types/networkDependencies';

export function useNetworkAnalysis(
  nodes: NetworkNode[],
  dependencies: NetworkDependency[]
) {
  // Build adjacency lists for efficient graph traversal
  const adjacencyLists = useMemo(() => {
    const forward = new Map<string, string[]>();
    const backward = new Map<string, string[]>();
    const dependencyMap = new Map<string, NetworkDependency>();

    nodes.forEach(node => {
      forward.set(node.id, []);
      backward.set(node.id, []);
    });

    dependencies.forEach(dep => {
      dependencyMap.set(dep.id, dep);
      
      if (dep.direction === 'outgoing' || dep.direction === 'bidirectional') {
        forward.get(dep.source)?.push(dep.target);
        backward.get(dep.target)?.push(dep.source);
      }
      
      if (dep.direction === 'incoming' || dep.direction === 'bidirectional') {
        forward.get(dep.target)?.push(dep.source);
        backward.get(dep.source)?.push(dep.target);
      }
    });

    return { forward, backward, dependencyMap };
  }, [nodes, dependencies]);

  // Find all dependency paths between two nodes
  const findDependencyPaths = useCallback((
    sourceId: string,
    targetId: string,
    options: DependencyAnalysisOptions
  ): DependencyPath[] => {
    const paths: DependencyPath[] = [];
    const visited = new Set<string>();
    const currentPath: string[] = [];

    const dfs = (currentId: string, depth: number) => {
      if (depth > options.maxDepth) return;
      if (visited.has(currentId) && currentPath.length > 0) {
        // Found circular dependency
        const circularPath = [...currentPath, currentId];
        paths.push({
          id: `circular-${paths.length}`,
          path: circularPath,
          type: 'circular',
          length: circularPath.length,
          totalLatency: calculatePathLatency(circularPath),
          reliability: calculatePathReliability(circularPath),
          bottlenecks: findBottlenecks(circularPath),
        });
        return;
      }

      if (currentId === targetId && currentPath.length > 0) {
        const path = [...currentPath, currentId];
        paths.push({
          id: `path-${paths.length}`,
          path,
          type: currentPath.length === 1 ? 'direct' : 'transitive',
          length: path.length,
          totalLatency: calculatePathLatency(path),
          reliability: calculatePathReliability(path),
          bottlenecks: findBottlenecks(path),
        });
        return;
      }

      visited.add(currentId);
      currentPath.push(currentId);

      const neighbors = adjacencyLists.forward.get(currentId) || [];
      neighbors.forEach(neighborId => {
        if (!visited.has(neighborId) || options.includeTransitive) {
          dfs(neighborId, depth + 1);
        }
      });

      currentPath.pop();
      visited.delete(currentId);
    };

    const calculatePathLatency = (path: string[]): number => {
      let totalLatency = 0;
      for (let i = 0; i < path.length - 1; i++) {
        const dep = dependencies.find(d => 
          (d.source === path[i] && d.target === path[i + 1]) ||
          (d.target === path[i] && d.source === path[i + 1])
        );
        if (dep) {
          totalLatency += dep.latency;
        }
      }
      return totalLatency;
    };

    const calculatePathReliability = (path: string[]): number => {
      let reliability = 1;
      for (let i = 0; i < path.length - 1; i++) {
        const dep = dependencies.find(d => 
          (d.source === path[i] && d.target === path[i + 1]) ||
          (d.target === path[i] && d.source === path[i + 1])
        );
        if (dep && dep.sla) {
          reliability *= dep.sla.availability / 100;
        }
      }
      return reliability * 100;
    };

    const findBottlenecks = (path: string[]): string[] => {
      const bottlenecks: string[] = [];
      path.forEach(nodeId => {
        const node = nodes.find(n => n.id === nodeId);
        if (node && node.criticality === 'critical') {
          const nodeDeps = dependencies.filter(d => d.source === nodeId || d.target === nodeId);
          const avgLatency = nodeDeps.reduce((sum, dep) => sum + dep.latency, 0) / nodeDeps.length;
          if (avgLatency > 100) { // High latency threshold
            bottlenecks.push(nodeId);
          }
        }
      });
      return bottlenecks;
    };

    dfs(sourceId, 0);
    return paths;
  }, [nodes, dependencies, adjacencyLists]);

  // Analyze impact of a node failure
  const analyzeImpact = useCallback((nodeId: string): ImpactAnalysis => {
    const directDependents = adjacencyLists.backward.get(nodeId) || [];
    const transitiveDependents = new Set<string>();
    
    // Find all transitive dependents using BFS
    const queue = [...directDependents];
    const visited = new Set([nodeId]);
    
    while (queue.length > 0) {
      const current = queue.shift()!;
      if (visited.has(current)) continue;
      
      visited.add(current);
      transitiveDependents.add(current);
      
      const dependents = adjacencyLists.backward.get(current) || [];
      queue.push(...dependents.filter(dep => !visited.has(dep)));
    }

    const node = nodes.find(n => n.id === nodeId);
    const criticalityScore = {
      critical: 4,
      high: 3,
      medium: 2,
      low: 1,
    }[node?.criticality || 'low'];

    const impactRadius = directDependents.length + (Array.from(transitiveDependents).length * 0.5);
    
    // Estimate downtime based on node criticality and dependencies
    const estimatedDowntime = Math.min(240, criticalityScore * 20 + directDependents.length * 10);
    
    // Estimate cost based on business criticality and impact radius
    const costPerMinute = {
      critical: 1000,
      high: 500,
      medium: 100,
      low: 10,
    }[node?.criticality || 'low'];
    
    const estimatedCost = estimatedDowntime * costPerMinute * (1 + impactRadius * 0.1);

    const mitigationStrategies: string[] = [];
    if (directDependents.length > 3) {
      mitigationStrategies.push('Implement circuit breaker pattern');
    }
    if (node?.criticality === 'critical') {
      mitigationStrategies.push('Deploy redundant instances');
      mitigationStrategies.push('Implement health checks and auto-recovery');
    }
    if (Array.from(transitiveDependents).length > 5) {
      mitigationStrategies.push('Consider service decomposition');
    }

    return {
      nodeId,
      directDependents,
      transitiveDependents: Array.from(transitiveDependents),
      impactRadius,
      businessCriticality: node?.criticality || 'low',
      estimatedDowntime,
      estimatedCost,
      mitigationStrategies,
    };
  }, [nodes, adjacencyLists]);

  // Analyze security vulnerabilities
  const analyzeSecurityVulnerabilities = useCallback((nodeId: string): SecurityAnalysis => {
    const node = nodes.find(n => n.id === nodeId);
    if (!node) {
      return {
        nodeId,
        vulnerabilities: [],
        compliance: [],
        riskScore: 0,
        recommendations: [],
      };
    }

    const vulnerabilities = [];
    const recommendations = [];

    // Check for unencrypted traffic
    const nodeConnections = dependencies.filter(d => d.source === nodeId || d.target === nodeId);
    const unencryptedConnections = nodeConnections.filter(d => !d.security.encrypted);
    
    if (unencryptedConnections.length > 0) {
      vulnerabilities.push({
        id: `unenc-${nodeId}`,
        type: 'unencrypted_traffic' as const,
        severity: 'high' as const,
        description: `${unencryptedConnections.length} unencrypted connections detected`,
        impact: 'Data in transit can be intercepted and read by attackers',
        mitigation: 'Enable TLS/SSL encryption for all connections',
      });
      recommendations.push('Implement TLS/SSL encryption');
    }

    // Check authentication
    if (!node.security.authentication) {
      vulnerabilities.push({
        id: `noauth-${nodeId}`,
        type: 'weak_authentication' as const,
        severity: 'critical' as const,
        description: 'No authentication mechanism configured',
        impact: 'Unauthorized access to service resources',
        mitigation: 'Implement strong authentication (OAuth2, JWT, mTLS)',
      });
      recommendations.push('Enable authentication');
    }

    // Check authorization
    if (!node.security.authorization) {
      vulnerabilities.push({
        id: `noauthz-${nodeId}`,
        type: 'privilege_escalation' as const,
        severity: 'high' as const,
        description: 'No authorization controls configured',
        impact: 'Users may access resources beyond their privileges',
        mitigation: 'Implement role-based access control (RBAC)',
      });
      recommendations.push('Implement authorization controls');
    }

    // Check firewall
    if (!node.security.firewall) {
      vulnerabilities.push({
        id: `nofw-${nodeId}`,
        type: 'open_port' as const,
        severity: 'medium' as const,
        description: 'No firewall protection detected',
        impact: 'Network-based attacks may succeed',
        mitigation: 'Configure network firewall rules',
      });
      recommendations.push('Configure firewall rules');
    }

    // External connections increase risk
    const externalConnections = nodeConnections.filter(d => 
      nodes.find(n => n.id === (d.source === nodeId ? d.target : d.source))?.type === 'external'
    );
    
    if (externalConnections.length > 0 && !node.security.encrypted) {
      vulnerabilities.push({
        id: `extconn-${nodeId}`,
        type: 'unencrypted_traffic' as const,
        severity: 'critical' as const,
        description: 'Unencrypted external connections detected',
        impact: 'External attackers can intercept sensitive data',
        mitigation: 'Use VPN or encrypted tunnels for external connections',
      });
    }

    const riskScore = calculateRiskScore(node, dependencies);

    // Compliance assessment (simplified)
    const compliance = [
      {
        standard: 'SOC2' as const,
        status: (node.security.encrypted && node.security.authentication && node.security.authorization) ? 'compliant' as const : 'non_compliant' as const,
        requirements: ['Encryption in transit', 'Access controls', 'Authentication'],
      },
      {
        standard: 'ISO27001' as const,
        status: node.security.firewall ? 'partial' as const : 'non_compliant' as const,
        requirements: ['Network security controls', 'Access management'],
      },
    ];

    return {
      nodeId,
      vulnerabilities,
      compliance,
      riskScore,
      recommendations,
    };
  }, [nodes, dependencies]);

  // Analyze performance metrics and predict bottlenecks
  const analyzePerformance = useCallback((nodeId: string): PerformanceAnalytics => {
    const nodeConnections = dependencies.filter(d => d.source === nodeId || d.target === nodeId);
    
    // Calculate average metrics
    const avgLatency = nodeConnections.reduce((sum, dep) => sum + dep.latency, 0) / nodeConnections.length || 0;
    const totalFrequency = nodeConnections.reduce((sum, dep) => sum + dep.frequency, 0);
    const avgErrorRate = nodeConnections.reduce((sum, dep) => sum + dep.errorRate, 0) / nodeConnections.length || 0;
    
    // Mock performance metrics (in real implementation, these would come from monitoring systems)
    const metrics = {
      cpu: Math.min(100, avgLatency / 2 + totalFrequency / 100),
      memory: Math.min(100, totalFrequency / 50 + avgErrorRate * 1000),
      network: Math.min(100, nodeConnections.length * 10 + avgLatency / 10),
      storage: Math.min(100, Math.random() * 40 + 30), // Mock data
    };

    // Trend analysis
    const performanceScore = (metrics.cpu + metrics.memory + metrics.network + metrics.storage) / 4;
    const trends = {
      direction: performanceScore > 70 ? 'degrading' as const : 
                 performanceScore < 40 ? 'improving' as const : 'stable' as const,
      confidence: 0.8,
    };

    // Predictions
    const bottleneckMetric = Math.max(metrics.cpu, metrics.memory, metrics.network, metrics.storage);
    const bottleneckType = bottleneckMetric === metrics.cpu ? 'CPU' :
                          bottleneckMetric === metrics.memory ? 'Memory' :
                          bottleneckMetric === metrics.network ? 'Network' : 'Storage';
    
    const timeToBottleneck = Math.max(1, 48 - (bottleneckMetric - 70) * 2);
    
    const recommendedActions = [];
    if (metrics.cpu > 80) recommendedActions.push('Scale CPU resources');
    if (metrics.memory > 80) recommendedActions.push('Increase memory allocation');
    if (metrics.network > 80) recommendedActions.push('Optimize network bandwidth');
    if (avgLatency > 100) recommendedActions.push('Optimize database queries');
    if (avgErrorRate > 0.05) recommendedActions.push('Review error handling logic');

    return {
      nodeId,
      metrics,
      trends,
      predictions: {
        nextBottleneck: `${bottleneckType} usage approaching critical levels`,
        timeToBottleneck,
        recommendedActions,
      },
    };
  }, [dependencies]);

  // Find critical paths in the network
  const findCriticalPaths = useCallback((options: DependencyAnalysisOptions): DependencyPath[] => {
    const criticalPaths: DependencyPath[] = [];
    
    // Find all paths between critical nodes
    const criticalNodes = nodes.filter(n => n.criticality === 'critical');
    
    criticalNodes.forEach(source => {
      criticalNodes.forEach(target => {
        if (source.id !== target.id) {
          const paths = findDependencyPaths(source.id, target.id, options);
          criticalPaths.push(...paths.filter(p => p.type !== 'circular'));
        }
      });
    });

    // Sort by criticality (shortest paths with highest latency are most critical)
    return criticalPaths.sort((a, b) => {
      const aCriticality = a.totalLatency / a.length;
      const bCriticality = b.totalLatency / b.length;
      return bCriticality - aCriticality;
    }).slice(0, 10); // Return top 10 critical paths
  }, [nodes, findDependencyPaths]);

  // Convert nodes and dependencies to React Flow format
  const convertToFlowData = useCallback((
    highlightedPaths: DependencyPath[] = []
  ): { nodes: NetworkFlowNode[]; edges: NetworkFlowEdge[] } => {
    const highlightedNodeIds = new Set<string>();
    const highlightedEdgeIds = new Set<string>();
    
    highlightedPaths.forEach(path => {
      path.path.forEach(nodeId => highlightedNodeIds.add(nodeId));
      for (let i = 0; i < path.path.length - 1; i++) {
        const edgeId = dependencies.find(d => 
          (d.source === path.path[i] && d.target === path.path[i + 1]) ||
          (d.target === path.path[i] && d.source === path.path[i + 1])
        )?.id;
        if (edgeId) highlightedEdgeIds.add(edgeId);
      }
    });

    const flowNodes: NetworkFlowNode[] = nodes.map((node, index) => ({
      id: node.id,
      type: 'network',
      position: { x: 0, y: 0 }, // Will be set by layout algorithm
      data: {
        ...node,
        isHighlighted: highlightedNodeIds.has(node.id),
        flowMetrics: {
          inbound: dependencies.filter(d => d.target === node.id).length,
          outbound: dependencies.filter(d => d.source === node.id).length,
          errors: dependencies.filter(d => 
            (d.source === node.id || d.target === node.id) && d.errorRate > 0.05
          ).length,
        },
      },
    }));

    const flowEdges: NetworkFlowEdge[] = dependencies.map(dep => ({
      id: dep.id,
      source: dep.source,
      target: dep.target,
      type: 'network',
      data: {
        ...dep,
        isHighlighted: highlightedEdgeIds.has(dep.id),
        isAnimated: dep.frequency > 500,
      },
    }));

    return { nodes: flowNodes, edges: flowEdges };
  }, [nodes, dependencies]);

  return {
    findDependencyPaths,
    analyzeImpact,
    analyzeSecurityVulnerabilities,
    analyzePerformance,
    findCriticalPaths,
    convertToFlowData,
    adjacencyLists,
  };
}