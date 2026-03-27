package tools

import (
	"encoding/json"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// textResult serializes any value to a JSON MCP text response.
func textResult(data interface{}) (*mcplib.CallToolResult, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return &mcplib.CallToolResult{
		Content: []mcplib.Content{
			mcplib.TextContent{Type: "text", Text: string(b)},
		},
	}, nil
}

// errorResult returns an MCP error response with isError=true.
func errorResult(msg string) (*mcplib.CallToolResult, error) {
	return &mcplib.CallToolResult{
		IsError: true,
		Content: []mcplib.Content{
			mcplib.TextContent{Type: "text", Text: msg},
		},
	}, nil
}
