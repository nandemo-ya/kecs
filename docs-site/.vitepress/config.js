import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'KECS Documentation',
  description: 'Kubernetes-based ECS Compatible Service',
  base: '/kecs/',
  
  head: [
    ['link', { rel: 'icon', href: '/favicon.ico' }]
  ],

  themeConfig: {
    logo: '/logo.svg',
    
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
            { text: 'Installation', link: '/guides/installation' },
            { text: 'Quick Start', link: '/guides/quick-start' }
          ]
        },
        {
          text: 'Core Concepts',
          items: [
            { text: 'Clusters', link: '/guides/clusters' },
            { text: 'Services', link: '/guides/services' },
            { text: 'Tasks', link: '/guides/tasks' },
            { text: 'Task Definitions', link: '/guides/task-definitions' }
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
            { text: 'Task Definition APIs', link: '/api/task-definitions' }
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
            { text: 'Testing', link: '/development/testing' },
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
            { text: 'Web UI', link: '/architecture/web-ui' }
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
      copyright: 'Copyright © 2024 KECS Project'
    }
  }
})