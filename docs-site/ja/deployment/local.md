# ローカル開発

## 概要

このガイドでは、ソースからのビルドとローカル実行を含む、ローカル開発用の KECS のセットアップについて説明します。

## 前提条件

- Go 1.21 以降
- Docker Desktop（Kind 統合用）
- Make
- Git

## ソースからのビルド

### リポジトリのクローン

```bash
git clone https://github.com/nandemo-ya/kecs.git
cd kecs
```

### バイナリのビルド

```bash
# コントロールプレーンのビルド
make build

# バイナリは ./bin/kecs に作成されます
```

### Web UI を含むビルド

```bash
# Web UI をビルドしてバイナリに埋め込む
./scripts/build-webui.sh

# または手動で:
cd web-ui
npm install
npm run build
cd ../controlplane
go build -tags webui -o ../bin/kecs ./cmd/controlplane
```

## ローカルでの実行

### 基本セットアップ

```bash
# デフォルト設定で KECS を実行
./bin/kecs server

# KECS は以下で起動します:
# - API サーバー: http://localhost:8080
# - 管理サーバー: http://localhost:8081
# - Web UI: http://localhost:8080/ui
```

### カスタム設定

```bash
# カスタムポートで実行
./bin/kecs server --api-port 9080 --admin-port 9081

# デバッグログで実行
./bin/kecs server --log-level debug

# カスタムデータディレクトリで実行
./bin/kecs server --data-dir ./data
```

### 環境変数

```bash
# ログレベルの設定
export KECS_LOG_LEVEL=debug

# データディレクトリの設定
export KECS_DATA_DIR=/path/to/data

# LocalStack 統合の有効化
export KECS_LOCALSTACK_ENABLED=true
export KECS_LOCALSTACK_ENDPOINT=http://localhost:4566
```

## 開発ワークフロー

### テストの実行

```bash
# すべてのテストを実行
make test

# カバレッジ付きで実行
make test-coverage

# 特定のパッケージテストを実行
go test ./internal/controlplane/api/...
```

### コード品質

```bash
# コードのフォーマット
make fmt

# リンターの実行
make vet

# すべてのチェックを実行
make all
```

### ホットリロード

開発には、ホットリロード用の `air` を使用:

```bash
# air のインストール
go install github.com/cosmtrek/air@latest

# ホットリロードで実行
air -c .air.toml
```

`.air.toml` の例:
```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = ["server"]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./controlplane/cmd/controlplane"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "web-ui"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_error = true

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
```

## Docker Compose の使用

ローカル開発用の `docker-compose.yml` を作成:

```yaml
version: '3.8'

services:
  kecs:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      - KECS_LOG_LEVEL=debug
      - KECS_LOCALSTACK_ENABLED=true
      - KECS_LOCALSTACK_ENDPOINT=http://localstack:4566
    volumes:
      - ./data:/data
      - /var/run/docker.sock:/var/run/docker.sock
    depends_on:
      - localstack

  localstack:
    image: localstack/localstack:latest
    ports:
      - "4566:4566"
    environment:
      - SERVICES=s3,dynamodb,sqs,sns,secretsmanager,ssm,iam,logs,cloudwatch
      - DEBUG=1
    volumes:
      - ./localstack:/var/lib/localstack
      - /var/run/docker.sock:/var/run/docker.sock
```

実行:
```bash
docker-compose up
```

## IDE セットアップ

### VS Code

推奨拡張機能:
- Go
- Prettier
- ESLint
- Docker
- GitLens

`.vscode/launch.json` の例:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch KECS",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/controlplane/cmd/controlplane",
      "args": ["server", "--log-level", "debug"],
      "env": {
        "KECS_DATA_DIR": "${workspaceFolder}/data"
      }
    }
  ]
}
```

### GoLand

1. プロジェクトルートを開く
2. Go モジュールサポートを設定
3. 実行構成を設定:
   - プログラム引数: `server --log-level debug`
   - 環境変数: `KECS_DATA_DIR=/path/to/data`

## 一般的な開発タスク

### 新しい API エンドポイントの追加

1. `internal/controlplane/api/types.go` でタイプを定義
2. 適切なファイル（例: `clusters.go`）でハンドラーを実装
3. `internal/controlplane/api/server.go` でハンドラーを登録
4. `*_test.go` ファイルでテストを追加

### データベーススキーマの変更

1. `internal/storage/duckdb/schema.sql` でスキーマを更新
2. `internal/storage/duckdb/migrations/` でマイグレーションを追加
3. 必要に応じてストレージインターフェイスを更新
4. テストを実行して変更を確認

### Web UI の作業

```bash
# Web UI 開発サーバーの起動
cd web-ui
npm install
npm run dev

# UI は http://localhost:5173 で利用可能
# API プロキシは http://localhost:8080 に転送するよう設定済み
```

## トラブルシューティング

### ポートがすでに使用中

```bash
# ポートを使用しているプロセスを見つける
lsof -i :8080

# プロセスを終了
kill -9 <PID>
```

### Docker ソケットの権限

```bash
# ユーザーを docker グループに追加
sudo usermod -aG docker $USER

# 変更を適用
newgrp docker
```

### ビルドエラー

```bash
# ビルドアーティファクトをクリーン
make clean

# 依存関係を更新
go mod tidy
go mod download

# 再ビルド
make build
```

### データベースの問題

```bash
# データベースファイルを削除
rm -rf ~/.kecs/data/kecs.db

# KECS は次回起動時にデータベースを再作成します
```

## 次のステップ

- [Kind デプロイメント](./kind) - Kind クラスターへのデプロイ
- [テストガイド](/ja/guides/integration-testing) - テストの作成と実行
- [コントリビューション](/ja/development/contributing) - KECS への貢献