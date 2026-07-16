package cx

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/Checkmarx/ast-cx-hooks/claude"
	"github.com/Checkmarx/ast-cx-hooks/cursor"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/asca"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/kics"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/sca"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/sessiontally"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// canonical per-engine labels for the session summary (normalizes the legacy SCA/Oss split used by
// the per-event remediation telemetry; ASCA already uses "Asca").
const (
	engineAsca = "Asca"
	engineSca  = "Sca"
	engineKics = "Kics"
)

// markerDirPerm / markerFilePerm are the permissions used when creating the session findings
// marker file's directory and the file itself.
const (
	markerDirPerm  = 0o700
	markerFilePerm = 0o600
)

// sessionFindingsMarker is the path of the per-session marker file that records
// whether at least one real Checkmarx finding was blocked this session.
// cxBeforeFileEdit / cxBeforeToolCall create it on any scanner denial;
// cxWhenAgentIdle consumes it to decide whether to force a summary turn.
func sessionFindingsMarkerPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".checkmarx", ".session-cx-findings")
}

// touchSessionFindingsMarker creates the marker file best-effort.
// A write failure must never affect the scan decision that called it.
func touchSessionFindingsMarker() {
	path := sessionFindingsMarkerPath()
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), markerDirPerm)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, markerFilePerm)
	if err == nil {
		_ = f.Close()
	}
}

// DISABLED (was commit 21d62843): the session summary — both the machine audit log and the
// forced final "Security Scan Session Summary" turn — is turned off because the forced turn
// fired mid-work and disrupted the agent. Kept commented (not deleted) so it can be re-enabled.
//
// sessionSummaryPrompt is injected into Claude as a mandatory final turn when
// the session marker is present at session end (exit code 2 from cxWhenAgentIdle).
/*
const sessionSummaryPrompt = `Checkmarx DevAssist — REQUIRED FINAL OUTPUT

One or more Checkmarx scanner findings were blocked and remediated in this session.
You MUST now output the Security Scan Session Summary before this session ends.

Count from this session's context:
- "Vulnerabilities Found" = number of distinct findings that the Checkmarx hook blocked
  (each blocking message from ASCA, SCA, or any other engine is one finding per rule/package)
- "Remediations Applied" = number of those findings you successfully fixed
  (each completed mcp__Checkmarx__codeRemediation or mcp__Checkmarx__packageRemediation call
   that produced and applied a fix counts as one remediation)

Output the following table, filling in the counts. Omit a row only if that engine had
zero findings AND zero remediations this session. Include the Total row whenever two or
more engine rows are present.

Security Scan Session Summary
═══════════════════════════════════════════════════════════════════
Engine          │ Vulnerabilities Found │ Remediations Applied
────────────────┼──────────────────────┼───────────────────────
ASCA (SAST)     │ ?                    │ ?
SCA (OSS)       │ ?                    │ ?
KICS (IaC)      │ ?                    │ ?
────────────────┼──────────────────────┼───────────────────────
Total           │ ?                    │ ?
═══════════════════════════════════════════════════════════════════

Replace each ? with the actual count. Do not output anything else after this table.
`
*/

// scaScanner is the package-level SCA scanner used by the guardrail handlers.
// It is set by RegisterGuardrails so the handlers (free functions registered
// with the agenthooks library) can reach it without an injection mechanism.
var scaScanner *sca.Scanner

// kicsScanner is the package-level KICS scanner used by the file-edit guardrail.
// It is set by RegisterGuardrails and cleared by RegisterPassThrough.
var kicsScanner *kics.Scanner

var telemetryWrapper wrappers.TelemetryWrapper

