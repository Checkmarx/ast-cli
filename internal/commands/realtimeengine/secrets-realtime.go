package realtimeengine

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func RunScanSecretsRealtimeCommand(realtimeScannerWrapper wrappers.RealtimeScannerWrapper,
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		if fileSourceFlag == "" {
			return errorconstants.NewSecretRealtimeError("file path is required").Error()
		}
		ossRealtimeService := secretsrealtime.NewSecretsRealtimeService(jwtWrapper, featureFlagWrapper, realtimeScannerWrapper)

		results, err := ossRealtimeService.RunSecretsRealtimeScan(fileSourceFlag)
		if err != nil {
			return err
		}
		err = printer.Print(cmd.OutOrStdout(), results, printer.FormatJSON)
		if err != nil {
			return errorconstants.NewOssRealtimeError("failed to return packages").Error()
		}

		return nil
	}
}
