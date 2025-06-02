import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation } from 'react-router-dom';
import './App.css';
import { Dashboard } from './components/Dashboard';
import { ClusterList } from './components/ClusterList';
import { ClusterDetail } from './components/ClusterDetail';
import { ServiceDetail } from './components/ServiceDetail';
import { TaskDetail } from './components/TaskDetail';

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
        <Route path="/services/:serviceName" element={<ServiceDetail />} />
        <Route path="/tasks/:taskId" element={<TaskDetail />} />
        <Route path="/services" element={<div className="placeholder">Services List (Coming Soon)</div>} />
        <Route path="/tasks" element={<div className="placeholder">Tasks List (Coming Soon)</div>} />
        <Route path="/task-definitions" element={<div className="placeholder">Task Definitions List (Coming Soon)</div>} />
        <Route path="*" element={<div className="placeholder">Page Not Found</div>} />
      </Routes>
      
      <footer className="App-footer">
        <p>&copy; 2025 KECS - Kubernetes-based ECS Compatible Service</p>
      </footer>
    </div>
  );
}

function App() {
  return (
    <Router>
      <AppContent />
    </Router>
  );
}

export default App;