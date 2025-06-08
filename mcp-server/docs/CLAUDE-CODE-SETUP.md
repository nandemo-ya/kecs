# Claude Code (VS Code) Setup Guide for KECS MCP Server

## Overview

This guide explains how to configure the KECS MCP Server for use with Claude Code in VS Code.

## Installation Steps

### 1. Build the MCP Server

First, build the KECS MCP server:

```bash
cd mcp-server
npm install
npm run build
```

### 2. Configure Claude Code

Claude Code uses a configuration file to specify MCP servers. The location depends on your operating system:

- **macOS**: `~/Library/Application Support/Claude/claude_code_config.json`
- **Windows**: `%APPDATA%\Claude\claude_code_config.json`
- **Linux**: `~/.config/claude/claude_code_config.json`

### 3. Add KECS MCP Server Configuration

Edit the configuration file and add the KECS MCP server:

```json
{
  "mcpServers": {
    "kecs": {
      "command": "node",
      "args": ["/absolute/path/to/kecs/mcp-server/dist/index.js"],
      "env": {
        "KECS_API_URL": "http://localhost:8080",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

**Important**: Replace `/absolute/path/to/kecs` with the actual absolute path to your KECS repository.

### 4. Alternative Configuration Options

#### Using npm/npx

If you prefer to run through npm:

```json
{
  "mcpServers": {
    "kecs": {
      "command": "npm",
      "args": ["run", "start"],
      "cwd": "/absolute/path/to/kecs/mcp-server",
      "env": {
        "KECS_API_URL": "http://localhost:8080",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

#### Development Mode

For development with hot-reloading:

```json
{
  "mcpServers": {
    "kecs-dev": {
      "command": "npm",
      "args": ["run", "dev"],
      "cwd": "/absolute/path/to/kecs/mcp-server",
      "env": {
        "KECS_API_URL": "http://localhost:8080",
        "LOG_LEVEL": "debug",
        "NODE_ENV": "development"
      }
    }
  }
}
```

### 5. Verify Configuration

1. Restart VS Code after updating the configuration
2. Open Claude Code in VS Code
3. The KECS MCP server should be available automatically

### 6. Using the MCP Server

Once configured, you can interact with KECS through Claude Code:

```
"List all clusters in KECS"
"Show me the services running in the default cluster"
"Create a new service called web-app"
"Describe the task definition nginx:1"
```

## Troubleshooting

### MCP Server Not Available

1. Check the configuration file path is correct
2. Ensure the MCP server is built (`npm run build`)
3. Verify the absolute path in the configuration
4. Check VS Code logs for errors

### Connection Errors

1. Ensure KECS is running on the configured port:
   ```bash
   curl http://localhost:8080/health
   ```

2. Check environment variables in the configuration

### View Logs

To see MCP server logs:

1. Open VS Code Developer Tools: `Help > Toggle Developer Tools`
2. Check the Console tab for MCP-related messages

### Debug Mode

Enable debug logging by setting `LOG_LEVEL` to `debug` in the configuration:

```json
{
  "env": {
    "KECS_API_URL": "http://localhost:8080",
    "LOG_LEVEL": "debug"
  }
}
```

## Environment Variables

The following environment variables can be configured:

- `KECS_API_URL`: KECS API endpoint (default: `http://localhost:8080`)
- `KECS_API_TIMEOUT`: Request timeout in milliseconds (default: `30000`)
- `LOG_LEVEL`: Logging level - `error`, `warn`, `info`, `debug` (default: `info`)
- `NODE_ENV`: Environment - `production`, `development` (default: `development`)

## Example Configurations

### Multiple KECS Environments

```json
{
  "mcpServers": {
    "kecs-local": {
      "command": "node",
      "args": ["/path/to/kecs/mcp-server/dist/index.js"],
      "env": {
        "KECS_API_URL": "http://localhost:8080",
        "LOG_LEVEL": "info"
      }
    },
    "kecs-staging": {
      "command": "node",
      "args": ["/path/to/kecs/mcp-server/dist/index.js"],
      "env": {
        "KECS_API_URL": "https://kecs-staging.example.com",
        "KECS_API_TOKEN": "staging-token",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

### With Custom Node Version

```json
{
  "mcpServers": {
    "kecs": {
      "command": "/usr/local/bin/node",
      "args": ["/path/to/kecs/mcp-server/dist/index.js"],
      "env": {
        "KECS_API_URL": "http://localhost:8080"
      }
    }
  }
}
```

## Notes

- The MCP server must be built before use (`npm run build`)
- Paths must be absolute, not relative
- Environment variables override default values
- Multiple MCP servers can be configured simultaneously
- The server name (e.g., "kecs") is used internally by Claude Code