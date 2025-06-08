#!/usr/bin/env node
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  ListToolsRequestSchema,
  CallToolRequestSchema,
  ErrorCode,
  McpError,
} from '@modelcontextprotocol/sdk/types.js';
import { KecsApiClient } from './client/kecs-api.js';
import { setupTools } from './tools/index.js';
import { logger } from './utils/logger.js';

// Initialize KECS API client
const kecsClient = new KecsApiClient({
  baseUrl: process.env.KECS_API_URL || 'http://localhost:8080',
  timeout: parseInt(process.env.KECS_API_TIMEOUT || '30000'),
});

// Create MCP server
const server = new Server(
  {
    name: 'kecs-mcp-server',
    version: '0.1.0',
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// Setup available tools
const tools = setupTools(kecsClient);

// Handle list tools request
server.setRequestHandler(ListToolsRequestSchema, async () => {
  logger.info('Listing available tools');
  return {
    tools: Array.from(tools.values()).map((tool) => ({
      name: tool.name,
      description: tool.description,
      inputSchema: tool.inputSchema,
    })),
  };
});

// Handle tool execution
server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;
  logger.info(`Executing tool: ${name}`, { args });

  const tool = tools.get(name);
  if (!tool) {
    throw new McpError(
      ErrorCode.MethodNotFound,
      `Tool not found: ${name}`
    );
  }

  try {
    const result = await tool.execute(args);
    logger.info(`Tool ${name} executed successfully`);
    return result;
  } catch (error) {
    logger.error(`Tool ${name} execution failed:`, error);
    if (error instanceof McpError) {
      throw error;
    }
    throw new McpError(
      ErrorCode.InternalError,
      `Tool execution failed: ${error instanceof Error ? error.message : 'Unknown error'}`
    );
  }
});

// Start the server
async function main() {
  logger.info('Starting KECS MCP Server...');
  
  const transport = new StdioServerTransport();
  await server.connect(transport);
  
  logger.info('KECS MCP Server is running');
  
  // Handle graceful shutdown
  process.on('SIGINT', async () => {
    logger.info('Shutting down KECS MCP Server...');
    await server.close();
    process.exit(0);
  });
}

// Run the server
main().catch((error) => {
  logger.error('Failed to start KECS MCP Server:', error);
  process.exit(1);
});