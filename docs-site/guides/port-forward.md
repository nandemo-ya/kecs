# Port Forwarding Guide

KECS provides a powerful port forwarding system that enables local access to services and tasks running in your KECS clusters. This feature is equivalent to AWS ECS's `assignPublicIp` functionality, making it seamless to develop and test locally.

## Overview

The port-forward feature allows you to:
- 🌐 Access ECS services running in KECS clusters from your local machine
- 🔄 Automatically reconnect when connections are lost
- 📦 Forward ports for both services and individual tasks
- 🏷️ Use tags to dynamically select tasks
- 💾 Persist configurations across KECS restarts
- 🔀 Support for both NodePort and LoadBalancer service types

## Quick Start

Let's forward a simple nginx service to your local machine:

```bash
# 1. Deploy a service with public IP enabled
cat > nginx-service.json <<EOF
{
  "serviceName": "nginx",
  "taskDefinition": "nginx:1",
  "desiredCount": 1,
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345678"],
      "assignPublicIp": "ENABLED"
    }
  }
}
EOF

aws ecs create-service --cli-input-json file://nginx-service.json

# 2. Forward the service to local port 8080
kecs port-forward start service default/nginx --local-port 8080

# 3. Access your service
curl http://localhost:8080
```

That's it! Your nginx service is now accessible locally.

## Basic Usage

### Starting Port Forwards

#### For Services

Forward a service with automatic port assignment:
```bash
kecs port-forward start service <cluster>/<service-name>
```

Forward with specific ports:
```bash
kecs port-forward start service default/web --local-port 3000 --target-port 80
```

#### For Tasks

Forward a specific task:
```bash
kecs port-forward start task <cluster>/<task-id> --local-port 9000
```

Forward using tags (automatically selects the newest matching task):
```bash
kecs port-forward start task default --tags app=api,version=v2
```

### Managing Port Forwards

List all active forwards:
```bash
kecs port-forward list
```

Example output:
```
ID                           TYPE     CLUSTER     TARGET        LOCAL   TARGET   STATUS
svc-default-nginx-1234       service  default     nginx         8080    80       active
task-default-api-5678        task     default     api-task      9000    8080     active
```

Stop a specific forward:
```bash
kecs port-forward stop svc-default-nginx-1234
```

Stop all forwards:
```bash
kecs port-forward stop --all
```

## Tutorial: Web Application Development

Let's walk through a typical development workflow with a multi-tier application.

### Step 1: Deploy Your Application Stack

```bash
# Deploy frontend service
aws ecs create-service \
  --service-name frontend \
  --task-definition frontend:1 \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],assignPublicIp=ENABLED}"

# Deploy API service
aws ecs create-service \
  --service-name api \
  --task-definition api:1 \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],assignPublicIp=ENABLED}"

# Deploy service with ALB (LoadBalancer type)
aws ecs create-service \
  --service-name webapp-alb \
  --task-definition webapp:1 \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345678],assignPublicIp=ENABLED}" \
  --load-balancers targetGroupArn=arn:aws:elasticloadbalancing:us-east-1:000000000000:targetgroup/webapp-tg/xxx

# Deploy database (without public IP for security)
aws ecs create-service \
  --service-name database \
  --task-definition postgres:1
```

### Step 2: Set Up Port Forwarding

```bash
# Forward frontend service
kecs port-forward start service default/frontend --local-port 3000 --target-port 3000

# Forward API service
kecs port-forward start service default/api --local-port 8080 --target-port 8080

# Forward ALB service (LoadBalancer type)
kecs port-forward start service default/webapp-alb --local-port 8090 --target-port 80
```

### Step 3: Develop with Live Services

Now you can:
- Access frontend at `http://localhost:3000`
- Make API calls to `http://localhost:8080`
- Access ALB-integrated service at `http://localhost:8090`
- Changes to your services are immediately accessible

### Step 4: Debug a Specific Task

If you need to debug a specific task instance:

```bash
# Find the problematic task
aws ecs list-tasks --cluster default --service-name api

# Forward debug port
kecs port-forward start task default/arn:aws:ecs:task:abc123 \
  --local-port 5005 --target-port 5005

# Connect your debugger to localhost:5005
```

## Advanced Features

### Tag-Based Task Selection

Tags are powerful for dynamic environments:

```bash
# Always forward to the latest canary deployment
kecs port-forward start task default \
  --tags deployment=canary,service=api \
  --local-port 8081
```

When tasks are replaced (e.g., during deployments), the forward automatically switches to the new task.

### Auto-Reconnection

Port forwards automatically reconnect when:
- Network connectivity is lost
- The target pod restarts
- The task is replaced

Monitor reconnection status:
```bash
kecs port-forward list --watch
```

### Multiple Instances

Run multiple KECS instances for different projects or features:

```bash
# Project A instance
kecs start --instance project-a
kecs port-forward start service default/web --local-port 3000

# Project B instance
kecs start --instance project-b
KECS_INSTANCE=project-b kecs port-forward start service default/web --local-port 4000
```


## Troubleshooting

### Service Not Accessible

**Problem**: Can't connect to forwarded port

**Solution**:
1. Check service has `assignPublicIp: ENABLED`
2. Verify the service is running:
   ```bash
   aws ecs describe-services --cluster default --services nginx
   ```
3. Check forward status:
   ```bash
   kecs port-forward list
   ```

### Port Already in Use

**Problem**: Error "port 8080 is already in use"

**Solution**:
1. Check what's using the port:
   ```bash
   lsof -i :8080
   ```
2. Either stop the conflicting process or use a different port:
   ```bash
   kecs port-forward start service default/nginx --local-port 8081
   ```

### Connection Drops Frequently

**Problem**: Forward keeps disconnecting

**Solution**:
1. Check KECS controlplane logs:
   ```bash
   kubectl logs -n kecs-system deployment/kecs-server -f
   ```
2. Ensure stable network connection
3. Verify task isn't being frequently restarted:
   ```bash
   aws ecs describe-tasks --cluster default --tasks <task-id>
   ```

### Forward Not Reconnecting

**Problem**: Auto-reconnect not working

**Solution**:
1. Check if auto-reconnect is enabled:
   ```bash
   kecs port-forward list --format json | jq '.[] | select(.id=="<forward-id>")'
   ```
2. Manually restart the forward:
   ```bash
   kecs port-forward stop <forward-id>
   kecs port-forward start service <cluster>/<service>
   ```

## Best Practices

1. **Use Consistent Port Mappings**: Document your port assignments for team consistency

2. **Reserve Port Ranges**: Establish conventions for port assignments:
   - 3000-3999: Frontend applications
   - 8000-8999: Backend APIs
   - 9000-9999: Admin/monitoring tools
   - 5000-5999: Debug ports

3. **Label Services Clearly**: Use descriptive service names to make port management easier

4. **Monitor Forward Health**: Regularly check `kecs port-forward list` to ensure connections are healthy

5. **Clean Up Unused Forwards**: Stop forwards when not needed to free resources:
   ```bash
   kecs port-forward stop --all
   ```

## Related Documentation

- [Getting Started](./getting-started.md) - Initial KECS setup
- [Services Guide](./services.md) - Managing ECS services
- [CLI Commands](./cli-commands.md) - Complete CLI reference
- [Troubleshooting](./troubleshooting.md) - Common issues and solutions