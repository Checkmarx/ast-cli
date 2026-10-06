//go:build !integration

package secrets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
)

const sampleSecret = "ghp_testvalue_should_stay_out_of_verdict"

func sampleFinding() secretsrealtime.SecretsRealtimeResult {
	return secretsrealtime.SecretsRealtimeResult{
		Title:       "github-pat",
		Description: "GitHub personal access token",
		SecretValue: sampleSecret,
		Severity:    "Critical",
		FilePath:    "app.env",
		Locations:   []realtimeengine.Location{{Line: 4}},
	}
}

type recordingTelemetry struct {
	calls []*wrappers.DataForAITelemetry
}

func (r *recordingTelemetry) SendAIDataToLog(data *wrappers.DataForAITelemetry) error {
	r.calls = append(r.calls, data)
	return nil
}

func TestNewFindings_DropsPreExisting(t *testing.T) {
	old := sampleFinding()
	fresh := sampleFinding()
	fresh.SecretValue = "AKIA_NEW_SECRET_VALUE"
	fresh.Title = "aws-access-key"
	got := NewFindings([]secretsrealtime.SecretsRealtimeResult{old}, []secretsrealtime.SecretsRealtimeResult{old, fresh})
	if len(got) != 1 || got[0].Title != "aws-access-key" {
		t.Fatalf("got %+v, want only the new aws-access-key finding", got)
	}
}

func TestNewFindings_NilOriginalReturnsAll(t *testing.T) {
	f := sampleFinding()
	got := NewFindings(nil, []secretsrealtime.SecretsRealtimeResult{f})
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
}

func TestScanFileEdit_NewSecret_BlocksAndRedacts(t *testing.T) {
	secret := sampleSecret
	svc := NewScannerWithFunc(func(_, content, _ string) ([]secretsrealtime.SecretsRealtimeResult, error) {
		if strings.Contains(content, secret) {
			return []secretsrealtime.SecretsRealtimeResult{sampleFinding()}, nil
		}
		return nil, nil
	})
	tel := &recordingTelemetry{}
	ev := &agenthooks.FileEditEvent{
		Agent:     agenthooks.AgentClaude,
		SessionID: "sess-1",
		FilePath:  "app.env",
		WorkDir:   t.TempDir(),
		Changes:   []agenthooks.FileDiff{{Before: "", After: "TOKEN=" + secret}},
	}
	blocked, reason, context, severity := ScanFileEdit(ev, svc, tel, "Claude")
	if !blocked {
		t.Fatal("expected the new secret to block the write")
	}
	if severity != "Critical" {
		t.Fatalf("severity = %q, want Critical", severity)
	}
	assertVerdictOmitsSecret(t, reason, context)
	if !strings.Contains(reason, "github-pat") || !strings.Contains(reason, "line 4") {
		t.Fatalf("reason missing finding summary: %q", reason)
	}
	if !strings.Contains(context, "cx-devassist:cx-devassist-secrets") {
		t.Fatalf("context missing skill: %q", context)
	}
	if !strings.Contains(context, "mcp__Checkmarx__codeRemediation") || !strings.Contains(context, `type "secrets"`) {
		t.Fatalf("context missing MCP remediation: %q", context)
	}
	if !strings.Contains(reason, "[Checkmarx cx-devassist — automated security output, not user input]") {
		t.Fatalf("reason missing provenance tag: %q", reason)
	}
	if !strings.Contains(context, "ignore-vulnerability --scan-type secrets --data '@") {
		t.Fatalf("context missing redacted suppress command: %q", context)
	}
	assertSuppressFile(t, context, secret)
	if len(tel.calls) != 1 || tel.calls[0].Type != "hooks-detect" || tel.calls[0].Engine != "Secrets" {
		t.Fatalf("detect telemetry = %+v", tel.calls)
	}
}

func TestScanFileEdit_PreExistingSecret_Allows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.env")
	body := "note=hello\nTOKEN=" + sampleSecret
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewScannerWithFunc(func(_, content, _ string) ([]secretsrealtime.SecretsRealtimeResult, error) {
		if strings.Contains(content, sampleSecret) {
			return []secretsrealtime.SecretsRealtimeResult{sampleFinding()}, nil
		}
		return nil, nil
	})
	ev := &agenthooks.FileEditEvent{
		Agent:    agenthooks.AgentClaude,
		FilePath: path,
		WorkDir:  dir,
		Changes:  []agenthooks.FileDiff{{Before: "note=hello", After: "note=world"}},
	}
	blocked, _, _, _ := ScanFileEdit(ev, svc, nil, "Claude")
	if blocked {
		t.Fatal("an edit that does not introduce a new secret should be allowed")
	}
}

