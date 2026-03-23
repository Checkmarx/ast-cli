package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc"
	agenthooks "github.com/cx-amol-mane/hooks"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// =============================================================================
// Unified logic — written ONCE, runs for all 5 AI agents.
//
// Each function maps to one event type across the full agent matrix:
//
//   cxWhenAgentIdle   → Claude stop / Cursor stop / Windsurf cascade /
//                        Factory Droid stop / Gemini after-agent
//   cxBeforeToolCall  → Claude pre-tool-use / Cursor before-shell+mcp /
//                        Windsurf pre-run+mcp / Droid pre-tool-use /
//                        Gemini before-tool
//   cxAfterFileWrite  → Claude post-tool-use / Cursor after-file-edit /
//                        Windsurf post-write-code / Droid post-tool-use /
//                        Gemini after-file-tool
//   cxBeforePrompt    → Claude user-prompt / Cursor before-submit /
//                        Windsurf pre-user-prompt / Droid user-prompt /
//                        Gemini before-agent
// =============================================================================

// cxWhenAgentIdle is called when any agent finishes its response turn.
// ev.Agent identifies which IDE triggered it (claude / cursor / windsurf / droid / gemini).
func cxWhenAgentIdle(ev agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
	// TODO: post-session audit log, compliance check, scan trigger
	return agenthooks.Resume()
}

// blockedShellPatterns lists shell command patterns that cx will always deny.
var blockedShellPatterns = []string{
	"rm -rf",
	"rm -fr",
	"rmdir /s",
	"del /f /s /q",
	"format ",
	"mkfs.",
	":(){:|:&};:", // fork bomb
	"curl | bash",
	"curl | sh",
	"wget -O- | bash",
	"wget -O- | sh",
}

// cxBeforeToolCall is called before any shell command or MCP tool executes.
// ev.IsShell() — bash/terminal command, ev.Command holds the command string.
// ev.IsMCP()   — MCP tool call, ev.ToolName holds the tool identifier.
func cxBeforeToolCall(ev agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
	if ev.IsShell() {
		lower := strings.ToLower(ev.Command)
		for _, pattern := range blockedShellPatterns {
			if strings.Contains(lower, strings.ToLower(pattern)) {
				return agenthooks.Deny(
					fmt.Sprintf("Blocked by Checkmarx policy: command contains dangerous pattern %q", pattern),
				)
			}
		}
	}
	// TODO: check ev.ToolName against MCP server allow-list
	// TODO: call Checkmarx API for real-time secret scan before execution
	return agenthooks.Allow()
}

// cxAfterFileWrite is called after any file is written or edited by an agent.
// ev.FilePath holds the path of the written file.
// ev.Changes   holds before/after diffs (where available).
func cxAfterFileWrite(ev agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
	// TODO: run Checkmarx secret detection on ev.FilePath
	// TODO: run SAST/SCA quick scan on the modified file
	return agenthooks.AcceptWrite()
}

// cxBeforePrompt is called before a user prompt is sent to any agent.
// ev.Text holds the raw prompt text.
func cxBeforePrompt(ev agenthooks.PromptEvent) agenthooks.PromptVerdict {
	lower := strings.ToLower(ev.Text)
	for _, pattern := range blockedShellPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return agenthooks.RejectPrompt(
				fmt.Sprintf("Blocked by Checkmarx policy: prompt contains dangerous pattern %q", pattern),
			)
		}
	}
	// TODO: scan ev.Text for credentials / secret patterns
	// TODO: enforce prompt policy (e.g. block PII, block internal data leakage)
	return agenthooks.AcceptPrompt()
}

// setupCXHooks registers all four unified handlers.
// Called once per dispatch — cheap, idempotent (map overwrites same values).
func setupCXHooks() {
	agenthooks.WhenAgentIdle(cxWhenAgentIdle)
	agenthooks.BeforeToolCall(cxBeforeToolCall)
	agenthooks.AfterFileWrite(cxAfterFileWrite)
	agenthooks.BeforePrompt(cxBeforePrompt)
}

