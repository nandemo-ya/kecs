# Header-Based Routing for Microservices

This example demonstrates how to use header-based routing to implement advanced microservices patterns with KECS and ELBv2.

## Scenario: E-commerce Microservices Platform

We'll implement a microservices architecture with:
- API versioning
- Feature flags
- A/B testing
- Tenant isolation
- Service mesh patterns

## Architecture Overview

```
                          ┌─────────────────┐
                          │   Load Balancer │
                          │   (ELBv2/KECS)  │
                          └────────┬────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              │                    │                    │
         X-API-Version        X-Feature-Flag      X-Tenant-ID
              │                    │                    │
      ┌───────┴───────┐    ┌──────┴──────┐    ┌───────┴───────┐
      │               │    │             │    │               │
   ┌──▼──┐        ┌──▼──┐ ┌▼──┐     ┌──▼──┐ ┌▼──┐        ┌──▼──┐
   │ v1  │        │ v2  │ │Old│     │New │ │Std│        │Ent │
   │ API │        │ API │ │UI │     │UI  │ │App│        │App │
   └─────┘        └─────┘ └───┘     └────┘ └───┘        └────┘
```

## Step 1: Create Target Groups

```bash
# API v1 target group
aws elbv2 create-target-group \
    --name api-v1 \
    --protocol HTTP \
    --port 8080 \
    --vpc-id vpc-12345678 \
    --health-check-path /health \
    --health-check-interval-seconds 30 \
    --endpoint-url http://localhost:8080

# API v2 target group
aws elbv2 create-target-group \
    --name api-v2 \
    --protocol HTTP \
    --port 8080 \
    --vpc-id vpc-12345678 \
    --health-check-path /v2/health \
    --endpoint-url http://localhost:8080

# UI services target groups
aws elbv2 create-target-group \
    --name ui-stable \
    --protocol HTTP \
    --port 3000 \
    --vpc-id vpc-12345678 \
    --endpoint-url http://localhost:8080

aws elbv2 create-target-group \
    --name ui-beta \
    --protocol HTTP \
    --port 3000 \
    --vpc-id vpc-12345678 \
    --endpoint-url http://localhost:8080

# Tenant-specific target groups
aws elbv2 create-target-group \
    --name standard-tenant \
    --protocol HTTP \
    --port 8080 \
    --vpc-id vpc-12345678 \
    --endpoint-url http://localhost:8080

aws elbv2 create-target-group \
    --name enterprise-tenant \
    --protocol HTTP \
    --port 8080 \
    --vpc-id vpc-12345678 \
    --endpoint-url http://localhost:8080
```

## Step 2: API Version Routing

```bash
# Route API v2 requests
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 10 \
    --conditions '[
        {
            "Field": "path-pattern",
            "Values": ["/api/*"]
        },
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-API-Version",
                "Values": ["2.0", "2.1", "2.*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$API_V2_TG \
    --endpoint-url http://localhost:8080

# Route API v1 requests (default)
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 20 \
    --conditions '[
        {
            "Field": "path-pattern",
            "Values": ["/api/*"]
        },
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-API-Version",
                "Values": ["1.0", "1.1", "1.*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$API_V1_TG \
    --endpoint-url http://localhost:8080

# Fallback for unversioned API requests
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 30 \
    --conditions Field=path-pattern,Values="/api/*" \
    --actions Type=forward,TargetGroupArn=$API_V1_TG \
    --endpoint-url http://localhost:8080
```

## Step 3: Feature Flag Based Routing

```bash
# Route users with new UI feature flag
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 40 \
    --conditions '[
        {
            "Field": "path-pattern",
            "Values": ["/", "/app/*"]
        },
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Feature-Flags",
                "Values": ["*new-ui*", "*beta-ui*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$UI_BETA_TG \
    --endpoint-url http://localhost:8080

# A/B testing with percentage-based routing
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 50 \
    --conditions '[
        {
            "Field": "path-pattern",
            "Values": ["/", "/app/*"]
        },
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-AB-Test",
                "Values": ["group-b"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$UI_BETA_TG \
    --endpoint-url http://localhost:8080
```

## Step 4: Tenant-Based Routing

```bash
# Route enterprise tenants to dedicated infrastructure
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 5 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Tenant-ID",
                "Values": ["enterprise-*", "premium-*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$ENTERPRISE_TG \
    --endpoint-url http://localhost:8080

# Route specific high-value tenants
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 3 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Tenant-ID",
                "Values": ["acme-corp", "globex-industries", "initech-systems"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$DEDICATED_TG \
    --endpoint-url http://localhost:8080
```

## Step 5: Service Mesh Integration

```bash
# Route based on service mesh headers
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 60 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Envoy-Upstream-Service-Time",
                "Values": ["*"]
            }
        },
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-B3-TraceId",
                "Values": ["*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$SERVICE_MESH_TG \
    --endpoint-url http://localhost:8080

# Canary deployment with custom headers
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 70 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Canary-Deployment",
                "Values": ["true"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$CANARY_TG \
    --endpoint-url http://localhost:8080
```

## Step 6: Complex Routing Scenarios

### Geographic + Feature Flag Routing

