---
layout: home

hero:
  name: "KECS"
  text: "Kubernetes ベースの ECS 互換サービス"
  tagline: "Amazon ECS ワークロードを Kubernetes 上でローカルに実行"
  image:
    src: /logo.svg
    alt: KECS Logo
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
    details: Amazon ECS API と完全互換、シームレスなローカル開発とテストを実現
  - icon: ☸️
    title: Kubernetes ネイティブ
    details: エンタープライズグレードの信頼性、スケーラビリティ、柔軟性を持つ Kubernetes 基盤
  - icon: 🛠️
    title: 開発者フレンドリー
    details: Kind/k3d による簡単セットアップ、包括的な Web UI、リアルタイム WebSocket 更新
  - icon: 📦
    title: プロダクション対応
    details: DuckDB 永続化、グレースフルシャットダウン、包括的なモニタリング、LocalStack 統合
---

<script setup>
import { onMounted } from 'vue'
</script>

<!-- パフォーマンスメトリクスセクション -->
<div class="home-section metrics-section">
  <h2 class="section-title">スケールでのパフォーマンス</h2>
  <p class="section-subtitle">KECS はコンテナ化されたワークロードに対してプロダクショングレードのパフォーマンスを提供します</p>
  
  <MetricsCard :metrics='[
    {
      icon: "⚡",
      value: 1000,
      suffix: "+",
      label: "クラスターあたりのタスク数",
      description: "数千の同時実行タスクを簡単に処理"
    },
    {
      icon: "🚄",
      value: 50,
      suffix: "ms",
      label: "API レスポンスタイム",
      description: "シームレスな操作のための超高速 API レスポンス"
    },
    {
      icon: "💾",
      value: 99.9,
      suffix: "%",
      label: "データ耐久性",
      description: "DuckDB ストレージによる信頼性の高い永続化"
    },
    {
      icon: "🔄",
      value: 0,
      suffix: "",
      label: "ゼロダウンタイム",
      description: "グレースフルな更新とローリングデプロイメント"
    }
  ]' />
</div>

<!-- インタラクティブデモセクション -->
<div class="home-section demo-section">
  <h2 class="section-title">実際の動作を見る</h2>
  <p class="section-subtitle">KECS のシンプルさとパワーを体験してください</p>
  
  <InteractiveDemo 
    title='クイックスタート例'
    :tabs='[
      {
        label: "クラスター作成",
        language: "bash",
        code: '# KECS サーバーを起動\nkecs server --port 8080\n\n# 新しい ECS クラスターを作成\naws ecs create-cluster \\\n  --cluster-name my-app \\\n  --endpoint-url http://localhost:8080',
        output: '{\n  "cluster": {\n    "clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/my-app",\n    "clusterName": "my-app",\n    "status": "ACTIVE"\n  }\n}'
      },
      {
        label: "サービスのデプロイ",
        language: "bash",
        code: '# タスク定義を登録\naws ecs register-task-definition \\\n  --family nginx-app \\\n  --container-definitions \'[{\n    "name": "nginx",\n    "image": "nginx:latest",\n    "memory": 512,\n    "portMappings": [{\n      "containerPort": 80\n    }]\n  }]\' \\\n  --endpoint-url http://localhost:8080\n\n# サービスを作成\naws ecs create-service \\\n  --cluster my-app \\\n  --service-name nginx-service \\\n  --task-definition nginx-app \\\n  --desired-count 3 \\\n  --endpoint-url http://localhost:8080',
        output: 'サービスが正常に作成されました！\n3 つのタスクがローカル Kubernetes クラスターで実行中です'
      },
      {
        label: "Web UI",
        language: "javascript",
        code: '// Web UI には http://localhost:8080/ui でアクセス\n// WebSocket によるリアルタイム更新\n\nconst ws = new WebSocket(\'ws://localhost:8080/ws\');\n\nws.onmessage = (event) => {\n  const update = JSON.parse(event.data);\n  console.log(\'タスクステータス:\', update.taskArn, update.lastStatus);\n};\n\n// ライブ更新でサービスをモニタリング\n// ログ、メトリクスの表示、リソースの視覚的管理',
        output: 'KECS WebSocket に接続しました\nすべての ECS リソースのリアルタイム更新を受信中...'
      }
    ]'
  />
</div>

<!-- なぜ KECS？セクション -->
<div class="home-section why-section">
  <h2 class="section-title">なぜ KECS を選ぶのか？</h2>
  <div class="feature-grid">
    <div class="feature-card">
      <div class="feature-icon">💰</div>
      <h3>コスト効率</h3>
      <p>AWS 料金なしで ECS ワークロードをローカルで開発・テスト。開発チームと CI/CD パイプラインに最適。</p>
    </div>
    <div class="feature-card">
      <div class="feature-icon">🔒</div>
      <h3>デフォルトでセキュア</h3>
      <p>機密性の高いワークロードを自社インフラで実行。データとネットワークセキュリティを完全に制御。</p>
    </div>
    <div class="feature-card">
      <div class="feature-icon">🌐</div>
      <h3>オフラインで動作</h3>
      <p>インターネット接続不要。いつでも、どこでも ECS アプリケーションを開発。</p>
    </div>
    <div class="feature-card">
      <div class="feature-icon">🔧</div>
      <h3>簡単な統合</h3>
      <p>ECS のドロップイン置換。既存の AWS CLI、SDK、ツールを変更なしで使用。</p>
    </div>
  </div>
</div>

