//go:build !integration

package asca

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ignore"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
)

// ── ProposedContent ─────────────────────────────────────────────────────────

func TestProposedContent_FullFileWrite(t *testing.T) {
	newContent, _, err := ProposedContent("/nonexistent/auth.py", []agenthooks.FileDiff{
		{Before: "", After: "print('hello')"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if newContent != "print('hello')" {
		t.Fatalf("want %q, got %q", "print('hello')", newContent)
	}
}

func TestProposedContent_FullFileWrite_OriginalEmpty_WhenFileAbsent(t *testing.T) {
	_, orig, err := ProposedContent("/nonexistent/auth.py", []agenthooks.FileDiff{
		{Before: "", After: "new content"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if orig != "" {
		t.Fatalf("expected empty originalContent for absent file, got %q", orig)
	}
}

func TestProposedContent_StringReplaceEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.py")
	if err := os.WriteFile(path, []byte("x = 1\ny = 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	newContent, origContent, err := ProposedContent(path, []agenthooks.FileDiff{
		{Before: "y = 2", After: "y = 99"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if origContent != "x = 1\ny = 2\n" {
		t.Fatalf("unexpected orig: %q", origContent)
	}
	if newContent != "x = 1\ny = 99\n" {
		t.Fatalf("unexpected new: %q", newContent)
	}
}

func TestProposedContent_MissingBeforeFailsOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.py")
	if err := os.WriteFile(path, []byte("a = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Before string not present → returns original unchanged
	newContent, origContent, err := ProposedContent(path, []agenthooks.FileDiff{
		{Before: "NOTHERE", After: "replacement"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if newContent != origContent {
		t.Fatalf("expected content unchanged, got %q", newContent)
	}
}

func TestProposedContent_MultiEdit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.py")
	if err := os.WriteFile(path, []byte("a\nb\nc\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	newContent, _, err := ProposedContent(path, []agenthooks.FileDiff{
		{Before: "a", After: "A"},
		{Before: "b", After: "B"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if newContent != "A\nB\nc\n" {
		t.Fatalf("unexpected multi-edit result: %q", newContent)
	}
}

// ── stageForScan / safeSessionTag ───────────────────────────────────────────

func TestSafeSessionTag_Empty(t *testing.T) {
	if got := safeSessionTag(""); got != "anon" {
		t.Fatalf("want anon, got %q", got)
	}
}

func TestSafeSessionTag_AllSpecialChars(t *testing.T) {
	if got := safeSessionTag("!!!???"); got != "anon" {
		t.Fatalf("want anon, got %q", got)
	}
}

func TestSafeSessionTag_UUID(t *testing.T) {
	got := safeSessionTag("550e8400-e29b-41d4-a716-446655440000")
	if len(got) > 8 {
		t.Fatalf("expected ≤8 chars, got %q (len %d)", got, len(got))
	}
	for _, r := range got {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_') {
			t.Fatalf("unexpected char %q in tag %q", r, got)
		}
	}
}

func TestStageForScan_CreatesFileWithOriginalBasename(t *testing.T) {
	staged, cleanup, err := stageForScan("/some/path/auth.py", "content", "sess123")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if filepath.Base(staged) != "auth.py" {
		t.Fatalf("expected basename auth.py, got %q", filepath.Base(staged))
	}
	data, err := os.ReadFile(staged)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "content" {
		t.Fatalf("file content mismatch: %q", string(data))
	}
}

func TestStageForScan_DirNameContainsSessionTag(t *testing.T) {
	staged, cleanup, err := stageForScan("/tmp/foo.py", "x", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	dir := filepath.Dir(staged)
	base := filepath.Base(dir)
	if !strings.Contains(base, "asca-hook-") {
		t.Fatalf("expected dir name to contain asca-hook-, got %q", base)
	}
	if !strings.Contains(base, "abc123") {
		t.Fatalf("expected dir name to contain session tag, got %q", base)
	}
}

func TestStageForScan_CleanupRemovesDir(t *testing.T) {
	staged, cleanup, err := stageForScan("/tmp/foo.py", "x", "sess")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(staged)
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("expected temp dir to be removed after cleanup")
	}
}

func TestStageForScan_FileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits (0600) are not enforced on Windows; validated on Linux/macOS CI")
	}
	staged, cleanup, err := stageForScan("/tmp/secret.py", "secret", "s1")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	info, err := os.Stat(staged)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected mode 0600, got %04o", perm)
	}
}

// ── NewFindings ──────────────────────────────────────────────────────────────

func scanDetail(ruleID uint32, line string) grpcs.ScanDetail {
	return grpcs.ScanDetail{
		RuleID:          ruleID,
		ProblematicLine: line,
		Severity:        "HIGH",
		RuleName:        "test-rule",
	}
}

func TestNewFindings_NilOriginalReturnsAll(t *testing.T) {
	newScan := []grpcs.ScanDetail{scanDetail(1, "bad code")}
	got := NewFindings(nil, newScan)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
}

func TestNewFindings_IdenticalScansReturnsEmpty(t *testing.T) {
	scan := []grpcs.ScanDetail{scanDetail(42, "subprocess.run(cmd, shell=True)")}
	got := NewFindings(scan, scan)
	if len(got) != 0 {
		t.Fatalf("expected 0 new findings, got %d", len(got))
	}
}

func TestNewFindings_NewVulnReturned(t *testing.T) {
	orig := []grpcs.ScanDetail{scanDetail(1, "line A")}
	newScan := []grpcs.ScanDetail{
		scanDetail(1, "line A"),
		scanDetail(2, "line B"),
	}
	got := NewFindings(orig, newScan)
	if len(got) != 1 || got[0].RuleID != 2 {
		t.Fatalf("expected finding for ruleID 2, got %v", got)
	}
}

func TestNewFindings_OldVulnNotInNewIsIgnored(t *testing.T) {
	orig := []grpcs.ScanDetail{scanDetail(99, "old line")}
	newScan := []grpcs.ScanDetail{scanDetail(1, "new line")}
	got := NewFindings(orig, newScan)
	if len(got) != 1 || got[0].RuleID != 1 {
		t.Fatalf("unexpected findings: %v", got)
	}
}

func TestNewFindings_TrimSpaceDeduplication(t *testing.T) {
	// Same rule + same line but with different surrounding whitespace → treated as same
	orig := []grpcs.ScanDetail{scanDetail(5, "  shell=True  ")}
	newScan := []grpcs.ScanDetail{scanDetail(5, "shell=True")}
	got := NewFindings(orig, newScan)
	if len(got) != 0 {
		t.Fatalf("expected trimspace dedup, got %d findings", len(got))
	}
}

func TestNewFindings_EmptyNewScanReturnsEmpty(t *testing.T) {
	orig := []grpcs.ScanDetail{scanDetail(1, "x")}
	got := NewFindings(orig, nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(got))
	}
}

// ── additionalContext ────────────────────────────────────────────────────────

func TestAdditionalContext_SingleFinding_PreFilledCommand(t *testing.T) {
	findings := []grpcs.ScanDetail{
		{FileName: "billing.py", Line: 5, RuleID: 4059},
	}
	ctx := additionalContext("billing.py", "cx", findings, "")
	if !strings.Contains(ctx, "ignore-vulnerability") {
		t.Errorf("expected ignore-vulnerability command, got %q", ctx)
	}
	if !strings.Contains(ctx, `"FileName":"billing.py"`) {
		t.Errorf("expected FileName in command, got %q", ctx)
	}
	if !strings.Contains(ctx, `"Line":5`) {
		t.Errorf("expected Line in command, got %q", ctx)
	}
	if !strings.Contains(ctx, `"RuleID":4059`) {
		t.Errorf("expected RuleID in command, got %q", ctx)
	}
}

func TestAdditionalContext_MultipleFindings_EachGetsCommand(t *testing.T) {
	findings := []grpcs.ScanDetail{
		{FileName: "billing.py", Line: 5, RuleID: 4059},
		{FileName: "billing.py", Line: 12, RuleID: 4027},
	}
	ctx := additionalContext("billing.py", "cx", findings, "")
	if strings.Count(ctx, "ignore-vulnerability") != 2 {
		t.Errorf("expected 2 ignore commands for 2 findings, got: %q", ctx)
	}
	if !strings.Contains(ctx, `"RuleID":4059`) {
		t.Errorf("expected RuleID 4059, got %q", ctx)
	}
	if !strings.Contains(ctx, `"RuleID":4027`) {
		t.Errorf("expected RuleID 4027, got %q", ctx)
	}
}

func TestAdditionalContext_EmptyFindings_StillContainsRemediationInstruction(t *testing.T) {
	ctx := additionalContext("main.py", "cx", nil, "")
	if !strings.Contains(ctx, "mcp__Checkmarx__codeRemediation") {
		t.Errorf("expected codeRemediation instruction even with no findings, got %q", ctx)
	}
}

func TestAdditionalContext_PinsIgnoredFilePathToWorkDir(t *testing.T) {
	findings := []grpcs.ScanDetail{
		{FileName: "billing.py", Line: 5, RuleID: 4059},
	}
	workDir := filepath.Join("repo", "ws")
	ctx := additionalContext("billing.py", "cx", findings, workDir)
	want := "--ignored-file-path '" + ignore.PathFor(workDir) + "'"
	if !strings.Contains(ctx, want) {
		t.Errorf("expected context to pin %q, got %q", want, ctx)
	}
}

func TestAdditionalContext_EmptyWorkDirOmitsIgnoredFilePath(t *testing.T) {
	findings := []grpcs.ScanDetail{
		{FileName: "billing.py", Line: 5, RuleID: 4059},
	}
	ctx := additionalContext("billing.py", "cx", findings, "")
	if strings.Contains(ctx, "--ignored-file-path") {
		t.Errorf("expected no ignored-file-path flag for empty workDir, got %q", ctx)
	}
}
