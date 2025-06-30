# KataGo-MCP Kickstart Guide

## Quick Start

This document helps you create a new KataGo-MCP server from scratch.

## Step 1: Create New Repository

```bash
cd ~/src/github.com/dmmcquay
mkdir katago-mcp
cd katago-mcp
git init
```

## Step 2: Initialize Go Module

```bash
go mod init github.com/dmmcquay/katago-mcp
go get github.com/mark3labs/mcp-go@v0.8.0
```

## Step 3: Core Files to Create

### 1. `cmd/katago-mcp/main.go`
```go
package main

import (
    "context"
    "log"
    "os"
    
    "github.com/dmmcquay/katago-mcp/internal/engine"
    "github.com/dmmcquay/katago-mcp/internal/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    logger := log.New(os.Stderr, "[katago-mcp] ", log.LstdFlags)
    
    // Start KataGo engine
    eng, err := engine.NewKataGoEngine(logger)
    if err != nil {
        logger.Fatalf("Failed to start KataGo: %v", err)
    }
    defer eng.Close()
    
    // Create MCP server
    s := server.NewMCPServer(
        "katago-mcp",
        "0.1.0",
        server.WithLogger(logger),
    )
    
    // Register tools
    handler := mcp.NewHandler(eng, logger)
    handler.RegisterTools(s)
    
    // Start server
    logger.Println("KataGo MCP Server started")
    if err := s.Serve(context.Background()); err != nil {
        logger.Fatalf("Server error: %v", err)
    }
}
```

### 2. `internal/engine/katago.go`
```go
package engine

import (
    "bufio"
    "encoding/json"
    "fmt"
    "log"
    "os/exec"
)

type KataGoEngine struct {
    cmd    *exec.Cmd
    stdin  *bufio.Writer
    stdout *bufio.Scanner
    logger *log.Logger
}

func NewKataGoEngine(logger *log.Logger) (*KataGoEngine, error) {
    // Start KataGo in analysis mode
    cmd := exec.Command("katago", "analysis", 
        "-model", "default",  // Uses default model
        "-config", "analysis.cfg")
    
    // Setup pipes...
    // Return engine
}

func (e *KataGoEngine) Analyze(request AnalysisRequest) (*AnalysisResult, error) {
    // Send JSON request
    // Read JSON response
    // Return parsed result
}
```

### 3. `internal/mcp/handler.go`
```go
package mcp

import (
    "context"
    "github.com/dmmcquay/katago-mcp/internal/engine"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

type Handler struct {
    engine *engine.KataGoEngine
    logger *log.Logger
}

func (h *Handler) RegisterTools(s *server.MCPServer) {
    // Register analyzePosition
    analyzeTool := mcp.NewTool("analyzePosition",
        mcp.WithDescription("Analyze a Go position with KataGo"),
        mcp.WithString("sgf", mcp.Required()),
        mcp.WithNumber("moveNumber", mcp.Required()),
    )
    s.AddTool(analyzeTool, h.handleAnalyzePosition)
    
    // Register findMistakes
    mistakesTool := mcp.NewTool("findMistakes",
        mcp.WithDescription("Find mistakes in a game"),
        mcp.WithString("sgf", mcp.Required()),
        mcp.WithNumber("threshold"),
    )
    s.AddTool(mistakesTool, h.handleFindMistakes)
}
```

## Step 4: Setup KataGo

### Install KataGo
```bash
# macOS
brew install katago

# Ubuntu/Debian
sudo apt install katago

# Or download from https://github.com/lightvector/KataGo/releases
```

### Download Neural Network
```bash
# Create KataGo directory
mkdir -p ~/.katago

# Download a network (example - check for latest)
cd ~/.katago
wget https://media.katagotraining.org/g170/neuralnets/g170-b18c384nbt-s8996141312-d4316597426.bin.gz
```

### Generate Config
```bash
katago genconfig -model ~/.katago/g170-b18c384nbt-s8996141312-d4316597426.bin.gz -output ~/.katago/analysis.cfg
```

## Step 5: Build and Test

### Build
```bash
go build -o katago-mcp ./cmd/katago-mcp
```

### Test Script
Create `test.py`:
```python
import json
import subprocess

# Start server
proc = subprocess.Popen(["./katago-mcp"], 
                       stdin=subprocess.PIPE,
                       stdout=subprocess.PIPE,
                       text=True)

# Initialize
init = {"jsonrpc": "2.0", "method": "initialize", 
        "params": {"protocolVersion": "0.8.0"}, "id": 1}
proc.stdin.write(json.dumps(init) + "\n")
proc.stdin.flush()

# Test analysis
sgf = "(;GM[1]FF[4]SZ[19];B[pd];W[dd];B[pq];W[dp];B[fq])"
req = {
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
        "name": "analyzePosition",
        "arguments": {"sgf": sgf, "moveNumber": 5}
    },
    "id": 2
}
proc.stdin.write(json.dumps(req) + "\n")
proc.stdin.flush()

# Read response
print(proc.stdout.readline())
```

## Step 6: Configuration

### Add to Claude MCP settings
```json
{
  "mcpServers": {
    "katago-mcp": {
      "command": "/path/to/katago-mcp"
    }
  }
}
```

## Key Implementation Notes

1. **SGF Parsing**: Keep it simple - just extract moves
2. **Error Handling**: KataGo can timeout or crash - handle gracefully
3. **Resource Management**: KataGo uses significant CPU/memory
4. **Analysis Depth**: Balance speed vs accuracy (visits parameter)

## Next Steps

1. Start with basic `analyzePosition` tool
2. Add `findMistakes` once working
3. Incrementally add `evaluateTerritory` and `explainMove`
4. Test with real games from OGS

## Useful Resources

- KataGo docs: https://github.com/lightvector/KataGo/blob/master/docs/Analysis_Engine.md
- MCP-Go docs: https://github.com/mark3labs/mcp-go
- Example analysis: https://github.com/lightvector/KataGo/blob/master/docs/Analysis_Engine.md#example-analysis-query

### Notes from Derek
- i want this to mirror how the /Users/dmcquay/src/github.com/dmmcquay/ogs-mcp is set up, please reference how it is done to match the same expectations and flows 
