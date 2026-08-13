//go:build !integration

package cx

import (
	"encoding/json"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"os"
	"path/filepath"
	"runtime"

	"strings"

	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/Checkmarx/ast-cx-hooks/claude"

	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/guardrails/kics"
	"github.com/checkmarx/ast-cli/internal/commands/agenthooks/sca"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/iacrealtime"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"

	"github.com/Checkmarx/ast-cx-hooks/cursor"

	"github.com/stretchr/testify/assert"
)

// sampleJWT is a well-known test JWT (no real value) used to trigger the 2ms secret scanner.
const (
	sampleJWT = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
		"eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ." +
		"SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	osWindows = "windows"
)

type recordingTelemetry struct {
	calls []*wrappers.DataForAITelemetry
	err   error
}

func (r *recordingTelemetry) SendAIDataToLog(data *wrappers.DataForAITelemetry) error {
	r.calls = append(r.calls, data)
	return r.err
}

func resetHookGlobals(t *testing.T) {
	t.Helper()
	prevSCA, prevKICS, prevTel := scaScanner, kicsScanner, telemetryWrapper
	t.Cleanup(func() {
		scaScanner = prevSCA
		kicsScanner = prevKICS
		telemetryWrapper = prevTel
		guardrails.ResetBlastRadiusCount()
		guardrails.ResetTotalFileSizeCount()
	})
	scaScanner = nil
	kicsScanner = nil
	telemetryWrapper = nil
	guardrails.ResetBlastRadiusCount()
	guardrails.ResetTotalFileSizeCount()
}

func setHomeDir(dir string) func() {
	const osWindows = "windows"
	if runtime.GOOS == osWindows {
		orig, had := os.LookupEnv("USERPROFILE")
		_ = os.Setenv("USERPROFILE", dir)
		return func() {
			if had {
				_ = os.Setenv("USERPROFILE", orig)
			} else {
				_ = os.Unsetenv("USERPROFILE")
			}
		}
	}
	orig, had := os.LookupEnv("HOME")
	_ = os.Setenv("HOME", dir)
	return func() {
		if had {
			_ = os.Setenv("HOME", orig)
		} else {
			_ = os.Unsetenv("HOME")
		}
	}
}

func writePolicy(t *testing.T, policy *guardrails.HooksPolicy) func() {
	t.Helper()
	data, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal policy: %v", err)
	}
	dir := t.TempDir()
	cxDir := filepath.Join(dir, ".checkmarx")
	if err := os.MkdirAll(cxDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cxDir, "policyhooks.json"), data, 0o644); err != nil {
		t.Fatalf("write policy: %v", err)
	}
	return setHomeDir(dir)
}

func currentOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case osWindows:
		return osWindows
	default:
		return "linux"
	}
}

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
	if got := sessionIDFromToolCall(&agenthooks.ToolCallEvent{Raw: "not-claude"}); got != "" {
		t.Errorf("non-claude raw: want empty, got %q", got)
	}
}

func TestCxWhenAgentIdle(t *testing.T) {
	v := cxWhenAgentIdle(agenthooks.AgentIdleEvent{Agent: agenthooks.AgentClaude})
	if !v.Proceed {
		t.Fatal("cxWhenAgentIdle should Resume (Proceed=true)")
	}
}

func TestCxBeforeToolCall_Blacklisted_Denies(t *testing.T) {
	resetHookGlobals(t)
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	policy.DefaultPolicy.BlacklistTools.Tools = []guardrails.BlacklistedTool{
		{Name: "rm -rf", OS: []string{currentOS()}, Category: "destructive", Risk: "wipes files"},
	}
	defer writePolicy(t, &policy)()

	v := cxBeforeToolCall(agenthooks.ToolCallEvent{
		Kind:    agenthooks.ToolKindShell,
		Command: "rm -rf /tmp/foo",
	})
	if v.Permit {
		t.Fatal("blacklisted shell should Deny")
	}
	if v.NeedsConfirm {
		t.Fatal("blacklist should hard-deny, not AskUser")
	}
	if v.Message == "" {
		t.Fatal("expected deny reason")
	}
}

