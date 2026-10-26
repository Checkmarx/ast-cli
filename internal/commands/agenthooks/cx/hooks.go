package cx

import (
	"log"
	"os"
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/Checkmarx/ast-cx-hooks/claude"
	"github.com/Checkmarx/ast-cx-hooks/cursor"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/asca"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/kics"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/secrets"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/hostenv"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/sca"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/scannerconfig"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/sessiontally"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// canonical per-engine labels for the session summary (normalizes the legacy SCA/Oss split used by
// the per-event remediation telemetry; ASCA already uses "Asca").
const (
	engineAsca    = "Asca"
	engineSca     = "Sca"
	engineKics    = "Kics"
	engineSecrets = "Secrets"
)

// scaScanner is the package-level SCA scanner used by the guardrail handlers.
// It is set by RegisterGuardrails so the handlers (free functions registered
// with the agenthooks library) can reach it without an injection mechanism.
var scaScanner *sca.Scanner

// kicsScanner is the package-level KICS scanner used by the file-edit guardrail.
// It is set by RegisterGuardrails and cleared by RegisterPassThrough.
var kicsScanner *kics.Scanner

// bindThreshold points each guardrail's finding filter at the severity threshold for this event's
// agent, so findings below cx_severity_threshold are left out of a deny and the agent is not told to
// fix them. Agents the switches do not apply to keep every finding.
func bindThreshold(agent agenthooks.AgentID) {
	below := func(severity string) bool { return belowThreshold(agent, severity) }
	secrets.BelowThreshold = below
	asca.BelowThreshold = below
	kics.BelowThreshold = below
	sca.BelowThreshold = below
}

// secretsScanner is the package-level secret-detection scanner used by the file-edit guardrail.
// It is set by RegisterGuardrails and cleared by RegisterPassThrough.
var secretsScanner *secrets.Scanner

var telemetryWrapper wrappers.TelemetryWrapper

func cxWhenAgentIdle(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
	return agenthooks.Resume()
}

// sessionIDFromToolCall recovers the Claude session id for a ToolCall event. Unlike FileEditEvent,
// the library's ToolCallEvent carries no SessionID field; only Claude's raw payload is assertable
// (mirrors promptWorkspaceRoots' Raw type-assertion). Every non-Claude agent falls back to "" (the
// shared default tally bucket), which the Stop handler also reads and clears.
func sessionIDFromToolCall(ev *agenthooks.ToolCallEvent) string {
	if e, ok := ev.Raw.(*claude.PreToolUseEvent); ok {
		return e.SessionID
	}
	return ""
}

// cxBeforeToolCall gates shell execution against the organization's blacklist,
// tool rules, and the SCA guardrail (malicious / vulnerable package installs).
func cxBeforeToolCall(ev agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
	bindThreshold(ev.Agent)
	if !ev.IsShell() {
		return agenthooks.Allow()
	}
	blocked, needsConfirm, reason := guardrails.CheckShellCommand(ev.Command, ev.WorkDir)
	if blocked {
		if needsConfirm {
			return agenthooks.AskUser(reason)
		}
		return agenthooks.Deny(reason)
	}
	if scaScanner != nil && scannerOn(ev.Agent, scannerconfig.SCA) {
		agent := agentToString(ev.Agent)
		sid := sessionIDFromToolCall(&ev)
		if finding, remediation, severity := scaScanner.CheckBashInstall(ev.Command, ev.WorkDir, agent, sid); finding != "" && !belowThreshold(ev.Agent, severity) {
			sessiontally.Add(sid, engineSca, 1, 1)
			logRemediationTelemetry(agent, "SCA", severity, sid)
			return agenthooks.DenyWithContext(finding, remediation)
		}
	}
	return agenthooks.Allow()
}

// cxBeforeFileEdit gates two distinct events the library multiplexes through
// the same handler signature:
//
//  1. File EDITS (Claude / Windsurf / Droid / Gemini) — ev.Changes is populated.
//     Enforce blast_radius_limit, files_limits.max_total_file_size_kb, the ASCA
//     guardrail (AI-introduced code vulnerabilities), the KICS guardrail
//     (IaC security vulnerabilities), the SCA guardrail
//     (malicious / vulnerable manifest additions), and the secret-detection
//     guardrail (hardcoded credentials introduced by the edit) before any bytes
//     are written to disk. MultiEdit and multi-file edits are handled uniformly
//     by iterating ev.Changes.
//
//  2. Cursor file READS (beforeReadFile) — ev.Changes is empty and ev.FilePath
//     points to a file the agent is about to ingest into the LLM context.
//     Cursor's hook payload carries only the path, so we open the file and run
//     the 2ms scanner over its contents. Blocks the read if secrets are found
//     or if the file exceeds the policy size cap. Reads do NOT count toward
//     the blast-radius budget (that limit is about writes).
func cxBeforeFileEdit(ev agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
	if selfHealingOff(ev.Agent, "pre-file-write") {
		return agenthooks.AcceptEdit()
	}
	bindThreshold(ev.Agent)
	if ev.Agent == agenthooks.AgentCursor && len(ev.Changes) == 0 {
		if reason := guardrails.ScanFileForSecrets(ev.FilePath); reason != "" {
			return agenthooks.RejectEdit(reason)
		}
		return agenthooks.AcceptEdit()
	}

	if blocked, reason := guardrails.CheckAndIncrementBlastRadius(); blocked {
		return agenthooks.RejectEdit(reason)
	}
	var totalBytes int64
	for _, diff := range ev.Changes {
		totalBytes += int64(len(diff.After))
	}
	if blocked, reason := guardrails.CheckAndIncrementTotalFileSize(totalBytes); blocked {
		return agenthooks.RejectEdit(reason)
	}
	agent := agentToString(ev.Agent)
	// Each engine below is skipped (not just its verdict ignored) when the developer switched it off in
	// ~/.checkmarx/cx-onboarding.properties; an unreadable config leaves every engine on.
	// cx_severity_threshold (same file): a deny whose highest severity is below the threshold is let
	// through for ASCA, KICS, SCA and secrets. Malicious packages and an unknown severity
	// are never relaxed. A blank or invalid threshold means low (nothing filtered).
	if scannerOn(ev.Agent, scannerconfig.ASCA) {
		if blocked, reason, context, severity := asca.ScanFileEdit(&ev, telemetryWrapper, agent); blocked && !belowThreshold(ev.Agent, severity) {
			sessiontally.Add(ev.SessionID, engineAsca, 1, 1)
			logRemediationTelemetry(agent, "Asca", severity, ev.SessionID)
			return agenthooks.RejectEditWithContext(reason, context)
		}
	}
	var kicsNote string
	if kicsScanner != nil && scannerOn(ev.Agent, scannerconfig.KICS) {
		blocked, reason, context, note, severity := kics.ScanFileEdit(&ev, kicsScanner, telemetryWrapper, agent)
		if blocked && !belowThreshold(ev.Agent, severity) {
			sessiontally.Add(ev.SessionID, engineKics, 1, 1)
			logRemediationTelemetry(agent, "IaC", severity, ev.SessionID)
			return agenthooks.RejectEditWithContext(reason, context)
		}
		kicsNote = note
	}
	if scaScanner != nil && scannerOn(ev.Agent, scannerconfig.SCA) {
		for _, diff := range ev.Changes {
			if finding, remediation, severity := scaScanner.CheckManifestEdit(ev.FilePath, fullAfterContent(ev.FilePath, diff), ev.WorkDir, agent, ev.SessionID); finding != "" && !belowThreshold(ev.Agent, severity) {
				sessiontally.Add(ev.SessionID, engineSca, 1, 1)
				logRemediationTelemetry(agent, "Oss", severity, ev.SessionID)
				return agenthooks.RejectEditWithContext(finding, remediation)
			}
		}
	}
	if secretsScanner != nil && secretsApply(ev.Agent) && scannerOn(ev.Agent, scannerconfig.Secrets) {
		if blocked, reason, context, severity := secrets.ScanFileEdit(&ev, secretsScanner, telemetryWrapper, agent); blocked && !belowThreshold(ev.Agent, severity) {
			sessiontally.Add(ev.SessionID, engineSecrets, 1, 1)
			logRemediationTelemetry(agent, "Secrets", severity, ev.SessionID)
			return agenthooks.RejectEditWithContext(reason, context)
		}
	}
	// A note on an ALLOW, deliberately: blocking every IaC edit because a
	// container engine is down would be worse than an unscanned edit.
	if kicsNote != "" {
		return agenthooks.AllowWithNote(kicsNote)
	}
	return agenthooks.AcceptEdit()
}

// fullAfterContent returns the complete new file content for a diff.
// Write ops set diff.Before to "" and diff.After to the full new content.
// Edit ops set diff.After only to the replacement snippet, so we
// reconstruct by applying the replacement to the current file on disk.
//
// Reconstruction must not depend on the checkout's line-ending style. An
// agent/editor may send the replaced region with LF endings while the file on
// disk uses CRLF (Windows / git core.autocrlf=true), or vice versa. A
// byte-exact strings.Replace then finds no match, returns the file unchanged,
// and any newly added (possibly vulnerable) dependency slips past the scanner
// silently. We therefore try an exact match first, then fall back to a
// line-ending–normalized match. Line endings are irrelevant to manifest
// dependency parsing, so scanning the normalized content is safe.
func fullAfterContent(filePath string, diff agenthooks.FileDiff) []byte {
	if diff.Before == "" {
		return []byte(diff.After)
	}
	current, err := os.ReadFile(filePath)
	if err != nil {
		return []byte(diff.After)
	}
	cur := string(current)

	// 1) Exact replacement (fast path; preserves original bytes).
	if out := strings.Replace(cur, diff.Before, diff.After, 1); out != cur {
		return []byte(out)
	}

	// 2) Line-ending–agnostic replacement. Normalize both the file and the
	// diff region to LF, then replace. This makes reconstruction independent
	// of CRLF vs LF differences between machines and checkouts.
	curN := normalizeNewlines(cur)
	if out := strings.Replace(curN, normalizeNewlines(diff.Before), normalizeNewlines(diff.After), 1); out != curN {
		return []byte(out)
	}

	// 3) Fail-safe: the replaced region could not be located even after
	// normalization. Do not silently accept by returning the unchanged file.
	// Surface the anomaly and fall back to scanning the proposed snippet so a
	// newly added dependency is still given a chance to be detected.
	log.Printf("sca guardrail: could not locate edited region in %q (line-ending or whitespace mismatch); scanning proposed snippet as fallback", filePath)
	return []byte(normalizeNewlines(diff.After))
}

// normalizeNewlines converts CRLF and lone CR line endings to LF.
func normalizeNewlines(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

// cxBeforePrompt runs all prompt guardrails before the prompt reaches the AI agent.
func cxBeforePrompt(ev agenthooks.PromptEvent) agenthooks.PromptVerdict {
	if reason := guardrails.ScanPrompt(ev.Text); reason != "" {
		return agenthooks.RejectPrompt(reason)
	}
	roots := promptWorkspaceRoots(ev.Raw)
	if reason := guardrails.ScanReferencedFiles(ev.Text, roots); reason != "" {
		return agenthooks.RejectPrompt(reason)
	}
	if reason := guardrails.ScanWorkspaceFilesByPromptName(ev.Text, roots); reason != "" {
		return agenthooks.RejectPrompt(reason)
	}
	return agenthooks.AcceptPrompt()
}

// promptWorkspaceRoots returns the anchor(s) for resolving relative file paths
// in the prompt. Cursor sends workspace_roots in its hook payload; when present
// we use them directly. Otherwise (other agents, or missing field) fall back to
// the hook process's CWD.
func promptWorkspaceRoots(raw any) []string {
	if cev, ok := raw.(*cursor.PromptPreEvent); ok && len(cev.WorkspaceRoots) > 0 {
		return cev.WorkspaceRoots
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	return []string{cwd}
}

// RegisterGuardrails wires the four guardrail handlers and instantiates the
// SCA, KICS, and secret-detection scanners used by the Bash and FileEdit handlers.
func RegisterGuardrails(jwt wrappers.JWTWrapper, ff wrappers.FeatureFlagsWrapper, rt wrappers.RealtimeScannerWrapper, tel wrappers.TelemetryWrapper) {
	scaScanner = sca.NewScanner(jwt, ff, rt)
	kicsScanner = kics.NewScanner(jwt, ff)
	secretsScanner = secrets.NewScanner(jwt, ff)
	telemetryWrapper = tel
	agenthooks.WhenAgentIdle(cxWhenAgentIdle)
	agenthooks.BeforeToolCall(cxBeforeToolCall)
	agenthooks.BeforeFileEdit(cxBeforeFileEdit)
	agenthooks.BeforePrompt(cxBeforePrompt)
}

// RegisterPassThrough wires no-op handlers that always allow the action.
// Used when the license check fails so we still emit valid JSON (fail-open).
func RegisterPassThrough() {
	scaScanner = nil
	kicsScanner = nil
	secretsScanner = nil
	agenthooks.WhenAgentIdle(func(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Resume() })
	agenthooks.BeforeToolCall(func(_ agenthooks.ToolCallEvent) agenthooks.ToolVerdict { return agenthooks.Allow() })
	agenthooks.BeforeFileEdit(func(_ agenthooks.FileEditEvent) agenthooks.FileEditVerdict { return agenthooks.AcceptEdit() })
	agenthooks.BeforePrompt(func(_ agenthooks.PromptEvent) agenthooks.PromptVerdict { return agenthooks.AcceptPrompt() })
}

// logRemediationTelemetry sends telemetry when remediation context is delivered to the agent.
func logRemediationTelemetry(agent, engine, severity, sessionID string) {
	if telemetryWrapper == nil {
		return
	}

	telemetryData := &wrappers.DataForAITelemetry{
		AIProvider:       agent,
		Agent:            hostenv.TelemetryAgent(agent),
		Engine:           engine,
		ScanType:         strings.ToLower(engine),
		UniqueID:         wrappers.GetUniqueID(),
		Type:             "hooks-remediate",
		SubType:          "fixWithAIAssist",
		ProblemSeverity:  severity,
		AiAgentSessionId: sessionID,
	}

	if err := telemetryWrapper.SendAIDataToLog(telemetryData); err != nil {
		// fail-open
	}
}

// switchesApply reports whether the ~/.checkmarx/cx-onboarding.properties scanner switches and severity
// threshold are honoured for this agent. Claude, Cursor, Gemini and Codex are excluded for now: they keep
// scanning with ASCA / KICS / SCA and blocking at any severity even when the shared file exists on the
// machine. Every other agent (GitHub Copilot, Droid, Windsurf, ...) honours it. Add an agent to
// the case list below to hold it back, or remove one to roll the feature out to it.
func switchesApply(agent agenthooks.AgentID) bool {
	switch agent {
	case agenthooks.AgentClaude, agenthooks.AgentCursor, agenthooks.AgentGemini, agenthooks.AgentCodex:
		return false
	}
	return true
}

// secretsApply reports whether the secret-detection guardrail runs for this agent. Claude, Cursor, Gemini
// and Codex skip it entirely and keep the ASCA / KICS / SCA guardrails; every other agent runs it. It
// shares switchesApply's agent list so the two cannot drift apart.
func secretsApply(agent agenthooks.AgentID) bool {
	return switchesApply(agent)
}

// scannerOn is scannerconfig.Enabled, limited to the agents the switches apply to. GitHub Copilot inside
// VS Code takes the switches from the VS Code extension's IDE file instead, while that file is valid.
func scannerOn(agent agenthooks.AgentID, s scannerconfig.Scanner) bool {
	if !switchesApply(agent) {
		return true
	}
	if agent == agenthooks.AgentCopilotCLI && hostenv.InVSCode() {
		return scannerconfig.IDEEnabled(s)
	}
	return scannerconfig.Enabled(s)
}

// selfHealingOff reports whether this GitHub Copilot hook runs inside VS Code with the VS Code extension's
// "Checkmarx Self-Healing" setting unticked (a valid IDE file saying cx_self_healing_enabled=false). The
// file-write scans then stand down without telemetry; shell commands and prompts are handled as before.
// The plugin's own hook scripts already stop before running cx in that case; this covers hooks that
// call cx directly. A missing or invalid IDE file means on, and the terminal Copilot CLI never reads it.
func selfHealingOff(agent agenthooks.AgentID, hook string) bool {
	if agent != agenthooks.AgentCopilotCLI || !hostenv.InVSCode() || scannerconfig.SelfHealingEnabled() {
		return false
	}
	logger.PrintfIfVerbose("hooks: copilot-cli %s allowed without scanning host=vscode skip_reason=self_healing_disabled", hook)
	return true
}

// belowThreshold is scannerconfig.BelowThreshold, limited to the agents the switches apply to. GitHub
// Copilot inside VS Code has no threshold setting, so it uses the IDE file's threshold, low when unset.
func belowThreshold(agent agenthooks.AgentID, severity string) bool {
	if !switchesApply(agent) {
		return false
	}
	if agent == agenthooks.AgentCopilotCLI && hostenv.InVSCode() {
		return scannerconfig.IDEBelowThreshold(severity)
	}
	return scannerconfig.BelowThreshold(severity)
}

// agentToString converts agenthooks.AgentID enum to string representation for telemetry.
func agentToString(agent agenthooks.AgentID) string {
	switch agent {
	case agenthooks.AgentClaude:
		return "Claude"
	case agenthooks.AgentCopilot:
		return "Copilot"
	case agenthooks.AgentCopilotCLI:
		return "Copilot"
	case agenthooks.AgentCursor:
		return "Cursor"
	case agenthooks.AgentGemini:
		return "Gemini"
	case agenthooks.AgentDroid:
		return "Droid"
	case agenthooks.AgentWindsurf:
		return "Windsurf"
	case agenthooks.AgentCodex:
		return "Codex"
	default:
		return "Unknown"
	}
}
