package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedGettingEnvironments = "Failed getting environments"
)

var (
	filterEnvironmentsListFlagUsage = fmt.Sprintf(
		"Filter the list of environments. Use ';' as the delimiter for arrays. Available filters are: %s",
		strings.Join(
			[]string{
				commonParams.LimitQueryParam,
				commonParams.OffsetQueryParam,
				commonParams.FromDateQueryParam,
				commonParams.ToDateQueryParam,
				commonParams.SearchQueryParam,
				commonParams.SortQueryParam,
			}, ",",
		),
	)
)

// NewEnvironmentsCommand creates the environments command
func NewEnvironmentsCommand(environmentsWrapper wrappers.EnvironmentsWrapper) *cobra.Command {
	environmentsCmd := &cobra.Command{
		Use:   "environments",
		Short: "Manage DAST environments",
		Long:  "The environments command enables the ability to manage DAST environments in Checkmarx One",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68634-environments.html
			`,
			),
		},
	}

	listEnvironmentsCmd := &cobra.Command{
		Use:   "list",
		Short: "List all DAST environments in the system",
		Example: heredoc.Doc(
			`
			$ cx environments list --format list
			$ cx environments list --filter "from=2024-01-01,to=2024-12-31"
			$ cx environments list --filter "search=production,sort=created"
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68634-environments.html
			`,
			),
		},
		RunE: runListEnvironmentsCommand(environmentsWrapper),
	}
	listEnvironmentsCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterEnvironmentsListFlagUsage)

	addFormatFlagToMultipleCommands(
		[]*cobra.Command{listEnvironmentsCmd},
		printer.FormatTable,
		printer.FormatJSON,
		printer.FormatList,
	)

	environmentsCmd.AddCommand(listEnvironmentsCmd)
	return environmentsCmd
}

func runListEnvironmentsCommand(environmentsWrapper wrappers.EnvironmentsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var allEnvironmentsModel *wrappers.EnvironmentsCollectionResponseModel
		var errorModel *wrappers.ErrorModel

		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingEnvironments)
		}

		// Map filter parameters to API query parameters
		// The API expects: from, to, search, sort
		if from, ok := params[commonParams.FromDateQueryParam]; ok {
			params["from"] = from
			delete(params, commonParams.FromDateQueryParam)
		}
		if to, ok := params[commonParams.ToDateQueryParam]; ok {
			params["to"] = to
			delete(params, commonParams.ToDateQueryParam)
		}
		if search, ok := params[commonParams.SearchQueryParam]; ok {
			params["search"] = search
			delete(params, commonParams.SearchQueryParam)
		}
		if sort, ok := params[commonParams.SortQueryParam]; ok {
			params["sort"] = sort
			delete(params, commonParams.SortQueryParam)
		}

		allEnvironmentsModel, errorModel, err = environmentsWrapper.Get(params)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingEnvironments)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedGettingEnvironments, errorModel.Code, errorModel.Message)
		} else if allEnvironmentsModel != nil && allEnvironmentsModel.Environments != nil {
			err = printByFormat(cmd, toEnvironmentViews(allEnvironmentsModel.Environments))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func toEnvironmentViews(models []wrappers.EnvironmentResponseModel) []environmentView {
	result := make([]environmentView, len(models))
	for i := 0; i < len(models); i++ {
		result[i] = toEnvironmentView(models[i])
	}
	return result
}

func toEnvironmentView(model wrappers.EnvironmentResponseModel) environmentView {
	return environmentView{
		EnvironmentID: model.EnvironmentID,
		Domain:        model.Domain,
		URL:           model.URL,
		ScanType:      model.ScanType,
		Created:       model.Created,
		RiskRating:    model.RiskRating,
		LastScanTime:  model.LastScanTime,
		LastStatus:    model.LastStatus,
	}
}

type environmentView struct {
	EnvironmentID string `format:"name:Environment ID"`
	Domain        string
	URL           string
	ScanType      string `format:"name:Scan Type"`
	Created       string
	RiskRating    string `format:"name:Risk Rating"`
	LastScanTime  string `format:"name:Last Scan Time"`
	LastStatus    string `format:"name:Last Status"`
}

