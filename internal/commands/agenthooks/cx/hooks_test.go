//go:build !integration

package cx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/Checkmarx/ast-cx-hooks/claude"
	"github.com/Checkmarx/ast-cx-hooks/cursor"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/sca"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"github.com/stretchr/testify/assert"
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

// setEmptyHomeDir redirects the OS-specific home-dir env var to a fresh empty
// temp directory so guardrail policy loading (~/.checkmarx/policyhooks.json)
// fails open deterministically, regardless of the real machine's home dir.
func setEmptyHomeDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	} else {
		t.Setenv("HOME", dir)
	}
}

func TestCxWhenAgentIdle_AlwaysResumes(t *testing.T) {
	verdict := cxWhenAgentIdle(agenthooks.AgentIdleEvent{})
	assert.True(t, verdict.Proceed)
}

func TestAgentToString(t *testing.T) {
	tests := []struct {
		name  string
		agent agenthooks.AgentID
		want  string
	}{
		{"claude", agenthooks.AgentClaude, "Claude"},
		{"copilot", agenthooks.AgentCopilot, "Copilot"},
		{"cursor", agenthooks.AgentCursor, "Cursor"},
		{"gemini", agenthooks.AgentGemini, "Gemini"},
		{"droid", agenthooks.AgentDroid, "Droid"},
		{"windsurf", agenthooks.AgentWindsurf, "Windsurf"},
		{"unknown", agenthooks.AgentID("something-else"), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, agentToString(tt.agent))
		})
	}
}

func TestNormalizeNewlines(t *testing.T) {
	assert.Equal(t, "a\nb\nc", normalizeNewlines("a\r\nb\rc"))
	assert.Equal(t, "no-newlines", normalizeNewlines("no-newlines"))
}

func TestFullAfterContent_FullWrite_ReturnsAfterAsIs(t *testing.T) {
	diff := agenthooks.FileDiff{Before: "", After: "brand new content"}
	got := fullAfterContent(filepath.Join(t.TempDir(), "missing.txt"), diff)
	assert.Equal(t, "brand new content", string(got))
}

func TestFullAfterContent_ExactReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	assert.NoError(t, os.WriteFile(path, []byte("hello old world"), 0600))

	diff := agenthooks.FileDiff{Before: "old", After: "new"}
	got := fullAfterContent(path, diff)
	assert.Equal(t, "hello new world", string(got))
}

func TestFullAfterContent_LineEndingNormalizedReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	assert.NoError(t, os.WriteFile(path, []byte("line1\r\nold-region\r\nline3"), 0600))

	diff := agenthooks.FileDiff{Before: "old-region\n", After: "new-region\n"}
	got := fullAfterContent(path, diff)
	assert.Equal(t, "line1\nnew-region\nline3", string(got))
}

func TestFullAfterContent_RegionNotFound_FallsBackToNormalizedAfter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	assert.NoError(t, os.WriteFile(path, []byte("completely unrelated content"), 0600))

	diff := agenthooks.FileDiff{Before: "not-present-anywhere", After: "fallback\r\ncontent"}
	got := fullAfterContent(path, diff)
	assert.Equal(t, "fallback\ncontent", string(got))
}

func TestFullAfterContent_MissingFileWithBefore_ReturnsAfter(t *testing.T) {
	diff := agenthooks.FileDiff{Before: "old", After: "new content"}
	got := fullAfterContent(filepath.Join(t.TempDir(), "missing.txt"), diff)
	assert.Equal(t, "new content", string(got))
}

func TestPromptWorkspaceRoots_CursorEventWithRoots(t *testing.T) {
	raw := &cursor.PromptPreEvent{EventBase: cursor.EventBase{WorkspaceRoots: []string{"/repo/a", "/repo/b"}}}
	roots := promptWorkspaceRoots(raw)
	assert.Equal(t, []string{"/repo/a", "/repo/b"}, roots)
}

func TestPromptWorkspaceRoots_NonCursorEvent_FallsBackToCwd(t *testing.T) {
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	roots := promptWorkspaceRoots(nil)
	assert.Equal(t, []string{cwd}, roots)
}

func TestCxBeforeToolCall_NonShell_Allows(t *testing.T) {
	verdict := cxBeforeToolCall(agenthooks.ToolCallEvent{Kind: agenthooks.ToolKindBuiltin})
	assert.True(t, verdict.Permit)
}

func TestCxBeforeToolCall_ShellNoScanner_Allows(t *testing.T) {
	setEmptyHomeDir(t)
	scaScanner = nil

	verdict := cxBeforeToolCall(agenthooks.ToolCallEvent{Kind: agenthooks.ToolKindShell, Command: "ls -la"})
	assert.True(t, verdict.Permit)
}

