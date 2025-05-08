package commands

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/wrappers"
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
