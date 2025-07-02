# Rate Limiting

The KataGo MCP server includes built-in rate limiting to protect against abuse and ensure fair resource usage.

## Overview

The rate limiting system uses a token bucket algorithm with support for:
- Global rate limits across all clients and tools
- Per-tool rate limits for resource-intensive operations  
- Per-client tracking to prevent single client abuse
- Burst capacity for handling spike traffic

## Configuration

Rate limiting is configured in the `rateLimit` section of the configuration file:

```json
{
  "rateLimit": {
    "enabled": true,
    "requestsPerMin": 60,
    "burstSize": 10,
    "perToolLimits": {
      "analyzePosition": 30,
      "findMistakes": 10,
      "evaluateTerritory": 20,
      "explainMove": 20
    }
  }
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | boolean | true | Enable/disable rate limiting |
| `requestsPerMin` | number | 60 | Global requests per minute limit |
| `burstSize` | number | 10 | Maximum burst capacity |
| `perToolLimits` | object | {} | Per-tool request limits |

## How It Works

### Token Bucket Algorithm

Each rate limit uses a token bucket that:
1. Starts with `burstSize` tokens
2. Refills at `requestsPerMin / 60` tokens per second
3. Consumes 1 token per request
4. Rejects requests when no tokens available

### Rate Limit Hierarchy

Requests must pass all applicable rate limits:

1. **Global Limit**: Applies to all requests
2. **Tool Limit**: Applies to specific tool if configured
3. **Client Limit**: Tracks per-client usage

If any limit is exceeded, the request is rejected.

### Client Tracking

- Clients are identified by:
  - Context value `clientID`
  - Request argument `clientID`
  - Default: "anonymous"
- Client tracking expires after 30 minutes of inactivity
- Each client has their own token buckets

## Response Behavior

When rate limited, the server returns:
- Error message: "rate limit exceeded for tool {toolName}"
- The error is logged with client and tool information
- Metrics track rate limit hits

## Monitoring

Rate limit status is available via:

### Health Check Endpoint

```bash
curl http://localhost:8080/health
```

Returns rate limit status:
```json
{
  "rate_limits": {
    "enabled": true,
    "requestsPerMin": 60,
    "burstSize": 10,
    "activeClients": 5
  }
}
```

### MCP Health Tool

```
health
```

Shows current rate limit configuration and active clients.

## Best Practices

### For Server Operators

1. **Set Appropriate Limits**: Base limits on your hardware capacity
2. **Monitor Usage**: Track rate limit hits in logs/metrics
3. **Adjust Per-Tool Limits**: Resource-intensive tools should have lower limits
4. **Use Burst Capacity**: Allow reasonable bursts for legitimate usage

### For Clients

1. **Implement Backoff**: Retry with exponential backoff on rate limit errors
2. **Batch Operations**: Combine multiple analyses when possible
3. **Cache Results**: Avoid redundant requests for same positions
4. **Provide Client ID**: Help with debugging and monitoring

## Examples

### Disable Rate Limiting

```json
{
  "rateLimit": {
    "enabled": false
  }
}
```

### Strict Limits for Production

```json
{
  "rateLimit": {
    "enabled": true,
    "requestsPerMin": 30,
    "burstSize": 5,
    "perToolLimits": {
      "findMistakes": 5,
      "analyzePosition": 15
    }
  }
}
```

### Development Settings

```json
{
  "rateLimit": {
    "enabled": true,
    "requestsPerMin": 600,
    "burstSize": 100,
    "perToolLimits": {}
  }
}
```

## Troubleshooting

### Constant Rate Limit Errors

1. Check if limits are too low for your usage
2. Verify client isn't making duplicate requests
3. Look for retry loops in client code

### Specific Tool Limited

1. Check per-tool limits in configuration
2. Consider if tool is resource-intensive
3. Increase limit if hardware allows

### Debug Rate Limiting

Enable debug logging to see detailed rate limit decisions:

```bash
KATAGO_MCP_LOG_LEVEL=debug katago-mcp
```

## Implementation Details

The rate limiter is implemented in `internal/ratelimit/` with:
- `TokenBucket`: Core token bucket algorithm
- `Limiter`: Manages multiple buckets and client tracking
- Thread-safe with mutex protection
- Automatic token refilling based on elapsed time
- Client cleanup every 5 minutes for stale entries