# Local Development Setup for KataGo MCP

This guide helps you quickly set up and run the KataGo MCP server locally for testing and development.

## Quick Start

1. **Setup KataGo and download model**:
   ```bash
   ./scripts/setup-local.sh
   ```

2. **Build the server**:
   ```bash
   ./build.sh
   ```

3. **Run the server**:
   ```bash
   KATAGO_MCP_CONFIG=./config.local.json ./katago-mcp
   ```

4. **Test the server** (in another terminal):
   ```bash
   ./scripts/test-local.sh
   ```

## Local Configuration

The `config.local.json` file is configured for local development with:
- Lower resource usage (2 threads, 200 max visits)
- Debug logging in text format
- Small cache (100 items, 10MB)
- 5 minute cache TTL
- Rate limiting at 5 requests/second

## Endpoints

- **Health Check**: http://localhost:8080/health
- **Readiness Check**: http://localhost:8080/ready  
- **Metrics**: http://localhost:9090/metrics

## Using with Claude Desktop

Add this to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "katago-local": {
      "command": "/path/to/katago-mcp",
      "env": {
        "KATAGO_MCP_CONFIG": "/path/to/config.local.json"
      }
    }
  }
}
```

## Monitoring

Watch key metrics in real-time:
```bash
watch -n 1 'curl -s http://localhost:9090/metrics | grep -E "(katago_engine_up|katago_cache_hit_rate|katago_analysis_duration)"'
```

## Troubleshooting

1. **KataGo not found**: Install with `brew install katago` (macOS) or `apt install katago` (Ubuntu)
2. **Model download fails**: Check internet connection and disk space
3. **Server won't start**: Check logs and ensure ports 8080/9090 are free
4. **Analysis timeout**: Increase `maxTime` in config.local.json

## Development Tips

- Set `KATAGO_MCP_LOG_LEVEL=debug` for verbose logging
- Use smaller models for faster testing
- Reduce `maxVisits` for quicker analysis during development
- Monitor cache hit rates to verify caching works