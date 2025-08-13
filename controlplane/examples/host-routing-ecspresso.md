# Host-Based Routing with ecspresso

This document shows how to use host-based routing with ecspresso and KECS for multi-environment deployments.

## Use Case: Multi-Environment Application

Deploy the same application to different environments using host-based routing.

### 1. Directory Structure

```
myapp/
├── ecspresso.yml
├── ecs-task-def.json
├── ecs-service-def.json
├── environments/
│   ├── dev/
│   │   └── config.yaml
│   ├── staging/
│   │   └── config.yaml
│   └── prod/
│       └── config.yaml
└── alb-rules/
    ├── dev-rule.json
    ├── staging-rule.json
    └── prod-rule.json
```

### 2. ecspresso Configuration

**ecspresso.yml:**
```yaml
region: us-east-1
cluster: myapp-cluster
service: myapp-{{ .Env }}
task_definition: ecs-task-def.json
service_definition: ecs-service-def.json
plugins:
  - name: tfstate
config_files:
  - environments/{{ .Env }}/config.yaml
```

### 3. Environment-Specific Configurations

**environments/dev/config.yaml:**
```yaml
app_image: myapp:dev-latest
app_host: dev.myapp.com
target_group_arn: arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/myapp-dev/73e2d6bc24d8a067
cpu: "256"
memory: "512"
desired_count: 1
```

**environments/staging/config.yaml:**
```yaml
app_image: myapp:staging-latest
app_host: staging.myapp.com
target_group_arn: arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/myapp-staging/83e2d6bc24d8a067
cpu: "512"
memory: "1024"
desired_count: 2
```

**environments/prod/config.yaml:**
```yaml
app_image: myapp:v1.2.3
app_host: www.myapp.com
target_group_arn: arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/myapp-prod/93e2d6bc24d8a067
cpu: "1024"
memory: "2048"
desired_count: 3
```

### 4. Task Definition Template

**ecs-task-def.json:**
```json
{
  "family": "myapp-{{ .Env }}",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "{{ .cpu }}",
  "memory": "{{ .memory }}",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "{{ .app_image }}",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "APP_ENV",
          "value": "{{ .Env }}"
        },
        {
          "name": "APP_HOST",
          "value": "{{ .app_host }}"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/myapp-{{ .Env }}",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

### 5. Service Definition Template

**ecs-service-def.json:**
```json
{
  "serviceName": "myapp-{{ .Env }}",
  "taskDefinition": "myapp-{{ .Env }}",
  "desiredCount": {{ .desired_count }},
  "launchType": "FARGATE",
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": [
        "subnet-12345678",
        "subnet-87654321"
      ],
      "securityGroups": [
        "sg-12345678"
      ],
      "assignPublicIp": "ENABLED"
    }
  },
  "loadBalancers": [
    {
      "targetGroupArn": "{{ .target_group_arn }}",
      "containerName": "app",
      "containerPort": 8080
    }
  ],
  "healthCheckGracePeriodSeconds": 60
}
```

### 6. Create ALB Rules for Each Environment

**Create shared ALB and listener:**
```bash
# Create ALB
aws elbv2 create-load-balancer \
    --name myapp-alb \
    --subnets subnet-12345678 subnet-87654321 \
    --security-groups sg-12345678 \
    --endpoint-url http://localhost:8080

# Create HTTPS listener
aws elbv2 create-listener \
    --load-balancer-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/myapp-alb/50dc6c495c0c9188 \
    --protocol HTTPS \
    --port 443 \
    --certificates CertificateArn=arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012 \
    --default-actions Type=fixed-response,FixedResponseConfig="{StatusCode=404}" \
    --endpoint-url http://localhost:8080
```

**Create target groups:**
```bash
# Dev target group
aws elbv2 create-target-group \
    --name myapp-dev \
    --protocol HTTP \
    --port 8080 \
    --vpc-id vpc-12345678 \
    --target-type ip \
    --health-check-path /health \
    --endpoint-url http://localhost:8080

# Staging target group
aws elbv2 create-target-group \
    --name myapp-staging \
    --protocol HTTP \
    --port 8080 \
    --vpc-id vpc-12345678 \
    --target-type ip \
    --health-check-path /health \
    --endpoint-url http://localhost:8080

