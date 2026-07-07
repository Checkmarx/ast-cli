package wrappers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/viper"
)

// withTempConfigDir points viper at a temp directory for the duration of one
// test, so the session_global helpers operate on a sandbox rather than the
// real user's ~/.checkmarx. Restores prior state via t.Cleanup.
func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev := viper.GetString(params.ConfigFilePathKey)
	viper.Set(params.ConfigFilePathKey, filepath.Join(dir, "checkmarxcli.yaml"))
	t.Cleanup(func() {
		viper.Set(params.ConfigFilePathKey, prev)
	})
	return dir
}

func TestSessionGlobalFilePath_ReturnsPathInConfigDir(t *testing.T) {
	dir := withTempConfigDir(t)
	got, err := SessionGlobalFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, params.SessionGlobalFileName)
	if got != want {
		t.Errorf("SessionGlobalFilePath() = %q, want %q", got, want)
	}
}

func TestReadSessionGlobal_ReturnsEmptyWhenFileMissing(t *testing.T) {
	withTempConfigDir(t)
	got, err := ReadSessionGlobal()
	if err != nil {
		t.Fatalf("expected nil error when file does not exist, got: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestWriteAndReadSessionGlobal_RoundTrip(t *testing.T) {
	withTempConfigDir(t)
	token := "eyJhbGc.test-refresh-token.xyz"
	if err := WriteSessionGlobal(token); err != nil {
		t.Fatalf("WriteSessionGlobal failed: %v", err)
	}
	got, err := ReadSessionGlobal()
	if err != nil {
		t.Fatalf("ReadSessionGlobal failed: %v", err)
	}
	if got != token {
		t.Errorf("round-trip mismatch: wrote %q, read %q", token, got)
	}
}

func TestReadSessionGlobal_TrimsWhitespace(t *testing.T) {
	dir := withTempConfigDir(t)
	// Write a token with trailing newline directly to disk to simulate a file
	// edited by hand.
	path := filepath.Join(dir, params.SessionGlobalFileName)
	if err := os.WriteFile(path, []byte("the-token\n"), 0o600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	got, err := ReadSessionGlobal()
	if err != nil {
		t.Fatalf("ReadSessionGlobal failed: %v", err)
	}
	if got != "the-token" {
		t.Errorf("expected trailing whitespace trimmed, got %q", got)
	}
}

func TestClearSessionGlobal_RemovesFile(t *testing.T) {
	withTempConfigDir(t)
	if err := WriteSessionGlobal("some-token"); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	if err := ClearSessionGlobal(); err != nil {
		t.Fatalf("ClearSessionGlobal failed: %v", err)
	}
	got, err := ReadSessionGlobal()
	if err != nil {
		t.Fatalf("ReadSessionGlobal after clear failed: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty after clear, got %q", got)
	}
}

func TestClearSessionGlobal_IdempotentWhenFileMissing(t *testing.T) {
	withTempConfigDir(t)
	// File never created; clearing it should not error.
	if err := ClearSessionGlobal(); err != nil {
		t.Errorf("expected nil error when file does not exist, got: %v", err)
	}
}

func TestLoadActiveCredential_GlobalModeLoadsFile(t *testing.T) {
	withTempConfigDir(t)
	t.Setenv(params.AstAPIKeyEnv, "")
	if err := WriteSessionGlobal("global-token"); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	if err := WriteActiveMode(params.SessionGlobalValue); err != nil {
		t.Fatalf("WriteActiveMode failed: %v", err)
	}
	viper.Set(params.AstAPIKey, "")
	LoadActiveCredential()
	if got := viper.GetString(params.AstAPIKey); got != "global-token" {
		t.Errorf("expected global mode to load token into viper, got %q", got)
	}
}

func TestLoadActiveCredential_GlobalOverridesStaleEnv(t *testing.T) {
	withTempConfigDir(t)
	t.Setenv(params.AstAPIKeyEnv, "stale-env-token")
	if err := WriteSessionGlobal("global-token"); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	if err := WriteActiveMode(params.SessionGlobalValue); err != nil {
		t.Fatalf("WriteActiveMode failed: %v", err)
	}
	viper.Set(params.AstAPIKey, "")
	LoadActiveCredential()
	if got := viper.GetString(params.AstAPIKey); got != "global-token" {
		t.Errorf("global mode must win over stale env, got %q", got)
	}
}

func TestLoadActiveCredential_LocalModeNoOpLetsEnvWin(t *testing.T) {
	withTempConfigDir(t)
	t.Setenv(params.AstAPIKeyEnv, "local-token")
	if err := WriteActiveMode(params.SessionLocalValue); err != nil {
		t.Fatalf("WriteActiveMode failed: %v", err)
	}
	viper.Set(params.AstAPIKey, "")
	LoadActiveCredential()
	// We don't viper.Set for local mode — env binding does the work.
	// Verify that we didn't overwrite anything.
	// (We can't easily verify env-binding inside this test without
	// going through viper, so just confirm no error and no surprise Set.)
	if got := viper.GetString(params.AstAPIKey); got != "" && got != "local-token" {
		t.Errorf("local mode should not viper.Set anything; got unexpected %q", got)
	}
}

func TestLoadActiveCredential_NoActiveModeIsNoOp(t *testing.T) {
	withTempConfigDir(t)
	// No WriteActiveMode call — file is absent.
	viper.Set(params.AstAPIKey, "")
	LoadActiveCredential()
	if got := viper.GetString(params.AstAPIKey); got != "" {
		t.Errorf("expected viper.AstAPIKey to remain empty when no active mode, got %q", got)
	}
}
