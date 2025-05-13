package commands

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewHooksCommand creates the hooks command with pre-commit and pre-receive subcommand
func NewHooksCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	hooksCmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage Git hooks",
		Long:  "The hooks command enables the ability to manage Git hooks for Checkmarx One.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-commit secrets-install-git-hook
            $ cx hooks pre-commit secrets-scan
			$ cx hooks pre-receive secrets-scan
        `,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
                https://checkmarx.com/resource/documents/en/xxxxx-xxxxx-hooks.html
            `,
			),
		},
	}

	// Add pre-commit subcommand
	hooksCmd.AddCommand(PreCommitCommand(jwtWrapper))

	// Add pre-receive subcommand
	hooksCmd.AddCommand(PreReceiveCommand(jwtWrapper))

	return hooksCmd
}

// validateLicense verifies the user has the required license for secret detection
func validateLicense(jwtWrapper wrappers.JWTWrapper) error {

	allowed, err := jwtWrapper.IsAllowedEngine(params.EnterpriseSecretsLabel)
	if err != nil {
		return errors.Wrapf(err, "Failed checking license")
	}
	if !allowed {
		return errors.New("Error: License validation failed. Please verify your CxOne license includes Enterprise Secrets.")
	}
	return nil
}
