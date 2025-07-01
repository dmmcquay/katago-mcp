#!/bin/bash
set -e

# Build and push the base KataGo image
# This should be run manually when we want to update the base image

REGISTRY="ghcr.io"
REPO="dmmcquay/katago-base"
TAG="v1.14.1"

echo "Building KataGo base image..."
docker build -t ${REGISTRY}/${REPO}:${TAG} .

echo "Tagging as latest..."
docker tag ${REGISTRY}/${REPO}:${TAG} ${REGISTRY}/${REPO}:latest

echo "Pushing to registry..."
docker push ${REGISTRY}/${REPO}:${TAG}
docker push ${REGISTRY}/${REPO}:latest

echo "✅ Base image built and pushed:"
echo "  ${REGISTRY}/${REPO}:${TAG}"
echo "  ${REGISTRY}/${REPO}:latest"