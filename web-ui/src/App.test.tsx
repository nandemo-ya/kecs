import React from 'react';
import { render } from '@testing-library/react';

// Mock all the components to avoid complex dependencies
jest.mock('./components/Dashboard', () => ({
  __esModule: true,
  default: () => <div>Dashboard Component</div>
}));

jest.mock('./components/ClusterList', () => ({
  __esModule: true,
  default: () => <div>ClusterList Component</div>
}));

jest.mock('./components/ClusterDetail', () => ({
  __esModule: true,
  default: () => <div>ClusterDetail Component</div>
}));

jest.mock('./components/ServiceList', () => ({
  __esModule: true,
  default: () => <div>ServiceList Component</div>
}));

jest.mock('./components/ServiceDetail', () => ({
  __esModule: true,
  default: () => <div>ServiceDetail Component</div>
}));

jest.mock('./components/TaskList', () => ({
  __esModule: true,
  default: () => <div>TaskList Component</div>
}));

jest.mock('./components/TaskDetail', () => ({
  __esModule: true,
  default: () => <div>TaskDetail Component</div>
}));

jest.mock('./components/TaskDefinitionList', () => ({
  __esModule: true,
  default: () => <div>TaskDefinitionList Component</div>
}));

jest.mock('./components/TaskDefinitionDetail', () => ({
  __esModule: true,
  default: () => <div>TaskDefinitionDetail Component</div>
}));

jest.mock('./components/CreateService', () => ({
  __esModule: true,
  default: () => <div>CreateService Component</div>
}));

jest.mock('./components/UpdateService', () => ({
  __esModule: true,
  default: () => <div>UpdateService Component</div>
}));

jest.mock('./components/RegisterTaskDefinition', () => ({
  __esModule: true,
  default: () => <div>RegisterTaskDefinition Component</div>
}));

jest.mock('./components/WebSocketDemo', () => ({
  __esModule: true,
  default: () => <div>WebSocketDemo Component</div>
}));

jest.mock('./components/MetricsDashboard', () => ({
  __esModule: true,
  default: () => <div>MetricsDashboard Component</div>
}));

jest.mock('./components/ServiceTopologyDashboard', () => ({
  __esModule: true,
  default: () => <div>ServiceTopologyDashboard Component</div>
}));

jest.mock('./components/LogViewerDashboard', () => ({
  __esModule: true,
  default: () => <div>LogViewerDashboard Component</div>
}));

jest.mock('./components/charts/InteractiveChartsDashboard', () => ({
  __esModule: true,
  default: () => <div>InteractiveChartsDashboard Component</div>
}));

jest.mock('./components/TimeSeriesDashboard', () => ({
  __esModule: true,
  default: () => <div>TimeSeriesDashboard Component</div>
}));

jest.mock('./components/ResourceUsageDashboard', () => ({
  __esModule: true,
  default: () => <div>ResourceUsageDashboard Component</div>
}));

jest.mock('./components/NetworkDependencyDashboard', () => ({
  __esModule: true,
  default: () => <div>NetworkDependencyDashboard Component</div>
}));

import App from './App';

test('renders without crashing', () => {
  const { container } = render(<App />);
  expect(container).toBeTruthy();
});
