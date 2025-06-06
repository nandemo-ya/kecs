import React, { useMemo } from 'react';
import { ServiceTopology } from './topology/ServiceTopology';
import { ServiceFlowNode, ServiceFlowEdge, ServiceNodeData } from '../types/topology';

// Mock data for demonstration
const generateMockServices = (): ServiceNodeData[] => [
  {
    id: 'web-frontend',
    serviceName: 'web-frontend',
    clusterName: 'production',
    taskCount: 5,
    desiredCount: 5,
    runningCount: 5,
    pendingCount: 0,
    serviceType: 'web',
    healthStatus: 'healthy',
    deploymentStatus: 'active',
    launchType: 'FARGATE',
    taskDefinition: 'web-frontend:1',
    createdAt: '2025-01-01T10:00:00Z',
  },
  {
    id: 'api-gateway',
    serviceName: 'api-gateway',
    clusterName: 'production',
    taskCount: 3,
    desiredCount: 3,
    runningCount: 2,
    pendingCount: 1,
    serviceType: 'api',
    healthStatus: 'degraded',
    deploymentStatus: 'updating',
    launchType: 'FARGATE',
    taskDefinition: 'api-gateway:2',
    createdAt: '2025-01-01T10:15:00Z',
  },
  {
    id: 'user-service',
    serviceName: 'user-service',
    clusterName: 'production',
    taskCount: 2,
    desiredCount: 2,
    runningCount: 2,
    pendingCount: 0,
    serviceType: 'api',
    healthStatus: 'healthy',
    deploymentStatus: 'active',
    launchType: 'EC2',
    taskDefinition: 'user-service:1',
    createdAt: '2025-01-01T10:30:00Z',
  },
  {
    id: 'product-service',
    serviceName: 'product-service',
    clusterName: 'production',
    taskCount: 2,
    desiredCount: 2,
    runningCount: 1,
    pendingCount: 0,
    serviceType: 'api',
    healthStatus: 'unhealthy',
    deploymentStatus: 'active',
    launchType: 'FARGATE',
    taskDefinition: 'product-service:1',
    createdAt: '2025-01-01T10:45:00Z',
  },
  {
    id: 'postgres-db',
    serviceName: 'postgres-db',
    clusterName: 'production',
    taskCount: 1,
    desiredCount: 1,
    runningCount: 1,
    pendingCount: 0,
    serviceType: 'database',
    healthStatus: 'healthy',
    deploymentStatus: 'active',
    launchType: 'EC2',
    taskDefinition: 'postgres-db:1',
    createdAt: '2025-01-01T09:00:00Z',
  },
  {
    id: 'redis-cache',
    serviceName: 'redis-cache',
    clusterName: 'production',
    taskCount: 1,
    desiredCount: 1,
    runningCount: 1,
    pendingCount: 0,
    serviceType: 'cache',
    healthStatus: 'healthy',
    deploymentStatus: 'active',
    launchType: 'FARGATE',
    taskDefinition: 'redis-cache:1',
    createdAt: '2025-01-01T09:15:00Z',
  },
  {
    id: 'notification-queue',
    serviceName: 'notification-queue',
    clusterName: 'production',
    taskCount: 1,
    desiredCount: 1,
    runningCount: 1,
    pendingCount: 0,
    serviceType: 'queue',
    healthStatus: 'healthy',
    deploymentStatus: 'active',
    launchType: 'FARGATE',
    taskDefinition: 'notification-queue:1',
    createdAt: '2025-01-01T09:30:00Z',
  },
  {
    id: 'file-storage',
    serviceName: 'file-storage',
    clusterName: 'production',
    taskCount: 1,
    desiredCount: 1,
    runningCount: 1,
    pendingCount: 0,
    serviceType: 'storage',
    healthStatus: 'healthy',
    deploymentStatus: 'active',
    launchType: 'EC2',
    taskDefinition: 'file-storage:1',
    createdAt: '2025-01-01T09:45:00Z',
  },
];

