# コード生成

KECSは、公式のAWS API定義からAWS API互換の型を生成するコード生成を使用しています。このアプローチにより、型安全性を維持しながら、AWS CLIおよびSDKとの完全な互換性を確保できます。

## 概要

AWS SDKを直接使用する代わりに、KECSはAWS Smithy JSON定義から独自の型を生成します。これにより以下が提供されます：

- **AWS CLI互換性**: 生成された型は適切なJSONフィールド名を使用
- **型安全性**: すべてのAPI操作に対するコンパイル時チェック
- **SDKの依存関係なし**: バイナリサイズと複雑性を削減
- **一貫したインターフェース**: すべてのサービスが同じパターンに従う

## クイックスタート

### サービスの型を生成

```bash
# AWS API定義をダウンロード
cd controlplane
./scripts/download-aws-api-definitions.sh ecs

# Goコードを生成
cd cmd/codegen
go run . -input ecs.json -output ../../internal/ecs/generated
```

### 生成された型を使用

```go
import api "github.com/nandemo-ya/kecs/controlplane/internal/ecs/generated"

// リクエストを作成
req := &api.CreateClusterRequest{
    ClusterName: aws.String("my-cluster"),
}

// APIを呼び出し
resp, err := handler.CreateCluster(ctx, req)
```

## 生成されたコードの構造

各サービスは3つのファイルを生成します：

### types.go
JSONタグ付きのすべてのリクエスト/レスポンス型：
```go
type CreateClusterRequest struct {
    ClusterName *string `json:"clusterName,omitempty"`
    Tags []Tag `json:"tags,omitempty"`
}
```

### operations.go
サービスインターフェースの定義：
```go
type AmazonECSAPI interface {
    CreateCluster(ctx context.Context, input *CreateClusterRequest) (*CreateClusterResponse, error)
    DeleteCluster(ctx context.Context, input *DeleteClusterRequest) (*DeleteClusterResponse, error)
    // ... その他の操作
}
```

### routing.go
HTTPリクエストのルーティングとマーシャリング：
```go
func (r *Router) Route(w http.ResponseWriter, req *http.Request) {
    action := r.extractAction(req)
    switch action {
    case "CreateCluster":
        r.handleCreateCluster(w, req)
    // ... その他のアクション
    }
}
```

## サービスの実装

### 1. インターフェースを実装

```go
type ecsHandler struct {
    storage storage.Storage
}

func (h *ecsHandler) CreateCluster(ctx context.Context, input *api.CreateClusterRequest) (*api.CreateClusterResponse, error) {
    // 入力を検証
    if input.ClusterName == nil || *input.ClusterName == "" {
        return nil, errors.New("cluster name is required")
    }

    // ストレージにクラスタを作成
    cluster := &model.Cluster{
        Name:   *input.ClusterName,
        Status: "ACTIVE",
    }
    
    if err := h.storage.CreateCluster(ctx, cluster); err != nil {
        return nil, err
    }

    // レスポンスを返す
    return &api.CreateClusterResponse{
        Cluster: &api.Cluster{
            ClusterArn:  aws.String(cluster.ARN),
            ClusterName: aws.String(cluster.Name),
            Status:      aws.String(cluster.Status),
        },
    }, nil
}
```

### 2. HTTPサーバーをセットアップ

```go
// ハンドラーを作成
handler := &ecsHandler{
    storage: storage,
}

// ルーターを作成
router := api.NewRouter(handler)

// HTTPサーバーをセットアップ
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    router.Route(w, r)
})

http.ListenAndServe(":8080", nil)
```

## サポートされているサービス

| サービス | ステータス | 備考 |
|---------|------------|------|
| ECS | ✅ 完了 | すべての操作が実装済み |
| STS | ✅ 生成済み | 実装準備ができた型 |
| Secrets Manager | ✅ 生成済み | 実装準備ができた型 |
| IAM | ⚠️ 部分的 | 一部のユニオン型が欠落 |
| CloudWatch Logs | ⚠️ 部分的 | ユニオン型とストリーミング型が欠落 |
| S3 | ⚠️ 部分的 | ユニオン型とストリーミング型が欠落 |
| SSM | ⚠️ 部分的 | ユニオン型が欠落 |

## 新しいサービスの追加

### 1. API定義をダウンロード

```bash
cd controlplane
./scripts/download-aws-api-definitions.sh <service-name>
```

利用可能なサービス：
- `cloudwatch-logs`
- `iam`
- `s3`
- `secretsmanager`
- `ssm`
- `sts`

### 2. コードを生成

```bash
cd cmd/codegen
go run . -input <service>.json -output ../../internal/<service>/generated
```

### 3. コンパイルを確認

```bash
cd ../../internal/<service>/generated
go build ./...
```

### 4. テストを作成

```go
package generated_test

import (
    "encoding/json"
    "testing"
    
    api "github.com/nandemo-ya/kecs/controlplane/internal/<service>/generated"
)

func TestJSONMarshaling(t *testing.T) {
    // JSONフィールドがcamelCaseを使用することをテスト
    req := &api.SomeRequest{
        FieldName: aws.String("value"),
    }
    
    data, _ := json.Marshal(req)
    var m map[string]interface{}
    json.Unmarshal(data, &m)
    
    if _, ok := m["fieldName"]; !ok {
        t.Error("Expected camelCase field name")
    }
}
```

## トラブルシューティング

### よくある問題

**1. コンパイルエラー**

生成されたコードがコンパイルされない場合、以下を確認してください：
- 欠落しているユニオン型定義
- サポートされていないストリーミング型
- 循環依存関係

**2. 欠落している型**

一部の複雑な型は正しく生成されない可能性があります：
```bash
# エラーを確認
go build ./...

# 未定義の型エラーを探す
# 必要に応じて手動で型定義を追加
```

**3. JSONフィールド名**

フィールド名がcamelCaseであることを確認：
```bash
# curlでテスト
curl -X POST http://localhost:8080/ \
  -H "X-Amz-Target: AmazonECS.ListClusters" \
  -d '{}' | jq .
```

## 高度なトピック

### カスタム型の処理

正しく生成されない型には、手動定義を作成します：

```go
// internal/<service>/generated/custom_types.go
package api

// ユニオン型の例
type FilterType struct {
    Name   *string
    Values []string
}

// ドキュメント型の例  
type DocumentValue map[string]interface{}
```

### 生成されたコードの拡張

生成されたファイルを直接変更しないでください。代わりに：

1. ラッパー型を作成
2. 埋め込みを使用
3. 別のファイルにヘルパー関数を追加

```go
// internal/<service>/helpers.go
package service

import api "github.com/nandemo-ya/kecs/controlplane/internal/<service>/generated"

// ヘルパー関数
func NewCreateRequest(name string) *api.CreateRequest {
    return &api.CreateRequest{
        Name: aws.String(name),
    }
}
```

## ベストプラクティス

1. **定期的に再生成**: API定義を最新に保つ
2. **JSON出力をテスト**: AWS CLI互換性を確認
3. **制限事項を文書化**: 欠落している操作を記録
4. **ポインタを使用**: オプションフィールドにはAWS SDKパターンに従う
5. **エラーを処理**: 適切なAWSエラーコードを返す

## 次のステップ

- [アーキテクチャ概要](./architecture.md) - KECSアーキテクチャを理解
- [KECSをビルド](./building.md) - ソースからビルド
- [テストガイド](./testing.md) - 実装のテストを書く