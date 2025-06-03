import React, { useState, useCallback, useMemo } from 'react';
import { InteractivePieChart } from './InteractivePieChart';
import { InteractiveBarChart } from './InteractiveBarChart';
import { InteractiveSankey } from './InteractiveSankey';
import { InteractiveTreemap } from './InteractiveTreemap';
import { InteractiveRadar } from './InteractiveRadar';
import {
  InteractivePieData,
  InteractiveBarData,
  SankeyNode,
  SankeyLink,
  TreemapNode,
  RadarAxis,
  RadarSeries,
} from '../../types/interactiveCharts';
import './InteractiveCharts.css';

// Generate sample data
const generatePieData = (): InteractivePieData[] => [
  {
    id: 'compute',
    label: 'Compute Resources',
    value: 450,
    color: '#3b82f6',
    children: [
      { id: 'compute-cpu', label: 'CPU', value: 200, color: '#60a5fa' },
      { id: 'compute-memory', label: 'Memory', value: 150, color: '#93c5fd' },
      { id: 'compute-gpu', label: 'GPU', value: 100, color: '#dbeafe' },
    ],
  },
  {
    id: 'storage',
    label: 'Storage',
    value: 320,
    color: '#10b981',
    children: [
      { id: 'storage-ssd', label: 'SSD', value: 200, color: '#34d399' },
      { id: 'storage-hdd', label: 'HDD', value: 120, color: '#86efac' },
    ],
  },
  {
    id: 'network',
    label: 'Network',
    value: 180,
    color: '#f59e0b',
    children: [
      { id: 'network-ingress', label: 'Ingress', value: 100, color: '#fbbf24' },
      { id: 'network-egress', label: 'Egress', value: 80, color: '#fde68a' },
    ],
  },
  {
    id: 'services',
    label: 'Services',
    value: 250,
    color: '#ef4444',
  },
];

const generateBarData = (): InteractiveBarData[] => {
  const categories = ['Web Service', 'API Gateway', 'Database', 'Cache', 'Queue', 'Storage', 'Analytics'];
  const regions = ['us-east-1', 'us-west-2', 'eu-west-1'];
  
  return categories.flatMap(category => 
    regions.map(region => ({
      id: `${category}-${region}`,
      category: category,
      value: Math.floor(Math.random() * 500) + 100,
      metadata: {
        region,
        status: Math.random() > 0.2 ? 'healthy' : 'warning',
      },
    }))
  );
};

const generateSankeyData = (): { nodes: SankeyNode[], links: SankeyLink[] } => {
  const nodes: SankeyNode[] = [
    // Source nodes
    { id: 'frontend', name: 'Frontend', value: 1000, color: '#3b82f6' },
    { id: 'mobile', name: 'Mobile App', value: 800, color: '#8b5cf6' },
    { id: 'api', name: 'External API', value: 600, color: '#ec4899' },
    // Middle layer
    { id: 'gateway', name: 'API Gateway', value: 2400, color: '#f59e0b' },
    { id: 'auth', name: 'Auth Service', value: 1200, color: '#10b981' },
    // Backend services
    { id: 'users', name: 'User Service', value: 800, color: '#ef4444' },
    { id: 'products', name: 'Product Service', value: 1000, color: '#06b6d4' },
    { id: 'orders', name: 'Order Service', value: 600, color: '#6366f1' },
    // Data layer
    { id: 'db-main', name: 'Main Database', value: 1800, color: '#84cc16' },
    { id: 'cache', name: 'Redis Cache', value: 600, color: '#f97316' },
  ];

  const links: SankeyLink[] = [
    // Frontend to Gateway
    { source: 'frontend', target: 'gateway', value: 1000 },
    { source: 'mobile', target: 'gateway', value: 800 },
    { source: 'api', target: 'gateway', value: 600 },
    // Gateway to Auth
    { source: 'gateway', target: 'auth', value: 1200 },
    // Gateway to Services
    { source: 'gateway', target: 'users', value: 400 },
    { source: 'gateway', target: 'products', value: 500 },
    { source: 'gateway', target: 'orders', value: 300 },
    // Auth to Services
    { source: 'auth', target: 'users', value: 400 },
    { source: 'auth', target: 'products', value: 500 },
    { source: 'auth', target: 'orders', value: 300 },
    // Services to Data
    { source: 'users', target: 'db-main', value: 600 },
    { source: 'products', target: 'db-main', value: 700 },
    { source: 'orders', target: 'db-main', value: 500 },
    { source: 'users', target: 'cache', value: 200 },
    { source: 'products', target: 'cache', value: 300 },
    { source: 'orders', target: 'cache', value: 100 },
  ];

  return { nodes, links };
};

