# Header-Based Routing Example

This example demonstrates how to use ELBv2 header-based routing with KECS to route traffic based on HTTP headers.

## Prerequisites

1. KECS running with Kubernetes integration
2. A load balancer created
3. Target groups created for different applications or versions
4. A listener created

## Creating Header-Based Routing Rules

### 1. Route Based on User-Agent

Route different clients to appropriate backends:

```bash
# Route mobile apps to optimized backend
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 10 \
    --conditions Field=http-header,HttpHeaderName=User-Agent,Values="*Mobile*,*Android*,*iOS*" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/mobile-backend/73e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080

# Route desktop browsers to standard backend
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 20 \
    --conditions Field=http-header,HttpHeaderName=User-Agent,Values="*Chrome*,*Firefox*,*Safari*" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/web-backend/83e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 2. API Version Routing

Route requests to different API versions based on custom headers:

```bash
# Route to v2 API
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 5 \
    --conditions Field=http-header,HttpHeaderName=X-API-Version,Values="2.0,2.1,2.*" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v2/93e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080

# Route to v1 API (default)
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 15 \
    --conditions Field=http-header,HttpHeaderName=X-API-Version,Values="1.0,1.1,1.*" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v1/a3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 3. A/B Testing with Custom Headers

Route users to different versions for A/B testing:

```bash
# Route beta users to new version
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 25 \
    --conditions Field=http-header,HttpHeaderName=X-Beta-User,Values="true" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/beta-app/b3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080

# Route canary users to experimental features
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 30 \
    --conditions Field=http-header,HttpHeaderName=X-Feature-Flag,Values="canary-*" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/canary-app/c3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 4. Authentication-Based Routing

Route based on authentication headers:

```bash
# Route authenticated API requests
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 35 \
    --conditions Field=http-header,HttpHeaderName=Authorization,Values="Bearer *" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/auth-api/d3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080

# Route based on custom auth token
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 40 \
    --conditions Field=http-header,HttpHeaderName=X-Auth-Token,Values="*" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/secure-api/e3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 5. Content-Type Based Routing

Route based on content types:

```bash
# Route GraphQL requests
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 45 \
    --conditions Field=http-header,HttpHeaderName=Content-Type,Values="application/graphql" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/graphql-api/f3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080

# Route JSON API requests
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 50 \
    --conditions Field=http-header,HttpHeaderName=Content-Type,Values="application/json" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/rest-api/g3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 6. Combine Multiple Headers

Create complex routing rules with multiple header conditions:

```bash
# Route mobile v2 API requests
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 8 \
    --conditions \
        Field=http-header,HttpHeaderName=User-Agent,Values="*Mobile*" \
        Field=http-header,HttpHeaderName=X-API-Version,Values="2.*" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/mobile-api-v2/h3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

## How It Works

When you create a rule with header-based conditions, KECS:

1. **Stores the rule** in DuckDB with the specified priority and conditions
2. **Converts ELBv2 header conditions** to Traefik match expressions:
   - `X-API-Version: 2.0` → `Header('X-API-Version', '2.0')`
   - `User-Agent: *Mobile*` → `HeaderRegexp('User-Agent', '^.*Mobile.*$')`
   - Multiple values → `(Header('X-API-Version', '2.0') || Header('X-API-Version', '2.1'))`
3. **Updates the Traefik IngressRoute** with all rules sorted by priority
4. **Traefik inspects headers** and routes traffic to the appropriate target group

## Traefik IngressRoute Result

The above rules would generate a Traefik IngressRoute like:

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
    # Priority 5 - API v2
    - match: (Header(`X-API-Version`, `2.0`) || Header(`X-API-Version`, `2.1`) || HeaderRegexp(`X-API-Version`, `^2\..*$`))
      kind: Rule
      priority: 5
      services:
        - name: tg-api-v2
          port: 80
    
    # Priority 8 - Mobile v2 API (multiple conditions)
    - match: (HeaderRegexp(`User-Agent`, `^.*Mobile.*$`) && HeaderRegexp(`X-API-Version`, `^2\..*$`))
      kind: Rule
      priority: 8
      services:
        - name: tg-mobile-api-v2
          port: 80
    
    # Priority 10 - Mobile backend
    - match: (HeaderRegexp(`User-Agent`, `^.*Mobile.*$`) || HeaderRegexp(`User-Agent`, `^.*Android.*$`) || HeaderRegexp(`User-Agent`, `^.*iOS.*$`))
      kind: Rule
      priority: 10
      services:
        - name: tg-mobile-backend
          port: 80
    
    # Default catch-all
    - match: PathPrefix(`/`)
      kind: Rule
      priority: 99999
      services:
        - name: default-backend
          port: 80
```

## Common Use Cases

### 1. API Versioning
- Route different API versions without changing URLs
- Gradual migration from v1 to v2
- Support multiple API versions simultaneously

### 2. A/B Testing
- Test new features with specific user groups
- Canary deployments based on headers
- Feature flag based routing

### 3. Client-Specific Backends
- Optimize backends for mobile vs desktop
- Different backends for different app versions
- Legacy client support

### 4. Security and Authentication
- Route authenticated requests to secure backends
- Different backends based on authorization levels
- API key based routing

### 5. Content Negotiation
- Route based on Accept headers
- Different backends for different content types
- GraphQL vs REST API routing

## Testing Header-Based Routing

Test your rules using curl:

```bash
# Test API version routing
curl -H "X-API-Version: 2.0" http://load-balancer-ip/api/users

# Test mobile routing
curl -H "User-Agent: MyApp/1.0 (iPhone; iOS 14.0)" http://load-balancer-ip/

# Test beta features
curl -H "X-Beta-User: true" http://load-balancer-ip/

# Test authentication routing
curl -H "Authorization: Bearer my-jwt-token" http://load-balancer-ip/api/secure

# Test multiple headers
curl -H "User-Agent: Mobile" -H "X-API-Version: 2.1" http://load-balancer-ip/api
```

## Best Practices

1. **Use specific header names** - Avoid generic headers that might conflict
2. **Document your headers** - Maintain clear documentation of custom headers
3. **Set appropriate priorities** - More specific rules should have lower priority numbers
4. **Consider security** - Don't expose sensitive routing logic through headers
5. **Test thoroughly** - Verify header matching with various client scenarios
6. **Monitor header usage** - Track which rules are being matched

## Advanced Patterns

### Dynamic Feature Flags

```bash
# Route based on multiple feature flags
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 100 \
    --conditions Field=http-header,HttpHeaderName=X-Features,Values="*new-ui*,*dark-mode*" \
    --actions Type=forward,TargetGroupArn=$FEATURE_TG \
    --endpoint-url http://localhost:8080
```

### Tenant Isolation

```bash
# Route based on tenant ID in header
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 110 \
    --conditions Field=http-header,HttpHeaderName=X-Tenant-ID,Values="enterprise-*" \
    --actions Type=forward,TargetGroupArn=$ENTERPRISE_TG \
    --endpoint-url http://localhost:8080
```

### Geographic Routing

```bash
# Route based on CloudFront geo headers
aws elbv2 create-rule \
    --listener-arn $LISTENER_ARN \
    --priority 120 \
    --conditions Field=http-header,HttpHeaderName=CloudFront-Viewer-Country,Values="US,CA,MX" \
    --actions Type=forward,TargetGroupArn=$NORTH_AMERICA_TG \
    --endpoint-url http://localhost:8080
```