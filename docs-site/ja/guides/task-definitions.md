# タスク定義ガイド

タスク定義は、コンテナーの実行方法を指定するアプリケーションの設計図です。このガイドでは、KECS でのタスク定義の作成と管理について説明します。

## タスク定義の理解

### タスク定義とは？

タスク定義は、アプリケーションを構成する1つ以上のコンテナーを記述する JSON ドキュメントです。以下を指定します：
- 使用する Docker イメージ
- CPU とメモリの要件
- ネットワークモード
- ログ設定
- 環境変数
- IAM ロール

### タスク定義ファミリー

タスク定義はファミリーにグループ化されます。タスク定義の各リビジョンは、ファミリー内のリビジョン番号を増加させます。

```
webapp:1  → webapp:2  → webapp:3
   ↓          ↓           ↓
 最初の     イメージ     最新の
リビジョン    更新     リビジョン
```

## タスク定義の作成

### 基本的なタスク定義

```json
{
  "family": "simple-app",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "nginx:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "protocol": "tcp"
        }
      ]
    }
  ]
}
```

### マルチコンテナータスク定義

```json
{
  "family": "webapp-with-sidecar",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "containerDefinitions": [
    {
      "name": "webapp",
      "image": "myapp:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080
        }
      ],
      "environment": [
        {
          "name": "LOG_LEVEL",
          "value": "info"
        }
      ],
      "dependsOn": [
        {
          "containerName": "log-router",
          "condition": "START"
        }
      ]
    },
    {
      "name": "log-router",
      "image": "fluentd:latest",
      "essential": true,
      "firelensConfiguration": {
        "type": "fluentd"
      }
    }
  ]
}
```

## コンテナー定義

### 必須プロパティ

```json
{
  "name": "myapp",
  "image": "myregistry/myapp:v1.2.3",
  "essential": true,
  "memory": 512,
  "memoryReservation": 256,
  "cpu": 256
}
```

- **name**: タスク内で一意の名前
- **image**: 使用する Docker イメージ
- **essential**: true の場合、コンテナーが停止するとタスクが失敗
- **memory**: ハードメモリ制限（MiB）
- **memoryReservation**: ソフトメモリ制限
- **cpu**: CPU ユニット（1024 = 1 vCPU）

### ポートマッピング

```json
{
  "portMappings": [
    {
      "containerPort": 8080,
      "hostPort": 80,
      "protocol": "tcp",
      "name": "web"
    }
  ]
}
```

### 環境設定

#### 環境変数

```json
{
  "environment": [
    {
      "name": "APP_ENV",
      "value": "production"
    },
    {
      "name": "API_URL",
      "value": "https://api.example.com"
    }
  ]
}
```

#### シークレット

```json
{
  "secrets": [
    {
      "name": "DB_PASSWORD",
      "valueFrom": "arn:aws:secretsmanager:region:account:secret:db-password"
    },
    {
      "name": "API_KEY",
      "valueFrom": "arn:aws:ssm:region:account:parameter/api-key"
    }
  ]
}
```

### ヘルスチェック

```json
{
  "healthCheck": {
    "command": ["CMD-SHELL", "curl -f http://localhost/health || exit 1"],
    "interval": 30,
    "timeout": 5,
    "retries": 3,
    "startPeriod": 60
  }
}
```

### ログ設定

#### CloudWatch Logs

```json
{
  "logConfiguration": {
    "logDriver": "awslogs",
    "options": {
      "awslogs-group": "/ecs/myapp",
      "awslogs-region": "us-east-1",
      "awslogs-stream-prefix": "webapp"
    }
  }
}
```

#### FireLens

```json
{
  "logConfiguration": {
    "logDriver": "awsfirelens",
    "options": {
      "Name": "cloudwatch",
      "region": "us-east-1",
      "log_group_name": "/ecs/myapp",
      "log_stream_prefix": "firelens/"
    }
  }
}
```

## 高度な機能

### コンテナー依存関係

```json
{
  "containerDefinitions": [
    {
      "name": "database",
      "image": "postgres:13",
      "essential": true
    },
    {
      "name": "app",
      "image": "myapp:latest",
      "essential": true,
      "dependsOn": [
        {
          "containerName": "database",
          "condition": "HEALTHY"
        }
      ]
    }
  ]
}
```

依存関係の条件:
- **START**: コンテナーが開始した
- **COMPLETE**: コンテナーが完了まで実行された
- **SUCCESS**: コンテナーが正常に終了した
- **HEALTHY**: コンテナーが正常である

### ボリューム

#### バインドマウント

```json
{
  "volumes": [
    {
      "name": "app-config",
      "host": {
        "sourcePath": "/etc/myapp"
      }
    }
  ],
  "containerDefinitions": [
    {
      "name": "app",
      "mountPoints": [
        {
          "sourceVolume": "app-config",
          "containerPath": "/config",
          "readOnly": true
        }
      ]
    }
  ]
}
```

#### EFS ボリューム

```json
{
  "volumes": [
    {
      "name": "efs-storage",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-12345678",
        "rootDirectory": "/data",
        "transitEncryption": "ENABLED",
        "authorizationConfig": {
          "accessPointId": "fsap-12345678",
          "iam": "ENABLED"
        }
      }
    }
  ]
}
```

