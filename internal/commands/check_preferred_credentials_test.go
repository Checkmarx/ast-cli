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

// An explicit --apikey flag sets the preferred credential type to "apikey".
func TestCheckPreferredCredentials_APIKeyFlagWins(t *testing.T) {
	cmd := newCredCmd(t, "--apikey", "flag-value")
	CheckPreferredCredentials(cmd)

	if got := viper.GetString(params.PreferredCredentialTypeKey); got != "apikey" {
		t.Errorf("expected preferred type to be apikey, got %q", got)
	}
}

// An explicit --client-secret flag (with --client-id) sets the preferred credential type to "oauth".
func TestCheckPreferredCredentials_ClientSecretFlagWins(t *testing.T) {
	cmd := newCredCmd(t, "--client-id", "flag-id", "--client-secret", "flag-secret")
	CheckPreferredCredentials(cmd)

	if got := viper.GetString(params.PreferredCredentialTypeKey); got != "oauth" {
		t.Errorf("expected preferred type to be oauth, got %q", got)
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
