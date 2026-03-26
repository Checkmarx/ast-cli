package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/MakeNowJust/heredoc"
	scanner "github.com/checkmarx/2ms/v3/pkg"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	agenthooks "github.com/cx-amol-mane/hooks"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// =============================================================================
// Guardrail handlers — written once, applied to all 5 AI agents.
// =============================================================================

// cxWhenAgentIdle: agent finished its turn. Nothing to enforce yet.
func cxWhenAgentIdle(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
	return agenthooks.Resume()
}

// =============================================================================
// Shell guardrail — blocks commands that match the organization's blocklist.
// =============================================================================

// shellPolicyPath returns the path to the policy file: ~/.checkmarx/policyhooks1.json
func shellPolicyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".checkmarx", "policyhooks1.json")
}

// shellPolicy is the minimal shape of the policy file we need.
type shellPolicy struct {
	BlocklistTools struct {
		Enabled bool          `json:"enabled"`
		Tools   []blockedTool `json:"tools"`
	} `json:"blocklist_tools"`
}

// blockedTool is a single entry in the policy blocklist.
type blockedTool struct {
	Name     string   `json:"name"`
	OS       []string `json:"os"`
	Category string   `json:"category"`
	Risk     string   `json:"risk"`
}

// denyMessage is the firm instruction appended to every denial. It tells the
// agent to stop — no retries, no workarounds, no alternative approaches.
const denyMessage = "\n\n" +
	"IMPORTANT: This action was blocked by your organization's security policy. " +
	"Do NOT attempt alternative commands, workarounds, or equivalent approaches to achieve the same result. " +
	"Any alternative method to accomplish this blocked action may be equally harmful to the system and the developer. " +
	"Instead, inform the user that this operation is restricted by organizational policy and suggest they contact their administrator if they need this action performed."

// loadBlockedCommands reads the policy file and returns all command names
// (lowercased) that are blocked on the current OS, together with their metadata.
func loadBlockedCommands() map[string]blockedTool {
	blocked := map[string]blockedTool{}

	data, err := os.ReadFile(shellPolicyPath())
	if err != nil {
		return blocked // fail-open: missing policy should not block the developer
	}
	var policy shellPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return blocked // fail-open
	}
	if !policy.BlocklistTools.Enabled {
		return blocked
	}

	currentOS := runtime.GOOS
	for _, t := range policy.BlocklistTools.Tools {
		if !matchesOS(t.OS, currentOS) {
			continue
		}
		blocked[strings.ToLower(t.Name)] = t
	}
	return blocked
}

// matchesOS returns true when any of the tool's OS labels match the current OS.
func matchesOS(toolOS []string, currentOS string) bool {
	for _, o := range toolOS {
		mapped := o
		if o == "mac" {
			mapped = "darwin"
		}
		if mapped == currentOS {
			return true
		}
	}
	return false
}

// cxBeforeToolCall gates shell execution against the organization's blocklist.
// Detection is simple and strong: if any blocked command name appears anywhere
// in the full command string (case-insensitive), the command is denied.
func cxBeforeToolCall(ev agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
	if !ev.IsShell() {
		return agenthooks.Allow()
	}

	blocked := loadBlockedCommands()
	cmdLower := strings.ToLower(ev.Command)

	for name, tool := range blocked {
		if strings.Contains(cmdLower, name) {
			return agenthooks.Deny(fmt.Sprintf(
				"Blocked by Checkmarx: command %q is not allowed.\nCategory: %s\nReason: %s%s",
				name, tool.Category, tool.Risk, denyMessage,
			))
		}
	}
	return agenthooks.Allow()
}

// cxAfterFileWrite: react to file edits. Nothing to enforce yet.
func cxAfterFileWrite(_ agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict {
	return agenthooks.AcceptWrite()
}

// cxBeforePrompt scans the user's prompt for leaked secrets using the 2ms
// engine before it reaches the AI agent. This is the prompt guardrail.
func cxBeforePrompt(ev agenthooks.PromptEvent) agenthooks.PromptVerdict {
	if reason := scanForSecrets(ev.Text); reason != "" {
		return agenthooks.RejectPrompt(reason)
	}
	return agenthooks.AcceptPrompt()
}

// =============================================================================
// Secret scanning — powered by the same 2ms engine used in cx realtime scan.
// =============================================================================

// scanForSecrets runs the 2ms secret scanner on arbitrary text (e.g. a prompt).
// Returns a human-readable rejection reason, or "" when the text is clean.
func scanForSecrets(text string) string {
	content := text
	report, err := scanner.NewScanner().Scan(
		[]scanner.ScanItem{{Content: &content, Source: "prompt"}},
		scanner.ScanConfig{WithValidation: true},
	)
	if err != nil {
		return "" // fail-open: scanner error should not block the developer
	}

	var findings []string
	for _, group := range report.Results {
		for _, secret := range group {
			severity := severityFromValidation(string(secret.ValidationStatus))
			findings = append(findings, fmt.Sprintf("  - %s (severity: %s)", secret.RuleID, severity))
		}
	}
	if len(findings) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"Blocked by Checkmarx: prompt contains %d secret(s):\n%s\nRemove the secrets and try again.",
		len(findings), strings.Join(findings, "\n"),
	)
}

