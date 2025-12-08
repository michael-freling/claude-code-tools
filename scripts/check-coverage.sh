#!/bin/bash

set -euo pipefail

THRESHOLD=90
COVERAGE_FILE="coverage.txt"

echo "Running tests with coverage..."
if ! go test -coverprofile="${COVERAGE_FILE}" ./...; then
    echo "ERROR: Tests failed"
    exit 1
fi

if [ ! -f "${COVERAGE_FILE}" ]; then
    echo "ERROR: Coverage file ${COVERAGE_FILE} not found"
    exit 1
fi

# Extract total coverage percentage
# The coverage.txt file has lines like:
# mode: set
# github.com/user/repo/pkg/file.go:10.2,12.3 2 1
# And ends with total coverage
# We need to calculate total coverage from all lines

# Use go tool cover to get the coverage percentage
COVERAGE=$(go tool cover -func="${COVERAGE_FILE}" | grep total: | awk '{print $3}' | sed 's/%//')

if [ -z "${COVERAGE}" ]; then
    echo "ERROR: Could not extract coverage percentage"
    exit 1
fi

# Check if bc is available for floating point comparison
if command -v bc &> /dev/null; then
    # Use bc for precise floating point comparison
    PASS=$(echo "${COVERAGE} >= ${THRESHOLD}" | bc -l)
    if [ "${PASS}" -eq 1 ]; then
        echo "PASS: Coverage is ${COVERAGE}% (threshold: ${THRESHOLD}%)"
        exit 0
    else
        echo "FAIL: Coverage is ${COVERAGE}% (threshold: ${THRESHOLD}%)"
        exit 1
    fi
else
    # Fallback to integer comparison (convert to integer by removing decimal point)
    COVERAGE_INT=$(echo "${COVERAGE}" | sed 's/\..*//')
    if [ "${COVERAGE_INT}" -ge "${THRESHOLD}" ]; then
        echo "PASS: Coverage is ${COVERAGE}% (threshold: ${THRESHOLD}%)"
        exit 0
    else
        echo "FAIL: Coverage is ${COVERAGE}% (threshold: ${THRESHOLD}%)"
        exit 1
    fi
fi
