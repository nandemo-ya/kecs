# Web UI Integration

This document describes how the Web UI is integrated into the KECS Control Plane.

## Overview

The KECS Control Plane now includes an embedded Web UI that provides:
- Real-time monitoring of ECS resources
- WebSocket-based live updates
- Interactive dashboards
- Log streaming
- Metrics visualization

## Architecture

### Static File Serving

The Web UI is served from the `/ui/` endpoint:
- Production: Files are embedded into the binary using Go's `embed` package
- Development: Files are served from the `web-ui/build` directory

### WebSocket Endpoints

Real-time features are powered by WebSocket connections:
- `/ws` - General WebSocket endpoint
- `/ws/logs` - Log streaming
- `/ws/metrics` - Metrics updates
- `/ws/notifications` - System notifications
- `/ws/tasks` - Task status updates

### API Proxy

API requests from the Web UI are proxied through `/api/` to the ECS API endpoints.

## Building

### Development Mode

Run the Control Plane without embedding the Web UI:
```bash
cd controlplane
go run cmd/controlplane/main.go
```

The Web UI will be served from `../web-ui/build`. Make sure to build the Web UI first:
```bash
cd web-ui
npm install
npm run build
```

### Production Mode

Build the Control Plane with embedded Web UI:
```bash
cd controlplane
./scripts/build-webui.sh
```

This script will:
1. Build the React Web UI
2. Copy the build artifacts to `controlplane/internal/controlplane/api/webui_dist`
3. Build the Control Plane binary with the `embed_webui` build tag

## Accessing the Web UI

Once the Control Plane is running, access the Web UI at:
```
http://localhost:8080/ui/
```

## Security

### CORS
Cross-Origin Resource Sharing (CORS) is configured to allow the Web UI to make API calls.

### Security Headers
The following security headers are set:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- Content Security Policy for API endpoints

### WebSocket Security
WebSocket connections include:
- Origin validation
- Heartbeat/ping-pong mechanism
- Connection limits

## Development Tips

### Hot Reload
For development with hot reload:
1. Run the Control Plane in development mode
2. Run the Web UI dev server: `cd web-ui && npm start`
3. Configure the Web UI to proxy API calls to the Control Plane

### Adding New WebSocket Features
1. Add message type handling in `websocket_handler.go`
2. Create corresponding React hook in `web-ui/src/hooks/`
3. Update WebSocket message types

### Debugging WebSocket
- Check browser console for WebSocket connection status
- Monitor server logs for WebSocket events
- Use browser DevTools Network tab to inspect WebSocket messages