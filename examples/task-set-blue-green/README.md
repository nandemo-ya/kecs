# TaskSet Blue/Green Deployment Example

This example demonstrates how to perform Blue/Green deployments using Amazon ECS TaskSets with KECS.

## Overview

TaskSets enable advanced deployment strategies by allowing multiple versions of an application to run simultaneously within a single ECS service. This example shows:

- Running two TaskSets (Blue and Green) within one service
- Switching traffic between TaskSets
- Rolling back to the previous version
- Managing TaskSet lifecycle

## Architecture

```
┌─────────────────────────────────────────┐
│           ECS Service                   │
│  (EXTERNAL Deployment Controller)       │
├─────────────────┬───────────────────────┤
│   Blue TaskSet  │   Green TaskSet       │
│   (PRIMARY)     │   (Standby)           │
│   Scale: 100%   │   Scale: 0%           │
│   Version: 1.0  │   Version: 2.0        │
│   3 Tasks       │   0 Tasks             │
└─────────────────┴───────────────────────┘
```

## Files

- `task_def_blue.json` - Task definition for Blue version (v1.0.0)
- `task_def_green.json` - Task definition for Green version (v2.0.0)
- `service_def.json` - Service definition with EXTERNAL deployment controller
- `deploy.sh` - Deployment script for Blue/Green operations

## Prerequisites

1. KECS running locally:
```bash
kecs start
```

2. AWS CLI configured to use KECS:
```bash
export AWS_ENDPOINT_URL=http://localhost:8080
export AWS_REGION=us-east-1
```

## Usage

### 1. Initial Setup

Create the cluster, service, and both TaskSets:

```bash
./deploy.sh setup
```

This will:
- Create an ECS cluster
- Register both Blue and Green task definitions
- Create a service with EXTERNAL deployment controller
- Create Blue TaskSet as PRIMARY (100% scale)
- Create Green TaskSet in standby (0% scale)

### 2. Check Status

View the current TaskSet configuration:

```bash
./deploy.sh status
```

Output shows:
- External ID (blue-deployment/green-deployment)
- Status (ACTIVE/PRIMARY/DRAINING)
- Scale percentage
- Desired/Running/Pending task counts

### 3. Deploy Green Version

Switch from Blue to Green:

```bash
./deploy.sh deploy
```

This performs:
1. Scale Green TaskSet to 100%
2. Wait for Green tasks to stabilize
3. Set Green TaskSet as PRIMARY
4. Scale Blue TaskSet to 0%

### 4. Rollback to Blue

If issues occur, rollback to Blue:

```bash
./deploy.sh rollback
```

This reverses the deployment:
1. Scale Blue TaskSet to 100%
2. Wait for Blue tasks to stabilize
3. Set Blue TaskSet as PRIMARY
4. Scale Green TaskSet to 0%

### 5. Cleanup

Remove all resources:

```bash
./deploy.sh cleanup
```

## Key Concepts

### TaskSet States

- **PROVISIONING**: TaskSet is being created
- **ACTIVE**: TaskSet is running and can serve traffic
- **PRIMARY**: TaskSet is the primary traffic target
- **DRAINING**: TaskSet is being removed from service

### Scale Management

TaskSet scale is expressed as a percentage of the service's desired count:
- `100%` - Full capacity (all traffic)
- `50%` - Half capacity (canary deployment)
- `0%` - Standby (no traffic)

### Traffic Switching

The PRIMARY TaskSet receives production traffic. Only one TaskSet can be PRIMARY at a time.

## Advanced Scenarios

### Canary Deployment

For gradual rollout, scale both TaskSets partially:

```bash
# 90% Blue, 10% Green (Canary)
aws ecs update-task-set --cluster default --service webapp-service \
    --task-set $BLUE_TASKSET --scale "value=90,unit=PERCENT"

aws ecs update-task-set --cluster default --service webapp-service \
    --task-set $GREEN_TASKSET --scale "value=10,unit=PERCENT"
```

### A/B Testing

Run both versions at 50% for A/B testing:

```bash
# 50% Blue, 50% Green
aws ecs update-task-set --cluster default --service webapp-service \
    --task-set $BLUE_TASKSET --scale "value=50,unit=PERCENT"

aws ecs update-task-set --cluster default --service webapp-service \
    --task-set $GREEN_TASKSET --scale "value=50,unit=PERCENT"
```

## Monitoring

Monitor TaskSet deployment:

```bash
# Watch TaskSet status
watch -n 2 "./deploy.sh status"

# Check running pods (if using KECS)
kubectl get pods -n default-us-east-1 -l kecs.io/service=webapp-service

# View logs
aws logs tail /kecs/webapp --follow
```

## Troubleshooting

### TaskSet Not Scaling

If tasks don't start:
1. Check task definition is valid
2. Verify network configuration
3. Review task logs for errors

### Service Not Found

Ensure service was created with EXTERNAL deployment controller:
```bash
aws ecs describe-services --cluster default --services webapp-service \
    --query 'services[0].deploymentController'
```

### Tasks Failing Health Checks

Review health check configuration in task definition:
- Increase `startPeriod` for slow-starting applications
- Adjust `interval` and `timeout` values
- Check application logs for startup errors

## Notes

- This example uses FARGATE launch type
- Network configuration uses dummy subnet/security group IDs
- In production, use proper VPC configuration
- Task definitions use different nginx versions to simulate version changes