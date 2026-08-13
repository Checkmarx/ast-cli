//go:build !integration

package commands

import (
	"testing"

	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newCredCmd builds a cobra command carrying the credential flags and parses args.
func newCredCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "x", RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.Flags().String(params.AstAPIKeyFlag, "", "")
	cmd.Flags().String(params.AccessKeySecretFlag, "", "")
	cmd.Flags().String(params.AccessKeyIDFlag, "", "")
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return cmd
}

// An explicit --apikey flag must win over a startup-loaded stored secret.
func TestCheckPreferredCredentials_APIKeyFlagWins(t *testing.T) {
	viper.Set(params.AstAPIKey, "stored")
	t.Cleanup(func() { viper.Set(params.AstAPIKey, "") })

	cmd := newCredCmd(t, "--apikey", "flag-value")
	CheckPreferredCredentials(cmd)

	if got := viper.GetString(params.AstAPIKey); got != "flag-value" {
		t.Errorf("expected flag to win, got %q", got)
	}
}

// An explicit --client-secret flag must win over a startup-loaded stored secret.
func TestCheckPreferredCredentials_ClientSecretFlagWins(t *testing.T) {
	viper.Set(params.AccessKeySecretConfigKey, "stored")
	t.Cleanup(func() { viper.Set(params.AccessKeySecretConfigKey, "") })

	cmd := newCredCmd(t, "--client-secret", "flag-secret")
	CheckPreferredCredentials(cmd)

	if got := viper.GetString(params.AccessKeySecretConfigKey); got != "flag-secret" {
		t.Errorf("expected flag to win, got %q", got)
	}
}

// With no secret flags, the stored value is untouched.
func TestCheckPreferredCredentials_NoFlagKeepsStored(t *testing.T) {
	viper.Set(params.AstAPIKey, "stored")
	t.Cleanup(func() { viper.Set(params.AstAPIKey, "") })

	cmd := newCredCmd(t)
	CheckPreferredCredentials(cmd)

	if got := viper.GetString(params.AstAPIKey); got != "stored" {
		t.Errorf("expected stored value kept, got %q", got)
	}
}