const generateTreemapData = (): TreemapNode => ({
  id: 'root',
  name: 'Infrastructure',
  value: 0,
  children: [
    {
      id: 'compute',
      name: 'Compute',
      value: 0,
      children: [
        {
          id: 'ec2',
          name: 'EC2 Instances',
          value: 0,
          children: [
            { id: 'ec2-large', name: 'm5.large', value: 250 },
            { id: 'ec2-xlarge', name: 'm5.xlarge', value: 180 },
            { id: 'ec2-2xlarge', name: 'm5.2xlarge', value: 120 },
            { id: 'ec2-small', name: 't3.small', value: 80 },
          ],
        },
        {
          id: 'containers',
          name: 'Containers',
          value: 0,
          children: [
            { id: 'ecs-tasks', name: 'ECS Tasks', value: 300 },
            { id: 'fargate', name: 'Fargate', value: 200 },
            { id: 'kubernetes', name: 'Kubernetes Pods', value: 150 },
          ],
        },
        {
          id: 'serverless',
          name: 'Serverless',
          value: 0,
          children: [
            { id: 'lambda', name: 'Lambda Functions', value: 100 },
            { id: 'batch', name: 'Batch Jobs', value: 50 },
          ],
        },
      ],
    },
    {
      id: 'storage',
      name: 'Storage',
      value: 0,
      children: [
        {
          id: 'object',
          name: 'Object Storage',
          value: 0,
          children: [
            { id: 's3-standard', name: 'S3 Standard', value: 400 },
            { id: 's3-ia', name: 'S3 IA', value: 200 },
            { id: 's3-glacier', name: 'S3 Glacier', value: 100 },
          ],
        },
        {
          id: 'block',
          name: 'Block Storage',
          value: 0,
          children: [
            { id: 'ebs-gp3', name: 'EBS gp3', value: 300 },
            { id: 'ebs-io2', name: 'EBS io2', value: 150 },
          ],
        },
      ],
    },
    {
      id: 'database',
      name: 'Database',
      value: 0,
      children: [
        { id: 'rds', name: 'RDS', value: 350 },
        { id: 'dynamodb', name: 'DynamoDB', value: 250 },
        { id: 'elasticache', name: 'ElastiCache', value: 150 },
        { id: 'documentdb', name: 'DocumentDB', value: 100 },
      ],
    },
  ],
});

const generateRadarData = (): { axes: RadarAxis[], series: RadarSeries[] } => {
  const axes: RadarAxis[] = [
    { id: 'performance', label: 'Performance' },
    { id: 'scalability', label: 'Scalability' },
    { id: 'reliability', label: 'Reliability' },
    { id: 'security', label: 'Security' },
    { id: 'cost', label: 'Cost Efficiency' },
    { id: 'maintainability', label: 'Maintainability' },
  ];

  const series: RadarSeries[] = [
    {
      id: 'current',
      name: 'Current State',
      color: '#3b82f6',
      data: [
        { axis: 'performance', value: 75 },
        { axis: 'scalability', value: 85 },
        { axis: 'reliability', value: 70 },
        { axis: 'security', value: 90 },
        { axis: 'cost', value: 60 },
        { axis: 'maintainability', value: 65 },
      ],
    },
    {
      id: 'target',
      name: 'Target State',
      color: '#10b981',
      data: [
        { axis: 'performance', value: 90 },
        { axis: 'scalability', value: 95 },
        { axis: 'reliability', value: 85 },
        { axis: 'security', value: 95 },
        { axis: 'cost', value: 80 },
        { axis: 'maintainability', value: 85 },
      ],
    },
    {
      id: 'industry',
      name: 'Industry Average',
      color: '#f59e0b',
      data: [
        { axis: 'performance', value: 70 },
        { axis: 'scalability', value: 75 },
        { axis: 'reliability', value: 80 },
        { axis: 'security', value: 85 },
        { axis: 'cost', value: 70 },
        { axis: 'maintainability', value: 75 },
      ],
    },
  ];

  return { axes, series };
};

