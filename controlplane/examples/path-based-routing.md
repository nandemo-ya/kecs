# Path-Based Routing Example

This example demonstrates how to use ELBv2 path-based routing with KECS.

## Prerequisites

1. KECS running with Kubernetes integration
2. A load balancer created
3. Target groups created
4. A listener created

## Creating Path-Based Routing Rules

### 1. Route /api/* to API Target Group

```bash
# Create a rule that routes all /api/* requests to the API target group
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 10 \
    --conditions Field=path-pattern,Values="/api/*" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-targets/73e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 2. Route /static/* to Static Content Target Group

```bash
# Create a rule for static content
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 20 \
    --conditions Field=path-pattern,Values="/static/*" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/static-targets/83e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 3. Route Specific Path to Admin Target Group

```bash
# Create a rule for exact path matching
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 5 \
    --conditions Field=path-pattern,Values="/admin" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/admin-targets/93e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### 4. Complex Path Pattern

```bash
# Create a rule with complex path pattern
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --priority 30 \
    --conditions Field=path-pattern,Values="/api/*/users" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/user-api-targets/a3e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

## How It Works

When you create a rule with path-based conditions, KECS:

1. **Stores the rule** in DuckDB with the specified priority and conditions
2. **Converts ELBv2 conditions** to Traefik match expressions:
   - `/api/*` → `PathPrefix('/api/')`
   - `/admin` → `Path('/admin')`
   - `/api/*/users` → `PathRegexp('^/api/.*/users$')`
3. **Updates the Traefik IngressRoute** with all rules sorted by priority
4. **Traefik routes traffic** based on the first matching rule

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
    # Priority 5 - Exact admin path
    - match: Path(`/admin`)
      kind: Rule
      priority: 5
      services:
        - name: tg-admin-targets
          port: 80
    
    # Priority 10 - API prefix
    - match: PathPrefix(`/api/`)
      kind: Rule
      priority: 10
      services:
        - name: tg-api-targets
          port: 80
    
    # Priority 20 - Static content
    - match: PathPrefix(`/static/`)
      kind: Rule
      priority: 20
      services:
        - name: tg-static-targets
          port: 80
    
    # Priority 30 - Complex pattern
    - match: PathRegexp(`^/api/.*/users$`)
      kind: Rule
      priority: 30
      services:
        - name: tg-user-api-targets
          port: 80
    
    # Default catch-all (lowest priority)
    - match: PathPrefix(`/`)
      kind: Rule
      priority: 99999
      services:
        - name: default-backend
          port: 80
```

## Listing Rules

```bash
# List all rules for a listener
aws elbv2 describe-rules \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2 \
    --endpoint-url http://localhost:8080
```

## Deleting Rules

```bash
# Delete a rule
aws elbv2 delete-rule \
    --rule-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener-rule/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2/8a2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

When a rule is deleted, KECS automatically re-syncs the remaining rules to the Traefik IngressRoute.