func TestCxBeforeToolCall_ToolRule_AsksUser(t *testing.T) {
	resetHookGlobals(t)
	policy := guardrails.HooksPolicy{}
	policy.Tools.Enabled = true
	policy.Tools.Rules = []guardrails.ToolRule{{
		ID:          "t2",
		Tool:        []string{"mvn"},
		OS:          []string{currentOS()},
		ArgsInclude: []string{"compile", "test"},
	}}
	defer writePolicy(t, &policy)()

	v := cxBeforeToolCall(agenthooks.ToolCallEvent{
		Kind:    agenthooks.ToolKindShell,
		Command: "mvn unknown-goal",
	})
	if v.Permit {
		t.Fatal("unknown arg should not Permit")
	}
	if !v.NeedsConfirm {
		t.Fatal("unknown arg should AskUser (NeedsConfirm=true)")
	}
}

func TestCxBeforeToolCall_SCAFinding_DeniesWithContext(t *testing.T) {
	resetHookGlobals(t)
	tel := &recordingTelemetry{}
	telemetryWrapper = tel
	scaScanner = sca.NewScannerWithFunc(func(string) (*ossrealtime.OssPackageResults, error) {
		return &ossrealtime.OssPackageResults{Packages: []ossrealtime.OssPackage{{
			PackageName: "lodash", PackageVersion: "4.17.21", Status: "Malicious",
		}}}, nil
	})

	v := cxBeforeToolCall(agenthooks.ToolCallEvent{
		Agent:   agenthooks.AgentClaude,
		Kind:    agenthooks.ToolKindShell,
		Command: "npm install lodash@4.17.21",
		Raw:     &claude.PreToolUseEvent{EventBase: claude.EventBase{SessionID: "sess-sca"}},
	})
	if v.Permit {
		t.Fatal("malicious install should Deny")
	}
	if v.Context == "" {
		t.Fatal("expected remediation Context")
	}
	if !strings.Contains(v.Message, "MALICIOUS") {
		t.Errorf("expected MALICIOUS in finding, got %q", v.Message)
	}
	if len(tel.calls) != 1 {
		t.Fatalf("expected 1 telemetry call, got %d", len(tel.calls))
	}
	if tel.calls[0].Engine != "SCA" {
		t.Errorf("Engine = %q, want SCA", tel.calls[0].Engine)
	}
}

func TestCxBeforeToolCall_CleanShell_Allows(t *testing.T) {
	resetHookGlobals(t)
	scaScanner = sca.NewScannerWithFunc(func(string) (*ossrealtime.OssPackageResults, error) {
		return &ossrealtime.OssPackageResults{Packages: []ossrealtime.OssPackage{{
			PackageName: "lodash", Status: "OK",
		}}}, nil
	})
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlacklistTools.Enabled = true
	policy.DefaultPolicy.BlacklistTools.Tools = []guardrails.BlacklistedTool{
		{Name: "rm -rf", OS: []string{currentOS()}, Category: "destructive", Risk: "bad"},
	}
	defer writePolicy(t, &policy)()

	v := cxBeforeToolCall(agenthooks.ToolCallEvent{
		Kind:    agenthooks.ToolKindShell,
		Command: "npm install lodash",
	})
	if !v.Permit {
		t.Fatalf("clean install should Allow, got Message=%q", v.Message)
	}
}

