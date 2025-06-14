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

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS] [<branch-name> ...]"
    echo ""
    echo "Update existing Git worktrees with latest changes from main branch"
    echo ""
    echo "Options:"
    echo "  -h, --help          Show this help message"
    echo "  -a, --all           Update all existing worktrees"
    echo "  -m, --merge         Use merge instead of rebase (default: rebase)"
    echo "  -s, --status        Show status of all worktrees without updating"
    echo "  -c, --continue      Continue after resolving conflicts"
    echo "  -A, --abort         Abort current rebase/merge operation"
    echo ""
    echo "Examples:"
    echo "  $0 --all                    # Update all worktrees"
    echo "  $0 feat/localstack-integration  # Update specific worktree"
    echo "  $0 --merge feat/proxy-modes     # Update using merge"
    echo "  $0 --status                 # Show status only"
}

# Function to get all worktree branches
get_worktree_branches() {
    git worktree list --porcelain | grep "^worktree" | awk '{print $2}' | grep -v "^${PROJECT_ROOT}$" | while read -r path; do
        if [ -d "$path" ]; then
            basename "$path"
        fi
    done
}

# Function to check if branch has uncommitted changes
has_uncommitted_changes() {
    local worktree_path=$1
    cd "$worktree_path"
    if [ -n "$(git status --porcelain)" ]; then
        return 0
    else
        return 1
    fi
}

# Function to show worktree status
show_worktree_status() {
    local branch=$1
    local worktree_path="${WORKTREE_DIR}/${branch}"
    
    if [ ! -d "$worktree_path" ]; then
        print_error "Worktree not found: $worktree_path"
        return 1
    fi
    
    cd "$worktree_path"
    
    # Get current branch
    local current_branch=$(git branch --show-current)
    
    # Check for uncommitted changes
    local changes=""
    if has_uncommitted_changes "$worktree_path"; then
        changes="${RED}[uncommitted changes]${NC}"
    else
        changes="${GREEN}[clean]${NC}"
    fi
    
    # Get commits behind/ahead of main
    git fetch origin main &>/dev/null
    local behind=$(git rev-list --count HEAD..origin/main)
    local ahead=$(git rev-list --count origin/main..HEAD)
    
    echo -e "${BLUE}${branch}${NC}: ${changes} - ${behind} behind, ${ahead} ahead of origin/main"
}

# Function to update worktree
update_worktree() {
    local branch=$1
    local use_merge=$2
    local worktree_path="${WORKTREE_DIR}/${branch}"
    
    print_info "Updating worktree: ${branch}"
    
    if [ ! -d "$worktree_path" ]; then
        print_error "Worktree not found: $worktree_path"
        return 1
    fi
    
    cd "$worktree_path"
    
    # Check for uncommitted changes
    if has_uncommitted_changes "$worktree_path"; then
        print_warning "Worktree has uncommitted changes. Please commit or stash them first."
        echo "  cd $worktree_path"
        echo "  git status"
        return 1
    fi
    
    # Fetch latest changes
    print_info "Fetching latest changes..."
    git fetch origin
    
    # Check if update is needed
    local behind=$(git rev-list --count HEAD..origin/main)
    if [ "$behind" -eq 0 ]; then
        print_success "Already up to date with origin/main"
        return 0
    fi
    
    # Perform update
    if [ "$use_merge" = true ]; then
        print_info "Merging origin/main..."
        if git merge origin/main; then
            print_success "Successfully merged origin/main"
        else
            print_error "Merge conflict detected. Resolve conflicts and run:"
            echo "  cd $worktree_path"
            echo "  git merge --continue"
            return 1
        fi
    else
        print_info "Rebasing onto origin/main..."
        if git rebase origin/main; then
            print_success "Successfully rebased onto origin/main"
        else
            print_error "Rebase conflict detected. Resolve conflicts and run:"
            echo "  cd $worktree_path"
            echo "  git rebase --continue"
            return 1
        fi
    fi
}

