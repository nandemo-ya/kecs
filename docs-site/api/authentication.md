# Authentication Guide

## Overview

KECS provides multiple authentication methods to secure API access. By default, authentication is disabled for local development, but it should be enabled for production deployments.

## Authentication Methods

### 1. API Key Authentication

Simple token-based authentication using API keys.

#### Configuration

```yaml
# kecs-config.yaml
auth:
  type: api-key
  apiKeys:
    - name: "production-key"
      key: "kecs_prod_1234567890abcdef"
      permissions: ["read", "write"]
    - name: "readonly-key"
      key: "kecs_read_0987654321fedcba"
      permissions: ["read"]
```

#### Usage

Include the API key in the `X-API-Key` header:

```bash
curl -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -H "X-API-Key: kecs_prod_1234567890abcdef" \
  -d '{}'
```

### 2. JWT Authentication

JSON Web Token based authentication for more secure, stateless authentication.

#### Configuration

```yaml
# kecs-config.yaml
auth:
  type: jwt
  jwt:
    secret: "your-secret-key"  # Use environment variable in production
    issuer: "kecs"
    audience: "kecs-api"
    expirationTime: "24h"
```

#### Obtaining a Token

```bash
# Login endpoint
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "secure-password"
  }'

# Response
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresIn": 86400
}
```

#### Using the Token

Include the JWT token in the `Authorization` header:

```bash
curl -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{}'
```

### 3. AWS IAM Authentication

Use AWS IAM credentials for authentication, compatible with AWS SDK.

#### Configuration

```yaml
# kecs-config.yaml
auth:
  type: iam
  iam:
    region: "us-east-1"
    verifySignature: true
```

#### Usage with AWS CLI

```bash
# Configure AWS credentials
aws configure

# Use AWS CLI with KECS endpoint
aws ecs list-clusters --endpoint-url http://localhost:8080
```

#### Usage with AWS SDK

```python
import boto3

# Python SDK automatically signs requests
ecs = boto3.client(
    'ecs',
    endpoint_url='http://localhost:8080',
    region_name='us-east-1'
)

response = ecs.list_clusters()
```

### 4. mTLS (Mutual TLS)

Certificate-based authentication for maximum security.

#### Configuration

```yaml
# kecs-config.yaml
auth:
  type: mtls
  tls:
    certFile: "/path/to/server-cert.pem"
    keyFile: "/path/to/server-key.pem"
    caFile: "/path/to/ca-cert.pem"
    clientAuth: "RequireAndVerifyClientCert"
```

#### Client Configuration

```bash
# Using curl with client certificate
curl -X POST https://localhost:8080/v1/ListClusters \
  --cert /path/to/client-cert.pem \
  --key /path/to/client-key.pem \
  --cacert /path/to/ca-cert.pem \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -d '{}'
```

## Authorization

### Role-Based Access Control (RBAC)

Define roles and permissions for fine-grained access control.

#### Role Configuration

```yaml
# kecs-config.yaml
auth:
  rbac:
    enabled: true
    roles:
      - name: "admin"
        permissions:
          - "clusters:*"
          - "services:*"
          - "tasks:*"
          - "taskDefinitions:*"
      
      - name: "developer"
        permissions:
          - "clusters:read"
          - "services:*"
          - "tasks:*"
          - "taskDefinitions:*"
      
      - name: "viewer"
        permissions:
          - "*:read"
          - "*:list"
          - "*:describe"
```

#### User-Role Mapping

```yaml
users:
  - username: "alice"
    roles: ["admin"]
  
  - username: "bob"
    roles: ["developer"]
  
  - username: "charlie"
    roles: ["viewer"]
```

### Permission Format

Permissions follow the format: `resource:action`

Resources:
- `clusters`
- `services`
- `tasks`
- `taskDefinitions`
- `containerInstances`
- `attributes`

Actions:
- `create`
- `read` (includes list, describe)
- `update`
- `delete`
- `*` (all actions)

Examples:
- `clusters:create` - Create clusters
- `services:*` - All service operations
- `*:read` - Read all resources

## OAuth 2.0 Integration

### Configuration

```yaml
# kecs-config.yaml
auth:
  type: oauth2
  oauth2:
    provider: "google"  # or "github", "okta", etc.
    clientId: "your-client-id"
    clientSecret: "your-client-secret"
    redirectURL: "http://localhost:8080/auth/callback"
    scopes: ["openid", "profile", "email"]
```

### OAuth Flow

1. **Redirect to Authorization**
   ```
   GET /auth/login?provider=google
   ```

2. **Handle Callback**
   ```
   GET /auth/callback?code=...&state=...
   ```

