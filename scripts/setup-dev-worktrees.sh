#!/bin/bash
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT=$(git rev-parse --show-toplevel)
WORKTREE_DIR="${PROJECT_ROOT}/../kecs-worktrees"
MAIN_BRANCH="main"

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to create worktree
create_worktree() {
    local branch_name=$1
    local worktree_path="${WORKTREE_DIR}/${branch_name}"
    
    if [ -d "$worktree_path" ]; then
        print_warning "Worktree for ${branch_name} already exists at ${worktree_path}"
        return 0
    fi
    
    print_info "Creating worktree for branch: ${branch_name}"
    
    # Check if branch exists remotely
    if git ls-remote --heads origin "${branch_name}" | grep -q "${branch_name}"; then
        # Branch exists, create worktree from it
        git worktree add "$worktree_path" "${branch_name}"
    else
        # Branch doesn't exist, create new branch from main
        git worktree add -b "${branch_name}" "$worktree_path" "${MAIN_BRANCH}"
    fi
    
    print_success "Worktree created at: ${worktree_path}"
}

# Function to setup .env files
setup_env_files() {
    local worktree_path=$1
    local branch_name=$2
    
    print_info "Setting up .env files for ${branch_name}"
    
    # Create controlplane .env
    cat > "${worktree_path}/controlplane/.env" << EOF
# Environment configuration for ${branch_name}
KECS_API_PORT=8080
KECS_ADMIN_PORT=8081
KECS_REGION=ap-northeast-1
KECS_ACCOUNT_ID=123456789012
KECS_STORAGE_PATH=/tmp/kecs-${branch_name}.db
KECS_LOG_LEVEL=debug
EOF

    
    # Create mcp-server .env
    if [ -d "${worktree_path}/mcp-server" ]; then
        cat > "${worktree_path}/mcp-server/.env" << EOF
# MCP Server configuration for ${branch_name}
MCP_KECS_ENDPOINT=http://localhost:8080
MCP_LOG_LEVEL=debug
EOF
    fi
    
    print_success ".env files created for ${branch_name}"
}

# Function to setup development dependencies
setup_dependencies() {
    local worktree_path=$1
    local branch_name=$2
    
    print_info "Setting up dependencies for ${branch_name}"
    
    # Go dependencies
    if [ -f "${worktree_path}/controlplane/go.mod" ]; then
        print_info "Downloading Go dependencies..."
        (cd "${worktree_path}/controlplane" && go mod download)
    fi
    
    
    # Node dependencies for mcp-server
    if [ -f "${worktree_path}/mcp-server/package.json" ]; then
        print_info "Installing mcp-server dependencies..."
        (cd "${worktree_path}/mcp-server" && npm install)
    fi
    
    # Node dependencies for docs-site
    if [ -f "${worktree_path}/docs-site/package.json" ]; then
        print_info "Installing docs-site dependencies..."
        (cd "${worktree_path}/docs-site" && npm install)
    fi
    
    print_success "Dependencies installed for ${branch_name}"
}

# Function to setup Claude settings
setup_claude_settings() {
    local worktree_path=$1
    local branch_name=$2
    local main_repo_path="${PROJECT_ROOT}"
    local claude_dir="${worktree_path}/.claude"
    
    print_info "Setting up Claude settings for ${branch_name}"
    
    # Create .claude directory if it doesn't exist
    mkdir -p "$claude_dir"
    
    # Copy settings.local.json from main repo if it exists
    if [ -f "${main_repo_path}/.claude/settings.local.json" ]; then
        # Read the original file and update paths
        local old_path=$(echo "$main_repo_path" | sed 's/[[\.*^$()+?{|]/\\&/g')
        local new_path=$(echo "$worktree_path" | sed 's/[[\.*^$()+?{|]/\\&/g')
        
        # Copy and update paths in settings.local.json
        sed "s|$old_path|$new_path|g" "${main_repo_path}/.claude/settings.local.json" > "${claude_dir}/settings.local.json"
        
        print_success "Claude settings copied and paths updated"
    else
        print_warning "No .claude/settings.local.json found in main repository"
    fi
}

