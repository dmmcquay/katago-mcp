#!/usr/bin/env bash
# Installation script for katago-mcp

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="dmmcquay/katago-mcp"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${HOME}/.katago-mcp"

# Functions
error() {
    echo -e "${RED}Error: $1${NC}" >&2
    exit 1
}

success() {
    echo -e "${GREEN}✓ $1${NC}"
}

info() {
    echo -e "${YELLOW}→ $1${NC}"
}

detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case "$arch" in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) error "Unsupported architecture: $arch" ;;
    esac
    
    case "$os" in
        linux|darwin) ;;
        *) error "Unsupported operating system: $os" ;;
    esac
    
    echo "${os}-${arch}"
}

check_dependencies() {
    info "Checking dependencies..."
    
    if ! command -v katago &> /dev/null; then
        echo -e "${YELLOW}Warning: KataGo not found in PATH${NC}"
        echo "Please install KataGo first:"
        echo "  macOS:  brew install katago"
        echo "  Linux:  sudo apt install katago"
        echo "  Or download from: https://github.com/lightvector/KataGo/releases"
        echo ""
        read -p "Continue anyway? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        success "KataGo found at $(which katago)"
    fi
    
    if ! command -v curl &> /dev/null; then
        error "curl is required but not installed"
    fi
}

get_latest_version() {
    curl -s "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

download_and_install() {
    local version="$1"
    local platform="$2"
    
    info "Downloading katago-mcp ${version} for ${platform}..."
    
    local url="https://github.com/${REPO}/releases/download/${version}/katago-mcp-${platform}.tar.gz"
    local temp_dir=$(mktemp -d)
    
    trap "rm -rf $temp_dir" EXIT
    
    if ! curl -sL "$url" | tar xz -C "$temp_dir"; then
        error "Failed to download or extract katago-mcp"
    fi
    
    info "Installing to ${INSTALL_DIR}..."
    
    if [[ -w "$INSTALL_DIR" ]]; then
        mv "$temp_dir/katago-mcp" "$INSTALL_DIR/"
    else
        info "Requesting sudo access to install to ${INSTALL_DIR}"
        sudo mv "$temp_dir/katago-mcp" "$INSTALL_DIR/"
    fi
    
    chmod +x "${INSTALL_DIR}/katago-mcp"
    success "Installed katago-mcp to ${INSTALL_DIR}/katago-mcp"
}

setup_config() {
    info "Setting up configuration..."
    
    mkdir -p "$CONFIG_DIR"
    
    if [[ ! -f "$CONFIG_DIR/config.json" ]]; then
        cat > "$CONFIG_DIR/config.json" << 'EOF'
{
  "katago": {
    "binaryPath": "katago",
    "modelPath": "",
    "configPath": "",
    "numThreads": 4,
    "maxVisits": 1000,
    "maxTime": 10.0
  },
  "server": {
    "name": "katago-mcp",
    "version": "1.0.0",
    "description": "KataGo analysis server for MCP"
  },
  "logging": {
    "level": "info",
    "prefix": "[katago-mcp] "
  }
}
EOF
        success "Created default config at $CONFIG_DIR/config.json"
    else
        info "Config already exists at $CONFIG_DIR/config.json"
    fi
}

setup_katago() {
    info "Checking KataGo setup..."
    
    local katago_home="${HOME}/.katago"
    
    # Check for models
    if [[ -d "$katago_home" ]] && ls "$katago_home"/*.bin.gz &> /dev/null 2>&1; then
        success "Found KataGo models in $katago_home"
    else
        echo -e "${YELLOW}No KataGo models found${NC}"
        echo "Would you like to download a model? [Y/n] "
        read -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
            mkdir -p "$katago_home"
            info "Downloading KataGo model..."
            curl -L "https://media.katagotraining.org/g170/neuralnets/g170-b18c384nbt-s8996141312-d4316597426.bin.gz" \
                -o "$katago_home/g170-b18c384nbt-s8996141312-d4316597426.bin.gz"
            success "Downloaded model to $katago_home"
            
            # Generate config if katago is available
            if command -v katago &> /dev/null; then
                info "Generating KataGo analysis config..."
                katago genconfig -model "$katago_home/g170-b18c384nbt-s8996141312-d4316597426.bin.gz" \
                    -output "$katago_home/analysis.cfg" 2>/dev/null || true
            fi
        fi
    fi
}

print_claude_config() {
    local config_path
    
    case "$(uname -s)" in
        Darwin)
            config_path="~/Library/Application Support/Claude/claude_desktop_config.json"
            ;;
        Linux)
            config_path="~/.config/Claude/claude_desktop_config.json"
            ;;
        *)
            config_path="%APPDATA%\\Claude\\claude_desktop_config.json"
            ;;
    esac
    
    echo ""
    echo "To use katago-mcp with Claude, add this to your Claude config:"
    echo "Location: $config_path"
    echo ""
    echo '  "katago-mcp": {'
    echo '    "command": "'${INSTALL_DIR}'/katago-mcp",'
    echo '    "env": {'
    echo '      "KATAGO_MCP_CONFIG": "'${CONFIG_DIR}'/config.json"'
    echo '    }'
    echo '  }'
    echo ""
}

main() {
    echo "KataGo MCP Server Installer"
    echo "=========================="
    echo ""
    
    check_dependencies
    
    local platform=$(detect_platform)
    info "Detected platform: $platform"
    
    local version=$(get_latest_version)
    if [[ -z "$version" ]]; then
        error "Could not determine latest version"
    fi
    info "Latest version: $version"
    
    download_and_install "$version" "$platform"
    setup_config
    setup_katago
    
    echo ""
    success "Installation complete!"
    
    # Test the installation
    if "${INSTALL_DIR}/katago-mcp" -version &> /dev/null; then
        success "katago-mcp is working correctly"
    else
        echo -e "${YELLOW}Warning: Could not verify installation${NC}"
    fi
    
    print_claude_config
}

# Run main function
main "$@"