# ELBv2 SNI Support with Traefik

This document describes how KECS handles Server Name Indication (SNI) for HTTPS listeners with host-based routing.

## Overview

Server Name Indication (SNI) is a TLS extension that allows a client to indicate which hostname it is attempting to connect to at the start of the TLS handshake. This enables serving multiple SSL certificates on the same IP address and port.

## How KECS Implements SNI Support

When you create host-based routing rules for an HTTPS listener, KECS automatically configures Traefik to handle SNI based on the host conditions in your rules.

### 1. Certificate Management

KECS integrates with Kubernetes TLS secrets to manage certificates for different hosts:

```yaml
# Example: TLS secret for api.example.com
apiVersion: v1
kind: Secret
metadata:
  name: api-example-com-tls
  namespace: kecs-system
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-certificate>
  tls.key: <base64-encoded-private-key>
```

### 2. Traefik Configuration

When KECS creates or updates an IngressRoute for HTTPS listeners, it automatically configures TLS options:

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: listener-my-lb-443
  namespace: kecs-system
spec:
  entryPoints:
    - listener443
  routes:
    - match: Host(`api.example.com`)
      kind: Rule
      priority: 10
      services:
        - name: tg-api-targets
          port: 8080
    - match: Host(`www.example.com`)
      kind: Rule
      priority: 20
      services:
        - name: tg-web-targets
          port: 8081
  tls:
    # SNI configuration happens automatically based on Host rules
    domains:
      - main: api.example.com
        secretName: api-example-com-tls
      - main: www.example.com
        secretName: www-example-com-tls
```

## Creating HTTPS Listeners with SNI

### Step 1: Create Certificates

Store your certificates as Kubernetes secrets:

```bash
# Create TLS secret for api.example.com
kubectl create secret tls api-example-com-tls \
  --cert=api.example.com.crt \
  --key=api.example.com.key \
  -n kecs-system

# Create TLS secret for www.example.com
kubectl create secret tls www-example-com-tls \
  --cert=www.example.com.crt \
  --key=www.example.com.key \
  -n kecs-system
```

### Step 2: Create HTTPS Listener

```bash
aws elbv2 create-listener \
    --load-balancer-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-lb/50dc6c495c0c9188 \
    --protocol HTTPS \
    --port 443 \
    --certificates CertificateArn=arn:aws:acm:us-east-1:123456789012:certificate/default-cert \
    --default-actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/default-targets/73e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

### Step 3: Create Host-Based Rules

```bash
# Rule for api.example.com
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/https-listener \
    --priority 10 \
    --conditions Field=host-header,Values="api.example.com" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-targets/73e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080

# Rule for www.example.com
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/https-listener \
    --priority 20 \
    --conditions Field=host-header,Values="www.example.com" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/web-targets/83e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

## How SNI Works in KECS

1. **Client connects** to the load balancer IP on port 443
2. **Client sends SNI** extension with the desired hostname (e.g., "api.example.com")
3. **Traefik receives** the TLS handshake and reads the SNI hostname
4. **Traefik matches** the SNI hostname against the Host rules in the IngressRoute
5. **Traefik selects** the appropriate certificate based on the matched host
6. **TLS handshake** completes with the correct certificate
7. **HTTP request** is routed to the target group based on the Host header

## Wildcard Certificates

KECS supports wildcard certificates for handling multiple subdomains:

```bash
# Create wildcard certificate secret
kubectl create secret tls wildcard-example-com-tls \
  --cert=wildcard.example.com.crt \
  --key=wildcard.example.com.key \
  -n kecs-system
```

When you create a rule with wildcard host pattern:

```bash
aws elbv2 create-rule \
    --listener-arn arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/https-listener \
    --priority 30 \
    --conditions Field=host-header,Values="*.example.com" \
    --actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/tenant-targets/93e2d6bc24d8a067 \
    --endpoint-url http://localhost:8080
```

Traefik will use the wildcard certificate for any subdomain that matches.

## Certificate Priority

When multiple certificates could match a hostname, Traefik follows this priority:

1. **Exact match** - Certificate with CN or SAN exactly matching the hostname
2. **Wildcard match** - Certificate with wildcard that matches the hostname
3. **Default certificate** - The certificate specified in the listener configuration

## Monitoring SNI

You can monitor SNI usage through Traefik metrics:

```bash
# Check Traefik metrics endpoint
curl http://traefik-service:8080/metrics | grep tls

# Example metrics:
# traefik_tls_certs_not_after{cn="api.example.com",sans="api.example.com"} 1.7035968e+09
# traefik_tls_certs_not_after{cn="*.example.com",sans="*.example.com"} 1.7035968e+09
```

## Best Practices

1. **Use specific certificates** for production domains rather than wildcards when possible
2. **Monitor certificate expiration** to ensure continuous service
3. **Test SNI routing** using tools like OpenSSL:
   ```bash
   openssl s_client -connect load-balancer-ip:443 -servername api.example.com
   ```
4. **Implement certificate rotation** before expiration
5. **Use separate namespaces** for certificate secrets in multi-tenant environments

## Limitations

1. **Certificate storage**: Certificates must be stored as Kubernetes secrets
2. **ACM integration**: Direct AWS Certificate Manager integration is not yet supported
3. **Dynamic certificate loading**: New certificates require IngressRoute update

## Future Enhancements

1. **ACM integration**: Direct integration with AWS Certificate Manager
2. **Automatic certificate discovery**: Based on host rules
3. **Let's Encrypt integration**: Automatic certificate provisioning
4. **Certificate validation**: Pre-flight checks for certificate validity