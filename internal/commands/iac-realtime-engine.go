package commands

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	iac_realtime "github.com/checkmarx/ast-cli/internal/services/realtimeengine/iac-realtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func RunScanIacRealtimeCommand(
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		if fileSourceFlag == "" {
			return errorconstants.NewRealtimeEngineError("file path is required").Error()
		}

		ignoredFilePathFlag, _ := cmd.Flags().GetString(commonParams.IgnoredFilePathFlag)
		engine, _ := cmd.Flags().GetString(commonParams.IacRealtimeEngineFlag)

		iacRealtimeService := iac_realtime.NewIacRealtimeService(jwtWrapper, featureFlagWrapper)

		results, err := iacRealtimeService.RunIacRealtimeScan(fileSourceFlag, engine, ignoredFilePathFlag)
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
