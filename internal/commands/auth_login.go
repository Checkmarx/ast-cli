package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/credentialstore"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/configuration"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// defaultLoginClientID is the public PKCE client the IDE plugins use; its
// localhost callbacks are whitelisted and it needs no client secret.
const defaultLoginClientID = "ide-integration"

// configFilePerm: owner-only, since the file holds a long-lived refresh token.
const configFilePerm = 0o600

func newAuthLoginCommand(telemetryWrapper wrappers.TelemetryWrapper) *cobra.Command {
	if transport, ok := telemetryWrapper.(*wrappers.TelemetryHTTPWrapper); ok {
		telemetryWrapper = transport.ForAgentHooks()
	}
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate to Checkmarx One via browser-based OAuth or API key",
		Long: "Opens the default browser, walks the user through the Checkmarx One IAM login " +
			"(including MFA), and stores the resulting refresh token securely in the OS keyring " +
			"under the api-key credential slot — the same slot cx configure writes to, so every " +
			"other command picks it up automatically.\n\n" +
			"Browser sign-in requires --tenant and --base-uri (or --base-auth-uri). Pass them as flags, or run " +
			"cx auth login with none and it prompts for the missing ones like cx configure.\n\n" +
			"Use --apikey for API-key sign-in. A successful login with a supported --agent sends authentication telemetry.",
		Example: heredoc.Doc(`
			# With flags — stores the refresh token in the OS keyring
			$ cx auth login --base-uri https://<region>.ast.checkmarx.net --tenant my-tenant

			# Sign in with an API key
			$ cx auth login --apikey <key> --agent=copilot-cli

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAuthLogin(cmd, telemetryWrapper)
		},
	}
	cmd.Flags().Int(params.LoginPortFlag, 0, params.LoginPortFlagUsage)
	cmd.Flags().Bool(params.LoginNoBrowserFlag, false, params.LoginNoBrowserFlagUsage)
	return cmd
}

func runAuthLogin(cmd *cobra.Command, telemetryWrapper wrappers.TelemetryWrapper) error {
	if cmd.Flags().Changed(params.AstAPIKeyFlag) {
		apiKey, err := cmd.Flags().GetString(params.AstAPIKeyFlag)
		if err != nil || strings.TrimSpace(apiKey) == "" {
			return errors.New("--apikey requires a nonempty API key")
		}
		accessToken, err := wrappers.ExchangeLoginAPIKey(apiKey)
		if err != nil {
			return err
		}
		return completeAuthLogin(cmd, telemetryWrapper, apiKey, accessToken, "APIKey")
	}

	// Prompt for connection details like cx configure when none were passed as
	// flags, so a re-login can still review/change base-uri / tenant.
	if !connectionFlagsProvided(cmd) {
		configuration.PromptAuthConnection()
	}

	// Force explicit --tenant/--base-* to win over the realm in any stored key.
	// Set after the prompt so the prompt's write isn't persisted here.
	previousOverride := viper.GetBool(params.ApikeyOverrideFlag)
	viper.Set(params.ApikeyOverrideFlag, true)
	realmURL, err := wrappers.GetRealmURL()
	viper.Set(params.ApikeyOverrideFlag, previousOverride)
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

	return completeAuthLogin(cmd, telemetryWrapper, tokens.RefreshToken, tokens.AccessToken, "OAuth")
}

func loginTelemetryAgent(cmd *cobra.Command) string {
	if !cmd.Flags().Changed(params.AgentFlag) {
		return ""
	}
	agent, _ := cmd.Flags().GetString(params.AgentFlag)
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "copilot-cli":
		return "Copilot-cli"
	case "claude-cli":
		return "Claude-cli"
	case "cursor-cli":
		return "Cursor-cli"
	case "gemini-cli":
		return "Gemini-cli"
	case "codex-cli":
		return "Codex-cli"
	default:
		return ""
	}
}

func completeAuthLogin(cmd *cobra.Command, telemetryWrapper wrappers.TelemetryWrapper, credential, accessToken, authMethod string) error {
	if err := persistLogin(cmd, credential); err != nil {
		return err
	}
	agent := loginTelemetryAgent(cmd)
	if telemetryWrapper == nil || agent == "" || accessToken == "" || os.Getenv("CX_TELEMETRY_DISABLE") == "1" {
		return nil
	}
	wrappers.CacheAuthenticatedAccessToken(accessToken)
	attrs := wrappers.EventAttributes{"authMethod": authMethod}
	data := &wrappers.DataForAITelemetry{
		Type:            authenticationEventType,
		SubType:         authenticationEventType,
		Status:          telemetrySuccessStatus,
		StatusResult:    "Authentication successful.",
		Agent:           agent,
		AgentVersion:    strings.TrimSpace(os.Getenv(params.AgentVersionEnv)),
		CxPluginVersion: strings.TrimSpace(os.Getenv(params.CxPluginVersionEnv)),
		PluginType:      "CLI",
		UniqueID:        wrappers.GetUniqueID(),
		SessionID:       wrappers.GetSessionID(),
		EventAttributes: &attrs,
	}
	if err := telemetryWrapper.SendAIDataToLog(data); err != nil {
		logger.PrintIfVerbose("Failed to send authentication telemetry: " + err.Error())
	}
	return nil
}

// connectionFlagsProvided reports whether any connection detail was passed as a flag.
func connectionFlagsProvided(cmd *cobra.Command) bool {
	return cmd.Flags().Changed(params.BaseURIFlag) ||
		cmd.Flags().Changed(params.BaseAuthURIFlag) ||
		cmd.Flags().Changed(params.TenantFlag)
}

// persistLogin stores the credential without echoing it to stdout.
func persistLogin(cmd *cobra.Command, credential string) error {
	if err := credentialstore.Default().Store(context.Background(), credentialstore.CredentialAPIKey, credential); err != nil {
		return errors.Wrap(err, "failed to save login credential")
	}
	if !credentialstore.Default().StoresInConfigFile() {
		if removeErr := credentialstore.Default().RemoveConfigFileEntry(credentialstore.CredentialAPIKey); removeErr != nil {
			logger.PrintIfVerbose(fmt.Sprintf("failed to remove old refresh token from config file: %v", removeErr))
		}
	}
	// Restrict a pre-existing config file to owner-only; best-effort no-op on
	// Windows and when the file does not exist.
	configPath, err := configuration.GetConfigFilePath()
	if err == nil {
		if chErr := os.Chmod(configPath, configFilePerm); chErr != nil {
			logger.PrintIfVerbose(fmt.Sprintf("failed to restrict config file permissions: %v", chErr))
		}
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Successfully authenticated to Checkmarx One server!")
	return nil
}
