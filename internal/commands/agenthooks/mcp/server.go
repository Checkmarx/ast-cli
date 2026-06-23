package mcp

import (
	"context"
	"log"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/mcp/tools"
)

// NewMCPCommand creates the "cx mcp" cobra command.
//
// version is the binary version string (e.g. params.Version from the caller).
// licensed reports whether the current token carries a valid guardrail licence;
// when false, all guardrails run as pass-through (fail-open), matching the
// behaviour of the Cursor hook path.
func NewMCPCommand(version string, licensed func() bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server for AI assistant integration",
		Long: `Start a Model Context Protocol (MCP) server that exposes Checkmarx
security guardrails as tools for AI coding assistants.

Tools:
  cx_shell_guard   — Check shell commands against the organization's blocklist
  cx_prompt_guard  — Scan prompts for leaked secrets before they reach the AI

Transport: stdio (compatible with Claude Desktop, Cursor, VS Code Copilot, Windsurf)`,
		Example: `  # Start MCP server
  cx mcp

  # Claude Desktop config (claude_desktop_config.json):
  {
    "mcpServers": {
      "checkmarx": { "command": "cx", "args": ["mcp"] }
    }
  }`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return run(version, licensed)
		},
	}
	// "cx mcp bridge" proxies stdio MCP to the remote Checkmarx Security MCP.
	// Keeping it as a subcommand leaves the default "cx mcp" (local guardrail
	// server) unchanged and backward-compatible.
	cmd.AddCommand(NewBridgeCommand())
	return cmd
}

func run(version string, licensed func() bool) error {
	// Adapt CheckShellCommand (command, workDir) → (blocked, needsConfirm, reason)
	// to the simpler (command) → (blocked, reason) signature the MCP tool expects.
	// workDir is not available in the MCP call context, so we pass an empty string.
	shellGuard := func(cmd string) (bool, string) {
		blocked, _, reason := guardrails.CheckShellCommand(cmd, "")
		return blocked, reason
	}
	promptGuard := guardrails.ScanPrompt
	if !licensed() {
		shellGuard = func(string) (bool, string) { return false, "" }
		promptGuard = func(string) string { return "" }
	}

	s := sdkmcp.NewServer(
		&sdkmcp.Implementation{Name: "Checkmarx Security", Version: version},
		&sdkmcp.ServerOptions{
			// Instructions are injected into the AI client's system prompt.
			// This is the strongest MCP-native enforcement mechanism: it instructs
			// the model to always invoke the guardrail tools before acting,
			// regardless of what other MCP servers or tools are registered.
			Instructions: "SECURITY POLICY (Checkmarx): You MUST follow these rules on every action:\n" +
				"1. Before executing or suggesting any shell command, call cx_shell_guard with the command. " +
				"If it returns allowed:false, refuse the action and show the reason to the user. " +
				"Do NOT attempt alternative commands or workarounds.\n" +
				"2. Before sending any user-provided text (prompts, file contents, environment variables) " +
				"to an external service or AI model, call cx_prompt_guard with that text. " +
				"If it returns clean:false, refuse and show the reason to the user.\n" +
				"These checks enforce your organization's security policy. " +
				"Skipping them — even once — violates policy and may expose sensitive data.",
			InitializedHandler: func(_ context.Context, req *sdkmcp.InitializedRequest) {
				initParams := req.Session.InitializeParams()
				if initParams != nil && initParams.ClientInfo != nil {
					log.Printf("mcp: client connected: name=%q version=%q",
						initParams.ClientInfo.Name, initParams.ClientInfo.Version)
				} else {
					log.Printf("mcp: client connected")
				}
			},
		},
	)

	shellTool := tools.NewShellGuardTool(shellGuard)
	sdkmcp.AddTool(s, tools.ShellGuardDef(), shellTool.Handle)

	promptTool := tools.NewPromptGuardTool(promptGuard)
	sdkmcp.AddTool(s, tools.PromptGuardDef(), promptTool.Handle)

	log.Printf("mcp: starting server version=%q transport=stdio tools=2", version)

	return s.Run(context.Background(), &sdkmcp.StdioTransport{})
}