```bash
# Route EU users with GDPR feature to compliant backend
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 15 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "CloudFront-Viewer-Country",
                "Values": ["DE", "FR", "IT", "ES", "NL", "BE", "AT", "PL"]
            }
        },
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "X-Feature-Flags",
                "Values": ["*gdpr-compliant*"]
            }
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$EU_GDPR_TG \
    --endpoint-url http://localhost:8080
```

### Mobile App Version Routing

```bash
# Route old mobile app versions to legacy API
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 25 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "User-Agent",
                "Values": ["MyApp/1.*", "MyApp/2.*"]
            }
        },
        {
            "Field": "path-pattern",
            "Values": ["/api/*"]
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$LEGACY_API_TG \
    --endpoint-url http://localhost:8080

# Route new mobile app versions to modern API
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 26 \
    --conditions '[
        {
            "Field": "http-header",
            "HttpHeaderConfig": {
                "HttpHeaderName": "User-Agent",
                "Values": ["MyApp/3.*", "MyApp/4.*"]
            }
        },
        {
            "Field": "path-pattern",
            "Values": ["/api/*"]
        }
    ]' \
    --actions Type=forward,TargetGroupArn=$MODERN_API_TG \
    --endpoint-url http://localhost:8080
```

## Implementation Best Practices

### 1. Header Injection at Gateway

```javascript
// API Gateway or Edge Proxy
app.use((req, res, next) => {
  // Inject API version from JWT claims
  const token = parseJWT(req.headers.authorization);
  req.headers['x-api-version'] = token.apiVersion || '1.0';
  
  // Inject tenant ID
  req.headers['x-tenant-id'] = token.tenantId;
  
  // Inject feature flags
  const features = getUserFeatures(token.userId);
  req.headers['x-feature-flags'] = features.join(',');
  
  next();
});
```

### 2. Client SDK Implementation

```python
# Python SDK example
class APIClient:
    def __init__(self, api_version='2.0', tenant_id=None):
        self.api_version = api_version
        self.tenant_id = tenant_id
        self.feature_flags = []
    
    def request(self, method, path, **kwargs):
        headers = kwargs.get('headers', {})
        
        # Add routing headers
        headers['X-API-Version'] = self.api_version
        if self.tenant_id:
            headers['X-Tenant-ID'] = self.tenant_id
        if self.feature_flags:
            headers['X-Feature-Flags'] = ','.join(self.feature_flags)
        
        kwargs['headers'] = headers
        return requests.request(method, path, **kwargs)
```

### 3. Testing Headers

```bash
# Test API v2 routing
curl -H "X-API-Version: 2.0" https://api.example.com/api/users

# Test feature flag routing
curl -H "X-Feature-Flags: new-ui,dark-mode" https://app.example.com/

# Test tenant routing
curl -H "X-Tenant-ID: enterprise-acme" https://api.example.com/api/data

# Test combined conditions
curl -H "X-API-Version: 2.0" \
     -H "X-Tenant-ID: enterprise-acme" \
     -H "X-Feature-Flags: beta" \
     https://api.example.com/api/users
```

## Monitoring and Observability

### CloudWatch Metrics

```bash
# Monitor rule matches
aws cloudwatch get-metric-statistics \
    --namespace AWS/ApplicationELB \
    --metric-name RuleEvaluations \
    --dimensions Name=LoadBalancer,Value=app/my-lb/50dc6c495c0c9188 \
                 Name=Rule,Value=app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2/8a2d6bc24d8a067 \
    --start-time 2024-01-01T00:00:00Z \
    --end-time 2024-01-01T23:59:59Z \
    --period 3600 \
    --statistics Sum
```

### Custom Metrics

```python
# Log header routing decisions
import logging
import json

def log_routing_decision(headers, target_group):
    routing_info = {
        'timestamp': datetime.utcnow().isoformat(),
        'api_version': headers.get('X-API-Version'),
        'tenant_id': headers.get('X-Tenant-ID'),
        'feature_flags': headers.get('X-Feature-Flags'),
        'user_agent': headers.get('User-Agent'),
        'target_group': target_group
    }
    
    # Send to CloudWatch Logs
    logger.info(json.dumps(routing_info))
    
    # Send custom metric
    cloudwatch.put_metric_data(
        Namespace='CustomApp/Routing',
        MetricData=[
            {
                'MetricName': 'RoutingDecisions',
                'Value': 1,
                'Dimensions': [
                    {'Name': 'APIVersion', 'Value': routing_info['api_version']},
                    {'Name': 'TargetGroup', 'Value': target_group}
                ]
            }
        ]
    )
```

## Troubleshooting

### Common Issues

1. **Headers Not Matching**
   - Check header name case sensitivity
   - Verify wildcard patterns
   - Test with exact values first

2. **Priority Conflicts**
   - List all rules: `aws elbv2 describe-rules`
   - Check priority ordering
   - More specific rules should have lower priority numbers

3. **Missing Headers**
   - Check if proxy/CDN strips headers
   - Verify client sends headers
   - Use ALB access logs to debug

### Debug Headers

```bash
# Enable ALB access logs to see headers
aws elbv2 modify-load-balancer-attributes \
    --load-balancer-arn $LB_ARN \
    --attributes Key=access_logs.s3.enabled,Value=true \
                 Key=access_logs.s3.bucket,Value=my-alb-logs

# Test with verbose output
curl -v \
    -H "X-API-Version: 2.0" \
    -H "X-Debug: true" \
    https://api.example.com/api/test
```