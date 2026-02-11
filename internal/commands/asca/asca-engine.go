package asca

import (
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/grpcs"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"strings"
)

const (
	ascaLocationParam = "asca-location"
)

func RunScanASCACommand(jwtWrapper wrappers.JWTWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var vorpalLocation string
		ASCALatestVersion, _ := cmd.Flags().GetBool(commonParams.ASCALatestVersion)
		fileSourceFlag, _ := cmd.Flags().GetString(commonParams.SourcesFlag)
		ignoredFilePathFlag, _ := cmd.Flags().GetString(commonParams.IgnoredFilePathFlag)
		agent, _ := cmd.Flags().GetString(commonParams.AgentFlag)
		err := validateASCALocationFlags(cmd)
		if err != nil {
			return err
		}

		vorpal := strings.TrimSpace(viper.GetString(commonParams.VorpalCustomPathKey))
		if vorpal != "" {
			vorpalLocation = vorpal
		} else if location := utils.GetOptionalParam(ascaLocationParam); location != "" {
			vorpalLocation = location
		}

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
		scanResult, err := services.CreateASCAScanRequest(ASCAParams, wrapperParams, vorpalLocation)
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

func validateASCALocationFlags(cmd *cobra.Command) error {
	if cmd.Flags().Changed(commonParams.ASCALocationFlag) {
		flagVal, err := cmd.Flags().GetString(commonParams.ASCALocationFlag)
		if err != nil {
			return err
		}
		if strings.TrimSpace(flagVal) == "" {
			return errors.Errorf("%s flag is provided but empty", commonParams.ASCALocationFlag)
		}
	}
	return nil
}