// cxWhenAgentIdle: fires at session end (claude-stop).
// If the session findings marker exists, at least one real Checkmarx finding was
// blocked this session. Consume the marker and return Interrupt(sessionSummaryPrompt),
// which serialises to {"decision":"block","reason":"..."} via the agenthooks library
// and exits 0. The harness injects the reason as feedback that forces one final turn
// where Claude outputs the summary, then the stop hook fires again with no marker
// and returns Resume() — clean stop.
//
//nolint:gocritic // summary disabled; body kept commented for easy re-enable
func cxWhenAgentIdle(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
	// DISABLED (was commit 21d62843): both the machine summary (tally Load/Clear + emitSessionSummary)
	// and the human summary (findings-marker → Interrupt(sessionSummaryPrompt)) are turned off. The
	// forced final summary turn fired mid-work and disrupted the agent, so the Stop hook now always
	// Resumes. Kept commented (not deleted) so the feature can be re-enabled.
	/*
		// A repeat/looping idle fire (Claude's stop_hook_active, etc.) must never emit telemetry again or
		// re-interrupt: the library exposes IsLooping() precisely to break Stop-hook loops. Guarding on it
		// makes emit-once and interrupt-once independent of whether the first fire's marker/tally removal
		// succeeded — a failed os.Remove can no longer cause a double-emit or an infinite continuation
		// loop. Emit + interrupt happen only on the first (non-looping) fire.
		if ev.IsLooping() {
			return agenthooks.Resume()
		}

		// (1) Machine telemetry: fold this session's per-engine tally, clear it (so the second Stop fire
		// — the one after the forced summary turn — finds nothing and does not double-emit), then emit
		// under a hard time budget. Load/Clear happen BEFORE emit so the verdict is never hostage to a
		// slow telemetry backend.
		sid := ev.SessionID
		tally := sessiontally.Load(sid)
		sessiontally.Clear(sid)
		emitSessionSummary(ev, tally)

		// (2) Human summary: unchanged — if a real finding was blocked this session, force one final turn
		// where the agent prints the summary table, then the next Stop fire finds no marker → Resume.
		markerPath := sessionFindingsMarkerPath()
		if markerPath != "" {
			if _, err := os.Stat(markerPath); err == nil {
				_ = os.Remove(markerPath)
				return agenthooks.Interrupt(sessionSummaryPrompt)
			}
		}
	*/
	return agenthooks.Resume()
}

