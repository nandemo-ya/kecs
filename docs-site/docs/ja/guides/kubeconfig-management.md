# Kubeconfig 管理

このガイドでは、KECSが作成したk3dクラスターにアクセスするためのkubeconfigコマンドの使い方を説明します。

## 概要

KECSがk3dクラスターを作成する際、生成されるkubeconfigファイルは、ローカルマシンから使用するために手動での修正が必要な場合があります。`kecs kubeconfig`コマンドは、これらの修正を自動的に処理します。

## k3d Kubeconfigの一般的な問題

コンテナ外部からアクセスする際、k3dが生成するkubeconfigファイルには以下の問題がある場合があります：

1. **host.docker.internal**: Linuxホストではこのホスト名が解決できません
2. **不正なポート番号**: APIサーバーのポートが正しく公開されていない場合があります
3. **空のポート値**: ポート番号が欠落し、末尾にコロンだけが残る場合があります

## kubeconfigコマンドの使用方法

### 利用可能なクラスターの一覧表示

対応するk3dクラスターを持つすべてのKECSクラスターを表示するには：

```bash
kecs kubeconfig list
```

出力例：
```
Available KECS clusters:
  test-cluster
  microservices-cluster
```

### クラスターのKubeconfig取得

適切に設定されたkubeconfigを取得するには：

```bash
# 標準出力に出力
kecs kubeconfig get test-cluster

# ファイルに保存
kecs kubeconfig get test-cluster -o ~/.kube/kecs-test-cluster

# kubectlで直接使用
kecs kubeconfig get test-cluster | kubectl --kubeconfig=/dev/stdin get nodes
```

### 元のk3d Kubeconfigを取得

修正を適用せずに元のk3d kubeconfigが必要な場合：

```bash
kecs kubeconfig get test-cluster --raw
```

## コマンドが修正する内容

`kecs kubeconfig get`コマンドは以下を自動的に行います：

1. `host.docker.internal`を`127.0.0.1`に置換
2. k3dロードバランサーコンテナから正しいAPIサーバーポートを抽出
3. 不正な形式のサーバーURLを修正（例：`https://host.docker.internal:` → `https://127.0.0.1:50715`）

## kubectlとの統合

### 環境変数の使用

```bash
# KUBECONFIG環境変数を設定
export KUBECONFIG=$(kecs kubeconfig get test-cluster -o /tmp/kecs-test.kubeconfig && echo /tmp/kecs-test.kubeconfig)

# これでkubectlはこの設定をデフォルトで使用します
kubectl get nodes
```

### kubectlコンテキストの使用

```bash
# kubeconfigを保存して既存の設定とマージ
kecs kubeconfig get test-cluster -o ~/.kube/kecs-test-cluster
export KUBECONFIG=~/.kube/config:~/.kube/kecs-test-cluster
kubectl config view --flatten > ~/.kube/config.new
mv ~/.kube/config.new ~/.kube/config

# コンテキストを使用
kubectl config use-context k3d-kecs-test-cluster
```

### クイックアクセス用のワンライナー

```bash
# クイックアクセス用のエイリアスを作成
alias kube-test='kubectl --kubeconfig=<(kecs kubeconfig get test-cluster)'

# 使用例
kube-test get pods -A
```

## トラブルシューティング

### クラスターが見つからない

「k3d cluster 'kecs-test-cluster' does not exist」のようなエラーが表示された場合：

1. KECSにクラスターが存在するか確認：
   ```bash
   curl -s http://localhost:8080/v1/DescribeClusters | jq
   ```

2. k3dクラスターが存在するか確認：
   ```bash
   k3d cluster list
   ```

3. k3dクラスターが欠落している場合、KECSを再起動してクラスターを再作成：
   ```bash
   # KECSを再起動してクラスターの再作成をトリガー
   kecs stop && kecs start
   ```

### 接続が拒否される

kubectlコマンドが「connection refused」で失敗する場合：

1. k3dクラスターが実行中であることを確認：
   ```bash
   docker ps | grep k3d-kecs
   ```

2. APIサーバーポートが公開されているか確認：
   ```bash
   docker ps --format "table {{.Names}}\t{{.Ports}}" | grep serverlb
   ```

### ポート抽出の失敗

コマンドがAPIサーバーポートの抽出に失敗した場合：

1. ロードバランサーコンテナ名を確認：
   ```bash
   docker ps --format "{{.Names}}" | grep serverlb
   ```

2. 手動でポートを取得：
   ```bash
   docker ps --format "{{.Ports}}" --filter "name=k3d-kecs-.*-serverlb"
   ```

## ベストプラクティス

1. **kubeconfigファイルを保存**: 毎回コマンドを実行する代わりに、kubeconfigをファイルに保存します
2. **特定のコンテキストを使用**: 複数のクラスターで作業する場合は、kubectlコンテキストを使用して切り替えます
3. **スクリプトで自動化**: 頻繁にアクセスするクラスター用のシェルスクリプトやエイリアスを作成します

## 関連コマンド

- `kecs cluster create`: 新しいECSクラスターを作成
- `k3d cluster list`: すべてのk3dクラスターを一覧表示
- `kubectl config`: kubectl設定を管理