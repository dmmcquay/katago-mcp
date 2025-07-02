#!/bin/bash

set -e

# Docker build script for production images

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get version information
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "-dirty" || echo "")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Default values
IMAGE_NAME="katago-mcp"
TAG="latest"
REGISTRY=""
PLATFORM="linux/amd64"
PUSH="false"
NO_CACHE="false"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --image|-i)
            IMAGE_NAME="$2"
            shift 2
            ;;
        --tag|-t)
            TAG="$2"
            shift 2
            ;;
        --registry|-r)
            REGISTRY="$2"
            shift 2
            ;;
        --platform|-p)
            PLATFORM="$2"
            shift 2
            ;;
        --push)
            PUSH="true"
            shift
            ;;
        --no-cache)
            NO_CACHE="true"
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --image, -i       Image name (default: katago-mcp)"
            echo "  --tag, -t         Image tag (default: latest)"
            echo "  --registry, -r    Registry prefix (e.g., ghcr.io/user)"
            echo "  --platform, -p    Target platform (default: linux/amd64)"
            echo "  --push            Push image to registry after build"
            echo "  --no-cache        Build without using cache"
            echo "  --help, -h        Show this help message"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Construct full image name
FULL_IMAGE_NAME="$IMAGE_NAME:$TAG"
if [[ -n "$REGISTRY" ]]; then
    FULL_IMAGE_NAME="$REGISTRY/$FULL_IMAGE_NAME"
fi

log_info "Building Docker image: $FULL_IMAGE_NAME"
log_info "Platform: $PLATFORM"
log_info "Git commit: ${GIT_COMMIT}${GIT_DIRTY}"
log_info "Build time: $BUILD_TIME"

# Check if base image exists
BASE_IMAGE="ghcr.io/dmmcquay/katago-base:v1.14.1"
log_info "Checking base image: $BASE_IMAGE"
if ! docker manifest inspect "$BASE_IMAGE" > /dev/null 2>&1; then
    log_warn "Base image $BASE_IMAGE not found locally or remotely"
    log_info "Building base image first..."
    
    # Build base image if it doesn't exist
    cd docker/katago-base
    docker build --platform "$PLATFORM" -t "$BASE_IMAGE" .
    cd ../..
    
    log_info "Base image built successfully"
fi

# Build Docker command
DOCKER_CMD="docker build"
DOCKER_CMD="$DOCKER_CMD --platform $PLATFORM"
DOCKER_CMD="$DOCKER_CMD --build-arg GIT_COMMIT=${GIT_COMMIT}${GIT_DIRTY}"
DOCKER_CMD="$DOCKER_CMD --build-arg BUILD_TIME=$BUILD_TIME"
DOCKER_CMD="$DOCKER_CMD -t $FULL_IMAGE_NAME"

if [[ "$NO_CACHE" == "true" ]]; then
    DOCKER_CMD="$DOCKER_CMD --no-cache"
fi

DOCKER_CMD="$DOCKER_CMD ."

# Execute build
log_info "Executing: $DOCKER_CMD"
eval $DOCKER_CMD

if [[ $? -eq 0 ]]; then
    log_info "Docker image built successfully: $FULL_IMAGE_NAME"
    
    # Show image size
    SIZE=$(docker images "$FULL_IMAGE_NAME" --format "table {{.Size}}" | tail -n +2)
    log_info "Image size: $SIZE"
    
    # Push if requested
    if [[ "$PUSH" == "true" ]]; then
        log_info "Pushing image to registry..."
        docker push "$FULL_IMAGE_NAME"
        if [[ $? -eq 0 ]]; then
            log_info "Image pushed successfully"
        else
            log_error "Failed to push image"
            exit 1
        fi
    fi
    
    # Show usage instructions
    echo ""
    log_info "Image ready! You can run it with:"
    echo "  docker run --rm -p 8080:8080 $FULL_IMAGE_NAME"
    echo ""
    log_info "Or with custom configuration:"
    echo "  docker run --rm -p 8080:8080 \\"
    echo "    -v /path/to/config.json:/app/config/config.json \\"
    echo "    -v /path/to/model.bin.gz:/app/models/model.bin.gz \\"
    echo "    -e KATAGO_MODEL_PATH=/app/models/model.bin.gz \\"
    echo "    $FULL_IMAGE_NAME"
    
else
    log_error "Docker build failed"
    exit 1
fi