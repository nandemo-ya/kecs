---
sidebar_position: 5
---

# Usage Examples

Practical examples of using the KECS MCP server with Claude Desktop or Claude Code.

## Cluster Management

### Creating and Managing Clusters

```
User: "Create a new ECS cluster called production"
Claude: I'll create a new ECS cluster called "production" for you.

[Creates cluster]

The production cluster has been created successfully with:
- ARN: arn:aws:ecs:region:account:cluster/production
- Status: ACTIVE
```

### Viewing Cluster Information

```
User: "Show me all clusters with their statistics"
Claude: I'll list all ECS clusters with their statistics.

[Lists clusters with details]

Found 2 clusters:
1. default
   - Active Services: 3
   - Running Tasks: 7
   - Container Instances: 2

2. production
   - Active Services: 0
   - Running Tasks: 0
   - Container Instances: 0
```

## Service Deployment

### Creating a Web Service

```
User: "Deploy an nginx web server with 3 replicas"
Claude: I'll help you deploy an nginx web server with 3 replicas.

First, let me check if there's an nginx task definition available.
[Checks task definitions]

I found nginx:latest task definition. Now I'll create the service.
[Creates service]

The nginx service has been created successfully:
- Service Name: nginx-web
- Desired Count: 3
- Task Definition: nginx:latest
- Status: ACTIVE
```

### Updating Service Configuration

```
User: "Scale the web-app service to 5 instances and force a new deployment"
Claude: I'll scale the web-app service to 5 instances and force a new deployment.

[Updates service]

The web-app service has been updated:
- Previous Desired Count: 3
- New Desired Count: 5
- Deployment Status: IN_PROGRESS
- Force New Deployment: Enabled
```

## Task Management

### Running One-off Tasks

```
User: "Run a database migration task"
Claude: I'll run the database migration task for you.

[Checks for migration task definition]
[Runs task]

The migration task has been started:
- Task ID: arn:aws:ecs:region:account:task/cluster/abc123
- Task Definition: db-migration:1
- Status: RUNNING
- Started At: 2024-01-15 10:30:00
```

### Monitoring Tasks

```
User: "Show me all running tasks in the production cluster"
Claude: I'll show you all running tasks in the production cluster.

[Lists tasks]

Found 5 running tasks in the production cluster:
1. web-app-task-1 (nginx:latest) - Running for 2 hours
2. web-app-task-2 (nginx:latest) - Running for 2 hours
3. api-service-task-1 (api:v2.1) - Running for 45 minutes
4. api-service-task-2 (api:v2.1) - Running for 45 minutes
5. worker-task-1 (worker:latest) - Running for 1 hour
```

## Task Definition Management

### Creating a Complex Task Definition

```
User: "Create a task definition for a Node.js API with 1GB memory, 512 CPU units, 
       environment variables for DATABASE_URL and API_KEY, and a health check"

Claude: I'll create a comprehensive task definition for your Node.js API.

[Creates task definition with all specifications]

The task definition has been registered:
- Family: nodejs-api
- Revision: 1
- Container Configuration:
  - Image: node:18-alpine
  - Memory: 1024 MB
  - CPU: 512 units
  - Environment Variables: DATABASE_URL, API_KEY
  - Health Check: HTTP check on port 3000
```

### Managing Task Definition Versions

```
User: "List all revisions of the web-app task definition"
Claude: I'll list all revisions of the web-app task definition.

[Lists task definitions]

Found 4 revisions of web-app:
1. web-app:1 (INACTIVE) - Created: 2024-01-01
2. web-app:2 (INACTIVE) - Created: 2024-01-05
3. web-app:3 (ACTIVE) - Created: 2024-01-10
4. web-app:4 (ACTIVE) - Created: 2024-01-15 (Latest)
```

## Complex Workflows

### Blue-Green Deployment

```
User: "Help me do a blue-green deployment for the api service"
Claude: I'll help you perform a blue-green deployment for the api service.

Step 1: Checking current service configuration...
[Describes current service]
Current: api service running api:v1.0 with 3 tasks

Step 2: Creating new task definition revision...
[Registers new task definition api:v2.0]

Step 3: Creating green service...
[Creates api-green service with new task definition]

Step 4: Verifying green service is healthy...
[Monitors tasks until healthy]
All 3 tasks are running and healthy.

Step 5: Switching traffic (updating original service)...
[Updates api service to use api:v2.0]

Step 6: Cleaning up green service...
[Deletes api-green service]

Blue-green deployment completed successfully!
```

