package cx

import (
	"log"
	"os"
	"strings"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
	"github.com/CheckmarxDev/ast-cx-hooks/cursor"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/asca"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/sca"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

// scaScanner is the package-level SCA scanner used by the guardrail handlers.
// It is set by RegisterGuardrails so the handlers (free functions registered
// with the agenthooks library) can reach it without an injection mechanism.
var scaScanner *sca.Scanner

// cxWhenAgentIdle: agent finished its turn. Nothing to enforce yet.
func cxWhenAgentIdle(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
	return agenthooks.Resume()
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
//     guardrail (AI-introduced code vulnerabilities), and the SCA guardrail
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
	if blocked, reason, context := asca.ScanFileEdit(ev); blocked {
		return agenthooks.RejectEditWithContext(reason, context)
	}
	if scaScanner != nil {
		for _, diff := range ev.Changes {
			if finding, remediation := scaScanner.CheckManifestEdit(ev.FilePath, fullAfterContent(ev.FilePath, diff)); finding != "" {
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
// SCA scanner used by the Bash and FileEdit handlers.
func RegisterGuardrails(jwt wrappers.JWTWrapper, ff wrappers.FeatureFlagsWrapper, rt wrappers.RealtimeScannerWrapper) {
	scaScanner = sca.NewScanner(jwt, ff, rt)
	agenthooks.WhenAgentIdle(cxWhenAgentIdle)
	agenthooks.BeforeToolCall(cxBeforeToolCall)
	agenthooks.BeforeFileEdit(cxBeforeFileEdit)
	agenthooks.BeforePrompt(cxBeforePrompt)
}

// RegisterPassThrough wires no-op handlers that always allow the action.
// Used when the license check fails so we still emit valid JSON (fail-open).
func RegisterPassThrough() {
	scaScanner = nil
	agenthooks.WhenAgentIdle(func(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Resume() })
	agenthooks.BeforeToolCall(func(_ agenthooks.ToolCallEvent) agenthooks.ToolVerdict { return agenthooks.Allow() })
	agenthooks.BeforeFileEdit(func(_ agenthooks.FileEditEvent) agenthooks.FileEditVerdict { return agenthooks.AcceptEdit() })
	agenthooks.BeforePrompt(func(_ agenthooks.PromptEvent) agenthooks.PromptVerdict { return agenthooks.AcceptPrompt() })
}
