package tools

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
)

// ShellGuardTool wraps the existing shell guardrail as an MCP tool.
type ShellGuardTool struct {
	guard  func(string) (bool, string)
	logger zerolog.Logger
}

func NewShellGuardTool(guard func(string) (bool, string), logger zerolog.Logger) *ShellGuardTool {
	return &ShellGuardTool{guard: guard, logger: logger}
}

func (t *ShellGuardTool) Handle(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	command, _ := req.GetArguments()["command"].(string)
	if command == "" {
		return errorResult("command is required")
	}

	t.logger.Info().Str("command", command).Msg("cx_shell_guard invoked")

	blocked, reason := t.guard(command)

	result := map[string]interface{}{
		"command": command,
		"allowed": !blocked,
	}
	if blocked {
		result["reason"] = reason
	}

	return textResult(result)
}

// ShellGuardDef defines the MCP tool schema for cx_shell_guard.
func ShellGuardDef() mcplib.Tool {
	return mcplib.NewTool(
		"cx_shell_guard",
		mcplib.WithDescription(
			"Check a shell command against the organization's security blocklist. "+
				"Returns whether the command is allowed or blocked with a reason. "+
				"Use before executing any shell command to enforce security policy.",
		),
		mcplib.WithString("command", mcplib.Required(), mcplib.Description("The shell command to check")),
	)
}