# Prod target group
aws elbv2 create-target-group \
    --name myapp-prod \
    --protocol HTTP \
    --port 8080 \
    --vpc-id vpc-12345678 \
    --target-type ip \
    --health-check-path /health \
    --endpoint-url http://localhost:8080
```

**Create host-based routing rules:**
```bash
# Dev environment rule
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/myapp-alb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 10 \
    --conditions Field=host-header,Values="dev.myapp.com" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/myapp-dev/73e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080

# Staging environment rule
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/myapp-alb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 20 \
    --conditions Field=host-header,Values="staging.myapp.com" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/myapp-staging/83e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080

# Production environment rule
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/myapp-alb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 30 \
    --conditions Field=host-header,Values="www.myapp.com,myapp.com" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/myapp-prod/93e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 7. Deploy with ecspresso

**Deploy to dev:**
```bash
export Env=dev
ecspresso deploy --config ecspresso.yml
```

**Deploy to staging:**
```bash
export Env=staging
ecspresso deploy --config ecspresso.yml
```

**Deploy to production:**
```bash
export Env=prod
ecspresso deploy --config ecspresso.yml
```

### 8. Testing Host-Based Routing

```bash
# Test dev environment
curl -H "Host: dev.myapp.com" https://alb-dns-name/
# Response from dev environment

# Test staging environment
curl -H "Host: staging.myapp.com" https://alb-dns-name/
# Response from staging environment

# Test production environment
curl -H "Host: www.myapp.com" https://alb-dns-name/
# Response from production environment
```

## Advanced Patterns

### 1. Blue-Green Deployment with Host Routing

Create temporary host rules for blue-green deployments:

```bash
# Create blue environment rule (new version)
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 5 \
    --conditions Field=host-header,Values="blue.myapp.com" \
    --actions Type=forward,TargetGroupArn=$BLUE_TG_ARN \
    --endpoint-url http://localhost:8080

# Test blue environment
curl -H "Host: blue.myapp.com" https://alb-dns-name/

# Switch production traffic to blue
aws elbv2 modify-rule \
    --rule-arn $PROD_RULE_ARN \
    --actions Type=forward,TargetGroupArn=$BLUE_TG_ARN \
    --endpoint-url http://localhost:8080

# Clean up blue rule
aws elbv2 delete-rule \
    --rule-arn $BLUE_RULE_ARN \
    --endpoint-url http://localhost:8080
```

### 2. A/B Testing with Host and Path

Combine host and path conditions for A/B testing:

```bash
# Version A (default)
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 40 \
    --conditions Field=host-header,Values="www.myapp.com" \
    --actions Type=forward,TargetGroupArn=$VERSION_A_TG \
    --endpoint-url http://localhost:8080

# Version B (beta users)
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 35 \
    --conditions \
        Field=host-header,Values="www.myapp.com" \
        Field=path-pattern,Values="/beta/*" \
    --actions Type=forward,TargetGroupArn=$VERSION_B_TG \
    --endpoint-url http://localhost:8080
```

### 3. Multi-Region Failover

Use host-based routing for region-specific endpoints:

```bash
# US East region
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 50 \
    --conditions Field=host-header,Values="us-east.myapp.com" \
    --actions Type=forward,TargetGroupArn=$US_EAST_TG \
    --endpoint-url http://localhost:8080

# EU West region
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 51 \
    --conditions Field=host-header,Values="eu-west.myapp.com" \
    --actions Type=forward,TargetGroupArn=$EU_WEST_TG \
    --endpoint-url http://localhost:8080
```

## Best Practices

1. **Use priority wisely**: Lower numbers = higher priority
2. **Plan for wildcards**: Place wildcard rules at lower priority
3. **Monitor rule usage**: Check CloudWatch metrics for rule matches
4. **Document your rules**: Keep a registry of all host mappings
5. **Test before DNS changes**: Use curl with Host header to verify routing
6. **Use health checks**: Ensure target groups have proper health checks
7. **SSL certificate planning**: Ensure certificates cover all host names