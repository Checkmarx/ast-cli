package commands

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/credentialstore"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear the stored Checkmarx One credential",
		Long: "Clears the stored Checkmarx One credentials: the login token / API key " +
			"(cx_apikey) and the OAuth2 client credentials (cx_client_secret + cx_client_id), " +
			"from both the OS keyring and the config file. Idempotent — running it when no " +
			"credential is stored is a no-op. Credentials provided via the CX_APIKEY or " +
			"CX_CLIENT_ID/CX_CLIENT_SECRET environment variables are not affected.",
		Example: heredoc.Doc(`
			$ cx auth logout
		`),
		RunE: runAuthLogout,
	}
}

// runAuthLogout clears both stored secrets (cx_apikey, cx_client_secret) and blanks
// the plaintext cx_client_id; connection settings and env credentials are left alone.
func runAuthLogout(cmd *cobra.Command, _ []string) error {
	for _, key := range []string{params.AstAPIKey, params.AccessKeySecretConfigKey} {
		if err := credentialstore.Default.DeleteSecret(key); err != nil {
			return errors.Wrap(err, "failed to clear stored credential")
		}
	}
	// Blank the non-secret client id best-effort: the secrets are already cleared, so a
	// yaml write failure here must not fail logout.
	if configPath, err := configuration.GetConfigFilePath(); err == nil {
		if wErr := configuration.SafeWriteSingleConfigKeyString(configPath, params.AccessKeyIDConfigKey, ""); wErr != nil {
			logger.PrintIfVerbose(fmt.Sprintf("failed to clear client id: %v", wErr))
		}
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Successfully logged out of Checkmarx One server!")
	return nil
}
