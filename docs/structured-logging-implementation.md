# Structured Logging Implementation Plan

## Overview

This document outlines the implementation of structured JSON logging with correlation IDs for the KataGo MCP server.

## Implementation Summary

We implemented the following components:

### 1. StructuredLogger (`structured.go`)
- JSON output format to stderr
- Correlation ID and Request ID support
- Field-based logging for additional context
- Thread-safe implementation
- Caller information (file:line)

### 2. Logger Interfaces (`interface.go`)
- `LoggerInterface` - Basic logging interface
- `ContextLogger` - Extended interface with context support
- Allows swapping between text and JSON loggers

### 3. LoggerAdapter (`adapter.go`)
- Adapts the existing text logger to work with the new interface
- Provides backward compatibility
- Adds field support to text output

### 4. Factory Functions (`factory.go`)
- `NewLoggerFromConfig()` - Creates appropriate logger based on config
- Supports `KATAGO_LOG_FORMAT` environment variable (text/json)
- Defaults to JSON format for production

### 5. Helper Functions
- `GenerateCorrelationID()` - Creates UUID for request correlation
- `GenerateRequestID()` - Creates short ID for individual requests
- Context helpers for passing IDs through the call chain

## Usage Examples

### Creating a Logger
```go
cfg := &logging.Config{
    Level:   "info",
    Format:  logging.FormatJSON,
    Service: "katago-mcp",
    Version: "1.0.0",
}
logger := logging.NewLoggerFromConfig(cfg)
```

### Logging with Context
```go
ctx := logging.ContextWithCorrelationID(ctx, logging.GenerateCorrelationID())
ctx = logging.ContextWithRequestID(ctx, logging.GenerateRequestID())

logger.WithContext(ctx).Info("Processing request")
```

### Adding Fields
```go
logger.WithFields(map[string]interface{}{
    "user_id": "user-123",
    "action":  "analyze",
    "board_size": 19,
}).Info("Starting analysis")
```

### JSON Output Example
```json
{
  "timestamp": "2025-01-01T12:34:56.789Z",
  "level": "INFO",
  "service": "katago-mcp",
  "version": "1.0.0",
  "message": "Starting analysis",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
  "request_id": "a1b2c3d4e5f6",
  "caller": "internal/mcp/tools.go:123",
  "fields": {
    "user_id": "user-123",
    "action": "analyze",
    "board_size": 19
  }
}
```

## Integration Points

1. **MCP Server** - Add correlation ID generation for each request
2. **KataGo Process** - Log process lifecycle events with structured format
3. **Error Handling** - Include error details in structured fields
4. **Performance Metrics** - Log timing information as fields

## Benefits

1. **Machine-readable** - Easy to parse and index in log aggregation systems
2. **Correlation** - Track requests across the entire system
3. **Searchable** - Query logs by any field
4. **Context-rich** - Include relevant metadata with each log entry
5. **Backward compatible** - Can still use text format if needed

## Next Steps

1. Update all existing logging calls to use the new interface
2. Add correlation ID generation in MCP request handler
3. Configure log aggregation to parse JSON format
4. Add dashboard queries for common patterns
5. Document field conventions for consistency