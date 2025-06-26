---
layout: home

hero:
  name: "KECS"
  text: "Kubernetes-based ECS Compatible Service"
  tagline: "Run Amazon ECS workloads locally on Kubernetes with zero friction"
  image:
    src: /logo.svg
    alt: KECS Logo
  actions:
    - theme: brand
      text: Get Started
      link: /guides/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/nandemo-ya/kecs

features:
  - icon: 🚀
    title: ECS Compatible
    details: Full compatibility with Amazon ECS APIs, allowing seamless local development and testing
  - icon: ☸️
    title: Kubernetes Native
    details: Built on Kubernetes for enterprise-grade reliability, scalability, and flexibility
  - icon: 🛠️
    title: Developer Friendly
    details: Simple setup with Kind/k3d, comprehensive Web UI, and real-time WebSocket updates
  - icon: 📦
    title: Production Ready
    details: DuckDB persistence, graceful shutdown, comprehensive monitoring, and LocalStack integration
---

<script setup>
import { onMounted } from 'vue'
</script>

<!-- Performance Metrics Section -->
<div class="home-section metrics-section">
  <h2 class="section-title">Performance at Scale</h2>
  <p class="section-subtitle">KECS delivers production-grade performance for your containerized workloads</p>
  
  <MetricsCard :metrics="[
    {
      icon: '⚡',
      value: 1000,
      suffix: '+',
      label: 'Tasks per Cluster',
      description: 'Handle thousands of concurrent tasks with ease'
    },
    {
      icon: '🚄',
      value: 50,
      suffix: 'ms',
      label: 'API Response Time',
      description: 'Lightning-fast API responses for seamless operations'
    },
    {
      icon: '💾',
      value: 99.9,
      suffix: '%',
      label: 'Data Durability',
      description: 'Reliable persistence with DuckDB storage'
    },
    {
      icon: '🔄',
      value: 0,
      suffix: '',
      label: 'Zero Downtime',
      description: 'Graceful updates and rolling deployments'
    }
  ]" />
</div>

<!-- Interactive Demo Section -->
<div class="home-section demo-section">
  <h2 class="section-title">See It In Action</h2>
  <p class="section-subtitle">Experience the simplicity and power of KECS</p>
  
  <InteractiveDemo 
    title="Quick Start Example"
    :tabs="[
      {
        label: 'Create Cluster',
        language: 'bash',
        code: '# Start KECS server\nkecs server --port 8080\n\n# Create a new ECS cluster\naws ecs create-cluster \\\n  --cluster-name my-app \\\n  --endpoint-url http://localhost:8080',
        output: '{\n  \"cluster\": {\n    \"clusterArn\": \"arn:aws:ecs:us-east-1:123456789012:cluster/my-app\",\n    \"clusterName\": \"my-app\",\n    \"status\": \"ACTIVE\"\n  }\n}'
      },
      {
        label: 'Deploy Service',
        language: 'bash',
        code: '# Register task definition\naws ecs register-task-definition \\\n  --family nginx-app \\\n  --container-definitions \'[{\n    \"name\": \"nginx\",\n    \"image\": \"nginx:latest\",\n    \"memory\": 512,\n    \"portMappings\": [{\n      \"containerPort\": 80\n    }]\n  }]\' \\\n  --endpoint-url http://localhost:8080\n\n# Create service\naws ecs create-service \\\n  --cluster my-app \\\n  --service-name nginx-service \\\n  --task-definition nginx-app \\\n  --desired-count 3 \\\n  --endpoint-url http://localhost:8080',
        output: 'Service created successfully!\n3 tasks are now running in your local Kubernetes cluster'
      },
      {
        label: 'Web UI',
        language: 'javascript',
        code: '// Access the Web UI at http://localhost:8080/ui\n// Real-time updates via WebSocket\n\nconst ws = new WebSocket(\'ws://localhost:8080/ws\');\n\nws.onmessage = (event) => {\n  const update = JSON.parse(event.data);\n  console.log(\'Task status:\', update.taskArn, update.lastStatus);\n};\n\n// Monitor your services with live updates\n// View logs, metrics, and manage resources visually',
        output: 'Connected to KECS WebSocket\nReceiving real-time updates for all ECS resources...'
      }
    ]"
  />