func TestCxBeforeFileEdit_CursorRead_SecretsReject(t *testing.T) {
	resetHookGlobals(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.env")
	if err := os.WriteFile(path, []byte("TOKEN="+sampleJWT+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	v := cxBeforeFileEdit(agenthooks.FileEditEvent{
		Agent:    agenthooks.AgentCursor,
		FilePath: path,
		Changes:  nil,
	})
	if v.Permit {
		t.Fatal("Cursor read of secret file should RejectEdit")
	}
	if v.Message == "" {
		t.Fatal("expected rejection reason")
	}
}

func TestCxBeforeFileEdit_CursorRead_CleanAccept(t *testing.T) {
	resetHookGlobals(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "readme.md")
	if err := os.WriteFile(path, []byte("hello world\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	v := cxBeforeFileEdit(agenthooks.FileEditEvent{
		Agent:    agenthooks.AgentCursor,
		FilePath: path,
		Changes:  nil,
	})
	if !v.Permit {
		t.Fatalf("clean Cursor read should AcceptEdit, got Message=%q", v.Message)
	}
}

func TestCxBeforeFileEdit_BlastRadius_Rejects(t *testing.T) {
	resetHookGlobals(t)
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.BlastRadiusLimit = guardrails.BlastRadiusLimit{Enabled: true, Threshold: 1}
	defer writePolicy(t, &policy)()

	// Consume the single allowed write so the next edit is blocked.
	if blocked, _ := guardrails.CheckAndIncrementBlastRadius(); blocked {
		t.Fatal("first blast-radius increment should be allowed")
	}

	v := cxBeforeFileEdit(agenthooks.FileEditEvent{
		Agent:    agenthooks.AgentClaude,
		FilePath: "notes.txt",
		Changes:  []agenthooks.FileDiff{{Before: "", After: "hi"}},
	})
	if v.Permit {
		t.Fatal("edit past blast-radius threshold should RejectEdit")
	}
	if !strings.Contains(v.Message, "blast radius") {
		t.Errorf("expected blast radius reason, got %q", v.Message)
	}
}

func TestCxBeforeFileEdit_TotalFileSize_Rejects(t *testing.T) {
	resetHookGlobals(t)
	policy := guardrails.HooksPolicy{}
	policy.DefaultPolicy.ContextPolicy.Enabled = true
	policy.DefaultPolicy.ContextPolicy.FilesLimits = guardrails.FilesLimits{
		Enabled:            true,
		MaxTotalFileSizeKB: 1,
	}
	defer writePolicy(t, &policy)()

	big := strings.Repeat("a", 1100)
	v := cxBeforeFileEdit(agenthooks.FileEditEvent{
		Agent:    agenthooks.AgentClaude,
		FilePath: "notes.txt",
		Changes:  []agenthooks.FileDiff{{Before: "", After: big}},
	})
	if v.Permit {
		t.Fatal("oversized edit should RejectEdit")
	}
	if !strings.Contains(v.Message, "total file size") {
		t.Errorf("expected total file size reason, got %q", v.Message)
	}
}

func TestCxBeforeFileEdit_KICSFinding_RejectsWithContext(t *testing.T) {
	resetHookGlobals(t)
	kicsScanner = kics.NewScannerWithFunc(func(string) ([]iacrealtime.IacRealtimeResult, error) {
		return []iacrealtime.IacRealtimeResult{{
			Title:        "Privileged Container",
			SimilarityID: "sim123",
			Severity:     "HIGH",
			Description:  "Container runs as privileged",
			Locations:    []realtimeengine.Location{{Line: 5}},
		}}, nil
	})

	v := cxBeforeFileEdit(agenthooks.FileEditEvent{
		Agent:     agenthooks.AgentClaude,
		SessionID: "kics-sess",
		FilePath:  "/project/Dockerfile",
		Changes:   []agenthooks.FileDiff{{Before: "", After: "FROM ubuntu\nUSER root\n"}},
	})
	if v.Permit {
		t.Fatal("KICS finding should RejectEdit")
	}
	if v.Context == "" {
		t.Fatal("expected remediation Context")
	}
	if !strings.Contains(v.Message, "KICS") {
		t.Errorf("expected KICS in reason, got %q", v.Message)
	}
}

func TestCxBeforeFileEdit_SCAManifest_RejectsWithContext(t *testing.T) {
	resetHookGlobals(t)
	tel := &recordingTelemetry{}
	telemetryWrapper = tel
	scaScanner = sca.NewScannerWithFunc(func(string) (*ossrealtime.OssPackageResults, error) {
		return &ossrealtime.OssPackageResults{Packages: []ossrealtime.OssPackage{{
			PackageName: "evil-pkg", PackageVersion: "1.0.0", Status: "Malicious",
		}}}, nil
	})

	dir := t.TempDir()
	manifest := filepath.Join(dir, "package.json")
	before := `{"dependencies":{}}`
	after := `{"dependencies":{"evil-pkg":"1.0.0"}}`
	if err := os.WriteFile(manifest, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}

	v := cxBeforeFileEdit(agenthooks.FileEditEvent{
		Agent:     agenthooks.AgentClaude,
		SessionID: "sca-edit",
		FilePath:  manifest,
		WorkDir:   dir,
		Changes:   []agenthooks.FileDiff{{Before: before, After: after}},
	})
	if v.Permit {
		t.Fatal("malicious manifest edit should RejectEdit")
	}
	if v.Context == "" {
		t.Fatal("expected remediation Context")
	}
	if len(tel.calls) != 1 {
		t.Fatalf("expected 1 telemetry call, got %d", len(tel.calls))
	}
	if tel.calls[0].Engine != "Oss" {
		t.Errorf("Engine = %q, want Oss", tel.calls[0].Engine)
	}
}

func TestCxBeforeFileEdit_CleanEdit_Accepts(t *testing.T) {
	resetHookGlobals(t)
	v := cxBeforeFileEdit(agenthooks.FileEditEvent{
		Agent:    agenthooks.AgentClaude,
		FilePath: "notes.txt",
		Changes:  []agenthooks.FileDiff{{Before: "", After: "hello"}},
	})
	if !v.Permit {
		t.Fatalf("clean edit should AcceptEdit, got Message=%q", v.Message)
	}
}

func TestFullAfterContent(t *testing.T) {
	t.Run("write_op_returns_after", func(t *testing.T) {
		got := fullAfterContent("/no/such/file", agenthooks.FileDiff{Before: "", After: "new"})
		if string(got) != "new" {
			t.Errorf("got %q, want new", got)
		}
	})

	t.Run("exact_replace", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "f.txt")
		if err := os.WriteFile(path, []byte("hello world"), 0o600); err != nil {
			t.Fatal(err)
		}
		got := fullAfterContent(path, agenthooks.FileDiff{Before: "world", After: "there"})
		if string(got) != "hello there" {
			t.Errorf("got %q, want hello there", got)
		}
	})

	t.Run("crlf_normalized_replace", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "f.txt")
		if err := os.WriteFile(path, []byte("line1\r\nline2\r\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		// Diff region uses LF while file on disk uses CRLF.
		got := fullAfterContent(path, agenthooks.FileDiff{
			Before: "line1\nline2\n",
			After:  "line1\nline2-changed\n",
		})
		if !strings.Contains(string(got), "line2-changed") {
			t.Errorf("normalized replace failed, got %q", got)
		}
	})

	t.Run("missing_file_falls_back_to_after", func(t *testing.T) {
		got := fullAfterContent(filepath.Join(t.TempDir(), "missing.txt"), agenthooks.FileDiff{
			Before: "old", After: "snippet",
		})
		if string(got) != "snippet" {
			t.Errorf("got %q, want snippet", got)
		}
	})

	t.Run("unmatched_region_scans_normalized_after", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "f.txt")
		if err := os.WriteFile(path, []byte("unchanged content"), 0o600); err != nil {
			t.Fatal(err)
		}
		got := fullAfterContent(path, agenthooks.FileDiff{
			Before: "not-in-file", After: "proposed\r\nsnippet",
		})
		if string(got) != "proposed\nsnippet" {
			t.Errorf("got %q, want LF-normalized snippet", got)
		}
	})
}