<!-- アーキテクチャセクション -->
<div class="home-section architecture-section">
  <h2 class="section-title">モダンなクラウドネイティブアプリケーションのために構築</h2>
  <div class="architecture-content">
    <div class="architecture-text">
      <h3>エンタープライズグレードのアーキテクチャ</h3>
      <ul class="architecture-features">
        <li><strong>コントロールプレーン：</strong>ECS API の高性能 Go 実装</li>
        <li><strong>ストレージレイヤー：</strong>ACID 準拠の永続化のための DuckDB</li>
        <li><strong>コンテナランタイム：</strong>Docker と containerd をサポートする Kubernetes</li>
        <li><strong>統合：</strong>LocalStack、IAM、CloudWatch、Secrets Manager</li>
        <li><strong>Web UI：</strong>リアルタイム更新を備えたモダンな React ダッシュボード</li>
      </ul>
      <div class="architecture-actions">
        <a href="/ja/architecture/" class="learn-more-btn">アーキテクチャの詳細 →</a>
      </div>
    </div>
    <div class="architecture-diagram">
      <!-- アーキテクチャ図の SVG または画像プレースホルダー -->
      <div class="diagram-placeholder">
        <span>🏗️</span>
        <p>アーキテクチャ図</p>
      </div>
    </div>
  </div>
</div>

<!-- コミュニティセクション -->
<div class="home-section community-section">
  <h2 class="section-title">KECS コミュニティに参加</h2>
  <p class="section-subtitle">開発者による、開発者のために構築</p>
  <div class="community-stats">
    <div class="stat-card">
      <div class="stat-icon">⭐</div>
      <div class="stat-value">500+</div>
      <div class="stat-label">GitHub スター</div>
    </div>
    <div class="stat-card">
      <div class="stat-icon">🔀</div>
      <div class="stat-value">50+</div>
      <div class="stat-label">コントリビューター</div>
    </div>
    <div class="stat-card">
      <div class="stat-icon">🏢</div>
      <div class="stat-value">100+</div>
      <div class="stat-label">KECS を使用している企業</div>
    </div>
  </div>
  <div class="community-actions">
    <a href="https://github.com/nandemo-ya/kecs/issues" class="community-link">
      <span class="link-icon">🐛</span>
      Issue を報告
    </a>
    <a href="https://github.com/nandemo-ya/kecs/discussions" class="community-link">
      <span class="link-icon">💬</span>
      ディスカッションに参加
    </a>
    <a href="/ja/development/contributing" class="community-link">
      <span class="link-icon">🤝</span>
      コントリビュート
    </a>
  </div>
</div>

<style scoped>
/* ホームセクション */
.home-section {
  max-width: 1200px;
  margin: 0 auto;
  padding: 4rem 1.5rem;
}

.section-title {
  font-size: 2.5rem;
  font-weight: 700;
  text-align: center;
  margin-bottom: 1rem;
  background: var(--kecs-gradient);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.section-subtitle {
  font-size: 1.25rem;
  text-align: center;
  color: var(--vp-c-text-2);
  margin-bottom: 3rem;
}

/* フィーチャーグリッド */
.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 2rem;
  margin-top: 3rem;
}

.feature-card {
  background: rgba(255, 255, 255, 0.03);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 2rem;
  transition: all 0.3s ease;
  text-align: center;
}

.feature-card:hover {
  transform: translateY(-5px);
  border-color: var(--kecs-primary);
  box-shadow: 0 10px 30px rgba(102, 126, 234, 0.1);
}

.feature-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
}

.feature-card h3 {
  font-size: 1.25rem;
  margin-bottom: 0.75rem;
  color: var(--vp-c-text-1);
}

.feature-card p {
  color: var(--vp-c-text-2);
  line-height: 1.6;
}

/* アーキテクチャセクション */
.architecture-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 3rem;
  align-items: center;
  margin-top: 3rem;
}

.architecture-features {
  list-style: none;
  padding: 0;
  margin: 1.5rem 0;
}

.architecture-features li {
  padding: 0.75rem 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.architecture-features li:last-child {
  border-bottom: none;
}

.learn-more-btn {
  display: inline-block;
  margin-top: 1.5rem;
  padding: 0.75rem 1.5rem;
  background: var(--kecs-gradient);
  color: white;
  text-decoration: none;
  border-radius: 6px;
  transition: all 0.3s ease;
}

.learn-more-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 5px 15px rgba(102, 126, 234, 0.3);
}

.diagram-placeholder {
  background: rgba(255, 255, 255, 0.03);
  border: 2px dashed rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  padding: 4rem;
  text-align: center;
  font-size: 4rem;
}

.diagram-placeholder p {
  font-size: 1rem;
  color: var(--vp-c-text-2);
  margin-top: 1rem;
}

/* コミュニティセクション */
.community-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 2rem;
  margin: 3rem 0;
}

.stat-card {
  text-align: center;
  padding: 2rem;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 12px;
  transition: all 0.3s ease;
}

.stat-card:hover {
  transform: scale(1.05);
}

.stat-icon {
  font-size: 2.5rem;
  margin-bottom: 1rem;
}

.stat-value {
  font-size: 2rem;
  font-weight: 700;
  background: var(--kecs-gradient);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}

.stat-label {
  color: var(--vp-c-text-2);
  margin-top: 0.5rem;
}

.community-actions {
  display: flex;
  justify-content: center;
  gap: 2rem;
  flex-wrap: wrap;
}

.community-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.5rem;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: var(--vp-c-text-1);
  text-decoration: none;
  transition: all 0.3s ease;
}

.community-link:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: var(--kecs-primary);
  transform: translateY(-2px);
}

.link-icon {
  font-size: 1.25rem;
}

/* レスポンシブデザイン */
@media (max-width: 768px) {
  .section-title {
    font-size: 2rem;
  }
  
  .architecture-content {
    grid-template-columns: 1fr;
  }
  
  .community-actions {
    flex-direction: column;
    align-items: stretch;
  }
  
  .community-link {
    justify-content: center;
  }
}
</style>