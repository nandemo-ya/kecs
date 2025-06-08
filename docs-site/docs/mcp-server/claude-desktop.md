---
sidebar_position: 3
---

# Claude Desktop Setup

Configure the KECS MCP server for use with Claude Desktop application.

## Configuration File Location

Claude Desktop uses a configuration file to specify MCP servers. The location depends on your operating system:

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/claude/claude_desktop_config.json`

## Basic Configuration

### 1. Create or Edit the Configuration File

Open the configuration file in your preferred text editor and add the KECS MCP server:

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

### 2. Restart Claude Desktop

After saving the configuration file, restart Claude Desktop for the changes to take effect.

## Advanced Configurations

### Multiple Environments

You can configure multiple KECS environments:

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
        "KECS_API_URL": "https://staging.example.com",
        "KECS_API_TOKEN": "staging-auth-token",
        "LOG_LEVEL": "warn"
      }
    },
    "kecs-production": {
      "command": "node",
      "args": ["/path/to/kecs/mcp-server/dist/index.js"],
      "env": {
        "KECS_API_URL": "https://api.example.com",
        "KECS_API_TOKEN": "production-auth-token",
        "LOG_LEVEL": "error"
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
      "cwd": "/path/to/kecs/mcp-server",
      "env": {
        "KECS_API_URL": "http://localhost:8080",
        "LOG_LEVEL": "debug",
        "NODE_ENV": "development"
      }
    }
  }
}
```

### With Authentication

If your KECS instance requires authentication:

```json
{
  "mcpServers": {
    "kecs-auth": {
      "command": "node",
      "args": ["/path/to/kecs/mcp-server/dist/index.js"],
      "env": {
        "KECS_API_URL": "https://kecs.example.com",
        "KECS_API_TOKEN": "your-auth-token-here",
        "KECS_API_SSL_VERIFY": "true",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

## Usage Examples

Once configured, you can interact with KECS using natural language in Claude Desktop:

### Cluster Operations
- "List all ECS clusters"
- "Create a new cluster called production"
- "Show me details about the default cluster"
- "Delete the test cluster"

### Service Management
- "List all services in the default cluster"
- "Create a new service running nginx"
- "Scale the web-app service to 5 instances"
- "Update the api service to use the latest task definition"

### Task Operations
- "Show me all running tasks"
- "Run a new task using the batch-job task definition"
- "Stop task with ID 12345"
- "Describe the tasks in the web-app service"

### Task Definitions
- "List all task definitions"
- "Show me the nginx:latest task definition"
- "Register a new task definition for my application"
- "Deregister the old-app:1 task definition"

## Troubleshooting

### MCP Server Not Available

If the KECS MCP server doesn't appear in Claude Desktop:

1. **Check the configuration file**
   - Ensure the JSON syntax is valid
   - Verify the file is in the correct location
   - Confirm absolute paths are used

2. **Verify the MCP server is built**
   ```bash
   cd /path/to/kecs/mcp-server
   npm run build
   ls dist/index.js  # Should exist
   ```

3. **Test the MCP server manually**
   ```bash
   node /path/to/kecs/mcp-server/dist/index.js
   ```
   You should see the server start without errors.

### Connection Errors

If Claude can connect to the MCP server but operations fail:

1. **Check KECS is running**
   ```bash
   curl http://localhost:8080/health
   ```

2. **Verify environment variables**
   - Ensure `KECS_API_URL` matches your KECS instance
   - Check authentication tokens if required

3. **Enable debug logging**
   Set `"LOG_LEVEL": "debug"` in the configuration

### Viewing Logs

To see MCP server logs in Claude Desktop:

1. Open Claude Desktop's developer tools (if available)
2. Check the console for MCP-related messages
3. Look for errors or warnings from the KECS MCP server

## Security Considerations

- Store authentication tokens securely
- Use HTTPS for production KECS instances
- Limit MCP server access to trusted environments
- Regularly update the MCP server for security patches

## Next Steps

- [Claude Code Setup](./claude-code.md) - Configure VS Code integration
- [Usage Examples](./examples.md) - See more usage patterns
- [API Reference](./api-reference.md) - Detailed tool documentation