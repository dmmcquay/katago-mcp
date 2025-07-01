#!/bin/bash
set -e

echo "🐳 Running E2E tests in Docker..."

# Ensure we're in the project root
cd "$(dirname "$0")/.."

# Check if artifacts exist, download if missing
ARTIFACTS_DIR="docker/katago-artifacts"
if [ ! -f "${ARTIFACTS_DIR}/katago-v1.16.3-eigen-linux-x64.zip" ] || [ ! -f "${ARTIFACTS_DIR}/test-model.bin.gz" ]; then
  echo "📥 KataGo artifacts not found, downloading..."
  ./scripts/download-katago-artifacts.sh
fi

# Build the Docker image
echo "Building E2E test image..."
docker build -f Dockerfile.e2e -t katago-mcp-e2e .

# Run the tests
echo "Running tests..."
docker run --rm katago-mcp-e2e

echo "✅ E2E tests completed successfully"