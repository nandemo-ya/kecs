---
layout: home

hero:
  name: "KECS"
  text: "Kubernetes ベースの ECS 互換サービス"
  tagline: "Amazon ECS ワークロードを Kubernetes 上でローカル実行"
  actions:
    - theme: brand
      text: はじめる
      link: /ja/guides/getting-started
    - theme: alt
      text: GitHub で見る
      link: https://github.com/nandemo-ya/kecs

features:
  - icon: 🚀
    title: ECS 互換
    details: Amazon ECS API との完全な互換性により、シームレスなローカル開発を実現
  - icon: ☸️
    title: Kubernetes ネイティブ
    details: 信頼性とスケーラビリティのために Kubernetes 上に構築
  - icon: 🛠️
    title: 開発者フレンドリー
    details: Kind による簡単なセットアップ、WebSocket によるリアルタイム更新
  - icon: 📦
    title: 本番環境対応
    details: PostgreSQL による永続化、グレースフルシャットダウン、包括的なモニタリング
---

## クイックスタート

KECS を数分で始めましょう：

```bash
# リポジトリをクローン
git clone https://github.com/nandemo-ya/kecs.git
cd kecs

# ビルドと実行
make build
./bin/kecs server
```

## なぜ KECS？

KECS は AWS から独立して動作する、完全にローカルな ECS 互換環境を提供します。以下の用途に最適です：

- **ローカル開発**: AWS のコストなしで ECS ワークロードをテスト
- **CI/CD パイプライン**: 分離された環境で統合テストを実行
- **学習**: AWS アカウントなしで ECS の概念を理解
- **オフライン開発**: インターネットなしで ECS アプリケーションを開発

## アーキテクチャ概要

KECS は Kubernetes 上に ECS API 仕様を実装しています：

- **コントロールプレーン**: ECS API リクエストの処理と状態管理
- **ストレージレイヤー**: PostgreSQL による永続的なストレージ
- **Kubernetes バックエンド**: ECS の概念を Kubernetes リソースに変換

## コミュニティ

コミュニティに参加して貢献しましょう：

- [GitHub Issues](https://github.com/nandemo-ya/kecs/issues)
- [ディスカッション](https://github.com/nandemo-ya/kecs/discussions)
- [コントリビューションガイド](/ja/development/contributing)