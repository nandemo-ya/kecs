# Conditional Routing Example

This example demonstrates how to use ELBv2 conditional routing with KECS to implement complex routing logic based on multiple conditions.

## Overview

Conditional routing allows you to create sophisticated routing rules that evaluate multiple conditions and perform different actions based on the results. This is useful for:

- A/B testing with multiple variants
- Geographic routing with fallbacks
- Feature flag based routing with conditions
- API versioning with backward compatibility
- Multi-tenant routing with custom logic

## Conditional Routing Patterns

### 1. If-Then-Else Routing

Route traffic based on multiple conditions with fallback options:

```bash
# Primary condition: Route beta users to v2
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 10 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Beta-User",
                "Values": ["true"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/api/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$API_V2_TG \
    --endpoint-url http://localhost:8080

# Secondary condition: Route mobile users to mobile-optimized API
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 20 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "User-Agent",
                "Values": ["*Mobile*", "*Android*", "*iOS*"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/api/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$MOBILE_API_TG \
    --endpoint-url http://localhost:8080

# Default condition: All other API traffic
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 30 \
    --conditions Field=path-pattern,PathPatternConfig={Values=["/api/*"]} \
    --actions Type=forward,TargetGroupArn=$API_V1_TG \
    --endpoint-url http://localhost:8080
```

### 2. Multi-Stage Feature Rollout

Implement progressive feature rollout with multiple conditions:

```bash
# Stage 1: Internal users (highest priority)
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 5 \
    --conditions '[
        {
            "Field": "source-ip",
            "SourceIpConfig": {
                "Values": ["10.0.0.0/8", "172.16.0.0/12"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/feature/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$FEATURE_INTERNAL_TG \
    --endpoint-url http://localhost:8080

# Stage 2: Beta users
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 15 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-User-Group",
                "Values": ["beta", "early-adopter"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/feature/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$FEATURE_BETA_TG \
    --endpoint-url http://localhost:8080

# Stage 3: Percentage-based rollout (using weighted routing)
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 25 \
    --conditions Field=path-pattern,PathPatternConfig={Values=["/feature/*"]} \
    --actions '[
        {
            "Type": "forward",
            "ForwardConfig": {
                "TargetGroups": [
                    {
                        "TargetGroupArn": "'$FEATURE_NEW_TG'",
                        "Weight": 20
                    },
                    {
                        "TargetGroupArn": "'$FEATURE_OLD_TG'",
                        "Weight": 80
                    }
                ]
            }
        }
    ]' \
    --endpoint-url http://localhost:8080
```

### 3. Geographic Routing with Conditions

Route based on location with specific overrides:

```bash
# EU users with GDPR compliance requirement
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 100 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "CloudFront-Viewer-Country",
                "Values": ["DE", "FR", "IT", "ES", "GB", "NL", "BE", "AT", "PL", "SE", "DK", "FI", "NO"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/user/*", "/account/*", "/profile/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$EU_GDPR_TG \
    --endpoint-url http://localhost:8080

# Asia-Pacific users with local data residency
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 110 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "CloudFront-Viewer-Country",
                "Values": ["JP", "SG", "AU", "KR", "IN", "CN", "HK", "TW"]
            }
        },
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Data-Residency",
                "Values": ["required"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$APAC_LOCAL_TG \
    --endpoint-url http://localhost:8080

# US users - default region
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 120 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "CloudFront-Viewer-Country",
                "Values": ["US", "CA", "MX"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$US_TG \
    --endpoint-url http://localhost:8080
```

### 4. API Version Negotiation

Complex API versioning with backward compatibility:

```bash
# Exact version match
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 200 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "Accept",
                "Values": ["application/vnd.api+json;version=2.1"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/api/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$API_V2_1_TG \
    --endpoint-url http://localhost:8080

# Version range support
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 210 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-API-Version",
                "Values": ["2.0", "2.1", "2.2"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/api/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$API_V2_TG \
    --endpoint-url http://localhost:8080

# Legacy support with deprecation warning
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 220 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-API-Version",
                "Values": ["1.0", "1.1", "1.2"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/api/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$API_V1_LEGACY_TG \
    --endpoint-url http://localhost:8080
```

### 5. Multi-Tenant Conditional Routing

Route tenants based on multiple criteria:

```bash
# Premium tenant with specific features
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 300 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Tenant-Type",
                "Values": ["premium", "enterprise"]
            }
        },
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Feature-Flags",
                "Values": ["*advanced-analytics*", "*custom-reports*"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/analytics/*", "/reports/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$PREMIUM_ANALYTICS_TG \
    --endpoint-url http://localhost:8080

# Tenant with data isolation requirement
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 310 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Tenant-ID",
                "Values": ["isolated-*"]
            }
        },
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Isolation-Level",
                "Values": ["strict", "complete"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$ISOLATED_TENANT_TG \
    --endpoint-url http://localhost:8080
```

