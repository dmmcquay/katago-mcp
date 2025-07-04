#!/bin/bash
# Setup script for local KataGo MCP development

set -e

echo "Setting up KataGo MCP for local development..."

# Check if KataGo is installed
if ! command -v katago &> /dev/null; then
    echo "KataGo not found. Please install KataGo first:"
    echo "  macOS: brew install katago"
    echo "  Ubuntu: sudo apt install katago"
    exit 1
fi

# Create KataGo directory structure
KATAGO_DIR="$HOME/.katago"
mkdir -p "$KATAGO_DIR"

# Download model if not present
MODEL_FILE="$KATAGO_DIR/g170-b30c320x2-s4824661760-d1229536699.bin.gz"
if [ ! -f "$MODEL_FILE" ]; then
    echo "Downloading KataGo model..."
    curl -L -o "$MODEL_FILE" \
        "https://media.katagoarchive.org/g170-b30c320x2-s4824661760-d1229536699.bin.gz"
else
    echo "Model already exists at $MODEL_FILE"
fi

# Generate analysis config if not present
CONFIG_FILE="$KATAGO_DIR/analysis.cfg"
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Generating KataGo analysis configuration..."
    katago genconfig -model "$MODEL_FILE" -output "$CONFIG_FILE"
    
    # Optimize for local development (fewer threads, faster analysis)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS sed syntax
        sed -i '' 's/numSearchThreads = [0-9]*/numSearchThreads = 2/' "$CONFIG_FILE"
        sed -i '' 's/maxPlayouts = [0-9]*/maxPlayouts = 100/' "$CONFIG_FILE"
    else
        # Linux sed syntax
        sed -i 's/numSearchThreads = [0-9]*/numSearchThreads = 2/' "$CONFIG_FILE"
        sed -i 's/maxPlayouts = [0-9]*/maxPlayouts = 100/' "$CONFIG_FILE"
    fi
else
    echo "Config already exists at $CONFIG_FILE"
fi

# Test KataGo setup
echo "Testing KataGo installation..."
echo '{"id":"test","boardXSize":19,"boardYSize":19,"rules":"tromp-taylor","komi":7.5,"moves":[]}' | \
    katago analysis -config "$CONFIG_FILE" -model "$MODEL_FILE" | head -n 5

echo ""
echo "Setup complete! You can now run the KataGo MCP server locally:"
echo ""
echo "  # Build the server"
echo "  ./build.sh"
echo ""
echo "  # Run with local config"
echo "  export KATAGO_MCP_CONFIG=./config.local.json"
echo "  ./katago-mcp"
echo ""
echo "Or run directly:"
echo "  KATAGO_MCP_CONFIG=./config.local.json ./katago-mcp"
echo ""
echo "The server will be available at:"
echo "  - Health: http://localhost:8080/health"
echo "  - Ready:  http://localhost:8080/ready"
echo "  - Metrics: http://localhost:9090/metrics"