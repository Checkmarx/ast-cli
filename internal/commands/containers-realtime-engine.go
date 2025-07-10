package commands

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/containersrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func RunScanContainersRealtimeCommand(realtimeScannerWrapper wrappers.RealtimeScannerWrapper,
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		if fileSourceFlag == "" {
			return errorconstants.NewRealtimeEngineError("file path is required").Error()
		}
		containersRealtimeService := containersrealtime.NewContainersRealtimeService(jwtWrapper, featureFlagWrapper, realtimeScannerWrapper)

		images, err := containersRealtimeService.RunContainersRealtimeScan(fileSourceFlag)
		if err != nil {
			return err
		}
		err = printer.Print(cmd.OutOrStdout(), images, printer.FormatJSON)
		if err != nil {
			return errorconstants.NewRealtimeEngineError("failed to return images").Error()
		}

		return nil
	}
}
