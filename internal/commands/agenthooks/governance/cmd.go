package governance

import (
	"fmt"
	"os"
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// governanceRoute is a single dispatch route entry with its agent affinity.
type governanceRoute struct {
	Use   string
	Short string
	Agent string // "claude" | "cursor"
}

// claudeRoutes lists all governance hook dispatch routes for Claude Code.
var claudeRoutes = []governanceRoute{
	{"gov-claude-session-start", "Governance session-start handler for Claude Code", "claude"},
	{"gov-claude-pre-tool-use", "Governance pre-tool-use handler for Claude Code", "claude"},
	{"gov-claude-post-tool-use", "Governance post-tool-use handler for Claude Code", "claude"},
	{"gov-claude-prompt-submit", "Governance user-prompt-submit handler for Claude Code", "claude"},
	{"gov-claude-prompt-expansion", "Governance slash-command skill handler for Claude Code", "claude"},
	{"gov-claude-session-end", "Governance session-end handler for Claude Code", "claude"},
}

// cursorRoutes lists all governance hook dispatch routes for Cursor.
var cursorRoutes = []governanceRoute{
	{"gov-cursor-session-start", "Governance session-start handler for Cursor", "cursor"},
	{"gov-cursor-before-shell", "Governance shell gate handler for Cursor", "cursor"},
	{"gov-cursor-before-mcp", "Governance MCP gate handler for Cursor", "cursor"},
	{"gov-cursor-before-submit-prompt", "Governance prompt gate handler for Cursor", "cursor"},
	{"gov-cursor-session-end", "Governance session-end handler for Cursor", "cursor"},
}

// allGovernanceRoutes is the complete set of governance dispatch routes across all agents.
var allGovernanceRoutes = append(claudeRoutes, cursorRoutes...)

// GovernanceDispatchCommands returns hidden Cobra commands for every governance route.
// Each command is invoked by the AI agent as: cx hooks <route-name>
// It loads config from checkmarxcli.yaml, registers governance handlers, and dispatches.
func GovernanceDispatchCommands() []*cobra.Command {
	cmds := make([]*cobra.Command, 0, len(allGovernanceRoutes))
	for _, r := range allGovernanceRoutes {
		r := r
		cmds = append(cmds, &cobra.Command{
			Use:    r.Use,
			Short:  r.Short,
			Hidden: true,
			// Skip the root PersistentPreRunE — any config output would corrupt
			// the JSON verdict the agent reads from stdout.
			PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
			Run: func(cmd *cobra.Command, _ []string) {
				cfg := LoadConfig(agentTypeFromRoute(r))
				switch r.Agent {
				case "cursor":
					RegisterGovernanceCursorHooks(cfg)
				default:
					RegisterGovernanceHooks(cfg)
				}
				dispatchGovernanceRoute(cmd.Use)
			},
		})
	}
	return cmds
}

// NewGovernanceCommand creates the `cx hooks governance` management subcommand tree.
func NewGovernanceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "governance",
		Short: "Manage AI governance hook configuration",
		Long:  "Install and manage governance hooks for AI coding agents. Governance hooks enforce organizational policies, audit tool usage, and stream events to the Checkmarx governance backend.",
		Example: `
  $ cx hooks governance install           # install governance hooks for all supported agents
  $ cx hooks governance install claude    # install governance hooks for Claude Code only
  $ cx hooks governance install cursor    # install governance hooks for Cursor only`,
	}
	cmd.AddCommand(newGovernanceInstallCommand())
	return cmd
}

// NewPolicySyncCommand creates the standalone `cx policy sync` command.
func NewPolicySyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Pull the latest governance policy pack from the server",
		Long:  "Downloads the latest policy-pack.json from the governance server and atomically replaces the local copy. Use this to force a policy refresh without waiting for the next session start.",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg := LoadConfig("manual")
			if cfg.ServerURL == "" {
				return errors.New("governance: no server URL configured — run `cx configure` to set the base URI")
			}
			fmt.Fprintln(os.Stdout, "governance: syncing policy pack from", cfg.ServerURL)
			if err := SyncOnce(cfg.ServerURL, cfg.Token); err != nil {
				return errors.Wrap(err, "governance sync failed")
			}
			pack := Load()
			if pack != nil {
				fmt.Fprintf(os.Stdout, "governance: policy pack v%d synced (%d policies)\n",
					pack.PackVersion, len(pack.Policies))
			} else {
				fmt.Fprintln(os.Stdout, "governance: policy pack synced")
			}
			return nil
		},
	}
}

func newGovernanceInstallCommand() *cobra.Command {
	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Write governance hook config for all supported AI coding agents",
		Long: `Patches the hook configuration for supported AI coding agents so they invoke
governance hooks on every session start, tool call, and prompt submission.

Supported agents: claude, cursor`,
		Example: `
  $ cx hooks governance install           # all agents
  $ cx hooks governance install claude    # Claude Code only
  $ cx hooks governance install cursor    # Cursor only`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runGovernanceInstall("claude", "cursor")
		},
	}
	installCmd.AddCommand(&cobra.Command{
		Use:   "claude",
		Short: "Write governance hook config for Claude Code",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runGovernanceInstall("claude")
		},
	})
	installCmd.AddCommand(&cobra.Command{
		Use:   "cursor",
		Short: "Write governance hook config for Cursor",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runGovernanceInstall("cursor")
		},
	})
	return installCmd
}

// runGovernanceInstall installs governance hooks for the named agents.
func runGovernanceInstall(agentIDs ...string) error {
	cxPath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "resolving cx binary path")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "finding home directory")
	}

	var failed int
	for _, id := range agentIDs {
		var installErr error
		switch id {
		case "claude":
			installErr = InstallGovernanceClaude(home, cxPath)
		case "cursor":
			installErr = InstallGovernanceCursor(home, cxPath)
		default:
			fmt.Fprintf(os.Stderr, "✗ unknown agent %q — supported: claude, cursor\n", id)
			failed++
			continue
		}
		if installErr != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", id, installErr)
			failed++
		} else {
			fmt.Fprintf(os.Stdout, "✓ %s governance hooks configured\n", id)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d agent(s) failed to configure", failed)
	}
	return nil
}

// dispatchGovernanceRoute temporarily sets os.Args[1] to the route name so that
// agenthooks.Dispatch() finds the registered handler.
func dispatchGovernanceRoute(route string) {
	saved := os.Args
	os.Args = []string{saved[0], route}
	agenthooks.Dispatch()
	os.Args = saved
}

// agentTypeFromRoute converts a governance route to the GovernanceConfig AgentType string.
func agentTypeFromRoute(r governanceRoute) string {
	switch r.Agent {
	case "claude":
		return "claude-code"
	case "cursor":
		return "cursor"
	}
	return r.Agent
}

// IsGovernanceRoute returns true when the given route name belongs to the governance package.
func IsGovernanceRoute(name string) bool {
	return strings.HasPrefix(name, "gov-")
}