# Function to create VS Code workspace file
create_vscode_workspace() {
    local branch_name=$1
    local worktree_path="${WORKTREE_DIR}/${branch_name}"
    # Replace slashes with dashes for filename
    local safe_branch_name=$(echo "$branch_name" | tr '/' '-')
    local workspace_file="${worktree_path}/kecs-${safe_branch_name}.code-workspace"
    
    print_info "Creating VS Code workspace file for ${branch_name}"
    
    cat > "$workspace_file" << EOF
{
    "folders": [
        {
            "name": "Control Plane",
            "path": "controlplane"
        },
        {
            "name": "MCP Server",
            "path": "mcp-server"
        },
        {
            "name": "Docs Site",
            "path": "docs-site"
        },
        {
            "name": "Root",
            "path": "."
        }
    ],
    "settings": {
        "go.gopath": "",
        "go.inferGopath": false,
        "go.useLanguageServer": true,
        "editor.formatOnSave": true,
        "editor.codeActionsOnSave": {
            "source.organizeImports": true
        },
        "window.title": "KECS - ${branch_name}"
    },
    "launch": {
        "version": "0.2.0",
        "configurations": [
            {
                "name": "Launch Control Plane",
                "type": "go",
                "request": "launch",
                "mode": "auto",
                "program": "\${workspaceFolder:Control Plane}/cmd/controlplane",
                "args": ["server"],
                "env": {
                    "KECS_STORAGE_PATH": "/tmp/kecs-${branch_name}.db"
                }
            }
        ]
    }
}
EOF
    
    print_success "VS Code workspace created at: ${workspace_file}"
}

# Function to show worktree status
show_worktree_status() {
    print_info "Current worktree status:"
    git worktree list
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS] <branch-name> [<branch-name> ...]"
    echo ""
    echo "Options:"
    echo "  -h, --help          Show this help message"
    echo "  -s, --skip-deps     Skip dependency installation"
    echo "  -u, --no-update     Don't update main branch"
    echo "  -U, --update        Update existing worktrees instead of creating new ones"
    echo ""
    echo "Examples:"
    echo "  $0 feat/localstack-integration feat/proxy-modes"
    echo "  $0 --skip-deps fix/kubeconfig-path"
    echo "  $0 feat/new-feature"
    echo "  $0 --update feat/localstack-integration  # Update existing worktree"
    echo ""
    echo "Note: Branch names can be with or without 'feat/' or 'fix/' prefix"
    echo ""
    echo "To update worktrees later, use:"
    echo "  ./scripts/update-worktrees.sh --all"
    echo "  ./scripts/update-worktrees.sh feat/localstack-integration"
}

# Main script
main() {
    local SKIP_DEPS=false
    local UPDATE_MAIN=true
    local UPDATE_MODE=false
    local BRANCHES=()
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -s|--skip-deps)
                SKIP_DEPS=true
                shift
                ;;
            -u|--no-update)
                UPDATE_MAIN=false
                shift
                ;;
            -U|--update)
                UPDATE_MODE=true
                shift
                ;;
            -*)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
            *)
                BRANCHES+=("$1")
                shift
                ;;
        esac
    done
    
    # Check if branches were provided
    if [ ${#BRANCHES[@]} -eq 0 ]; then
        print_error "No branch names provided"
        show_usage
        exit 1
    fi
    
    echo "Git Worktree Development Setup for KECS"
    echo "======================================="
    
    # If update mode, delegate to update script
    if [ "$UPDATE_MODE" = true ]; then
        print_info "Delegating to update-worktrees.sh..."
        exec "${PROJECT_ROOT}/scripts/update-worktrees.sh" "${BRANCHES[@]}"
    fi
    
    # Create worktree directory if it doesn't exist
    mkdir -p "$WORKTREE_DIR"
    
    # Update main branch if requested
    if [ "$UPDATE_MAIN" = true ]; then
        print_info "Updating main branch..."
        git checkout main
        git pull origin main
    fi
    
    # Create worktrees for each specified branch
    for branch in "${BRANCHES[@]}"; do
        echo ""
        print_info "Processing branch: ${branch}"
        create_worktree "$branch"
        setup_env_files "${WORKTREE_DIR}/${branch}" "$branch"
        setup_claude_settings "${WORKTREE_DIR}/${branch}" "$branch"
        
        if [ "$SKIP_DEPS" = false ]; then
            setup_dependencies "${WORKTREE_DIR}/${branch}" "$branch"
        else
            print_info "Skipping dependency installation for ${branch}"
        fi
        
        create_vscode_workspace "$branch"
    done
    
    echo ""
    show_worktree_status
    
    echo ""
    print_success "Worktree setup complete!"
    echo ""
    echo "To work on a feature:"
    echo "  cd ${WORKTREE_DIR}/<branch-name>"
    echo "  code kecs-<branch-name>.code-workspace  # Note: slashes in branch names are replaced with dashes"
    echo ""
    echo "To run tests in a worktree:"
    echo "  cd ${WORKTREE_DIR}/<branch-name>/controlplane"
    echo "  make test"
    echo ""
    echo "To remove a worktree when done:"
    echo "  git worktree remove ${WORKTREE_DIR}/<branch-name>"
}

# Run main function
main "$@"