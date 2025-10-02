# ALB Port Mapping Conventions

## Overview

KECS provides automatic port mapping for Application Load Balancers (ALBs) to enable direct local access without requiring `kubectl port-forward`. When you create an ALB listener, KECS automatically configures k3d port mappings that allow you to access your services through fixed local ports.

## Default Port Mappings

KECS instances are created with the following default port mappings:

| ALB Listener Port | Local Host Port | Kubernetes NodePort | Description |
|------------------|-----------------|---------------------|-------------|
| 80 (HTTP)        | 8080           | 30880              | Standard HTTP traffic |
| 443 (HTTPS)      | 8443           | 30443              | Standard HTTPS traffic |

## How It Works

1. **Instance Creation**: When a KECS instance starts, k3d creates a cluster with pre-configured port mappings
2. **Global Traefik**: A global Traefik instance is deployed in the `kecs-system` namespace with NodePort services
3. **ALB Creation**: When you create an ALB, it's registered with the global Traefik
4. **Listener Creation**: When you create an ALB listener, Traefik is configured to route traffic based on the Host header
5. **Direct Access**: You can access your ALB directly through the mapped local port

## Usage Example

```bash
# 1. Start KECS instance (port mappings are automatically configured)
kecs start --instance myapp

# 2. Create an ECS cluster and deploy your service
aws ecs create-cluster --cluster-name myapp-cluster
aws ecs create-service ...

# 3. Create an ALB and listener
aws elbv2 create-load-balancer --name myapp-alb ...
aws elbv2 create-listener --port 80 ...

# 4. Access your service directly (no kubectl port-forward needed!)
curl -H 'Host: myapp-alb.elb.amazonaws.com' http://localhost:8080/
```

## Custom Port Range Configuration

You can configure additional port ranges for custom ALB listeners using environment variables:

```bash
# Add port range 9000-9099 mapped to NodePorts 30800-30899
export KECS_ALB_PORT_RANGE_START=9000
export KECS_ALB_PORT_RANGE_END=9099

# Start KECS with custom port range
kecs start --instance myapp
```

This allows you to:
- Create ALB listeners on custom ports (e.g., 8081, 8082)
- Access them through predictable local ports
- Support multiple ALBs with different port configurations

## Load Balancing

When multiple ECS tasks are running for a service:
- Traffic is automatically load-balanced across all healthy tasks
- The global Traefik instance handles the load distribution
- No additional configuration is required

## Architecture

```
User Request (localhost:8080)
    ↓
k3d Port Mapping (8080→30880)
    ↓
Traefik NodePort Service (30880)
    ↓
Traefik Ingress Controller
    ↓ (Host header routing)
ALB IngressRoute
    ↓
Target Group Service
    ↓
ECS Task Pods (load balanced)
```

## Troubleshooting

### Port Already in Use
If port 8080 or 8443 is already in use on your system:
1. Stop the conflicting service, or
2. Use custom port range configuration to avoid conflicts

### ALB Not Accessible
Verify that:
1. Traefik is running: `kubectl get pods -n kecs-system | grep traefik`
2. NodePort service exists: `kubectl get svc -n kecs-system traefik`
3. Port mapping is configured: `docker inspect k3d-kecs-<instance>-serverlb`

### Host Header Required
Always include the Host header when accessing ALBs locally:
```bash
curl -H 'Host: <alb-dns-name>' http://localhost:8080/
```

## Benefits

1. **Fixed Ports**: Predictable port assignments for consistent local development
2. **No Port Forwarding**: Direct access without `kubectl port-forward`
3. **Load Balancing**: Automatic distribution across multiple tasks
4. **AWS-Compatible**: Mimics AWS ALB behavior for local testing
5. **Multi-Instance**: Each KECS instance can have its own port mappings