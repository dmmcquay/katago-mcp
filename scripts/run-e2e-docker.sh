#!/bin/bash
set -e

echo "🐳 Running E2E tests in Docker..."

# Try to use pre-built base image if available
BASE_IMAGE="ubuntu:22.04"
if docker pull ghcr.io/dmmcquay/katago-base:1.15.3-cpu 2>/dev/null; then
  echo "✅ Using pre-built KataGo base image"
  BASE_IMAGE="ghcr.io/dmmcquay/katago-base:1.15.3-cpu"
else
  echo "ℹ️  Pre-built image not available, will build KataGo from scratch"
fi

# Build the Docker image
echo "Building E2E test image..."
docker build -f Dockerfile.e2e \
  --build-arg BASE_IMAGE="${BASE_IMAGE}" \
  -t katago-mcp-e2e .

# Run the tests
echo "Running tests..."
docker run --rm katago-mcp-e2e

echo "✅ E2E tests completed successfully"