#!/bin/bash
set -e

echo "🐳 Running E2E tests in Docker..."

# Build the Docker image
echo "Building Docker image..."
docker build -f Dockerfile.e2e -t katago-mcp-e2e .

# Run the tests
echo "Running tests..."
docker run --rm katago-mcp-e2e

echo "✅ E2E tests completed successfully"