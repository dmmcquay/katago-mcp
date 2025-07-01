#!/bin/bash
set -e

# Script to set up KataGo for e2e testing
# This checks for KataGo installation and downloads a test model if needed

echo "🔍 Checking KataGo installation..."

# Check if katago is installed
if ! command -v katago &> /dev/null; then
    echo "❌ KataGo not found. Please install KataGo first:"
    echo "   macOS: brew install katago"
    echo "   Linux: sudo apt install katago"
    echo "   Or download from: https://github.com/lightvector/KataGo/releases"
    exit 1
fi

echo "✅ KataGo found at: $(which katago)"
katago version

# Create test directory
TEST_DIR="$HOME/.katago-mcp-test"
mkdir -p "$TEST_DIR"

# Check for model
MODEL_PATH="$TEST_DIR/test-model.bin.gz"
if [ ! -f "$MODEL_PATH" ]; then
    echo "📥 Downloading test model..."
    # Download a smaller model for faster testing
    if command -v wget &> /dev/null; then
        wget -O "$MODEL_PATH" "https://katagoarchive.org/g170/neuralnets/g170-b15c192-s1672170752-d466197061.bin.gz"
    else
        curl -L -o "$MODEL_PATH" "https://katagoarchive.org/g170/neuralnets/g170-b15c192-s1672170752-d466197061.bin.gz"
    fi
    echo "✅ Model downloaded to: $MODEL_PATH"
else
    echo "✅ Model already exists at: $MODEL_PATH"
fi

# Generate test config if it doesn't exist
CONFIG_PATH="$TEST_DIR/test-config.cfg"
if [ ! -f "$CONFIG_PATH" ]; then
    echo "⚙️ Generating test configuration..."
    katago genconfig -model "$MODEL_PATH" -output "$CONFIG_PATH" <<EOF
chinese
1
1
1
EOF
    
    # Modify config for faster testing
    sed -i.bak 's/numSearchThreads = .*/numSearchThreads = 1/' "$CONFIG_PATH"
    sed -i.bak 's/maxVisits = .*/maxVisits = 100/' "$CONFIG_PATH"
    sed -i.bak 's/maxTime = .*/maxTime = 1.0/' "$CONFIG_PATH"
    echo "✅ Config generated at: $CONFIG_PATH"
else
    echo "✅ Config already exists at: $CONFIG_PATH"
fi

# Test KataGo with the model
echo "🧪 Testing KataGo setup..."
if echo "quit" | katago gtp -model "$MODEL_PATH" -config "$CONFIG_PATH" > /dev/null 2>&1; then
    echo "✅ KataGo test successful!"
else
    echo "❌ KataGo test failed. Please check your installation."
    exit 1
fi

# Export paths for tests
echo ""
echo "📝 Test environment ready! Export these variables:"
echo "export KATAGO_TEST_MODEL=\"$MODEL_PATH\""
echo "export KATAGO_TEST_CONFIG=\"$CONFIG_PATH\""
echo ""
echo "Or run: source <(./scripts/setup-katago-test.sh | tail -2)"