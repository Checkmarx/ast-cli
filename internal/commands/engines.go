package commands

import (
	"github.com/MakeNowJust/heredoc"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/spf13/cobra"
)

type EngineAPIDetails struct {
	EngineId    string `format:"name:Engine ID"`
	EngineName  string `format:"name:Engine Name"`
	ApiName     string `format:"name:Api Name"`
	ApiURL      string `format:"name:Api URL"`
	Description string `format:"name:Description"`
}

func NewEnginesCommand(
	enginesWrapper wrappers.EnginesWrapper,
) *cobra.Command {
	enginesCmd := &cobra.Command{
		Use:   "engines",
		Short: "Manage engines",
		Long:  "The engines command enables the ability to manage engines in Checkmarx One.",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				``,
			),
		},
	}

	getEngineListCmd := getEnginesListSubCommand(
		enginesWrapper,
	)
	enginesCmd.AddCommand(
		getEngineListCmd,
	)
	return enginesCmd
}

func getEnginesListSubCommand(
	enginesWrapper wrappers.EnginesWrapper,
) *cobra.Command {
	getEngineListCmd := &cobra.Command{
		Use:   "list-api",
		Short: "Get list of all engines",
		Long:  "The list-api command is used to get list of all engine apis in Checkmarx One.",
		Example: heredoc.Doc(
			`
			$ cx engines list-api --engine-name <Engine Name> --output-format <output-format>
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				``,
			),
		},
		RunE: runEngineGetListCommand(
			enginesWrapper,
		),
	}
	getEngineListCmd.PersistentFlags().String(commonParams.EngineName, "", "Filters Engines by EngineName")
	getEngineListCmd.PersistentFlags().String(commonParams.OutputFormat, "table", "Show Engine Details based on Output Format")
	return getEngineListCmd
}

func runEngineGetListCommand(
	enginesWrapper wrappers.EnginesWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var engineApiResponse *wrappers.EngineListResponseModel
		engineName, _ := cmd.Flags().GetString(commonParams.EngineName)
		engineApiResponse = enginesWrapper.Get(engineName)

		if engineApiResponse != nil {
			views := ShowEngineList(engineApiResponse.Engines)
			return printByOutputFormat(cmd, views)
		}
		return nil
	}
}

func ShowEngineList(engines []wrappers.EngineList) []*EngineAPIDetails {

	views := make([]*EngineAPIDetails, 0)
	for i := 0; i < len(engines); i++ {
		for j := 0; j < len(engines[i].APIs); j++ {
			views = append(views, &EngineAPIDetails{
				EngineId:    engines[i].EngineId,
				EngineName:  engines[i].EngineName,
				ApiName:     engines[i].APIs[j].ApiName,
				ApiURL:      engines[i].APIs[j].ApiURL,
				Description: engines[i].APIs[j].Description,
			})
		}
	}
	return views
}
