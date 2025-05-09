package commands

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewEnginesCommand(
	enginesWrapper wrappers.EnginesWrapper,
) *cobra.Command {
	enginesCmd := &cobra.Command{
		Use:   "engines",
		Short: "Fetch supported API of scanner engines",
		Long:  "The engines command enables the ability to fetch engines APIs list in Checkmarx One.",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68643-scan.html
			`,
			),
		},
	}
	listEngineAPIcmd := enginesListAPISubCommand(enginesWrapper)
	enginesCmd.AddCommand(listEngineAPIcmd)
	return enginesCmd
}

func enginesListAPISubCommand(
	enginesWrapper wrappers.EnginesWrapper,
) *cobra.Command {
	enginesListAPIcmd := &cobra.Command{
		Use:   "list-api",
		Short: "fetch the API list of scanner engines",
		Long:  "The create list-api fetch the API list of scanner engines in Checkmarx One.",
		Example: heredoc.Doc(
			`
			$ cx engines list-api --engine-name <Engine Name>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68643-scan.html#UUID-a0bb20d5-5182-3fb4-3da0-0e263344ffe7
			`,
			),
		},
		RunE: runEnginesListAPICommand(enginesWrapper),
	}
	enginesListAPIcmd.PersistentFlags().String("engine-name", "", "The name of the Checkmarx scanner engine to use.")

	addOutputFormatFlag(
		enginesListAPIcmd,
		printer.FormatTable,
		printer.FormatJSON,
		printer.FormatYAML,
	)
	return enginesListAPIcmd
}

func runEnginesListAPICommand(enginesWrapper wrappers.EnginesWrapper) func(cmd *cobra.Command, args []string) error {
	//fmt.Println("Inside the command execution runEnginesListAPICommand function")
	return func(cmd *cobra.Command, args []string) error {
		var apiModels []wrappers.ApiModel
		var errorModel *wrappers.ErrorModel
		//fmt.Println("Before flag")
		engineName, err := cmd.Flags().GetString("engine-name")
		if err != nil {
			return errors.Wrapf(err, "%s", "Invalid 'engine-name' flag")
		}
		apiModels, errorModel, err = enginesWrapper.GetAllAPIs(engineName)
		if err != nil {
			return errors.Wrapf(err, "%s\n", "Failed to fetch all engines APIs")
		}

		//fmt.Println(apiModels)
		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, "Failed to Getting All apis in error model", errorModel.Code, errorModel.Message)
		} else if apiModels != nil && len(apiModels) > 0 {
			f1, _ := cmd.Flags().GetString(params.OutputFormatFlag)
			if f1 == "table" {
				views := toAPIsViews(apiModels)
				if err != nil {
					return err
				}
				err = printByOutputFormat(cmd, views)
				if err != nil {
					return err
				}
			} else {
				views := toEnginesView(apiModels)
				if err != nil {
					return err
				}
				err = printByOutputFormat(cmd, views)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

}

type Engine struct {
	EngineID   string `json:"engine_id"`
	EngineName string `json:"engine_name"`
	APIs       []API  `json:"apis"`
}

type API struct {
	ApiUrl      string `json:"api_url"`
	ApiName     string `json:"api_name"`
	Description string `json:"description"`
}

type EnginesView struct {
	Engines []Engine `json:"engines"`
}

func toEnginesView(models []wrappers.ApiModel) EnginesView {
	engineMap := make(map[string]Engine)

	// Group APIs by engine
	for _, model := range models {
		api := API{
			ApiUrl:      model.ApiUrl,
			ApiName:     model.ApiName,
			Description: model.Description,
		}

		engine, exists := engineMap[model.EngineId]
		if !exists {
			engine = Engine{
				EngineID:   model.EngineId,
				EngineName: model.EngineName,
				APIs:       []API{},
			}
		}
		engine.APIs = append(engine.APIs, api)
		engineMap[model.EngineId] = engine
	}

	// Collect all engines
	var engines []Engine
	for _, engine := range engineMap {
		engines = append(engines, engine)
	}

	return EnginesView{
		Engines: engines,
	}
}

func toAPIsViews(models []wrappers.ApiModel) []apiView {
	result := make([]apiView, len(models))
	for i := 0; i < len(models); i++ {
		result[i] = toAPIView(models[i])
	}
	return result
}
func toAPIView(model wrappers.ApiModel) apiView {
	return apiView{
		ApiName:     model.ApiName,
		Description: model.Description,
		ApiUrl:      model.ApiUrl,
		EngineName:  model.EngineName,
		EngineId:    model.EngineId,
	}
}

type apiView struct {
	ApiName     string
	Description string
	ApiUrl      string
	EngineName  string
	EngineId    string
}
