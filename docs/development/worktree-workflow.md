# Git Worktree Development Workflow

This document describes how to use Git worktrees for parallel development of multiple features in the KECS project.

## Overview

Git worktrees allow you to have multiple branches checked out simultaneously in different directories. This is particularly useful when:

- Working on multiple features in parallel
- Switching between branches without losing local changes
- Testing different branches side by side
- Using AI agents that work in different directories

## Quick Start

```bash
# Run the setup script with branch names
./scripts/setup-dev-worktrees.sh feat/localstack-integration feat/proxy-modes

# Multiple branches at once
./scripts/setup-dev-worktrees.sh feat/taskset-implementation fix/kubeconfig-path feat/new-feature

# Skip dependency installation for faster setup
./scripts/setup-dev-worktrees.sh --skip-deps feat/quick-fix

# Skip main branch update
./scripts/setup-dev-worktrees.sh --no-update feat/experimental
```

This will:
1. Create worktrees for specified branches
2. Set up .env files for each worktree
3. Copy and update Claude settings (.claude/settings.local.json) with corrected paths
4. Install dependencies (Go modules, npm packages) unless --skip-deps is used
5. Create VS Code workspace files

## Workflow Structure

```
kecs/                          # Main repository
kecs-worktrees/               # Worktree directory
├── feat/localstack-integration/
│   ├── .claude/settings.local.json  # Claude settings with updated paths
│   ├── controlplane/.env
│   ├── mcp-server/.env
│   └── kecs-feat-localstack-integration.code-workspace
├── feat/proxy-modes/
├── feat/taskset-implementation/
└── fix/kubeconfig-path/
```

## Working with Worktrees

### Creating a New Worktree

```bash
# Create worktree for existing branch
git worktree add ../kecs-worktrees/feat/new-feature feat/new-feature

# Create worktree with new branch
git worktree add -b feat/new-feature ../kecs-worktrees/feat/new-feature main
```

### Switching Between Worktrees

```bash
# Simply navigate to the worktree directory
cd ../kecs-worktrees/feat/localstack-integration

# Open in VS Code
code kecs-feat-localstack-integration.code-workspace
```

### Running Services in Worktrees

Each worktree has its own .env configuration with unique ports and database paths:

```bash
# Terminal 1: Control Plane
cd ../kecs-worktrees/feat/localstack-integration/controlplane
make run

# Terminal 2: MCP Server
cd ../kecs-worktrees/feat/localstack-integration/mcp-server
npm run dev
```

### Testing in Worktrees

```bash
# Run control plane tests
cd ../kecs-worktrees/feat/taskset-implementation/controlplane
make test

# Run scenario tests
cd ../kecs-worktrees/feat/taskset-implementation/tests/scenarios
make test
```

## Environment Configuration

Each worktree gets its own set of .env files:

### controlplane/.env
```env
KECS_API_PORT=8080
KECS_ADMIN_PORT=8081
KECS_STORAGE_PATH=/tmp/kecs-<branch-name>.db
```

### mcp-server/.env
```env
MCP_KECS_ENDPOINT=http://localhost:8080
MCP_LOG_LEVEL=debug
```

## VS Code Integration

Each worktree includes a VS Code workspace file with:
- Multi-folder workspace setup
- Configured launch configurations
- Go and TypeScript settings
- Branch-specific window titles

Open a workspace:
```bash
code ../kecs-worktrees/feat/proxy-modes/kecs-feat-proxy-modes.code-workspace
```

Note: For branch names with slashes (e.g., `feat/proxy-modes`), the workspace filename will have dashes instead (e.g., `kecs-feat-proxy-modes.code-workspace`).

## AI Agent Development

When using AI agents (like Claude) for development:

1. **Open separate agent sessions** for different features:
   ```bash
   # Agent 1: LocalStack integration
   cd ../kecs-worktrees/feat/localstack-integration
   
   # Agent 2: Proxy modes
   cd ../kecs-worktrees/feat/proxy-modes
   ```

