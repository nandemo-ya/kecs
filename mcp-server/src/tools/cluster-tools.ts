import { z } from 'zod';
import { KecsApiClient } from '../client/kecs-api.js';
import { Tool } from './index.js';

export function listClustersTools(client: KecsApiClient): Tool[] {
  return [
    {
      name: 'list-clusters',
      description: 'List all ECS clusters',
      inputSchema: z.object({
        maxResults: z.number().min(1).max(100).optional(),
        nextToken: z.string().optional(),
      }),
      execute: async (args) => {
        const result = await client.listClusters(args);
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
    {
      name: 'describe-clusters',
      description: 'Get detailed information about one or more ECS clusters',
      inputSchema: z.object({
        clusters: z.array(z.string()).optional(),
        include: z.array(z.enum(['ATTACHMENTS', 'SETTINGS', 'STATISTICS', 'TAGS'])).optional(),
      }),
      execute: async (args) => {
        const result = await client.describeClusters(args);
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
    {
      name: 'create-cluster',
      description: 'Create a new ECS cluster',
      inputSchema: z.object({
        clusterName: z.string(),
        tags: z
          .array(
            z.object({
              key: z.string(),
              value: z.string(),
            })
          )
          .optional(),
      }),
      execute: async (args) => {
        const result = await client.createCluster(args);
        return {
          content: [
            {
              type: 'text',
              text: `Cluster created successfully:\n${JSON.stringify(result, null, 2)}`,
            },
          ],
        };
      },
    },
    {
      name: 'delete-cluster',
      description: 'Delete an ECS cluster',
      inputSchema: z.object({
        cluster: z.string(),
      }),
      execute: async (args) => {
        const result = await client.deleteCluster(args);
        return {
          content: [
            {
              type: 'text',
              text: `Cluster deleted successfully:\n${JSON.stringify(result, null, 2)}`,
            },
          ],
        };
      },
    },
  ];
}