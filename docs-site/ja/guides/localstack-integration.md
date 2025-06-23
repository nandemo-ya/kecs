# LocalStack 統合ガイド

KECS は LocalStack と統合して、完全なローカル AWS 環境を提供します。このガイドでは、KECS での LocalStack のセットアップと使用方法について説明します。

## 概要

LocalStack 統合により以下が可能になります：
- ローカル AWS サービスエミュレーション（S3、DynamoDB、SQS など）
- IAM ロールシミュレーション
- CloudWatch ログとメトリクス
- Secrets Manager と SSM パラメータストア
- Route 53 によるサービスディスカバリ

## 自動サイドカーインジェクション（透過的プロキシ）

KECS は **AWS_ENDPOINT 設定を必要とせずに** AWS API 呼び出しを自動的に LocalStack にルーティングする強力な透過的プロキシ機能を提供します。これは通常の LocalStack セットアップと比較して大きな利点です。

### 仕組み

1. KECS がコンテナ内の AWS SDK 使用を検出
2. 透過的プロキシサイドカーを自動的に注入
3. すべての AWS API 呼び出しを透過的に LocalStack にルーティング
4. **コード変更や AWS_ENDPOINT 設定は不要**

### 主な利点

- **ゼロ設定**：既存の AWS アプリケーションが変更なしで動作
- **AWS_ENDPOINT 不要**：標準的な LocalStack 使用とは異なり、`endpoint_url` や `AWS_ENDPOINT_URL` の設定が不要
- **本番環境対応コード**：同じコードがローカル（LocalStack）と本番（実際の AWS）の両方で動作
- **自動検出**：KECS がプロキシ注入の必要性を自動的に検出

### 透過的プロキシの動作原理

透過的プロキシメカニズムは以下のように動作します：

1. **iptables ルール**：KECS が Pod 内で iptables ルールを設定し、AWS ドメインへの送信 HTTPS トラフィックを傍受
2. **DNS 解決**：AWS サービスドメイン（例：`s3.amazonaws.com`）は通常通り解決
3. **トラフィック傍受**：プロキシサイドカーが AWS エンドポイントへの接続を傍受
4. **リクエストルーティング**：すべてのヘッダーと認証を保持したまま、リクエストを透過的に LocalStack に転送
5. **レスポンス処理**：LocalStack のレスポンスが AWS から来たかのようにアプリケーションに返される

この方式が環境変数注入より優れている理由：
- あらゆる AWS SDK やツールで動作（`AWS_ENDPOINT_URL` を尊重するものだけでなく）
- 環境変数が無視されたり上書きされるリスクがない
- 動的エンドポイント検出をサポート（例：S3 仮想ホスト形式の URL）
- アプリケーション側の認識が一切不要

## AWS サービスの使用

### S3 統合

コンテナから S3 バケットにアクセス：

```python
import boto3

# endpoint_url パラメータは不要！
# KECS の透過的プロキシが自動的に LocalStack にルーティング
s3 = boto3.client('s3')  # そのまま動作、AWS_ENDPOINT_URL 不要！

# バケット一覧
buckets = s3.list_buckets()

# ファイルアップロード
s3.upload_file('local.txt', 'my-bucket', 'remote.txt')

# 通常の LocalStack 使用時との比較（KECS では不要）：
# s3 = boto3.client('s3', endpoint_url='http://localhost:4566')  # 不要！
```

### DynamoDB 統合

DynamoDB テーブルの使用：

```python
import boto3

# こちらもエンドポイント設定は不要！
dynamodb = boto3.resource('dynamodb')  # 自動的に LocalStack を使用
table = dynamodb.Table('users')

# アイテム追加
table.put_item(Item={
    'userId': '123',
    'name': 'John Doe',
    'email': 'john@example.com'
})

# クエリ
response = table.get_item(Key={'userId': '123'})

# KECS の透過的プロキシなしでは以下が必要：
# dynamodb = boto3.resource('dynamodb', endpoint_url='http://localhost:4566')
```

---

*完全な英語版ドキュメントは[こちら](/guides/localstack-integration)をご覧ください。*