package commands

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/logger"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const ()

func NewEnginesCommand(
	engineWrapper wrappers.EnginesWrapper,
	exportWrapper wrappers.ExportWrapper,
	logsWrapper wrappers.LogsWrapper,
	groupsWrapper wrappers.GroupsWrapper,
	jwtWrapper wrappers.JWTWrapper,
) *cobra.Command {
	enginesCmd := &cobra.Command{
		Use:   "engines",
		Short: "Know/Manage your engines.",
		Long:  "The engines command enables the ability to know/configure engines in Checkmarx One.",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://docs.checkmarx.com/en/34965-322308-checkmarx-one-scanners.html
			`,
			),
		},
	}

	enginesListApiCmd := enginesListAPISubCommand(engineWrapper)

	addFormatFlagToMultipleCommands(
		[]*cobra.Command{enginesListApiCmd},
		printer.FormatTable, printer.FormatList, printer.FormatJSON,
	)

	enginesCmd.AddCommand(
		enginesListApiCmd,
	)
	return enginesCmd
}

func enginesListAPISubCommand(engineWrapper wrappers.EnginesWrapper) *cobra.Command {
	enginesListApiCmd := &cobra.Command{
		Use:   "list-api",
		Short: "List all APIs for all or a given engine.",
		Long:  "The list-api command enables the ability to get list of all Engine APIs exposed by Checkmarx One.",
		Example: heredoc.Doc(
			`
			$ cx engines list-api --format table|json|indented-json
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.stoplight.io/docs/checkmarx-one-api-reference-guide/branches/main/jkna3cb6zhcjp-introduction
			`,
			),
		},
		RunE: runEnginesListApiCommand(engineWrapper),
	}

	enginesListApiCmd.PersistentFlags().String(commonParams.EngineName, "sast,iac-security,sca,api-security,container-security,scs", "Name of the engine.")

	return enginesListApiCmd
}

func runEnginesListApiCommand(engineWrapper wrappers.EnginesWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var engineAPIModel *wrappers.EnginesAPIInventoryResponseModel
		var errorModel *wrappers.ErrorModel
		engineName, _ := cmd.Flags().GetString(commonParams.EngineName)
		engineNames := strings.Split(engineName, ",")

		logger.PrintIfVerbose(
			fmt.Sprintf(
				"Fetching API details for %s",
				engineNames),
		)

		engineAPIModel, errorModel, err := engineWrapper.Get(engineName)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingAll)
		}
		logger.PrintIfVerbose(
			fmt.Sprintf(
				"Inside runEnginesListApiCommand 2 %s",
				engineName),
		)
		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedGettingAll, errorModel.Code, errorModel.Message)
		} else if engineAPIModel != nil && engineAPIModel.Engines != nil {
			logger.PrintIfVerbose(
				fmt.Sprintf(
					"Inside runEnginesListApiCommand 3%s",
					engineName),
			)
			views, err := toEngineAPIViews(engineAPIModel.Engines)
			if err != nil {
				return err
			}
			err = printByFormat(cmd, views.Engines)
			if err != nil {
				return err
			}
		}
		logger.PrintIfVerbose(
			fmt.Sprintf(
				"Inside runEnginesListApiCommand 4 %s",
				engineName),
		)
		return nil
	}
}

type engineListAPIView struct {
	Engines []EnginesViews `json:"engines"`
}

type EnginesViews struct {
	Engine_Id   string           `json:"engine_id"`
	Engine_Name string           `json:"engine_name"`
	Apis        []EnginesAPIView `json:"apis"`
}

type EnginesAPIView struct {
	Api_Url     string `json:"api_url"`
	Api_Name    string `json:"api_name"`
	Description string `json:"description"`
}

func toEngineAPIViews(engines []wrappers.EnginesAPIResponseModel) (*engineListAPIView, error) {

	views := make([]EnginesViews, len(engines))
	for i := 0; i < len(engines); i++ {
		views[i] = toEngineView(&engines[i])
	}

	return &engineListAPIView{
		Engines: views,
	}, nil
}

func toEngineView(engine *wrappers.EnginesAPIResponseModel) EnginesViews {

	engineApis := make([]EnginesAPIView, len(engine.Apis))
	for i := 0; i < len(engine.Apis); i++ {
		engineApis[i] = toEngineApis(&engine.Apis[i])
	}

	return EnginesViews{
		Engine_Id:   engine.Engine_Id,
		Engine_Name: engine.Engine_Name,
		Apis:        engineApis,
	}
}

func toEngineApis(engineApi *wrappers.EnginesAPIData) EnginesAPIView {

	return EnginesAPIView{
		Api_Url:     engineApi.Api_Url,
		Api_Name:    engineApi.Api_Name,
		Description: engineApi.Description,
	}
}
