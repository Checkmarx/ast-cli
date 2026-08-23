package commands

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/credentialstore"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear the stored Checkmarx One credential",
		Long: "Clears the cx_apikey stored in the config file. Idempotent — running it when no " +
			"credential is stored is a no-op. Credentials provided via the CX_APIKEY or " +
			"CX_CLIENT_ID/CX_CLIENT_SECRET environment variables are not affected.",
		Example: heredoc.Doc(`
			$ cx auth logout
		`),
		RunE: runAuthLogout,
	}
}

// runAuthLogout clears the stored api-key credential (keyring and any leftover
// plaintext config-file entry). The client-credentials and env-provided credentials are
// intentionally left alone.
func runAuthLogout(cmd *cobra.Command, _ []string) error {
	err := credentialstore.Default().Clear(context.Background(), credentialstore.CredentialAPIKey)
	if err != nil && !stderrors.Is(err, credentialstore.ErrNotFound) {
		return errors.Wrap(err, "failed to clear stored credential")
	}
	if !credentialstore.Default().StoresInConfigFile() {
		if removeErr := credentialstore.Default().RemoveConfigFileEntry(credentialstore.CredentialAPIKey); removeErr != nil {
			logger.PrintIfVerbose(fmt.Sprintf("failed to remove old credential from config file: %v", removeErr))
		}
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Successfully logged out of Checkmarx One server!")
	return nil
}
