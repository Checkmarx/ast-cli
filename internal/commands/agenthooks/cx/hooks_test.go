//go:build !integration

package cx

import (
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/Checkmarx/ast-cx-hooks/claude"
)

func TestSessionIDFromToolCall(t *testing.T) {
	claudeEv := agenthooks.ToolCallEvent{
		Raw: &claude.PreToolUseEvent{EventBase: claude.EventBase{SessionID: "S9"}},
	}
	if got := sessionIDFromToolCall(&claudeEv); got != "S9" {
		t.Errorf("claude raw: want S9, got %q", got)
	}
	if got := sessionIDFromToolCall(&agenthooks.ToolCallEvent{Raw: nil}); got != "" {
		t.Errorf("nil raw: want empty, got %q", got)
	}
}
