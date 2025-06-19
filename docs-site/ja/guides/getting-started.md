# はじめに

KECS へようこそ！このガイドでは、ローカルな ECS 互換環境のセットアップと実行方法を説明します。

## 前提条件

始める前に、以下がインストールされていることを確認してください：

- **Go 1.21+**: ソースからのビルドに必要
- **Docker**: コンテナの実行に必要
- **Kind**: ローカル Kubernetes クラスター用（オプションですが推奨）
- **kubectl**: Kubernetes との対話用（オプション）

## インストール

### ソースから

```bash
# リポジトリをクローン
git clone https://github.com/nandemo-ya/kecs.git
cd kecs

# バイナリをビルド
make build

# バイナリは ./bin/kecs で利用可能になります
```

### Docker を使用

```bash
# Docker を使用して KECS を実行
docker run -p 8080:8080 -p 8081:8081 ghcr.io/nandemo-ya/kecs:latest
```

## KECS の起動

### ローカル開発

```bash
# サーバーを起動
./bin/kecs server

# または make を使用
make run
```

### Kind を使用

```bash
# Kind クラスターを作成（存在しない場合）
kind create cluster --name kecs-dev

# Kind 統合で KECS を起動
./bin/kecs server --kubernetes-mode=kind
```

## インストールの確認

### ヘルスチェック

```bash
# KECS が実行中か確認
curl http://localhost:8081/health
```

### Web UI

ブラウザを開いて以下にアクセス：
```
http://localhost:8080
```

KECS Web UI ダッシュボードが表示されるはずです。

### API テスト

```bash
# クラスターの一覧
curl -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -d '{}'
```

## 次のステップ

KECS が実行されたので、以下のことができます：

1. [最初のクラスターを作成](/ja/guides/quick-start)
2. [サービスをデプロイ](/ja/guides/services)
3. [Web UI を探索](/ja/guides/web-ui)
4. [タスク定義について学ぶ](/ja/guides/task-definitions)

## トラブルシューティング

### ポートが使用中

ポートが使用中というエラーが表示される場合：

```bash
# ポート 8080 を使用しているものを確認
lsof -i :8080

# 別のポートで KECS を実行
./bin/kecs server --api-port=9080 --admin-port=9081
```

### Kind 接続の問題

KECS が Kind に接続できない場合：

```bash
# Kind クラスターが実行中か確認
kind get clusters

# kubectl コンテキストを確認
kubectl config current-context
```

さらなるトラブルシューティングのヒントについては、[トラブルシューティングガイド](/ja/guides/troubleshooting)を参照してください。