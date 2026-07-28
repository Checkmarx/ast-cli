//go:build !integration

package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// The full runAuthLogin (browser + network) is out of scope; these cover the
// deterministic pieces: persistYamlLogin and runAuthLogout.

// withTempConfigDir sandboxes viper at a temp config file and clears CX_APIKEY.
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

// readYamlAPIKey reads cx_apikey directly from the sandbox yaml file.
func readYamlAPIKey(t *testing.T) string {
	t.Helper()
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		t.Fatalf("GetConfigFilePath failed: %v", err)
	}
	yamlConfig, err := configuration.LoadConfig(configPath)
	if err != nil {
		return ""
	}
	if v, ok := yamlConfig[params.AstAPIKey].(string); ok {
		return v
	}
	return ""
}

// Token must be saved to yaml but never echoed to stdout.
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
	if !strings.Contains(stdout, "Successfully authenticated to Checkmarx One server!") {
		t.Errorf("expected confirmation line, got: %q", stdout)
	}
	if got := readYamlAPIKey(t); got != token {
		t.Errorf("expected token persisted to yaml, got %q", got)
	}
}

// Prompt is skipped only when a connection detail is passed as a flag; with no
// flags login always prompts (parity with cx configure, incl. re-login after logout).
func TestConnectionFlagsProvided(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"no flags", []string{}, false},
		{"base-uri", []string{"--base-uri", "https://x"}, true},
		{"base-auth-uri", []string{"--base-auth-uri", "https://x"}, true},
		{"tenant", []string{"--tenant", "t"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "login", RunE: func(*cobra.Command, []string) error { return nil }}
			cmd.Flags().String(params.BaseURIFlag, "", "")
			cmd.Flags().String(params.BaseAuthURIFlag, "", "")
			cmd.Flags().String(params.TenantFlag, "", "")
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute: %v", err)
			}
			if got := connectionFlagsProvided(cmd); got != tc.want {
				t.Errorf("connectionFlagsProvided() = %v, want %v", got, tc.want)
			}
		})
	}
}

// Logout clears cx_apikey and is idempotent.
func TestRunAuthLogout_ClearsYaml(t *testing.T) {
	dir := withTempConfigDir(t)
	configPath := filepath.Join(dir, "checkmarxcli.yaml")
	if err := configuration.SafeWriteSingleConfigKeyString(configPath, params.AstAPIKey, "stored-token"); err != nil {
		t.Fatalf("setup yaml write failed: %v", err)
	}

	cmd, out, _ := newBufferedCmd()
	if err := runAuthLogout(cmd, nil); err != nil {
		t.Fatalf("runAuthLogout failed: %v", err)
	}
	if got := readYamlAPIKey(t); got != "" {
		t.Errorf("expected yaml cx_apikey cleared, got %q", got)
	}
	if !strings.Contains(out.String(), "Successfully logged out of Checkmarx One server!") {
		t.Errorf("expected logout confirmation, got: %q", out.String())
	}

	// Idempotent: running again on empty storage must not error.
	if err := runAuthLogout(cmd, nil); err != nil {
		t.Fatalf("second runAuthLogout failed: %v", err)
	}
}
