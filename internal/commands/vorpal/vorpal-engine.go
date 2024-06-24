package vorpal

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RunScanVorpalCommand(jwtWrapper wrappers.JWTWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		vorpalLatestVersion, _ := cmd.Flags().GetBool(commonParams.VorpalLatestVersion)
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		agent, _ := cmd.Flags().GetString(commonParams.AgentFlag)
		var port = viper.GetInt(commonParams.VorpalPortKey)
		vorpalWrapper := grpcs.NewVorpalGrpcWrapper(port)
		vorpalParams := services.VorpalScanParams{
			FilePath:            fileSourceFlag,
			VorpalUpdateVersion: vorpalLatestVersion,
			IsDefaultAgent:      agent == commonParams.DefaultAgent,
		}
		wrapperParams := services.VorpalWrappersParam{
			JwtWrapper:          jwtWrapper,
			FeatureFlagsWrapper: featureFlagsWrapper,
			VorpalWrapper:       vorpalWrapper,
		}
		scanResult, err := services.CreateVorpalScanRequest(vorpalParams, wrapperParams)
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
