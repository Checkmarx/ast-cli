//go:build !integration

package commands

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/credentialstore"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// The full runAuthLogin (browser + network) is out of scope; these cover the
// deterministic pieces: persistLogin and runAuthLogout.

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

// swapCredentialResolver binds a mock-backed resolver to the sandbox config
// path so credential reads/writes never reach the real OS keyring.
func swapCredentialResolver(t *testing.T) *mock.CredentialStoreMock {
	t.Helper()
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		t.Fatalf("GetConfigFilePath failed: %v", err)
	}
	store := mock.NewCredentialStoreMock()
	credentialstore.SetDefaultResolverForTest(credentialstore.NewResolver(configPath, credentialstore.PolicyAuto, store))
	t.Cleanup(credentialstore.ResetForTest)
	return store
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
	swapCredentialResolver(t)
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

// Logout does not clear OAuth2 client credentials - they are intentionally left alone.
func TestRunAuthLogout_DoesNotClearClientCredentials(t *testing.T) {
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
	if got := readYamlKey(t, params.AccessKeyIDConfigKey); got != "stored-client-id" {
		t.Errorf("expected yaml cx_client_id preserved, got %q", got)
	}
	if got := readYamlKey(t, params.AccessKeySecretConfigKey); got != "stored-client-secret" {
		t.Errorf("expected yaml cx_client_secret preserved, got %q", got)
	}
}

// persistLogin saves the refresh token to the credential store.
func TestPersistLogin_SavesTokenAndPrintsSuccess(t *testing.T) {
	_ = withTempConfigDir(t)
	store := swapCredentialResolver(t)
	cmd, out, _ := newBufferedCmd()
	refreshToken := "refresh-token-abc123"

	if err := persistLogin(cmd, refreshToken); err != nil {
		t.Fatalf("persistLogin failed: %v", err)
	}

	if got := store.Store[credentialstore.CredentialAPIKey]; got != refreshToken {
		t.Errorf("expected token saved to credential store, got %q want %q", got, refreshToken)
	}
	if got := readYamlAPIKey(t); got != "" {
		t.Errorf("expected legacy yaml entry scrubbed, got %q", got)
	}

	if !strings.Contains(out.String(), "Successfully authenticated to Checkmarx One server!") {
		t.Errorf("expected success message, got: %q", out.String())
	}
}

// persistLogin does not echo the token to stdout
func TestPersistLogin_DoesNotEchoToken(t *testing.T) {
	_ = withTempConfigDir(t)
	cmd, out, _ := newBufferedCmd()
	refreshToken := "secret-refresh-token-12345"

	if err := persistLogin(cmd, refreshToken); err != nil {
		t.Fatalf("persistLogin failed: %v", err)
	}

	output := out.String()
	if strings.Contains(output, refreshToken) {
		t.Errorf("token should not be echoed to stdout, but got: %q", output)
	}
}

// persistLogin handles different token formats
func TestPersistLogin_DifferentTokenFormats(t *testing.T) {
	testTokens := []string{
		"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
		"simple-token",
		"token-with-special-chars-!@#$%^&*()",
	}

	for _, token := range testTokens {
		t.Run("token format", func(t *testing.T) {
			_ = withTempConfigDir(t)
			store := swapCredentialResolver(t)
			cmd, _, _ := newBufferedCmd()

			if err := persistLogin(cmd, token); err != nil {
				t.Fatalf("persistLogin failed for token %q: %v", token, err)
			}

			if got := store.Store[credentialstore.CredentialAPIKey]; got != token {
				t.Errorf("token mismatch for %q: got %q", token, got)
			}
		})
	}
}

