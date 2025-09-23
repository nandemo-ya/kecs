# トラブルシューティングガイド

このガイドは、KECS の一般的な問題を診断して解決するのに役立ちます。

## 診断ツール

### ヘルスチェック

KECS コンポーネントの健全性を確認:

```bash
# API サーバーの健全性
curl http://localhost:8081/health

# 詳細な健全性ステータス
curl http://localhost:8081/health/detailed

# Kubernetes 接続性
kubectl cluster-info
```

### ログ

KECS ログの表示:

```bash
# 直接実行している場合
./bin/kecs server 2>&1 | tee kecs.log

# Docker で実行している場合
docker logs kecs-container

# Kubernetes で実行している場合
kubectl logs -n kecs-system deployment/kecs-control-plane
```

### デバッグモード

デバッグログを有効化:

```bash
# コマンドライン経由
./bin/kecs server --log-level debug

# 環境変数経由
export KECS_LOG_LEVEL=debug
./bin/kecs server
```

## 一般的な問題

### インストールの問題

#### 問題: ビルドが失敗する

**症状:**
```
go: cannot find main module
```

**解決策:**
```bash
# 正しいディレクトリにいることを確認
cd /path/to/kecs

# クリーンして再ビルド
make clean
make deps
make build
```

#### 問題: 依存関係が不足

**症状:**
```
package github.com/... is not in GOROOT
```

**解決策:**
```bash
# 依存関係を更新
go mod download
go mod tidy

# Go バージョンを確認
go version  # 1.21+ である必要があります
```

### 起動の問題

#### 問題: ポートがすでに使用中

**症状:**
```
listen tcp :8080: bind: address already in use
```

**解決策:**
```bash
# ポートを使用しているプロセスを見つける
lsof -i :8080

# プロセスを終了
kill -9 <PID>

# または別のポートを使用
./bin/kecs server --api-port 9080
```

#### 問題: Kubernetes に接続できない

**症状:**
```
failed to get kubernetes config: stat /home/user/.kube/config: no such file or directory
```

**解決策:**
```bash
# kubeconfig が存在することを確認
ls ~/.kube/config

# kubeconfig を明示的に設定
./bin/kecs server --kubeconfig /path/to/kubeconfig

# またはクラスター内設定を使用
kubectl apply -f deploy/kubernetes/rbac.yaml
```

### クラスター操作

#### 問題: クラスター作成が失敗する

**症状:**
```
failed to create kind cluster: exit status 1
```

**解決策:**
```bash
# Docker が実行中であることを確認
docker ps

# Kind がインストールされていることを確認
kind version

# 手動でクラスターを作成
kind create cluster --name kecs-cluster

# クラスターを確認
kubectl get nodes
```

#### 問題: クラスターがすでに存在する

**症状:**
```
cluster already exists
```

**解決策:**
```bash
# 既存のクラスターを一覧表示
aws ecs list-clusters --endpoint-url http://localhost:8080

# 削除して再作成
aws ecs delete-cluster --cluster <name> --endpoint-url http://localhost:8080
```

### サービスデプロイメントの問題

#### 問題: サービスが起動しない

**症状:**
- サービスが PENDING のままになる
- 実行中のタスクがない

**解決策:**
1. タスク定義を確認:
   ```bash
   aws ecs describe-task-definition \
     --task-definition <family:revision> \
     --endpoint-url http://localhost:8080
   ```

2. クラスターリソースを確認:
   ```bash
   kubectl top nodes
   kubectl describe nodes
   ```

3. サービスイベントを確認:
   ```bash
   aws ecs describe-services \
     --cluster <cluster> \
     --services <service> \
     --endpoint-url http://localhost:8080
   ```

#### 問題: タスクが停止し続ける

**症状:**
- タスクが STOPPED に遷移する
- サービスが希望数を維持できない

**解決策:**
1. タスク停止理由を確認:
   ```bash
   aws ecs describe-tasks \
     --cluster <cluster> \
     --tasks <task-arn> \
     --endpoint-url http://localhost:8080 \
     | jq '.tasks[0].stoppedReason'
   ```