### 6. Canary Deployment with Conditions

Advanced canary deployment based on multiple factors:

```bash
# Canary for specific user segments
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 400 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-User-Segment",
                "Values": ["power-user", "developer", "qa-tester"]
            }
        },
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Canary-Opt-In",
                "Values": ["true"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/app/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$CANARY_TG \
    --endpoint-url http://localhost:8080

# Time-based canary (business hours only)
# Note: This would require custom logic in the application
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 410 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Business-Hours",
                "Values": ["true"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/app/*"]
            }
        }
    ]' \
    --actions '[
        {
            "Type": "forward",
            "ForwardConfig": {
                "TargetGroups": [
                    {
                        "TargetGroupArn": "'$CANARY_TG'",
                        "Weight": 10
                    },
                    {
                        "TargetGroupArn": "'$STABLE_TG'",
                        "Weight": 90
                    }
                ]
            }
        }
    ]' \
    --endpoint-url http://localhost:8080
```

## Complex Conditional Logic Implementation

### Nested Conditions with Priority Management

```bash
#!/bin/bash
# deploy-conditional-routes.sh

# Function to create rules with automatic priority assignment
create_conditional_rule() {
    local description=$1
    local conditions=$2
    local target_group=$3
    local priority_range=$4
    
    # Use KECS priority management to find appropriate priority
    priority=$(aws elbv2 describe-rules \
        --listener-arn $LISTENER_ARN \
        --endpoint-url http://localhost:8080 \
        | jq --arg range "$priority_range" '
            .Rules | map(.Priority) | sort | 
            if $range == "critical" then
                map(select(. < 100)) | (. + [1]) | max
            elif $range == "specific" then
                map(select(. >= 100 and . < 1000)) | (. + [100]) | max // 100
            elif $range == "general" then
                map(select(. >= 1000 and . < 10000)) | (. + [1000]) | max // 1000
            else
                map(select(. >= 10000 and . < 50000)) | (. + [10000]) | max // 10000
            end')
    
    echo "Creating rule: $description with priority $priority"
    
    aws elbv2 create-rule \
        --listener-arn $LISTENER_ARN \
        --priority $priority \
        --conditions "$conditions" \
        --actions Type=forward,TargetGroupArn=$target_group \
        --endpoint-url http://localhost:8080
}

# Deploy conditional routing rules
create_conditional_rule \
    "Critical: Health check bypass" \
    '[{"Field":"path-pattern","PathPatternConfig":{"Values":["/health","/healthz","/ping"]}}]' \
    $HEALTH_CHECK_TG \
    "critical"

create_conditional_rule \
    "Specific: Beta API v2 users" \
    '[
        {"Field":"http-header","HttpHeaderConfig":{"HttpHeaderName":"X-Beta-User","Values":["true"]}},
        {"Field":"http-header","HttpHeaderConfig":{"HttpHeaderName":"X-API-Version","Values":["2.*"]}},
        {"Field":"path-pattern","PathPatternConfig":{"Values":["/api/*"]}}
    ]' \
    $BETA_API_V2_TG \
    "specific"

create_conditional_rule \
    "General: Mobile users" \
    '[
        {"Field":"http-header","HttpHeaderConfig":{"HttpHeaderName":"User-Agent","Values":["*Mobile*"]}},
        {"Field":"path-pattern","PathPatternConfig":{"Values":["/api/*","/app/*"]}}
    ]' \
    $MOBILE_TG \
    "general"

create_conditional_rule \
    "Catch-all: Default route" \
    '[{"Field":"path-pattern","PathPatternConfig":{"Values":["/*"]}}]' \
    $DEFAULT_TG \
    "catchall"
```

## How KECS Handles Conditional Routing

1. **Rule Evaluation Order**: Rules are evaluated in priority order (lowest number first)
2. **First Match Wins**: The first rule that matches all conditions is used
3. **Multiple Conditions**: All conditions in a rule must match (AND logic)
4. **Condition Types**: Supports path, host, header, method, query string, and source IP conditions
5. **Actions**: Forward to single target group or weighted distribution across multiple groups

### Traefik Translation

