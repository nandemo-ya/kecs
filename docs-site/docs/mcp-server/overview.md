---
sidebar_position: 1
---

# MCP Server Overview

KECS provides a Model Context Protocol (MCP) server that enables AI assistants like Claude to interact with your ECS-compatible environment.

## What is MCP?

The Model Context Protocol (MCP) is a standard protocol that allows AI assistants to interact with external tools and services. KECS's MCP server implements this protocol to provide seamless integration with Claude Desktop, Claude Code (VS Code), and other MCP-compatible clients.

## Features

- **Full ECS API Coverage**: Access all KECS functionality through natural language
- **Real-time Operations**: Create, update, and manage ECS resources interactively
- **Type-safe Implementation**: Built with TypeScript for reliability
- **Easy Integration**: Simple configuration for Claude Desktop and VS Code

## Available Tools

The KECS MCP server provides the following tools:

### Cluster Management
- `list-clusters` - List all ECS clusters
- `describe-clusters` - Get detailed cluster information
- `create-cluster` - Create new clusters
- `delete-cluster` - Remove clusters

### Service Management
- `list-services` - List services in a cluster
- `describe-services` - Get service details
- `create-service` - Deploy new services
- `update-service` - Modify existing services
- `delete-service` - Remove services

### Task Management
- `list-tasks` - List running tasks
- `describe-tasks` - Get task details
- `run-task` - Start new tasks
- `stop-task` - Stop running tasks

### Task Definition Management
- `list-task-definitions` - List task definitions
- `describe-task-definition` - Get task definition details
- `register-task-definition` - Create new task definitions
- `deregister-task-definition` - Remove task definitions

## Use Cases

- **Interactive Development**: Use natural language to manage your local ECS environment
- **Learning and Exploration**: Explore ECS concepts with AI assistance
- **Automation**: Build conversational workflows for common tasks
- **Troubleshooting**: Get AI-powered help debugging ECS issues

## Next Steps

- [Installation Guide](./installation.md) - Set up the MCP server
- [Claude Desktop Setup](./claude-desktop.md) - Configure Claude Desktop
- [Claude Code Setup](./claude-code.md) - Configure VS Code integration
- [API Reference](./api-reference.md) - Detailed tool documentation