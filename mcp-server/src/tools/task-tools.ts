import { z } from 'zod';
import { KecsApiClient } from '../client/kecs-api.js';
import { Tool } from './index.js';

export function taskTools(client: KecsApiClient): Tool[] {
  return [
    {
      name: 'list-tasks',
      description: 'List tasks in an ECS cluster',
      inputSchema: z.object({
        cluster: z.string().optional(),
        containerInstance: z.string().optional(),
        family: z.string().optional(),
        serviceName: z.string().optional(),
        desiredStatus: z.enum(['RUNNING', 'PENDING', 'STOPPED']).optional(),
        startedBy: z.string().optional(),
        maxResults: z.number().min(1).max(100).optional(),
        nextToken: z.string().optional(),
      }),
      execute: async (args) => {
        const result = await client.listTasks(args);
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
      name: 'describe-tasks',
      description: 'Get detailed information about one or more ECS tasks',
      inputSchema: z.object({
        cluster: z.string().optional(),
        tasks: z.array(z.string()),
        include: z.array(z.enum(['TAGS'])).optional(),
      }),
      execute: async (args) => {
        const result = await client.describeTasks(args);
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
      name: 'run-task',
      description: 'Run a new task from a task definition',
      inputSchema: z.object({
        cluster: z.string().optional(),
        taskDefinition: z.string(),
        count: z.number().min(1).max(10).optional(),
        startedBy: z.string().optional(),
        group: z.string().optional(),
        launchType: z.enum(['EC2', 'FARGATE', 'EXTERNAL']).optional(),
        platformVersion: z.string().optional(),
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
        overrides: z
          .object({
            containerOverrides: z
              .array(
                z.object({
                  name: z.string(),
                  command: z.array(z.string()).optional(),
                  environment: z
                    .array(
                      z.object({
                        name: z.string(),
                        value: z.string(),
                      })
                    )
                    .optional(),
                  cpu: z.number().optional(),
                  memory: z.number().optional(),
                  memoryReservation: z.number().optional(),
                })
              )
              .optional(),
            cpu: z.string().optional(),
            memory: z.string().optional(),
            taskRoleArn: z.string().optional(),
            executionRoleArn: z.string().optional(),
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
        const result = await client.runTask(args);
        return {
          content: [
            {
              type: 'text',
              text: `Task(s) started successfully:\n${JSON.stringify(result, null, 2)}`,
            },
          ],
        };
      },
    },
    {
      name: 'stop-task',
      description: 'Stop a running ECS task',
      inputSchema: z.object({
        cluster: z.string().optional(),
        task: z.string(),
        reason: z.string().optional(),
      }),
      execute: async (args) => {
        const result = await client.stopTask(args);
        return {
          content: [
            {
              type: 'text',
              text: `Task stopped successfully:\n${JSON.stringify(result, null, 2)}`,
            },
          ],
        };
      },
    },
  ];
}