const generateMockConnections = (): ServiceFlowEdge[] => [
  {
    id: 'web-api',
    source: 'web-frontend',
    target: 'api-gateway',
    type: 'service',
    data: {
      id: 'web-api',
      source: 'web-frontend',
      target: 'api-gateway',
      connectionType: 'http',
      protocol: 'HTTPS',
      port: 443,
      isSecure: true,
      trafficFlow: 'unidirectional',
      requestsPerMinute: 150,
      latencyMs: 45,
      errorRate: 0.02,
      animated: true,
      label: 'REST API',
    },
  },
  {
    id: 'api-user',
    source: 'api-gateway',
    target: 'user-service',
    type: 'service',
    data: {
      id: 'api-user',
      source: 'api-gateway',
      target: 'user-service',
      connectionType: 'grpc',
      protocol: 'gRPC',
      port: 9090,
      isSecure: true,
      trafficFlow: 'unidirectional',
      requestsPerMinute: 80,
      latencyMs: 25,
      errorRate: 0.01,
      animated: true,
    },
  },
  {
    id: 'api-product',
    source: 'api-gateway',
    target: 'product-service',
    type: 'service',
    data: {
      id: 'api-product',
      source: 'api-gateway',
      target: 'product-service',
      connectionType: 'grpc',
      protocol: 'gRPC',
      port: 9091,
      isSecure: true,
      trafficFlow: 'unidirectional',
      requestsPerMinute: 120,
      latencyMs: 150,
      errorRate: 0.08,
      animated: true,
    },
  },
  {
    id: 'user-db',
    source: 'user-service',
    target: 'postgres-db',
    type: 'service',
    data: {
      id: 'user-db',
      source: 'user-service',
      target: 'postgres-db',
      connectionType: 'database',
      protocol: 'PostgreSQL',
      port: 5432,
      isSecure: true,
      trafficFlow: 'bidirectional',
      requestsPerMinute: 200,
      latencyMs: 15,
      errorRate: 0.001,
    },
  },
  {
    id: 'product-db',
    source: 'product-service',
    target: 'postgres-db',
    type: 'service',
    data: {
      id: 'product-db',
      source: 'product-service',
      target: 'postgres-db',
      connectionType: 'database',
      protocol: 'PostgreSQL',
      port: 5432,
      isSecure: true,
      trafficFlow: 'bidirectional',
      requestsPerMinute: 180,
      latencyMs: 18,
      errorRate: 0.002,
    },
  },
  {
    id: 'user-cache',
    source: 'user-service',
    target: 'redis-cache',
    type: 'service',
    data: {
      id: 'user-cache',
      source: 'user-service',
      target: 'redis-cache',
      connectionType: 'cache',
      protocol: 'Redis',
      port: 6379,
      isSecure: false,
      trafficFlow: 'bidirectional',
      requestsPerMinute: 300,
      latencyMs: 5,
      errorRate: 0.0005,
      animated: true,
    },
  },
  {
    id: 'product-queue',
    source: 'product-service',
    target: 'notification-queue',
    type: 'service',
    data: {
      id: 'product-queue',
      source: 'product-service',
      target: 'notification-queue',
      connectionType: 'queue',
      protocol: 'AMQP',
      port: 5672,
      isSecure: true,
      trafficFlow: 'unidirectional',
      requestsPerMinute: 50,
      latencyMs: 10,
      errorRate: 0.001,
    },
  },
  {
    id: 'product-storage',
    source: 'product-service',
    target: 'file-storage',
    type: 'service',
    data: {
      id: 'product-storage',
      source: 'product-service',
      target: 'file-storage',
      connectionType: 'tcp',
      protocol: 'S3',
      port: 9000,
      isSecure: true,
      trafficFlow: 'bidirectional',
      requestsPerMinute: 30,
      latencyMs: 35,
      errorRate: 0.003,
    },
  },
];

