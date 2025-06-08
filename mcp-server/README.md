# KECS MCP Server

MCP (Model Context Protocol) server for KECS - Kubernetes-based ECS Compatible Service.

## Overview

This MCP server provides tools for interacting with KECS through Claude Desktop or other MCP-compatible clients. It enables AI assistants to manage ECS-compatible resources running on Kubernetes.

## Features

### Available Tools

#### Cluster Management
- `list-clusters` - List all ECS clusters
- `describe-clusters` - Get detailed information about clusters
- `create-cluster` - Create a new cluster
- `delete-cluster` - Delete a cluster

#### Service Management
- `list-services` - List services in a cluster
- `describe-services` - Get detailed information about services
- `create-service` - Create a new service
- `update-service` - Update an existing service
- `delete-service` - Delete a service

#### Task Management
- `list-tasks` - List tasks in a cluster
- `describe-tasks` - Get detailed information about tasks
- `run-task` - Run a new task
- `stop-task` - Stop a running task

#### Task Definition Management
- `list-task-definitions` - List task definitions
- `describe-task-definition` - Get detailed information about a task definition
- `register-task-definition` - Register a new task definition
- `deregister-task-definition` - Deregister a task definition

## Installation

1. Clone the repository and navigate to the MCP server directory:
```bash
git clone https://github.com/nandemo-ya/kecs.git
cd kecs/mcp-server
```

2. Install dependencies:
```bash
npm install
```

3. Build the TypeScript code:
```bash
npm run build
```

## Configuration

### Environment Variables

- `KECS_API_URL` - KECS API endpoint (default: `http://localhost:8080`)
- `KECS_API_TIMEOUT` - API request timeout in milliseconds (default: `30000`)
- `LOG_LEVEL` - Logging level: `error`, `warn`, `info`, `debug` (default: `info`)
- `NODE_ENV` - Environment: `production`, `development` (default: `development`)

### Claude Desktop Configuration

Add the following to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "kecs": {
      "command": "node",
      "args": ["/path/to/kecs/mcp-server/dist/index.js"],
      "env": {
        "KECS_API_URL": "http://localhost:8080",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

## Development

### Running in Development Mode

```bash
npm run dev
```

This will start the server with hot-reloading enabled.

### Running Tests

```bash
npm test
```

### Linting and Formatting

```bash
npm run lint
npm run format
```

## Usage Examples

Once configured in Claude Desktop, you can interact with KECS using natural language:

- "List all ECS clusters"
- "Show me the services running in the default cluster"
- "Create a new service called web-app using the nginx:latest task definition"
- "Scale the web-app service to 3 instances"
- "Stop all tasks in the test cluster"

## Architecture

The MCP server acts as a bridge between AI assistants and the KECS API:

```
Claude Desktop <-> MCP Protocol <-> KECS MCP Server <-> KECS API <-> Kubernetes
```

## Troubleshooting

### Connection Issues

1. Verify KECS is running:
```bash
curl http://localhost:8080/health
```

2. Check MCP server logs:
- Look for error messages in the Claude Desktop logs
- Run the server manually to see output:
```bash
node dist/index.js
```

### Common Errors

- **ECONNREFUSED**: KECS API is not running or not accessible
- **ETIMEDOUT**: Network timeout - check `KECS_API_TIMEOUT` setting
- **401 Unauthorized**: Authentication required (if KECS has auth enabled)

## Contributing

See the main KECS [CONTRIBUTING.md](../CONTRIBUTING.md) guide.

## License

MIT - See the main KECS [LICENSE](../LICENSE) file.