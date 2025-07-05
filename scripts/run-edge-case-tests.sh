#!/bin/bash
set -euo pipefail

echo "Running edge case tests..."

# Find and run all edge case test files
echo "Finding edge case test files..."
edge_test_files=$(find . -name "*edge*test*.go" -not -path "./vendor/*" -not -path "./.git/*")

if [ -z "$edge_test_files" ]; then
    echo "No edge case test files found"
    exit 0
fi

echo "Found edge case test files:"
echo "$edge_test_files"

# Run edge case tests with race detection
echo "Running edge case tests with race detection..."
go test -v -race $(echo "$edge_test_files" | xargs dirname | sort -u)

echo "Edge case tests completed successfully"