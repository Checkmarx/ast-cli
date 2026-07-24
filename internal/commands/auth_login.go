package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/checkmarx/ast-cli/internal/wrappers/credentialstore"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// defaultLoginClientID is the public PKCE client the IDE plugins use; its
// localhost callbacks are whitelisted and it needs no client secret.
const defaultLoginClientID = "ide-integration"

// configFilePerm: owner-only, since the file holds a long-lived refresh token.
const configFilePerm = 0o600

func newAuthLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate to Checkmarx One via browser-based OAuth",
		Long: "Opens the default browser, walks the user through the Checkmarx One IAM login " +
			"(including MFA), and saves the resulting refresh token to the OS keyring (falling back " +
			"to the config file's cx_apikey field) — the same credential slot cx configure writes " +
			"to, so every other command picks it up automatically.\n\n" +
			"Requires --tenant and --base-uri (or --base-auth-uri). Pass them as flags, or run " +
			"cx auth login with none and it prompts for the missing ones like cx configure.",
		Example: heredoc.Doc(`
			# With flags — saves the refresh token to ~/.checkmarx/checkmarxcli.yaml
			$ cx auth login --base-uri https://<region>.ast.checkmarx.net --tenant my-tenant

			# No flags — prompts for base URI / tenant, then opens the browser
			$ cx auth login

			# Print the authorization URL instead of opening a browser
			$ cx auth login --base-uri https://<region>.ast.checkmarx.net --tenant my-tenant --no-browser
		`),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(`
				https://checkmarx.com/resource/documents/en/34965-68627-auth.html
			`),
		},
		RunE: runAuthLogin,
	}
	cmd.Flags().Int(params.LoginPortFlag, 0, params.LoginPortFlagUsage)
	cmd.Flags().Bool(params.LoginNoBrowserFlag, false, params.LoginNoBrowserFlagUsage)
	return cmd
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	// Prompt for connection details like cx configure when none were passed as
	// flags, so a re-login can still review/change base-uri / tenant.
	if !connectionFlagsProvided(cmd) {
		configuration.PromptAuthConnection()
	}

	// Force explicit --tenant/--base-* to win over the realm in any stored key.
	// Set after the prompt so the prompt's write isn't persisted here.
	viper.Set(params.ApikeyOverrideFlag, true)

	realmURL, err := wrappers.GetRealmURL()
	if err != nil {
		return errors.Wrap(err, "failed to resolve IAM realm URL")
	}

	port, _ := cmd.Flags().GetInt(params.LoginPortFlag)
	noBrowser, _ := cmd.Flags().GetBool(params.LoginNoBrowserFlag)

	tokens, err := wrappers.LoginWithPKCE(context.Background(), wrappers.PKCELoginOptions{
		RealmURL:    realmURL,
		ClientID:    defaultLoginClientID,
		Port:        port,
		OpenBrowser: !noBrowser,
	})
	if err != nil {
		return err
	}

	return persistLogin(cmd, tokens.RefreshToken)
}

// connectionFlagsProvided reports whether any connection detail was passed as a flag.
func connectionFlagsProvided(cmd *cobra.Command) bool {
	return cmd.Flags().Changed(params.BaseURIFlag) ||
		cmd.Flags().Changed(params.BaseAuthURIFlag) ||
		cmd.Flags().Changed(params.TenantFlag)
}

// persistLogin stores the refresh token (keyring, yaml fallback); never echoes it.
func persistLogin(cmd *cobra.Command, refreshToken string) error {
	if err := credentialstore.Default.SetSecret(params.AstAPIKey, refreshToken); err != nil {
		return errors.Wrap(err, "failed to save refresh token")
	}
	// Restrict the file in case the token fell back to yaml; best-effort no-op on Windows.
	if configPath, err := configuration.GetConfigFilePath(); err == nil {
		if chErr := os.Chmod(configPath, configFilePerm); chErr != nil {
			logger.PrintIfVerbose(fmt.Sprintf("failed to restrict config file permissions: %v", chErr))
		}
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Successfully authenticated to Checkmarx One server!")
	return nil
}
