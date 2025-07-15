# Rule Priority Management Example

This example demonstrates how to effectively manage multiple ELBv2 rules with priorities to ensure correct routing behavior.

## Overview

ELBv2 rules are evaluated in priority order (lowest number = highest priority). KECS maintains this priority ordering when converting to Traefik IngressRoute rules. Proper priority management is crucial for:

- Ensuring more specific rules match before general ones
- Preventing rule conflicts
- Maintaining predictable routing behavior
- Implementing complex routing strategies

## Priority Best Practices

### 1. Priority Ranges

Organize your rules into priority ranges:

```
1-99:     Critical system routes (health checks, admin)
100-999:  Specific application routes
1000-9999: General application routes
10000+:   Catch-all and default routes
```

### 2. Rule Specificity

More specific rules should have lower priority numbers:

```bash
# Priority 10: Most specific - exact path + header
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 10 \
    --conditions \
        Field=path-pattern,Values="/api/v2/users/123" \
        Field=http-header,HttpHeaderName=X-API-Key,Values="special-key" \
    --actions Type=forward,TargetGroupArn=$SPECIAL_API_TG \
    --endpoint-url http://localhost:8080

# Priority 100: Specific path pattern
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 100 \
    --conditions Field=path-pattern,Values="/api/v2/users/*" \
    --actions Type=forward,TargetGroupArn=$API_V2_TG \
    --endpoint-url http://localhost:8080

# Priority 1000: General path pattern
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 1000 \
    --conditions Field=path-pattern,Values="/api/*" \
    --actions Type=forward,TargetGroupArn=$API_GENERAL_TG \
    --endpoint-url http://localhost:8080

# Priority 50000: Default catch-all (automatically added by KECS)
```

## Managing Rule Priorities

### 1. List All Rules with Priorities

```bash
# List rules sorted by priority
aws elbv2 describe-rules \
    --listener-arn $LISTENER_ARN \
    --endpoint-url http://localhost:8080 \
    | jq '.Rules | sort_by(.Priority) | .[] | {Priority, Conditions, Actions}'
```

### 2. Update Rule Priority

```bash
# Change a rule's priority
aws elbv2 set-rule-priorities \
    --rule-priorities RuleArn=$RULE_ARN,Priority=150 \
    --endpoint-url http://localhost:8080
```

### 3. Batch Priority Updates

```bash
# Update multiple rule priorities at once
aws elbv2 set-rule-priorities \
    --rule-priorities \
        RuleArn=$RULE1_ARN,Priority=10 \
        RuleArn=$RULE2_ARN,Priority=20 \
        RuleArn=$RULE3_ARN,Priority=30 \
    --endpoint-url http://localhost:8080
```

## Complex Routing Examples

### Example 1: API Version Migration

Gradually migrate from v1 to v2 API with controlled routing:

```bash
# Priority 50: Beta users get v2
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 50 \
    --conditions \
        Field=path-pattern,Values="/api/*" \
        Field=http-header,HttpHeaderName=X-Beta-User,Values="true" \
    --actions Type=forward,TargetGroupArn=$API_V2_TG \
    --endpoint-url http://localhost:8080

# Priority 100: Specific v2 endpoints
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 100 \
    --conditions \
        Field=path-pattern,Values="/api/v2/*" \
    --actions Type=forward,TargetGroupArn=$API_V2_TG \
    --endpoint-url http://localhost:8080

# Priority 200: Mobile apps get special handling
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 200 \
    --conditions \
        Field=path-pattern,Values="/api/*" \
        Field=http-header,HttpHeaderName=User-Agent,Values="*Mobile*" \
    --actions Type=forward,TargetGroupArn=$API_MOBILE_TG \
    --endpoint-url http://localhost:8080

# Priority 1000: Default to v1
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 1000 \
    --conditions Field=path-pattern,Values="/api/*" \
    --actions Type=forward,TargetGroupArn=$API_V1_TG \
    --endpoint-url http://localhost:8080
```

### Example 2: Multi-Tenant Routing

Route tenants based on multiple criteria:

```bash
# Priority 10: Premium tenant with specific path
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 10 \
    --conditions \
        Field=host-header,Values="premium.example.com" \
        Field=path-pattern,Values="/admin/*" \
    --actions Type=forward,TargetGroupArn=$PREMIUM_ADMIN_TG \
    --endpoint-url http://localhost:8080

# Priority 20: Premium tenant general
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 20 \
    --conditions Field=host-header,Values="premium.example.com" \
    --actions Type=forward,TargetGroupArn=$PREMIUM_TG \
    --endpoint-url http://localhost:8080

# Priority 100: Tenant by header with path
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 100 \
    --conditions \
        Field=http-header,HttpHeaderName=X-Tenant-ID,Values="tenant-123" \
        Field=path-pattern,Values="/api/*" \
    --actions Type=forward,TargetGroupArn=$TENANT_123_API_TG \
    --endpoint-url http://localhost:8080

# Priority 500: General tenant routing
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 500 \
    --conditions Field=http-header,HttpHeaderName=X-Tenant-ID,Values="*" \
    --actions Type=forward,TargetGroupArn=$MULTI_TENANT_TG \
    --endpoint-url http://localhost:8080
```

