package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

// Fixtures below are verbatim shapes taken from real Claude Code transcripts,
// so the parser is tested against what the agent actually writes.
const (
	// A third-party hook that genuinely blocked a tool call.
	lineAegisBlock = `{"parentUuid":"p1","isSidechain":false,"attachment":{"type":"hook_blocking_error","hookName":"PreToolUse:Read","toolUseID":"toolu_016iMJ","hookEvent":"PreToolUse","blockingError":{"blockingError":"Blocked by Aegis Gateway | Policy: YAML File Access Prevention | Accessing YAML configuration files is blocked: 'C:/x/deployement.yaml'","command":"powershell -NonInteractive -NoProfile -ExecutionPolicy Bypass -File C:/Users/a/Desktop/Aegis/tools/aegis-hook/check.ps1"}},"type":"attachment","uuid":"u1","timestamp":"2026-08-24T04:49:24.372Z"}`

	// Our own governance hook allowing a call.
	lineGovAllow = `{"attachment":{"type":"hook_success","hookName":"PreToolUse:Bash","toolUseID":"toolu_016iMJ","hookEvent":"PreToolUse","content":"","stdout":"{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"permissionDecision\":\"allow\"}}\n","stderr":"","exitCode":0,"command":"/c/Users/a/cx.exe hooks gov-claude-pre-tool-use","durationMs":397},"type":"attachment","uuid":"u2","timestamp":"2026-08-24T04:49:24.000Z"}`

	// A third-party hook that failed instead of deciding — the call proceeded
	// unvetted, which is a governance gap worth surfacing.
	lineDevAssistFail = `{"attachment":{"type":"hook_non_blocking_error","hookName":"PreToolUse:Bash","toolUseID":"toolu_01W9TA","hookEvent":"PreToolUse","stderr":"Failed with non-blocking status code: /usr/bin/sh: cannot execute binary file\nsecond line","stdout":"","exitCode":126,"command":"Checking command against security policy...","durationMs":638},"type":"attachment","uuid":"u3","timestamp":"2026-08-21T10:41:12.568Z"}`

	// A third-party hook allowing but explaining itself.
	lineThirdPartyAllowWithReason = `{"attachment":{"type":"hook_success","hookName":"PreToolUse:Write","toolUseID":"toolu_02AB","hookEvent":"PreToolUse","stdout":"{\"hookSpecificOutput\":{\"permissionDecision\":\"allow\",\"permissionDecisionReason\":\"Path is on the reviewed allowlist\"}}","stderr":"","exitCode":0,"command":"/opt/vendor/guard/run.sh","durationMs":12},"type":"attachment","uuid":"u4","timestamp":"2026-08-24T05:00:00.000Z"}`

	// Legacy decision/reason form.
	lineLegacyBlock = `{"attachment":{"type":"hook_success","hookName":"PreToolUse:Bash","toolUseID":"toolu_03CD","hookEvent":"PreToolUse","stdout":"{\"decision\":\"block\",\"reason\":\"secret detected in command\"}","stderr":"","exitCode":0,"command":"/usr/local/bin/secretscan","durationMs":40},"type":"attachment","uuid":"u5","timestamp":"2026-08-24T05:01:00.000Z"}`

	// Noise the parser must skip without complaint.
	lineNoise  = `{"type":"user","message":{"role":"user","content":"hello"},"uuid":"u6"}`
	lineBroken = `{"attachment":{"hookName":"PreToolUse:Bash",`
)

func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	body := ""
	for _, l := range lines {
		body += l + "\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParse_BlockingRecordCarriesNameAndReason(t *testing.T) {
	path := writeTranscript(t, lineNoise, lineAegisBlock, lineBroken)
	recs, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	r := recs[0]
	if r.Decision != DecisionDeny || !r.Blocking {
		t.Errorf("decision=%q blocking=%v, want deny/true", r.Decision, r.Blocking)
	}
	if r.ToolUseID != "toolu_016iMJ" {
		t.Errorf("toolUseID=%q", r.ToolUseID)
	}
	want := "Blocked by Aegis Gateway | Policy: YAML File Access Prevention"
	if len(r.Reason) < len(want) || r.Reason[:len(want)] != want {
		t.Errorf("reason=%q, want prefix %q", r.Reason, want)
	}
	// The nested command must win — it is the one that did the blocking.
	if got := r.DisplayName(); got != "aegis-hook" {
		t.Errorf("DisplayName()=%q, want %q", got, "aegis-hook")
	}
}

func TestParse_SkipsMalformedAndUnrelatedLines(t *testing.T) {
	path := writeTranscript(t, lineNoise, lineBroken, lineNoise)
	recs, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Fatalf("got %d records, want 0", len(recs))
	}
}

func TestParse_AllowWithReason(t *testing.T) {
	recs, _ := Parse(writeTranscript(t, lineThirdPartyAllowWithReason))
	if len(recs) != 1 {
		t.Fatalf("got %d records", len(recs))
	}
	if recs[0].Decision != DecisionAllow {
		t.Errorf("decision=%q, want allow", recs[0].Decision)
	}
	if recs[0].Reason != "Path is on the reviewed allowlist" {
		t.Errorf("reason=%q", recs[0].Reason)
	}
	if recs[0].Blocking {
		t.Error("an allow must not be marked blocking")
	}
}

