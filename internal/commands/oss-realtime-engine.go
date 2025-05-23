package commands

import (
	"errors"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services/ossrealtime"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

func RunScanOssRealtimeCommand(realtimeScannerWrapper wrappers.RealtimeScannerWrapper,
	jwtWrapper wrappers.JWTWrapper,
	featureFlagWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		if fileSourceFlag == "" {
			return errors.New("file source flag is required")
		}
		ossRealtimeService := ossrealtime.NewOssRealtimeService(jwtWrapper, featureFlagWrapper, realtimeScannerWrapper)

		packages, err := ossRealtimeService.RunOssRealtimeScan(fileSourceFlag)
		if err != nil {
			return errors.New("failed to run oss-realtime scan: " + err.Error())
		}
		err = printer.Print(cmd.OutOrStdout(), packages, printer.FormatJSON)
		if err != nil {
			return err
		}

		return nil
	}
}