### Example 3: Geographic + Feature Routing

Combine geographic and feature-based routing:

```bash
# Priority 5: EU users with GDPR requirements
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 5 \
    --conditions \
        Field=http-header,HttpHeaderName=CloudFront-Viewer-Country,Values="DE,FR,IT,ES" \
        Field=path-pattern,Values="/user-data/*" \
    --actions Type=forward,TargetGroupArn=$EU_GDPR_TG \
    --endpoint-url http://localhost:8080

# Priority 10: US users with beta features
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 10 \
    --conditions \
        Field=http-header,HttpHeaderName=CloudFront-Viewer-Country,Values="US" \
        Field=http-header,HttpHeaderName=X-Feature-Flag,Values="beta" \
    --actions Type=forward,TargetGroupArn=$US_BETA_TG \
    --endpoint-url http://localhost:8080

# Priority 50: Asia-Pacific optimized route
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 50 \
    --conditions Field=http-header,HttpHeaderName=CloudFront-Viewer-Country,Values="JP,SG,AU,KR" \
    --actions Type=forward,TargetGroupArn=$APAC_TG \
    --endpoint-url http://localhost:8080
```

## Priority Conflict Resolution

### Detecting Conflicts

Use this script to detect potential rule conflicts:

```bash
#!/bin/bash
# check-rule-conflicts.sh

LISTENER_ARN=$1

# Get all rules
rules=$(aws elbv2 describe-rules \
    --listener-arn $LISTENER_ARN \
    --endpoint-url http://localhost:8080 \
    --query 'Rules[?Priority < `50000`]' \
    --output json)

# Check for overlapping conditions
echo "$rules" | jq -r '.[] | 
    {
        Priority, 
        PathPatterns: [.Conditions[] | select(.Field == "path-pattern") | .Values[]], 
        HostHeaders: [.Conditions[] | select(.Field == "host-header") | .Values[]],
        Headers: [.Conditions[] | select(.Field == "http-header") | .HttpHeaderName]
    }' | jq -s 'sort_by(.Priority)'
```

### Priority Gap Analysis

Find available priority slots:

```bash
#!/bin/bash
# find-priority-gaps.sh

LISTENER_ARN=$1

# Get used priorities
used_priorities=$(aws elbv2 describe-rules \
    --listener-arn $LISTENER_ARN \
    --endpoint-url http://localhost:8080 \
    --query 'Rules[].Priority' \
    --output json | jq -r '.[]' | sort -n)

# Find gaps
prev=0
for priority in $used_priorities; do
    if [ $((priority - prev)) -gt 1 ]; then
        echo "Available range: $((prev + 1)) to $((priority - 1))"
    fi
    prev=$priority
done
```

## How KECS Handles Priorities

1. **Rule Storage**: Rules are stored in DuckDB with their priority values
2. **Sorting**: When syncing to Traefik, rules are sorted by priority (ascending)
3. **Traefik Mapping**: Rules are added to IngressRoute in priority order
4. **First Match Wins**: Traefik evaluates rules in order and uses the first match

### Traefik IngressRoute Result

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
    # Priority 5: EU GDPR route
    - match: (Header(`CloudFront-Viewer-Country`, `DE`) || Header(`CloudFront-Viewer-Country`, `FR`)) && PathPrefix(`/user-data/`)
      kind: Rule
      priority: 5
      services:
        - name: tg-eu-gdpr
          port: 80
    
    # Priority 10: Premium admin route
    - match: Host(`premium.example.com`) && PathPrefix(`/admin/`)
      kind: Rule
      priority: 10
      services:
        - name: tg-premium-admin
          port: 80
    
    # Priority 50: Beta users
    - match: PathPrefix(`/api/`) && Header(`X-Beta-User`, `true`)
      kind: Rule
      priority: 50
      services:
        - name: tg-api-v2
          port: 80
    
    # Default catch-all
    - match: PathPrefix(`/`)
      kind: Rule
      priority: 99999
      services:
        - name: default-backend
          port: 80
```

## Best Practices Summary

1. **Plan Your Priority Ranges**: Define clear ranges for different rule types
2. **Document Priority Scheme**: Maintain documentation of your priority strategy
3. **Leave Gaps**: Don't use consecutive numbers - leave room for future rules
4. **Test Rule Order**: Verify rules match in the expected order
5. **Monitor Rule Evaluation**: Use CloudWatch metrics to track rule matches
6. **Regular Audits**: Periodically review and optimize rule priorities

## Troubleshooting Priority Issues

### Rule Not Matching

1. Check if a higher priority rule is catching the traffic:
   ```bash
   # Test with specific headers and path
   curl -v -H "X-Debug: true" -H "X-Beta-User: true" https://lb.example.com/api/test
   ```

2. List rules in priority order:
   ```bash
   aws elbv2 describe-rules \
       --listener-arn $LISTENER_ARN \
       --endpoint-url http://localhost:8080 \
       | jq '.Rules | sort_by(.Priority)'
   ```

3. Temporarily disable higher priority rules to test

### Performance Considerations

- Keep the number of rules reasonable (< 100 per listener)
- Use specific conditions to reduce evaluation overhead
- Combine related conditions in a single rule when possible
- Monitor rule evaluation metrics