# Function to continue rebase/merge
continue_operation() {
    local branch=$1
    local worktree_path="${WORKTREE_DIR}/${branch}"
    
    if [ ! -d "$worktree_path" ]; then
        print_error "Worktree not found: $worktree_path"
        return 1
    fi
    
    cd "$worktree_path"
    
    # Check if in rebase
    if [ -d ".git/rebase-merge" ] || [ -d ".git/rebase-apply" ]; then
        print_info "Continuing rebase..."
        git rebase --continue
    # Check if in merge
    elif [ -f ".git/MERGE_HEAD" ]; then
        print_info "Continuing merge..."
        git merge --continue
    else
        print_warning "No rebase or merge in progress"
    fi
}

# Function to abort rebase/merge
abort_operation() {
    local branch=$1
    local worktree_path="${WORKTREE_DIR}/${branch}"
    
    if [ ! -d "$worktree_path" ]; then
        print_error "Worktree not found: $worktree_path"
        return 1
    fi
    
    cd "$worktree_path"
    
    # Check if in rebase
    if [ -d ".git/rebase-merge" ] || [ -d ".git/rebase-apply" ]; then
        print_info "Aborting rebase..."
        git rebase --abort
        print_success "Rebase aborted"
    # Check if in merge
    elif [ -f ".git/MERGE_HEAD" ]; then
        print_info "Aborting merge..."
        git merge --abort
        print_success "Merge aborted"
    else
        print_warning "No rebase or merge in progress"
    fi
}

# Main script
main() {
    local UPDATE_ALL=false
    local USE_MERGE=false
    local SHOW_STATUS=false
    local CONTINUE_OP=false
    local ABORT_OP=false
    local BRANCHES=()
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -a|--all)
                UPDATE_ALL=true
                shift
                ;;
            -m|--merge)
                USE_MERGE=true
                shift
                ;;
            -s|--status)
                SHOW_STATUS=true
                shift
                ;;
            -c|--continue)
                CONTINUE_OP=true
                shift
                ;;
            -A|--abort)
                ABORT_OP=true
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
    
    echo "Git Worktree Update Tool for KECS"
    echo "================================="
    echo ""
    
    # First, update main branch in the main repository
    cd "$PROJECT_ROOT"
    print_info "Updating main branch..."
    git fetch origin main:main
    
    # Get all branches if --all is specified
    if [ "$UPDATE_ALL" = true ]; then
        mapfile -t BRANCHES < <(get_worktree_branches)
        if [ ${#BRANCHES[@]} -eq 0 ]; then
            print_warning "No worktrees found"
            exit 0
        fi
    fi
    
    # Check if branches were provided
    if [ ${#BRANCHES[@]} -eq 0 ] && [ "$SHOW_STATUS" = false ]; then
        print_error "No branch names provided"
        show_usage
        exit 1
    fi
    
    # Show status only
    if [ "$SHOW_STATUS" = true ]; then
        print_info "Worktree status:"
        echo ""
        if [ ${#BRANCHES[@]} -eq 0 ]; then
            mapfile -t BRANCHES < <(get_worktree_branches)
        fi
        for branch in "${BRANCHES[@]}"; do
            show_worktree_status "$branch"
        done
        exit 0
    fi
    
    # Handle continue operation
    if [ "$CONTINUE_OP" = true ]; then
        if [ ${#BRANCHES[@]} -ne 1 ]; then
            print_error "Please specify exactly one branch for --continue"
            exit 1
        fi
        continue_operation "${BRANCHES[0]}"
        exit $?
    fi
    
    # Handle abort operation
    if [ "$ABORT_OP" = true ]; then
        if [ ${#BRANCHES[@]} -ne 1 ]; then
            print_error "Please specify exactly one branch for --abort"
            exit 1
        fi
        abort_operation "${BRANCHES[0]}"
        exit $?
    fi
    
    # Update worktrees
    local failed=0
    for branch in "${BRANCHES[@]}"; do
        echo ""
        if ! update_worktree "$branch" "$USE_MERGE"; then
            ((failed++))
        fi
    done
    
    echo ""
    if [ $failed -eq 0 ]; then
        print_success "All worktrees updated successfully!"
    else
        print_warning "$failed worktree(s) failed to update"
        exit 1
    fi
}

# Run main function
main "$@"