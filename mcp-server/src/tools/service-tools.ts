import { z } from 'zod';
import { KecsApiClient } from '../client/kecs-api.js';
import { Tool } from './index.js';

export function serviceTools(client: KecsApiClient): Tool[] {
  return [
    {
      name: 'list-services',
      description: 'List services in an ECS cluster',
      inputSchema: z.object({
        cluster: z.string().optional(),
        maxResults: z.number().min(1).max(100).optional(),
        nextToken: z.string().optional(),
      }),
      execute: async (args) => {
        const result = await client.listServices(args);
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
      name: 'describe-services',
      description: 'Get detailed information about one or more ECS services',
      inputSchema: z.object({
        cluster: z.string().optional(),
        services: z.array(z.string()),
        include: z.array(z.enum(['TAGS'])).optional(),
      }),
      execute: async (args) => {
        const result = await client.describeServices(args);
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
      name: 'create-service',
      description: 'Create a new ECS service',
      inputSchema: z.object({
        cluster: z.string().optional(),
        serviceName: z.string(),
        taskDefinition: z.string(),
        desiredCount: z.number().min(0).optional(),
        launchType: z.enum(['EC2', 'FARGATE', 'EXTERNAL']).optional(),
        platformVersion: z.string().optional(),
        deploymentConfiguration: z
          .object({
            maximumPercent: z.number().optional(),
            minimumHealthyPercent: z.number().optional(),
          })
          .optional(),
        networkConfiguration: z
          .object({
            awsvpcConfiguration: z
              .object({
                subnets: z.array(z.string()),
                securityGroups: z.array(z.string()).optional(),
                assignPublicIp: z.enum(['ENABLED', 'DISABLED']).optional(),
              })
              .optional(),
          })
          .optional(),
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
        const result = await client.createService(args);
        return {
          content: [
            {
              type: 'text',
              text: `Service created successfully:\n${JSON.stringify(result, null, 2)}`,
            },
          ],
        };
      },
    },
    {
      name: 'update-service',
      description: 'Update an existing ECS service',
      inputSchema: z.object({
        cluster: z.string().optional(),
        service: z.string(),
        desiredCount: z.number().min(0).optional(),
        taskDefinition: z.string().optional(),
        forceNewDeployment: z.boolean().optional(),
        deploymentConfiguration: z
          .object({
            maximumPercent: z.number().optional(),
            minimumHealthyPercent: z.number().optional(),
          })
          .optional(),
      }),
      execute: async (args) => {
        const result = await client.updateService(args);
        return {
          content: [
            {
              type: 'text',
              text: `Service updated successfully:\n${JSON.stringify(result, null, 2)}`,
            },
          ],
        };
      },
    },
    {
      name: 'delete-service',
      description: 'Delete an ECS service',
      inputSchema: z.object({
        cluster: z.string().optional(),
        service: z.string(),
        force: z.boolean().optional(),
      }),
      execute: async (args) => {
        const result = await client.deleteService(args);
        return {
          content: [
            {
              type: 'text',
              text: `Service deleted successfully:\n${JSON.stringify(result, null, 2)}`,
            },
          ],
        };
      },
    },
  ];
}