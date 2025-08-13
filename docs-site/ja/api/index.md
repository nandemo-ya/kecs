# API リファレンス

KECS は Amazon ECS API 仕様を実装し、既存の ECS ツールや SDK との完全な互換性を提供します。

## 概要

すべての API リクエストは AWS API の規約に従います：

- **エンドポイント**: `http://localhost:8080/v1/<Action>`
- **メソッド**: POST
- **Content-Type**: `application/x-amz-json-1.1`
- **Target ヘッダー**: `X-Amz-Target: AmazonEC2ContainerServiceV20141113.<Action>`

## 認証

現在、KECS はローカル開発では認証を必要としません。本番環境のデプロイメントでは、以下を通じて認証を設定できます：

- API キー
- JWT トークン
- mTLS

詳細は[認証ガイド](/ja/api/authentication)を参照してください。

## 利用可能な API

### クラスター管理
- [CreateCluster](/ja/api/clusters#createcluster)
- [DeleteCluster](/ja/api/clusters#deletecluster)
- [DescribeClusters](/ja/api/clusters#describeclusters)
- [ListClusters](/ja/api/clusters#listclusters)
- [UpdateCluster](/ja/api/clusters#updatecluster)

### サービス管理
- [CreateService](/ja/api/services#createservice)
- [DeleteService](/ja/api/services#deleteservice)
- [DescribeServices](/ja/api/services#describeservices)
- [ListServices](/ja/api/services#listservices)
- [UpdateService](/ja/api/services#updateservice)

### タスク管理
- [RunTask](/ja/api/tasks#runtask)
- [StopTask](/ja/api/tasks#stoptask)
- [DescribeTasks](/ja/api/tasks#describetasks)
- [ListTasks](/ja/api/tasks#listtasks)

### タスク定義管理
- [RegisterTaskDefinition](/ja/api/task-definitions#registertaskdefinition)
- [DeregisterTaskDefinition](/ja/api/task-definitions#deregistertaskdefinition)
- [DescribeTaskDefinition](/ja/api/task-definitions#describetaskdefinition)
- [ListTaskDefinitions](/ja/api/task-definitions#listtaskdefinitions)

## リクエスト例

```bash
curl -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -d '{
    "maxResults": 10
  }'
```

## レスポンス形式

すべてのレスポンスは標準的な AWS API レスポンス形式に従います：

```json
{
  "clusterArns": [
    "arn:aws:ecs:us-east-1:000000000000:cluster/default"
  ],
  "nextToken": null
}
```

## エラーハンドリング

エラーは適切な HTTP ステータスコードとエラー詳細とともに返されます：

```json
{
  "__type": "ClientException",
  "message": "Cluster not found"
}
```

一般的なエラータイプ：
- `ClientException`: クライアント側のエラー (400)
- `ServerException`: サーバー側のエラー (500)
- `ResourceNotFoundException`: リソースが見つからない (404)
- `InvalidParameterException`: 無効なパラメータ (400)

## SDK の使用

KECS は AWS SDK と互換性があります。エンドポイントを設定してください：

### AWS CLI
```bash
aws ecs list-clusters --endpoint-url http://localhost:8080
```

### Python (boto3)
```python
import boto3

ecs = boto3.client('ecs', endpoint_url='http://localhost:8080')
clusters = ecs.list_clusters()
```

### Go SDK
```go
import (
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/ecs"
)

sess := session.Must(session.NewSession(&aws.Config{
    Endpoint: aws.String("http://localhost:8080"),
}))

svc := ecs.New(sess)
```

## レート制限

KECS は乱用を防ぐためにレート制限を実装しています：
- デフォルト: IP あたり毎秒 100 リクエスト
- `--rate-limit` フラグで設定可能

## WebSocket API

リアルタイム更新のために、KECS は WebSocket エンドポイントを提供します：
- **エンドポイント**: `ws://localhost:8080/ws`
- **プロトコル**: JSON メッセージ
- 詳細は [WebSocket ガイド](/ja/api/websocket)を参照