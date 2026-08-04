// Package agentprofile centralizes the small agent-specific fragments used in Checkmarx remediation
// guidance. The remediation message itself is the same for every CLI agent this cx binary gates
// (Claude Code, Copilot, Cursor, Gemini, …); only the "how to reconnect the MCP" hint differs, since
// each client re-registers the MCP its own way. No agent is ever told to run a registration script.
package agentprofile

// McpReconnect returns the agent-specific "how" fragment for reconnecting the Checkmarx MCP after it
// becomes unavailable. Callers embed it inside
//
//	reconnect the Checkmarx MCP (<fragment>), then retry
//
// so each fragment is a bare method phrase that must NOT repeat the subject ("the Checkmarx MCP") —
// otherwise the sentence reads "reconnect the Checkmarx MCP (reconnect the Checkmarx MCP …)". CLI
// clients that expose a live MCP command lead with it (e.g. /mcp) and keep a restart fallback; none
// needs a registration script.
func McpReconnect(agent string) string {
	switch agent {
	case "Claude":
		return "run /mcp to reconnect it; if it isn't listed, run /reload-plugins first, then /mcp again, or restart Claude Code"
	case "Copilot":
		return "restart the Copilot CLI, or reconnect it via /mcp"
	case "Cursor":
		return "run /mcp to reconnect it, or restart cursor-agent"
	case "Codex":
		return "restart Codex so it reloads MCP servers from ~/.codex/config.toml"
	case "Gemini":
		return "restart the Gemini CLI"
	case "Windsurf":
		return "via Windsurf's MCP settings, or restart Windsurf"
	case "Droid":
		return "restart Droid"
	default:
		return "via your client's MCP settings, or restart the client"
	}
}
