package agentprofile

import (
	"strings"
	"testing"
)

func TestMcpReconnect_PerAgentPhrase(t *testing.T) {
	// Each recognized agent's fragment must contain a distinctive, correct token.
	cases := map[string]string{
		"Claude":   "restart Claude Code",
		"Copilot":  "Copilot CLI",
		"Cursor":   "cursor-agent",
		"Codex":    "Codex",
		"Gemini":   "Gemini",
		"Windsurf": "Windsurf",
		"Droid":    "Droid",
	}
	for agent, want := range cases {
		if got := McpReconnect(agent); !strings.Contains(got, want) {
			t.Errorf("McpReconnect(%q) = %q, want it to contain %q", agent, got, want)
		}
	}

	// An unrecognized agent gets a neutral, client-agnostic phrase.
	if got := McpReconnect("Aider"); !strings.Contains(got, "your client") {
		t.Errorf("McpReconnect(unknown) = %q, want a neutral phrase", got)
	}

	// The Claude hint must expose the full recovery path: the /mcp reconnect, the /reload-plugins
	// fallback when the server isn't listed, and a restart as last resort.
	claude := McpReconnect("Claude")
	for _, want := range []string{"/mcp", "/reload-plugins", "restart Claude Code"} {
		if !strings.Contains(claude, want) {
			t.Errorf("McpReconnect(%q) = %q, want it to contain %q", "Claude", claude, want)
		}
	}

	// No variant may reference the removed registration script or plugin-root env.
	for _, agent := range []string{"Claude", "Copilot", "Cursor", "Codex"} {
		got := McpReconnect(agent)
		if strings.Contains(got, "cx_mcp_register") || strings.Contains(got, "CLAUDE_PLUGIN_ROOT") {
			t.Errorf("McpReconnect(%q) = %q must not mention the removed script/env", agent, got)
		}
	}

	// Fragments embed as "reconnect the Checkmarx MCP (<fragment>)", so no fragment may repeat the
	// subject "the Checkmarx MCP" (which would read "... (reconnect the Checkmarx MCP …)").
	for _, agent := range []string{"Claude", "Copilot", "Cursor", "Codex", "Gemini", "Windsurf", "Droid", "Aider"} {
		if got := McpReconnect(agent); strings.Contains(strings.ToLower(got), "the checkmarx mcp") {
			t.Errorf("McpReconnect(%q) = %q must not repeat the subject 'the Checkmarx MCP'", agent, got)
		}
	}
}
