import React, { useMemo } from 'react';
import { NetworkDependencyGraph } from './network/NetworkDependencyGraph';
import { NetworkNode, NetworkDependency } from '../types/networkDependencies';

// Mock data for demonstration
const generateMockNetworkNodes = (): NetworkNode[] => [
  {
    id: 'web-frontend',
    name: 'Web Frontend',
    type: 'service',
    cluster: 'production',
    namespace: 'frontend',
    ip: '10.0.1.10',
    port: 3000,
    protocol: 'HTTP',
    status: 'active',
    criticality: 'high',
    security: {
      encrypted: true,
      authentication: true,
      authorization: true,
      firewall: true,
    },
  },
  {
    id: 'api-gateway',
    name: 'API Gateway',
    type: 'gateway',
    cluster: 'production',
    namespace: 'api',
    ip: '10.0.1.20',
    port: 8080,
    protocol: 'HTTPS',
    status: 'active',
    criticality: 'critical',
    security: {
      encrypted: true,
      authentication: true,
      authorization: true,
      firewall: true,
    },
  },
  {
    id: 'user-service',
    name: 'User Service',
    type: 'service',
    cluster: 'production',
    namespace: 'backend',
    ip: '10.0.2.10',
    port: 8081,
    protocol: 'gRPC',
    status: 'active',
    criticality: 'high',
    security: {
      encrypted: true,
      authentication: true,
      authorization: true,
      firewall: true,
    },
  },
  {
    id: 'product-service',
    name: 'Product Service',
    type: 'service',
    cluster: 'production',
    namespace: 'backend',
    ip: '10.0.2.20',
    port: 8082,
    protocol: 'gRPC',
    status: 'degraded',
    criticality: 'high',
    security: {
      encrypted: true,
      authentication: true,
      authorization: false,
      firewall: true,
    },
  },
  {
    id: 'postgres-db',
    name: 'PostgreSQL Database',
    type: 'database',
    cluster: 'production',
    namespace: 'data',
    ip: '10.0.3.10',
    port: 5432,
    protocol: 'SQL',
    status: 'active',
    criticality: 'critical',
    security: {
      encrypted: true,
      authentication: true,
      authorization: true,
      firewall: true,
    },
  },
  {
    id: 'redis-cache',
    name: 'Redis Cache',
    type: 'cache',
    cluster: 'production',
    namespace: 'data',
    ip: '10.0.3.20',
    port: 6379,
    protocol: 'Redis',
    status: 'active',
    criticality: 'medium',
    security: {
      encrypted: false,
      authentication: true,
      authorization: false,
      firewall: true,
    },
  },
  {
    id: 'message-queue',
    name: 'Message Queue',
    type: 'queue',
    cluster: 'production',
    namespace: 'messaging',
    ip: '10.0.4.10',
    port: 5672,
    protocol: 'AMQP',
    status: 'active',
    criticality: 'medium',
    security: {
      encrypted: true,
      authentication: true,
      authorization: true,
      firewall: true,
    },
  },
  {
    id: 'load-balancer',
    name: 'Load Balancer',
    type: 'load_balancer',
    cluster: 'production',
    namespace: 'ingress',
    ip: '10.0.0.10',
    port: 443,
    protocol: 'HTTPS',
    status: 'active',
    criticality: 'critical',
    security: {
      encrypted: true,
      authentication: false,
      authorization: false,
      firewall: true,
    },
  },
  {
    id: 'external-payment',
    name: 'External Payment API',
    type: 'external',
    cluster: 'external',
    namespace: 'external',
    ip: '203.0.113.10',
    port: 443,
    protocol: 'HTTPS',
    status: 'active',
    criticality: 'high',
    security: {
      encrypted: true,
      authentication: true,
      authorization: true,
      firewall: false,
    },
  },
  {
    id: 'monitoring-service',
    name: 'Monitoring Service',
    type: 'service',
    cluster: 'production',
    namespace: 'monitoring',
    ip: '10.0.5.10',
    port: 9090,
    protocol: 'HTTP',
    status: 'active',
    criticality: 'low',
    security: {
      encrypted: false,
      authentication: false,
      authorization: false,
      firewall: true,
    },
  },
];

