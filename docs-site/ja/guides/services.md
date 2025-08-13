# サービスの操作

ECS のサービスは、指定された数のタスクを維持する長時間実行されるアプリケーションです。このガイドでは、KECS でのサービスの作成、管理、監視について説明します。

## サービスの概念

### サービスとは？

サービスを使用すると、ECS クラスター内で指定された数のタスク定義のインスタンスを同時に実行および維持できます。タスクが失敗または停止した場合、サービススケジューラーは別のインスタンスを起動して置き換えます。

### 主な機能

- **希望数**: 指定された数の実行中のタスクを維持
- **ロードバランシング**: タスク間でトラフィックを分散
- **サービスディスカバリー**: サービス間通信を可能にする
- **ローリングアップデート**: ダウンタイムなしでタスクを更新
- **オートスケーリング**: メトリクスに基づいてスケール

## サービスの作成

### 基本的なサービス作成

```bash
# シンプルなサービスの作成
aws ecs create-service \
  --cluster production \
  --service-name web-app \
  --task-definition webapp:1 \
  --desired-count 3 \
  --launch-type FARGATE \
  --endpoint-url http://localhost:8080
```

### ロードバランサー付きサービス

```json
{
  "cluster": "production",
  "serviceName": "web-app",
  "taskDefinition": "webapp:1",
  "desiredCount": 3,
  "launchType": "FARGATE",
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:region:account-id:targetgroup/my-targets/1234567890123456",
      "containerName": "web",
      "containerPort": 80
    }
  ],
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345", "subnet-67890"],
      "securityGroups": ["sg-12345"],
      "assignPublicIp": "ENABLED"
    }
  }
}
```

### サービスディスカバリー付きサービス

```json
{
  "cluster": "production",
  "serviceName": "api-service",
  "taskDefinition": "api:1",
  "desiredCount": 2,
  "launchType": "FARGATE",
  "serviceRegistries": [
    {
      "registryArn": "arn:aws:servicediscovery:region:account-id:service/srv-1234567890",
      "containerName": "api",
      "containerPort": 8080
    }
  ],
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345"],
      "securityGroups": ["sg-12345"]
    }
  }
}
```

## サービス設定

### デプロイメント設定

サービスの更新方法を制御:

```json
{
  "deploymentConfiguration": {
    "maximumPercent": 200,
    "minimumHealthyPercent": 100,
    "deploymentCircuitBreaker": {
      "enable": true,
      "rollback": true
    }
  }
}
```

- **maximumPercent**: デプロイメント中の最大タスク数（希望数の %）
- **minimumHealthyPercent**: デプロイメント中の最小正常タスク数
- **deploymentCircuitBreaker**: 失敗したデプロイメントを自動的にロールバック

### 配置戦略

クラスター全体にタスクを分散:

```json
{
  "placementStrategy": [
    {
      "type": "spread",
      "field": "attribute:ecs.availability-zone"
    },
    {
      "type": "binpack",
      "field": "memory"
    }
  ]
}
```

戦略タイプ:
- **spread**: フィールドに基づいて均等に分散
- **binpack**: リソース使用率に基づいてタスクをパック
- **random**: タスクをランダムに配置

### 配置制約

タスクの実行場所を制御:

```json
{
  "placementConstraints": [
    {
      "type": "memberOf",
      "expression": "attribute:ecs.instance-type =~ t3.*"
    }
  ]
}
```

## サービスの管理

### サービスの更新

サービス設定またはタスク定義を更新:

```bash
# 新しいタスク定義に更新
aws ecs update-service \
  --cluster production \
  --service web-app \
  --task-definition webapp:2 \
  --endpoint-url http://localhost:8080

# 希望数を更新
aws ecs update-service \
  --cluster production \
  --service web-app \
  --desired-count 5 \
  --endpoint-url http://localhost:8080
```

### サービスのスケーリング

#### 手動スケーリング

```bash
aws ecs update-service \
  --cluster production \
  --service web-app \
  --desired-count 10 \
  --endpoint-url http://localhost:8080
```

#### オートスケーリング

オートスケーリングポリシーの設定:

```bash
# スケーラブルターゲットの登録
aws application-autoscaling register-scalable-target \
  --service-namespace ecs \
  --scalable-dimension ecs:service:DesiredCount \
  --resource-id service/production/web-app \
  --min-capacity 2 \
  --max-capacity 10

# スケーリングポリシーの作成
aws application-autoscaling put-scaling-policy \
  --policy-name cpu-scaling \
  --service-namespace ecs \
  --scalable-dimension ecs:service:DesiredCount \
  --resource-id service/production/web-app \
  --policy-type TargetTrackingScaling \
  --target-tracking-scaling-policy-configuration file://scaling-policy.json
```