// =============================================================================
// Cobra routing — 22 hidden commands, zero logic.
//
// These exist only because Cobra sits between os.Args and Dispatch():
//   os.Args = ["cx.exe", "hooks", "claude-pre-tool-use"]
//   os.Args[1] = "hooks"  ← Dispatch() would read this, not the route name
//
// Each command captures its own route name via cmd.Use and calls DispatchRoute,
// which looks up the route registered by setupCXHooks() and invokes it.
// =============================================================================

// HookDispatchCommands returns all 22 hidden route-dispatch subcommands.
// Registered under "cx hooks" — agents invoke: cx hooks <route-name>
func HookDispatchCommands() []*cobra.Command {
	type route struct{ use, short string }

	routes := []route{
		// ── WhenAgentIdle (5 routes) ─────────────────────────────────────────
		{"claude-stop", "Claude agent finished responding"},
		{"cursor-stop", "Cursor agent finished responding"},
		{"windsurf-post-cascade-response", "Windsurf agent finished responding"},
		{"droid-stop", "Factory Droid agent finished responding"},
		{"gemini-after-agent", "Gemini agent finished responding"},

		// ── BeforeToolCall (7 routes) ────────────────────────────────────────
		{"claude-pre-tool-use", "Gate Claude tool/shell execution"},
		{"cursor-before-shell", "Gate Cursor shell execution"},
		{"cursor-before-mcp", "Gate Cursor MCP tool execution"},
		{"windsurf-pre-run-command", "Gate Windsurf shell execution"},
		{"windsurf-pre-mcp-tool-use", "Gate Windsurf MCP tool execution"},
		{"droid-pre-tool-use", "Gate Factory Droid tool execution"},
		{"gemini-before-tool", "Gate Gemini tool execution"},

		// ── AfterFileWrite (5 routes) ────────────────────────────────────────
		{"claude-after-file-write", "React after Claude writes a file"},
		{"cursor-after-file-edit", "React after Cursor edits a file"},
		{"windsurf-post-write-code", "React after Windsurf writes a file"},
		{"droid-after-file-write", "React after Factory Droid writes a file"},
		{"gemini-after-file-tool", "React after Gemini writes a file"},

		// ── BeforePrompt (5 routes) ──────────────────────────────────────────
		{"claude-user-prompt-submit", "Gate Claude prompt submission"},
		{"cursor-before-submit-prompt", "Gate Cursor prompt submission"},
		{"windsurf-pre-user-prompt", "Gate Windsurf prompt submission"},
		{"droid-user-prompt-submit", "Gate Factory Droid prompt submission"},
		{"gemini-before-agent", "Gate Gemini prompt submission"},
	}

	cmds := make([]*cobra.Command, len(routes))
	for i, r := range routes {
		r := r
		cmds[i] = &cobra.Command{
			Use:    r.use,
			Short:  r.short,
			Hidden: true,
			// Override root PersistentPreRunE — CX credential/config loading must
			// never run during hook dispatch: any stdout output corrupts the JSON
			// response the agent reads.
			PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
				return nil
			},
			Run: func(cmd *cobra.Command, args []string) {
				setupCXHooks()                   // register the 4 unified handlers
				agenthooks.DispatchRoute(cmd.Use) // route to the right one
			},
		}
	}
	return cmds
}

// =============================================================================
// Management command — cx hooks agenthooks
// =============================================================================

// NewAgentHooksCommand creates the visible "cx hooks agenthooks" management command.
func NewAgentHooksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agenthooks",
		Short: "Manage AI coding agent hook configuration",
		Long:  "Configure AI coding agent hooks to invoke cx directly. No separate binary needed.",
		Example: heredoc.Doc(`
			$ cx hooks agenthooks install
		`),
	}
	cmd.AddCommand(agentHooksInstallCommand())
	return cmd
}

// =============================================================================
// install
// =============================================================================

func agentHooksInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Write hook configs for all agents pointing to this cx binary",
		Long: heredoc.Doc(`
			Patches the settings files for Claude Code, Cursor, Windsurf Cascade,
			and Factory Droid to invoke "cx hooks <route>" for each hook event.
			cx itself handles all dispatch — no separate binary needed.
		`),
		Example: heredoc.Doc(`
			$ cx hooks agenthooks install
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentHooksInstall()
		},
	}
}

func runAgentHooksInstall() error {
	cxPath, err := os.Executable()
	if err != nil {
		return errors.Wrapf(err, "resolving cx binary path")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrapf(err, "finding home directory")
	}

	agents := []struct {
		name string
		fn   func(string, string) error
	}{
		{"Claude Code", agentHooksInstallClaude},
		{"Cursor", agentHooksInstallCursor},
		{"Windsurf Cascade", agentHooksInstallWindsurf},
		{"Factory Droid", agentHooksInstallDroid},
	}

	for _, a := range agents {
		if err := a.fn(home, cxPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", a.name, err)
		} else {
			fmt.Fprintf(os.Stdout, "✓ %s configured\n", a.name)
		}
	}
	return nil
}

func agentHooksInstallClaude(home, cxPath string) error {
	return agentHooksPatchJSON(filepath.Join(home, ".claude", "settings.json"), func(m map[string]any) {
		h := agentHooksEnsureMap(m, "hooks")
		h["Stop"] = cxHookEntries(cxPath, "claude-stop")
		h["PreToolUse"] = cxHookEntries(cxPath, "claude-pre-tool-use")
		h["PostToolUse"] = cxHookEntries(cxPath, "claude-after-file-write")
		h["UserPromptSubmit"] = cxHookEntries(cxPath, "claude-user-prompt-submit")
	})
}

func agentHooksInstallCursor(home, cxPath string) error {
	return agentHooksPatchJSON(filepath.Join(home, ".cursor", "hooks.json"), func(m map[string]any) {
		m["stop"] = cxCmd(cxPath, "cursor-stop")
		m["beforeShellExecution"] = cxCmd(cxPath, "cursor-before-shell")
		m["beforeMCPExecution"] = cxCmd(cxPath, "cursor-before-mcp")
		m["afterFileEdit"] = cxCmd(cxPath, "cursor-after-file-edit")
		m["beforeSubmitPrompt"] = cxCmd(cxPath, "cursor-before-submit-prompt")
	})
}

func agentHooksInstallWindsurf(home, cxPath string) error {
	return agentHooksPatchJSON(filepath.Join(home, ".codeium", "windsurf", "hooks.json"), func(m map[string]any) {
		m["pre_run_command"] = cxCmd(cxPath, "windsurf-pre-run-command")
		m["pre_mcp_tool_use"] = cxCmd(cxPath, "windsurf-pre-mcp-tool-use")
		m["pre_user_prompt"] = cxCmd(cxPath, "windsurf-pre-user-prompt")
		m["post_write_code"] = cxCmd(cxPath, "windsurf-post-write-code")
		m["post_cascade_response"] = cxCmd(cxPath, "windsurf-post-cascade-response")
	})
}

func agentHooksInstallDroid(home, cxPath string) error {
	return agentHooksPatchJSON(filepath.Join(home, ".factory", "settings.json"), func(m map[string]any) {
		h := agentHooksEnsureMap(m, "hooks")
		h["Stop"] = cxHookEntries(cxPath, "droid-stop")
		h["PreToolUse"] = cxHookEntries(cxPath, "droid-pre-tool-use")
		h["PostToolUse"] = cxHookEntries(cxPath, "droid-after-file-write")
		h["UserPromptSubmit"] = cxHookEntries(cxPath, "droid-user-prompt-submit")
	})
}

// cxCommandString builds the shell command string pointing cx at a given route.
// Quotes the binary path if it contains spaces (e.g. C:\Program Files\cx.exe).
func cxCommandString(cxPath, route string) string {
	p := cxPath
	if strings.Contains(p, " ") {
		p = `"` + p + `"`
	}
	return p + " hooks " + route
}

// cxHookEntries builds a Claude/Droid-style hook entry: [{type, command}]
func cxHookEntries(cxPath, route string) []map[string]any {
	return []map[string]any{
		{"type": "command", "command": cxCommandString(cxPath, route)},
	}
}

// cxCmd builds a Cursor/Windsurf-style hook entry: {command}
func cxCmd(cxPath, route string) map[string]any {
	return map[string]any{"command": cxCommandString(cxPath, route)}
}

func agentHooksPatchJSON(path string, patch func(map[string]any)) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	m := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &m)
	}
	patch(m)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func agentHooksEnsureMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]any); ok {
			return sub
		}
	}
	sub := map[string]any{}
	m[key] = sub
	return sub
}