const generateMockNetworkDependencies = (): NetworkDependency[] => [
  {
    id: 'lb-web',
    source: 'load-balancer',
    target: 'web-frontend',
    dependencyType: 'load_balance',
    protocol: 'HTTPS',
    port: 3000,
    direction: 'outgoing',
    strength: 'critical',
    frequency: 2000,
    latency: 15,
    errorRate: 0.001,
    bandwidth: 10485760, // 10MB/s
    security: {
      encrypted: true,
      authenticated: true,
      authorized: true,
    },
    sla: {
      availability: 99.9,
      responseTime: 20,
      throughput: 2000,
    },
  },
  {
    id: 'web-api',
    source: 'web-frontend',
    target: 'api-gateway',
    dependencyType: 'api_call',
    protocol: 'HTTPS',
    port: 8080,
    direction: 'outgoing',
    strength: 'critical',
    frequency: 1500,
    latency: 25,
    errorRate: 0.002,
    bandwidth: 5242880, // 5MB/s
    security: {
      encrypted: true,
      authenticated: true,
      authorized: true,
    },
    sla: {
      availability: 99.95,
      responseTime: 30,
      throughput: 1500,
    },
  },
  {
    id: 'api-user',
    source: 'api-gateway',
    target: 'user-service',
    dependencyType: 'api_call',
    protocol: 'gRPC',
    port: 8081,
    direction: 'outgoing',
    strength: 'strong',
    frequency: 800,
    latency: 20,
    errorRate: 0.01,
    bandwidth: 2097152, // 2MB/s
    security: {
      encrypted: true,
      authenticated: true,
      authorized: true,
    },
  },
  {
    id: 'api-product',
    source: 'api-gateway',
    target: 'product-service',
    dependencyType: 'api_call',
    protocol: 'gRPC',
    port: 8082,
    direction: 'outgoing',
    strength: 'strong',
    frequency: 600,
    latency: 45,
    errorRate: 0.08,
    bandwidth: 1048576, // 1MB/s
    security: {
      encrypted: true,
      authenticated: true,
      authorized: false,
    },
  },
  {
    id: 'user-db',
    source: 'user-service',
    target: 'postgres-db',
    dependencyType: 'database_query',
    protocol: 'SQL',
    port: 5432,
    direction: 'bidirectional',
    strength: 'critical',
    frequency: 1200,
    latency: 8,
    errorRate: 0.001,
    bandwidth: 524288, // 512KB/s
    security: {
      encrypted: true,
      authenticated: true,
      authorized: true,
    },
  },
  {
    id: 'product-db',
    source: 'product-service',
    target: 'postgres-db',
    dependencyType: 'database_query',
    protocol: 'SQL',
    port: 5432,
    direction: 'bidirectional',
    strength: 'critical',
    frequency: 900,
    latency: 12,
    errorRate: 0.002,
    bandwidth: 262144, // 256KB/s
    security: {
      encrypted: true,
      authenticated: true,
      authorized: true,
    },
  },
  {
    id: 'user-cache',
    source: 'user-service',
    target: 'redis-cache',
    dependencyType: 'cache_access',
    protocol: 'Redis',
    port: 6379,
    direction: 'bidirectional',
    strength: 'moderate',
    frequency: 2000,
    latency: 3,
    errorRate: 0.0005,
    bandwidth: 1048576, // 1MB/s
    security: {
      encrypted: false,
      authenticated: true,
      authorized: false,
    },
  },
  {
    id: 'product-queue',
    source: 'product-service',
    target: 'message-queue',
    dependencyType: 'event_stream',
    protocol: 'AMQP',
    port: 5672,
    direction: 'outgoing',
    strength: 'moderate',
    frequency: 300,
    latency: 5,
    errorRate: 0.001,
    bandwidth: 131072, // 128KB/s
    security: {
      encrypted: true,
      authenticated: true,
      authorized: true,
    },
  },
  {
    id: 'payment-integration',
    source: 'api-gateway',
    target: 'external-payment',
    dependencyType: 'api_call',
    protocol: 'HTTPS',
    port: 443,
    direction: 'outgoing',
    strength: 'strong',
    frequency: 150,
    latency: 120,
    errorRate: 0.02,
    bandwidth: 65536, // 64KB/s
    security: {
      encrypted: true,
      authenticated: true,
      authorized: true,
    },
    sla: {
      availability: 99.5,
      responseTime: 150,
      throughput: 150,
    },
  },
  {
    id: 'monitoring-scrape',
    source: 'monitoring-service',
    target: 'api-gateway',
    dependencyType: 'api_call',
    protocol: 'HTTP',
    port: 9090,
    direction: 'incoming',
    strength: 'weak',
    frequency: 60,
    latency: 10,
    errorRate: 0.001,
    bandwidth: 32768, // 32KB/s
    security: {
      encrypted: false,
      authenticated: false,
      authorized: false,
    },
  },
];