func TestParse_LegacyDecisionBlockMapsToDeny(t *testing.T) {
	recs, _ := Parse(writeTranscript(t, lineLegacyBlock))
	if len(recs) != 1 || recs[0].Decision != DecisionDeny || !recs[0].Blocking {
		t.Fatalf("got %+v, want a blocking deny", recs)
	}
	if recs[0].Reason != "secret detected in command" {
		t.Errorf("reason=%q", recs[0].Reason)
	}
}

func TestParse_FailedHookIsErrorNotBlock(t *testing.T) {
	recs, _ := Parse(writeTranscript(t, lineDevAssistFail))
	if len(recs) != 1 {
		t.Fatalf("got %d records", len(recs))
	}
	r := recs[0]
	if r.Decision != DecisionError {
		t.Errorf("decision=%q, want error", r.Decision)
	}
	if r.Blocking {
		t.Error("a non-blocking failure must not be reported as a block")
	}
	// Only the first stderr line — the rest is noise.
	if r.Reason != "Failed with non-blocking status code: /usr/bin/sh: cannot execute binary file" {
		t.Errorf("reason=%q", r.Reason)
	}
	if r.ExitCode != 126 {
		t.Errorf("exitCode=%d", r.ExitCode)
	}
}

func TestRecordsFor_FiltersByToolUseID(t *testing.T) {
	path := writeTranscript(t, lineAegisBlock, lineGovAllow, lineDevAssistFail)
	got := RecordsFor(path, "toolu_016iMJ")
	if len(got) != 2 {
		t.Fatalf("got %d records for toolu_016iMJ, want 2", len(got))
	}
}

func TestRecordsFor_MissingTranscriptIsSilent(t *testing.T) {
	if got := RecordsFor(filepath.Join(t.TempDir(), "nope.jsonl"), "x"); got != nil {
		t.Fatalf("want nil for a missing transcript, got %v", got)
	}
	if got := RecordsFor("", "x"); got != nil {
		t.Fatalf("want nil for an empty path, got %v", got)
	}
}

func TestThirdParty_ExcludesOurOwnHooks(t *testing.T) {
	path := writeTranscript(t, lineAegisBlock, lineGovAllow)
	all := RecordsFor(path, "toolu_016iMJ")
	tp := ThirdParty(all)
	if len(tp) != 1 {
		t.Fatalf("got %d third-party records, want 1", len(tp))
	}
	if IsGovernanceHook(tp[0].Command) {
		t.Error("our own governance hook leaked into the third-party set")
	}
}

func TestFirstBlocking(t *testing.T) {
	path := writeTranscript(t, lineGovAllow, lineAegisBlock)
	rec, ok := FirstBlocking(RecordsFor(path, "toolu_016iMJ"))
	if !ok {
		t.Fatal("expected a blocking record")
	}
	if rec.DisplayName() != "aegis-hook" {
		t.Errorf("DisplayName()=%q", rec.DisplayName())
	}

	if _, ok := FirstBlocking(nil); ok {
		t.Error("no records must yield no blocker")
	}
}

func TestIsGovernanceHook(t *testing.T) {
	cases := map[string]bool{
		"/c/x/cx.exe hooks gov-claude-pre-tool-use": true,
		"/c/x/cx.exe hooks gov-cursor-before-shell": true,
		"cx.exe hooks gov-telemetry-watch --id 1":   true,
		"/c/x/cx.exe hooks claude-pre-tool-use":     false, // the security guardrail, not governance
		"powershell -File C:/Aegis/check.ps1":       false,
		"Checking command against security policy…": false,
	}
	for cmd, want := range cases {
		if got := IsGovernanceHook(cmd); got != want {
			t.Errorf("IsGovernanceHook(%q)=%v, want %v", cmd, got, want)
		}
	}
}

func TestDisplayName(t *testing.T) {
	cases := map[string]string{
		"powershell -NonInteractive -File C:/a/Aegis/tools/aegis-hook/check.ps1": "aegis-hook",
		"/usr/local/bin/secretscan":                                      "secretscan",
		"sh \"${CLAUDE_PLUGIN_ROOT}/hooks/cx_run.sh\" hooks claude-stop": "cx_run",
		"Checking command against security policy...":                    "Checking command against security policy",
		"/opt/vendor/guard/run.sh":                                       "guard",
		"":                                                               "",
	}
	for cmd, want := range cases {
		if got := DisplayName(cmd); got != want {
			t.Errorf("DisplayName(%q)=%q, want %q", cmd, got, want)
		}
	}
}

func TestParse_ReadsTailOfLargeTranscript(t *testing.T) {
	// A transcript bigger than tailBytes must still yield the recent records —
	// that is the whole point of tailing rather than reading the lot.
	path := filepath.Join(t.TempDir(), "big.jsonl")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	padding := lineNoise + "\n"
	for written := 0; written < tailBytes+(1<<20); written += len(padding) {
		if _, err := f.WriteString(padding); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.WriteString(lineAegisBlock + "\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	recs, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 || recs[0].DisplayName() != "aegis-hook" {
		t.Fatalf("got %d records from the tail, want the trailing block record", len(recs))
	}
}
