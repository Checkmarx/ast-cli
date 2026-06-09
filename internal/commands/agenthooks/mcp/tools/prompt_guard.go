package tools

import (
	"context"
	"fmt"
	"log"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// PromptGuardInput is the typed input for the cx_prompt_guard tool.
type PromptGuardInput struct {
	Text string `json:"text" jsonschema:"The text to scan for secrets"`
}

// PromptGuardTool wraps a prompt guardrail function as an MCP tool.
type PromptGuardTool struct {
	guard func(string) string
}

// NewPromptGuardTool returns a PromptGuardTool backed by the provided guard function.
func NewPromptGuardTool(guard func(string) string) *PromptGuardTool {
	return &PromptGuardTool{guard: guard}
}

func (t *PromptGuardTool) Handle(_ context.Context, _ *sdkmcp.CallToolRequest, args PromptGuardInput) (*sdkmcp.CallToolResult, any, error) {
	if args.Text == "" {
		return nil, nil, fmt.Errorf("text is required")
	}

	log.Printf("cx_prompt_guard invoked: length=%d", len(args.Text))

	reason := t.guard(args.Text)

	result := map[string]any{
		"clean": reason == "",
	}
	if reason != "" {
		result["blocked"] = true
		result["reason"] = reason
	}

	return nil, result, nil
}

// PromptGuardDef returns the MCP tool definition for cx_prompt_guard.
func PromptGuardDef() *sdkmcp.Tool {
	return &sdkmcp.Tool{
		Name: "cx_prompt_guard",
		Description: "REQUIRED: Call this before sending any user-provided text to an AI model or external service. " +
			"Scans for leaked secrets (API keys, tokens, passwords, certificates) using the Checkmarx 2ms engine " +
			"and custom policy patterns. Returns clean:true if safe to proceed, or clean:false with a reason " +
			"if the text must be blocked. Never skip this check.",
	}
}
