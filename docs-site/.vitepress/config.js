import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/kecs/',
  
  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '32x32', href: '/favicon-32.png' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '16x16', href: '/favicon-16.png' }],
    ['link', { rel: 'apple-touch-icon', sizes: '180x180', href: '/apple-touch-icon.png' }]
  ],

  // Locales configuration
  locales: {
    root: {
      label: 'English',
      lang: 'en',
      title: 'KECS Documentation',
      description: 'Kubernetes-based ECS Compatible Service',
      themeConfig: {
        nav: [
          { text: 'Home', link: '/' },
          { text: 'Guide', link: '/guides/getting-started' },
          { text: 'API Reference', link: '/api/' },
          { text: 'Architecture', link: '/architecture/' }
        ],
        sidebar: {
          '/guides/': [
            {
              text: 'Getting Started',
              items: [
                { text: 'Introduction', link: '/guides/getting-started' },
                { text: 'CLI Commands', link: '/guides/cli-commands' },
                { text: 'Quick Start', link: '/guides/quick-start' }
              ]
            },
            {
              text: 'Core Features',
              items: [
                { text: 'Services', link: '/guides/services' },
                { text: 'Task Definitions', link: '/guides/task-definitions' },
                { text: 'ELBv2 Integration', link: '/guides/elbv2-integration' },
                { text: 'TUI Interface', link: '/guides/tui-interface' }
              ]
            },
            {
              text: 'Operations',
              items: [
                { text: 'Kubeconfig Management', link: '/guides/kubeconfig-management' },
                { text: 'Hot Reload Development', link: '/guides/hot-reload' }
              ]
            },
            {
              text: 'Integration',
              items: [
                { text: 'LocalStack Integration', link: '/guides/localstack-integration' },
                { text: 'Networking', link: '/guides/networking' }
              ]
            },
            {
              text: 'Testing',
              items: [
                { text: 'Integration Testing', link: '/guides/integration-testing' },
                { text: 'Using Testcontainers', link: '/guides/testcontainers' }
              ]
            },
            {
              text: 'Reference',
              items: [
                { text: 'Troubleshooting', link: '/guides/troubleshooting' },
                { text: 'Examples', link: '/guides/examples' }
              ]
            }
          ],
          '/api/': [
            {
              text: 'API Reference',
              items: [
                { text: 'Overview', link: '/api/' },
                { text: 'Authentication', link: '/api/authentication' },
                { text: 'Cluster APIs', link: '/api/clusters' },
                { text: 'Service APIs', link: '/api/services' },
                { text: 'Task APIs', link: '/api/tasks' },
                { text: 'Task Definition APIs', link: '/api/task-definitions' },
                { text: 'WebSocket API', link: '/api/websocket' }
              ]
            }
          ],
          '/deployment/': [
            {
              text: 'Deployment',
              items: [
                { text: 'Local Development', link: '/deployment/local' },
                { text: 'Kind Deployment', link: '/deployment/kind' },
                { text: 'Production Deployment', link: '/deployment/production' },
                { text: 'Configuration', link: '/deployment/configuration' }
              ]
            }
          ],
          '/development/': [
            {
              text: 'Development',
              items: [
                { text: 'Contributing', link: '/development/contributing' },
                { text: 'Architecture', link: '/development/architecture' },
                { text: 'Code Generation', link: '/development/code-generation' },
                { text: 'Testing', link: '/development/testing' },
                { text: 'Testcontainers Integration', link: '/development/testcontainers' },
                { text: 'Building', link: '/development/building' }
              ]
            }
          ],
          '/architecture/': [
            {
              text: 'Architecture',
              items: [
                { text: 'Overview', link: '/architecture/' },
                { text: 'Control Plane', link: '/architecture/control-plane' },
                { text: 'Storage Layer', link: '/architecture/storage' },
                { text: 'Kubernetes Integration', link: '/architecture/kubernetes' },
              ]
            }
          ]
        }
      }
    },
    ja: {
      label: '日本語',
      lang: 'ja',
      title: 'KECS ドキュメント',
      description: 'Kubernetes ベースの ECS 互換サービス',
      themeConfig: {
        nav: [
          { text: 'ホーム', link: '/ja/' },
          { text: 'ガイド', link: '/ja/guides/getting-started' },
          { text: 'API リファレンス', link: '/ja/api/' },
          { text: 'アーキテクチャ', link: '/ja/architecture/' }
        ],
        sidebar: {
          '/ja/guides/': [
            {
              text: 'はじめに',
              items: [
                { text: 'イントロダクション', link: '/ja/guides/getting-started' },
                { text: 'クイックスタート', link: '/ja/guides/quick-start' }
              ]
            },
            {
              text: '主要概念',
              items: [
                { text: 'サービス', link: '/ja/guides/services' },
                { text: 'タスク定義', link: '/ja/guides/task-definitions' },
              ]
            },
            {
              text: '運用',
              items: [
                { text: 'Kubeconfig 管理', link: '/ja/guides/kubeconfig-management' }
              ]
            },
            {
              text: '統合',
              items: [
                { text: 'LocalStack 統合', link: '/ja/guides/localstack-integration' }
              ]
            },
            {
              text: 'テスト',
              items: [
                { text: '統合テスト', link: '/ja/guides/integration-testing' },
                { text: 'Testcontainers の使用', link: '/ja/guides/testcontainers' }
              ]
            },
            {
              text: 'リファレンス',
              items: [
                { text: 'トラブルシューティング', link: '/ja/guides/troubleshooting' }
              ]
            }
          ],
          '/ja/api/': [
            {
              text: 'API リファレンス',
              items: [
                { text: '概要', link: '/ja/api/' },
                { text: '認証', link: '/ja/api/authentication' },
                { text: 'クラスター API', link: '/ja/api/clusters' },
                { text: 'サービス API', link: '/ja/api/services' },
                { text: 'タスク API', link: '/ja/api/tasks' },
                { text: 'タスク定義 API', link: '/ja/api/task-definitions' },
                { text: 'WebSocket API', link: '/ja/api/websocket' }
              ]
            }
          ],
          '/ja/deployment/': [
            {
              text: 'デプロイメント',
              items: [
                { text: 'ローカル開発', link: '/ja/deployment/local' },
                { text: 'Kind デプロイメント', link: '/ja/deployment/kind' },
                { text: '本番環境デプロイメント', link: '/ja/deployment/production' },
                { text: '設定', link: '/ja/deployment/configuration' }
              ]
            }
          ],
          '/ja/development/': [
            {
              text: '開発',
              items: [
                { text: 'コントリビューション', link: '/ja/development/contributing' },
                { text: 'アーキテクチャ', link: '/ja/development/architecture' },
                { text: 'コード生成', link: '/ja/development/code-generation' },
                { text: 'テスト', link: '/ja/development/testing' },
                { text: 'Testcontainers 統合', link: '/ja/development/testcontainers' },
                { text: 'ビルド', link: '/ja/development/building' }
              ]
            }
          ],
          '/ja/architecture/': [
            {
              text: 'アーキテクチャ',
              items: [
                { text: '概要', link: '/ja/architecture/' },
                { text: 'コントロールプレーン', link: '/ja/architecture/control-plane' },
                { text: 'ストレージレイヤー', link: '/ja/architecture/storage' },
                { text: 'Kubernetes 統合', link: '/ja/architecture/kubernetes' },
              ]
            }
          ]
        }
      }
    }
  },

  // Common theme configuration
  themeConfig: {
    logo: '/logo.svg',
    
    socialLinks: [
      { icon: 'github', link: 'https://github.com/nandemo-ya/kecs' }
    ],

    search: {
      provider: 'local',
      options: {
        locales: {
          ja: {
            placeholder: '検索',
            translations: {
              button: {
                buttonText: '検索',
                buttonAriaLabel: '検索'
              },
              modal: {
                searchBox: {
                  resetButtonTitle: 'クリア',
                  resetButtonAriaLabel: 'クリア',
                  cancelButtonText: 'キャンセル',
                  cancelButtonAriaLabel: 'キャンセル'
                },
                startScreen: {
                  recentSearchesTitle: '検索履歴',
                  noRecentSearchesText: '検索履歴はありません',
                  saveRecentSearchButtonTitle: '検索履歴に保存',
                  removeRecentSearchButtonTitle: '検索履歴から削除',
                  favoriteSearchesTitle: 'お気に入り',
                  removeFavoriteSearchButtonTitle: 'お気に入りから削除'
                },
                errorScreen: {
                  titleText: '結果を取得できませんでした',
                  helpText: '接続を確認してください'
                },
                footer: {
                  selectText: '選択',
                  selectKeyAriaLabel: 'Enter',
                  navigateText: '移動',
                  navigateUpKeyAriaLabel: '上矢印',
                  navigateDownKeyAriaLabel: '下矢印',
                  closeText: '閉じる',
                  closeKeyAriaLabel: 'Escape'
                },
                noResultsScreen: {
                  noResultsText: '結果が見つかりませんでした',
                  suggestedQueryText: '次を検索してみてください',
                  reportMissingResultsText: '結果が表示されるべきですか？',
                  reportMissingResultsLinkText: '報告する'
                }
              }
            }
          }
        }
      }
    },

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2024 KECS Project'
    }
  }
})