package commands

import (
	"github.com/MakeNowJust/heredoc"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type EngineView struct {
	EngineID    string `format:"name:Engine ID"`
	EngineName  string `format:"name:Engine Name"`
	ApiName     string `format:"name:Api Name"`
	ApiURL      string `format:"name:Api URL"`
	Description string `format:"name:Description"`
}

// engine command implementation
func EngineCommand(engineWrapper wrappers.EngineWrapper) *cobra.Command {
	EngineCmd := &cobra.Command{
		Use:   "engine",
		Short: "manages the engine ",
		Long:  "The scan command enables the ability to manage engine in Checkmarx One.",
	}
	listScanCmd := listEngineSubCmd(engineWrapper)
	EngineCmd.AddCommand(listScanCmd)
	return EngineCmd
}

// list Api of engines implementation
func listEngineSubCmd(engineWrapper wrappers.EngineWrapper) *cobra.Command {
	enginelistCmd := &cobra.Command{
		Use:   "list-api",
		Short: "lists all APIs of CXOne engines",
		Long:  "The list command provides a list of all  APIs of the engines in Checkmarx One.",
		Example: heredoc.Doc(
			`$cx engine list-api
            `,
		),
		RunE: runListEnginesCommand(engineWrapper),
	}
	enginelistCmd.PersistentFlags().String(commonParams.EngineName, "", "Name of the engine")
	enginelistCmd.PersistentFlags().String(commonParams.EngineOutputFormat, "table", "Name of format")
	return enginelistCmd
}

// This will call the get method to list the APIs of the engine/engines

func runListEnginesCommand(engineWrapper wrappers.EngineWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var allEngineModel *wrappers.EnginesCollectionResponseModel
		var errorModel *wrappers.ErrorModel
		engineName, _ := cmd.Flags().GetString(commonParams.EngineName)
		var err error
		allEngineModel, err = engineWrapper.Get(engineName)
		if err != nil {
			return errors.Wrapf(err, "%s", "failed getting")
		}
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedGettingAll, errorModel.Code, errorModel.Message)
		} else if allEngineModel != nil {
			views, err := toEngineView(allEngineModel.Engines)
			if err != nil {
				return err
			}
			err = printByOutputFormat(cmd, views)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

// Converting the values of EngineResponseModel (returned from API) proper view which can be presented by format
func toEngineView(engines []wrappers.EngineResponseModel) ([]*EngineView, error) {

	views := make([]*EngineView, 0)
	for i := 0; i < len(engines); i++ {
		for j := 0; j < len(engines[i].Apis); j++ {
			views = append(views, &EngineView{
				EngineID:    engines[i].EngineID,
				EngineName:  engines[i].EngineName,
				ApiName:     engines[i].Apis[j].ApiName,
				ApiURL:      engines[i].Apis[j].ApiURL,
				Description: engines[i].Apis[j].Description,
			})
		}
	}

	return views, nil
}
