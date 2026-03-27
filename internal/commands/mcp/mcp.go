package mcp

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/checkmarx/ast-cli/internal/commands/mcp/tools"
	"github.com/checkmarx/ast-cli/internal/params"
)

// NewMCPCommand creates the "cx mcp" cobra command.
//
// The guardrail functions are injected from the commands package where the
// existing implementations live — no duplication, no circular imports.
//
//   - shellGuard: checks a shell command against the org blocklist.
//     Returns (true, reason) if blocked, (false, "") if allowed.
//   - promptGuard: scans text for leaked secrets via 2ms.
//     Returns a reason string if secrets found, "" if clean.
func NewMCPCommand(
	shellGuard func(command string) (blocked bool, reason string),
	promptGuard func(text string) (reason string),
) *cobra.Command {
	return &cobra.Command{
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
			return run(shellGuard, promptGuard)
		},
	}
}

func run(
	shellGuard func(string) (bool, string),
	promptGuard func(string) string,
) error {
	logger := zerolog.New(os.Stderr).With().Timestamp().Str("component", "mcp").Logger()

	s := server.NewMCPServer(
		"Checkmarx Security",
		params.Version,
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithHooks(newHooks(logger)),
	)

	shellTool := tools.NewShellGuardTool(shellGuard, logger)
	s.AddTool(tools.ShellGuardDef(), shellTool.Handle)

	promptTool := tools.NewPromptGuardTool(promptGuard, logger)
	s.AddTool(tools.PromptGuardDef(), promptTool.Handle)

	logger.Info().
		Str("version", params.Version).
		Str("transport", "stdio").
		Int("tools", 2).
		Msg("starting MCP server")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		logger.Info().Msg("shutting down")
		os.Exit(0)
	}()

	return server.ServeStdio(s)
}

func newHooks(logger zerolog.Logger) *server.Hooks {
	h := &server.Hooks{}

	h.AddAfterInitialize(func(_ context.Context, _ any, msg *mcplib.InitializeRequest, _ *mcplib.InitializeResult) {
		logger.Info().
			Str("client", msg.Params.ClientInfo.Name).
			Str("version", msg.Params.ClientInfo.Version).
			Msg("client connected")
	})

	h.AddOnError(func(_ context.Context, _ any, method mcplib.MCPMethod, _ any, err error) {
		logger.Error().Err(err).Str("method", string(method)).Msg("error")
	})

	h.AddAfterCallTool(func(_ context.Context, _ any, msg *mcplib.CallToolRequest, _ any) {
		logger.Info().Str("tool", msg.Params.Name).Msg("tool called")
	})

	return h
}
