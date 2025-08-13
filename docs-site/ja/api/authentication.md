# 認証ガイド

## 概要

KECS は API アクセスを保護するための複数の認証方法を提供します。デフォルトでは、ローカル開発環境では認証は無効になっていますが、本番環境のデプロイメントでは有効にする必要があります。

## 認証方法

### 1. API キー認証

API キーを使用したシンプルなトークンベース認証。

#### 設定

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

#### 使用方法

`X-API-Key` ヘッダーに API キーを含めます：

```bash
curl -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -H "X-API-Key: kecs_prod_1234567890abcdef" \
  -d '{}'
```

### 2. JWT 認証

より安全でステートレスな認証のための JSON Web Token ベース認証。

#### 設定

```yaml
# kecs-config.yaml
auth:
  type: jwt
  jwt:
    secret: "your-secret-key"  # 本番環境では環境変数を使用
    issuer: "kecs"
    audience: "kecs-api"
    expirationTime: "24h"
```

#### トークンの取得

```bash
# ログインエンドポイント
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "secure-password"
  }'

# レスポンス
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresIn": 86400
}
```

#### トークンの使用

`Authorization` ヘッダーに JWT トークンを含めます：

```bash
curl -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -d '{}'
```

### 3. AWS IAM 認証

AWS SDK と互換性のある AWS IAM 認証情報を使用した認証。

#### 設定

```yaml
# kecs-config.yaml
auth:
  type: iam
  iam:
    region: "us-east-1"
    verifySignature: true
```

#### AWS CLI での使用

```bash
# AWS 認証情報の設定
aws configure

# KECS エンドポイントで AWS CLI を使用
aws ecs list-clusters --endpoint-url http://localhost:8080
```

#### AWS SDK での使用

```python
import boto3

# Python SDK は自動的にリクエストに署名します
ecs = boto3.client(
    'ecs',
    endpoint_url='http://localhost:8080',
    region_name='us-east-1'
)

response = ecs.list_clusters()
```

### 4. mTLS (相互 TLS)

最大のセキュリティを実現する証明書ベースの認証。

#### 設定

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

#### クライアント設定

```bash
# クライアント証明書を使用した curl
curl -X POST https://localhost:8080/v1/ListClusters \
  --cert /path/to/client-cert.pem \
  --key /path/to/client-key.pem \
  --cacert /path/to/ca-cert.pem \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -d '{}'
```

## 認可

### ロールベースアクセス制御 (RBAC)

きめ細かいアクセス制御のためのロールと権限の定義。

#### ロール設定

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

#### ユーザーとロールのマッピング

```yaml
users:
  - username: "alice"
    roles: ["admin"]
  
  - username: "bob"
    roles: ["developer"]
  
  - username: "charlie"
    roles: ["viewer"]
```

### 権限フォーマット

権限は次の形式に従います: `resource:action`

リソース:
- `clusters`
- `services`
- `tasks`
- `taskDefinitions`
- `containerInstances`
- `attributes`

アクション:
- `create`
- `read` (list、describe を含む)
- `update`
- `delete`
- `*` (すべてのアクション)

例:
- `clusters:create` - クラスターの作成
- `services:*` - すべてのサービス操作
- `*:read` - すべてのリソースの読み取り

## OAuth 2.0 統合

### 設定

```yaml
# kecs-config.yaml
auth:
  type: oauth2
  oauth2:
    provider: "google"  # または "github"、"okta" など
    clientId: "your-client-id"
    clientSecret: "your-client-secret"
    redirectURL: "http://localhost:8080/auth/callback"
    scopes: ["openid", "profile", "email"]
```

### OAuth フロー

1. **認可へのリダイレクト**
   ```
   GET /auth/login?provider=google
   ```

2. **コールバックの処理**
   ```
   GET /auth/callback?code=...&state=...
   ```

3. **トークンの受信**
   ```json
   {
     "access_token": "kecs_oauth_token_...",
     "token_type": "Bearer",
     "expires_in": 3600
   }
   ```