export function ServiceTopologyDashboard() {
  const { nodes, edges } = useMemo(() => {
    const mockServices = generateMockServices();
    const mockConnections = generateMockConnections();

    const nodes: ServiceFlowNode[] = mockServices.map((service, index) => ({
      id: service.id,
      type: 'service',
      position: { x: 0, y: 0 }, // Will be set by layout algorithm
      data: service,
    }));

    const edges: ServiceFlowEdge[] = mockConnections;

    return { nodes, edges };
  }, []);

  const handleNodeClick = (node: ServiceFlowNode) => {
    console.log('Node clicked:', node);
  };

  const handleEdgeClick = (edge: ServiceFlowEdge) => {
    console.log('Edge clicked:', edge);
  };

  return (
    <main className="App-main">
      <div className="dashboard-header">
        <h2>🔗 Service Topology</h2>
        <p>Interactive visualization of service relationships and dependencies</p>
      </div>

      {/* Topology Overview Cards */}
      <div className="metric-cards-row">
        <div className="metric-card">
          <div className="metric-card-value">{nodes.length}</div>
          <div className="metric-card-label">Services</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">{edges.length}</div>
          <div className="metric-card-label">Connections</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">
            {nodes.filter(n => n.data.healthStatus === 'healthy').length}
          </div>
          <div className="metric-card-label">Healthy</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">
            {nodes.filter(n => n.data.healthStatus === 'unhealthy').length}
          </div>
          <div className="metric-card-label">Unhealthy</div>
        </div>
      </div>

      {/* Service Topology Visualization */}
      <div className="chart-container">
        <div className="chart-header">
          <h3>Service Relationship Diagram</h3>
          <p>Click nodes and edges to view details. Use controls to change layout and filters.</p>
        </div>
        <ServiceTopology
          initialNodes={nodes}
          initialEdges={edges}
          onNodeClick={handleNodeClick}
          onEdgeClick={handleEdgeClick}
        />
      </div>

      {/* Feature Information */}
      <div className="chart-container">
        <div className="chart-header">
          <h3>🔗 Service Topology Features</h3>
          <p>Interactive service relationship visualization with React Flow</p>
        </div>
        <div style={{ padding: '1rem 0' }}>
          <h4>Interactive Features:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>Drag and pan to navigate the topology</li>
            <li>Click nodes to view detailed service information</li>
            <li>Click edges to see connection details and metrics</li>
            <li>Use controls to change layout algorithms</li>
            <li>Filter services by cluster, type, or health status</li>
            <li>Auto-refresh for real-time updates</li>
            <li>Minimap for quick navigation</li>
            <li>Zoom controls for detailed inspection</li>
          </ul>
          
          <h4 style={{ marginTop: '1.5rem' }}>Service Types:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>🌐 Web - Frontend applications and web servers</li>
            <li>⚡ API - Backend APIs and microservices</li>
            <li>🗄️ Database - Data storage services</li>
            <li>💾 Cache - Redis, Memcached, and other caching layers</li>
            <li>📬 Queue - Message queues and event systems</li>
            <li>📦 Storage - File storage and object storage services</li>
          </ul>

          <h4 style={{ marginTop: '1.5rem' }}>Connection Types:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>HTTP/HTTPS - REST APIs and web traffic</li>
            <li>gRPC - High-performance RPC connections</li>
            <li>TCP - Raw TCP connections</li>
            <li>Database - SQL and NoSQL database connections</li>
            <li>Cache - Redis and Memcached protocols</li>
            <li>Queue - Message queue protocols (AMQP, SQS, etc.)</li>
          </ul>

          <h4 style={{ marginTop: '1.5rem' }}>Layout Algorithms:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>🌳 Hierarchical - Tree-like structure showing dependencies</li>
            <li>⚡ Force - Physics-based layout with attraction/repulsion</li>
            <li>⭕ Circular - Services arranged in a circle</li>
            <li>⚏ Grid - Regular grid layout for ordered view</li>
            <li>✋ Manual - Drag nodes to custom positions</li>
          </ul>
        </div>
      </div>
    </main>
  );
}