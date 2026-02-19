package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolHandler defines the interface for an MCP tool.
// Each tool handler encapsulates its own tool definition and execution logic.
type ToolHandler interface {
	Tool() mcp.Tool
	Handle(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
}
