# Weighted Routing Example

This example demonstrates how to implement weighted routing with ELBv2 and KECS to distribute traffic across multiple target groups based on weights.

## Overview

Weighted routing allows you to distribute traffic across multiple target groups using specified weights. This is useful for:
- Canary deployments (5% to new version, 95% to stable)
- A/B testing with specific traffic distribution
- Gradual rollouts of new features
- Blue-green deployments with controlled traffic shifting

## Prerequisites

1. KECS running with Kubernetes integration
2. A load balancer created
3. Multiple target groups created
4. A listener created

## Creating Weighted Routing Rules

### 1. Basic Weighted Routing (50/50 Split)

Split traffic equally between two versions:

```bash
# Create a rule with weighted forwarding
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 100 \
    --conditions Field=path-pattern,Values="/api/*" \
    --actions Type=forward,ForwardConfig='{
        "TargetGroups": [
            {
                "TargetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v1/73e2d6bc24d8a067",
                "Weight": 50
            },
            {
                "TargetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v2/83e2d6bc24d8a067",
                "Weight": 50
            }
        ]
    }' \
    --endpoint-url http://localhost:8080
```

### 2. Canary Deployment (5% Traffic)

Route a small percentage of traffic to test new version:

```bash
# 5% to canary, 95% to stable
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 90 \
    --conditions Field=path-pattern,Values="/app/*" \
    --actions Type=forward,ForwardConfig='{
        "TargetGroups": [
            {
                "TargetGroupArn": "'$STABLE_TG_ARN'",
                "Weight": 95
            },
            {
                "TargetGroupArn": "'$CANARY_TG_ARN'",
                "Weight": 5
            }
        ]
    }' \
    --endpoint-url http://localhost:8080
```

### 3. Progressive Rollout

Gradually increase traffic to new version:

```bash
# Stage 1: 10% to new version
aws elbv2 modify-rule \
    --rule-arn $RULE_ARN \
    --actions Type=forward,ForwardConfig='{
        "TargetGroups": [
            {
                "TargetGroupArn": "'$STABLE_TG_ARN'",
                "Weight": 90
            },
            {
                "TargetGroupArn": "'$NEW_TG_ARN'",
                "Weight": 10
            }
        ]
    }' \
    --endpoint-url http://localhost:8080

# Stage 2: 25% to new version
aws elbv2 modify-rule \
    --rule-arn $RULE_ARN \
    --actions Type=forward,ForwardConfig='{
        "TargetGroups": [
            {
                "TargetGroupArn": "'$STABLE_TG_ARN'",
                "Weight": 75
            },
            {
                "TargetGroupArn": "'$NEW_TG_ARN'",
                "Weight": 25
            }
        ]
    }' \
    --endpoint-url http://localhost:8080

# Stage 3: 50% to new version
aws elbv2 modify-rule \
    --rule-arn $RULE_ARN \
    --actions Type=forward,ForwardConfig='{
        "TargetGroups": [
            {
                "TargetGroupArn": "'$STABLE_TG_ARN'",
                "Weight": 50
            },
            {
                "TargetGroupArn": "'$NEW_TG_ARN'",
                "Weight": 50
            }
        ]
    }' \
    --endpoint-url http://localhost:8080
```

### 4. Blue-Green Deployment with Weighted Routing

Use weights to control traffic during blue-green deployment:

```bash
# Initially all traffic to blue
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 80 \
    --conditions Field=host-header,Values="app.example.com" \
    --actions Type=forward,ForwardConfig='{
        "TargetGroups": [
            {
                "TargetGroupArn": "'$BLUE_TG_ARN'",
                "Weight": 100
            },
            {
                "TargetGroupArn": "'$GREEN_TG_ARN'",
                "Weight": 0
            }
        ],
        "TargetGroupStickinessConfig": {
            "Enabled": true,
            "DurationSeconds": 3600
        }
    }' \
    --endpoint-url http://localhost:8080

# Switch traffic to green
aws elbv2 modify-rule \
    --rule-arn $RULE_ARN \
    --actions Type=forward,ForwardConfig='{
        "TargetGroups": [
            {
                "TargetGroupArn": "'$BLUE_TG_ARN'",
                "Weight": 0
            },
            {
                "TargetGroupArn": "'$GREEN_TG_ARN'",
                "Weight": 100
            }
        ],
        "TargetGroupStickinessConfig": {
            "Enabled": true,
            "DurationSeconds": 3600
        }
    }' \
    --endpoint-url http://localhost:8080
```

### 5. Multi-Target Group Distribution

Distribute traffic across multiple services:

```bash
# Distribute across 3 target groups
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 70 \
    --conditions Field=path-pattern,Values="/service/*" \
    --actions Type=forward,ForwardConfig='{
        "TargetGroups": [
            {
                "TargetGroupArn": "'$SERVICE_A_TG'",
                "Weight": 40
            },
            {
                "TargetGroupArn": "'$SERVICE_B_TG'",
                "Weight": 35
            },
            {
                "TargetGroupArn": "'$SERVICE_C_TG'",
                "Weight": 25
            }
        ]
    }' \
    --endpoint-url http://localhost:8080
```

## How It Works with Traefik

When KECS receives weighted routing rules, it:

1. **Stores the weights** in DuckDB with the rule configuration
2. **Converts to Traefik weighted services** in the IngressRoute:
   ```yaml
   services:
     - name: api-v1
       port: 80
       weight: 50
     - name: api-v2
       port: 80
       weight: 50
   ```
3. **Traefik distributes requests** based on the specified weights
4. **Session affinity** can be enabled to keep users on the same version

## Traefik Configuration Example

