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
	failedGettingDastAlerts = "Failed getting DAST alerts"

	// API query param names
	pageQueryParam    = "page"
	perPageQueryParam = "per_page"
	searchQueryParam  = "search"
	sortByQueryParam  = "sort_by"
)

var (
	filterDastAlertsListFlagUsage = fmt.Sprintf(
		"Filter the list of DAST alerts. Use ';' as the delimiter for arrays. Available filters are: %s",
		strings.Join(
			[]string{
				pageQueryParam,
				perPageQueryParam,
				searchQueryParam,
				sortByQueryParam,
			}, ",",
		),
	)
)

// NewDastAlertsCommand creates the DAST alerts command
func NewDastAlertsCommand(dastAlertsWrapper wrappers.DastAlertsWrapper) *cobra.Command {
	alertsCmd := &cobra.Command{
		Use:   "dast-alerts",
		Short: "Manage DAST alerts",
		Long:  "The alerts command enables the ability to manage DAST alerts in Checkmarx One",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/todo
			`,
			),
		},
	}

	listDastAlertsCmd := &cobra.Command{
		Use:   "list",
		Short: "List DAST alerts for a specific environment and scan",
		Example: heredoc.Doc(
			`
			$ cx dast-alerts list --environment-id <env-id> --scan-id <scan-id>
			$ cx dast-alerts list --environment-id <env-id> --scan-id <scan-id> --format json
			$ cx dast-alerts list --environment-id <env-id> --scan-id <scan-id> --filter "page=1,per_page=10"
			$ cx dast-alerts list --environment-id <env-id> --scan-id <scan-id> --filter "search=PII,sort_by=severity:desc"
		`,
		),
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/todo
			`,
			),
		},
		RunE: runListDastAlertsCommand(dastAlertsWrapper),
	}

	// Required flags
	listDastAlertsCmd.Flags().String(params.EnvironmentIDFlag, "", "Environment ID (required)")
	listDastAlertsCmd.Flags().String(params.ScanIDFlag, "", "Scan ID (required)")
	_ = listDastAlertsCmd.MarkFlagRequired(params.EnvironmentIDFlag)
	_ = listDastAlertsCmd.MarkFlagRequired(params.ScanIDFlag)

	// Optional filter flag
	listDastAlertsCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterDastAlertsListFlagUsage)

	commandutils.AddFormatFlagToMultipleCommands(
		[]*cobra.Command{listDastAlertsCmd},
		printer.FormatTable,
		printer.FormatJSON,
		printer.FormatList,
	)

	alertsCmd.AddCommand(listDastAlertsCmd)
	return alertsCmd
}

func runListDastAlertsCommand(dastAlertsWrapper wrappers.DastAlertsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var alertsModel *wrappers.DastAlertsCollectionResponseModel
		var errorModel *wrappers.ErrorModel

		environmentID, _ := cmd.Flags().GetString(params.EnvironmentIDFlag)
		scanID, _ := cmd.Flags().GetString(params.ScanIDFlag)

		params, err := commandutils.GetFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingDastAlerts)
		}

		alertsModel, errorModel, err = dastAlertsWrapper.GetAlerts(environmentID, scanID, params)
		if err != nil {
			return errors.Wrapf(err, "%s\n", failedGettingDastAlerts)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf(services.ErrorCodeFormat, failedGettingDastAlerts, errorModel.Code, errorModel.Message)
		} else if alertsModel != nil && alertsModel.Results != nil {
			err = commandutils.PrintByFormat(cmd, toAlertViews(alertsModel.Results))
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func toAlertViews(models []wrappers.DastAlertResponseModel) []alertView {
	result := make([]alertView, len(models))
	for i := 0; i < len(models); i++ {
		result[i] = toAlertView(&models[i])
	}
	return result
}

func toAlertView(model *wrappers.DastAlertResponseModel) alertView {
	return alertView{
		AlertSimilarityID: model.AlertSimilarityID,
		State:             model.State,
		Severity:          model.Severity,
		Name:              model.Name,
		NumInstances:      model.NumInstances,
		Status:            model.Status,
		NumNotes:          model.NumNotes,
		Systemic:          model.Systemic,
	}
}

type alertView struct {
	AlertSimilarityID string `format:"name:Alert Similarity ID"`
	State             string
	Severity          string
	Name              string
	NumInstances      int  `format:"name:Instances"`
	Status            string
	NumNotes          int  `format:"name:Notes"`
	Systemic          bool
}