// severityFromValidation maps 2ms validation status to a severity label.
func severityFromValidation(status string) string {
	switch status {
	case "Valid":
		return "Critical"
	case "Invalid":
		return "Medium"
	default: // "Unknown" or anything else
		return "High"
	}
}

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
// Hook registration
// =============================================================================

// registerGuardrails wires the four guardrail handlers.
func registerGuardrails() {
	agenthooks.WhenAgentIdle(cxWhenAgentIdle)
	agenthooks.BeforeToolCall(cxBeforeToolCall)
	agenthooks.AfterFileWrite(cxAfterFileWrite)
	agenthooks.BeforePrompt(cxBeforePrompt)
}

// registerPassThrough wires no-op handlers that always allow the action.
// Used when the license check fails so we still emit valid JSON (fail-open).
func registerPassThrough() {
	agenthooks.WhenAgentIdle(func(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Resume() })
	agenthooks.BeforeToolCall(func(_ agenthooks.ToolCallEvent) agenthooks.ToolVerdict { return agenthooks.Allow() })
	agenthooks.AfterFileWrite(func(_ agenthooks.FileWriteEvent) agenthooks.FileWriteVerdict { return agenthooks.AcceptWrite() })
	agenthooks.BeforePrompt(func(_ agenthooks.PromptEvent) agenthooks.PromptVerdict { return agenthooks.AcceptPrompt() })
}

// =============================================================================
// Cobra routing — 22 hidden subcommands, one per agent×event combination.
//
// Agents invoke:  cx hooks <route-name>
// Each route reads JSON from stdin and writes the verdict as JSON to stdout.
// =============================================================================

func HookDispatchCommands() []*cobra.Command {
	type route struct{ use, short string }

	routes := []route{
		// WhenAgentIdle
		{"claude-stop", "Claude agent finished"},
		{"cursor-stop", "Cursor agent finished"},
		{"windsurf-post-cascade-response", "Windsurf agent finished"},
		{"droid-stop", "Factory Droid agent finished"},
		{"gemini-after-agent", "Gemini agent finished"},

		// BeforeToolCall
		{"claude-pre-tool-use", "Gate Claude tool execution"},
		{"cursor-before-shell", "Gate Cursor shell execution"},
		{"cursor-before-mcp", "Gate Cursor MCP execution"},
		{"windsurf-pre-run-command", "Gate Windsurf shell execution"},
		{"windsurf-pre-mcp-tool-use", "Gate Windsurf MCP execution"},
		{"droid-pre-tool-use", "Gate Droid tool execution"},
		{"gemini-before-tool", "Gate Gemini tool execution"},

		// AfterFileWrite
		{"claude-after-file-write", "React to Claude file write"},
		{"cursor-after-file-edit", "React to Cursor file edit"},
		{"windsurf-post-write-code", "React to Windsurf file write"},
		{"droid-after-file-write", "React to Droid file write"},
		{"gemini-after-file-tool", "React to Gemini file write"},

		// BeforePrompt
		{"claude-user-prompt-submit", "Gate Claude prompt"},
		{"cursor-before-submit-prompt", "Gate Cursor prompt"},
		{"windsurf-pre-user-prompt", "Gate Windsurf prompt"},
		{"droid-user-prompt-submit", "Gate Droid prompt"},
		{"gemini-before-agent", "Gate Gemini prompt"},
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
					registerGuardrails()
				} else {
					registerPassThrough()
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
// Management command — cx hooks agenthooks install
// =============================================================================

func NewAgentHooksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agenthooks",
		Short: "Manage AI coding agent hook configuration",
		Long:  "Configure AI coding agent hooks to invoke cx directly. No separate binary needed.",
		Example: heredoc.Doc(`
			$ cx hooks agenthooks install
		`),
	}
	cmd.AddCommand(agentHooksInstallCmd())
	return cmd
}

func agentHooksInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Write hook configs for all supported agents",
		Long: heredoc.Doc(`
			Patches the settings files for Claude Code, Cursor, Windsurf Cascade,
			and Factory Droid so each agent invokes "cx hooks <route>" on hook events.
		`),
		Example: "  $ cx hooks agenthooks install",
		RunE: func(*cobra.Command, []string) error {
			return runInstall()
		},
	}
}

func runInstall() error {
	cxPath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "resolving cx binary path")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "finding home directory")
	}

	agents := []struct {
		name    string
		install func(home, cx string) error
	}{
		{"Claude Code", installClaude},
		{"Cursor", installCursor},
		{"Windsurf Cascade", installWindsurf},
		{"Factory Droid", installDroid},
	}

	for _, a := range agents {
		if err := a.install(home, cxPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", a.name, err)
		} else {
			fmt.Fprintf(os.Stdout, "✓ %s configured\n", a.name)
		}
	}
	return nil
}

// =============================================================================
// Per-agent install helpers
// =============================================================================

func installClaude(home, cx string) error {
	return patchJSON(filepath.Join(home, ".claude", "settings.json"), func(m map[string]any) {
		h := ensureMap(m, "hooks")
		h["Stop"] = claudeHook(cx, "claude-stop")
		h["PreToolUse"] = claudeHook(cx, "claude-pre-tool-use")
		h["PostToolUse"] = claudeHook(cx, "claude-after-file-write")
		h["UserPromptSubmit"] = claudeHook(cx, "claude-user-prompt-submit")
	})
}

func installCursor(home, cx string) error {
	return patchJSON(filepath.Join(home, ".cursor", "hooks.json"), func(m map[string]any) {
		m["stop"] = cursorHook(cx, "cursor-stop")
		m["beforeShellExecution"] = cursorHook(cx, "cursor-before-shell")
		m["beforeMCPExecution"] = cursorHook(cx, "cursor-before-mcp")
		m["afterFileEdit"] = cursorHook(cx, "cursor-after-file-edit")
		m["beforeSubmitPrompt"] = cursorHook(cx, "cursor-before-submit-prompt")
	})
}

func installWindsurf(home, cx string) error {
	return patchJSON(filepath.Join(home, ".codeium", "windsurf", "hooks.json"), func(m map[string]any) {
		m["pre_run_command"] = cursorHook(cx, "windsurf-pre-run-command")
		m["pre_mcp_tool_use"] = cursorHook(cx, "windsurf-pre-mcp-tool-use")
		m["pre_user_prompt"] = cursorHook(cx, "windsurf-pre-user-prompt")
		m["post_write_code"] = cursorHook(cx, "windsurf-post-write-code")
		m["post_cascade_response"] = cursorHook(cx, "windsurf-post-cascade-response")
	})
}

func installDroid(home, cx string) error {
	return patchJSON(filepath.Join(home, ".factory", "settings.json"), func(m map[string]any) {
		h := ensureMap(m, "hooks")
		h["Stop"] = claudeHook(cx, "droid-stop")
		h["PreToolUse"] = claudeHook(cx, "droid-pre-tool-use")
		h["PostToolUse"] = claudeHook(cx, "droid-after-file-write")
		h["UserPromptSubmit"] = claudeHook(cx, "droid-user-prompt-submit")
	})
}

// =============================================================================
// JSON config helpers
// =============================================================================

// cmdString builds "cx hooks <route>", quoting the path if it has spaces.
func cmdString(cx, route string) string {
	if strings.Contains(cx, " ") {
		cx = `"` + cx + `"`
	}
	return cx + " hooks " + route
}

// claudeHook builds a Claude/Droid hook entry: [{type: "command", command: "..."}]
func claudeHook(cx, route string) []map[string]any {
	return []map[string]any{{"type": "command", "command": cmdString(cx, route)}}
}

// cursorHook builds a Cursor/Windsurf hook entry: {command: "..."}
func cursorHook(cx, route string) map[string]any {
	return map[string]any{"command": cmdString(cx, route)}
}

func patchJSON(path string, patch func(map[string]any)) error {
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

func ensureMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]any); ok {
			return sub
		}
	}
	sub := map[string]any{}
	m[key] = sub
	return sub
}
