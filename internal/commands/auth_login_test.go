//go:build !integration

package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Full runAuthLogin coverage is out of scope for unit tests: it opens a browser
// and runs the PKCE network exchange (wrappers.LoginWithPKCE). These tests cover
// the deterministic, network-free pieces — the persist* writers, clearFileStorages,
// nukeAllStorages' env-safety, and the universal logout — which is also where the
// security fixes (#4 no token to stdout, #5 env token never revoked) live.

// withTempConfigDir points viper at a temp config file for one test so the auth
// storage helpers operate on a sandbox instead of the real ~/.checkmarx. Also
// clears CX_APIKEY so env never shadows the sandbox.
func withTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev := viper.GetString(params.ConfigFilePathKey)
	viper.Set(params.ConfigFilePathKey, filepath.Join(dir, "checkmarxcli.yaml"))
	t.Setenv(params.AstAPIKeyEnv, "")
	t.Cleanup(func() { viper.Set(params.ConfigFilePathKey, prev) })
	return dir
}

// newBufferedCmd returns a cobra command whose stdout/stderr are captured.
func newBufferedCmd() (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	cmd := &cobra.Command{}
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	return cmd, &out, &errOut
}

// TestPersistYamlLogin_DoesNotPrintToken locks in fix #4: the refresh token must
// be saved to the yaml file but never echoed to stdout (it would leak into shell
// history / CI logs).
func TestPersistYamlLogin_DoesNotPrintToken(t *testing.T) {
	withTempConfigDir(t)
	const token = "super-secret-refresh-token"

	cmd, out, _ := newBufferedCmd()
	if err := persistYamlLogin(cmd, token); err != nil {
		t.Fatalf("persistYamlLogin failed: %v", err)
	}

	stdout := out.String()
	if strings.Contains(stdout, token) {
		t.Errorf("refresh token leaked to stdout: %q", stdout)
	}
	if !strings.Contains(stdout, "Authenticated. Token saved to") {
		t.Errorf("expected confirmation line, got: %q", stdout)
	}
	// Token must actually be persisted to the yaml file.
	if got := wrappers.ReadYamlAPIKey(); got != token {
		t.Errorf("expected token persisted to yaml, got %q", got)
	}
	if mode, _ := wrappers.ReadActiveMode(); mode != params.SessionYamlValue {
		t.Errorf("expected active mode %q, got %q", params.SessionYamlValue, mode)
	}
}

// TestPersistGlobalLogin_WritesFileAndNoToken: global mode persists to the global
// session file and prints only the path — never the token.
func TestPersistGlobalLogin_WritesFileAndNoToken(t *testing.T) {
	withTempConfigDir(t)
	const token = "global-refresh-token"

	cmd, out, _ := newBufferedCmd()
	if err := persistGlobalLogin(cmd, token); err != nil {
		t.Fatalf("persistGlobalLogin failed: %v", err)
	}

	if strings.Contains(out.String(), token) {
		t.Errorf("refresh token leaked to stdout: %q", out.String())
	}
	if got, err := wrappers.ReadSessionGlobal(); err != nil || got != token {
		t.Errorf("expected token in global session file, got %q (err=%v)", got, err)
	}
	if mode, _ := wrappers.ReadActiveMode(); mode != params.SessionGlobalValue {
		t.Errorf("expected active mode %q, got %q", params.SessionGlobalValue, mode)
	}
}

// TestPersistLocalLogin_EmitsShellEval: local mode intentionally emits a single
// shell-evaluable line (reset + assignment) to stdout — the token IS present
// there by design (it lives only in the shell env).
func TestPersistLocalLogin_EmitsShellEval(t *testing.T) {
	withTempConfigDir(t)
	const token = "local-refresh-token"

	cmd, out, errOut := newBufferedCmd()
	if err := persistLocalLogin(cmd, token); err != nil {
		t.Fatalf("persistLocalLogin failed: %v", err)
	}

	stdout := out.String()
	if !strings.Contains(stdout, params.AstAPIKeyEnv) {
		t.Errorf("expected env-var name in stdout, got: %q", stdout)
	}
	if !strings.Contains(stdout, token) {
		t.Errorf("local mode must emit the token for eval, got: %q", stdout)
	}
	if !strings.Contains(errOut.String(), "Authenticated") {
		t.Errorf("expected info line on stderr, got: %q", errOut.String())
	}
	if mode, _ := wrappers.ReadActiveMode(); mode != params.SessionLocalValue {
		t.Errorf("expected active mode %q, got %q", params.SessionLocalValue, mode)
	}
}

// TestClearFileStorages_ClearsYamlAndGlobal: clearing empties the yaml cx_apikey
// and removes the global session file.
func TestClearFileStorages_ClearsYamlAndGlobal(t *testing.T) {
	dir := withTempConfigDir(t)
	configPath := filepath.Join(dir, "checkmarxcli.yaml")
	if err := configuration.SafeWriteSingleConfigKeyString(configPath, params.AstAPIKey, "yaml-token"); err != nil {
		t.Fatalf("setup yaml write failed: %v", err)
	}
	if err := wrappers.WriteSessionGlobal("global-token"); err != nil {
		t.Fatalf("setup global write failed: %v", err)
	}

	clearFileStorages()

	if got := wrappers.ReadYamlAPIKey(); got != "" {
		t.Errorf("expected yaml cx_apikey cleared, got %q", got)
	}
	if got, _ := wrappers.ReadSessionGlobal(); got != "" {
		t.Errorf("expected global session cleared, got %q", got)
	}
}

// TestNukeAllStorages_DoesNotRevokeOrClearEnv locks in fix #5: an env-var token is
// neither cleared nor touched by the nuke (the CLI can't clear a parent shell's
// env, and it is often a deliberate CI credential). With empty file storages this
// makes no network call.
func TestNukeAllStorages_DoesNotRevokeOrClearEnv(t *testing.T) {
	withTempConfigDir(t)
	const envToken = "ci-env-refresh-token"
	t.Setenv(params.AstAPIKeyEnv, envToken)

	// No yaml or global token present → no revocation network call is attempted.
	nukeAllStorages(defaultLoginClientID)

	// The env var must remain exactly as the caller set it.
	if got := os.Getenv(params.AstAPIKeyEnv); got != envToken {
		t.Errorf("nukeAllStorages must not alter the env token: got %q, want %q", got, envToken)
	}
}

// TestRunAuthLogout_EmptyStorage emits a shell-clear of CX_APIKEY, clears the
// active mode, and makes no network call when no token is stored.
func TestRunAuthLogout_EmptyStorage(t *testing.T) {
	withTempConfigDir(t)
	if err := wrappers.WriteActiveMode(params.SessionYamlValue); err != nil {
		t.Fatalf("setup active mode failed: %v", err)
	}

	cmd, out, errOut := newBufferedCmd()
	if err := runAuthLogout(cmd, nil); err != nil {
		t.Fatalf("runAuthLogout failed: %v", err)
	}

	if !strings.Contains(out.String(), params.AstAPIKeyEnv) {
		t.Errorf("expected shell-clear of %s on stdout, got: %q", params.AstAPIKeyEnv, out.String())
	}
	if !strings.Contains(errOut.String(), "Logged out") {
		t.Errorf("expected logout info on stderr, got: %q", errOut.String())
	}
	if mode, _ := wrappers.ReadActiveMode(); mode != "" {
		t.Errorf("expected active mode cleared after logout, got %q", mode)
	}
}