// DISABLED (was commit 21d62843): the session-summary writers below are turned off together with
// cxWhenAgentIdle. Kept commented (not deleted) so the feature can be re-enabled; re-add the
// "encoding/json" and "time" imports when uncommenting.
/*
// emitSessionSummary writes the per-engine session summary to the local audit log. It intentionally
// does NOT POST to the telemetry API: the backend has not confirmed support for the aiAgentSessionId
// field or a session-summary event type, so this data is kept local-only until it does. Nothing here
// affects the idle verdict.
func emitSessionSummary(ev agenthooks.AgentIdleEvent, tally map[string]sessiontally.Counts) {
	if len(tally) == 0 {
		return
	}
	// The per-engine counts and aiAgentSessionId are written to disk so an operator can `cat` what
	// each session found/remediated. This is deliberately local-only — see the function comment.
	logSessionSummaryLocal(ev, tally)
}

// sessionSummaryRecord is the on-disk audit line written at session end, mirroring the aiTelemetry
// payload's key fields (aiAgentSessionId + per-engine counts) so they are visible locally without a
// network capture. Written to cx-session-summary.jsonl alongside the plugin's own logs.
type sessionSummaryRecord struct {
	TS               string                         `json:"ts"`
	Event            string                         `json:"event"`
	AiAgentSessionID string                         `json:"aiAgentSessionId"`
	Agent            string                         `json:"agent"`
	Engines          map[string]sessiontally.Counts `json:"engines"`
}

// agentLogDir mirrors the plugin's cx_log.py convention: CX_LOG_DIR override, else
// ~/.checkmarx/agent-logs/<CX_ASSISTANT or "claude">. Returns "" if the home dir is unavailable.
func agentLogDir() string {
	if d := os.Getenv("CX_LOG_DIR"); d != "" {
		return d
	}
	assistant := os.Getenv("CX_ASSISTANT")
	if assistant == "" {
		assistant = "claude"
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".checkmarx", "agent-logs", assistant)
}

// logSessionSummaryLocal appends one best-effort JSONL record of the session summary. Never returns
// or panics — a logging failure must not affect the Stop verdict.
func logSessionSummaryLocal(ev agenthooks.AgentIdleEvent, tally map[string]sessiontally.Counts) {
	defer func() { _ = recover() }()
	dir := agentLogDir()
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	rec := sessionSummaryRecord{
		TS:               time.Now().UTC().Format(time.RFC3339),
		Event:            "session_summary",
		AiAgentSessionID: ev.SessionID,
		Agent:            agentToString(ev.Agent),
		Engines:          tally,
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "cx-session-summary.jsonl"),
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(line, '\n'))
}
*/

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
	if scaScanner != nil {
		if finding, remediation := scaScanner.CheckBashInstall(ev.Command, ev.WorkDir); finding != "" {
			touchSessionFindingsMarker()
			agent := agentToString(ev.Agent)
			sid := sessionIDFromToolCall(&ev)
			sessiontally.Add(sid, engineSca, 1, 1)
			logRemediationTelemetry(agent, "SCA", finding, remediation)
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
//     (IaC security vulnerabilities), and the SCA guardrail
//     (malicious / vulnerable manifest additions) before any bytes are written
//     to disk. MultiEdit and multi-file edits are handled uniformly by iterating
//     ev.Changes.
//
//  2. Cursor file READS (beforeReadFile) — ev.Changes is empty and ev.FilePath
//     points to a file the agent is about to ingest into the LLM context.
//     Cursor's hook payload carries only the path, so we open the file and run
//     the 2ms scanner over its contents. Blocks the read if secrets are found
//     or if the file exceeds the policy size cap. Reads do NOT count toward
//     the blast-radius budget (that limit is about writes).
func cxBeforeFileEdit(ev agenthooks.FileEditEvent) agenthooks.FileEditVerdict {
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
	if blocked, reason, context := asca.ScanFileEdit(ev, telemetryWrapper, agent); blocked {
		touchSessionFindingsMarker()
		sessiontally.Add(ev.SessionID, engineAsca, 1, 1)
		logRemediationTelemetry(agent, "Asca", reason, context)
		return agenthooks.RejectEditWithContext(reason, context)
	}
	if kicsScanner != nil {
		if blocked, reason, context := kics.ScanFileEdit(ev, kicsScanner); blocked {
			touchSessionFindingsMarker()
			sessiontally.Add(ev.SessionID, engineKics, 1, 1)
			return agenthooks.RejectEditWithContext(reason, context)
		}
	}
	if scaScanner != nil {
		for _, diff := range ev.Changes {
			if finding, remediation := scaScanner.CheckManifestEdit(ev.FilePath, fullAfterContent(ev.FilePath, diff), ev.WorkDir); finding != "" {
				touchSessionFindingsMarker()
				sessiontally.Add(ev.SessionID, engineSca, 1, 1)
				logRemediationTelemetry(agent, "Oss", finding, remediation)
				return agenthooks.RejectEditWithContext(finding, remediation)
			}
		}
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
// SCA and KICS scanners used by the Bash and FileEdit handlers.
func RegisterGuardrails(jwt wrappers.JWTWrapper, ff wrappers.FeatureFlagsWrapper, rt wrappers.RealtimeScannerWrapper, tel wrappers.TelemetryWrapper) {
	scaScanner = sca.NewScanner(jwt, ff, rt)
	kicsScanner = kics.NewScanner(jwt, ff)
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
	agenthooks.WhenAgentIdle(func(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Resume() })
	agenthooks.BeforeToolCall(func(_ agenthooks.ToolCallEvent) agenthooks.ToolVerdict { return agenthooks.Allow() })
	agenthooks.BeforeFileEdit(func(_ agenthooks.FileEditEvent) agenthooks.FileEditVerdict { return agenthooks.AcceptEdit() })
	agenthooks.BeforePrompt(func(_ agenthooks.PromptEvent) agenthooks.PromptVerdict { return agenthooks.AcceptPrompt() })
}

// logRemediationTelemetry sends telemetry when remediation context is delivered to the agent.
func logRemediationTelemetry(agent, engine, finding, remediationContext string) {
	if telemetryWrapper == nil {
		return
	}

	telemetryData := &wrappers.DataForAITelemetry{
		//agent = aiProvider
		//hooks-detect for detection
		//subtype = scan
		//  hooks-remeditae
		//subType = fixWithAIchet

		AIProvider: agent,
		Agent:      agent + "-cli",
		Engine:     engine,
		ScanType:   strings.ToLower(engine),
		UniqueID:   wrappers.GetUniqueID(),
		Type:       "hooks-remediate",
		SubType:    "fixWithAIAssist",
	}

	if err := telemetryWrapper.SendAIDataToLog(telemetryData); err != nil {
		// fail-open
	}
}

// agentToString converts agenthooks.AgentID enum to string representation for telemetry.
func agentToString(agent agenthooks.AgentID) string {
	switch agent {
	case agenthooks.AgentClaude:
		return "Claude"
	case agenthooks.AgentCopilot:
		return "Copilot"
	case agenthooks.AgentCursor:
		return "Cursor"
	case agenthooks.AgentGemini:
		return "Gemini"
	case agenthooks.AgentDroid:
		return "Droid"
	case agenthooks.AgentWindsurf:
		return "Windsurf"
	default:
		return "Unknown"
	}
}