func TestCxBeforeToolCall_ShellWithMaliciousPackage_DeniesWithContext(t *testing.T) {
	setEmptyHomeDir(t)
	prevScanner, prevTelemetry := scaScanner, telemetryWrapper
	defer func() { scaScanner, telemetryWrapper = prevScanner, prevTelemetry }()

	scaScanner = sca.NewScannerWithFunc(func(string) (*ossrealtime.OssPackageResults, error) {
		return &ossrealtime.OssPackageResults{
			Packages: []ossrealtime.OssPackage{{PackageName: "lodash", PackageVersion: "4.17.21", Status: "Malicious"}},
		}, nil
	})
	telemetryWrapper = mock.TelemetryMockWrapper{}

	verdict := cxBeforeToolCall(agenthooks.ToolCallEvent{Kind: agenthooks.ToolKindShell, Command: "npm install lodash@4.17.21"})
	assert.False(t, verdict.Permit)
	assert.Contains(t, verdict.Message, "MALICIOUS")
}

func TestCxBeforeFileEdit_CursorRead_NoSecrets_Accepts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "readme.txt")
	assert.NoError(t, os.WriteFile(path, []byte("just some plain text, nothing sensitive here"), 0600))

	verdict := cxBeforeFileEdit(agenthooks.FileEditEvent{Agent: agenthooks.AgentCursor, FilePath: path})
	assert.True(t, verdict.Permit)
}

func TestCxBeforeFileEdit_UnsupportedFileType_Accepts(t *testing.T) {
	setEmptyHomeDir(t)
	prevSca, prevKics := scaScanner, kicsScanner
	defer func() { scaScanner, kicsScanner = prevSca, prevKics }()
	scaScanner = nil
	kicsScanner = nil

	path := filepath.Join(t.TempDir(), "notes.txt")
	ev := agenthooks.FileEditEvent{
		Agent:    agenthooks.AgentClaude,
		FilePath: path,
		Changes:  []agenthooks.FileDiff{{Before: "", After: "hello world"}},
	}

	verdict := cxBeforeFileEdit(ev)
	assert.True(t, verdict.Permit)
}

func TestCxBeforePrompt_Benign_Accepts(t *testing.T) {
	setEmptyHomeDir(t)
	verdict := cxBeforePrompt(agenthooks.PromptEvent{Text: "please explain how this function works"})
	assert.True(t, verdict.Accept)
}

func TestRegisterGuardrails_SetsScanners(t *testing.T) {
	prevSca, prevKics, prevTelemetry := scaScanner, kicsScanner, telemetryWrapper
	defer func() { scaScanner, kicsScanner, telemetryWrapper = prevSca, prevKics, prevTelemetry }()

	telemetry := mock.TelemetryMockWrapper{}
	RegisterGuardrails(&mock.JWTMockWrapper{}, &mock.FeatureFlagsMockWrapper{}, &mock.RealtimeScannerMockWrapper{}, telemetry)

	assert.NotNil(t, scaScanner)
	assert.NotNil(t, kicsScanner)
	assert.Equal(t, telemetry, telemetryWrapper)
}

func TestRegisterPassThrough_ClearsScanners(t *testing.T) {
	prevSca, prevKics := scaScanner, kicsScanner
	defer func() { scaScanner, kicsScanner = prevSca, prevKics }()

	RegisterGuardrails(&mock.JWTMockWrapper{}, &mock.FeatureFlagsMockWrapper{}, &mock.RealtimeScannerMockWrapper{}, mock.TelemetryMockWrapper{})
	assert.NotNil(t, scaScanner)

	RegisterPassThrough()
	assert.Nil(t, scaScanner)
	assert.Nil(t, kicsScanner)
}

func TestLogRemediationTelemetry_NilWrapper_NoOp(t *testing.T) {
	prevTelemetry := telemetryWrapper
	defer func() { telemetryWrapper = prevTelemetry }()

	telemetryWrapper = nil
	assert.NotPanics(t, func() {
		logRemediationTelemetry("Claude", "SCA", "finding", "remediation")
	})
}

func TestLogRemediationTelemetry_WithWrapper_Sends(t *testing.T) {
	prevTelemetry := telemetryWrapper
	defer func() { telemetryWrapper = prevTelemetry }()

	telemetryWrapper = mock.TelemetryMockWrapper{}
	assert.NotPanics(t, func() {
		logRemediationTelemetry("Claude", "SCA", "finding", "remediation")
	})
}
