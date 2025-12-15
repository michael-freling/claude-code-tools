#!/bin/bash

set -euo pipefail

echo "Setting up development environment..."

# Check if pre-commit is installed
if ! command -v pre-commit &> /dev/null; then
    echo "ERROR: pre-commit is not installed"
    echo "Please install pre-commit:"
    echo "  - Using pip: pip install pre-commit"
    echo "  - Using homebrew: brew install pre-commit"
    echo "  - See https://pre-commit.com/#installation for more options"
    exit 1
fi

echo "pre-commit version: $(pre-commit --version)"

# Check if golangci-lint is installed
if ! command -v golangci-lint &> /dev/null; then
    echo "ERROR: golangci-lint is not installed"
    echo "Please install golangci-lint v2.0+:"
    echo "  - Using go install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
    echo "  - Using homebrew: brew install golangci-lint"
    echo "  - See https://golangci-lint.run/welcome/install/ for more options"
    exit 1
fi

echo "golangci-lint version: $(golangci-lint version 2>&1 | head -1)"

# Check if goimports is installed
if ! command -v goimports &> /dev/null; then
    echo "ERROR: goimports is not installed"
    echo "Please install goimports:"
    echo "  - Using go install: go install golang.org/x/tools/cmd/goimports@latest"
    exit 1
fi

echo "goimports installed: $(which goimports)"

# Install pre-commit hooks (idempotent)
echo "Installing pre-commit hooks..."
pre-commit install

# Install pre-push hooks for e2e tests (idempotent)
echo "Installing pre-push hooks..."
pre-commit install --hook-type pre-push

# Run initial validation on all files
echo "Running initial validation on all files..."
if pre-commit run --all-files; then
    echo ""
    echo "SUCCESS: All pre-commit hooks passed!"
    echo "Your development environment is ready."
else
    echo ""
    echo "WARNING: Some pre-commit hooks failed."
    echo "Please fix the issues above before committing."
    echo "The hooks have been installed and will run automatically on commit."
    exit 1
fi

echo ""
echo "Development environment setup complete!"
echo "Pre-commit hooks will now run automatically before each commit."
echo "Pre-push hooks (e2e tests) will run automatically before each push."