export function NetworkDependencyDashboard() {
  const { nodes, dependencies } = useMemo(() => {
    const mockNodes = generateMockNetworkNodes();
    const mockDependencies = generateMockNetworkDependencies();
    return { nodes: mockNodes, dependencies: mockDependencies };
  }, []);

  const networkMetrics = useMemo(() => {
    const securityVulnerabilities = nodes.reduce((count, node) => {
      let vulns = 0;
      if (!node.security.encrypted) vulns++;
      if (!node.security.authentication) vulns++;
      if (!node.security.authorization) vulns++;
      if (!node.security.firewall) vulns++;
      return count + vulns;
    }, 0);

    const averageLatency = dependencies.reduce((sum, dep) => sum + dep.latency, 0) / dependencies.length;
    const totalTrafficVolume = dependencies.reduce((sum, dep) => sum + dep.bandwidth, 0);
    const errorRate = dependencies.reduce((sum, dep) => sum + dep.errorRate, 0) / dependencies.length;

    return {
      securityVulnerabilities,
      averageLatency: Math.round(averageLatency),
      totalTrafficVolume: Math.round(totalTrafficVolume / 1024 / 1024), // MB/s
      errorRate: (errorRate * 100).toFixed(2),
    };
  }, [nodes, dependencies]);

  return (
    <main className="App-main">
      <div className="dashboard-header">
        <h2>🔗 Network Dependencies</h2>
        <p>Comprehensive network dependency analysis and visualization</p>
      </div>

      {/* Network Overview Cards */}
      <div className="metric-cards-row">
        <div className="metric-card">
          <div className="metric-card-value">{nodes.length}</div>
          <div className="metric-card-label">Network Nodes</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">{dependencies.length}</div>
          <div className="metric-card-label">Dependencies</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">{networkMetrics.securityVulnerabilities}</div>
          <div className="metric-card-label">Security Issues</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">{networkMetrics.averageLatency}ms</div>
          <div className="metric-card-label">Avg Latency</div>
        </div>
      </div>

      {/* Additional Metrics */}
      <div className="metric-cards-row">
        <div className="metric-card">
          <div className="metric-card-value">
            {nodes.filter(n => n.criticality === 'critical').length}
          </div>
          <div className="metric-card-label">Critical Nodes</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">
            {dependencies.filter(d => d.security.encrypted).length}
          </div>
          <div className="metric-card-label">Encrypted Connections</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">{networkMetrics.totalTrafficVolume}MB/s</div>
          <div className="metric-card-label">Total Bandwidth</div>
        </div>
        <div className="metric-card">
          <div className="metric-card-value">{networkMetrics.errorRate}%</div>
          <div className="metric-card-label">Error Rate</div>
        </div>
      </div>

      {/* Network Dependency Visualization */}
      <div className="chart-container">
        <div className="chart-header">
          <h3>Network Dependency Graph</h3>
          <p>Interactive visualization of network dependencies, security, and performance metrics</p>
        </div>
        <NetworkDependencyGraph
          initialNodes={nodes}
          initialDependencies={dependencies}
        />
      </div>

      {/* Feature Information */}
      <div className="chart-container">
        <div className="chart-header">
          <h3>🔗 Network Dependency Features</h3>
          <p>Advanced network analysis and dependency visualization</p>
        </div>
        <div style={{ padding: '1rem 0' }}>
          <h4>Analysis Features:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>📊 **Impact Analysis** - Assess downstream effects of node failures</li>
            <li>🔒 **Security Analysis** - Identify vulnerabilities and compliance gaps</li>
            <li>🔍 **Path Tracing** - Find dependency paths between any two nodes</li>
            <li>⚡ **Performance Monitoring** - Track latency, throughput, and error rates</li>
            <li>🎯 **Critical Path Detection** - Automatically identify high-risk dependency chains</li>
            <li>📈 **Bottleneck Analysis** - Locate performance and capacity constraints</li>
          </ul>
          
          <h4 style={{ marginTop: '1.5rem' }}>Visualization Features:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>🎨 **Interactive Nodes** - Click to view detailed information and analysis</li>
            <li>🌊 **Traffic Flow Animation** - Visual representation of data flow and volume</li>
            <li>🔐 **Security Indicators** - Color-coded encryption and authentication status</li>
            <li>⚠️ **Risk Highlighting** - Automatic highlighting of high-risk components</li>
            <li>📏 **Multiple Layouts** - Hierarchical, force-directed, and clustered arrangements</li>
            <li>🔍 **Zoom and Pan** - Detailed exploration of complex network topologies</li>
          </ul>

          <h4 style={{ marginTop: '1.5rem' }}>Node Types:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>⚙️ **Service** - Application services and microservices</li>
            <li>🗄️ **Database** - Data storage and management systems</li>
            <li>🌐 **External** - Third-party APIs and external dependencies</li>
            <li>⚖️ **Load Balancer** - Traffic distribution and routing</li>
            <li>🚪 **Gateway** - API gateways and service meshes</li>
            <li>💾 **Cache** - Redis, Memcached, and other caching layers</li>
            <li>📬 **Queue** - Message queues and event streaming platforms</li>
          </ul>

          <h4 style={{ marginTop: '1.5rem' }}>Dependency Types:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>🌐 **API Calls** - REST, GraphQL, and HTTP-based communications</li>
            <li>⚡ **gRPC** - High-performance RPC connections</li>
            <li>🗄️ **Database Queries** - SQL and NoSQL database operations</li>
            <li>💾 **Cache Access** - Redis and in-memory cache operations</li>
            <li>📬 **Event Streams** - Message queues and pub/sub patterns</li>
            <li>⚖️ **Load Balancing** - Traffic distribution patterns</li>
            <li>🔗 **Proxy** - Reverse proxy and forwarding relationships</li>
          </ul>

          <h4 style={{ marginTop: '1.5rem' }}>Security Analysis:</h4>
          <ul style={{ margin: '0.5rem 0', paddingLeft: '1.5rem', color: '#6b7280' }}>
            <li>🔒 **Encryption Status** - TLS/SSL and end-to-end encryption</li>
            <li>👤 **Authentication** - Identity verification mechanisms</li>
            <li>🛡️ **Authorization** - Access control and permissions</li>
            <li>🔥 **Firewall Protection** - Network-level security controls</li>
            <li>📋 **Compliance Checking** - SOC2, ISO27001, and other standards</li>
            <li>⚠️ **Vulnerability Assessment** - Risk scoring and recommendations</li>
          </ul>
        </div>
      </div>
    </main>
  );
}