## 多要素認証 (MFA)

### TOTP 設定

```yaml
# kecs-config.yaml
auth:
  mfa:
    enabled: true
    type: "totp"
    issuer: "KECS"
```

### MFA フロー

1. **初回ログイン**
   ```bash
   curl -X POST http://localhost:8080/auth/login \
     -d '{"username": "alice", "password": "password"}'
   ```

2. **MFA チャレンジレスポンス**
   ```json
   {
     "challenge": "mfa_required",
     "sessionId": "mfa_session_123"
   }
   ```

3. **MFA コードの送信**
   ```bash
   curl -X POST http://localhost:8080/auth/mfa/verify \
     -d '{"sessionId": "mfa_session_123", "code": "123456"}'
   ```

## サービスアカウント

自動化システムや CI/CD パイプライン用。

### サービスアカウントの作成

```bash
# サービスアカウントの作成
curl -X POST http://localhost:8080/auth/service-accounts \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{
    "name": "ci-pipeline",
    "description": "CI/CD パイプラインサービスアカウント",
    "roles": ["developer"]
  }'

# レスポンス
{
  "serviceAccountId": "sa_1234567890",
  "apiKey": "kecs_sa_abcdef123456",
  "createdAt": "2024-01-01T00:00:00Z"
}
```

### サービスアカウントの使用

```bash
# サービスアカウント API キーの使用
curl -X POST http://localhost:8080/v1/UpdateService \
  -H "X-API-Key: kecs_sa_abcdef123456" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.UpdateService" \
  -d '{"cluster": "production", "service": "api", "desiredCount": 3}'
```

## セキュリティのベストプラクティス

### 1. トークン管理

- **ローテーション**: API キーとシークレットを定期的にローテーション
- **有効期限**: 適切なトークン有効期限の設定
- **保存**: 認証情報をコードやバージョン管理に保存しない

### 2. TLS 設定

本番環境では常に TLS を使用:

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

### 3. レート制限

ブルートフォース攻撃からの保護:

```yaml
auth:
  rateLimiting:
    enabled: true
    loginAttempts: 5
    windowMinutes: 15
    blockDurationMinutes: 30
```

### 4. 監査ログ

認証監査ログの有効化:

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

## トラブルシューティング

### 一般的な認証エラー

#### 401 Unauthorized

```json
{
  "__type": "UnauthorizedException",
  "message": "認証が必要です"
}
```

**原因:**
- 認証ヘッダーの欠落
- トークンの有効期限切れ
- 無効な認証情報

#### 403 Forbidden

```json
{
  "__type": "AccessDeniedException",
  "message": "ユーザーにはこのアクションを実行する権限がありません"
}
```

**原因:**
- 権限不足
- ロールが割り当てられていない
- リソースへのアクセスが拒否された

### 認証のデバッグ

デバッグログを有効化:

```yaml
auth:
  debug: true
  logLevel: "debug"
```

デバッグヘッダー:
```bash
curl -v -X POST http://localhost:8080/v1/ListClusters \
  -H "X-Debug-Auth: true" \
  -H "Authorization: Bearer $TOKEN"
```

## 移行ガイド

### 認証なしから API キーへ

1. セキュアな API キーの生成:
   ```bash
   openssl rand -hex 32
   ```

2. 設定の更新:
   ```yaml
   auth:
     type: api-key
     apiKeys:
       - name: "migration-key"
         key: "generated-key-here"
         permissions: ["read", "write"]
   ```

3. API キーを含めるようクライアントを更新:
   ```bash
   export KECS_API_KEY="generated-key-here"
   ```

### API キーから JWT へ

1. JWT 設定のセットアップ
2. ログインエンドポイントの実装
3. JWT トークンを取得して使用するようクライアントを更新
4. 移行期間後に API キーを廃止

## 環境変数

環境変数経由での認証設定:

```bash
# API キー
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