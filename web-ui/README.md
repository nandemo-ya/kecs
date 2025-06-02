# KECS Web UI

React/TypeScript-based Web UI for KECS (Kubernetes-based ECS Compatible Service).

## Overview

The KECS Web UI provides a modern, intuitive interface for managing and monitoring your Kubernetes-based ECS compatible services. It offers real-time dashboards, service management, and comprehensive system monitoring.

## Features

### Current Implementation
- ✅ **Dashboard**: Real-time overview of clusters, services, tasks, and task definitions
- ✅ **Detail Views**: Individual cluster, service, and task detail pages with comprehensive information
- ✅ **Navigation**: React Router-based SPA with proper URL routing and navigation
- ✅ **API Integration**: Full integration with KECS Control Plane REST API
- ✅ **Health Monitoring**: Real-time connection status and health checks
- ✅ **Auto-refresh**: Automatic data updates every 30 seconds
- ✅ **Error Handling**: Graceful error handling with retry functionality
- ✅ **Loading States**: Visual feedback during API calls
- ✅ **Quick Access**: Direct links to resources from dashboard
- ✅ **Responsive Design**: Mobile-friendly interface
- ✅ **Modern UI**: Clean, professional design with Tailwind-inspired styling

### Planned Features
- 📊 **Metrics Visualization**: Charts and graphs using Recharts
- 🔗 **Service Topology**: Interactive service maps with React Flow
- 📝 **Log Viewer**: Real-time container and service logs
- 🌐 **WebSocket Support**: Real-time updates without polling
- 🔧 **Service Management**: Create, update, and delete services through the UI
- 📋 **List Views**: Comprehensive list pages for all resource types

## Technology Stack

- **React 19** with **TypeScript** - Modern React with type safety
- **React Router** - Client-side routing for SPA navigation
- **Create React App** - Standard React development environment
- **CSS3** - Custom styling with modern design patterns
- **Future additions**: Recharts, React Flow, WebSocket client

## Getting Started

### Prerequisites
- Node.js 16+ and npm
- KECS Control Plane running on http://localhost:8080

### Development

1. **Install dependencies:**
   ```bash
   npm install
   ```

2. **Start the development server:**
   ```bash
   npm start
   ```
   
   The app will open at [http://localhost:3000](http://localhost:3000)

3. **Build for production:**
   ```bash
   npm run build
   ```

### Available Scripts

- `npm start` - Runs the development server
- `npm test` - Launches the test runner
- `npm run build` - Builds the app for production
- `npm run eject` - Ejects from Create React App (one-way operation)

## Project Structure

```
src/
├── App.tsx          # Main application component
├── App.css          # Application styles
├── index.tsx        # React app entry point
├── index.css        # Global styles
└── ...              # Additional components (to be added)
```

## Configuration

The Web UI is configured to work with the KECS Control Plane API at `http://localhost:8080`. This can be configured through environment variables in future versions.

## Integration with KECS Control Plane

The Web UI will communicate with the KECS Control Plane through:
- REST API calls to `http://localhost:8080/v1/*` endpoints
- Future WebSocket connections for real-time updates
- Health checks via `/health` endpoint

## Development Notes

This project was bootstrapped with [Create React App](https://github.com/facebook/create-react-app) using the TypeScript template.

For more information about React development, see the [React documentation](https://reactjs.org/).

## Contributing

When adding new features:
1. Follow the existing code style and structure
2. Add TypeScript types for all new components and data
3. Ensure responsive design compatibility
4. Update this README with new features