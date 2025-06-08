---
sidebar_position: 2
---

# Installation Guide

This guide walks you through installing and setting up the KECS MCP server.

## Prerequisites

- Node.js 18 or higher
- npm (comes with Node.js)
- Running KECS instance

## Installation Steps

### 1. Clone the Repository

```bash
git clone https://github.com/nandemo-ya/kecs.git
cd kecs/mcp-server
```

### 2. Install Dependencies

```bash
npm install
```

### 3. Build the Server

```bash
npm run build
```

This compiles the TypeScript code to JavaScript in the `dist/` directory.

### 4. Configure Environment

Create a `.env` file from the example:

```bash
cp .env.example .env
```

Edit `.env` to match your KECS setup:

```bash
# KECS API endpoint
KECS_API_URL=http://localhost:8080

# API request timeout in milliseconds
KECS_API_TIMEOUT=30000

# Logging level: error, warn, info, debug
LOG_LEVEL=info
```

## Verification

### Test the MCP Server

You can verify the installation using the MCP Inspector:

```bash
# Install MCP Inspector globally
npm install -g @modelcontextprotocol/inspector

# Run the inspector
npx mcp-inspector node dist/index.js
```

The inspector provides a web interface to test your MCP server's tools.

### Test KECS Connection

Ensure KECS is running and accessible:

```bash
curl http://localhost:8080/health
```

You should see a response indicating KECS is healthy.

## Development Mode

For development with hot-reloading:

```bash
npm run dev
```

This watches for file changes and automatically restarts the server.

## Building for Production

For production deployment:

```bash
# Clean build
npm run build

# Run production server
NODE_ENV=production node dist/index.js
```

## Docker Support

You can also run the MCP server in Docker:

```dockerfile
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY dist ./dist
ENV NODE_ENV=production
CMD ["node", "dist/index.js"]
```

## Troubleshooting

### Common Issues

1. **Build Errors**
   - Ensure Node.js 18+ is installed
   - Run `npm install` to get all dependencies
   - Check for TypeScript errors: `npm run typecheck`

2. **Connection Failed**
   - Verify KECS is running on the configured port
   - Check `KECS_API_URL` in your `.env` file
   - Look for firewall or network issues

3. **Permission Errors**
   - Ensure you have write permissions in the project directory
   - On Unix systems, you may need to use `sudo` for global installs

### Getting Help

- Check the [troubleshooting guide](./troubleshooting.md)
- Open an issue on [GitHub](https://github.com/nandemo-ya/kecs/issues)
- Review the [API documentation](./api-reference.md)

## Next Steps

- [Claude Desktop Setup](./claude-desktop.md) - Configure Claude Desktop
- [Claude Code Setup](./claude-code.md) - Configure VS Code integration
- [Usage Examples](./examples.md) - See the MCP server in action