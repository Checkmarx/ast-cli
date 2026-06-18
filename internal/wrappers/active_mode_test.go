package wrappers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
)

func TestActiveModeFilePath_ReturnsPathInConfigDir(t *testing.T) {
	dir := withTempConfigDir(t)
	got, err := ActiveModeFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, params.ActiveModeFileName)
	if got != want {
		t.Errorf("ActiveModeFilePath() = %q, want %q", got, want)
	}
}

func TestReadActiveMode_EmptyWhenFileMissing(t *testing.T) {
	withTempConfigDir(t)
	got, err := ReadActiveMode()
	if err != nil {
		t.Fatalf("expected nil error when file absent, got: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestWriteAndReadActiveMode_RoundTrip(t *testing.T) {
	withTempConfigDir(t)
	for _, mode := range []string{params.SessionYamlValue, params.SessionLocalValue, params.SessionGlobalValue} {
		if err := WriteActiveMode(mode); err != nil {
			t.Fatalf("WriteActiveMode(%q) failed: %v", mode, err)
		}
		got, err := ReadActiveMode()
		if err != nil {
			t.Fatalf("ReadActiveMode after writing %q failed: %v", mode, err)
		}
		if got != mode {
			t.Errorf("round-trip mismatch: wrote %q, read %q", mode, got)
		}
	}
}

func TestWriteActiveMode_RejectsInvalidValue(t *testing.T) {
	withTempConfigDir(t)
	err := WriteActiveMode("invalid-mode-value")
	if err == nil {
		t.Fatal("expected error for invalid mode value, got nil")
	}
}

func TestReadActiveMode_TreatsCorruptValueAsAbsent(t *testing.T) {
	dir := withTempConfigDir(t)
	// Manually write garbage to the active-mode file.
	if err := os.WriteFile(filepath.Join(dir, params.ActiveModeFileName), []byte("garbage-mode"), 0o600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	got, err := ReadActiveMode()
	if err != nil {
		t.Fatalf("ReadActiveMode returned unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected corrupt value to be treated as absent (empty), got %q", got)
	}
}

func TestClearActiveMode_RemovesFile(t *testing.T) {
	withTempConfigDir(t)
	if err := WriteActiveMode(params.SessionGlobalValue); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	if err := ClearActiveMode(); err != nil {
		t.Fatalf("ClearActiveMode failed: %v", err)
	}
	got, err := ReadActiveMode()
	if err != nil {
		t.Fatalf("ReadActiveMode after clear failed: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty after clear, got %q", got)
	}
}

func TestClearActiveMode_IdempotentWhenFileMissing(t *testing.T) {
	withTempConfigDir(t)
	if err := ClearActiveMode(); err != nil {
		t.Errorf("expected nil error when file absent, got: %v", err)
	}
}
