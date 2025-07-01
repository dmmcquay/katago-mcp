#!/bin/bash
set -e

# Build the KataGo base image locally
# This script is for local testing. CI will use GitHub Actions.

REGISTRY="ghcr.io"
REPO="dmmcquay/katago-base"
TAG="${1:-v1.14.1}"

echo "🔨 Building KataGo base image..."
docker build \
    --build-arg KATAGO_VERSION="${TAG}" \
    -t "${REGISTRY}/${REPO}:${TAG}" \
    -t "${REGISTRY}/${REPO}:latest" \
    .

echo "✅ Build complete!"
echo ""
echo "To push to registry (requires authentication):"
echo "  docker push ${REGISTRY}/${REPO}:${TAG}"
echo "  docker push ${REGISTRY}/${REPO}:latest"
echo ""
echo "To test locally:"
echo "  docker run --rm ${REGISTRY}/${REPO}:${TAG} katago version"