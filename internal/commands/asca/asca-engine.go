package asca

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RunScanASCACommand(jwtWrapper wrappers.JWTWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ASCALatestVersion, _ := cmd.Flags().GetBool(commonParams.ASCALatestVersion)
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		ignoredFilePathFlag, _ := cmd.Flags().GetString(commonParams.IgnoredFilePathFlag)
		agent, _ := cmd.Flags().GetString(commonParams.AgentFlag)
		var port = viper.GetInt(commonParams.ASCAPortKey)
		ASCAWrapper := grpcs.NewASCAGrpcWrapper(port)
		ASCAParams := services.AscaScanParams{
			FilePath:          fileSourceFlag,
			ASCAUpdateVersion: ASCALatestVersion,
			IsDefaultAgent:    agent == commonParams.DefaultAgent,
			IgnoredFilePath:   ignoredFilePathFlag,
		}
		wrapperParams := services.AscaWrappersParam{
			JwtWrapper:  jwtWrapper,
			ASCAWrapper: ASCAWrapper,
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
