package dast

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/commandutils"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedGettingDastAlerts = "Failed getting DAST alerts"

	// Flag names
	environmentIDFlag = "environment-id"
	scanIDFlag        = "scan-id"
	pageFlag          = "page"
	perPageFlag       = "per-page"
	searchFlag        = "search"
	sortFlag          = "sort"

	// API query param names
	pageQueryParam    = "page"
	perPageQueryParam = "per_page"
	searchQueryParam  = "search"
	sortByQueryParam  = "sort_by"
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
			$ cx dast-alerts list --environment-id <env-id> --scan-id <scan-id> --page 1 --per-page 10
			$ cx dast-alerts list --environment-id <env-id> --scan-id <scan-id> --search "PII" --sort "severity:desc"
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
	listDastAlertsCmd.Flags().String(environmentIDFlag, "", "Environment ID (required)")
	listDastAlertsCmd.Flags().String(scanIDFlag, "", "Scan ID (required)")
	_ = listDastAlertsCmd.MarkFlagRequired(environmentIDFlag)
	_ = listDastAlertsCmd.MarkFlagRequired(scanIDFlag)

	// Optional flags
	listDastAlertsCmd.Flags().Int(pageFlag, 0, "Page number for pagination")
	listDastAlertsCmd.Flags().Int(perPageFlag, 0, "Number of results per page")
	listDastAlertsCmd.Flags().String(searchFlag, "", "Search term to filter alerts")
	listDastAlertsCmd.Flags().String(sortFlag, "", "Sort order (e.g., 'severity:desc')")

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

		environmentID, _ := cmd.Flags().GetString(environmentIDFlag)
		scanID, _ := cmd.Flags().GetString(scanIDFlag)

		params := buildQueryParams(cmd)

		alertsModel, errorModel, err := dastAlertsWrapper.GetAlerts(environmentID, scanID, params)
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

func buildQueryParams(cmd *cobra.Command) map[string]string {
	params := make(map[string]string)

	if page, _ := cmd.Flags().GetInt(pageFlag); page > 0 {
		params[pageQueryParam] = cmd.Flag(pageFlag).Value.String()
	}

	if perPage, _ := cmd.Flags().GetInt(perPageFlag); perPage > 0 {
		params[perPageQueryParam] = cmd.Flag(perPageFlag).Value.String()
	}

	if search, _ := cmd.Flags().GetString(searchFlag); search != "" {
		params[searchQueryParam] = search
	}

	if sort, _ := cmd.Flags().GetString(sortFlag); sort != "" {
		params[sortByQueryParam] = sort
	}

	return params
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

