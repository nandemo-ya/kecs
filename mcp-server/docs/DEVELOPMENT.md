# KECS MCP Server Development Guide

## Prerequisites

- Node.js 18+ and npm
- TypeScript knowledge
- Understanding of MCP (Model Context Protocol)
- Running KECS instance for testing

## Development Setup

1. **Clone and Install**
```bash
git clone https://github.com/nandemo-ya/kecs.git
cd kecs/mcp-server
npm install
```

2. **Configure Environment**
```bash
cp .env.example .env
# Edit .env with your KECS API endpoint
```

3. **Start Development Server**
```bash
npm run dev
```

## Project Structure

```
mcp-server/
├── src/
│   ├── index.ts           # Main server entry point
│   ├── client/
│   │   └── kecs-api.ts    # KECS API client
│   ├── tools/
│   │   ├── index.ts       # Tool registry
│   │   ├── cluster-tools.ts
│   │   ├── service-tools.ts
│   │   ├── task-tools.ts
│   │   └── task-definition-tools.ts
│   └── utils/
│       └── logger.ts      # Logging utilities
├── docs/
│   ├── API.md            # API documentation
│   └── DEVELOPMENT.md    # This file
├── examples/
│   └── claude-desktop-config.json
└── tests/
    └── ... (test files)
```

## Adding New Tools

1. **Create Tool Definition**

Create a new file in `src/tools/`:
```typescript
// src/tools/my-new-tools.ts
import { z } from 'zod';
import { KecsApiClient } from '../client/kecs-api.js';
import { Tool } from './index.js';

export function myNewTools(client: KecsApiClient): Tool[] {
  return [
    {
      name: 'my-new-tool',
      description: 'Description of what this tool does',
      inputSchema: z.object({
        requiredParam: z.string(),
        optionalParam: z.number().optional(),
      }),
      execute: async (args) => {
        const result = await client.myNewOperation(args);
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(result, null, 2),
            },
          ],
        };
      },
    },
  ];
}
```

2. **Add API Client Method**

Update `src/client/kecs-api.ts`:
```typescript
async myNewOperation(params: MyNewOperationParams): Promise<MyNewOperationResponse> {
  return this.request('/v1/MyNewOperation', params);
}
```

3. **Register Tool**

Update `src/tools/index.ts`:
```typescript
import { myNewTools } from './my-new-tools.js';

export function getAllTools(client: KecsApiClient): Tool[] {
  return [
    ...clusterTools(client),
    ...serviceTools(client),
    ...taskTools(client),
    ...taskDefinitionTools(client),
    ...myNewTools(client), // Add here
  ];
}
```

## Testing

### Unit Tests

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage
```

### Integration Tests

Test against a running KECS instance:
```bash
# Start KECS
cd ../controlplane
make run

# In another terminal, run integration tests
cd mcp-server
npm run test:integration
```

### Manual Testing with MCP Inspector

```bash
# Install MCP Inspector globally
npm install -g @modelcontextprotocol/inspector

# Run the inspector
npx mcp-inspector node dist/index.js
```

## Debugging

### Enable Debug Logging

```bash
LOG_LEVEL=debug npm run dev
```

### VS Code Launch Configuration

Add to `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "type": "node",
      "request": "launch",
      "name": "Debug MCP Server",
      "skipFiles": ["<node_internals>/**"],
      "program": "${workspaceFolder}/dist/index.js",
      "preLaunchTask": "npm: build",
      "env": {
        "LOG_LEVEL": "debug",
        "KECS_API_URL": "http://localhost:8080"
      }
    }
  ]
}
```

### Common Issues

1. **Connection Refused**
   - Ensure KECS is running on the configured port
   - Check `KECS_API_URL` environment variable

2. **Type Errors**
   - Run `npm run build` to check TypeScript compilation
   - Ensure all imports use `.js` extension for ESM compatibility

3. **Tool Not Found**
   - Verify tool is registered in `src/tools/index.ts`
   - Check tool name matches exactly in the implementation

## Best Practices

### 1. Input Validation

Always use Zod schemas for input validation:
```typescript
inputSchema: z.object({
  cluster: z.string().optional(),
  service: z.string().min(1).max(255),
  tags: z.array(z.object({
    key: z.string().regex(/^[\w\-\.\/]+$/),
    value: z.string(),
  })).optional(),
})
```

### 2. Error Handling

Wrap API calls with proper error handling:
```typescript
execute: async (args) => {
  try {
    const result = await client.operation(args);
    return {
      content: [{
        type: 'text',
        text: JSON.stringify(result, null, 2),
      }],
    };
  } catch (error) {
    logger.error('Operation failed', { error, args });
    throw new Error(`Failed to execute operation: ${error.message}`);
  }
}
```

### 3. Response Formatting

Format responses consistently:
```typescript
// For successful operations
return {
  content: [{
    type: 'text',
    text: `Operation completed successfully:\n${JSON.stringify(result, null, 2)}`,
  }],
};

// For lists
return {
  content: [{
    type: 'text',
    text: `Found ${result.items.length} items:\n${JSON.stringify(result, null, 2)}`,
  }],
};
```

### 4. Documentation

- Add JSDoc comments to all exported functions
- Update API.md when adding new tools
- Include examples in tool descriptions

## Release Process

1. **Update Version**
```bash
npm version patch|minor|major
```

2. **Build and Test**
```bash
npm run build
npm test
```

3. **Create Release**
```bash
git tag mcp-server-v$(node -p "require('./package.json').version")
git push origin --tags
```

## Contributing

1. Create a feature branch
2. Add tests for new functionality
3. Ensure all tests pass
4. Update documentation
5. Submit a pull request

See the main KECS [CONTRIBUTING.md](../../CONTRIBUTING.md) for more details.