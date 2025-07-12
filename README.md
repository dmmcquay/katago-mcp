# KataGo MCP Server

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![MCP Protocol](https://img.shields.io/badge/MCP-Compatible-blue)](https://modelcontextprotocol.io/)

An MCP (Model Context Protocol) server that provides KataGo analysis capabilities to AI assistants like Claude.

katago-mcp enables AI assistants to analyze Go games using the powerful KataGo engine, providing features like position analysis, game review, territory estimation, and move explanations through a simple MCP interface.

## Features

### Core Capabilities
- MCP server implementation for KataGo integration
- Automatic KataGo binary and model detection
- Configuration via environment variables or JSON
- Structured JSON logging with correlation IDs for request tracing
- Graceful shutdown handling

### MCP Tools

#### Core Analysis
- **analyzePosition** - Analyze a specific board position with win rates, score estimates, and best moves
- **getEngineStatus** - Check if the KataGo engine is running
- **startEngine** - Start the KataGo engine manually
- **stopEngine** - Stop the KataGo engine

#### Advanced Analysis
- **findMistakes** - Analyze a complete game to identify mistakes, blunders, and inaccuracies with customizable thresholds
- **evaluateTerritory** - Estimate territory ownership and calculate the final score with visual board representation
- **explainMove** - Get detailed explanations for why a specific move is good or bad, including strategic analysis

For detailed API documentation including parameters, response formats, and examples, see [API.md](docs/API.md).

## Quick Start

```bash
# Install KataGo (macOS)
brew install katago

# Download and build katago-mcp
git clone https://github.com/dmmcquay/katago-mcp.git
cd katago-mcp
./build.sh

# Run with auto-detection
./bin/katago-mcp
```

## Installation

### Prerequisites

1. Install KataGo:
   - macOS: `brew install katago`
   - Linux: `sudo apt install katago` or download from [releases](https://github.com/lightvector/KataGo/releases)
   - Windows: Download from [releases](https://github.com/lightvector/KataGo/releases)

2. Download a neural network:
   ```bash
   mkdir -p ~/.katago
   cd ~/.katago
   wget https://media.katagotraining.org/g170/neuralnets/g170-b18c384nbt-s8996141312-d4316597426.bin.gz
   ```

3. Generate KataGo config:
   ```bash
   katago genconfig -model ~/.katago/g170-b18c384nbt-s8996141312-d4316597426.bin.gz -output ~/.katago/analysis.cfg
   ```

### Building from Source

```bash
git clone https://github.com/dmmcquay/katago-mcp.git
cd katago-mcp
./build.sh
```

The binary will be created at `./bin/katago-mcp`.

### Installing from Release

```bash
# Download the latest release for your platform
curl -L https://github.com/dmmcquay/katago-mcp/releases/latest/download/katago-mcp-$(uname -s)-$(uname -m).tar.gz | tar xz

# Move to a directory in your PATH
sudo mv katago-mcp /usr/local/bin/
```

### Configuration

1. Copy the example config:
   ```bash
   cp config/examples/config.example.json config.json
   ```

2. Edit `config.json` to match your setup, or use environment variables:
   ```bash
   export KATAGO_BINARY_PATH=/path/to/katago
   export KATAGO_MODEL_PATH=/path/to/model.bin.gz
   export KATAGO_CONFIG_PATH=/path/to/analysis.cfg
   ```

3. For production deployments, enable JSON logging:
   ```bash
   export KATAGO_LOG_FORMAT=json  # Default is 'json' for structured logs
   export KATAGO_LOG_FORMAT=text  # Use 'text' for human-readable logs
   ```

### Adding to Claude

Add to your Claude Desktop configuration:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "katago-mcp": {
      "command": "/path/to/katago-mcp",
      "env": {
        "KATAGO_MCP_CONFIG": "/path/to/config.json"
      }
    }
  }
}
```

Or with auto-detection (if KataGo is in PATH):

```json
{
  "mcpServers": {
    "katago-mcp": {
      "command": "/usr/local/bin/katago-mcp"
    }
  }
}
```

## Usage Examples

Once configured, you can ask Claude to:

### Analyze a Position
"Analyze this Go position: (;GM[1]FF[4]SZ[19];B[dd];W[pp];B[dp];W[pd])"

### Review a Game
"Find mistakes in this game and suggest better moves: [paste SGF]"

### Evaluate Territory
"Show me the territory estimate for this position: [paste SGF]"

### Explain Moves
"Why is Q16 a good move in this position? [paste SGF]"

### Common Commands
- "Start the KataGo engine" - Manually start the engine
- "What's the engine status?" - Check if KataGo is running
- "Stop the engine" - Stop KataGo to free resources

## Project Structure

```
katago-mcp/
├── cmd/katago-mcp/     # Main application entry point
├── internal/           # Private packages
├── config/             # Configuration files
│   └── examples/       # Example configurations
├── docker/             # Docker-related files
├── docs/               # Documentation
├── examples/           # Example files
│   └── test-games/     # Sample SGF files for testing
├── scripts/            # Build and utility scripts
└── e2e/                # End-to-end tests
```

## Development

### Running Tests

Run unit tests:
```bash
go test ./...
```

Run with race detection:
```bash
go test -race ./...
```

### End-to-End Testing

The project includes comprehensive e2e tests that run against a real KataGo instance.

#### Setup E2E Test Environment

First, ensure KataGo is installed (see Prerequisites above), then run:

```bash
make setup-e2e
```

This will download a test model and generate a test configuration.

#### Run E2E Tests

```bash
make e2e-test
```

Or manually:
```bash
# Set environment variables
export KATAGO_TEST_MODEL="$HOME/.katago-mcp-test/test-model.bin.gz"
export KATAGO_TEST_CONFIG="$HOME/.katago-mcp-test/test-config.cfg"

# Run tests
go test -tags=e2e ./e2e/... -v
```

The e2e tests cover:
- Position analysis with real KataGo engine
- Game review and mistake detection
- Territory estimation
- Move explanations
- MCP server integration

## Contributing

All changes must be submitted via pull requests and require:
- All CI checks to pass
- No merge conflicts with main branch

### Branch Protection

The `main` branch is protected with the following rules:
- Require a pull request before merging
- Require status checks to pass before merging
- Require branches to be up to date before merging
- Include administrators in these restrictions

## Troubleshooting

### KataGo Not Found
If you get "KataGo binary not found", ensure KataGo is installed and in your PATH:
```bash
which katago  # Should show the path to katago
```

### Model Not Found
The server will auto-detect models in `~/.katago/`. If not found, download one:
```bash
mkdir -p ~/.katago
cd ~/.katago
wget https://media.katagotraining.org/g170/neuralnets/g170-b18c384nbt-s8996141312-d4316597426.bin.gz
```

### Permission Denied
If you get permission errors, ensure the binary is executable:
```bash
chmod +x /path/to/katago-mcp
```

### Debug Logging
Enable debug logging to troubleshoot issues:
```bash
export KATAGO_MCP_LOG_LEVEL=debug
./katago-mcp
```

## License

MIT License - see [LICENSE](LICENSE) file for details.