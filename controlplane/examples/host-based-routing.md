# Host-Based Routing Example

This example demonstrates how to use ELBv2 host-based routing with KECS to route traffic to different target groups based on the Host header.

## Prerequisites

1. KECS running with Kubernetes integration
2. A load balancer created
3. Multiple target groups created for different applications
4. A listener created

## Creating Host-Based Routing Rules

### 1. Route api.example.com to API Target Group

```bash
# Create a rule that routes all requests for api.example.com to the API target group
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 10 \
    --conditions Field=host-header,Values="api.example.com" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-targets/73e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 2. Route www.example.com to Web Target Group

```bash
# Create a rule for the main website
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 20 \
    --conditions Field=host-header,Values="www.example.com" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/web-targets/83e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 3. Route Wildcard Subdomains to Multi-Tenant App

```bash
# Create a rule for wildcard subdomains (e.g., customer1.example.com, customer2.example.com)
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 30 \
    --conditions Field=host-header,Values="*.example.com" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/tenant-app-targets/93e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 4. Route Multiple Hosts to the Same Target Group

```bash
# Create a rule that handles multiple domains
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 40 \
    --conditions Field=host-header,Values="blog.example.com,news.example.com,media.example.com" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/content-targets/a3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 5. Combine Host and Path Conditions

```bash
# Create a rule with both host and path conditions
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 5 \
    --conditions \
        Field=host-header,Values="admin.example.com" \
        Field=path-pattern,Values="/dashboard/*" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/admin-targets/b3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

## How It Works

When you create a rule with host-based conditions, KECS:

1. **Stores the rule** in DuckDB with the specified priority and conditions
2. **Converts ELBv2 host conditions** to Traefik match expressions:
   - `api.example.com` → `Host('api.example.com')`
   - `*.example.com` → `HostRegexp('^[^.]+.example.com$')`
   - Multiple hosts → `(Host('blog.example.com') || Host('news.example.com') || Host('media.example.com'))`
3. **Updates the Traefik IngressRoute** with all rules sorted by priority
4. **Traefik routes traffic** based on the Host header in incoming requests

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
    # Priority 5 - Admin dashboard (host + path)
    - match: (Host(`admin.example.com`) && PathPrefix(`/dashboard/`))
      kind: Rule
      priority: 5
      services:
        - name: tg-admin-targets
          port: 80
    
    # Priority 10 - API subdomain
    - match: Host(`api.example.com`)
      kind: Rule
      priority: 10
      services:
        - name: tg-api-targets
          port: 80
    
    # Priority 20 - Main website
    - match: Host(`www.example.com`)
      kind: Rule
      priority: 20
      services:
        - name: tg-web-targets
          port: 80
    
    # Priority 30 - Wildcard subdomains
    - match: HostRegexp(`^[^.]+.example.com$`)
      kind: Rule
      priority: 30
      services:
        - name: tg-tenant-app-targets
          port: 80
    
    # Priority 40 - Multiple content hosts
    - match: (Host(`blog.example.com`) || Host(`news.example.com`) || Host(`media.example.com`))
      kind: Rule
      priority: 40
      services:
        - name: tg-content-targets
          port: 80
    
    # Default catch-all (lowest priority)
    - match: PathPrefix(`/`)
      kind: Rule
      priority: 99999
      services:
        - name: default-backend
          port: 80
```

## Common Use Cases

### 1. Multi-Tenant SaaS Applications
Use wildcard host routing to serve different tenants from the same application:
- `customer1.app.com` → Tenant 1's instance
- `customer2.app.com` → Tenant 2's instance
- `*.app.com` → Multi-tenant application

### 2. Microservices Architecture
Route different subdomains to different microservices:
- `api.company.com` → API service
- `auth.company.com` → Authentication service
- `dashboard.company.com` → Dashboard UI

### 3. Blue-Green Deployments
Use host headers for testing new versions:
- `www.example.com` → Production (green)
- `beta.example.com` → Beta version (blue)
- `staging.example.com` → Staging environment

### 4. Content Separation
Separate different types of content:
- `blog.company.com` → Blog platform
- `docs.company.com` → Documentation site
- `support.company.com` → Support portal

## Advanced Features

### SNI (Server Name Indication) Support

When using HTTPS listeners, Traefik automatically handles SNI based on the host rules. This allows serving multiple SSL certificates on the same IP address and port.

### Priority Considerations

1. **More specific rules should have lower priority numbers** (higher precedence)
2. **Exact host matches** should come before wildcard matches
3. **Combined conditions** (host + path) should have higher precedence than single conditions

### Testing Host-Based Routing

You can test host-based routing using curl with the Host header:

```bash
# Test api.example.com routing
curl -H "Host: api.example.com" http://load-balancer-ip/

# Test wildcard routing
curl -H "Host: customer123.example.com" http://load-balancer-ip/

# Test combined conditions
curl -H "Host: admin.example.com" http://load-balancer-ip/dashboard/
```

## Listing Rules

```bash
# List all rules for a listener
aws elbv2 describe-rules \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --endpoint-url http://localhost:8080
```

## Modifying Host-Based Rules

```bash
# Modify an existing rule to add more hosts
aws elbv2 modify-rule \
    --rule-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener-rule/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2/8a2d6bc24d8a067 \
    --conditions Field=host-header,Values="blog.example.com,news.example.com,media.example.com,forum.example.com" \
    --endpoint-url http://localhost:8080
```

## Best Practices

1. **Use specific host rules** for production traffic
2. **Place wildcard rules at lower priority** to avoid catching specific subdomains
3. **Combine host and path conditions** for fine-grained routing control
4. **Test with curl** before pointing DNS to the load balancer
5. **Monitor rule matches** to ensure traffic is routed correctly