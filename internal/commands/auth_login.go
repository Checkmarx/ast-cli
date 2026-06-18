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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// defaultLoginClientID matches the Keycloak client used by the Checkmarx One
// VS Code extension's OAuth flow. Confirmed via the official extension source
// (Checkmarx/ast-vscode-extension, packages/core/src/services/authService.ts).
// This client has localhost callbacks whitelisted across production tenants.
const defaultLoginClientID = "ide-integration"

func newAuthLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate to Checkmarx One via browser-based OAuth",
		Long: "Opens the default browser, walks the user through the Checkmarx One IAM login " +
			"(including MFA), and persists the resulting refresh token. The --session flag picks " +
			"the storage mode: default (yaml) for backward-compatible cross-shell persistence, " +
			"'local' for current-shell env-only via Invoke-Expression / eval, or 'global' for a " +
			"dedicated disk file shared across shells. Every login revokes any existing token " +
			"server-side and clears file storage before issuing the new credential.",
		Example: heredoc.Doc(`
			# Default (yaml) — saves refresh token to ~/.checkmarx/checkmarxcli.yaml
			$ cx auth login --tenant my-tenant

			# Local session mode — refresh token lives in current shell's env var only
			# PowerShell:
			$ Invoke-Expression (cx auth login --tenant my-tenant --session local)
			# bash / zsh:
			$ eval "$(cx auth login --tenant my-tenant --session local)"

			# Global session mode — refresh token persists in ~/.checkmarx/session_global,
			# accessible to every shell, until explicit logout
			$ cx auth login --tenant my-tenant --session global
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
	cmd.Flags().String(params.SessionFlag, "", params.SessionLoginFlagUsage)
	return cmd
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	// cx auth login starts a new login session. The user's explicit --tenant /
	// --base-auth-uri flags must win over the realm URL embedded in any existing
	// API key's JWT claims — they may be logging into a different tenant than
	// their current credential is for.
	viper.Set(params.ApikeyOverrideFlag, true)

	sessionMode, _ := cmd.Flags().GetString(params.SessionFlag)
	if err := validateSessionFlag(sessionMode); err != nil {
		return err
	}

	realmURL, err := wrappers.GetRealmURL()
	if err != nil {
		return errors.Wrap(err, "failed to resolve IAM realm URL")
	}

	clientID := viper.GetString(params.AccessKeyIDConfigKey)
	if clientID == "" {
		clientID = defaultLoginClientID
	}

	// Nuke phase: revoke every existing refresh token server-side and clear
	// the file storages. After this, the system has no active credentials
	// anywhere (modulo any stale env-var bytes in OTHER shells, which the
	// CLI can't reach). The new login that follows establishes exactly one
	// fresh credential in the storage matching --session.
	nukeAllStorages(clientID)

	port, _ := cmd.Flags().GetInt(params.LoginPortFlag)
	noBrowser, _ := cmd.Flags().GetBool(params.LoginNoBrowserFlag)

	tokens, err := wrappers.LoginWithPKCE(context.Background(), wrappers.PKCELoginOptions{
		RealmURL:    realmURL,
		ClientID:    clientID,
		Port:        port,
		OpenBrowser: !noBrowser,
	})
	if err != nil {
		return err
	}

	switch sessionMode {
	case params.SessionLocalValue:
		return persistLocalLogin(cmd, tokens.RefreshToken)
	case params.SessionGlobalValue:
		return persistGlobalLogin(cmd, tokens.RefreshToken)
	default:
		return persistYamlLogin(cmd, tokens.RefreshToken)
	}
}

// validateSessionFlag enforces that --session is either unset, "local", or
// "global". Any other value gets a clear error instead of silently falling
// through to default-mode behavior.
func validateSessionFlag(sessionMode string) error {
	switch sessionMode {
	case "", params.SessionLocalValue, params.SessionGlobalValue:
		return nil
	default:
		return errors.Errorf("invalid --session value %q: must be %q or %q",
			sessionMode, params.SessionLocalValue, params.SessionGlobalValue)
	}
}

// nukeAllStorages reads every storage location, revokes any non-empty token
// at IAM (best-effort, via the OAuth 2.0 revocation endpoint), and clears
// file storages. Env is read but cannot be cleared from a child process —
// its token is revoked server-side, so the bytes that remain in the parent
// shell are inert.
//
// This is called as the first step of every login (regardless of mode) and
// of every logout, ensuring that there is at most one active credential
// anywhere after the operation completes.
func nukeAllStorages(clientID string) {
	// Revoke yaml's token first — read the yaml file directly to bypass any
	// stale env shadowing in viper's normal lookup.
	if yamlRT := readYamlAPIKeyForLogin(); yamlRT != "" {
		revokeOldRefreshToken(yamlRT, clientID, "yaml")
	}
	if envRT := os.Getenv(params.AstAPIKeyEnv); envRT != "" {
		revokeOldRefreshToken(envRT, clientID, "env")
	}
	if globalRT, err := wrappers.ReadSessionGlobal(); err == nil && globalRT != "" {
		revokeOldRefreshToken(globalRT, clientID, "global")
	}
	clearFileStorages()
}

// revokeOldRefreshToken POSTs the given refresh token to the realm extracted
// from its own JWT "aud" claim. Best-effort — failures are logged at verbose
// level so a missing realm claim or a non-2xx response doesn't block the new
// login.
func revokeOldRefreshToken(refreshToken, clientID, sourceLabel string) {
	realmURL, err := wrappers.ExtractFromTokenClaims(refreshToken, audClaim)
	if err != nil || realmURL == "" {
		logger.PrintIfVerbose(fmt.Sprintf("could not extract realm from %s refresh token (skipping revoke): %v", sourceLabel, err))
		return
	}
	if err := wrappers.RevokeRefreshToken(context.Background(), realmURL, clientID, refreshToken); err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("revoke of %s refresh token failed (continuing): %v", sourceLabel, err))
	}
}

// clearFileStorages empties the yaml cx_apikey field and deletes the global
// session file. Best-effort — failures are logged at verbose level. Env is
// not touched here; that's done via shell-eval emission for local-mode
// logins or by the user closing their shell.
func clearFileStorages() {
	if configPath, err := configuration.GetConfigFilePath(); err == nil {
		if writeErr := configuration.SafeWriteSingleConfigKeyString(configPath, params.AstAPIKey, ""); writeErr != nil {
			logger.PrintIfVerbose(fmt.Sprintf("failed to clear yaml cx_apikey: %v", writeErr))
		}
	}
	if err := wrappers.ClearSessionGlobal(); err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("failed to clear global session file: %v", err))
	}
}

// readYamlAPIKeyForLogin reads cx_apikey directly from the yaml file, bypassing
// viper. Used during the nuke phase so we revoke whatever yaml had, not what
// viper currently resolves to (which could be a stale env var).
func readYamlAPIKeyForLogin() string {
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		return ""
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

// persistYamlLogin writes the new refresh token to the yaml config file,
// records yaml as the active mode, and prints CX_APIKEY=<token> + path to
// stdout for scripting parity with cx auth register.
func persistYamlLogin(cmd *cobra.Command, refreshToken string) error {
	configPath, err := configuration.GetConfigFilePath()
	if err != nil {
		return errors.Wrap(err, "failed to resolve config file path")
	}
	if err := configuration.SafeWriteSingleConfigKeyString(configPath, params.AstAPIKey, refreshToken); err != nil {
		return errors.Wrap(err, "failed to save refresh token to config file")
	}
	if err := wrappers.WriteActiveMode(params.SessionYamlValue); err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("failed to write active-mode file: %v", err))
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", params.AstAPIKeyEnv, refreshToken)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Authenticated. Token saved to %s\n", configPath)
	return nil
}

// persistGlobalLogin writes the new refresh token to the dedicated global
// session file and records global as the active mode. No env-var emission —
// global mode is a plain command (no Invoke-Expression wrapper).
func persistGlobalLogin(cmd *cobra.Command, refreshToken string) error {
	if err := wrappers.WriteSessionGlobal(refreshToken); err != nil {
		return errors.Wrap(err, "failed to write global session file")
	}
	if err := wrappers.WriteActiveMode(params.SessionGlobalValue); err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("failed to write active-mode file: %v", err))
	}
	path, _ := wrappers.SessionGlobalFilePath()
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Authenticated. Token saved to %s (global session — persists across shells until explicit logout).\n", path)
	return nil
}

// persistLocalLogin records local as the active mode and emits a single
// shell-evaluable line to stdout: a defensive reset of CX_APIKEY followed by
// the new refresh-token assignment, separated by `;` so the whole emission
// stays on one line. PowerShell's Invoke-Expression accepts only a single
// string argument, so multi-line stdout would be captured as a string array
// and rejected. Bash's `eval` and fish's `;` statement separator handle the
// same single-line form correctly. Informational text goes to stderr to
// keep stdout strictly evaluable.
func persistLocalLogin(cmd *cobra.Command, refreshToken string) error {
	if err := wrappers.WriteActiveMode(params.SessionLocalValue); err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("failed to write active-mode file: %v", err))
	}
	shell := detectShell()
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s; %s\n",
		formatEnvAssignment(shell, params.AstAPIKeyEnv, ""),
		formatEnvAssignment(shell, params.AstAPIKeyEnv, refreshToken))
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Authenticated. Wrap with Invoke-Expression (PowerShell) or eval (bash) to apply.")
	return nil
}
