# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-01-12

### Added
- Initial release of katago-mcp server
- Core MCP tools for KataGo analysis:
  - `analyzePosition` - Analyze specific board positions with win rates and best moves
  - `findMistakes` - Review complete games to identify mistakes and blunders
  - `evaluateTerritory` - Estimate territory ownership and final score
  - `explainMove` - Get detailed explanations for specific moves
  - `getEngineStatus`, `startEngine`, `stopEngine` - Engine management tools
- Comprehensive configuration system with environment variable overrides
- File logging support with automatic rotation
- Health check endpoint for monitoring
- Rate limiting to prevent resource exhaustion
- In-memory LRU cache for analysis results
- Structured logging with JSON format support
- Docker support for containerized deployment
- Comprehensive test suite including edge cases and security tests
- API documentation and architecture documentation
- Example configuration files for various deployment scenarios
- SGF validation and security hardening against malicious inputs
- Automatic KataGo subprocess management with proper cleanup
- Support for both SGF string and position object inputs
- Configurable analysis parameters (maxVisits, maxTime)
- Proper error handling with MCP-compliant error responses

### Security
- Input validation for all SGF data to prevent command injection
- File path traversal protection
- Rate limiting to prevent DoS attacks
- Secure file permissions (0600) for log files

### Documentation
- Comprehensive API documentation
- Architecture overview
- Setup and installation guides
- Example usage patterns
- Troubleshooting guide

[1.0.0]: https://github.com/dmmcquay/katago-mcp/releases/tag/v1.0.0