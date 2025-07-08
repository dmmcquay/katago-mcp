# KataGo MCP Server

An MCP (Model Context Protocol) server that provides KataGo analysis capabilities to AI assistants like Claude.

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

For detailed API documentation including parameters, response formats, and examples, see [API.md](API.md).

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

### Building

```bash
git clone https://github.com/dmmcquay/katago-mcp.git
cd katago-mcp
./build.sh
```

### Configuration

1. Copy the example config:
   ```bash
   cp config.example.json config.json
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

Add to your Claude MCP settings:

```json
{
  "mcpServers": {
    "katago-mcp": {
      "command": "/path/to/katago-mcp"
    }
  }
}
```

## Usage

Once configured, you can ask Claude to:
- Analyze Go positions from SGF files
- Find mistakes in your games
- Evaluate territory control
- Explain why certain moves are good or bad

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

## License

MIT License