//go:build !integration

package sessiontally

import (
	"os"
	"sync"
	"testing"
)

// sandbox points ~/.checkmarx at a temp dir on every OS (os.UserHomeDir uses USERPROFILE on Windows,
// HOME elsewhere), so tests never touch the real home directory.
func sandbox(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestAddLoadClearRoundTrip(t *testing.T) {
	sandbox(t)
	Add("S1", "Asca", 2, 2)
	Add("S1", "Asca", 1, 1)
	Add("S1", "Sca", 3, 1)

	got := Load("S1")
	if got["Asca"].VulnerabilitiesFound != 3 || got["Asca"].RemediationsOffered != 3 {
		t.Errorf("Asca fold wrong: %+v", got["Asca"])
	}
	if got["Sca"].VulnerabilitiesFound != 3 || got["Sca"].RemediationsOffered != 1 {
		t.Errorf("Sca fold wrong: %+v", got["Sca"])
	}

	Clear("S1")
	if n := len(Load("S1")); n != 0 {
		t.Errorf("expected empty after Clear, got %d engines", n)
	}
}

func TestConcurrentAddDoesNotLoseRecords(t *testing.T) {
	sandbox(t)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); Add("S1", "Asca", 1, 0) }()
	}
	wg.Wait()
	if got := Load("S1")["Asca"].VulnerabilitiesFound; got != 50 {
		t.Errorf("concurrent append lost records: got %d want 50", got)
	}
}

func TestEmptyIDUsesDefaultBucketAndIsMergedAndCleared(t *testing.T) {
	sandbox(t)
	Add("", "Sca", 1, 1) // empty id → shared default bucket
	if got := Load("S1")["Sca"].VulnerabilitiesFound; got != 1 {
		t.Errorf("Load should merge the default bucket: got %d", got)
	}
	Clear("S1") // Clear removes the default bucket too
	if n := len(Load("other")); n != 0 {
		t.Errorf("default bucket not cleared: got %d engines", n)
	}
}

func TestMissingHomeIsNoOpNoPanic(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	// os.UserHomeDir fails → every function must no-op without panicking.
	Add("S1", "Asca", 1, 1)
	if n := len(Load("S1")); n != 0 {
		t.Errorf("Load with no home should be empty, got %d", n)
	}
	Clear("S1")
}

func TestHostileIDStaysInsideCheckmarxDir(t *testing.T) {
	sandbox(t)
	id := "../../etc/passwd weird/id"
	Add(id, "Asca", 1, 1)
	if Load(id)["Asca"].VulnerabilitiesFound != 1 {
		t.Errorf("sanitized id round-trip failed")
	}
	dir, ok := baseDir()
	if !ok {
		t.Fatal("baseDir unavailable")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected a tally file inside %s (err=%v)", dir, err)
	}
	// The traversal must not have created a file outside ~/.checkmarx.
	if _, err := os.Stat("etc/passwd weird"); err == nil {
		t.Errorf("hostile id escaped the sandbox directory")
	}
}

func TestMalformedLinesSkipped(t *testing.T) {
	sandbox(t)
	Add("S1", "Asca", 1, 1)
	path, ok := tallyPath("S1")
	if !ok {
		t.Fatal("no tally path")
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("not json\n{}\n{\"engine\":\"\",\"found\":9}\n")
	_ = f.Close()

	if got := Load("S1")["Asca"].VulnerabilitiesFound; got != 1 {
		t.Errorf("malformed lines should be skipped, Asca=%d want 1", got)
	}
}