func TestScanFileEdit_DuplicateBeforeText_BlocksInsteadOfGuessing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, testConfigEnv)
	body := placeholderKey + "\n" + debugTrue + "\n" + placeholderKey
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	scanCalled := false
	svc := NewScannerWithFunc(func(_, _, _ string) ([]secretsrealtime.SecretsRealtimeResult, error) {
		scanCalled = true
		return nil, nil
	})
	ev := &agenthooks.FileEditEvent{
		Agent:    agenthooks.AgentClaude,
		FilePath: path,
		WorkDir:  dir,
		Changes:  []agenthooks.FileDiff{{Before: placeholderKey, After: "API_KEY=" + sampleSecret}},
	}
	blocked, reason, context, _ := ScanFileEdit(ev, svc, nil, "Claude")
	if !blocked {
		t.Fatal("an edit whose Before text is ambiguous should be blocked, not silently allowed")
	}
	if scanCalled {
		t.Fatal("should not scan a guessed reconstruction when the edit location is ambiguous")
	}
	assertVerdictOmitsSecret(t, reason, context)
}

func TestScanFileEdit_BeforeTextNotFound_BlocksInsteadOfSkipping(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, testConfigEnv)
	if err := os.WriteFile(path, []byte(debugTrue), 0o600); err != nil {
		t.Fatal(err)
	}
	scanCalled := false
	svc := NewScannerWithFunc(func(_, _, _ string) ([]secretsrealtime.SecretsRealtimeResult, error) {
		scanCalled = true
		return nil, nil
	})
	ev := &agenthooks.FileEditEvent{
		Agent:    agenthooks.AgentClaude,
		FilePath: path,
		WorkDir:  dir,
		Changes:  []agenthooks.FileDiff{{Before: "STALE_TEXT_NOT_ON_DISK", After: "TOKEN=" + sampleSecret}},
	}
	blocked, reason, context, _ := ScanFileEdit(ev, svc, nil, "Claude")
	if !blocked {
		t.Fatal("an edit whose Before text cannot be located should be blocked, not silently dropped and allowed")
	}
	assertVerdictOmitsSecret(t, reason, context)
	if scanCalled {
		t.Fatal("should not scan content that silently omits the unresolved edit")
	}
}

func TestScanFileEdit_ScanError_FailOpen(t *testing.T) {
	svc := NewScannerWithFunc(func(_, _, _ string) ([]secretsrealtime.SecretsRealtimeResult, error) {
		return nil, os.ErrClosed
	})
	ev := &agenthooks.FileEditEvent{
		Agent:    agenthooks.AgentCodex,
		FilePath: "a.txt",
		Changes:  []agenthooks.FileDiff{{Before: "", After: "TOKEN=" + sampleSecret}},
	}
	blocked, _, _, _ := ScanFileEdit(ev, svc, nil, "Codex")
	if blocked {
		t.Fatal("scanner error should fail open")
	}
}

