package commands

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
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

// runAuthLogout clears the cx_apikey field in the yaml config file. The
// client-credentials and env-provided credentials are intentionally left alone.
func runAuthLogout(cmd *cobra.Command, _ []string) error {
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		return errors.Wrap(err, "failed to resolve config file path")
	}
	if err := configuration.SafeWriteSingleConfigKeyString(configPath, params.AstAPIKey, ""); err != nil {
		return errors.Wrap(err, "failed to clear stored credential")
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Successfully logged out of Checkmarx One server!")
	return nil
}
