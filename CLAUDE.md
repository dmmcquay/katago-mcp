# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Development Commands

Build the project:
```bash
./build.sh
```

Run tests:
```bash
go test ./...
go test -race ./...  # Run with race detection
./scripts/run-edge-case-tests.sh  # Run edge case tests
```

Run linting:
```bash
go vet ./...
golangci-lint run  # If installed
```

Format code:
```bash
go fmt ./...
```

## Architecture Overview

This project implements an MCP (Model Context Protocol) server for KataGo, following the same patterns as the ogs-mcp project. The architecture consists of:

### Core Structure
- `cmd/katago-mcp/` - Entry point with version injection
- `internal/` - Private packages organized by concern:
  - `config/` - Configuration management (env vars + JSON)
  - `katago/` - KataGo engine client (analysis mode)
  - `mcp/` - MCP tools and handlers
  - `logging/` - Structured logging to stderr
  - `health/` - Health monitoring
  - `metrics/` - Performance metrics
  - `ratelimit/` - Token bucket rate limiting
  - `retry/` - Exponential backoff retry logic
  - `validation/` - Input validation

### Key Design Patterns

1. **MCP Tool Registration**: Tools are registered in `internal/mcp/tools.go` with centralized rate limiting, metrics, and error handling.

2. **KataGo Integration**: The `internal/katago/` package manages the KataGo subprocess in analysis mode, handling JSON protocol communication.

3. **Configuration**: Environment variables (prefixed with `KATAGO_`) take precedence over JSON config file specified by `KATAGO_MCP_CONFIG`.

4. **Error Handling**: All errors are wrapped with context and logged structurally. MCP errors use proper error codes.

5. **Testing**: Edge cases, security, and concurrency tests are in separate `*_test.go` files.

## MCP Tools

The server implements these KataGo analysis tools:
- `analyzePosition` - Analyze a specific board position
- `findMistakes` - Identify mistakes in a game
- `evaluateTerritory` - Estimate territory control
- `explainMove` - Get explanations for move choices

## Development Guidelines

1. **Follow ogs-mcp patterns**: This project mirrors the structure and patterns from `/Users/dmcquay/src/github.com/dmmcquay/ogs-mcp`.

2. **Version injection**: Use `build.sh` to inject git commit, dirty state, and build time into the binary.

3. **Logging**: All logs go to stderr to keep stdout clean for MCP protocol. Use structured logging with appropriate levels.

4. **Security**: Validate all inputs, especially SGF data. Prevent command injection when spawning KataGo.

5. **Testing**: Write edge case tests for boundary conditions, security tests for input validation, and integration tests for the full flow.

6. **Rate Limiting**: Each tool should respect rate limits to prevent KataGo resource exhaustion.

7. **Documentation**: When creating diagrams or flowcharts, always use Mermaid syntax instead of ASCII art. This applies to architecture diagrams, sequence diagrams, flowcharts, and any other visual representations.

8. **Planning and Phases**: When creating implementation plans or project phases, do not include time measurements (like "Week 1", "2 weeks", etc.). Focus on the logical progression of phases and let the user determine timelines.

9. **Pre-Push Validation**: Always run the PR checks that will happen in CI locally (linter, tests, e2e, security) before pushing a commit/PR. Use `make ci` or individual commands (`make lint`, `make test`, `make build`) to validate your changes locally first. This prevents CI failures and ensures code quality.

## KataGo Setup

KataGo must be installed and configured:
1. Install KataGo binary (via package manager or from releases)
2. Download neural network model to `~/.katago/`
3. Generate analysis config: `katago genconfig -model <model-path> -output ~/.katago/analysis.cfg`
4. The MCP server will spawn KataGo in analysis mode using this config

## Configuration

The server looks for configuration in this order:
1. Environment variables (e.g., `KATAGO_BINARY_PATH`, `KATAGO_MODEL_PATH`)
2. JSON config file at path specified by `KATAGO_MCP_CONFIG`
3. Default values

Key configuration:
- KataGo binary path
- Model file path
- Analysis config path
- Rate limits per tool
- Logging level
- Health check intervals