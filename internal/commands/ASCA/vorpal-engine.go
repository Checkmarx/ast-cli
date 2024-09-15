package ASCA

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RunScanASCACommand(jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ASCALatestVersion, _ := cmd.Flags().GetBool(commonParams.ASCALatestVersion)
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		agent, _ := cmd.Flags().GetString(commonParams.AgentFlag)
		var port = viper.GetInt(commonParams.ASCAPortKey)
		ASCAWrapper := grpcs.NewASCAGrpcWrapper(port)
		ASCAParams := services.ASCAScanParams{
			FilePath:          fileSourceFlag,
			ASCAUpdateVersion: ASCALatestVersion,
			IsDefaultAgent:    agent == commonParams.DefaultAgent,
		}
		wrapperParams := services.ASCAWrappersParam{
			JwtWrapper:          jwtWrapper,
			FeatureFlagsWrapper: featureFlagsWrapper,
			ASCAWrapper:         ASCAWrapper,
		}
		scanResult, err := services.CreateASCAScanRequest(ASCAParams, wrapperParams)
		if err != nil {
			return err
		}

		err = printer.Print(cmd.OutOrStdout(), scanResult, printer.FormatJSON)
		if err != nil {
			return err
		}

		return nil
	}
}
