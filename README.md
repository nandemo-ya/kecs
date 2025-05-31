# kecs

[![CI/CD Pipeline](https://github.com/nandemo-ya/kecs/actions/workflows/ci.yml/badge.svg)](https://github.com/nandemo-ya/kecs/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/nandemo-ya/kecs/branch/main/graph/badge.svg)](https://codecov.io/gh/nandemo-ya/kecs)
[![Go Version](https://img.shields.io/badge/Go-1.24.3-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Release](https://img.shields.io/github/release/nandemo-ya/kecs.svg)](https://github.com/nandemo-ya/kecs/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/nandemo-ya/kecs)](https://goreportcard.com/report/github.com/nandemo-ya/kecs)
[![GoDoc](https://pkg.go.dev/badge/github.com/nandemo-ya/kecs)](https://pkg.go.dev/github.com/nandemo-ya/kecs)

## Overview

KECS (Kubernetes-based ECS Compatible Service) is a standalone service that provides Amazon ECS compatible APIs running on Kubernetes. It enables a fully local ECS-compatible environment that operates independently of AWS environments.

### Key Features

- **ECS API Compatibility**: Provides API endpoints compatible with Amazon ECS
- **Kubernetes Backend**: Leverages Kubernetes for container orchestration
- **Local Execution**: Runs completely locally without AWS dependencies
- **CI/CD Integration**: Easily integrates with CI/CD pipelines

## Documentation

Architectural Decision Records (ADRs) for this project are stored in the `docs/adr/records` directory.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.