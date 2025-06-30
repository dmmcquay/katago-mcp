#!/bin/bash

set -e

# Get version information
GIT_COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "-dirty" || echo "")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags for version injection
LDFLAGS="-X main.GitCommit=${GIT_COMMIT}${GIT_DIRTY} -X main.BuildTime=${BUILD_TIME}"

# Build the binary
echo "Building katago-mcp..."
echo "Git commit: ${GIT_COMMIT}${GIT_DIRTY}"
echo "Build time: ${BUILD_TIME}"

go build -ldflags "$LDFLAGS" -o katago-mcp ./cmd/katago-mcp

echo "Build complete: ./katago-mcp"