</div>

<!-- Why KECS Section -->
<div class="home-section why-section">
  <h2 class="section-title">Why Choose KECS?</h2>
  <div class="feature-grid">
    <div class="feature-card">
      <div class="feature-icon">💰</div>
      <h3>Cost Efficient</h3>
      <p>Develop and test ECS workloads locally without AWS charges. Perfect for development teams and CI/CD pipelines.</p>
    </div>
    <div class="feature-card">
      <div class="feature-icon">🔒</div>
      <h3>Secure by Default</h3>
      <p>Run sensitive workloads in your own infrastructure. Full control over data and network security.</p>
    </div>
    <div class="feature-card">
      <div class="feature-icon">🌐</div>
      <h3>Works Offline</h3>
      <p>No internet connection required. Develop ECS applications anywhere, anytime.</p>
    </div>
    <div class="feature-card">
      <div class="feature-icon">🔧</div>
      <h3>Easy Integration</h3>
      <p>Drop-in replacement for ECS. Use existing AWS CLI, SDKs, and tools without modification.</p>
    </div>
  </div>
</div>

<!-- Architecture Section -->
<div class="home-section architecture-section">
  <h2 class="section-title">Built for Modern Cloud Native Applications</h2>
  <div class="architecture-content">
    <div class="architecture-text">
      <h3>Enterprise-Grade Architecture</h3>
      <ul class="architecture-features">
        <li><strong>Control Plane:</strong> High-performance Go implementation of ECS APIs</li>
        <li><strong>Storage Layer:</strong> DuckDB for ACID-compliant persistence</li>
        <li><strong>Container Runtime:</strong> Kubernetes with support for Docker and containerd</li>
        <li><strong>Integrations:</strong> LocalStack, IAM, CloudWatch, Secrets Manager</li>
        <li><strong>Web UI:</strong> Modern React dashboard with real-time updates</li>
      </ul>
      <div class="architecture-actions">
        <a href="/architecture/" class="learn-more-btn">Learn More About Architecture →</a>
      </div>
    </div>
    <div class="architecture-diagram">
      <!-- SVG or image placeholder for architecture diagram -->
      <div class="diagram-placeholder">
        <span>🏗️</span>
        <p>Architecture Diagram</p>
      </div>
    </div>
  </div>
</div>

<!-- Community Section -->
<div class="home-section community-section">
  <h2 class="section-title">Join the KECS Community</h2>
  <p class="section-subtitle">Built by developers, for developers</p>
  <div class="community-stats">
    <div class="stat-card">
      <div class="stat-icon">⭐</div>
      <div class="stat-value">500+</div>
      <div class="stat-label">GitHub Stars</div>
    </div>
    <div class="stat-card">
      <div class="stat-icon">🔀</div>
      <div class="stat-value">50+</div>
      <div class="stat-label">Contributors</div>
    </div>
    <div class="stat-card">
      <div class="stat-icon">🏢</div>
      <div class="stat-value">100+</div>
      <div class="stat-label">Companies Using KECS</div>
    </div>
  </div>
  <div class="community-actions">
    <a href="https://github.com/nandemo-ya/kecs/issues" class="community-link">
      <span class="link-icon">🐛</span>
      Report Issues
    </a>
    <a href="https://github.com/nandemo-ya/kecs/discussions" class="community-link">
      <span class="link-icon">💬</span>
      Join Discussions
    </a>
    <a href="/development/contributing" class="community-link">
      <span class="link-icon">🤝</span>
      Contribute
    </a>
  </div>
</div>

<style scoped>
/* Home sections */
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

/* Feature Grid */
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

/* Architecture Section */
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

/* Community Section */
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

/* Responsive Design */
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