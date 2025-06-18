package commands

import (
	prereceive "github.com/Checkmarx/secret-detection/pkg/hooks/pre-receive"
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func PreReceiveCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	preReceiveCmd := &cobra.Command{
		Use:   "pre-receive",
		Short: "Manage pre-receive hooks and run secret detection scans",
		Long:  "The pre-receive command is used for managing Git pre-receive hooks for secret detection",
		Example: heredoc.Doc(
			`
		    $ cx hooks pre-receive secrets-scan
			`,
		),
	}
	preReceiveCmd.AddCommand(scanSecretsPreReceiveCommand(jwtWrapper))

	return preReceiveCmd
}

func scanSecretsPreReceiveCommand(jwtWrapper wrappers.JWTWrapper) *cobra.Command {
	var configFile string
	scanPrereceiveCmd := &cobra.Command{
		Use:   "secrets-scan",
		Short: "Run a pre-receive secret detection scan on the pushed branch",
		Long:  "Runs pre-receive secret detection scans on each pushed branch that is about to enter the remote git repository",
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
			return prereceive.Scan(configFile)
		},
	}

	scanPrereceiveCmd.Flags().StringVarP(&configFile, "config", "c", "", "path to config.yaml file")

	return scanPrereceiveCmd
}