### サービスヘルスチェック

サービスはヘルスチェックを使用してタスクの健全性を判断:

```json
{
  "healthCheckGracePeriodSeconds": 60,
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:...",
      "containerName": "web",
      "containerPort": 80
    }
  ]
}
```

## サービスの監視

### サービスメトリクス

サービスステータスの表示:

```bash
# サービスの詳細表示
aws ecs describe-services \
  --cluster production \
  --services web-app \
  --endpoint-url http://localhost:8080
```

監視すべき主要メトリクス:
- **runningCount**: 実行中のタスク数
- **pendingCount**: 保留中のタスク数
- **desiredCount**: 希望タスク数
- **deployments**: アクティブなデプロイメント

### サービスイベント

サービスイベントの表示:

```bash
aws ecs describe-services \
  --cluster production \
  --services web-app \
  --endpoint-url http://localhost:8080 \
  | jq '.services[0].events[:5]'
```

### タスクステータス

個々のタスクステータスの確認:

```bash
# サービスのタスク一覧
aws ecs list-tasks \
  --cluster production \
  --service-name web-app \
  --endpoint-url http://localhost:8080

# タスクの詳細表示
aws ecs describe-tasks \
  --cluster production \
  --tasks <task-arn> \
  --endpoint-url http://localhost:8080
```

## サービスパターン

### ブルー/グリーンデプロイメント

1. 新しいタスク定義を作成
2. 新しいバージョンで新しいサービスを作成
3. 新しいサービスをテスト
4. トラフィックを新しいサービスに切り替え
5. 古いサービスを削除

### カナリアデプロイメント

重み付けターゲットグループを使用:

```json
{
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:...:targetgroup/blue/...",
      "containerName": "app",
      "containerPort": 80
    },
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:...:targetgroup/green/...",
      "containerName": "app",
      "containerPort": 80
    }
  ]
}
```

### サイドカーパターン

タスク内に複数のコンテナーをデプロイ:

```json
{
  "containerDefinitions": [
    {
      "name": "app",
      "image": "myapp:latest",
      "portMappings": [{"containerPort": 8080}]
    },
    {
      "name": "envoy",
      "image": "envoyproxy/envoy:latest",
      "portMappings": [{"containerPort": 9901}]
    }
  ]
}
```

## サービスディスカバリー

### プライベート DNS 名前空間

サービスディスカバリー用の名前空間を作成:

```bash
aws servicediscovery create-private-dns-namespace \
  --name local \
  --vpc vpc-12345 \
  --endpoint-url http://localhost:8080
```

### サービスの登録

```json
{
  "serviceRegistries": [
    {
      "registryArn": "arn:aws:servicediscovery:...",
      "containerName": "app",
      "containerPort": 8080
    }
  ]
}
```

### サービスの検出

サービスは DNS を使用して相互に検出可能:
```
http://service-name.namespace.local:8080
```

## ベストプラクティス

### 1. リソース割り当て

- 適切な CPU とメモリ制限を設定
- 重要なサービスにはリソース予約を使用
- リソース使用率を監視

### 2. ヘルスチェック

- 適切なヘルスチェック間隔を設定
- 妥当な猶予期間を設定
- コンテナーヘルスチェックを使用

### 3. デプロイメント戦略

- ゼロダウンタイムデプロイメントにはローリングアップデートを使用
- 自動ロールバックのためにサーキットブレーカーを有効化
- まずステージング環境でデプロイメントをテスト

### 4. 監視

- CloudWatch アラームを設定
- サービスイベントを監視
- デプロイメント成功率を追跡

### 5. セキュリティ

- タスクに IAM ロールを使用
- セキュリティグループを制限
- 機密データの暗号化を有効化

## トラブルシューティング

### サービスが起動しない

1. タスク定義が有効であることを確認
2. クラスターに利用可能なリソースがあることを確認
3. セキュリティグループとネットワーク設定を確認
4. エラーのサービスイベントを確認

### タスクが失敗し続ける

1. コンテナーログを確認
2. イメージがアクセス可能であることを確認
3. リソース制約を確認
4. タスク停止理由を確認

### デプロイメントが遅い

1. デプロイメント設定を調整
2. ヘルスチェック設定を確認
3. リソースの可用性を監視
4. 配置制約を確認

詳細なトラブルシューティングについては、[トラブルシューティングガイド](/ja/guides/troubleshooting)を参照してください。