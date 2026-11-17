package telemetry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfirmFileWritten_DetectsNewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "created.txt")
	before := Snapshot(path)
	if before.Exists {
		t.Fatal("baseline should record a missing file")
	}

	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := ConfirmFileWritten(before, 500*time.Millisecond); got != Confirmed {
		t.Fatalf("ConfirmFileWritten = %v, want Confirmed", got)
	}
}

func TestConfirmFileWritten_DetectsContentChangeAtSameSize(t *testing.T) {
	// Same byte count and a preserved mtime — only the content hash can tell
	// these apart, which is exactly the case stat-only detection would miss.
	path := filepath.Join(t.TempDir(), "edited.txt")
	if err := os.WriteFile(path, []byte("aaaa"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := Snapshot(path)

	if err := os.WriteFile(path, []byte("bbbb"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, before.ModTime, before.ModTime); err != nil {
		t.Fatal(err)
	}

	if got := ConfirmFileWritten(before, 300*time.Millisecond); got != Confirmed {
		t.Fatalf("ConfirmFileWritten = %v, want Confirmed", got)
	}
}

func TestConfirmFileWritten_NotObservedWhenNothingHappens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "untouched.txt")
	if err := os.WriteFile(path, []byte("stable"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := Snapshot(path)

	if got := ConfirmFileWritten(before, 200*time.Millisecond); got != NotObserved {
		t.Fatalf("ConfirmFileWritten = %v, want NotObserved", got)
	}
}

func TestConfirmFileWritten_InconclusiveWithoutPath(t *testing.T) {
	if got := ConfirmFileWritten(FileSnapshot{}, time.Millisecond); got != Inconclusive {
		t.Fatalf("ConfirmFileWritten = %v, want Inconclusive", got)
	}
}

func TestConfirmAnyFileWritten_OneOfSeveralIsEnough(t *testing.T) {
	dir := t.TempDir()
	untouched := filepath.Join(dir, "a.txt")
	written := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(untouched, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := SnapshotAll([]string{untouched, written})
	if len(before) != 2 {
		t.Fatalf("SnapshotAll returned %d snapshots, want 2", len(before))
	}

	if err := os.WriteFile(written, []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := ConfirmAnyFileWritten(before, 300*time.Millisecond); got != Confirmed {
		t.Fatalf("ConfirmAnyFileWritten = %v, want Confirmed", got)
	}
}

func TestConfirmAnyFileWritten_InconclusiveWithoutSnapshots(t *testing.T) {
	if got := ConfirmAnyFileWritten(nil, time.Millisecond); got != Inconclusive {
		t.Fatalf("ConfirmAnyFileWritten = %v, want Inconclusive", got)
	}
}

func TestSnapshotAll_SkipsEmptyPaths(t *testing.T) {
	got := SnapshotAll([]string{"", ""})
	if len(got) != 0 {
		t.Fatalf("SnapshotAll = %d snapshots, want 0", len(got))
	}
}
