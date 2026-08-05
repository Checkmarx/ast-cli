//go:build !integration

package sessiontally

import (
	"os"
	"path/filepath"
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

// Add is currently disabled (see its doc comment): these tests assert that no matter what is passed
// in, Add never writes a tally file under ~/.checkmarx and Load/Clear stay safe no-ops as a result.

func TestAddIsNoOpAndWritesNoFile(t *testing.T) {
	sandbox(t)
	Add("S1", "Asca", 2, 2)
	Add("S1", "Sca", 3, 1)
	Add("", "Sca", 1, 1)

	if got := Load("S1"); len(got) != 0 {
		t.Errorf("expected no tallies recorded, got %+v", got)
	}

	dir, ok := baseDir()
	if !ok {
		t.Fatal("baseDir unavailable")
	}
	if _, err := os.Stat(dir); err == nil {
		entries, _ := os.ReadDir(dir)
		if len(entries) != 0 {
			t.Errorf("expected no files created under %s, got %v", dir, entries)
		}
	}
}

func TestConcurrentAddStillNoOp(t *testing.T) {
	sandbox(t)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); Add("S1", "Asca", 1, 0) }()
	}
	wg.Wait()
	if got := Load("S1")["Asca"].VulnerabilitiesFound; got != 0 {
		t.Errorf("Add is disabled, expected 0 got %d", got)
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
	if got := Load(id)["Asca"].VulnerabilitiesFound; got != 0 {
		t.Errorf("Add is disabled, expected 0 got %d", got)
	}
	// The traversal must not have created a file outside ~/.checkmarx.
	if _, err := os.Stat("etc/passwd weird"); err == nil {
		t.Errorf("hostile id escaped the sandbox directory")
	}
}

func TestMalformedLinesSkipped(t *testing.T) {
	sandbox(t)
	// Add is disabled, so write the tally file directly to exercise Load's fold/skip logic.
	path, ok := tallyPath("S1")
	if !ok {
		t.Fatal("no tally path")
	}
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, filePerm)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("not json\n{}\n{\"engine\":\"\",\"found\":9}\n{\"engine\":\"Asca\",\"found\":1,\"remOffered\":1}\n")
	_ = f.Close()

	if got := Load("S1")["Asca"].VulnerabilitiesFound; got != 1 {
		t.Errorf("malformed lines should be skipped, Asca=%d want 1", got)
	}
}
