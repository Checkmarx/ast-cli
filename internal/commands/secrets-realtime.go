package commands

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/secretsrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func RunScanSecretsRealtimeCommand(
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		if fileSourceFlag == "" {
			return errorconstants.NewRealtimeEngineError(errorconstants.RealtimeEngineFilePathRequired).Error()
		}
		secretsRealtimeService := secretsrealtime.NewSecretsRealtimeService(jwtWrapper, featureFlagWrapper)

		results, err := secretsRealtimeService.RunSecretsRealtimeScan(fileSourceFlag)
		if err != nil {
			return err
		}
		err = printer.Print(cmd.OutOrStdout(), results, printer.FormatJSON)
		if err != nil {
			return errorconstants.NewRealtimeEngineError("failed to return packages").Error()
		}

		return nil
	}
}
