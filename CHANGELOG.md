# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Container-based background execution commands (#252)
  - `kecs start` - Start KECS in a Docker container with customizable options
  - `kecs stop` - Stop and remove KECS container
  - `kecs status` - Show container status
  - `kecs logs` - Display container logs with follow support
- Multiple instance support (#252 Phase 2)
  - Run multiple KECS instances with different names and ports
  - `--auto-port` flag for automatic port assignment
  - Configuration file support for managing multiple instances
  - `kecs instances` command for batch management
- Environment variable management using Viper (#250)
  - Centralized configuration management
  - Support for config files, environment variables, and CLI flags
  - Type-safe configuration access
- Container features
  - Health check with 30-second timeout
  - Data persistence through volume mounts
  - Local build support with `--local-build` flag
  - Container labeling for better identification
- Web UI configuration options (#253 Phase 1)
  - `--no-webui` flag to disable Web UI
  - `KECS_WEBUI_ENABLED` environment variable
  - Configurable via `ui.enabled` in config file
  - Improves resource usage and startup time when UI not needed
- Separated UI/API deployment support (#253 Phase 2)
  - `kecs start-ui` command to run UI in separate container
  - Traefik-powered Web UI with advanced routing
  - Separate Docker images: `kecs-api` and `kecs-ui`
  - Docker Compose profiles for combined/separated modes
  - Runtime configuration injection for API endpoints

### Changed

- Improved configuration management with structured config types
- Enhanced error messages for Docker operations
- Status command now uses container labels for filtering

### Fixed

- Port conflict detection before container creation
- Proper cleanup of data directories on container removal
- Local build path detection for different execution contexts

## [Previous Releases]

### Features in Development

- ECS API Compatibility
- Kubernetes Backend Integration
- Web UI Dashboard
- MCP Server for AI Assistant Integration
- LocalStack Integration
- DuckDB Storage Layer