2. コンテナーログを表示:
   ```bash
   kubectl logs -n <cluster-name> <pod-name>
   ```

3. 一般的な原因:
   - イメージプル失敗
   - ヘルスチェック失敗
   - リソース制約
   - アプリケーションエラー

### タスクの問題

#### 問題: イメージプルエラー

**症状:**
```
CannotPullContainerError: Error response from daemon: pull access denied
```

**解決策:**
1. イメージが存在することを確認:
   ```bash
   docker pull <image-name>
   ```

2. イメージレジストリの認証情報を確認:
   ```bash
   # プライベートレジストリの場合
   kubectl create secret docker-registry regcred \
     --docker-server=<registry> \
     --docker-username=<username> \
     --docker-password=<password> \
     -n <cluster-name>
   ```

3. 認証情報でタスク定義を更新:
   ```json
   {
     "containerDefinitions": [{
       "repositoryCredentials": {
         "credentialsParameter": "arn:aws:secretsmanager:region:account:secret:name"
       }
     }]
   }
   ```

#### 問題: メモリ不足

**症状:**
```
OutOfMemoryError: Container killed due to memory limit
```

**解決策:**
1. メモリ制限を増やす:
   ```json
   {
     "memory": "1024",
     "memoryReservation": "512"
   }
   ```

2. メモリ使用量を確認:
   ```bash
   kubectl top pod -n <cluster-name>
   ```

3. アプリケーションのメモリ使用量を最適化

### ネットワークの問題

#### 問題: サービスディスカバリーが機能しない

**症状:**
- サービスが通信できない
- DNS 解決が失敗する

**解決策:**
1. サービス登録を確認:
   ```bash
   aws servicediscovery list-services \
     --endpoint-url http://localhost:4566
   ```

2. DNS 解決をテスト:
   ```bash
   kubectl exec -n <namespace> <pod> -- nslookup <service-name>
   ```

3. ネットワークポリシーを確認:
   ```bash
   kubectl get networkpolicies -n <namespace>
   ```

#### 問題: ロードバランサーが機能しない

**症状:**
- 外部からサービスにアクセスできない
- ヘルスチェックが失敗する

**解決策:**
1. サービスタイプを確認:
   ```bash
   kubectl get svc -n <namespace>
   ```

2. ターゲットの健全性を確認:
   ```bash
   aws elbv2 describe-target-health \
     --target-group-arn <arn> \
     --endpoint-url http://localhost:4566
   ```

3. セキュリティグループとポートを確認

### LocalStack 統合の問題

#### 問題: LocalStack 接続失敗

**症状:**
```
Could not connect to the endpoint URL: "http://localhost:4566/"
```

**解決策:**
1. LocalStack が実行中であることを確認:
   ```bash
   docker ps | grep localstack
   curl http://localhost:4566/_localstack/health
   ```

2. KECS 設定を確認:
   ```yaml
   localstack:
     enabled: true
     endpoint: http://localhost:4566
   ```

3. 両方のサービスを再起動:
   ```bash
   docker-compose restart
   ```

#### 問題: AWS SDK が LocalStack を使用しない

**症状:**
- リクエストが実際の AWS に送信される
- 認証エラー

**解決策:**
1. サイドカーインジェクションを確認:
   ```bash
   kubectl describe pod <pod> -n <namespace> | grep localstack-proxy
   ```

2. AWS エンドポイントを明示的に設定:
   ```python
   boto3.client('s3', endpoint_url='http://localhost:4566')
   ```

3. 環境変数を確認:
   ```bash
   kubectl exec <pod> -n <namespace> -- env | grep AWS
   ```

### パフォーマンスの問題

#### 問題: API レスポンスが遅い

**解決策:**
1. リソース使用状況を確認:
   ```bash
   # KECS サーバー
   top -p $(pgrep kecs)
   
   # データベース
   ls -la ~/.kecs/data/kecs.db
   ```

2. パフォーマンスメトリクスを有効化:
   ```bash
   curl http://localhost:8081/metrics
   ```

