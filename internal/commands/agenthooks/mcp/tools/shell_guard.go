package tools

import (
	"context"
	"fmt"
	"log"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ShellGuardInput is the typed input for the cx_shell_guard tool.
type ShellGuardInput struct {
	Command string `json:"command" jsonschema:"The shell command to check"`
}

// ShellGuardTool wraps a shell guardrail function as an MCP tool.
type ShellGuardTool struct {
	guard func(string) (bool, string)
}

// NewShellGuardTool returns a ShellGuardTool backed by the provided guard function.
func NewShellGuardTool(guard func(string) (bool, string)) *ShellGuardTool {
	return &ShellGuardTool{guard: guard}
}

func (t *ShellGuardTool) Handle(_ context.Context, _ *sdkmcp.CallToolRequest, args ShellGuardInput) (*sdkmcp.CallToolResult, any, error) {
	if args.Command == "" {
		return nil, nil, fmt.Errorf("command is required")
	}

	log.Printf("cx_shell_guard invoked: command=%q", args.Command)

	blocked, reason := t.guard(args.Command)

	result := map[string]any{
		"command": args.Command,
		"allowed": !blocked,
	}
	if blocked {
		result["reason"] = reason
	}

	return nil, result, nil
}

// ShellGuardDef returns the MCP tool definition for cx_shell_guard.
func ShellGuardDef() *sdkmcp.Tool {
	return &sdkmcp.Tool{
		Name: "cx_shell_guard",
		Description: "REQUIRED: Call this before executing any shell command. " +
			"Checks the command against the organization's security blocklist. " +
			"Returns allowed:true if safe to proceed, or allowed:false with a reason if blocked. " +
			"Never execute a shell command without calling this first.",
	}
}
