package commands

import (
	prereceive "github.com/Checkmarx/secret-detection/pkg/hooks/pre-receive"
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

// PreReceiveCommand creates the pre-receive subcommand
func PreReceiveCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	preReceiveCmd := &cobra.Command{
		Use:   "pre-receive",
		Short: "Manage pre-receive hooks and run secret detection scans",
		Long:  "The pre-receive command enables the ability to manage Git pre-receive hooks for secret detection.",
		Example: heredoc.Doc(
			`
            $ cx hooks pre-receive secrets-scan
        `,
		),
	}

	preReceiveCmd.AddCommand(preReceiveSecretsScanCommand(jwtWrapper))

	return preReceiveCmd
}

func preReceiveSecretsScanCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "secrets-scan",
		Short: "Scan all commits about to enter the remote git repository.",
		Example: heredoc.Doc(
			`
			$ cx hooks pre-receive secrets-scan
            $ cx hooks pre-receive secrets-scan --config /path/to/config.yaml
        `,
		),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateLicense(jwtWrapper)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return prereceive.Scan(configPath)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to configuration file")

	return cmd
}
