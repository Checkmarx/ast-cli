package realtimeengine

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorconstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/realtimeengine/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func RunScanOssRealtimeCommand(realtimeScannerWrapper wrappers.RealtimeScannerWrapper,
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		if fileSourceFlag == "" {
			return errorconstants.NewOssRealtimeError("file path is required").Error()
		}
		ossRealtimeService := ossrealtime.NewOssRealtimeService(jwtWrapper, featureFlagWrapper, realtimeScannerWrapper)

		packages, err := ossRealtimeService.RunOssRealtimeScan(fileSourceFlag)
		if err != nil {
			return err
		}
		err = printer.Print(cmd.OutOrStdout(), packages, printer.FormatJSON)
		if err != nil {
			return errorconstants.NewOssRealtimeError("failed to return packages").Error()
		}

		return nil
	}
}
