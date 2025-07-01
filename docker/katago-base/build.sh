#!/bin/bash
set -e

# Configuration
IMAGE_NAME="katago-base"
IMAGE_TAG="1.15.3-cpu"
REGISTRY="ghcr.io/dmmcquay"  # Change this to your registry

# Parse arguments
PUSH=false
if [[ "$1" == "--push" ]]; then
    PUSH=true
fi

echo "🐳 Building KataGo base image..."
echo "Image: ${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"

# Build the image
docker build -t ${IMAGE_NAME}:${IMAGE_TAG} \
    -t ${IMAGE_NAME}:latest \
    -f Dockerfile \
    .

# Tag for registry
docker tag ${IMAGE_NAME}:${IMAGE_TAG} ${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}
docker tag ${IMAGE_NAME}:latest ${REGISTRY}/${IMAGE_NAME}:latest

echo "✅ Build complete!"

# Push if requested
if [[ "$PUSH" == true ]]; then
    echo "📤 Pushing to registry..."
    docker push ${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}
    docker push ${REGISTRY}/${IMAGE_NAME}:latest
    echo "✅ Push complete!"
else
    echo "ℹ️  To push to registry, run: $0 --push"
fi

echo ""
echo "📝 To use this image in Dockerfile.e2e:"
echo "FROM ${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"