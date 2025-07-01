#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 KataGo MCP E2E Test Runner${NC}"
echo "=================================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed${NC}"
    exit 1
fi

# Check for existing KataGo installation
KATAGO_FOUND=false
if command -v katago &> /dev/null; then
    echo -e "${GREEN}✅ KataGo found at: $(which katago)${NC}"
    KATAGO_FOUND=true
else
    echo -e "${YELLOW}⚠️  KataGo not found in PATH${NC}"
fi

# Check for test environment variables
if [[ -n "$KATAGO_TEST_MODEL" && -n "$KATAGO_TEST_CONFIG" ]]; then
    echo -e "${GREEN}✅ Test environment variables set${NC}"
    echo "   Model: $KATAGO_TEST_MODEL"
    echo "   Config: $KATAGO_TEST_CONFIG"
    
    # Verify files exist
    if [[ ! -f "$KATAGO_TEST_MODEL" ]]; then
        echo -e "${RED}❌ Model file not found: $KATAGO_TEST_MODEL${NC}"
        exit 1
    fi
    
    if [[ ! -f "$KATAGO_TEST_CONFIG" ]]; then
        echo -e "${RED}❌ Config file not found: $KATAGO_TEST_CONFIG${NC}"
        exit 1
    fi
    
elif [[ "$KATAGO_FOUND" == true ]]; then
    echo -e "${YELLOW}⚠️  Test environment variables not set, attempting auto-setup...${NC}"
    
    # Try to find KaTrain installation
    echo "🔍 Looking for existing KaTrain installation..."
    
    KATRAIN_PATHS=(
        "$HOME/venvs/system-venv/lib/python3.12/site-packages/katrain"
        "$HOME/Library/Python/3.9/lib/python/site-packages/katrain"
        "$HOME/.local/lib/python3.*/site-packages/katrain"
        "$HOME/katrain"
    )
    
    KATRAIN_FOUND=false
    for path in "${KATRAIN_PATHS[@]}"; do
        # Handle glob patterns
        for expanded_path in $path; do
            if [[ -d "$expanded_path" ]]; then
                MODEL_PATH="$expanded_path/models/kata1-b18c384nbt-s9996604416-d4316597426.bin.gz"
                CONFIG_PATH="$expanded_path/KataGo/analysis_config.cfg"
                
                if [[ -f "$MODEL_PATH" && -f "$CONFIG_PATH" ]]; then
                    echo -e "${GREEN}✅ Found KaTrain installation at: $expanded_path${NC}"
                    export KATAGO_TEST_MODEL="$MODEL_PATH"
                    export KATAGO_TEST_CONFIG="$CONFIG_PATH"
                    KATRAIN_FOUND=true
                    break 2
                fi
            fi
        done
    done
    
    if [[ "$KATRAIN_FOUND" == false ]]; then
        echo -e "${YELLOW}⚠️  KaTrain not found, will download test model...${NC}"
        
        # Create test directory
        TEST_DIR="$HOME/.katago-mcp-test"
        mkdir -p "$TEST_DIR"
        
        MODEL_PATH="$TEST_DIR/test-model.bin.gz"
        CONFIG_PATH="$TEST_DIR/test-config.cfg"
        
        # Download model if not exists
        if [[ ! -f "$MODEL_PATH" ]]; then
            echo "📥 Downloading test model (this may take a moment)..."
            if command -v wget &> /dev/null; then
                wget -q --show-progress -O "$MODEL_PATH" \
                    "https://media.katagotraining.org/g170/neuralnets/g170-b15c192-s1672170752-d466197061.bin.gz"
            elif command -v curl &> /dev/null; then
                curl -L --progress-bar -o "$MODEL_PATH" \
                    "https://media.katagotraining.org/g170/neuralnets/g170-b15c192-s1672170752-d466197061.bin.gz"
            else
                echo -e "${RED}❌ Neither wget nor curl found. Please install one of them.${NC}"
                exit 1
            fi
            echo -e "${GREEN}✅ Model downloaded${NC}"
        fi
        
        # Generate config if not exists
        if [[ ! -f "$CONFIG_PATH" ]]; then
            echo "⚙️ Generating test configuration..."
            if katago genconfig -model "$MODEL_PATH" -output "$CONFIG_PATH" > /dev/null 2>&1 << 'EOF'; then
