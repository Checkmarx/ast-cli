//go:build !integration

package cx

import (
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/Checkmarx/ast-cx-hooks/claude"
)

// NOTE: the session-summary tests (TestCxWhenAgentIdle_*, TestEmitSessionSummary_*) and their
// helpers were removed when the session summary was disabled in hooks.go (see the DISABLED notes
// there, commit 21d62843). Restore them from git history if the summary is re-enabled.

func TestSessionIDFromToolCall(t *testing.T) {
	claudeEv := agenthooks.ToolCallEvent{
		Raw: &claude.PreToolUseEvent{EventBase: claude.EventBase{SessionID: "S9"}},
	}
	if got := sessionIDFromToolCall(claudeEv); got != "S9" {
		t.Errorf("claude raw: want S9, got %q", got)
	}
	if got := sessionIDFromToolCall(agenthooks.ToolCallEvent{Raw: nil}); got != "" {
		t.Errorf("nil raw: want empty, got %q", got)
	}
}
