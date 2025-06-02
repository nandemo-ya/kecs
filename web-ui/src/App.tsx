import React from 'react';
import './App.css';

function App() {
  return (
    <div className="App">
      <header className="App-header">
        <h1>KECS Web UI</h1>
        <p>Kubernetes-based ECS Compatible Service</p>
        <nav>
          <ul>
            <li><a href="#clusters">Clusters</a></li>
            <li><a href="#services">Services</a></li>
            <li><a href="#tasks">Tasks</a></li>
            <li><a href="#task-definitions">Task Definitions</a></li>
          </ul>
        </nav>
      </header>
      
      <main className="App-main">
        <section id="dashboard">
          <h2>Dashboard</h2>
          <div className="dashboard-cards">
            <div className="card">
              <h3>Clusters</h3>
              <p className="metric">-</p>
              <small>Active clusters</small>
            </div>
            <div className="card">
              <h3>Services</h3>
              <p className="metric">-</p>
              <small>Running services</small>
            </div>
            <div className="card">
              <h3>Tasks</h3>
              <p className="metric">-</p>
              <small>Active tasks</small>
            </div>
            <div className="card">
              <h3>Task Definitions</h3>
              <p className="metric">-</p>
              <small>Registered definitions</small>
            </div>
          </div>
        </section>
        
        <section id="status">
          <h2>System Status</h2>
          <div className="status-info">
            <p><strong>KECS Control Plane:</strong> <span className="status-indicator">Connecting...</span></p>
            <p><strong>API Endpoint:</strong> http://localhost:8080</p>
            <p><strong>Version:</strong> 0.1.0</p>
          </div>
        </section>
      </main>
      
      <footer className="App-footer">
        <p>&copy; 2025 KECS - Kubernetes-based ECS Compatible Service</p>
      </footer>
    </div>
  );
}

export default App;