### Troubleshooting Failed Deployments

```
User: "My web service keeps failing to start. Can you help me debug?"
Claude: I'll help you debug the web service issues.

Let me first check the service status:
[Describes service]
Service "web" shows 0/3 running tasks with multiple failed deployments.

Checking recent task failures:
[Lists stopped tasks]
Found 5 stopped tasks in the last hour.

Examining task details:
[Describes failed tasks]
All tasks failed with: "Essential container exited: code 1"

Looking at the task definition:
[Describes task definition]
The container is trying to bind to port 80, but the image runs as non-root.

Recommendation: Update the task definition to either:
1. Use port 8080 (or another high port)
2. Add capabilities to bind to privileged ports
3. Use a root user (not recommended for security)

Would you like me to update the task definition for you?
```

### Multi-Service Application Deployment

```
User: "Deploy a complete microservices application with frontend, API, and Redis cache"

Claude: I'll help you deploy a complete microservices application.

Step 1: Setting up Redis cache service
[Creates Redis task definition]
[Creates Redis service as daemon]
Redis cache deployed on all container instances.

Step 2: Deploying API service
[Creates API task definition with Redis connection]
[Creates API service with 2 replicas]
API service deployed and connected to Redis.

Step 3: Deploying frontend service
[Creates frontend task definition]
[Creates frontend service with 3 replicas]
Frontend service deployed with load balancer.

Application stack deployed successfully:
- Redis Cache: Running on all instances
- API Service: 2 replicas running
- Frontend: 3 replicas running with load balancer

All services are healthy and connected!
```

## Monitoring and Maintenance

### Resource Usage Analysis

```
User: "Show me which services are using the most resources"
Claude: I'll analyze resource usage across your services.

[Lists all services and tasks]
[Analyzes resource allocation]

Resource Usage Summary:
1. data-processor service
   - 5 tasks × 2048 MB = 10.24 GB memory
   - 5 tasks × 1024 CPU = 5120 CPU units

2. web-app service
   - 10 tasks × 512 MB = 5.12 GB memory
   - 10 tasks × 256 CPU = 2560 CPU units

3. api service
   - 3 tasks × 1024 MB = 3.07 GB memory
   - 3 tasks × 512 CPU = 1536 CPU units

Total cluster usage:
- Memory: 18.43 GB / 32 GB (57.6%)
- CPU: 9216 / 16384 units (56.3%)
```

### Cleanup Operations

```
User: "Clean up all stopped tasks and unused task definitions"
Claude: I'll help you clean up stopped tasks and unused task definitions.

Step 1: Removing stopped tasks...
[Lists stopped tasks]
Found 23 stopped tasks older than 1 hour.
[Removes stopped tasks]
All stopped tasks have been removed.

Step 2: Identifying unused task definitions...
[Lists all task definitions]
[Checks which are in use by services]

Found 15 unused task definition revisions:
- app:1 through app:5 (superseded by app:6)
- test-app:1 through test-app:10 (no active services)

Step 3: Deregistering unused definitions...
[Deregisters each unused definition]

Cleanup completed:
- Removed: 23 stopped tasks
- Deregistered: 15 task definitions
- Freed resources: ~5GB memory, 2048 CPU units
```

## Best Practices Examples

### Setting Up Development Environment

```
User: "Set up a development environment that mirrors production"
Claude: I'll help you set up a development environment that mirrors your production setup.

Analyzing production environment...
[Describes production cluster]

Creating development cluster:
[Creates dev cluster]

Copying task definitions:
[Registers task definitions with dev- prefix]

Creating services with reduced capacity:
- web-app: 1 replica (prod has 5)
- api: 1 replica (prod has 3)
- worker: 1 replica (prod has 2)

Development environment created successfully:
- Cluster: development
- Services: 3 (matching production)
- Total resources: 20% of production
- Cost savings: ~80%
```

## Next Steps

- [API Reference](./api-reference.md) - Detailed documentation of all tools
- [Troubleshooting Guide](./troubleshooting.md) - Common issues and solutions
- [Best Practices](./best-practices.md) - Recommendations for production use