3. データベースを最適化:
   ```bash
   # データベースをバキューム
   sqlite3 ~/.kecs/data/kecs.db "VACUUM;"
   ```

#### 問題: 高メモリ使用率

**解決策:**
1. メモリリークを確認:
   ```bash
   go tool pprof http://localhost:8081/debug/pprof/heap
   ```

2. 同時操作を制限:
   ```yaml
   server:
     maxConcurrentRequests: 100
   ```

3. キャッシュ設定を調整:
   ```yaml
   cache:
     maxSize: 1000
     ttl: 5m
   ```

## 高度なデバッグ

### 詳細ログを有効化

```bash
# すべてのコンポーネント
export KECS_LOG_LEVEL=trace

# 特定のコンポーネント
export KECS_API_LOG_LEVEL=debug
export KECS_STORAGE_LOG_LEVEL=trace
export KECS_K8S_LOG_LEVEL=debug
```

### リクエストをトレース

```bash
# リクエストトレーシングを有効化
curl -H "X-Debug-Trace: true" \
  -X POST http://localhost:8080/v1/ListClusters \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" \
  -d '{}'
```

### データベース検査

```bash
# データベースを開く
sqlite3 ~/.kecs/data/kecs.db

# テーブル一覧
.tables

# クラスターを確認
SELECT * FROM clusters;

# サービスを確認
SELECT * FROM services WHERE cluster_arn = 'arn:...';
```

### Kubernetes デバッグ

```bash
# 名前空間内のすべてのリソースを取得
kubectl get all -n <cluster-name>

# 問題のある Pod を詳細表示
kubectl describe pod <pod-name> -n <cluster-name>

# Pod イベントを取得
kubectl get events -n <cluster-name> --sort-by='.lastTimestamp'

# コンテナーをデバッグ
kubectl debug -it <pod-name> -n <cluster-name> --image=busybox
```

## ヘルプを得る

### 診断情報の収集

診断スクリプトを実行:
```bash
./scripts/collect-diagnostics.sh
```

これにより以下が収集されます:
- KECS ログ
- 設定ファイル
- Kubernetes クラスター状態
- システム情報

### 問題の報告

問題を報告する際は、以下を含めてください:

1. **環境の詳細**
   - KECS バージョン: `kecs version`
   - OS: `uname -a`
   - Kubernetes バージョン: `kubectl version`
   - Docker バージョン: `docker version`

2. **再現手順**
   - 実行した正確なコマンド
   - 使用した設定ファイル
   - 期待される動作と実際の動作

3. **ログとエラー**
   - KECS サーバーログ
   - 関連する Kubernetes イベント
   - エラーメッセージ

4. **診断バンドル**
   - 診断スクリプトの出力

### コミュニティサポート

- GitHub Issues: [github.com/nandemo-ya/kecs/issues](https://github.com/nandemo-ya/kecs/issues)

## 予防のヒント

### 定期的なメンテナンス

1. **定期的な更新**
   ```bash
   git pull origin main
   make build
   ```

2. **リソースの監視**
   - ディスクスペースのアラートを設定
   - メモリ使用量を監視
   - API レスポンスタイムを追跡

3. **データのバックアップ**
   ```bash
   # データベースをバックアップ
   cp ~/.kecs/data/kecs.db ~/.kecs/data/kecs.db.backup
   ```

4. **リソースのクリーンアップ**
   ```bash
   # 停止したタスクを削除
   kubectl delete pods -n <namespace> --field-selector=status.phase=Succeeded
   
   # 未使用のイメージをプルーン
   docker image prune -a
   ```

### ベストプラクティス

1. **リソース制限を使用**
   - 適切な CPU/メモリ制限を設定
   - 実際の使用量を監視
   - スパイクのための余裕を残す

2. **ヘルスチェックを有効化**
   - liveness プローブを設定
   - readiness プローブを設定
   - 健全性メトリクスを監視

3. **障害に備える**
   - 障害シナリオをテスト
   - リカバリー手順を文書化
   - バックアップを最新に保つ

4. **情報を入手する**
   - リリースノートを読む
   - セキュリティアドバイザリーをフォロー
   - コミュニティディスカッションに参加