chinese
1
1
1
EOF
                # Modify for faster testing
                sed -i.bak 's/numSearchThreads = .*/numSearchThreads = 1/' "$CONFIG_PATH" 2>/dev/null || \
                sed -i 's/numSearchThreads = .*/numSearchThreads = 1/' "$CONFIG_PATH"
                
                sed -i.bak 's/maxVisits = .*/maxVisits = 100/' "$CONFIG_PATH" 2>/dev/null || \
                sed -i 's/maxVisits = .*/maxVisits = 100/' "$CONFIG_PATH"
                
                sed -i.bak 's/maxTime = .*/maxTime = 2.0/' "$CONFIG_PATH" 2>/dev/null || \
                sed -i 's/maxTime = .*/maxTime = 2.0/' "$CONFIG_PATH"
                
                echo -e "${GREEN}✅ Config generated${NC}"
            else
                echo -e "${RED}❌ Failed to generate config${NC}"
                exit 1
            fi
        fi
        
        export KATAGO_TEST_MODEL="$MODEL_PATH"
        export KATAGO_TEST_CONFIG="$CONFIG_PATH"
    fi
    
else
    echo -e "${RED}❌ KataGo not found and test environment not configured${NC}"
    echo ""
    echo "To run E2E tests, you need either:"
    echo "1. Install KataGo and this script will auto-configure"
    echo "   - macOS: brew install katago"
    echo "   - Linux: Download from https://github.com/lightvector/KataGo/releases"
    echo ""
    echo "2. Or manually set test environment:"
    echo "   export KATAGO_TEST_MODEL=/path/to/model.bin.gz"
    echo "   export KATAGO_TEST_CONFIG=/path/to/config.cfg"
    echo ""
    echo "3. Or install KaTrain which includes KataGo and models:"
    echo "   pip install katrain"
    exit 1
fi

# Test KataGo setup
echo ""
echo "🧪 Testing KataGo setup..."

# Test KataGo version instead of running analysis (faster and more reliable)
if katago version > /dev/null 2>&1; then
    echo -e "${GREEN}✅ KataGo test successful${NC}"
    katago version | head -1
else
    echo -e "${RED}❌ KataGo test failed${NC}"
    echo "Please check your KataGo installation"
    exit 1
fi

# Verify model and config files exist
if [[ ! -f "$KATAGO_TEST_MODEL" ]]; then
    echo -e "${RED}❌ Model file not found: $KATAGO_TEST_MODEL${NC}"
    exit 1
fi

if [[ ! -f "$KATAGO_TEST_CONFIG" ]]; then
    echo -e "${RED}❌ Config file not found: $KATAGO_TEST_CONFIG${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Model and config files verified${NC}"

# Run tests
echo ""
echo "🚀 Running E2E tests..."
echo "Model: $KATAGO_TEST_MODEL"
echo "Config: $KATAGO_TEST_CONFIG"
echo ""

# Parse command line arguments
TIMEOUT="60s"
VERBOSE=""
TEST_PATTERN=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE="-v"
            shift
            ;;
        -r|--run)
            TEST_PATTERN="-run $2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -t, --timeout DURATION    Test timeout (default: 60s)"
            echo "  -v, --verbose             Verbose output"
            echo "  -r, --run PATTERN         Run only tests matching pattern"
            echo "  -h, --help                Show this help"
            exit 0
            ;;
        *)
            echo -e "${RED}❌ Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Run the tests
if go test $VERBOSE -tags=e2e ./e2e -timeout "$TIMEOUT" $TEST_PATTERN; then
    echo ""
    echo -e "${GREEN}🎉 All E2E tests passed!${NC}"
else
    echo ""
    echo -e "${RED}❌ E2E tests failed${NC}"
    exit 1
fi