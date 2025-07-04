#!/bin/bash
# Test script for local KataGo MCP server

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "Testing KataGo MCP server locally..."

# Check if server is running
check_server() {
    if ! curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${RED}Server not running!${NC}"
        echo "Please start the server first:"
        echo "  KATAGO_MCP_CONFIG=./config.local.json ./katago-mcp"
        exit 1
    fi
}

# Test health endpoints
test_health() {
    echo -e "\n${YELLOW}Testing health endpoints...${NC}"
    
    echo -n "  /health: "
    if curl -sf http://localhost:8080/health > /dev/null; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        return 1
    fi
    
    echo -n "  /ready: "
    if curl -sf http://localhost:8080/ready > /dev/null; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        return 1
    fi
}

# Test metrics endpoint
test_metrics() {
    echo -e "\n${YELLOW}Testing metrics endpoint...${NC}"
    
    echo -n "  /metrics: "
    if curl -sf http://localhost:9090/metrics | grep -q "katago_engine_up"; then
        echo -e "${GREEN}OK${NC}"
        
        # Show some key metrics
        echo "  Key metrics:"
        curl -sf http://localhost:9090/metrics | grep -E "(katago_engine_up|katago_cache_hit_rate|katago_requests_total)" | head -5 | sed 's/^/    /'
    else
        echo -e "${RED}FAIL${NC}"
        return 1
    fi
}

# Test MCP analysis (requires MCP client)
test_mcp_analysis() {
    echo -e "\n${YELLOW}Testing MCP analysis...${NC}"
    
    # This is a placeholder - you would need an actual MCP client to test this
    echo "  Note: MCP analysis testing requires an MCP client"
    echo "  You can use the Claude Desktop app or another MCP-compatible client"
    echo ""
    echo "  Example MCP configuration for Claude Desktop:"
    echo '  {
    "mcpServers": {
      "katago-local": {
        "command": "/path/to/katago-mcp",
        "env": {
          "KATAGO_MCP_CONFIG": "/path/to/config.local.json"
        }
      }
    }
  }'
}

# Show logs
show_logs() {
    echo -e "\n${YELLOW}Recent server logs:${NC}"
    # This assumes the server is logging to stderr
    echo "  (Check your terminal where the server is running)"
}

# Main test flow
main() {
    check_server
    test_health
    test_metrics
    test_mcp_analysis
    show_logs
    
    echo -e "\n${GREEN}Basic tests completed!${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Connect with an MCP client (Claude Desktop, etc.)"
    echo "2. Try analyzing some Go positions"
    echo "3. Monitor the metrics at http://localhost:9090/metrics"
    echo "4. Check cache performance after a few analyses"
}

main