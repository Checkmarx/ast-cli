//go:build !integration

package secrets

import (
	"os"
	"path/filepath"
	"testing"

	agenthooks "github.com/Checkmarx/ast-cx-hooks"
)

const (
	testConfigEnv  = "config.env"
	debugFalse     = "DEBUG=false"
	debugTrue      = "DEBUG=true"
	placeholderKey = "API_KEY=placeholder"
)

func TestProposedContent_FullFileWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.env")
	changes := []agenthooks.FileDiff{{Before: "", After: "TOKEN=abc"}}
	newContent, originalContent, ok := proposedContent(path, changes)
	if !ok {
		t.Fatal("full-file write should always resolve")
	}
	if newContent != "TOKEN=abc" || originalContent != "" {
		t.Fatalf("newContent=%q originalContent=%q", newContent, originalContent)
	}
}

func TestProposedContent_UniqueMatch_Applies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, testConfigEnv)
	if err := os.WriteFile(path, []byte(debugFalse), 0o600); err != nil {
		t.Fatal(err)
	}
	changes := []agenthooks.FileDiff{{Before: debugFalse, After: debugTrue}}
	newContent, _, ok := proposedContent(path, changes)
	if !ok {
		t.Fatal("unique match should resolve")
	}
	if newContent != debugTrue {
		t.Fatalf("newContent = %q, want %s", newContent, debugTrue)
	}
}

func TestProposedContent_DuplicateBeforeText_IsUnresolved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, testConfigEnv)
	body := placeholderKey + "\n" + debugTrue + "\n" + placeholderKey
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	changes := []agenthooks.FileDiff{{Before: placeholderKey, After: "API_KEY=sk-live-123"}}
	_, _, ok := proposedContent(path, changes)
	if ok {
		t.Fatal("a Before string that matches more than once must not be guessed at")
	}
}

func TestProposedContent_BeforeTextMissing_IsUnresolved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, testConfigEnv)
	if err := os.WriteFile(path, []byte(debugTrue), 0o600); err != nil {
		t.Fatal(err)
	}
	changes := []agenthooks.FileDiff{{Before: "STALE_TEXT", After: "TOKEN=sk-live-123"}}
	newContent, _, ok := proposedContent(path, changes)
	if ok {
		t.Fatal("a Before string that is not found must not be silently skipped")
	}
	if newContent != "" {
		t.Fatalf("newContent should be empty when unresolved, got %q", newContent)
	}
}
