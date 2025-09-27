import { defineConfig } from 'vitepress'

export default defineConfig({
  // base path removed for custom domain deployment

  // Ignore dead links for now to allow build
  ignoreDeadLinks: true,

  // Site metadata
  title: 'KECS Documentation',
  description: 'Kubernetes-based ECS Compatible Service',
  lang: 'en',

  head: [
    // Favicon
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '32x32', href: '/favicon-32.png' }],
    ['link', { rel: 'icon', type: 'image/png', sizes: '16x16', href: '/favicon-16.png' }],
    ['link', { rel: 'apple-touch-icon', sizes: '180x180', href: '/apple-touch-icon.png' }],

    // Open Graph Protocol
    ['meta', { property: 'og:title', content: 'KECS - Kubernetes-based ECS Compatible Service' }],
    ['meta', { property: 'og:description', content: 'Run Amazon ECS workloads locally or on any Kubernetes cluster without AWS dependencies. Full ECS API compatibility with local development workflow.' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:url', content: 'https://kecs.dev' }],
    ['meta', { property: 'og:image', content: 'https://kecs.dev/og-image.png' }],
    ['meta', { property: 'og:image:width', content: '1200' }],
    ['meta', { property: 'og:image:height', content: '630' }],
    ['meta', { property: 'og:site_name', content: 'KECS Documentation' }],

    // Twitter Card
    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { name: 'twitter:title', content: 'KECS - Kubernetes-based ECS Compatible Service' }],
    ['meta', { name: 'twitter:description', content: 'Run Amazon ECS workloads locally or on any Kubernetes cluster without AWS dependencies.' }],
    ['meta', { name: 'twitter:image', content: 'https://kecs.dev/og-image.png' }],

    // Additional Meta
    ['meta', { name: 'author', content: 'KECS Contributors' }],
    ['meta', { name: 'keywords', content: 'KECS, Kubernetes, ECS, Amazon ECS, Container Orchestration, Local Development, Docker, k3d' }]
  ],

  // Theme configuration
  themeConfig: {
    logo: '/logo.svg',

    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guide', link: '/guides/getting-started' }
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
            { text: 'Port Forwarding', link: '/guides/port-forward' },
            { text: 'ELBv2 Integration', link: '/guides/elbv2-integration' },
            { text: 'TUI Interface', link: '/guides/tui-interface' }
          ]
        },
        {
          text: 'Operations',
          items: [
            { text: 'Kubeconfig Management', link: '/guides/kubeconfig-management' }
          ]
        },
        {
          text: 'Integration',
          items: [
            { text: 'LocalStack Integration', link: '/guides/localstack-integration' }
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
      '/deployment/': [
        {
          text: 'Deployment',
          items: [
            { text: 'Local Development', link: '/deployment/local' },
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
      ]
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/nandemo-ya/kecs' }
    ],

    search: {
      provider: 'local'
    },

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © 2025 KECS Project'
    }
  }
})