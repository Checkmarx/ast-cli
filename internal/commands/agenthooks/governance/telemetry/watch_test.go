package telemetry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVerdictRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := WriteVerdict(dir, "tool_use_123", NotObserved, "process"); err != nil {
		t.Fatal(err)
	}
	v, ok := ReadVerdict(dir, "tool_use_123")
	if !ok {
		t.Fatal("verdict not found after write")
	}
	if v.Result() != NotObserved {
		t.Fatalf("Result() = %v, want NotObserved", v.Result())
	}
	if v.Signal != "process" {
		t.Fatalf("Signal = %q, want %q", v.Signal, "process")
	}
}

func TestReadVerdict_MissingIsNotAnError(t *testing.T) {
	if _, ok := ReadVerdict(t.TempDir(), "nope"); ok {
		t.Fatal("ReadVerdict reported a verdict that was never written")
	}
}

func TestVerdictResult_UnknownOutcomeIsInconclusive(t *testing.T) {
	if got := (Verdict{Outcome: "garbage"}).Result(); got != Inconclusive {
		t.Fatalf("Result() = %v, want Inconclusive", got)
	}
}

func TestSignatureRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := ProcessSignature{Command: "npm run build", WorkDir: dir, StartedAt: time.Now().UTC().Truncate(time.Second)}
	if err := WriteSignature(dir, "abc", want); err != nil {
		t.Fatal(err)
	}
	got, ok := ReadSignature(dir, "abc")
	if !ok {
		t.Fatal("signature not found after write")
	}
	if got.Command != want.Command || !got.StartedAt.Equal(want.StartedAt) {
		t.Fatalf("ReadSignature = %+v, want %+v", got, want)
	}
}

func TestRunProcessWatcher_WritesInconclusiveWithoutSignature(t *testing.T) {
	dir := t.TempDir()
	if err := RunProcessWatcher(dir, "missing", 10*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	v, ok := ReadVerdict(dir, "missing")
	if !ok || v.Result() != Inconclusive {
		t.Fatalf("verdict = %+v (found=%v), want Inconclusive", v, ok)
	}
}

func TestCleanupRemovesBothArtifacts(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSignature(dir, "id1", ProcessSignature{Command: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := WriteVerdict(dir, "id1", Confirmed, "process"); err != nil {
		t.Fatal(err)
	}
	Cleanup(dir, "id1")

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("Cleanup left %d files behind", len(entries))
	}
}

func TestPurgeStale(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "old.verdict.json")
	if err := os.WriteFile(stale, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(dir, "new.verdict.json")
	if err := os.WriteFile(fresh, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	PurgeStale(dir, time.Minute)

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale verdict should have been purged")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Error("fresh verdict should have survived the purge")
	}
}

func TestSafeID(t *testing.T) {
	cases := map[string]string{
		"toolu_01ABC-def":  "toolu_01ABC-def",
		"../../etc/passwd": "etcpasswd",
		"":                 "unknown",
		"///":              "unknown",
	}
	for in, want := range cases {
		if got := safeID(in); got != want {
			t.Errorf("safeID(%q) = %q, want %q", in, got, want)
		}
	}
}
