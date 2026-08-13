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
// deterministic pieces: persistLogin and runAuthLogout.

// swapDefaultStore swaps credentialstore.Default for a mock and restores it.

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

// readYamlKey reads any key directly from the sandbox yaml file.
func readYamlKey(t *testing.T, key string) string {
	t.Helper()
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		t.Fatalf("GetConfigFilePath failed: %v", err)
	}
	yamlConfig, err := configuration.LoadConfig(configPath)
	if err != nil {
		return ""
	}
	if v, ok := yamlConfig[key].(string); ok {
		return v
	}
	return ""
}

// readYamlAPIKey reads cx_apikey directly from the sandbox yaml file.
func readYamlAPIKey(t *testing.T) string {
	t.Helper()
	return readYamlKey(t, params.AstAPIKey)
}

// Token must be saved to the yaml fallback but never echoed to stdout.

// persistLogin stores the token through the credential store (keyring in prod).

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

// Logout clears the OAuth2 client credentials (secret + id) so no half-config lingers.
func TestRunAuthLogout_ClearsClientCredentials(t *testing.T) {
	dir := withTempConfigDir(t)
	configPath := filepath.Join(dir, "checkmarxcli.yaml")
	if err := configuration.SafeWriteSingleConfigKeyString(configPath, params.AccessKeyIDConfigKey, "stored-client-id"); err != nil {
		t.Fatalf("setup client id write failed: %v", err)
	}
	if err := configuration.SafeWriteSingleConfigKeyString(configPath, params.AccessKeySecretConfigKey, "stored-client-secret"); err != nil {
		t.Fatalf("setup client secret write failed: %v", err)
	}

	cmd, _, _ := newBufferedCmd()
	if err := runAuthLogout(cmd, nil); err != nil {
		t.Fatalf("runAuthLogout failed: %v", err)
	}
	if got := readYamlKey(t, params.AccessKeyIDConfigKey); got != "" {
		t.Errorf("expected yaml cx_client_id cleared, got %q", got)
	}
	if got := readYamlKey(t, params.AccessKeySecretConfigKey); got != "" {
		t.Errorf("expected yaml cx_client_secret cleared, got %q", got)
	}
}
