package dast

import (
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/commandutils"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/params"
	commonParams "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedGettingDastScans  = "Failed getting DAST scans"
	environmentIDQueryParam = "environmentId"
	scanIDFilterKey         = "scanId"
	filterScanIDQueryParam  = "filter[scanId]"
)

var (
	filterDastScansListFlagUsage = fmt.Sprintf(
		"Filter the list of DAST scans. Use ';' as the delimiter for arrays. Available filters are: %s",
		strings.Join(
			[]string{
				commonParams.FromQueryParam,
				commonParams.ToQueryParam,
				commonParams.SortQueryParam,
				scanIDFilterKey,
			}, ",",
		),
	)
)

// NewDastScansCommand creates the DAST scans command
func NewDastScansCommand(dastScansWrapper wrappers.DastScansWrapper) *cobra.Command {
	scansCmd := &cobra.Command{
		Use:   "dast-scans",
		Short: "Manage DAST scans",
		Long:  "The scans command enables the ability to manage DAST scans in Checkmarx One",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/todo
			`,
			),
		},
	}

	listDastScansCmd := &cobra.Command{
		Use:   "list",
		Short: "List all DAST scans for an environment",
		Example: heredoc.Doc(
			`
			$ cx dast-scans list --environment-id <environment-id>
			$ cx dast-scans list --environment-id <environment-id> --format list
			$ cx dast-scans list --environment-id <environment-id> --filter "from=1,to=10"
			$ cx dast-scans list --environment-id <environment-id> --filter "scanId=<scan-id>"
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/todo
			`,
			),
		},
		RunE: runListDastScansCommand(dastScansWrapper),
	}
	listDastScansCmd.PersistentFlags().String(params.EnvironmentIDFlag, "", params.EnvironmentIDFlagUsage)
	_ = listDastScansCmd.MarkPersistentFlagRequired(params.EnvironmentIDFlag)
	listDastScansCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterDastScansListFlagUsage)

	commandutils.AddFormatFlagToMultipleCommands(
		[]*cobra.Command{listDastScansCmd},
		printer.FormatTable,
		printer.FormatJSON,
		printer.FormatList,
	)

	scansCmd.AddCommand(listDastScansCmd)
	return scansCmd
}

func runListDastScansCommand(dastScansWrapper wrappers.DastScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var allScansModel *wrappers.DastScansCollectionResponseModel
		var errorModel *wrappers.ErrorModel

		environmentID, err := cmd.Flags().GetString(params.EnvironmentIDFlag)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingDastScans)
		}

		params, err := commandutils.GetFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingDastScans)
		}

		// Add the required environmentId parameter
		params[environmentIDQueryParam] = environmentID

		// Transform scanId filter to filter[scanId] format for the API
		if scanID, exists := params[scanIDFilterKey]; exists {
			params[filterScanIDQueryParam] = scanID
			delete(params, scanIDFilterKey)
		}

		// The API expects: environmentId (required), from, to, sort, filter[scanId]
		allScansModel, errorModel, err = dastScansWrapper.Get(params)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingDastScans)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedGettingDastScans, errorModel.Code, errorModel.Message)
		} else if allScansModel != nil && allScansModel.Scans != nil {
			err = commandutils.PrintByFormat(cmd, toScanViews(allScansModel.Scans))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func toScanViews(models []wrappers.DastScanResponseModel) []scanView {
	result := make([]scanView, len(models))
	for i := 0; i < len(models); i++ {
		result[i] = toScanView(&models[i])
	}
	return result
}

func toScanView(model *wrappers.DastScanResponseModel) scanView {
	return scanView{
		ScanID:       model.ScanID,
		Initiator:    model.Initiator,
		ScanType:     model.ScanType,
		Created:      model.Created,
		RiskRating:   model.RiskRating,
		StartTime:    model.StartTime,
		ScanDuration: model.ScanDuration,
		LastStatus:   model.LastStatus,
		Statistics:   model.Statistics,
	}
}

type scanView struct {
	ScanID       string `format:"name:Scan ID"`
	Initiator    string
	ScanType     string `format:"name:Scan Type"`
	Created      string
	RiskRating   string `format:"name:Risk Rating"`
	StartTime    string `format:"name:Start Time"`
	ScanDuration int    `format:"name:Scan Duration"`
	LastStatus   string `format:"name:Last Status"`
	Statistics   string
}

