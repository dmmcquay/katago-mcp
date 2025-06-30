# KataGo MCP Server

An MCP (Model Context Protocol) server that provides KataGo analysis capabilities to AI assistants like Claude.

## Features

- **analyzePosition** - Analyze a specific board position from SGF
- **findMistakes** - Identify mistakes in a game with configurable thresholds
- **evaluateTerritory** - Estimate territory control and ownership
- **explainMove** - Get explanations for move choices and alternatives

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

Run tests:
```bash
go test ./...
```

Run with race detection:
```bash
go test -race ./...
```

## License

MIT License