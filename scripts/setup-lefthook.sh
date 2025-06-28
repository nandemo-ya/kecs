#!/bin/bash
# Setup script for Lefthook git hooks

set -e

echo "Setting up Lefthook for KECS development..."

# Detect OS
OS=$(uname -s)
ARCH=$(uname -m)

# Install lefthook based on OS
install_lefthook() {
    if command -v lefthook &> /dev/null; then
        echo "✓ Lefthook is already installed"
        return
    fi

    echo "Installing Lefthook..."
    
    if [[ "$OS" == "Darwin" ]]; then
        # macOS
        if command -v brew &> /dev/null; then
            echo "Installing via Homebrew..."
            brew install lefthook
        else
            echo "Installing via curl..."
            curl -sSfL https://raw.githubusercontent.com/evilmartians/lefthook/master/install.sh | sh -s -- -b /usr/local/bin
        fi
    elif [[ "$OS" == "Linux" ]]; then
        # Linux
        echo "Installing via curl..."
        curl -sSfL https://raw.githubusercontent.com/evilmartians/lefthook/master/install.sh | sh -s -- -b /usr/local/bin
    else
        echo "Unsupported OS: $OS"
        echo "Please install Lefthook manually: https://github.com/evilmartians/lefthook#installation"
        exit 1
    fi
}

# Install lefthook
install_lefthook

# Verify installation
if ! command -v lefthook &> /dev/null; then
    echo "❌ Lefthook installation failed"
    exit 1
fi

echo "✓ Lefthook $(lefthook version) installed successfully"

# Install git hooks
echo "Installing git hooks..."
lefthook install

if [ $? -eq 0 ]; then
    echo "✓ Git hooks installed successfully"
else
    echo "❌ Failed to install git hooks"
    exit 1
fi

# Add lefthook to PATH reminder
echo ""
echo "========================================="
echo "Lefthook setup completed!"
echo ""
echo "The following hooks are now active:"
echo "  - pre-commit: Runs tests and linting"
echo "  - pre-push: Runs full test suite"
echo ""
echo "To skip hooks temporarily, use:"
echo "  git commit --no-verify"
echo "  git push --no-verify"
echo ""
echo "To uninstall hooks:"
echo "  lefthook uninstall"
echo "========================================="