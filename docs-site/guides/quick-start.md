# Quick Start Guide

This guide will walk you through creating your first ECS cluster and deploying a simple application on KECS.

## Overview

In this tutorial, you will:
1. Create an ECS cluster
2. Register a task definition
3. Create and run a service
4. Access your application

## Prerequisites

- KECS is installed and running (see [Getting Started](/guides/getting-started))
- AWS CLI configured (for using ECS commands)

## Step 1: Create a Cluster

First, let's create an ECS cluster:

```bash
# Using AWS CLI
aws ecs create-cluster --cluster-name my-first-cluster \
  --endpoint-url http://localhost:8080

# Or using curl
curl -X POST http://localhost:8080/v1/CreateCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster" \
  -d '{
    "clusterName": "my-first-cluster"
  }'
```

Expected response:
```json
{
  "cluster": {
    "clusterArn": "arn:aws:ecs:ap-northeast-1:000000000000:cluster/my-first-cluster",
    "clusterName": "my-first-cluster",
    "status": "ACTIVE"
  }
}
```

## Step 2: Register a Task Definition

Create a file called `nginx-task.json`:

```json
{
  "family": "nginx-web",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "nginx",
      "image": "nginx:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "protocol": "tcp"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/nginx-web",
          "awslogs-region": "ap-northeast-1",
          "awslogs-stream-prefix": "nginx"
        }
      }
    }
  ]
}
```

Register the task definition:

```bash
# Using AWS CLI
aws ecs register-task-definition \
  --cli-input-json file://nginx-task.json \
  --endpoint-url http://localhost:8080

# Or using curl
curl -X POST http://localhost:8080/v1/RegisterTaskDefinition \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition" \
  -d @nginx-task.json
```

## Step 3: Create a Service

Now, let's create a service that will run our task:

```bash
# Using AWS CLI
aws ecs create-service \
  --cluster my-first-cluster \
  --service-name nginx-service \
  --task-definition nginx-web:1 \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}" \
  --endpoint-url http://localhost:8080
```

Or create a service definition file `nginx-service.json`:

```json
{
  "cluster": "my-first-cluster",
  "serviceName": "nginx-service",
  "taskDefinition": "nginx-web:1",
  "desiredCount": 2,
  "launchType": "FARGATE",
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345"],
      "securityGroups": ["sg-12345"],
      "assignPublicIp": "ENABLED"
    }
  }
}
```

And create the service:

```bash
curl -X POST http://localhost:8080/v1/CreateService \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateService" \
  -d @nginx-service.json
```

## Step 4: Verify the Deployment

### Check Service Status

```bash
# List services
aws ecs list-services --cluster my-first-cluster \
  --endpoint-url http://localhost:8080

# Describe the service
aws ecs describe-services \
  --cluster my-first-cluster \
  --services nginx-service \
  --endpoint-url http://localhost:8080
```

### Check Running Tasks

```bash
# List tasks
aws ecs list-tasks --cluster my-first-cluster \
  --endpoint-url http://localhost:8080

# Describe tasks
aws ecs describe-tasks \
  --cluster my-first-cluster \
  --tasks <task-arn> \
  --endpoint-url http://localhost:8080
```


## Step 5: Access Your Application

Since KECS runs containers in Kubernetes, you can access your application using kubectl:

```bash
# Get pods
kubectl get pods -n my-first-cluster

# Port forward to access nginx
kubectl port-forward -n my-first-cluster pod/nginx-service-0 8080:80

# Now access nginx at http://localhost:8080
```

## Next Steps

Congratulations! You've successfully deployed your first application on KECS. Here are some things to try next:

### 1. Scale Your Service

```bash
aws ecs update-service \
  --cluster my-first-cluster \
  --service nginx-service \
  --desired-count 3 \
  --endpoint-url http://localhost:8080
```

### 2. Update Your Task Definition

Modify `nginx-task.json` to use a different image or add environment variables, then:

```bash
# Register new revision
aws ecs register-task-definition \
  --cli-input-json file://nginx-task.json \
  --endpoint-url http://localhost:8080

# Update service to use new revision
aws ecs update-service \
  --cluster my-first-cluster \
  --service nginx-service \
  --task-definition nginx-web:2 \
  --endpoint-url http://localhost:8080
```

### 3. Explore Advanced Features

- [Load Balancing](/guides/load-balancing)
- [Service Discovery](/guides/service-discovery)
- [Auto Scaling](/guides/auto-scaling)
- [LocalStack Integration](/guides/localstack-integration)

## Cleanup

When you're done experimenting, clean up your resources:

```bash
# Delete service
aws ecs delete-service \
  --cluster my-first-cluster \
  --service nginx-service \
  --force \
  --endpoint-url http://localhost:8080

# Delete cluster
aws ecs delete-cluster \
  --cluster my-first-cluster \
  --endpoint-url http://localhost:8080
```

## Troubleshooting

### Service Not Starting

Check the task status:
```bash
aws ecs describe-tasks --cluster my-first-cluster \
  --tasks <task-arn> \
  --endpoint-url http://localhost:8080
```

Look for error messages in the `stoppedReason` field.

### Container Logs

View container logs using kubectl:
```bash
kubectl logs -n my-first-cluster <pod-name>
```

### Common Issues

1. **Image Pull Errors**: Ensure the container image is accessible
2. **Resource Constraints**: Check if your Kubernetes cluster has enough resources
3. **Network Issues**: Verify security groups and network configuration

For more help, see our [Troubleshooting Guide](/guides/troubleshooting).