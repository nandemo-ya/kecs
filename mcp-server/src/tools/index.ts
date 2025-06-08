import { z } from 'zod';
import { KecsApiClient } from '../client/kecs-api.js';
import { listClustersTools } from './cluster-tools.js';
import { serviceTools } from './service-tools.js';
import { taskTools } from './task-tools.js';
import { taskDefinitionTools } from './task-definition-tools.js';

export interface Tool {
  name: string;
  description: string;
  inputSchema: z.ZodSchema<any>;
  execute: (args: any) => Promise<any>;
}

export function setupTools(client: KecsApiClient): Map<string, Tool> {
  const tools = new Map<string, Tool>();

  // Register cluster tools
  listClustersTools(client).forEach((tool) => {
    tools.set(tool.name, tool);
  });

  // Register service tools
  serviceTools(client).forEach((tool) => {
    tools.set(tool.name, tool);
  });

  // Register task tools
  taskTools(client).forEach((tool) => {
    tools.set(tool.name, tool);
  });

  // Register task definition tools
  taskDefinitionTools(client).forEach((tool) => {
    tools.set(tool.name, tool);
  });

  return tools;
}