2. **Claude settings are automatically configured**:
   - `.claude/settings.local.json` is copied from the main repository
   - All paths in the settings are automatically updated to the worktree path
   - Permissions and configurations remain consistent across worktrees

3. **Provide context** to each agent:
   - Current branch and feature being worked on
   - Relevant ADRs and documentation
   - Any dependencies on other features

4. **Coordinate between agents** when needed:
   - Share common interfaces
   - Avoid conflicting changes
   - Merge regularly with main branch

## Updating Worktrees

### Update All Worktrees at Once
```bash
# Update all existing worktrees with latest main branch changes
./scripts/update-worktrees.sh --all

# Use merge instead of rebase
./scripts/update-worktrees.sh --all --merge
```

### Update Specific Worktrees
```bash
# Update single worktree
./scripts/update-worktrees.sh feat/localstack-integration

# Update multiple worktrees
./scripts/update-worktrees.sh feat/proxy-modes feat/taskset-implementation
```

### Check Worktree Status
```bash
# Show status of all worktrees
./scripts/update-worktrees.sh --status

# Show status of specific worktrees
./scripts/update-worktrees.sh --status feat/localstack-integration
```

### Handle Conflicts
```bash
# If conflicts occur during update
cd ../kecs-worktrees/feat/localstack-integration
# Fix conflicts in your editor
git add .
./scripts/update-worktrees.sh --continue feat/localstack-integration

# Or abort the operation
./scripts/update-worktrees.sh --abort feat/localstack-integration
```

## Best Practices

1. **Keep main branch updated**:
   ```bash
   git checkout main
   git pull origin main
   ```

2. **Update worktrees regularly**:
   ```bash
   # Update all worktrees
   ./scripts/update-worktrees.sh --all
   
   # Or manually in each worktree
   cd ../kecs-worktrees/feat/localstack-integration
   git fetch origin
   git rebase origin/main
   ```

3. **Clean up completed worktrees**:
   ```bash
   git worktree remove ../kecs-worktrees/feat/completed-feature
   ```

4. **List all worktrees**:
   ```bash
   git worktree list
   ```

5. **Prune stale worktrees**:
   ```bash
   git worktree prune
   ```

6. **Before pushing changes**:
   ```bash
   # Always update with main first
   ./scripts/update-worktrees.sh feat/my-feature
   
   # Run tests
   cd ../kecs-worktrees/feat/my-feature/controlplane
   make test
   
   # Push to remote
   git push origin feat/my-feature
   ```

## Troubleshooting

### Port Conflicts
If you encounter port conflicts when running multiple worktrees:
1. Modify the .env file in the specific worktree
2. Use different port numbers for each instance

### Database Conflicts
Each worktree uses a separate database file:
- `/tmp/kecs-feat-localstack-integration.db`
- `/tmp/kecs-feat-proxy-modes.db`

### Dependency Issues
If dependencies are out of sync:
```bash
# Go dependencies
cd controlplane && go mod download

# Node dependencies
cd mcp-server && npm install
```

## Manual Worktree Setup

If you prefer to set up worktrees manually:

```bash
# 1. Create worktree
git worktree add ../kecs-worktrees/feat/my-feature -b feat/my-feature

# 2. Navigate to worktree
cd ../kecs-worktrees/feat/my-feature

# 3. Create .env files
cat > controlplane/.env << EOF
KECS_API_PORT=8080
KECS_ADMIN_PORT=8081
KECS_STORAGE_PATH=/tmp/kecs-my-feature.db
EOF

# 4. Copy and update Claude settings
mkdir -p .claude
sed "s|/path/to/main/repo|$(pwd)|g" /path/to/main/repo/.claude/settings.local.json > .claude/settings.local.json

# 5. Install dependencies
cd controlplane && go mod download
cd ../mcp-server && npm install

# 6. Start development
make run
```

## Integration with CI/CD

Before pushing changes:
1. Run tests in your worktree
2. Test with `act` if modifying workflows:
   ```bash
   act -W .github/workflows/ci.yml --container-architecture linux/amd64
   ```
3. Create PR from your feature branch