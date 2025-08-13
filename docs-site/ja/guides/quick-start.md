# クイックスタートガイド

このガイドでは、最初の ECS クラスターを作成し、KECS 上にシンプルなアプリケーションをデプロイする手順を説明します。

## 概要

このチュートリアルでは、以下を行います：
1. ECS クラスターの作成
2. タスク定義の登録
3. サービスの作成と実行
4. アプリケーションへのアクセス

## 前提条件

- KECS がインストールされ、実行中であること（[はじめに](/ja/guides/getting-started)を参照）
- AWS CLI が設定されていること（ECS コマンドの使用のため）

## ステップ 1: クラスターの作成

まず、ECS クラスターを作成しましょう：

```bash
# AWS CLI を使用
aws ecs create-cluster --cluster-name my-first-cluster \
  --endpoint-url http://localhost:8080

# または curl を使用
curl -X POST http://localhost:8080/v1/CreateCluster \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster" \
  -d '{
    "clusterName": "my-first-cluster"
  }'
```

期待されるレスポンス：
```json
{
  "cluster": {
    "clusterArn": "arn:aws:ecs:ap-northeast-1:000000000000:cluster/my-first-cluster",
    "clusterName": "my-first-cluster",
    "status": "ACTIVE"
  }
}
```

## ステップ 2: タスク定義の登録

`nginx-task.json` というファイルを作成します：

```json
{
  "family": "nginx-web",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "nginx",
      "image": "nginx:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "protocol": "tcp"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/nginx-web",
          "awslogs-region": "ap-northeast-1",
          "awslogs-stream-prefix": "nginx"
        }
      }
    }
  ]
}
```

タスク定義を登録します：

```bash
# AWS CLI を使用
aws ecs register-task-definition \
  --cli-input-json file://nginx-task.json \
  --endpoint-url http://localhost:8080

# または curl を使用
curl -X POST http://localhost:8080/v1/RegisterTaskDefinition \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RegisterTaskDefinition" \
  -d @nginx-task.json
```

## ステップ 3: サービスの作成

次に、タスクを実行するサービスを作成しましょう：

```bash
# AWS CLI を使用
aws ecs create-service \
  --cluster my-first-cluster \
  --service-name nginx-service \
  --task-definition nginx-web:1 \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}" \
  --endpoint-url http://localhost:8080
```

またはサービス定義ファイル `nginx-service.json` を作成：

```json
{
  "cluster": "my-first-cluster",
  "serviceName": "nginx-service",
  "taskDefinition": "nginx-web:1",
  "desiredCount": 2,
  "launchType": "FARGATE",
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345"],
      "securityGroups": ["sg-12345"],
      "assignPublicIp": "ENABLED"
    }
  }
}
```

そしてサービスを作成：

```bash
curl -X POST http://localhost:8080/v1/CreateService \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateService" \
  -d @nginx-service.json
```

## ステップ 4: デプロイメントの確認

### サービスステータスの確認

```bash
# サービスの一覧
aws ecs list-services --cluster my-first-cluster \
  --endpoint-url http://localhost:8080

# サービスの詳細
aws ecs describe-services \
  --cluster my-first-cluster \
  --services nginx-service \
  --endpoint-url http://localhost:8080
```

### 実行中のタスクの確認

```bash
# タスクの一覧
aws ecs list-tasks --cluster my-first-cluster \
  --endpoint-url http://localhost:8080

# タスクの詳細
aws ecs describe-tasks \
  --cluster my-first-cluster \
  --tasks <task-arn> \
  --endpoint-url http://localhost:8080
```


## ステップ 5: アプリケーションへのアクセス

KECS は Kubernetes でコンテナを実行するため、kubectl を使用してアプリケーションにアクセスできます：

```bash
# Pod の取得
kubectl get pods -n my-first-cluster

# nginx にアクセスするためのポートフォワード
kubectl port-forward -n my-first-cluster pod/nginx-service-0 8080:80

# http://localhost:8080 で nginx にアクセス
```

## 次のステップ

おめでとうございます！KECS に最初のアプリケーションを正常にデプロイしました。次に試すことができること：

### 1. サービスのスケール

```bash
aws ecs update-service \
  --cluster my-first-cluster \
  --service nginx-service \
  --desired-count 3 \
  --endpoint-url http://localhost:8080
```

### 2. タスク定義の更新

`nginx-task.json` を修正して別のイメージを使用したり、環境変数を追加したりして：

```bash
# 新しいリビジョンを登録
aws ecs register-task-definition \
  --cli-input-json file://nginx-task.json \
  --endpoint-url http://localhost:8080

# 新しいリビジョンを使用するようサービスを更新
aws ecs update-service \
  --cluster my-first-cluster \
  --service nginx-service \
  --task-definition nginx-web:2 \
  --endpoint-url http://localhost:8080
```

### 3. 高度な機能の探索

- [ロードバランシング](/ja/guides/load-balancing)
- [サービスディスカバリ](/ja/guides/service-discovery)
- [オートスケーリング](/ja/guides/auto-scaling)
- [LocalStack 統合](/ja/guides/localstack-integration)

## クリーンアップ

実験が終わったら、リソースをクリーンアップしましょう：

```bash
# サービスの削除
aws ecs delete-service \
  --cluster my-first-cluster \
  --service nginx-service \
  --force \
  --endpoint-url http://localhost:8080

# クラスターの削除
aws ecs delete-cluster \
  --cluster my-first-cluster \
  --endpoint-url http://localhost:8080
```

## トラブルシューティング

### サービスが起動しない

タスクのステータスを確認：
```bash
aws ecs describe-tasks --cluster my-first-cluster \
  --tasks <task-arn> \
  --endpoint-url http://localhost:8080
```

`stoppedReason` フィールドのエラーメッセージを確認してください。

### コンテナログ

kubectl を使用してコンテナログを表示：
```bash
kubectl logs -n my-first-cluster <pod-name>
```

### よくある問題

1. **イメージプルエラー**: コンテナイメージがアクセス可能であることを確認
2. **リソース制約**: Kubernetes クラスターに十分なリソースがあるか確認
3. **ネットワークの問題**: セキュリティグループとネットワーク設定を確認

さらなるヘルプについては、[トラブルシューティングガイド](/ja/guides/troubleshooting)を参照してください。