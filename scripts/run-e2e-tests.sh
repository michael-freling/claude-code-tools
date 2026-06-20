#!/bin/bash

set -euo pipefail

# E2E Test Runner Script
# This script runs end-to-end tests that use real git, gh, and claude commands.
# Used by both CI and pre-commit hooks for consistency.

# Configuration
E2E_TIMEOUT="${E2E_TIMEOUT:-1m}"
E2E_VERBOSE="${E2E_VERBOSE:-false}"

echo "=== E2E Test Runner ==="
echo "Timeout: ${E2E_TIMEOUT}"

# Check for required tools
check_tool() {
    local tool=$1
    local required=${2:-false}

    if command -v "${tool}" &> /dev/null; then
        local version
        case "${tool}" in
            git)
                version=$(git --version 2>/dev/null | head -1)
                ;;
            gh)
                version=$(gh --version 2>/dev/null | head -1)
                ;;
            claude)
                version=$(claude --version 2>/dev/null | head -1 || echo "available")
                ;;
            *)
                version="available"
                ;;
        esac
        echo "✓ ${tool}: ${version}"
        return 0
    else
        if [ "${required}" = "true" ]; then
            echo "✗ ${tool}: NOT FOUND (required)"
            return 1
        else
            echo "○ ${tool}: NOT FOUND (optional - some tests will be skipped)"
            return 0
        fi
    fi
}

echo ""
echo "Checking tools..."
check_tool "git" "true" || exit 1
check_tool "gh" "false"
check_tool "claude" "false"

# Check gh authentication status
echo ""
echo "Checking gh authentication..."
if command -v gh &> /dev/null; then
    if gh auth status &> /dev/null; then
        echo "✓ gh: authenticated"
    else
        echo "○ gh: not authenticated (some tests will be skipped)"
    fi
fi

echo ""
echo "Running e2e tests..."

# Build test flags
TEST_FLAGS="-tags=e2e -timeout=${E2E_TIMEOUT}"

if [ "${E2E_VERBOSE}" = "true" ]; then
    TEST_FLAGS="${TEST_FLAGS} -v"
fi

# Run the tests
if go test ${TEST_FLAGS} ./test/e2e/...; then
    echo ""
    echo "=== E2E Tests PASSED ==="
    exit 0
else
    echo ""
    echo "=== E2E Tests FAILED ==="
    exit 1
fi
