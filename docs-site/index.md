---
layout: home

hero:
  name: "KECS"
  text: "Kubernetes-based ECS Compatible Service"
  tagline: "Run Amazon ECS workloads locally with full AWS service integration"
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
    title: ECS & ELBv2 Compatible
    details: Full compatibility with Amazon ECS and ELBv2 APIs, enabling seamless local development and testing
  - icon: 🌐
    title: Complete AWS Environment
    details: Includes LocalStack for IAM, S3, Secrets Manager, SSM, CloudWatch Logs, and more AWS services
  - icon: 🎯
    title: Single Endpoint
    details: Access all AWS services through one unified endpoint (port 5373) - no complex configuration needed
  - icon: 🖥️
    title: Interactive TUI
    details: Manage clusters, services, and tasks visually with the built-in Terminal User Interface
  - icon: 🔧
    title: Zero Configuration
    details: Start with a single command - KECS handles all the setup and configuration automatically
  - icon: 🚢
    title: Multiple Instances
    details: Run isolated KECS environments for different projects simultaneously without conflicts
---

## Quick Start

Get KECS running in under a minute:

```bash
# Install KECS
brew install nandemo-ya/tap/kecs
# Or download from GitHub releases

# Start KECS (creates k3d cluster automatically)
kecs start

# Check status
kecs cluster info

# Use with AWS CLI
export AWS_ENDPOINT_URL=http://localhost:5373
aws ecs create-cluster --cluster-name my-cluster
aws ecs list-clusters
```

## What is KECS?

KECS (Kubernetes-based ECS Compatible Service) provides a complete local Amazon ECS environment that runs entirely on your machine. It's designed for developers who want to:

- **Develop locally** without AWS costs or internet connectivity
- **Test ECS workloads** in a production-like environment
- **CI/CD integration** with consistent, reproducible environments
- **Learn ECS** without needing an AWS account

## Key Features

### 🏗️ Complete ECS Implementation
- Full ECS API compatibility (clusters, services, tasks, task definitions)
- ELBv2 support (Application and Network Load Balancers)
- Service Discovery integration
- Auto-scaling capabilities

### 🌐 Unified AWS Experience
- Single endpoint for all AWS services (port 5373)
- Built-in LocalStack integration
- Automatic IAM role creation (`ecsTaskExecutionRole`)
- Seamless AWS CLI/SDK compatibility

### 👨‍💻 Developer Focused
- Container-based execution - no complex setup
- Multiple instance support for different projects
- Interactive TUI for resource management
- Hot reload for rapid development

### 🔧 Production-Ready Features
- DuckDB for reliable state persistence
- Graceful shutdown and cleanup
- Resource monitoring and health checks
- Comprehensive logging and debugging

## Architecture Overview

KECS runs its control plane inside a k3d cluster, providing:

```
┌─────────────────────────────────────┐
│         Your Application            │
│         (AWS CLI/SDK)                │
└────────────┬────────────────────────┘
             │
             ▼ Port 5373
┌─────────────────────────────────────┐
│       Traefik Gateway               │
│   (Unified AWS API Endpoint)        │
├─────────────┬───────────────────────┤
│    KECS     │     LocalStack        │
│  ECS APIs   │   Other AWS APIs      │
│  ELBv2 APIs │  (IAM, S3, SSM, etc)  │
└─────────────┴───────────────────────┘
             │
             ▼
┌─────────────────────────────────────┐
│         k3d Cluster                 │
│    (Kubernetes Workloads)           │
└─────────────────────────────────────┘
```

## Use Cases

### Local Development
```bash
# Start development environment
kecs start --instance dev

# Deploy your application
aws ecs create-service \
  --cluster my-cluster \
  --service-name my-app \
  --task-definition my-app:1
```

### CI/CD Testing
```yaml
# GitHub Actions example
- name: Start KECS
  run: kecs start
  
- name: Run integration tests
  run: |
    export AWS_ENDPOINT_URL=http://localhost:5373
    npm run test:integration
```

### Multiple Projects
```bash
# Run multiple isolated environments
kecs start --instance project-a --api-port 8080
kecs start --instance project-b --api-port 8090

# List all instances
kecs cluster list
```

## Getting Help

- 📖 [Documentation](/guides/getting-started) - Comprehensive guides and tutorials
- 🐛 [Issue Tracker](https://github.com/nandemo-ya/kecs/issues) - Report bugs or request features
- 📝 [Examples](https://github.com/nandemo-ya/kecs/tree/main/examples) - Sample applications and configurations

## License

KECS is open source software licensed under the [Apache 2.0 License](https://github.com/nandemo-ya/kecs/blob/main/LICENSE).