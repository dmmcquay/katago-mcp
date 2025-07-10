#!/bin/bash
set -e

# Script to download KataGo artifacts for Docker builds
# These artifacts are not stored in git but are required for building the e2e test Docker image

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ARTIFACTS_DIR="${SCRIPT_DIR}/../docker/katago-artifacts"

# Artifact URLs - Using v1.14.1 which has a regular binary, not AppImage
KATAGO_VERSION="v1.14.1"
KATAGO_BINARY_URL="https://github.com/lightvector/KataGo/releases/download/${KATAGO_VERSION}/katago-${KATAGO_VERSION}-eigen-linux-x64.zip"
KATAGO_BINARY_FILE="katago-${KATAGO_VERSION}-eigen-linux-x64.zip"

# Neural network model - using a stable 18-block model
MODEL_URL="https://media.katagotraining.org/uploaded/networks/models/kata1/kata1-b18c384nbt-s9996604416-d4316597426.bin.gz"
MODEL_FILE="test-model.bin.gz"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "📥 Downloading KataGo artifacts for Docker builds..."

# Create artifacts directory if it doesn't exist
mkdir -p "$ARTIFACTS_DIR"

# Function to download a file if it doesn't exist
download_if_missing() {
    local url="$1"
    local file="$2"
    local path="${ARTIFACTS_DIR}/${file}"
    
    if [ -f "$path" ]; then
        echo -e "${GREEN}✅ ${file} already exists${NC}"
        return 0
    fi
    
    echo -e "${YELLOW}⬇️  Downloading ${file}...${NC}"
    if curl -L -f -o "$path" "$url" --progress-bar; then
        echo -e "${GREEN}✅ Downloaded ${file}${NC}"
        return 0
    else
        echo -e "${RED}❌ Failed to download ${file}${NC}"
        return 1
    fi
}

# Download KataGo binary
if ! download_if_missing "$KATAGO_BINARY_URL" "$KATAGO_BINARY_FILE"; then
    echo -e "${RED}Failed to download KataGo binary${NC}"
    exit 1
fi

# Extract KataGo binary from zip
if [ -f "${ARTIFACTS_DIR}/${KATAGO_BINARY_FILE}" ]; then
    echo -e "${YELLOW}📦 Extracting KataGo binary...${NC}"
    cd "$ARTIFACTS_DIR"
    
    # Extract from zip
    unzip -o "$KATAGO_BINARY_FILE" katago || {
        echo -e "${RED}Failed to extract katago from zip${NC}"
        exit 1
    }
    
    # Make it executable
    chmod +x katago
    echo -e "${GREEN}✅ KataGo binary extracted and ready${NC}"
    
    cd - >/dev/null
fi

# Download neural network model
if ! download_if_missing "$MODEL_URL" "$MODEL_FILE"; then
    echo -e "${RED}Failed to download neural network model${NC}"
    echo "Trying alternative sources..."
    
    # Try alternative model URLs
    ALT_MODELS=(
        "https://github.com/lightvector/KataGo/releases/download/v1.14.1/g170e-b20c256x2-s5303129600-d1228401921.bin.gz"
        "https://katagoarchive.org/g170/neuralnets/g170-b15c192-s1672170752-d466197061.bin.gz"
    )
    
    success=false
    for alt_url in "${ALT_MODELS[@]}"; do
        echo "Trying: $alt_url"
        if curl -L -f -o "${ARTIFACTS_DIR}/${MODEL_FILE}" "$alt_url" --progress-bar; then
            echo -e "${GREEN}✅ Downloaded model from alternative source${NC}"
            success=true
            break
        fi
    done
    
    if [ "$success" = false ]; then
        echo -e "${RED}❌ Failed to download model from any source${NC}"
        exit 1
    fi
fi

# Verify files exist
echo ""
echo "📦 Verifying artifacts..."
if [ -f "${ARTIFACTS_DIR}/${KATAGO_BINARY_FILE}" ] && [ -f "${ARTIFACTS_DIR}/${MODEL_FILE}" ]; then
    echo -e "${GREEN}✅ All artifacts downloaded successfully!${NC}"
    echo ""
    echo "Artifacts location: ${ARTIFACTS_DIR}"
    ls -lh "${ARTIFACTS_DIR}"/*.{zip,gz} 2>/dev/null || true
else
    echo -e "${RED}❌ Some artifacts are missing${NC}"
    exit 1
fi

echo ""
echo "🐳 You can now build the Docker image with:"
echo "   docker build -f docker/Dockerfile.e2e -t katago-mcp-e2e ."