package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

type ToolsHandler struct {
	// TODO: Add KataGo engine when implemented
}

func NewToolsHandler() *ToolsHandler {
	return &ToolsHandler{}
}

func (h *ToolsHandler) RegisterTools(s *server.MCPServer) {
	// TODO: Register analysis tools when implemented
}