### リソース要件

```json
{
  "requiresCompatibilities": ["EC2"],
  "placementConstraints": [
    {
      "type": "memberOf",
      "expression": "attribute:ecs.instance-type =~ c5.*"
    }
  ],
  "cpu": "2048",
  "memory": "4096",
  "gpuCount": 1
}
```

### ネットワーク設定

#### ネットワークモード

- **awsvpc**: 各タスクが独自のネットワークインターフェイスを取得
- **bridge**: Docker の組み込みブリッジネットワークを使用
- **host**: ホストのネットワークを使用
- **none**: ネットワークなし

```json
{
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"]
}
```

### IAM ロール

#### タスクロール

コンテナーに AWS サービスへの権限を付与:

```json
{
  "taskRoleArn": "arn:aws:iam::account:role/ecsTaskRole"
}
```

#### 実行ロール

ECS にイメージのプルとログの書き込み権限を付与:

```json
{
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole"
}
```

## タスク定義の操作

### タスク定義の登録

```bash
# ファイルから
aws ecs register-task-definition \
  --cli-input-json file://task-definition.json \
  --endpoint-url http://localhost:8080

# インライン
aws ecs register-task-definition \
  --family webapp \
  --network-mode awsvpc \
  --requires-compatibilities FARGATE \
  --cpu 256 \
  --memory 512 \
  --container-definitions '[
    {
      "name": "web",
      "image": "nginx:latest",
      "portMappings": [{"containerPort": 80}],
      "essential": true
    }
  ]' \
  --endpoint-url http://localhost:8080
```

### タスク定義の一覧表示

```bash
# ファミリーの一覧
aws ecs list-task-definition-families \
  --endpoint-url http://localhost:8080

# リビジョンの一覧
aws ecs list-task-definitions \
  --family-prefix webapp \
  --endpoint-url http://localhost:8080
```

### タスク定義の詳細表示

```bash
# 最新リビジョン
aws ecs describe-task-definition \
  --task-definition webapp \
  --endpoint-url http://localhost:8080

# 特定のリビジョン
aws ecs describe-task-definition \
  --task-definition webapp:3 \
  --endpoint-url http://localhost:8080
```

### タスク定義の登録解除

```bash
aws ecs deregister-task-definition \
  --task-definition webapp:1 \
  --endpoint-url http://localhost:8080
```

## ベストプラクティス

### 1. コンテナーイメージ

- `latest` ではなく特定のタグを使用
- イメージを小さく安全に保つ
- マルチステージビルドを使用
- 脆弱性のイメージをスキャン

### 2. リソース割り当て

- 制限とリクエストの両方を設定
- スパイクのための余裕を残す
- 実際の使用状況を監視
- リソース予約を賢く使用

### 3. 設定

- 設定には環境変数を使用
- シークレットは Secrets Manager または SSM に保存
- 機密でない設定にはパラメータストアを使用
- タスク定義をバージョン管理

### 4. ログ

- 常にログを設定
- 構造化ログを使用
- 適切な保持期間を設定
- ログ集約を検討

### 5. ヘルスチェック

- アプリケーションヘルスエンドポイントを実装
- 妥当なタイムアウトと間隔を設定
- 起動が遅いアプリには起動期間を使用
- ヘルスチェックメトリクスを監視

## 一般的なパターン

### サイドカーパターン

```json
{
  "containerDefinitions": [
    {
      "name": "app",
      "image": "myapp:latest"
    },
    {
      "name": "proxy",
      "image": "envoyproxy/envoy:latest",
      "links": ["app"]
    }
  ]
}
```

### Init コンテナーパターン

```json
{
  "containerDefinitions": [
    {
      "name": "init",
      "image": "busybox",
      "essential": false,
      "command": ["sh", "-c", "echo '初期化中...'"]
    },
    {
      "name": "app",
      "image": "myapp:latest",
      "essential": true,
      "dependsOn": [{
        "containerName": "init",
        "condition": "SUCCESS"
      }]
    }
  ]
}
```

### アンバサダーパターン

```json
{
  "containerDefinitions": [
    {
      "name": "app",
      "image": "myapp:latest",
      "environment": [{
        "name": "PROXY_URL",
        "value": "http://localhost:8080"
      }]
    },
    {
      "name": "ambassador",
      "image": "ambassador:latest",
      "portMappings": [{
        "containerPort": 8080
      }]
    }
  ]
}
```

## トラブルシューティング

### タスク定義検証エラー

- JSON 構文を確認
- 必須フィールドを検証
- CPU/メモリの組み合わせを検証
- イメージのアクセシビリティを確認

### コンテナー起動失敗

- イメージプル権限を確認
- 環境変数を検証
- ヘルスチェックコマンドを確認
- ボリュームマウントパスを確認

### パフォーマンスの問題

- リソース使用率を監視
- メモリリークを確認
- CPU スロットリングを確認
- コンテナー起動を最適化

トラブルシューティングの詳細については、[トラブルシューティングガイド](/ja/guides/troubleshooting)を参照してください。