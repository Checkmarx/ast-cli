package commands

import (
	"fmt"
	"log"

	prereceive "github.com/Checkmarx/secret-detection/pkg/hooks/pre-receive"
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

const (
	SuccessFullSecretsLicenceValidation = "License for pre-receive secret detection has been validated successfully"
)

func PreReceiveCommand(jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) *cobra.Command {
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
	preReceiveCmd.AddCommand(scanSecretsPreReceiveCommand())
	preReceiveCmd.AddCommand(validateSecretsLicence(jwtWrapper, featureFlagsWrapper))

	return preReceiveCmd
}

func scanSecretsPreReceiveCommand() *cobra.Command {
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return prereceive.Scan(configFile)
		},
	}

	scanPrereceiveCmd.Flags().StringVarP(&configFile, "config", "c", "", "path to config.yaml file")

	return scanPrereceiveCmd
}

func validateSecretsLicence(jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) *cobra.Command {
	validateLicence := &cobra.Command{
		Use:   "validate",
		Short: "Validates the license for pre-receive secret detection",
		Long:  "Validates the license for pre-receive secret detection",
		Example: heredoc.Doc(
			`
		    $ cx hooks pre-receive validate
			`,
		),
		RunE: checkLicence(jwtWrapper, featureFlagsWrapper),
	}
	return validateLicence
}

func checkLicence(jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scsLicensingV2Flag, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, wrappers.ScsLicensingV2Enabled)
		var licenseName string
		if scsLicensingV2Flag.Status {
			licenseName = params.SecretDetectionLabel
		} else {
			licenseName = params.EnterpriseSecretsLabel
		}

		isAllowed, err := jwtWrapper.IsAllowedEngine(licenseName)
		if err != nil {
			log.Fatalf("%s: %s", "Failed the licence check", err)
		}
		if !isAllowed {
			log.Fatalf("Error: License validation failed. Please ensure that your Checkmarx One license includes %s", licenseName)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), SuccessFullSecretsLicenceValidation)
		return nil
	}
}