export function InteractiveChartsDashboard() {
  const [selectedChart, setSelectedChart] = useState<string>('pie');
  const [selectedPieSegment, setSelectedPieSegment] = useState<string | null>(null);
  const [selectedBarId, setSelectedBarId] = useState<string | null>(null);
  const [selectedTreemapNode, setSelectedTreemapNode] = useState<string | null>(null);
  const [selectedRadarSeries, setSelectedRadarSeries] = useState<string | null>(null);

  // Generate data
  const pieData = useMemo(() => generatePieData(), []);
  const barData = useMemo(() => generateBarData(), []);
  const sankeyData = useMemo(() => generateSankeyData(), []);
  const treemapData = useMemo(() => generateTreemapData(), []);
  const radarData = useMemo(() => generateRadarData(), []);

  // Event handlers
  const handlePieSegmentClick = useCallback((segment: InteractivePieData) => {
    setSelectedPieSegment(segment.id);
    console.log('Pie segment clicked:', segment);
  }, []);

  const handleBarClick = useCallback((bar: InteractiveBarData) => {
    setSelectedBarId(bar.id);
    console.log('Bar clicked:', bar);
  }, []);

  const handleTreemapNodeClick = useCallback((node: TreemapNode) => {
    setSelectedTreemapNode(node.id);
    console.log('Treemap node clicked:', node);
  }, []);

  const handleRadarSeriesClick = useCallback((series: RadarSeries) => {
    setSelectedRadarSeries(series.id);
    console.log('Radar series clicked:', series);
  }, []);

  const renderChart = () => {
    switch (selectedChart) {
      case 'pie':
        return (
          <InteractivePieChart
            data={pieData}
            config={{ 
              width: 900, 
              height: 550, 
              animationDuration: 500,
              margin: { top: 20, right: 40, bottom: 60, left: 40 }
            }}
            drillDown={{
              enabled: true,
              levels: ['category', 'subcategory'],
              onDrillDown: (id, data) => console.log('Drill down:', id, data),
              onDrillUp: () => console.log('Drill up'),
            }}
            onSegmentClick={handlePieSegmentClick}
            selectedSegmentId={selectedPieSegment || undefined}
            innerRadius={70}
            outerRadius={140}
          />
        );
      
      case 'bar':
        return (
          <InteractiveBarChart
            data={barData}
            config={{ width: 800, height: 400 }}
            orientation="vertical"
            sortBy="value"
            sortOrder="desc"
            onBarClick={handleBarClick}
            selectedBarId={selectedBarId || undefined}
            enableSorting={true}
            enableFiltering={true}
            filterOptions={[
              {
                field: 'region',
                label: 'Filter by Region',
                type: 'select',
                values: ['us-east-1', 'us-west-2', 'eu-west-1'],
              },
              {
                field: 'status',
                label: 'Filter by Status',
                type: 'select',
                values: ['healthy', 'warning'],
              },
            ]}
          />
        );
      
      case 'sankey':
        return (
          <InteractiveSankey
            nodes={sankeyData.nodes}
            links={sankeyData.links}
            config={{ width: 800, height: 500 }}
            nodeWidth={20}
            nodePadding={15}
            onNodeClick={(node) => console.log('Sankey node clicked:', node)}
            onLinkClick={(link) => console.log('Sankey link clicked:', link)}
            highlightConnected={true}
            enableNodeDragging={true}
          />
        );
      
      case 'treemap':
        return (
          <InteractiveTreemap
            data={treemapData}
            config={{ width: 800, height: 500 }}
            tileType="squarify"
            onNodeClick={handleTreemapNodeClick}
            selectedNodeId={selectedTreemapNode || undefined}
            enableZoom={true}
            maxDepth={3}
          />
        );
      
      case 'radar':
        return (
          <InteractiveRadar
            axes={radarData.axes}
            series={radarData.series}
            config={{ width: 600, height: 600 }}
            maxValue={100}
            levels={5}
            onSeriesClick={handleRadarSeriesClick}
            selectedSeriesId={selectedRadarSeries || undefined}
            onAxisClick={(axis) => console.log('Axis clicked:', axis)}
            showGrid={true}
            showAxis={true}
            showLabels={true}
            showDots={true}
            animateOnLoad={true}
          />
        );
      
      default:
        return null;
    }
  };

  return (
    <div className="interactive-charts-dashboard">
      <h2>Interactive Charts</h2>
      
      <div className="chart-selector">
        <button
          className={selectedChart === 'pie' ? 'active' : ''}
          onClick={() => setSelectedChart('pie')}
        >
          Pie Chart
        </button>
        <button
          className={selectedChart === 'bar' ? 'active' : ''}
          onClick={() => setSelectedChart('bar')}
        >
          Bar Chart
        </button>
        <button
          className={selectedChart === 'sankey' ? 'active' : ''}
          onClick={() => setSelectedChart('sankey')}
        >
          Sankey Diagram
        </button>
        <button
          className={selectedChart === 'treemap' ? 'active' : ''}
          onClick={() => setSelectedChart('treemap')}
        >
          Treemap
        </button>
        <button
          className={selectedChart === 'radar' ? 'active' : ''}
          onClick={() => setSelectedChart('radar')}
        >
          Radar Chart
        </button>
      </div>

      <div className="chart-container">
        {renderChart()}
      </div>

      <style>{`
        .interactive-charts-dashboard {
          padding: 24px;
          max-width: 1200px;
          margin: 0 auto;
        }

        .interactive-charts-dashboard h2 {
          font-size: 24px;
          font-weight: 600;
          margin-bottom: 24px;
          color: #1f2937;
        }

        .chart-selector {
          display: flex;
          gap: 12px;
          margin-bottom: 24px;
          padding: 16px;
          background: #f9fafb;
          border-radius: 8px;
        }

        .chart-selector button {
          padding: 8px 16px;
          background: white;
          border: 1px solid #e5e7eb;
          border-radius: 6px;
          cursor: pointer;
          font-size: 14px;
          font-weight: 500;
          transition: all 0.2s;
        }

        .chart-selector button:hover {
          background: #f3f4f6;
          border-color: #d1d5db;
        }

        .chart-selector button.active {
          background: #3b82f6;
          color: white;
          border-color: #3b82f6;
        }

        .chart-container {
          background: white;
          border-radius: 8px;
          padding: 24px;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
          border: 1px solid #e5e7eb;
          min-height: 600px;
          display: flex;
          align-items: center;
          justify-content: center;
        }

        .radar-legend {
          display: flex;
          gap: 16px;
          margin-top: 16px;
          padding: 12px;
          background: #f9fafb;
          border-radius: 6px;
          font-size: 14px;
        }

        .radar-statistics {
          display: flex;
          gap: 24px;
          margin-top: 16px;
          padding: 16px;
          background: #f9fafb;
          border-radius: 6px;
        }

        .stat-item {
          display: flex;
          flex-direction: column;
          gap: 4px;
        }

        .stat-label {
          font-size: 12px;
          color: #6b7280;
          font-weight: 500;
        }

        .stat-value {
          font-size: 16px;
          font-weight: 600;
          color: #1f2937;
        }
      `}</style>
    </div>
  );
}