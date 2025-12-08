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

# Install pre-commit hooks (idempotent)
echo "Installing pre-commit hooks..."
pre-commit install

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
