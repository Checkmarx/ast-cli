package tools

import (
	"context"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
)

// PromptGuardTool wraps the existing prompt guardrail as an MCP tool.
type PromptGuardTool struct {
	guard  func(string) string
	logger zerolog.Logger
}

func NewPromptGuardTool(guard func(string) string, logger zerolog.Logger) *PromptGuardTool {
	return &PromptGuardTool{guard: guard, logger: logger}
}

func (t *PromptGuardTool) Handle(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	text, _ := req.GetArguments()["text"].(string)
	if text == "" {
		return errorResult("text is required")
	}

	t.logger.Info().Int("length", len(text)).Msg("cx_prompt_guard invoked")

	reason := t.guard(text)

	result := map[string]interface{}{
		"clean": reason == "",
	}
	if reason != "" {
		result["blocked"] = true
		result["reason"] = reason
	}

	return textResult(result)
}

// PromptGuardDef defines the MCP tool schema for cx_prompt_guard.
func PromptGuardDef() mcplib.Tool {
	return mcplib.NewTool(
		"cx_prompt_guard",
		mcplib.WithDescription(
			"Scan text for leaked secrets (API keys, tokens, passwords) using the "+
				"Checkmarx 2ms secret detection engine. Returns whether the text is clean "+
				"or contains secrets that should be removed before sending to an AI agent.",
		),
		mcplib.WithString("text", mcplib.Required(), mcplib.Description("The text to scan for secrets")),
	)
}
