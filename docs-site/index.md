---
layout: home

hero:
  name: "KECS"
  text: "Kubernetes-based ECS Compatible Service"
  tagline: "Run Amazon ECS workloads locally on Kubernetes"
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
    details: Full compatibility with Amazon ECS APIs, allowing seamless local development
  - icon: ☸️
    title: Kubernetes Native
    details: Built on Kubernetes for reliability and scalability
  - icon: 🛠️
    title: Developer Friendly
    details: Simple setup with Kind and WebSocket real-time updates
  - icon: 📦
    title: Production Ready
    details: DuckDB persistence, graceful shutdown, and comprehensive monitoring
---

## Quick Start

Get started with KECS in minutes:

```bash
# Clone the repository
git clone https://github.com/nandemo-ya/kecs.git
cd kecs

# Build and run
make build
./bin/kecs server

```

## Why KECS?

KECS provides a fully local ECS-compatible environment that operates independently of AWS. Perfect for:

- **Local Development**: Test ECS workloads without AWS costs
- **CI/CD Pipelines**: Run integration tests in isolated environments
- **Learning**: Understand ECS concepts without AWS account
- **Offline Development**: Work on ECS applications without internet

## Architecture Overview

KECS implements the ECS API specification on top of Kubernetes:

- **Control Plane**: Handles ECS API requests and manages state
- **Storage Layer**: DuckDB for persistent storage
- **Kubernetes Backend**: Translates ECS concepts to Kubernetes resources

## Community

Join our community and contribute:

- [GitHub Issues](https://github.com/nandemo-ya/kecs/issues)
- [Discussions](https://github.com/nandemo-ya/kecs/discussions)
- [Contributing Guide](/development/contributing)