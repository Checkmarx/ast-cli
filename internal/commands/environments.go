package commands

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/commandutils"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
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
				commonParams.FromQueryParam,
				commonParams.ToQueryParam,
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
				https://checkmarx.com/resource/documents/en/todo
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
			$ cx environments list --filter "from=1,to=10"
			$ cx environments list --filter "search=production,sort=created"
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/todo
			`,
			),
		},
		RunE: runListEnvironmentsCommand(environmentsWrapper),
	}
	listEnvironmentsCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterEnvironmentsListFlagUsage)

	commandutils.AddFormatFlagToMultipleCommands(
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

		params, err := commandutils.GetFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingEnvironments)
		}

		// The API expects: from, to, search, sort
		// from and to are pagination parameters (e.g., from=1, to=10 for first page)
		allEnvironmentsModel, errorModel, err = environmentsWrapper.Get(params)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingEnvironments)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedGettingEnvironments, errorModel.Code, errorModel.Message)
		} else if allEnvironmentsModel != nil && allEnvironmentsModel.Environments != nil {
			err = commandutils.PrintByFormat(cmd, toEnvironmentViews(allEnvironmentsModel.Environments))
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
		result[i] = toEnvironmentView(&models[i])
	}
	return result
}

func toEnvironmentView(model *wrappers.EnvironmentResponseModel) environmentView {
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
