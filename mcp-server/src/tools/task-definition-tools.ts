import { z } from 'zod';
import { KecsApiClient } from '../client/kecs-api.js';
import { Tool } from './index.js';

const containerDefinitionSchema = z.object({
  name: z.string(),
  image: z.string(),
  cpu: z.number().optional(),
  memory: z.number().optional(),
  memoryReservation: z.number().optional(),
  links: z.array(z.string()).optional(),
  portMappings: z
    .array(
      z.object({
        containerPort: z.number(),
        hostPort: z.number().optional(),
        protocol: z.enum(['tcp', 'udp']).optional(),
      })
    )
    .optional(),
  essential: z.boolean().optional(),
  entryPoint: z.array(z.string()).optional(),
  command: z.array(z.string()).optional(),
  environment: z
    .array(
      z.object({
        name: z.string(),
        value: z.string(),
      })
    )
    .optional(),
  environmentFiles: z
    .array(
      z.object({
        value: z.string(),
        type: z.enum(['s3']),
      })
    )
    .optional(),
  mountPoints: z
    .array(
      z.object({
        sourceVolume: z.string(),
        containerPath: z.string(),
        readOnly: z.boolean().optional(),
      })
    )
    .optional(),
  volumesFrom: z
    .array(
      z.object({
        sourceContainer: z.string(),
        readOnly: z.boolean().optional(),
      })
    )
    .optional(),
  secrets: z
    .array(
      z.object({
        name: z.string(),
        valueFrom: z.string(),
      })
    )
    .optional(),
  healthCheck: z
    .object({
      command: z.array(z.string()),
      interval: z.number().optional(),
      timeout: z.number().optional(),
      retries: z.number().optional(),
      startPeriod: z.number().optional(),
    })
    .optional(),
  logConfiguration: z
    .object({
      logDriver: z.string(),
      options: z.record(z.string()).optional(),
      secretOptions: z
        .array(
          z.object({
            name: z.string(),
            valueFrom: z.string(),
          })
        )
        .optional(),
    })
    .optional(),
});

export function taskDefinitionTools(client: KecsApiClient): Tool[] {
  return [
    {
      name: 'list-task-definitions',
      description: 'List task definitions',
      inputSchema: z.object({
        familyPrefix: z.string().optional(),
        status: z.enum(['ACTIVE', 'INACTIVE']).optional(),
        sort: z.enum(['ASC', 'DESC']).optional(),
        maxResults: z.number().min(1).max(100).optional(),
        nextToken: z.string().optional(),
      }),
      execute: async (args) => {
        const result = await client.listTaskDefinitions(args);
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
      name: 'describe-task-definition',
      description: 'Get detailed information about a task definition',
      inputSchema: z.object({
        taskDefinition: z.string(),
        include: z.array(z.enum(['TAGS'])).optional(),
      }),
      execute: async (args) => {
        const result = await client.describeTaskDefinition(args);
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
      name: 'register-task-definition',
      description: 'Register a new task definition',
      inputSchema: z.object({
        family: z.string(),
        taskRoleArn: z.string().optional(),
        executionRoleArn: z.string().optional(),
        networkMode: z.enum(['bridge', 'host', 'awsvpc', 'none']).optional(),
        containerDefinitions: z.array(containerDefinitionSchema),
        volumes: z
          .array(
            z.object({
              name: z.string(),
              host: z
                .object({
                  sourcePath: z.string().optional(),
                })
                .optional(),
              dockerVolumeConfiguration: z
                .object({
                  scope: z.enum(['task', 'shared']).optional(),
                  autoprovision: z.boolean().optional(),
                  driver: z.string().optional(),
                  driverOpts: z.record(z.string()).optional(),
                  labels: z.record(z.string()).optional(),
                })
                .optional(),
              efsVolumeConfiguration: z
                .object({
                  fileSystemId: z.string(),
                  rootDirectory: z.string().optional(),
                  transitEncryption: z.enum(['ENABLED', 'DISABLED']).optional(),
                  transitEncryptionPort: z.number().optional(),
                  authorizationConfig: z
                    .object({
                      accessPointId: z.string().optional(),
                      iam: z.enum(['ENABLED', 'DISABLED']).optional(),
                    })
                    .optional(),
                })
                .optional(),
            })
          )
          .optional(),
        placementConstraints: z
          .array(
            z.object({
              type: z.enum(['memberOf']),
              expression: z.string().optional(),
            })
          )
          .optional(),
        requiresCompatibilities: z.array(z.enum(['EC2', 'FARGATE'])).optional(),
        cpu: z.string().optional(),
        memory: z.string().optional(),
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
        const result = await client.registerTaskDefinition(args);
        return {
          content: [
            {
              type: 'text',
              text: `Task definition registered successfully:\n${JSON.stringify(result, null, 2)}`,
            },
          ],
        };
      },
    },
    {
      name: 'deregister-task-definition',
      description: 'Deregister a task definition',
      inputSchema: z.object({
        taskDefinition: z.string(),
      }),
      execute: async (args) => {
        const result = await client.deregisterTaskDefinition(args);
        return {
          content: [
            {
              type: 'text',
              text: `Task definition deregistered successfully:\n${JSON.stringify(result, null, 2)}`,
            },
          ],
        };
      },
    },
  ];
}