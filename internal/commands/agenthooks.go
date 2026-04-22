package commands

import (
	"fmt"
	"os"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
	cxhooks "github.com/CheckmarxDev/ast-cx-hooks/cx"
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// =============================================================================
// License check
// =============================================================================

// isLicensed loads CLI config silently and checks whether the token carries
// a CxOne Assist, AI Protection, or Developer Assist license.
func isLicensed() bool {
	_ = configuration.LoadConfiguration()
	jwt := wrappers.NewJwtWrapper()
	for _, engine := range []string{
		params.CheckmarxOneAssistType,
		params.AIProtectionType,
		params.CheckmarxDevAssistType,
	} {
		if ok, err := jwt.IsAllowedEngine(engine); err == nil && ok {
			return true
		}
	}
	return false
}

// =============================================================================
// Hook dispatch commands — hidden subcommands for all AI agent hook events.
//
// Agents invoke:  cx hooks <route-name>
// Each route reads JSON from stdin and writes the verdict as JSON to stdout.
// =============================================================================

func HookDispatchCommands() []*cobra.Command {
	type route struct{ use, short string }

	routes := []route{
		// Claude Code
		{"claude-stop", "Claude Code agent finished"},
		{"claude-pre-tool-use", "Gate Claude Code tool use"},
		{"claude-after-file-write", "React to Claude Code file write"},
		{"claude-user-prompt-submit", "Gate Claude Code prompt"},

		// Cursor
		{"cursor-stop", "Cursor agent finished"},
		{"cursor-before-shell", "Gate Cursor shell execution"},
		{"cursor-before-mcp", "Gate Cursor MCP execution"},
		{"cursor-after-file-edit", "React to Cursor file edit"},
		{"cursor-before-submit-prompt", "Gate Cursor prompt"},

		// Windsurf
		{"windsurf-pre-run-command", "Gate Windsurf shell execution"},
		{"windsurf-pre-mcp-tool-use", "Gate Windsurf MCP execution"},
		{"windsurf-pre-user-prompt", "Gate Windsurf prompt"},
		{"windsurf-post-write-code", "React to Windsurf file write"},
		{"windsurf-post-cascade-response", "Windsurf agent finished"},

		// Factory Droid
		{"droid-stop", "Factory Droid agent finished"},
		{"droid-pre-tool-use", "Gate Factory Droid tool use"},
		{"droid-after-file-write", "React to Factory Droid file write"},
		{"droid-user-prompt-submit", "Gate Factory Droid prompt"},

		// Gemini CLI
		{"gemini-before-agent", "Gemini CLI agent starting"},
		{"gemini-before-tool", "Gate Gemini CLI tool execution"},
		{"gemini-after-file-tool", "React to Gemini CLI file write"},
		{"gemini-after-agent", "Gemini CLI agent finished"},
	}

	cmds := make([]*cobra.Command, len(routes))
	for i, r := range routes {
		r := r
		cmds[i] = &cobra.Command{
			Use:    r.use,
			Short:  r.short,
			Hidden: true,
			// Override root PersistentPreRunE — any stdout from config loading
			// would corrupt the JSON response the agent expects.
			PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
			Run: func(cmd *cobra.Command, _ []string) {
				if isLicensed() {
					cxhooks.RegisterGuardrails()
				} else {
					cxhooks.RegisterPassThrough()
				}
				// Cobra consumed the route name from os.Args, so Dispatch()
				// would see "hooks" instead. Fix it before dispatching.
				os.Args = []string{os.Args[0], cmd.Use}
				agenthooks.Dispatch()
			},
		}
	}
	return cmds
}

// =============================================================================
// Management command — cx hooks agenthooks install [agent]
// =============================================================================

// agentDisplayNames maps agent IDs to human-readable names for output.
var agentDisplayNames = map[string]string{
	"claude":   "Claude Code",
	"cursor":   "Cursor",
	"windsurf": "Windsurf",
	"droid":    "Factory Droid",
	"gemini":   "Gemini CLI",
}

func NewAgentHooksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agenthooks",
		Short: "Manage AI coding agent hook configuration",
		Long:  "Configure AI coding agent hooks to invoke cx directly. Supports Claude Code, Cursor, Windsurf, Factory Droid, and Gemini CLI.",
		Example: heredoc.Doc(`
			$ cx hooks agenthooks install           # install for all agents
			$ cx hooks agenthooks install cursor    # install for Cursor only
			$ cx hooks agenthooks install claude    # install for Claude Code only
		`),
	}
	cmd.AddCommand(agentHooksInstallCmd())
	return cmd
}

func agentHooksInstallCmd() *cobra.Command {
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Write hook config for all AI coding agents",
		Long: heredoc.Doc(`
			Patches the hook configuration for all supported AI coding agents
			so they invoke "cx hooks <route>" on hook events.

			Supported agents and their config files:
			  claude    ~/.claude/settings.json
			  cursor    ~/.cursor/hooks.json
			  windsurf  ~/.codeium/windsurf/hooks.json
			  droid     ~/.factory/settings.json
			  gemini    ~/.gemini/settings.json
		`),
		Example: "  $ cx hooks agenthooks install",
		RunE: func(*cobra.Command, []string) error {
			return runInstallAll()
		},
	}

	// Per-agent subcommands.
	for _, agent := range []string{"claude", "cursor", "windsurf", "droid", "gemini"} {
		agent := agent
		installCmd.AddCommand(&cobra.Command{
			Use:     agent,
			Short:   "Write hook config for " + agentDisplayNames[agent],
			Example: "  $ cx hooks agenthooks install " + agent,
			RunE: func(*cobra.Command, []string) error {
				return runInstallAgent(agent)
			},
		})
	}

	return installCmd
}

func runInstallAll() error {
	cxPath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "resolving cx binary path")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "finding home directory")
	}

	var failed int
	for _, agent := range []string{"claude", "cursor", "windsurf", "droid", "gemini"} {
		if err := cxhooks.Installers[agent](home, cxPath); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", agentDisplayNames[agent], err)
			failed++
		} else {
			fmt.Fprintf(os.Stdout, "✓ %s configured\n", agentDisplayNames[agent])
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d agent(s) failed to configure", failed)
	}
	return nil
}

func runInstallAgent(agent string) error {
	fn, ok := cxhooks.Installers[agent]
	if !ok {
		return fmt.Errorf("unknown agent %q", agent)
	}
	cxPath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "resolving cx binary path")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "finding home directory")
	}
	if err := fn(home, cxPath); err != nil {
		return fmt.Errorf("%s: %w", agentDisplayNames[agent], err)
	}
	fmt.Fprintf(os.Stdout, "✓ %s configured\n", agentDisplayNames[agent])
	return nil
}
