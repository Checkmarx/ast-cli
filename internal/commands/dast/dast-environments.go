package dast

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
	failedGettingDastEnvironments = "Failed getting DAST environments"
)

var (
	filterDastEnvironmentsListFlagUsage = fmt.Sprintf(
		"Filter the list of DAST environments. Use ';' as the delimiter for arrays. Available filters are: %s",
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

// NewDastEnvironmentsCommand creates the DAST environments command
func NewDastEnvironmentsCommand(dastEnvironmentsWrapper wrappers.DastEnvironmentsWrapper) *cobra.Command {
	environmentsCmd := &cobra.Command{
		Use:   "dast-environments",
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

	listDastEnvironmentsCmd := &cobra.Command{
		Use:   "list",
		Short: "List all DAST environments in the system",
		Example: heredoc.Doc(
			`
			$ cx dast-environments list --format list
			$ cx dast-environments list --filter "from=1,to=10"
			$ cx dast-environments list --filter "search=production,sort=created"
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/todo
			`,
			),
		},
		RunE: runListDastEnvironmentsCommand(dastEnvironmentsWrapper),
	}
	listDastEnvironmentsCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterDastEnvironmentsListFlagUsage)

	commandutils.AddFormatFlagToMultipleCommands(
		[]*cobra.Command{listDastEnvironmentsCmd},
		printer.FormatTable,
		printer.FormatJSON,
		printer.FormatList,
	)

	environmentsCmd.AddCommand(listDastEnvironmentsCmd)
	return environmentsCmd
}

func runListDastEnvironmentsCommand(dastEnvironmentsWrapper wrappers.DastEnvironmentsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var allEnvironmentsModel *wrappers.DastEnvironmentsCollectionResponseModel
		var errorModel *wrappers.ErrorModel

		params, err := commandutils.GetFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingDastEnvironments)
		}

		// The API expects: from, to, search, sort
		// from and to are pagination parameters (e.g., from=1, to=10 for first page)
		allEnvironmentsModel, errorModel, err = dastEnvironmentsWrapper.Get(params)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingDastEnvironments)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedGettingDastEnvironments, errorModel.Code, errorModel.Message)
		} else if allEnvironmentsModel != nil && allEnvironmentsModel.Environments != nil {
			err = commandutils.PrintByFormat(cmd, toEnvironmentViews(allEnvironmentsModel.Environments))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func toEnvironmentViews(models []wrappers.DastEnvironmentResponseModel) []environmentView {
	result := make([]environmentView, len(models))
	for i := 0; i < len(models); i++ {
		result[i] = toEnvironmentView(&models[i])
	}
	return result
}

func toEnvironmentView(model *wrappers.DastEnvironmentResponseModel) environmentView {
	return environmentView{
		EnvironmentID: model.EnvironmentID,
		Domain:        model.Domain,
		URL:           model.URL,
		ScanType:      model.ScanType,
		Created:       model.Created,
		RiskRating:    model.RiskRating,
		LastScanID:    model.LastScanID,
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
	LastScanID    string `format:"name:Last Scan ID"`
	LastScanTime  string `format:"name:Last Scan Time"`
	LastStatus    string `format:"name:Last Status"`
}
