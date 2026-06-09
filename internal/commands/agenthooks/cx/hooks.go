package cx

import (
	"os"

	agenthooks "github.com/CheckmarxDev/ast-cx-hooks"
	"github.com/CheckmarxDev/ast-cx-hooks/cursor"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/asca"
)

// cxWhenAgentIdle: agent finished its turn. Nothing to enforce yet.
func cxWhenAgentIdle(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict {
	return agenthooks.Resume()
}

// cxBeforeToolCall gates shell execution against the organization's blacklist and tool rules.
func cxBeforeToolCall(ev agenthooks.ToolCallEvent) agenthooks.ToolVerdict {
	if !ev.IsShell() {
		return agenthooks.Allow()
	}
	blocked, needsConfirm, reason := guardrails.CheckShellCommand(ev.Command, ev.WorkDir)
	if !blocked {
		return agenthooks.Allow()
	}
	if needsConfirm {
		return agenthooks.AskUser(reason)
	}
	return agenthooks.Deny(reason)
}

// cxBeforeFileEdit gates two distinct events the library multiplexes through
// the same handler signature:
//
//  1. File EDITS (Claude / Windsurf / Droid / Gemini) — ev.Changes is populated.
//     Enforce blast_radius_limit and files_limits.max_total_file_size_kb before
//     any bytes are written to disk.
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
	return agenthooks.AcceptEdit()
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

// RegisterGuardrails wires the four guardrail handlers.
func RegisterGuardrails() {
	agenthooks.WhenAgentIdle(cxWhenAgentIdle)
	agenthooks.BeforeToolCall(cxBeforeToolCall)
	agenthooks.BeforeFileEdit(cxBeforeFileEdit)
	agenthooks.BeforePrompt(cxBeforePrompt)
}

// RegisterPassThrough wires no-op handlers that always allow the action.
// Used when the license check fails so we still emit valid JSON (fail-open).
func RegisterPassThrough() {
	agenthooks.WhenAgentIdle(func(_ agenthooks.AgentIdleEvent) agenthooks.IdleVerdict { return agenthooks.Resume() })
	agenthooks.BeforeToolCall(func(_ agenthooks.ToolCallEvent) agenthooks.ToolVerdict { return agenthooks.Allow() })
	agenthooks.BeforeFileEdit(func(_ agenthooks.FileEditEvent) agenthooks.FileEditVerdict { return agenthooks.AcceptEdit() })
	agenthooks.BeforePrompt(func(_ agenthooks.PromptEvent) agenthooks.PromptVerdict { return agenthooks.AcceptPrompt() })
}