3. **Receive Token**
   ```json
   {
     "access_token": "kecs_oauth_token_...",
     "token_type": "Bearer",
     "expires_in": 3600
   }
   ```

## Multi-Factor Authentication (MFA)

### TOTP Configuration

```yaml
# kecs-config.yaml
auth:
  mfa:
    enabled: true
    type: "totp"
    issuer: "KECS"
```

### MFA Flow

1. **Initial Login**
   ```bash
   curl -X POST http://localhost:8080/auth/login \
     -d '{"username": "alice", "password": "password"}'
   ```

2. **MFA Challenge Response**
   ```json
   {
     "challenge": "mfa_required",
     "sessionId": "mfa_session_123"
   }
   ```

3. **Submit MFA Code**
   ```bash
   curl -X POST http://localhost:8080/auth/mfa/verify \
     -d '{"sessionId": "mfa_session_123", "code": "123456"}'
   ```

## Service Accounts

For automated systems and CI/CD pipelines.

### Creating Service Accounts

```bash
# Create service account
curl -X POST http://localhost:8080/auth/service-accounts \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "ci-pipeline",
    "description": "CI/CD pipeline service account",
    "roles": ["developer"]
  }'

# Response
{
  "serviceAccountId": "sa_1234567890",
  "apiKey": "kecs_sa_abcdef123456",
  "createdAt": "2024-01-01T00:00:00Z"
}
```

### Using Service Accounts

```bash
# Use the service account API key
curl -X POST http://localhost:8080/v1/UpdateService \
  -H "X-API-Key: kecs_sa_abcdef123456" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.UpdateService" \
  -d '{"cluster": "production", "service": "api", "desiredCount": 3}'
```

## Security Best Practices

### 1. Token Management

- **Rotation**: Regularly rotate API keys and secrets
- **Expiration**: Set appropriate token expiration times
- **Storage**: Never store credentials in code or version control

### 2. TLS Configuration

Always use TLS in production:

```yaml
server:
  tls:
    enabled: true
    certFile: "/path/to/cert.pem"
    keyFile: "/path/to/key.pem"
    minVersion: "1.2"
    cipherSuites:
      - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
      - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
```

### 3. Rate Limiting

Protect against brute force attacks:

```yaml
auth:
  rateLimiting:
    enabled: true
    loginAttempts: 5
    windowMinutes: 15
    blockDurationMinutes: 30
```

### 4. Audit Logging

Enable authentication audit logs:

```yaml
auth:
  audit:
    enabled: true
    logFile: "/var/log/kecs/auth-audit.log"
    events:
      - "login"
      - "logout"
      - "token_created"
      - "token_revoked"
      - "permission_denied"
```

## Troubleshooting

### Common Authentication Errors

#### 401 Unauthorized

```json
{
  "__type": "UnauthorizedException",
  "message": "Authentication required"
}
```

**Causes:**
- Missing authentication headers
- Expired token
- Invalid credentials

#### 403 Forbidden

```json
{
  "__type": "AccessDeniedException",
  "message": "User does not have permission to perform this action"
}
```

**Causes:**
- Insufficient permissions
- Role not assigned
- Resource access denied

### Debug Authentication

Enable debug logging:

```yaml
auth:
  debug: true
  logLevel: "debug"
```

Debug headers:
```bash
curl -v -X POST http://localhost:8080/v1/ListClusters \
  -H "X-Debug-Auth: true" \
  -H "Authorization: Bearer $TOKEN"
```

## Migration Guide

### From No Auth to API Key

1. Generate secure API keys:
   ```bash
   openssl rand -hex 32
   ```

2. Update configuration:
   ```yaml
   auth:
     type: api-key
     apiKeys:
       - name: "migration-key"
         key: "generated-key-here"
         permissions: ["read", "write"]
   ```

3. Update clients to include API key:
   ```bash
   export KECS_API_KEY="generated-key-here"
   ```

### From API Key to JWT

1. Set up JWT configuration
2. Implement login endpoint
3. Update clients to obtain and use JWT tokens
4. Deprecate API keys after transition period

## Environment Variables

Authentication can be configured via environment variables:

```bash
# API Key
export KECS_AUTH_TYPE=api-key
export KECS_API_KEYS="key1:read,write;key2:read"

# JWT
export KECS_AUTH_TYPE=jwt
export KECS_JWT_SECRET="your-secret-key"
export KECS_JWT_EXPIRATION="24h"

# mTLS
export KECS_AUTH_TYPE=mtls
export KECS_TLS_CERT="/path/to/cert.pem"
export KECS_TLS_KEY="/path/to/key.pem"
export KECS_TLS_CA="/path/to/ca.pem"
```