// persistLogin prints success message to stdout
func TestPersistLogin_PrintsSuccessMessage(t *testing.T) {
	_ = withTempConfigDir(t)
	cmd, out, _ := newBufferedCmd()
	refreshToken := "test-token-456"

	if err := persistLogin(cmd, refreshToken); err != nil {
		t.Fatalf("persistLogin failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Successfully authenticated to Checkmarx One server!") {
		t.Errorf("expected success message in output, got: %q", output)
	}
}

func TestCompleteAuthLogin_TelemetryAgentAllowlist(t *testing.T) {
	const credential = "test-login-credential"
	cases := []struct {
		name  string
		agent *string
		want  string
	}{
		{name: "omitted"},
		{name: "blank", agent: testLoginAgent(" ")},
		{name: "default", agent: testLoginAgent("ASTCLI")},
		{name: "other", agent: testLoginAgent("Copilot-VSCode")},
		{name: "copilot", agent: testLoginAgent(" Copilot-CLI "), want: "Copilot-cli"},
		{name: "claude", agent: testLoginAgent("claude-cli"), want: "Claude-cli"},
		{name: "cursor", agent: testLoginAgent("cursor-cli"), want: "Cursor-cli"},
		{name: "gemini", agent: testLoginAgent("gemini-cli"), want: "Gemini-cli"},
		{name: "codex", agent: testLoginAgent("codex-cli"), want: "Codex-cli"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTempConfigDir(t)
			store := swapCredentialResolver(t)
			t.Setenv("CX_TELEMETRY_DISABLE", "")
			oldExpiry, oldSession := viper.Get(params.TokenExpirySecondsKey), viper.Get(params.SessionIDConfigKey)
			t.Cleanup(func() {
				viper.Set(params.TokenExpirySecondsKey, oldExpiry)
				viper.Set(params.SessionIDConfigKey, oldSession)
				wrappers.InvalidateAccessTokenCache()
			})
			viper.Set(params.TokenExpirySecondsKey, 120)
			viper.Set(params.SessionIDConfigKey, "")
			cmd, _, _ := newBufferedCmd()
			cmd.Flags().String(params.AgentFlag, params.DefaultAgent, "")
			if tc.agent != nil {
				if err := cmd.Flags().Set(params.AgentFlag, *tc.agent); err != nil {
					t.Fatal(err)
				}
			}
			calls := 0
			telemetry := mock.TelemetryMockWrapper{CustomSendAIDataToLog: func(data *wrappers.DataForAITelemetry) error {
				calls++
				if store.Store[credentialstore.CredentialAPIKey] != credential {
					t.Fatal("telemetry sent before credential was saved")
				}
				if data.Agent != tc.want || data.Type != authenticationEventType ||
					data.SubType != authenticationEventType || data.Status != telemetrySuccessStatus ||
					data.SessionID == "" || (*data.EventAttributes)["authMethod"] != "OAuth" {
					t.Fatalf("unexpected authentication telemetry: %+v", data)
				}
				return errors.New("telemetry unavailable")
			}}
			if err := completeAuthLogin(cmd, telemetry, credential, "test-access-token", "OAuth"); err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if calls != 0 || viper.GetString(params.SessionIDConfigKey) != "" {
					t.Fatal("non-eligible login sent telemetry or generated a session ID")
				}
			} else if calls != 1 || readYamlKey(t, params.SessionIDConfigKey) == "" {
				t.Fatal("eligible login should send one event and persist a session ID")
			}
			if store.Store[credentialstore.CredentialAPIKey] != credential {
				t.Fatal("login credential was not saved")
			}
		})
	}
}

func testLoginAgent(agent string) *string {
	return &agent
}

func TestRunAuthLogin_InvalidAPIKeyKeepsCredential(t *testing.T) {
	withTempConfigDir(t)
	store := swapCredentialResolver(t)
	const oldCredential = "existing-credential"
	store.Store[credentialstore.CredentialAPIKey] = oldCredential
	cmd, _, _ := newBufferedCmd()
	cmd.Flags().String(params.AstAPIKeyFlag, "", "")
	if err := cmd.Flags().Set(params.AstAPIKeyFlag, "not-a-token"); err != nil {
		t.Fatal(err)
	}
	if err := runAuthLogin(cmd, nil); err == nil {
		t.Fatal("expected API-key verification to fail")
	}
	if store.Store[credentialstore.CredentialAPIKey] != oldCredential {
		t.Fatal("invalid API key replaced the existing credential")
	}
}