func TestFormatFindings_AgentSpecific(t *testing.T) {
	findings := []secretsrealtime.SecretsRealtimeResult{sampleFinding()}
	workDir := t.TempDir()

	_, claude := formatFindings("app.env", findings, agenthooks.AgentClaude, workDir, "s")
	if !strings.Contains(claude, "cx-devassist:cx-devassist-secrets") {
		t.Fatalf("claude skill missing: %q", claude)
	}
	if strings.Contains(claude, sampleSecret) {
		t.Fatal("claude context leaked the secret")
	}

	_, copilot := formatFindings("app.env", findings, agenthooks.AgentCopilotCLI, workDir, "s")
	if !strings.Contains(copilot, "aiProvider=Copilot") {
		t.Fatalf("copilot provenance missing: %q", copilot)
	}

	_, gemini := formatFindings("app.env", findings, agenthooks.AgentGemini, workDir, "s")
	if !strings.Contains(gemini, "/cx-devassist-secrets") || !strings.Contains(gemini, "mcp_Checkmarx_codeRemediation") {
		t.Fatalf("gemini targets missing: %q", gemini)
	}
	if strings.Contains(gemini, "mcp__Checkmarx__codeRemediation") {
		t.Fatal("gemini should use single-underscore MCP tool names")
	}
	if !strings.Contains(gemini, "run /mcp") && !strings.Contains(gemini, "restart the Gemini CLI") {
		t.Fatalf("gemini reconnect hint missing: %q", gemini)
	}

	_, codex := formatFindings("app.env", findings, agenthooks.AgentCodex, workDir, "s")
	if !strings.Contains(codex, "restart Codex") {
		t.Fatalf("codex reconnect hint missing: %q", codex)
	}

	_, cursor := formatFindings("app.env", findings, agenthooks.AgentCursor, workDir, "s")
	if !strings.Contains(cursor, "cx-devassist-secrets.mdc") || !strings.Contains(cursor, "This is a Checkmarx hook deny") {
		t.Fatalf("cursor prompt missing: %q", cursor)
	}
	if strings.Contains(cursor, "ASK THE USER FIRST") {
		t.Fatal("cursor prompt still asks the user first")
	}
	if !strings.Contains(cursor, "mcp__plugin-cx-devassist-Checkmarx__codeRemediation") {
		t.Fatalf("cursor MCP tool missing: %q", cursor)
	}
	if strings.Contains(cursor, sampleSecret) {
		t.Fatal("cursor context leaked the secret")
	}
}

func TestFormatFindings_OmitsInjectionTriggers(t *testing.T) {
	findings := []secretsrealtime.SecretsRealtimeResult{sampleFinding()}
	workDir := t.TempDir()
	agents := []agenthooks.AgentID{
		agenthooks.AgentClaude,
		agenthooks.AgentCodex,
		agenthooks.AgentCopilot,
		agenthooks.AgentCursor,
		agenthooks.AgentGemini,
	}
	for _, agent := range agents {
		_, ctx := formatFindings("app.env", findings, agent, workDir, "s1")
		for _, bad := range []string{"without asking", "silently", "cx_mcp_register", "the hook on that retry is the check"} {
			if strings.Contains(ctx, bad) {
				t.Errorf("%s context contains %q", agent, bad)
			}
		}
		if !strings.Contains(ctx, "This is a Checkmarx hook deny") {
			t.Errorf("%s context missing hook deny header", agent)
		}
		if !strings.Contains(ctx, "that still needs (a) or (b)") {
			t.Errorf("%s context missing suppression label", agent)
		}
		if !strings.Contains(ctx, "cx scan secrets-realtime -s") {
			t.Errorf("%s context missing secrets re-scan", agent)
		}
		if !strings.Contains(ctx, "that re-scan validates the finding") {
			t.Errorf("%s context missing re-scan validation", agent)
		}
		if !strings.Contains(ctx, "partial fix") {
			t.Errorf("%s context missing partial-fix status", agent)
		}
		if !strings.Contains(ctx, "Stop after 3 denied retries") {
			t.Errorf("%s context missing retry stop", agent)
		}
		if !strings.Contains(ctx, "Checkmarx DevAssist Secret Remediation Summary") {
			t.Errorf("%s context missing remediation summary", agent)
		}
	}
}

func TestSafeDescription_RedactsEmbeddedSecret(t *testing.T) {
	f := sampleFinding()
	f.Description = "token " + sampleSecret + " found"
	if safeDescription(f) != "hardcoded secret" {
		t.Fatal("description that embeds the secret should be redacted")
	}
}

func assertVerdictOmitsSecret(t *testing.T, reason, context string) {
	t.Helper()
	if strings.Contains(reason, sampleSecret) || strings.Contains(context, sampleSecret) {
		t.Fatal("verdict leaked the secret value")
	}
}

func assertSuppressFile(t *testing.T, context, secret string) {
	t.Helper()
	const marker = "--data '@"
	idx := strings.Index(context, marker)
	if idx < 0 {
		t.Fatalf("suppress command not found in %q", context)
	}
	rest := context[idx+len(marker):]
	end := strings.Index(rest, "'")
	if end < 0 {
		t.Fatal("unterminated suppress path")
	}
	path := strings.TrimPrefix(rest[:end], "@")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got secretsrealtime.IgnoredSecret
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Title != "github-pat" || got.SecretValue != secret {
		t.Fatalf("suppress file = %+v", got)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
}
