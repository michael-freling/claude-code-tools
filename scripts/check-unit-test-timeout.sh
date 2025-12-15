#!/bin/bash

set -euo pipefail

# Unit Test Timeout Check Script
# This script verifies that unit tests complete within a specified timeout.
# Used to catch tests that may be doing real I/O operations instead of mocking.

# Configuration
# Note: The workflow package has tests that intentionally use delays for CI checking behavior,
# so we use a longer timeout (60s) to accommodate those tests.
UNIT_TIMEOUT="${UNIT_TIMEOUT:-60s}"

echo "=== Unit Test Timeout Check ==="
echo "Timeout: ${UNIT_TIMEOUT}"
echo ""

# Run unit tests with timeout
# Exclude e2e and integration tagged tests
echo "Running unit tests with ${UNIT_TIMEOUT} timeout..."

if go test -timeout="${UNIT_TIMEOUT}" ./...; then
    echo ""
    echo "=== Unit Tests PASSED (within ${UNIT_TIMEOUT}) ==="
    exit 0
else
    EXIT_CODE=$?
    echo ""
    if [ "${EXIT_CODE}" -eq 2 ]; then
        echo "=== Unit Tests FAILED: TIMEOUT EXCEEDED ==="
        echo ""
        echo "Some tests took longer than ${UNIT_TIMEOUT}."
        echo "This may indicate tests are doing real I/O instead of mocking."
        echo ""
        echo "To investigate slow tests, run:"
        echo "  go test -v -timeout=30s ./... 2>&1 | grep -E '(PASS|FAIL|---)'  "
        echo ""
        echo "To increase the timeout, set UNIT_TIMEOUT environment variable:"
        echo "  UNIT_TIMEOUT=10s ./scripts/check-unit-test-timeout.sh"
    else
        echo "=== Unit Tests FAILED ==="
    fi
    exit 1
fi
