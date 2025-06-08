---
sidebar_position: 4
---

# Claude Code (VS Code) Setup

Configure the KECS MCP server for use with Claude Code in Visual Studio Code.

## Configuration File Location

Claude Code uses a configuration file similar to Claude Desktop. The location depends on your operating system:

- **macOS**: `~/Library/Application Support/Claude/claude_code_config.json`
- **Windows**: `%APPDATA%\Claude\claude_code_config.json`
- **Linux**: `~/.config/claude/claude_code_config.json`

## Basic Configuration

### 1. Build the MCP Server

First, ensure the MCP server is built:

```bash
cd /path/to/kecs/mcp-server
npm install
npm run build
```

### 2. Configure Claude Code

Create or edit the configuration file:

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

:::important
Replace `/absolute/path/to/kecs` with the actual absolute path to your KECS repository.
:::

### 3. Restart VS Code

After saving the configuration, restart VS Code for the changes to take effect.

## Configuration Options

### Using npm Scripts

Run the MCP server using npm:

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

### Development Mode

For active development with hot-reloading:

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

### Multiple Configurations

Set up different environments:

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
    "kecs-docker": {
      "command": "node",
      "args": ["/path/to/kecs/mcp-server/dist/index.js"],
      "env": {
        "KECS_API_URL": "http://host.docker.internal:8080",
        "LOG_LEVEL": "info"
      }
    },
    "kecs-remote": {
      "command": "node",
      "args": ["/path/to/kecs/mcp-server/dist/index.js"],
      "env": {
        "KECS_API_URL": "https://kecs.example.com",
        "KECS_API_TOKEN": "auth-token",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

### Custom Node Version

Use a specific Node.js version:

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

## Using Claude Code with KECS

Once configured, you can use natural language commands in Claude Code:

### Quick Commands
- "List all clusters in KECS"
- "Show me the services in the default cluster"
- "Create a new service called web-app"
- "Describe the nginx task definition"

### Complex Operations
- "Create a new task definition for a Node.js application with 512MB memory and 256 CPU units"
- "Update the web-service to use the latest task definition and scale to 3 instances"
- "Show me all tasks that are currently running in the production cluster"

### Development Workflows
- "Set up a new microservice with nginx as a reverse proxy"
- "Deploy a Redis cache service as a daemon on all container instances"
- "Create a batch job task definition that runs every hour"

## VS Code Integration Features

### Command Palette

Access KECS operations through the VS Code command palette:
1. Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS)
2. Type your KECS-related query
3. Claude Code will execute the appropriate MCP tools

### Inline Assistance

Get help while editing:
- Task definition JSON files
- Service configuration
- Deployment scripts

### Code Generation

Claude Code can generate:
- Task definition templates
- Service deployment configurations
- Infrastructure as code snippets

## Debugging

### Enable Debug Logging

Set the log level to debug in your configuration:

```json
{
  "env": {
    "KECS_API_URL": "http://localhost:8080",
    "LOG_LEVEL": "debug"
  }
}
```

### View Logs

To see MCP server logs in VS Code:

1. Open the Output panel (`View > Output`)
2. Select the appropriate output channel
3. Look for KECS MCP server messages

### Developer Tools

Access VS Code Developer Tools:
1. `Help > Toggle Developer Tools`
2. Check the Console tab for MCP-related messages
3. Look for errors or warnings

## Environment Variables

Configure these environment variables in the MCP server configuration:

- `KECS_API_URL` - KECS API endpoint (default: `http://localhost:8080`)
- `KECS_API_TIMEOUT` - Request timeout in milliseconds (default: `30000`)
- `LOG_LEVEL` - Logging level: `error`, `warn`, `info`, `debug` (default: `info`)
- `NODE_ENV` - Environment: `production`, `development` (default: `development`)
- `KECS_API_TOKEN` - Authentication token (if required)

## Troubleshooting

### MCP Server Not Available

1. **Verify the configuration file**
   - Check JSON syntax
   - Ensure correct file location
   - Use absolute paths

2. **Check the build**
   ```bash
   cd /path/to/kecs/mcp-server
   ls dist/index.js  # Should exist
   ```

3. **Test manually**
   ```bash
   node /path/to/kecs/mcp-server/dist/index.js
   ```

### Connection Issues

1. **Verify KECS is running**
   ```bash
   curl http://localhost:8080/health
   ```

2. **Check network connectivity**
   - Firewall settings
   - Port availability
   - Docker networking (if applicable)

### Performance Issues

1. **Optimize configuration**
   - Adjust `KECS_API_TIMEOUT` for slow networks
   - Use appropriate `LOG_LEVEL` for production

2. **Monitor resources**
   - Check CPU and memory usage
   - Review VS Code performance

## Best Practices

1. **Use environment-specific configurations**
   - Separate development and production settings
   - Store sensitive data securely

2. **Keep the MCP server updated**
   - Regular updates for bug fixes
   - New features and improvements

3. **Monitor logs**
   - Review logs for errors
   - Track usage patterns

## Next Steps

- [Usage Examples](./examples.md) - See practical examples
- [API Reference](./api-reference.md) - Detailed tool documentation
- [Troubleshooting Guide](./troubleshooting.md) - Common issues and solutions