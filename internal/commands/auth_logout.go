package commands

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// audClaim is the OIDC "audience" JWT claim. For Keycloak refresh tokens it
// holds the realm URL — exactly the URL we POST to for revocation.
const audClaim = "aud"

func newAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke the current refresh token and clear stored credentials",
		Long: "Revokes the current refresh token at Checkmarx One IAM and clears every storage " +
			"location: yaml cx_apikey, the global session file, and emits a shell-evaluable " +
			"clear of CX_APIKEY for users who logged in via --session local. One universal " +
			"logout — no --session flag needed; the active mode tells the CLI what to clean up.",
		Example: heredoc.Doc(`
			# Default usage (clears yaml and the global file, revokes server-side)
			$ cx auth logout

			# If the current shell was logged in via --session local, also wrap the
			# logout with Invoke-Expression so $env:CX_APIKEY gets cleared too
			# PowerShell:
			$ Invoke-Expression (cx auth logout)
			# bash / zsh:
			$ eval "$(cx auth logout)"
		`),
		RunE: runAuthLogout,
	}
}

// runAuthLogout is the universal logout: it nukes every storage location's
// credential (server-side revoke + local clear), deletes the active-mode
// metadata file, and emits a shell-clear line so users who wrap the call
// with Invoke-Expression / eval also have CX_APIKEY cleared in their shell.
func runAuthLogout(cmd *cobra.Command, _ []string) error {
	clientID := viper.GetString(params.AccessKeyIDConfigKey)
	if clientID == "" {
		clientID = defaultLoginClientID
	}

	nukeAllStorages(clientID)

	if err := wrappers.ClearActiveMode(); err != nil {
		logger.PrintIfVerbose(fmt.Sprintf("failed to remove active-mode file: %v", err))
	}

	// Always emit a shell-clear of CX_APIKEY to stdout. Wrapping the logout
	// with Invoke-Expression (PowerShell) or eval (bash) clears the env var
	// in the current shell. Without the wrapper the line just prints — no
	// harm done for users who didn't use --session local.
	shell := detectShell()
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), formatEnvAssignment(shell, params.AstAPIKeyEnv, ""))
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Logged out. If you used --session local in this shell, wrap with Invoke-Expression (PowerShell) or eval (bash) to clear CX_APIKEY.")
	return nil
}