KECS converts ELBv2 conditional rules to Traefik IngressRoute format:

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
    # Priority 5: Beta users with API v2
    - match: Header(`X-Beta-User`, `true`) && HeaderRegexp(`X-API-Version`, `^2\\..*$`) && PathPrefix(`/api/`)
      kind: Rule
      priority: 5
      services:
        - name: tg-beta-api-v2
          port: 80
    
    # Priority 15: Mobile users
    - match: HeaderRegexp(`User-Agent`, `^.*Mobile.*$`) && (PathPrefix(`/api/`) || PathPrefix(`/app/`))
      kind: Rule
      priority: 15
      services:
        - name: tg-mobile
          port: 80
    
    # Priority 100: Geographic routing
    - match: (Header(`CloudFront-Viewer-Country`, `DE`) || Header(`CloudFront-Viewer-Country`, `FR`)) && PathPrefix(`/user/`)
      kind: Rule
      priority: 100
      services:
        - name: tg-eu-gdpr
          port: 80
```

## Best Practices for Conditional Routing

1. **Order Matters**: More specific conditions should have lower priority numbers
2. **Test Coverage**: Test all condition combinations thoroughly
3. **Fallback Routes**: Always have a catch-all route for unmatched requests
4. **Performance**: Minimize the number of conditions per rule
5. **Monitoring**: Track which rules are matching using metrics
6. **Documentation**: Document complex routing logic clearly

## Testing Conditional Routes

### Test Different Scenarios

```bash
# Test beta user with v2 API
curl -H "X-Beta-User: true" \
     -H "X-API-Version: 2.0" \
     http://load-balancer-ip/api/users

# Test mobile user
curl -H "User-Agent: Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)" \
     http://load-balancer-ip/app/dashboard

# Test geographic routing
curl -H "CloudFront-Viewer-Country: DE" \
     http://load-balancer-ip/user/profile

# Test multi-tenant routing
curl -H "X-Tenant-Type: premium" \
     -H "X-Feature-Flags: advanced-analytics,custom-reports" \
     http://load-balancer-ip/analytics/dashboard
```

### Validate Rule Matching

```bash
# Check which rule would match for given conditions
aws elbv2 describe-rules \
    --listener-arn $LISTENER_ARN \
    --endpoint-url http://localhost:8080 \
    | jq '.Rules | sort_by(.Priority) | 
        map(select(.Conditions | length > 1)) | 
        .[0:5] | 
        map({Priority, ConditionCount: (.Conditions | length), Conditions})'
```

## Troubleshooting

### Common Issues

1. **Rules Not Matching**
   - Check priority order
   - Verify all conditions are met
   - Look for typos in header names
   - Check header value case sensitivity

2. **Unexpected Routing**
   - A higher priority rule may be catching traffic
   - Use debug headers to trace routing decisions
   - Check rule conditions carefully

3. **Performance Issues**
   - Too many complex conditions
   - Consider consolidating similar rules
   - Use more specific match patterns

### Debug Headers

Add debug headers to trace routing:

```bash
# Add debug rule
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 1 \
    --conditions Field=http-header,HttpHeaderConfig={HttpHeaderName="X-Debug-Routing",Values=["true"]} \
    --actions Type=forward,TargetGroupArn=$DEBUG_TG \
    --endpoint-url http://localhost:8080
```

## Advanced Patterns

### Dynamic Routing Based on Request Content

While ELBv2 doesn't inspect request bodies, you can implement dynamic routing using headers set by an edge proxy:

```bash
# Edge proxy sets header based on request content
# Then ELBv2 routes based on that header
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 500 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Content-Type",
                "Values": ["graphql"]
            }
        },
        {
            "Field": "path-pattern",
            "PathPatternConfig": {
                "Values": ["/api/*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$GRAPHQL_TG \
    --endpoint-url http://localhost:8080
```

### Circuit Breaker Pattern

Implement circuit breaker by updating rules dynamically:

```bash
#!/bin/bash
# circuit-breaker.sh

check_health() {
    local target_group=$1
    # Check target group health
    healthy_count=$(aws elbv2 describe-target-health \
        --target-group-arn $target_group \
        --endpoint-url http://localhost:8080 \
        | jq '[.TargetHealthDescriptions[] | select(.TargetHealth.State == "healthy")] | length')
    
    echo $healthy_count
}

# If primary target group is unhealthy, route to fallback
primary_health=$(check_health $PRIMARY_TG)
if [ $primary_health -lt 2 ]; then
    echo "Primary unhealthy, updating rules to use fallback"
    aws elbv2 modify-rule \
        --rule-arn $PRIMARY_RULE_ARN \
        --actions Type=forward,TargetGroupArn=$FALLBACK_TG \
        --endpoint-url http://localhost:8080
fi
```

## Summary

Conditional routing in KECS provides powerful traffic management capabilities:

- Multiple condition types (path, host, header, method, query, source IP)
- Complex routing logic with priority-based evaluation
- Integration with weighted routing for gradual rollouts
- Support for A/B testing, geographic routing, and multi-tenancy
- Automatic translation to Traefik IngressRoute format

Use these patterns to implement sophisticated routing strategies while maintaining clear, manageable rules.