func TestCxBeforePrompt_Secret_Rejects(t *testing.T) {
	resetHookGlobals(t)
	v := cxBeforePrompt(agenthooks.PromptEvent{Text: "here is my token " + sampleJWT})
	if v.Accept {
		t.Fatal("prompt with JWT should RejectPrompt")
	}
	if v.Message == "" {
		t.Fatal("expected rejection message")
	}
}

func TestCxBeforePrompt_Clean_Accepts(t *testing.T) {
	resetHookGlobals(t)
	v := cxBeforePrompt(agenthooks.PromptEvent{Text: "please refactor the helper"})
	if !v.Accept {
		t.Fatalf("clean prompt should AcceptPrompt, got Message=%q", v.Message)
	}
}

func TestPromptWorkspaceRoots(t *testing.T) {
	t.Run("cursor_with_roots", func(t *testing.T) {
		roots := []string{"/ws/a", "/ws/b"}
		got := promptWorkspaceRoots(&cursor.PromptPreEvent{
			EventBase: cursor.EventBase{WorkspaceRoots: roots},
		})
		if len(got) != 2 || got[0] != "/ws/a" || got[1] != "/ws/b" {
			t.Errorf("got %v, want %v", got, roots)
		}
	})

	t.Run("fallback_cwd", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		got := promptWorkspaceRoots(nil)
		if len(got) != 1 || got[0] != cwd {
			t.Errorf("got %v, want [%q]", got, cwd)
		}
	})

	t.Run("cursor_empty_roots_falls_back", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		got := promptWorkspaceRoots(&cursor.PromptPreEvent{})
		if len(got) != 1 || got[0] != cwd {
			t.Errorf("got %v, want [%q]", got, cwd)
		}
	})
}

