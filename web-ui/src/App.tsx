import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation } from 'react-router-dom';
import './App.css';
import { Dashboard } from './components/Dashboard';
import { ClusterList } from './components/ClusterList';
import { ClusterDetail } from './components/ClusterDetail';
import { ServiceList } from './components/ServiceList';
import { ServiceDetail } from './components/ServiceDetail';
import { CreateService } from './components/CreateService';
import { UpdateService } from './components/UpdateService';
import { TaskList } from './components/TaskList';
import { TaskDetail } from './components/TaskDetail';
import { TaskDefinitionList } from './components/TaskDefinitionList';
import { TaskDefinitionDetail } from './components/TaskDefinitionDetail';
import { RegisterTaskDefinition } from './components/RegisterTaskDefinition';
import { MetricsDashboard } from './components/MetricsDashboard';
import { NotificationProvider } from './contexts/NotificationContext';
import { NotificationContainer } from './components/NotificationContainer';

function Navigation() {
  const location = useLocation();
  
  return (
    <header className="App-header">
      <Link to="/" className="header-title">
        <h1>KECS Web UI</h1>
        <p>Kubernetes-based ECS Compatible Service</p>
      </Link>
      <nav>
        <ul>
          <li>
            <Link 
              to="/" 
              className={location.pathname === '/' ? 'active' : ''}
            >
              Dashboard
            </Link>
          </li>
          <li>
            <Link 
              to="/clusters" 
              className={location.pathname.startsWith('/clusters') ? 'active' : ''}
            >
              Clusters
            </Link>
          </li>
          <li>
            <Link 
              to="/services" 
              className={location.pathname.startsWith('/services') ? 'active' : ''}
            >
              Services
            </Link>
          </li>
          <li>
            <Link 
              to="/tasks" 
              className={location.pathname.startsWith('/tasks') ? 'active' : ''}
            >
              Tasks
            </Link>
          </li>
          <li>
            <Link 
              to="/task-definitions" 
              className={location.pathname.startsWith('/task-definitions') ? 'active' : ''}
            >
              Task Definitions
            </Link>
          </li>
          <li>
            <Link 
              to="/metrics" 
              className={location.pathname.startsWith('/metrics') ? 'active' : ''}
            >
              Metrics
            </Link>
          </li>
        </ul>
      </nav>
    </header>
  );
}

function AppContent() {
  return (
    <div className="App">
      <Navigation />
      
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/clusters" element={<ClusterList />} />
        <Route path="/clusters/:clusterName" element={<ClusterDetail />} />
        <Route path="/services" element={<ServiceList />} />
        <Route path="/services/create" element={<CreateService />} />
        <Route path="/services/:serviceName" element={<ServiceDetail />} />
        <Route path="/services/:serviceName/update" element={<UpdateService />} />
        <Route path="/tasks" element={<TaskList />} />
        <Route path="/tasks/:taskId" element={<TaskDetail />} />
        <Route path="/task-definitions" element={<TaskDefinitionList />} />
        <Route path="/task-definitions/register" element={<RegisterTaskDefinition />} />
        <Route path="/task-definitions/:family/:revision" element={<TaskDefinitionDetail />} />
        <Route path="/metrics" element={<MetricsDashboard />} />
        <Route path="*" element={<div className="placeholder">Page Not Found</div>} />
      </Routes>
      
      <NotificationContainer />
      
      <footer className="App-footer">
        <p>&copy; 2025 KECS - Kubernetes-based ECS Compatible Service</p>
      </footer>
    </div>
  );
}

function App() {
  return (
    <NotificationProvider>
      <Router>
        <AppContent />
      </Router>
    </NotificationProvider>
  );
}

export default App;