The weighted routing rules generate Traefik configuration like:

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: listener-my-lb-80
  namespace: kecs-system
spec:
  entryPoints:
    - listener80
  routes:
    # Canary deployment route
    - match: PathPrefix(`/app/`)
      kind: Rule
      priority: 90
      services:
        - name: tg-stable
          port: 80
          weight: 95
        - name: tg-canary
          port: 80
          weight: 5
    
    # A/B test route
    - match: PathPrefix(`/api/`)
      kind: Rule
      priority: 100
      services:
        - name: tg-api-v1
          port: 80
          weight: 50
        - name: tg-api-v2
          port: 80
          weight: 50
```

## Session Stickiness

Enable session stickiness to keep users on the same target group:

```bash
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 60 \
    --conditions Field=path-pattern,Values="/*" \
    --actions Type=forward,ForwardConfig='{
        "TargetGroups": [
            {
                "TargetGroupArn": "'$TG1_ARN'",
                "Weight": 70
            },
            {
                "TargetGroupArn": "'$TG2_ARN'",
                "Weight": 30
            }
        ],
        "TargetGroupStickinessConfig": {
            "Enabled": true,
            "DurationSeconds": 7200
        }
    }' \
    --endpoint-url http://localhost:8080
```

This generates Traefik sticky session configuration:

```yaml
services:
  - name: tg-1
    port: 80
    weight: 70
    sticky:
      cookie:
        name: kecs-sticky
        secure: true
        httpOnly: true
        sameSite: lax
  - name: tg-2
    port: 80
    weight: 30
```

## Testing Weighted Distribution

### Using curl with multiple requests

```bash
# Test distribution with 100 requests
for i in {1..100}; do
  curl -s http://load-balancer-ip/api/version | grep -o "v[0-9]"
done | sort | uniq -c

# Expected output (approximately):
#  50 v1
#  50 v2
```

### Monitor target group metrics

```bash
# Check request count per target group
aws cloudwatch get-metric-statistics \
    --namespace AWS/ApplicationELB \
    --metric-name RequestCount \
    --dimensions Name=TargetGroup,Value=targetgroup/api-v1/73e2d6bc24d8a067 \
    --start-time 2024-01-01T00:00:00Z \
    --end-time 2024-01-01T01:00:00Z \
    --period 300 \
    --statistics Sum \
    --endpoint-url http://localhost:8080
```

## Best Practices

1. **Start with small percentages** - Begin canary deployments with 5-10%
2. **Monitor error rates** - Watch for increased errors in new versions
3. **Use session stickiness** - Keep users on consistent versions during testing
4. **Gradual rollout** - Increase weights gradually (5% → 10% → 25% → 50% → 100%)
5. **Have rollback plan** - Keep old version ready for quick rollback
6. **Test weight accuracy** - Verify actual distribution matches configuration

## Advanced Patterns

### Dynamic Weight Adjustment

Create a script to adjust weights based on metrics:

```bash
#!/bin/bash
# progressive-rollout.sh

RULE_ARN=$1
STABLE_TG=$2
NEW_TG=$3

# Array of weight stages
WEIGHTS=(5 10 25 50 75 100)

for weight in "${WEIGHTS[@]}"; do
  stable_weight=$((100 - weight))
  
  echo "Setting weights: Stable=$stable_weight%, New=$weight%"
  
  aws elbv2 modify-rule \
    --rule-arn $RULE_ARN \
    --actions Type=forward,ForwardConfig="{
      \"TargetGroups\": [
        {
          \"TargetGroupArn\": \"$STABLE_TG\",
          \"Weight\": $stable_weight
        },
        {
          \"TargetGroupArn\": \"$NEW_TG\",
          \"Weight\": $weight
        }
      ]
    }" \
    --endpoint-url http://localhost:8080
  
  # Wait and monitor
  sleep 300  # 5 minutes
  
  # Check error rate (implement your monitoring logic)
  # if error_rate > threshold; then
  #   rollback
  #   exit 1
  # fi
done

echo "Rollout complete!"
```

### Weight-Based Feature Flags

Combine weighted routing with feature flags:

```bash
# Route beta users with weighted distribution
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 50 \
    --conditions \
        Field=http-header,HttpHeaderName=X-Beta-User,Values="true" \
        Field=path-pattern,Values="/api/*" \
    --actions Type=forward,ForwardConfig='{
        "TargetGroups": [
            {
                "TargetGroupArn": "'$STABLE_API_TG'",
                "Weight": 30
            },
            {
                "TargetGroupArn": "'$BETA_API_TG'",
                "Weight": 70
            }
        ]
    }' \
    --endpoint-url http://localhost:8080
```

## Limitations and Considerations

1. **Weight precision** - Traefik uses integer weights (1-1000)
2. **Minimum traffic** - Very low weights (< 5%) may not be accurate with low traffic
3. **Session persistence** - Sticky sessions affect weight distribution
4. **Health checks** - Unhealthy targets are automatically removed from rotation
5. **Target group limits** - Maximum of 5 target groups per rule

## Troubleshooting

### Uneven Distribution

If traffic distribution doesn't match weights:

1. Check target health:
   ```bash
   aws elbv2 describe-target-health \
     --target-group-arn $TG_ARN \
     --endpoint-url http://localhost:8080
   ```

2. Verify sticky sessions aren't affecting distribution
3. Ensure sufficient traffic volume for accurate distribution
4. Check Traefik logs for routing decisions

### Weight Changes Not Applied

1. Verify rule modification succeeded
2. Check Traefik IngressRoute was updated
3. Clear any client-side caches or cookies
4. Restart Traefik if necessary