func TestRegisterGuardrails_AndPassThrough(t *testing.T) {
	resetHookGlobals(t)

	RegisterGuardrails(
		&mock.JWTMockWrapper{},
		mock.FeatureFlagsMockWrapper{},
		mock.NewRealtimeScannerMockWrapper(),
		mock.TelemetryMockWrapper{},
	)
	if scaScanner == nil {
		t.Fatal("RegisterGuardrails should set scaScanner")
	}
	if kicsScanner == nil {
		t.Fatal("RegisterGuardrails should set kicsScanner")
	}
	if telemetryWrapper == nil {
		t.Fatal("RegisterGuardrails should set telemetryWrapper")
	}

	RegisterPassThrough()
	if scaScanner != nil {
		t.Fatal("RegisterPassThrough should clear scaScanner")
	}
	if kicsScanner != nil {
		t.Fatal("RegisterPassThrough should clear kicsScanner")
	}
}

func TestLogRemediationTelemetry(t *testing.T) {
	resetHookGlobals(t)

	t.Run("nil_wrapper_noop", func(t *testing.T) {
		telemetryWrapper = nil
		logRemediationTelemetry("Claude", "SCA", "High", "s1") // must not panic
	})

	t.Run("sends_payload", func(t *testing.T) {
		tel := &recordingTelemetry{}
		telemetryWrapper = tel
		logRemediationTelemetry("Cursor", "Asca", "Critical", "sess-9")
		if len(tel.calls) != 1 {
			t.Fatalf("got %d calls, want 1", len(tel.calls))
		}
		got := tel.calls[0]
		if got.AIProvider != "Cursor" || got.Agent != "Cursor-cli" {
			t.Errorf("AIProvider/Agent = %q/%q", got.AIProvider, got.Agent)
		}
		if got.Engine != "Asca" || got.ScanType != "asca" {
			t.Errorf("Engine/ScanType = %q/%q", got.Engine, got.ScanType)
		}
		if got.Type != "hooks-remediate" || got.SubType != "fixWithAIAssist" {
			t.Errorf("Type/SubType = %q/%q", got.Type, got.SubType)
		}
		if got.ProblemSeverity != "Critical" || got.AiAgentSessionId != "sess-9" {
			t.Errorf("severity/session = %q/%q", got.ProblemSeverity, got.AiAgentSessionId)
		}
	})

	t.Run("send_error_fail_open", func(t *testing.T) {
		tel := &recordingTelemetry{err: os.ErrPermission}
		telemetryWrapper = tel
		logRemediationTelemetry("Claude", "SCA", "High", "s2") // must not panic
		if len(tel.calls) != 1 {
			t.Fatalf("got %d calls, want 1", len(tel.calls))
		}
	})
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
