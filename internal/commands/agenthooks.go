package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	cxhooks "github.com/checkmarx/ast-cli/internal/commands/agenthooks/cx"
	"github.com/checkmarx/ast-cli/internal/logger"
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
func isLicensed(jwt wrappers.JWTWrapper) bool {
	if err := configuration.LoadConfiguration(); err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("hooks: config load warning - %v", err))
	}
	authErrorLogged := false
	for _, engine := range []string{
		params.CheckmarxOneAssistType,
		params.AIProtectionType,
		params.CheckmarxDevAssistType,
	} {
		ok, err := jwt.IsAllowedEngine(engine)
		if err != nil {
			if !authErrorLogged {
				logger.PrintIfVerbose(fmt.Sprintf("hooks: authentication failed - %v", err))
				authErrorLogged = true
			}
			continue
		}
		if ok {
			logger.PrintIfVerbose(fmt.Sprintf("hooks: AI feature license found (%s)", engine))
			return true
		}
	}
	if authErrorLogged {
		logger.PrintIfVerbose("hooks: running in pass-through mode (not authenticated)")
	} else {
		logger.PrintIfVerbose("hooks: running in pass-through mode (no AI feature license)")
	}
	return false
}

// =============================================================================
// Hook dispatch commands — hidden subcommands for all AI agent hook events.
//
// Agents invoke:  cx hooks <route-name>
// Each route reads JSON from stdin and writes the verdict as JSON to stdout.
// Routes are declared per-agent in cxhooks.Agents (cx package).
// =============================================================================

func HookDispatchCommands(jwt wrappers.JWTWrapper, featureFlags wrappers.FeatureFlagsWrapper, realtimeScanner wrappers.RealtimeScannerWrapper) []*cobra.Command {
	var cmds []*cobra.Command
	for _, agent := range cxhooks.Agents {
		for _, r := range agent.Routes {
			r := r
			cmds = append(cmds, &cobra.Command{
				Use:    r.Use,
				Short:  r.Short,
				Hidden: true,
				// Override root PersistentPreRunE — any stdout from config loading
				// would corrupt the JSON response the agent expects.
				PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
				Run: func(cmd *cobra.Command, _ []string) {
					if isLicensed(jwt) {
						logger.PrintIfVerbose(fmt.Sprintf("hooks: registering security guardrails for %s", cmd.Use))
						cxhooks.RegisterGuardrails(jwt, featureFlags, realtimeScanner)
					} else {
						logger.PrintIfVerbose(fmt.Sprintf("hooks: registering pass-through for %s", cmd.Use))
						cxhooks.RegisterPassThrough()
					}
					cxhooks.DispatchRoute(cmd.Use)
				},
			})
		}
	}
	return cmds
}

// =============================================================================
// Management command — cx hooks agenthooks install [agent]
// =============================================================================

func NewAgentHooksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agenthooks",
		Short: "Manage AI coding agent hook configuration",
		Long:  "Configure AI coding agent hooks to invoke cx directly. Supports " + agentDisplayList() + ".",
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
		`) + agentConfigTable(),
		Example: "  $ cx hooks agenthooks install",
		RunE: func(*cobra.Command, []string) error {
			return runInstall(allAgentIDs()...)
		},
	}

	for _, agent := range cxhooks.Agents {
		agent := agent
		installCmd.AddCommand(&cobra.Command{
			Use:     agent.ID,
			Short:   "Write hook config for " + agent.DisplayName,
			Example: "  $ cx hooks agenthooks install " + agent.ID,
			RunE: func(*cobra.Command, []string) error {
				return runInstall(agent.ID)
			},
		})
	}

	return installCmd
}

// runInstall installs hooks for the given agent IDs. With a single ID it
// returns errors directly (so cobra surfaces them as the command failure);
// with multiple IDs it attempts all agents and aggregates failures.
func runInstall(ids ...string) error {
	cxPath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "resolving cx binary path")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "finding home directory")
	}
	cmdFor := cxhooks.CxCmdFor(cxPath)

	single := len(ids) == 1
	var failed int
	for _, id := range ids {
		agent := cxhooks.FindAgent(id)
		if agent == nil {
			if single {
				return fmt.Errorf("unknown agent %q", id)
			}
			fmt.Fprintf(os.Stderr, "✗ %s: no installer registered\n", id)
			failed++
			continue
		}
		if installErr := agent.Install(home, cmdFor); installErr != nil {
			if single {
				return fmt.Errorf("%s: %w", agent.DisplayName, installErr)
			}
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", agent.DisplayName, installErr)
			failed++
			continue
		}
		fmt.Fprintf(os.Stdout, "✓ %s configured\n", agent.DisplayName)
	}
	if failed > 0 {
		return fmt.Errorf("%d agent(s) failed to configure", failed)
	}
	return nil
}

func allAgentIDs() []string {
	ids := make([]string, len(cxhooks.Agents))
	for i, a := range cxhooks.Agents {
		ids[i] = a.ID
	}
	return ids
}

// agentDisplayList formats agent display names as a natural-language list:
// "A, B, C, and D" (Oxford comma, "and" before the last entry).
func agentDisplayList() string {
	names := make([]string, len(cxhooks.Agents))
	for i, a := range cxhooks.Agents {
		names[i] = a.DisplayName
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
}

// agentConfigTable formats agent ID + config path as an indented two-column
// table for inclusion in command help text.
func agentConfigTable() string {
	var b strings.Builder
	for _, a := range cxhooks.Agents {
		fmt.Fprintf(&b, "  %-8s  %s\n", a.ID, a.ConfigPath)
	}
	return b.String()
}
