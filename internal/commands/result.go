package commands

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util"
	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	"github.com/checkmarx/ast-cli/internal/logger"
	"github.com/checkmarx/ast-cli/internal/services"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	commonParams "github.com/checkmarx/ast-cli/internal/params"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedCreatingSummary     = "Failed creating summary"
	failedGettingScan         = "Failed getting scan"
	failedListingResults      = "Failed listing results"
	failedListingCodeBashing  = "Failed codebashing link"
	mediumLabel               = "medium"
	criticalLabel             = "critical"
	highLabel                 = "high"
	lowLabel                  = "low"
	infoLabel                 = "info"
	sonarTypeLabel            = "_sonar"
	glSastTypeLabel           = ".gl-sast-report"
	glScaTypeLabel            = ".gl-sca-report"
	directoryPermission       = 0700
	infoSonar                 = "INFO"
	lowSonar                  = "LOW"
	mediumSonar               = "MEDIUM"
	highSonar                 = "HIGH"
	criticalSonar             = "BLOCKER"
	infoLowSarif              = "note"
	mediumSarif               = "warning"
	highSarif                 = "error"
	vulnerabilitySonar        = "SECURITY"
	cleanCodeAttribute        = "FORMATTED"
	infoCx                    = "INFO"
	lowCx                     = "LOW"
	mediumCx                  = "MEDIUM"
	highCx                    = "HIGH"
	criticalCx                = "CRITICAL"
	tableResultsFormat        = "              | %-10s   %6v   %5d   %6d   %5d   %4d   %-9s   |\n"
	stringTableResultsFormat  = "              | %-10s    %5s  %6s   %6s   %5s   %4s   %5s       |\n"
	TableTitleFormat          = "              | %-11s   %4s   %4s    %6s  %4s   %4s   %6s     |\n"
	twoNewLines               = "\n\n"
	tableLine                 = "              ---------------------------------------------------------------------     "
	codeBashingKey            = "cb-url"
	failedGettingBfl          = "Failed getting BFL"
	notAvailableString        = "-"
	disabledString            = "N/A"
	scanFailedString          = "Failed   "
	scanCanceledString        = "Canceled"
	scanSuccessString         = "Completed"
	scanPartialString         = "Partial"
	scsScanUnavailableString  = ""
	notAvailableNumber        = -1
	scanFailedNumber          = -2
	scanCanceledNumber        = -3
	scanPartialNumber         = -4
	defaultPaddingSize        = -13
	scanPendingMessage        = "Scan triggered in asynchronous mode or still running. Click more details to get the full status."
	directDependencyType      = "Direct Dependency"
	indirectDependencyType    = "Transitive Dependency"
	startedStatus             = "started"
	requestedStatus           = "requested"
	completedStatus           = "completed"
	pdfToEmailFlagDescription = "Send the PDF report to the specified email address." +
		" Use \",\" as the delimiter for multiple emails"
	pdfOptionsFlagDescription = "Sections to generate PDF report. Available options: Iac-Security,Sast,Sca," +
		defaultPdfOptionsDataSections
	sbomReportFlagDescription               = "Sections to generate SBOM report. Available options: CycloneDxJson,CycloneDxXml,SpdxJson"
	reportNameScanReport                    = "scan-report"
	reportNameImprovedScanReport            = "improved-scan-report"
	reportTypeEmail                         = "email"
	defaultPdfOptionsDataSections           = "ScanSummary,ExecutiveSummary,ScanResults"
	exploitablePathFlagDescription          = "Enable or disable exploitable path in scan. Available options: true,false"
	scaLastScanTimeFlagDescription          = "SCA last scan time. Available options: integer above 1"
	projectPrivatePackageFlagDescription    = "Enable or disable project private package. Available options: true,false"
	scaPrivatePackageVersionFlagDescription = "SCA project private package version. Example: 0.1.1"
	scaHideDevAndTestDepFlagDescription     = "Filter SCA results to exclude dev and test dependencies"
	policeManagementNoneStatus              = "none"
	apiDocumentationFlagDescription         = "Swagger folder/file filter for API-Security scan. Example: ./swagger.json"
	summaryCreatedAtLayout                  = "2006-01-02, 15:04:05"
	glTimeFormat                            = "2006-01-02T15:04:05"
	sarifNodeFileLength                     = 2
	fixLabel                                = "fix"
	redundantLabel                          = "redundant"
	delayValueForReport                     = 10
	fixLinkPrefix                           = "https://devhub.checkmarx.com/cve-details/"
	ScaDevAndTestExclusionParam             = "DEV_AND_TEST"
	ScaExcludeResultTypesParam              = "exclude-result-types"
	noFileForScorecardResultString          = "Issue Found in your GitHub repository"
	CliType                                 = "cli"
	artifactLocationURIString               = "This alert has no associated file"
)

var (
	summaryFormats = []string{
		printer.FormatSummaryConsole,
		printer.FormatSummary,
		printer.FormatSummaryJSON,
		printer.FormatPDF,
		printer.FormatSummaryMarkdown,
		printer.FormatSbom,
		printer.FormatGLSast,
		printer.FormatGLSca,
		printer.FormatSonar,
	}

	filterResultsListFlagUsage = fmt.Sprintf(
		"Filter the list of results. Use ';' as the delimiter for arrays. Available filters are: %s",
		strings.Join(
			[]string{
				commonParams.ScanIDQueryParam,
				commonParams.LimitQueryParam,
				commonParams.OffsetQueryParam,
				commonParams.SortQueryParam,
				commonParams.IncludeNodesQueryParam,
				commonParams.NodeIDsQueryParam,
				commonParams.QueryQueryParam,
				commonParams.GroupQueryParam,
				commonParams.StatusQueryParam,
				commonParams.SeverityQueryParam,
				commonParams.StateQueryParam,
			}, ",",
		),
	)

	// Follows: over 9.0 is critical, 7.0 to 8.9 is high, 4.0 to 6.9 is medium and 3.9 or less is low.
	securities = map[string]string{
		infoCx:     "1.0",
		lowCx:      "2.0",
		mediumCx:   "4.0",
		highCx:     "7.0",
		criticalCx: "9.0",
	}

	// Match cx severity with sonar severity
	sonarSeverities = map[string]string{
		infoCx:     infoSonar,
		lowCx:      lowSonar,
		mediumCx:   mediumSonar,
		highCx:     highSonar,
		criticalCx: criticalSonar,
	}

	containerEngineUnsupportedAgents = []string{
		commonParams.JetbrainsAgent, commonParams.VSCodeAgent, commonParams.VisualStudioAgent, commonParams.EclipseAgent,
	}

	sscsEngineToOverviewEngineMap = map[string]string{
		commonParams.SCSScorecardType:       commonParams.SCSScorecardOverviewType,
		commonParams.SCSSecretDetectionType: commonParams.SCSSecretDetectionOverviewType,
	}
)

func NewResultsCommand(
	resultsWrapper wrappers.ResultsWrapper,
	scanWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	codeBashingWrapper wrappers.CodeBashingWrapper,
	bflWrapper wrappers.BflWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	riskManagementWrapper wrappers.RiskManagementWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	policyWrapper wrappers.PolicyWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
	jwtWrapper wrappers.JWTWrapper,
) *cobra.Command {
	resultCmd := &cobra.Command{
		Use:   "results",
		Short: "Retrieve results",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.com/resource/documents/en/34965-68640-results.html
			`,
			),
		},
	}
	showResultCmd := resultShowSubCommand(resultsWrapper, scanWrapper, exportWrapper, resultsPdfReportsWrapper, resultsJSONReportsWrapper,
		risksOverviewWrapper, scsScanOverviewWrapper, policyWrapper, featureFlagsWrapper, jwtWrapper)
	codeBashingCmd := resultCodeBashing(codeBashingWrapper)
	bflResultCmd := resultBflSubCommand(bflWrapper)
	exitCodeSubcommand := exitCodeSubCommand(scanWrapper)
	riskManagementSubCommand := riskManagementSubCommand(riskManagementWrapper, featureFlagsWrapper)
	resultCmd.AddCommand(
		showResultCmd, bflResultCmd, codeBashingCmd, exitCodeSubcommand, riskManagementSubCommand,
	)
	return resultCmd
}

func exitCodeSubCommand(scanWrapper wrappers.ScansWrapper) *cobra.Command {
	exitCodeCmd := &cobra.Command{
		Use:   "exit-code",
		Short: "Get exit code and details of a scan",
		Long:  "The exit-code command enables you to get the exit code and failure details of a requested scan in Checkmarx One",
		Example: heredoc.Doc(
			`
			$ cx results exit-code --scan-id <scan Id> --scan-types <sast | sca | iac-security | apisec>
		`,
		),
		RunE: runGetExitCodeCommand(scanWrapper),
	}

	exitCodeCmd.PersistentFlags().String(commonParams.ScanIDFlag, "", "Scan ID")
	exitCodeCmd.PersistentFlags().String(commonParams.ScanTypes, "", "Scan types")

	return exitCodeCmd
}
func riskManagementSubCommand(riskManagement wrappers.RiskManagementWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) *cobra.Command {
	riskManagementCmd := &cobra.Command{
		Use:   "risk-management",
		Short: "Show risk-management results of a project",
		Long:  "The risk-management command displays risk management results for a specific project in Checkmarx One",
		Example: heredoc.Doc(
			`
			$ cx results risk-management --project-id <project Id> --scan-id <scan ID> --limit <limit> (1-50, default: 50)
		`,
		),
		RunE: runRiskManagementCommand(riskManagement, featureFlagsWrapper),
	}

	riskManagementCmd.PersistentFlags().String(commonParams.ProjectIDFlag, "", "Project ID")
	riskManagementCmd.PersistentFlags().String(commonParams.ScanIDFlag, "", "Scan ID")
	riskManagementCmd.PersistentFlags().Int(commonParams.LimitFlag, -1, "Limit")

	addFormatFlag(riskManagementCmd, printer.FormatJSON, printer.FormatTable, printer.FormatList)

	return riskManagementCmd
}

func resultShowSubCommand(
	resultsWrapper wrappers.ResultsWrapper,
	scanWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	policyWrapper wrappers.PolicyWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
	jwtWrapper wrappers.JWTWrapper,
) *cobra.Command {
	resultShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show results of a scan",
		Long:  "The show command enables the ability to show results about a requested scan in Checkmarx One",
		Example: heredoc.Doc(
			`
			$ cx results show --scan-id <scan Id>
		`,
		),
		RunE: runGetResultCommand(resultsWrapper, scanWrapper, exportWrapper, resultsPdfReportsWrapper, resultsJSONReportsWrapper, risksOverviewWrapper, scsScanOverviewWrapper, policyWrapper, featureFlagsWrapper, jwtWrapper),
	}
	addScanIDFlag(resultShowCmd, "ID to report on")
	addResultFormatFlag(
		resultShowCmd,
		printer.FormatJSON,
		printer.FormatJSONv2,
		printer.FormatSummary,
		printer.FormatSummaryConsole,
		printer.FormatSarif,
		printer.FormatSummaryJSON,
		printer.FormatSbom,
		printer.FormatPDF,
		printer.FormatSummaryMarkdown,
		printer.FormatGLSast,
		printer.FormatGLSca,
		printer.FormatSonar,
	)
	resultShowCmd.PersistentFlags().String(commonParams.ReportFormatPdfToEmailFlag, "", pdfToEmailFlagDescription)
	resultShowCmd.PersistentFlags().String(commonParams.ReportSbomFormatFlag, services.DefaultSbomOption, sbomReportFlagDescription)
	resultShowCmd.PersistentFlags().String(commonParams.ReportFormatPdfOptionsFlag, defaultPdfOptionsDataSections, pdfOptionsFlagDescription)
	resultShowCmd.PersistentFlags().String(commonParams.TargetFlag, "cx_result", "Output file")
	resultShowCmd.PersistentFlags().String(commonParams.TargetPathFlag, ".", "Output Path")
	resultShowCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterResultsListFlagUsage)

	resultShowCmd.PersistentFlags().IntP(
		commonParams.WaitDelayFlag,
		"",
		commonParams.WaitDelayDefault,
		"Polling wait time in seconds",
	)
	resultShowCmd.PersistentFlags().Int(
		commonParams.PolicyTimeoutFlag,
		commonParams.ResultPolicyDefaultTimeout,
		"Cancel the policy evaluation and fail after the timeout in minutes",
	)
	resultShowCmd.PersistentFlags().Bool(commonParams.IgnorePolicyFlag, false, "Skip policy evaluation. Requires override-policy-management permission.")
	resultShowCmd.PersistentFlags().Bool(commonParams.SastRedundancyFlag, false,
		"Populate SAST results 'data.redundancy' with values '"+fixLabel+"' (to fix) or '"+redundantLabel+"' (no need to fix)")
	resultShowCmd.PersistentFlags().Bool(commonParams.ScaHideDevAndTestDepFlag, false, scaHideDevAndTestDepFlagDescription)

	return resultShowCmd
}

func resultBflSubCommand(bflWrapper wrappers.BflWrapper) *cobra.Command {
	resultBflCmd := &cobra.Command{
		Use:   "bfl",
		Short: "Show best fix location for a query id within the scan result",
		Long:  "The bfl command enables the ability to show best fix location for a querid within the scan result",
		Example: heredoc.Doc(
			`
			$ cx results bfl --scan-id <scan Id> --query-id <query Id>
		`,
		),
		RunE: runGetBestFixLocationCommand(bflWrapper),
	}
	addScanIDFlag(resultBflCmd, "ID to report on")
	addQueryIDFlag(resultBflCmd, "Query Id from the result")
	addFormatFlag(resultBflCmd, printer.FormatList, printer.FormatJSON)

	markFlagAsRequired(resultBflCmd, commonParams.ScanIDFlag)
	markFlagAsRequired(resultBflCmd, commonParams.QueryIDFlag)

	return resultBflCmd
}

func runGetExitCodeCommand(scanWrapper wrappers.ScansWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		scanID, _ := cmd.Flags().GetString(commonParams.ScanIDFlag)
		if scanID == "" {
			return errors.New(errorConstants.ScanIDRequired)
		}
		scanTypesFlagValue, _ := cmd.Flags().GetString(commonParams.ScanTypes)
		results, err := GetScannerResults(scanWrapper, scanID, scanTypesFlagValue)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			return nil
		}

		return printer.Print(cmd.OutOrStdout(), results, printer.FormatIndentedJSON)
	}
}

func runRiskManagementCommand(riskManagement wrappers.RiskManagementWrapper, featureFlagsWrapper wrappers.FeatureFlagsWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		projectID, _ := cmd.Flags().GetString(commonParams.ProjectIDFlag)
		scanID, _ := cmd.Flags().GetString(commonParams.ScanIDFlag)

		limit, _ := cmd.Flags().GetInt(commonParams.LimitFlag)

		flagResponse, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, wrappers.RiskManagementEnabled)
		ASPMEnabled := flagResponse.Status
		if !ASPMEnabled {
			return errors.Errorf("%s", "Risk management results are currently unavailable for your tenant.")
		}
		results, err := getRiskManagementResults(riskManagement, projectID, scanID)
		if err != nil {
			return err
		}
		results.Results = utils.LimitSlice(results.Results, limit)
		err = printByFormat(cmd, results)
		return err
	}
}

func getRiskManagementResults(riskManagement wrappers.RiskManagementWrapper, projectID, scanID string) (*wrappers.ASPMResult, error) {
	ASPMResult, errorModel, err := riskManagement.GetTopVulnerabilitiesByProjectID(projectID, scanID)
	if err != nil {
		return nil, errors.Wrapf(err, "%s", failedListingResults)
	}
	if errorModel != nil {
		return nil, errors.Errorf("%s: CODE: %d, %s", failedListingResults, errorModel.Code, errorModel.Message)
	}
	return ASPMResult, nil
}

func GetScannerResults(scanWrapper wrappers.ScansWrapper, scanID, scanTypesFlagValue string) ([]ScannerResponse, error) {
	scanResponseModel, errorModel, err := scanWrapper.GetByID(scanID)
	if err != nil {
		return nil, errors.Wrapf(err, "%s", failedGetting)
	}
	if errorModel != nil {
		return nil, errors.Errorf("%s: CODE: %d, %s", failedGettingScan, errorModel.Code, errorModel.Message)
	}
	results := getScannerResponse(scanTypesFlagValue, scanResponseModel)
	return results, nil
}

func getScannerResponse(scanTypesFlagValue string, scanResponseModel *wrappers.ScanResponseModel) []ScannerResponse {
	var results []ScannerResponse

	if scanResponseModel.Status == wrappers.ScanCanceled ||
		scanResponseModel.Status == wrappers.ScanRunning ||
		scanResponseModel.Status == wrappers.ScanQueued ||
		scanResponseModel.Status == wrappers.ScanPartial ||
		scanResponseModel.Status == wrappers.ScanCompleted {
		result := ScannerResponse{
			ScanID: scanResponseModel.ID,
			Status: string(scanResponseModel.Status),
		}
		results = append(results, result)
		return results
	}

	if scanTypesFlagValue == "" {
		results = createAllFailedScannersResponse(scanResponseModel)
	} else {
		scanTypes := sanitizeScannerNames(scanTypesFlagValue)
		results = createRequestedScannersResponse(scanTypes, scanResponseModel)
	}

	return results
}

func createRequestedScannersResponse(scanTypes map[string]string, scanResponseModel *wrappers.ScanResponseModel) []ScannerResponse {
	var results []ScannerResponse
	for i := range scanResponseModel.StatusDetails {
		if _, ok := scanTypes[scanResponseModel.StatusDetails[i].Name]; ok {
			results = append(results, createScannerResponse(&scanResponseModel.StatusDetails[i]))
		}
	}
	return results
}

func createAllFailedScannersResponse(scanResponseModel *wrappers.ScanResponseModel) []ScannerResponse {
	var results []ScannerResponse
	for i := range scanResponseModel.StatusDetails {
		if scanResponseModel.StatusDetails[i].Status == wrappers.ScanFailed {
			results = append(results, createScannerResponse(&scanResponseModel.StatusDetails[i]))
		}
	}
	return results
}

func sanitizeScannerNames(scanTypes string) map[string]string {
	scanTypeSlice := strings.Split(scanTypes, ",")
	scanTypeMap := make(map[string]string)
	for i := range scanTypeSlice {
		lowered := strings.ToLower(scanTypeSlice[i])
		scanTypeMap[lowered] = lowered
	}

	return scanTypeMap
}

func createScannerResponse(statusDetails *wrappers.StatusInfo) ScannerResponse {
	return ScannerResponse{
		Name:      statusDetails.Name,
		Status:    statusDetails.Status,
		Details:   statusDetails.Details,
		ErrorCode: stringifyErrorCode(statusDetails.ErrorCode),
	}
}

func stringifyErrorCode(errorCode int) string {
	if errorCode == 0 {
		return ""
	}
	return strconv.Itoa(errorCode)
}

func runGetBestFixLocationCommand(bflWrapper wrappers.BflWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var bflResponseModel *wrappers.BFLResponseModel
		var errorModel *wrappers.WebError
		var err error

		scanID, _ := cmd.Flags().GetString(commonParams.ScanIDFlag)
		queryID, _ := cmd.Flags().GetString(commonParams.QueryIDFlag)

		scanIds := strings.Split(scanID, ",")
		if len(scanIds) > 1 {
			return errors.Errorf("%s", "Multiple scan-ids are not allowed.")
		}
		queryIds := strings.Split(queryID, ",")
		if len(queryIds) > 1 {
			return errors.Errorf("%s", "Multiple query-ids are not allowed.")
		}

		params := make(map[string]string)
		params[commonParams.ScanIDQueryParam] = scanID
		params[commonParams.QueryIDQueryParam] = queryID

		bflResponseModel, errorModel, err = bflWrapper.GetBflByScanIDAndQueryID(params)

		if err != nil {
			return errors.Wrapf(err, "%s", failedGettingBfl)
		}

		// Checking the response
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingBfl, errorModel.Code, errorModel.Message)
		} else if bflResponseModel != nil {
			err = printByFormat(cmd, toBflView(*bflResponseModel))
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func toBflView(bflResponseModel wrappers.BFLResponseModel) []wrappers.ScanResultNode {
	if (bflResponseModel.TotalCount) > 0 {
		views := make([]wrappers.ScanResultNode, bflResponseModel.TotalCount)

		for i := 0; i < bflResponseModel.TotalCount; i++ {
			views[i] = wrappers.ScanResultNode{
				Name:       bflResponseModel.Trees[i].BFL.Name,
				FileName:   bflResponseModel.Trees[i].BFL.FileName,
				FullName:   bflResponseModel.Trees[i].BFL.FullName,
				Column:     bflResponseModel.Trees[i].BFL.Column,
				Length:     bflResponseModel.Trees[i].BFL.Length,
				Line:       bflResponseModel.Trees[i].BFL.Line,
				MethodLine: bflResponseModel.Trees[i].BFL.MethodLine,
				Method:     bflResponseModel.Trees[i].BFL.Method,
				DomType:    bflResponseModel.Trees[i].BFL.DomType,
			}
		}
		return views
	}
	views := make([]wrappers.ScanResultNode, 0)
	return views
}

func resultCodeBashing(codeBashingWrapper wrappers.CodeBashingWrapper) *cobra.Command {
	// Create a codeBashing wrapper
	resultCmd := &cobra.Command{
		Use:   "codebashing",
		Short: "Get codebashing lesson link",
		Long:  "The codebashing command enables the ability to retrieve the link about a specific vulnerability",
		Example: heredoc.Doc(
			`
			$ cx results codebashing --language <string> --vulnerability-type <string> --cwe-id <string> --format <string>
		`,
		),
		RunE: runGetCodeBashingCommand(codeBashingWrapper),
	}
	resultCmd.PersistentFlags().String(commonParams.QueryIDFlag, "", "QueryId of vulnerability")
	err := resultCmd.MarkPersistentFlagRequired(commonParams.QueryIDFlag)
	if err != nil {
		log.Fatal(err)
	}
	addFormatFlag(resultCmd, printer.FormatJSON, printer.FormatTable, printer.FormatList)
	return resultCmd
}

func convertScanToResultsSummary(scanInfo *wrappers.ScanResponseModel, resultsWrapper wrappers.ResultsWrapper) (*wrappers.ResultSummary, error) {
	if scanInfo == nil {
		return nil, errors.New(failedCreatingSummary)
	}

	scanInfo.ReplaceMicroEnginesWithSCS()

	sastIssues := 0
	scaIssues := 0
	kicsIssues := 0
	var containersIssues *int
	var scsIssues *int
	enginesStatusCode := map[string]int{
		commonParams.SastType:       0,
		commonParams.ScaType:        0,
		commonParams.KicsType:       0,
		commonParams.APISecType:     0,
		commonParams.ScsType:        0,
		commonParams.ContainersType: 0,
	}
	if wrappers.IsContainersEnabled {
		containersIssues = new(int)
		*containersIssues = 0
		enginesStatusCode[commonParams.ContainersType] = 0
	}

	scsIssues = new(int)
	*scsIssues = 0
	enginesStatusCode[commonParams.ScsType] = 0

	if len(scanInfo.StatusDetails) > 0 {
		for _, statusDetailItem := range scanInfo.StatusDetails {
			if statusDetailItem.Status == wrappers.ScanFailed || statusDetailItem.Status == wrappers.ScanCanceled {
				if statusDetailItem.Name == commonParams.SastType {
					sastIssues = notAvailableNumber
				} else if statusDetailItem.Name == commonParams.ScaType {
					scaIssues = notAvailableNumber
				} else if statusDetailItem.Name == commonParams.KicsType {
					kicsIssues = notAvailableNumber
				} else if statusDetailItem.Name == commonParams.ScsType {
					*scsIssues = notAvailableNumber
				} else if statusDetailItem.Name == commonParams.ContainersType && wrappers.IsContainersEnabled {
					*containersIssues = notAvailableNumber
				}
			}
			switch statusDetailItem.Status {
			case wrappers.ScanFailed:
				handleScanStatus(statusDetailItem, enginesStatusCode, scanFailedNumber)
			case wrappers.ScanCanceled:
				handleScanStatus(statusDetailItem, enginesStatusCode, scanCanceledNumber)
			}
		}
	}
	summary := &wrappers.ResultSummary{
		ScanID:           scanInfo.ID,
		Status:           string(scanInfo.Status),
		CreatedAt:        scanInfo.CreatedAt.Format("2006-01-02, 15:04:05"),
		ProjectID:        scanInfo.ProjectID,
		RiskStyle:        "",
		RiskMsg:          "",
		CriticalIssues:   0,
		HighIssues:       0,
		MediumIssues:     0,
		LowIssues:        0,
		InfoIssues:       0,
		SastIssues:       sastIssues,
		KicsIssues:       kicsIssues,
		ScaIssues:        scaIssues,
		ScsIssues:        scsIssues,
		ContainersIssues: containersIssues,
		Tags:             scanInfo.Tags,
		ProjectName:      scanInfo.ProjectName,
		BranchName:       scanInfo.Branch,
		EnginesEnabled:   scanInfo.Engines,
		EnginesResult: map[string]*wrappers.EngineResultSummary{
			commonParams.SastType:       {StatusCode: enginesStatusCode[commonParams.SastType]},
			commonParams.ScaType:        {StatusCode: enginesStatusCode[commonParams.ScaType]},
			commonParams.KicsType:       {StatusCode: enginesStatusCode[commonParams.KicsType]},
			commonParams.APISecType:     {StatusCode: enginesStatusCode[commonParams.APISecType]},
			commonParams.ContainersType: {StatusCode: enginesStatusCode[commonParams.ContainersType]},
		},
	}
	if wrappers.IsContainersEnabled {
		summary.EnginesResult[commonParams.ContainersType] = &wrappers.EngineResultSummary{StatusCode: enginesStatusCode[commonParams.ContainersType]}
	}

	summary.EnginesResult[commonParams.ScsType] = &wrappers.EngineResultSummary{StatusCode: enginesStatusCode[commonParams.ScsType]}

	baseURI, err := resultsWrapper.GetResultsURL(summary.ProjectID)
	if err != nil {
		return nil, err
	}

	summary.BaseURI = baseURI
	summary.BaseURI = generateScanSummaryURL(summary)
	if isScanPending(summary.Status) {
		summary.ScanInfoMessage = scanPendingMessage
	}

	return summary, nil
}

func handleScanStatus(statusDetailItem wrappers.StatusInfo, targetTypes map[string]int, statusCode int) {
	if _, ok := targetTypes[statusDetailItem.Name]; ok {
		targetTypes[statusDetailItem.Name] = statusCode
	}
}

func summaryReport(
	summary *wrappers.ResultSummary,
	policies *wrappers.PolicyResponseModel,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
	results *wrappers.ScanResultsCollection,
) (*wrappers.ResultSummary, error) {
	if summary.HasAPISecurity() {
		apiSecRisks, err := getResultsForAPISecScanner(risksOverviewWrapper, summary.ScanID)
		if err != nil {
			return nil, err
		}
		summary.APISecurity = *apiSecRisks
	}

	if summary.HasSCS() {
		// Getting the base SCS overview. Results counts are overwritten in enhanceWithScanSummary->countResult
		SCSOverview, err := getScanOverviewForSCSScanner(scsScanOverviewWrapper, summary.ScanID)
		if err != nil {
			return nil, err
		}
		summary.SCSOverview = SCSOverview
	}

	if policies != nil {
		summary.Policies = filterViolatedRules(*policies)
	}

	enhanceWithScanSummary(summary, results, featureFlagsWrapper)

	setNotAvailableNumberIfZero(summary, &summary.SastIssues, commonParams.SastType)
	setNotAvailableNumberIfZero(summary, &summary.ScaIssues, commonParams.ScaType)
	setNotAvailableNumberIfZero(summary, &summary.KicsIssues, commonParams.KicsType)
	setNotAvailableNumberIfZero(summary, summary.ScsIssues, commonParams.ScsType)

	if wrappers.IsContainersEnabled {
		setNotAvailableNumberIfZero(summary, summary.ContainersIssues, commonParams.ContainersType)
	}

	setRiskMsgAndStyle(summary)
	setNotAvailableEnginesStatusCode(summary)

	return summary, nil
}

func setNotAvailableEnginesStatusCode(summary *wrappers.ResultSummary) {
	for engineName, engineResult := range summary.EnginesResult {
		setNotAvailableNumberIfZero(summary, &engineResult.StatusCode, engineName)
	}
}

func setRiskMsgAndStyle(summary *wrappers.ResultSummary) {
	if summary.CriticalIssues > 0 {
		summary.RiskStyle = criticalLabel
		summary.RiskMsg = "Critical Risk"
	} else if summary.HighIssues > 0 {
		summary.RiskStyle = highLabel
		summary.RiskMsg = "High Risk"
	} else if summary.MediumIssues > 0 {
		summary.RiskStyle = mediumLabel
		summary.RiskMsg = "Medium Risk"
	} else if summary.LowIssues > 0 {
		summary.RiskStyle = lowLabel
		summary.RiskMsg = "Low Risk"
	} else if summary.TotalIssues == 0 {
		summary.RiskMsg = "No Risk"
	}
}

func setNotAvailableNumberIfZero(summary *wrappers.ResultSummary, counter *int, engineType string) {
	if *counter == 0 && !contains(summary.EnginesEnabled, engineType) {
		*counter = notAvailableNumber
	}
}

func enhanceWithScanSummary(summary *wrappers.ResultSummary, results *wrappers.ScanResultsCollection, featureFlagsWrapper wrappers.FeatureFlagsWrapper) {
	for _, result := range results.Results {
		countResult(summary, result)
	}
	// Set critical count for a specific engine if critical is disabled
	flagResponse, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, wrappers.CVSSV3Enabled)
	criticalEnabled := flagResponse.Status
	if summary.HasAPISecurity() {
		summary.EnginesResult[commonParams.APISecType].Low = summary.APISecurity.Risks[3]
		summary.EnginesResult[commonParams.APISecType].Medium = summary.APISecurity.Risks[2]
		summary.EnginesResult[commonParams.APISecType].High = summary.APISecurity.Risks[1]
		if !criticalEnabled {
			summary.EnginesResult[commonParams.APISecType].Critical = notAvailableNumber
		} else {
			summary.EnginesResult[commonParams.APISecType].Critical = summary.APISecurity.Risks[0]
		}
	}

	summary.TotalIssues = summary.SastIssues + summary.ScaIssues + summary.KicsIssues + summary.GetAPISecurityDocumentationTotal()

	if summary.HasSCS() {
		// Special case for SCS where status is partial if any microengines failed
		if summary.SCSOverview.Status == scanPartialString {
			summary.EnginesResult[commonParams.ScsType].StatusCode = scanPartialNumber
		}
		if !criticalEnabled {
			summary.EnginesResult[commonParams.ScsType].Critical = notAvailableNumber
			removeCriticalFromSCSOverview(summary)
		}
		if *summary.ScsIssues >= 0 {
			summary.TotalIssues += *summary.ScsIssues
		}
	}
	if wrappers.IsContainersEnabled {
		if *summary.ContainersIssues >= 0 {
			summary.TotalIssues += *summary.ContainersIssues
		}
	}
	if !criticalEnabled {
		summary.EnginesResult[commonParams.SastType].Critical = notAvailableNumber
		summary.EnginesResult[commonParams.KicsType].Critical = notAvailableNumber
		summary.EnginesResult[commonParams.ScaType].Critical = notAvailableNumber
		summary.EnginesResult[commonParams.ContainersType].Critical = notAvailableNumber
	}
}

func removeCriticalFromSCSOverview(summary *wrappers.ResultSummary) {
	criticalCount := summary.SCSOverview.RiskSummary[criticalLabel]
	summary.SCSOverview.TotalRisksCount -= criticalCount
	summary.SCSOverview.RiskSummary[criticalLabel] = notAvailableNumber
	for _, microEngineOverview := range summary.SCSOverview.MicroEngineOverviews {
		if microEngineOverview.RiskSummary != nil && microEngineOverview.RiskSummary[criticalLabel] != nil {
			engineCriticalCount := microEngineOverview.RiskSummary[criticalLabel]
			microEngineOverview.TotalRisks -= engineCriticalCount.(int)
			microEngineOverview.RiskSummary[criticalLabel] = disabledString
		}
	}
}

func writeHTMLSummary(targetFile string, summary *wrappers.ResultSummary) error {
	log.Println("Creating Summary Report: ", targetFile)
	summaryTemp, err := template.New("summaryTemplate").Parse(wrappers.SummaryTemplate(isScanPending(summary.Status)))
	if err == nil {
		f, err := os.Create(targetFile)
		if err == nil {
			_ = summaryTemp.ExecuteTemplate(f, "SummaryTemplate", summary)
			_ = f.Close()
		}
		return err
	}
	return nil
}
func writeMarkdownSummary(targetFile string, data *wrappers.ResultSummary) error {
	log.Println("Creating Markdown Summary Report: ", targetFile)
	tmpl, err := template.New(printer.FormatSummaryMarkdown).Parse(wrappers.SummaryMarkdownTemplate(isScanPending(data.Status)))
	if err != nil {
		return err
	}
	file, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer file.Close()

	err = tmpl.Execute(file, &data)
	if err != nil {
		return err
	}
	return nil
}

// nolint: whitespace
func writeConsoleSummary(summary *wrappers.ResultSummary, featureFlagsWrapper wrappers.FeatureFlagsWrapper, ignorePolicyFlagOmit bool) error {
	if !isScanPending(summary.Status) {
		fmt.Printf("            Scan Summary:                     \n")
		fmt.Printf("              Created At: %s\n", summary.CreatedAt)
		fmt.Printf("              Project Name: %s                        \n", summary.ProjectName)
		fmt.Printf("              Scan ID: %s                             \n\n", summary.ScanID)
		fmt.Printf("            Results Summary:                     \n")
		fmt.Printf(
			"              Risk Level: %s																									 \n",
			summary.RiskMsg,
		)
		if summary.Policies != nil && !strings.EqualFold(summary.Policies.Status, policeManagementNoneStatus) {
			printPoliciesSummary(summary, ignorePolicyFlagOmit)
		}

		printResultsSummaryTable(summary)

		if summary.HasAPISecurity() {
			printAPIsSecuritySummary(summary)
		}

		if summary.HasSCS() {
			printSCSSummary(summary.SCSOverview.MicroEngineOverviews, featureFlagsWrapper)
		}

		fmt.Printf("              Checkmarx One - Scan Summary & Details: %s\n", summary.BaseURI)
	} else {
		fmt.Printf("Scan executed in asynchronous mode or still running. Hence, no results generated.\n")
		fmt.Printf("For more information: %s\n", summary.BaseURI)
	}
	return nil
}

func printPoliciesSummary(summary *wrappers.ResultSummary, ignorePolicyFlagOmit bool) {
	hasViolations := false
	for _, policy := range summary.Policies.Policies {
		if len(policy.RulesViolated) > 0 {
			hasViolations = true
			break
		}
	}
	if hasViolations {
		fmt.Printf(tableLine + "\n")
		if ignorePolicyFlagOmit {
			printWarningIfIgnorePolicyOmiited()
		}
		if summary.Policies.BreakBuild {
			fmt.Printf("            Policy Management Violation - Break Build Enabled:                     \n")
		} else {
			fmt.Printf("            Policy Management Violation:                     \n")
		}
		for _, police := range summary.Policies.Policies {
			if len(police.RulesViolated) > 0 {
				fmt.Printf("              Policy: %s | Break Build: %t | Violated Rules: ", police.Name, police.BreakBuild)
				for _, violatedRule := range police.RulesViolated {
					fmt.Printf("%s;", violatedRule)
				}
			}
			fmt.Printf("\n")
		}
		fmt.Printf("\n")
	}
}

func printAPIsSecuritySummary(summary *wrappers.ResultSummary) {
	fmt.Printf("              API Security - Total Detected APIs: %d                       \n", summary.APISecurity.APICount)
	fmt.Printf("              APIS WITH RISK: %*d \n", defaultPaddingSize, summary.APISecurity.TotalRisksCount)
	if summary.HasAPISecurityDocumentation() {
		fmt.Printf("              APIS DOCUMENTATION: %*d \n", defaultPaddingSize, summary.GetAPISecurityDocumentationTotal())
	}
	fmt.Printf(tableLine + twoNewLines)
}

func printTableRow(title string, counts *wrappers.EngineResultSummary, statusNumber int) {
	switch statusNumber {
	case notAvailableNumber:
		fmt.Printf(stringTableResultsFormat, title, notAvailableString, notAvailableString, notAvailableString, notAvailableString, notAvailableString, notAvailableString)
	case scanFailedNumber:
		fmt.Printf(tableResultsFormat, title, getCountValue(counts.Critical), counts.High, counts.Medium, counts.Low, counts.Info, scanFailedString)
	case scanCanceledNumber:
		fmt.Printf(tableResultsFormat, title, getCountValue(counts.Critical), counts.High, counts.Medium, counts.Low, counts.Info, scanCanceledString)
	case scanPartialNumber:
		fmt.Printf(tableResultsFormat, title, getCountValue(counts.Critical), counts.High, counts.Medium, counts.Low, counts.Info, scanPartialString)
	default:
		fmt.Printf(tableResultsFormat, title, getCountValue(counts.Critical), counts.High, counts.Medium, counts.Low, counts.Info, scanSuccessString)
	}
}

func printSCSSummary(microEngineOverviews []*wrappers.MicroEngineOverview, featureFlagsWrapper wrappers.FeatureFlagsWrapper) {
	fmt.Printf("              Supply Chain Security Results\n")
	fmt.Printf("              --------------------------------------------------------------------------     \n")
	fmt.Println("              |                      Critical   High   Medium   Low   Info   Status    |")
	for _, microEngineOverview := range microEngineOverviews {
		printSCSTableRow(microEngineOverview, featureFlagsWrapper)
	}
	fmt.Printf("              --------------------------------------------------------------------------     \n\n")
}

func printSCSTableRow(microEngineOverview *wrappers.MicroEngineOverview, featureFlagsWrapper wrappers.FeatureFlagsWrapper) {
	formatString := "              | %-20s   %4v   %4v   %6v   %4v   %4v   %-9s  |\n"
	notAvailableFormatString := "              | %-20s   %4v   %4s   %6s   %4s   %4s   %5s      |\n"

	riskSummary := microEngineOverview.RiskSummary
	microEngineName := microEngineOverview.FullName

	switch microEngineOverview.Status {
	case scsScanUnavailableString:
		fmt.Printf(notAvailableFormatString, microEngineName, notAvailableString, notAvailableString, notAvailableString, notAvailableString, notAvailableString, notAvailableString)
	default:
		fmt.Printf(formatString, microEngineName, riskSummary[criticalLabel], riskSummary[highLabel], riskSummary[mediumLabel], riskSummary[lowLabel],
			riskSummary[infoLabel], microEngineOverview.Status)
	}
}

func getCountValue(count int) interface{} {
	if count < 0 {
		return disabledString
	}
	return count
}

func printResultsSummaryTable(summary *wrappers.ResultSummary) {
	totalCriticalIssues := summary.EnginesResult.GetCriticalIssues()
	totalHighIssues := summary.EnginesResult.GetHighIssues()
	totalMediumIssues := summary.EnginesResult.GetMediumIssues()
	totalLowIssues := summary.EnginesResult.GetLowIssues()
	totalInfoIssues := summary.EnginesResult.GetInfoIssues()
	fmt.Printf(tableLine + twoNewLines)
	fmt.Printf("              Total Results: %d                       \n", summary.TotalIssues)
	fmt.Println(tableLine)
	fmt.Printf(TableTitleFormat, "   ", "Critical", "High", "Medium", "Low", "Info", "Status")

	printTableRow("APIs", summary.EnginesResult[commonParams.APISecType], summary.EnginesResult[commonParams.APISecType].StatusCode)
	printTableRow("IAC", summary.EnginesResult[commonParams.KicsType], summary.EnginesResult[commonParams.KicsType].StatusCode)
	printTableRow("SAST", summary.EnginesResult[commonParams.SastType], summary.EnginesResult[commonParams.SastType].StatusCode)
	printTableRow("SCA", summary.EnginesResult[commonParams.ScaType], summary.EnginesResult[commonParams.ScaType].StatusCode)
	printTableRow("SCS", summary.EnginesResult[commonParams.ScsType], summary.EnginesResult[commonParams.ScsType].StatusCode)

	if wrappers.IsContainersEnabled {
		printTableRow("CONTAINERS", summary.EnginesResult[commonParams.ContainersType], summary.EnginesResult[commonParams.ContainersType].StatusCode)
	}

	fmt.Println(tableLine)
	fmt.Printf(tableResultsFormat,
		"TOTAL", getCountValue(totalCriticalIssues), totalHighIssues, totalMediumIssues, totalLowIssues, totalInfoIssues, summary.Status)
	fmt.Printf(tableLine + twoNewLines)
}

func generateScanSummaryURL(summary *wrappers.ResultSummary) string {
	summaryURL := fmt.Sprintf(
		strings.Replace(summary.BaseURI, "overview", "scans?id=%s&branch=%s", 1),
		summary.ScanID, url.QueryEscape(summary.BranchName),
	)
	return summaryURL
}

func runGetResultCommand(
	resultsWrapper wrappers.ResultsWrapper,
	scanWrapper wrappers.ScansWrapper,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	policyWrapper wrappers.PolicyWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
	jwtWrapper wrappers.JWTWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		targetFile, _ := cmd.Flags().GetString(commonParams.TargetFlag)
		targetPath, _ := cmd.Flags().GetString(commonParams.TargetPathFlag)
		format, _ := cmd.Flags().GetString(commonParams.TargetFormatFlag)
		formatPdfToEmail, _ := cmd.Flags().GetString(commonParams.ReportFormatPdfToEmailFlag)
		formatPdfOptions, _ := cmd.Flags().GetString(commonParams.ReportFormatPdfOptionsFlag)
		formatSbomOptions, _ := cmd.Flags().GetString(commonParams.ReportSbomFormatFlag)
		sastRedundancy, _ := cmd.Flags().GetBool(commonParams.SastRedundancyFlag)
		agent, _ := cmd.Flags().GetString(commonParams.AgentFlag)
		scaHideDevAndTestDep, _ := cmd.Flags().GetBool(commonParams.ScaHideDevAndTestDepFlag)
		ignorePolicy, _ := cmd.Flags().GetBool(commonParams.IgnorePolicyFlag)
		// Check if the user has permission to override policy management if --ignore-policy is set
		ignorePolicyFlagOmit := false
		if ignorePolicy {
			overridePolicyManagementPer, err := jwtWrapper.CheckPermissionByAccessToken(OverridePolicyManagement)
			if err != nil {
				return err
			}
			if !overridePolicyManagementPer {
				ignorePolicyFlagOmit = true
				ignorePolicy = false
			}
		}
		waitDelay, _ := cmd.Flags().GetInt(commonParams.WaitDelayFlag)
		policyTimeout, _ := cmd.Flags().GetInt(commonParams.PolicyTimeoutFlag)

		scanID, _ := cmd.Flags().GetString(commonParams.ScanIDFlag)
		if scanID == "" {
			return errors.Errorf("%s: Please provide a scan ID", failedListingResults)
		}

		resultsParams, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingResults)
		}

		if scaHideDevAndTestDep {
			resultsParams[ScaExcludeResultTypesParam] = ScaDevAndTestExclusionParam
		}

		scan, errorModel, scanErr := scanWrapper.GetByID(scanID)
		if scanErr != nil {
			return errors.Wrapf(scanErr, "%s", failedGetting)
		}
		if errorModel != nil {
			return errors.Errorf("%s: CODE: %d, %s", failedGettingScan, errorModel.Code, errorModel.Message)
		}

		var policyResponseModel *wrappers.PolicyResponseModel
		if !isScanPending(string(scan.Status)) {
			policyResponseModel, err = services.HandlePolicyEvaluation(cmd, policyWrapper, scan, ignorePolicy, agent, waitDelay, policyTimeout)
			if err != nil {
				return err
			}
		} else {
			logger.PrintIfVerbose("Policy violations aren't returned in the pipeline for scans run in async mode.")
		}

		if sastRedundancy {
			resultsParams[commonParams.SastRedundancyFlag] = ""
		}

		_, err = CreateScanReport(resultsWrapper, risksOverviewWrapper, scsScanOverviewWrapper, exportWrapper,
			policyResponseModel, resultsPdfReportsWrapper, resultsJSONReportsWrapper, scan, format, formatPdfToEmail, formatPdfOptions,
			formatSbomOptions, targetFile, targetPath, agent, resultsParams, featureFlagsWrapper, ignorePolicyFlagOmit)
		return err
	}
}

func runGetCodeBashingCommand(
	codeBashingWrapper wrappers.CodeBashingWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		//cmd.Flags().GetString(commonParams.LanguageFlag)
		//1089818565155602739
		queryId, _ := cmd.Flags().GetString(commonParams.QueryIDFlag)
		// Fetch the cached token or a new one to obtain the codebashing URL incoded in the jwt token
		_, err := codeBashingWrapper.BuildCodeBashingParams(
			wrappers.CodeBashingParamsCollection{
				QueryId: queryId,
			},
		)

		codeBashingURL, err := codeBashingWrapper.GetCodeBashingURL(codeBashingKey)
		if err != nil {
			return err
		}
		// Make the request to the api to obtain the codebashing link and send the codebashing url to enrich the path
		CodeBashingModel, webError, err := codeBashingWrapper.GetCodeBashingLinks(queryId, codeBashingURL)
		if err != nil {
			return err
		}
		if webError != nil {
			return errors.New(webError.Message)
		}
		err = printByFormat(cmd, *CodeBashingModel)
		if CodeBashingModel != nil {
			model := *CodeBashingModel
			if len(model) > 0 && model[0].Path != "" {
				logger.Printf("CodeBashing lesson available at: %s", model[0].Path)
			}
		}
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingCodeBashing)
		}
		return nil
	}
}

func setIsContainersEnabled(agent string) {
	wrappers.IsContainersEnabled = !containsIgnoreCase(containerEngineUnsupportedAgents, agent)
}

func filterResultsByType(results *wrappers.ScanResultsCollection, excludedTypes map[string]struct{}) *wrappers.ScanResultsCollection {
	var filteredResults []*wrappers.ScanResult

	for _, result := range results.Results {
		if _, shouldExclude := excludedTypes[result.Type]; shouldExclude {
			results.TotalCount--
		} else {
			filteredResults = append(filteredResults, result)
		}
	}
	results.Results = filteredResults
	return results
}

func filterScsResultsByAgent(results *wrappers.ScanResultsCollection, agent string) *wrappers.ScanResultsCollection {
	unsupportedTypesByAgent := map[string][]string{
		commonParams.VSCodeAgent:       {commonParams.SCSScorecardType},
		commonParams.JetbrainsAgent:    {commonParams.SCSScorecardType},
		commonParams.EclipseAgent:      {commonParams.SCSScorecardType, commonParams.SCSSecretDetectionType},
		commonParams.VisualStudioAgent: {commonParams.SCSScorecardType},
	}

	excludedTypes := make(map[string]struct{})

	if typesToExclude, exists := unsupportedTypesByAgent[agent]; exists {
		for _, excludeType := range typesToExclude {
			excludedTypes[excludeType] = struct{}{}
		}
	}

	results = filterResultsByType(results, excludedTypes)

	return results
}

func CreateScanReport(
	resultsWrapper wrappers.ResultsWrapper,
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	exportWrapper wrappers.ExportWrapper,
	policyResponseModel *wrappers.PolicyResponseModel,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	scan *wrappers.ScanResponseModel,
	reportTypes,
	formatPdfToEmail,
	formatPdfOptions,
	formatSbomOptions,
	targetFile,
	targetPath string,
	agent string,
	resultsParams map[string]string,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
	ignorePolicyFlagOmit bool,
) (*wrappers.ScanResultsCollection, error) {
	reportList := strings.Split(reportTypes, ",")
	results := &wrappers.ScanResultsCollection{}
	setIsContainersEnabled(agent)
	summary, err := convertScanToResultsSummary(scan, resultsWrapper)
	if err != nil {
		return nil, err
	}
	scanPending := isScanPending(summary.Status)

	err = createDirectory(targetPath)
	if err != nil {
		return nil, err
	}
	if !scanPending {
		results, err = ReadResults(resultsWrapper, exportWrapper, scan, resultsParams, agent, featureFlagsWrapper)
		if err != nil {
			return nil, err
		}
	}
	isSummaryNeeded := verifyFormatsByReportList(reportList, summaryFormats...)
	if isSummaryNeeded && !scanPending {
		summary, err = summaryReport(summary, policyResponseModel, risksOverviewWrapper, scsScanOverviewWrapper, featureFlagsWrapper, results)
		if err != nil {
			return nil, err
		}
	}
	for _, reportType := range reportList {
		err = createReport(reportType, formatPdfToEmail, formatPdfOptions, formatSbomOptions, targetFile,
			targetPath, results, summary, exportWrapper, resultsPdfReportsWrapper, resultsJSONReportsWrapper, featureFlagsWrapper, ignorePolicyFlagOmit)
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}

func countResult(summary *wrappers.ResultSummary, result *wrappers.ScanResult) {
	engineType := strings.TrimSpace(result.Type)
	severity := strings.ToLower(result.Severity)
	if contains(summary.EnginesEnabled, engineType) && isExploitable(result.State) {
		if engineType == commonParams.SastType {
			summary.SastIssues++
			summary.TotalIssues++
		} else if engineType == commonParams.ScaType {
			summary.ScaIssues++
			summary.TotalIssues++
		} else if engineType == commonParams.KicsType {
			summary.KicsIssues++
			summary.TotalIssues++
		} else if engineType == commonParams.ContainersType {
			if wrappers.IsContainersEnabled {
				*summary.ContainersIssues++
				summary.TotalIssues++
			} else {
				return
			}
		} else if strings.HasPrefix(engineType, commonParams.SscsType) {
			addResultToSCSOverview(summary, result)
			engineType = commonParams.ScsType
			*summary.ScsIssues++
			summary.TotalIssues++
		} else {
			return
		}

		switch severity {
		case criticalLabel:
			summary.CriticalIssues++
		case highLabel:
			summary.HighIssues++
		case mediumLabel:
			summary.MediumIssues++
		case lowLabel:
			summary.LowIssues++
		case infoLabel:
			summary.InfoIssues++
		}

		summary.UpdateEngineResultSummary(engineType, severity)
	}
}

func addResultToSCSOverview(summary *wrappers.ResultSummary, result *wrappers.ScanResult) {
	if engineOverviewName, engineExists := sscsEngineToOverviewEngineMap[result.Type]; engineExists {
		for _, microEngineOverview := range summary.SCSOverview.MicroEngineOverviews {
			if microEngineOverview.Name == engineOverviewName {
				if microEngineOverview.RiskSummary != nil {
					severity := strings.ToLower(result.Severity)
					if severityCount, exists := microEngineOverview.RiskSummary[severity]; exists {
						summary.SCSOverview.RiskSummary[severity]++
						microEngineOverview.TotalRisks++
						summary.SCSOverview.TotalRisksCount++
						microEngineOverview.RiskSummary[severity] = severityCount.(int) + 1
					}
				}
			}
		}
	}
}

func verifyFormatsByReportList(reportFormats []string, formats ...string) bool {
	for _, reportFormat := range reportFormats {
		for _, format := range formats {
			if printer.IsFormat(reportFormat, format) {
				return true
			}
		}
	}
	return false
}

func validateEmails(emailString string) ([]string, error) {
	re := regexp.MustCompile(`^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+$`)
	emails := strings.Split(emailString, ",")
	var validEmails []string
	for _, emailStr := range emails {
		email := strings.TrimSpace(emailStr)
		if re.MatchString(email) {
			validEmails = append(validEmails, email)
		} else {
			return nil, errors.Errorf("report not sent, invalid email address: %s", email)
		}
	}
	return validEmails, nil
}

func getResultsForAPISecScanner(
	risksOverviewWrapper wrappers.RisksOverviewWrapper,
	scanID string,
) (results *wrappers.APISecResult, err error) {
	var apiSecResultsModel *wrappers.APISecResult
	var errorModel *wrappers.WebError

	apiSecResultsModel, errorModel, err = risksOverviewWrapper.GetAllAPISecRisksByScanID(scanID)
	if err != nil {
		return nil, errors.Wrapf(err, "%s", failedListingResults)
	}
	if errorModel != nil {
		return nil, errors.Errorf("%s: CODE: %d, %s", failedListingResults, errorModel.Code, errorModel.Message)
	} else if apiSecResultsModel != nil {
		return apiSecResultsModel, nil
	}
	return nil, nil
}

func getScanOverviewForSCSScanner(
	scsScanOverviewWrapper wrappers.ScanOverviewWrapper,
	scanID string,
) (results *wrappers.SCSOverview, err error) {
	var scsOverview *wrappers.SCSOverview
	var errorModel *wrappers.WebError

	scsOverview, errorModel, err = scsScanOverviewWrapper.GetSCSOverviewByScanID(scanID)
	if err != nil {
		return nil, errors.Wrapf(err, "SCS: %s", failedListingResults)
	}
	if errorModel != nil {
		return nil, errors.Errorf("SCS: %s: CODE: %d, %s", failedListingResults, errorModel.Code, errorModel.Message)
	} else if scsOverview != nil {
		// Setting all counts to 0. Results are recounted in enhanceWithScanSummary->countResult
		scsOverview.TotalRisksCount = 0
		for key := range scsOverview.RiskSummary {
			scsOverview.RiskSummary[key] = 0
		}
		for _, microEngineOverview := range scsOverview.MicroEngineOverviews {
			microEngineOverview.TotalRisks = 0
			if microEngineOverview.RiskSummary != nil {
				for severity := range microEngineOverview.RiskSummary {
					microEngineOverview.RiskSummary[severity] = 0
				}
			}
		}
		return scsOverview, nil
	}
	return nil, nil
}

func isScanPending(scanStatus string) bool {
	return !(strings.EqualFold(scanStatus, "Completed") || strings.EqualFold(
		scanStatus,
		"Partial",
	) || strings.EqualFold(scanStatus, "Failed"))
}

func isValidScanStatus(status, format string) bool {
	if isScanPending(status) {
		log.Printf("Result format file %s not create because scan status is %s", format, status)
		return false
	}
	return true
}

func createReport(format,
	formatPdfToEmail,
	formatPdfOptions,
	formatSbomOptions,
	targetFile,
	targetPath string,
	results *wrappers.ScanResultsCollection,
	summary *wrappers.ResultSummary,
	exportWrapper wrappers.ExportWrapper,
	resultsPdfReportsWrapper wrappers.ResultsPdfWrapper,
	resultsJSONReportsWrapper wrappers.ResultsJSONWrapper,
	featureFlagsWrapper wrappers.FeatureFlagsWrapper,
	ignorePolicyFlagOmit bool) error {
	if printer.IsFormat(format, printer.FormatIndentedJSON) {
		return nil
	}
	if printer.IsFormat(format, printer.FormatSarif) && isValidScanStatus(summary.Status, printer.FormatSarif) {
		sarifRpt := createTargetName(targetFile, targetPath, printer.FormatSarif)
		return exportSarifResults(sarifRpt, results)
	}
	if printer.IsFormat(format, printer.FormatSonar) && isValidScanStatus(summary.Status, printer.FormatSonar) {
		sonarRpt := createTargetName(fmt.Sprintf("%s%s", targetFile, sonarTypeLabel), targetPath, printer.FormatJSON)
		return exportSonarResults(sonarRpt, results)
	}
	if printer.IsFormat(format, printer.FormatJSON) && isValidScanStatus(summary.Status, printer.FormatJSON) {
		jsonRpt := createTargetName(targetFile, targetPath, printer.FormatJSON)
		return exportJSONResults(jsonRpt, results)
	}
	if printer.IsFormat(format, printer.FormatJSONv2) && isValidScanStatus(summary.Status, printer.FormatJSONv2) {
		summaryRpt := createTargetName(targetFile, targetPath, printer.FormatJSON)
		return exportJSONReportResults(resultsJSONReportsWrapper, summary, summaryRpt, featureFlagsWrapper)
	}
	if printer.IsFormat(format, printer.FormatGLSast) {
		jsonRpt := createTargetName(fmt.Sprintf("%s%s", targetFile, glSastTypeLabel), targetPath, printer.FormatJSON)
		return exportGlSastResults(jsonRpt, results, summary)
	}
	if printer.IsFormat(format, printer.FormatGLSca) {
		jsonRpt := createTargetName(fmt.Sprintf("%s%s", targetFile, glScaTypeLabel), targetPath, printer.FormatJSON)
		return exportGlScaResults(jsonRpt, results, summary)
	}

	if printer.IsFormat(format, printer.FormatSummaryConsole) {
		return writeConsoleSummary(summary, featureFlagsWrapper, ignorePolicyFlagOmit)
	}
	if printer.IsFormat(format, printer.FormatSummary) {
		summaryRpt := createTargetName(targetFile, targetPath, printer.FormatHTML)
		convertNotAvailableNumberToZero(summary)
		return writeHTMLSummary(summaryRpt, summary)
	}
	if printer.IsFormat(format, printer.FormatSummaryJSON) {
		summaryRpt := createTargetName(targetFile, targetPath, printer.FormatJSON)
		convertNotAvailableNumberToZero(summary)
		return exportJSONSummaryResults(summaryRpt, summary)
	}
	if printer.IsFormat(format, printer.FormatPDF) && isValidScanStatus(summary.Status, printer.FormatPDF) {
		summaryRpt := createTargetName(targetFile, targetPath, printer.FormatPDF)
		return exportPdfResults(resultsPdfReportsWrapper, summary, summaryRpt, formatPdfToEmail, formatPdfOptions, featureFlagsWrapper)
	}
	if printer.IsFormat(format, printer.FormatSummaryMarkdown) {
		summaryRpt := createTargetName(targetFile, targetPath, "md")
		convertNotAvailableNumberToZero(summary)
		return writeMarkdownSummary(summaryRpt, summary)
	}
	if printer.IsFormat(format, printer.FormatSbom) && isValidScanStatus(summary.Status, printer.FormatSbom) {
		targetType := printer.FormatJSON
		if strings.Contains(strings.ToLower(formatSbomOptions), printer.FormatXML) {
			targetType = printer.FormatXML
		}
		summaryRpt := createTargetName(fmt.Sprintf("%s_%s", targetFile, printer.FormatSbom), targetPath, targetType)
		convertNotAvailableNumberToZero(summary)

		if !contains(summary.EnginesEnabled, commonParams.ScaType) {
			return fmt.Errorf("unable to generate %s report - SCA engine must be enabled on scan summary", printer.FormatSbom)
		}

		if summary.ScaIssues == notAvailableNumber {
			return fmt.Errorf("unable to generate %s report - SCA engine did not complete successfully", printer.FormatSbom)
		}

		return services.ExportSbomResults(exportWrapper, summaryRpt, summary, formatSbomOptions)
	}
	return fmt.Errorf("bad report format %s", format)
}

func createTargetName(targetFile, targetPath, targetType string) string {
	return filepath.Join(targetPath, targetFile+"."+targetType)
}

func createDirectory(targetPath string) error {
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		log.Printf("\nOutput path not found: %s\n", targetPath)
		log.Printf("Creating directory: %s\n", targetPath)
		err = os.Mkdir(targetPath, directoryPermission)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadResults(
	resultsWrapper wrappers.ResultsWrapper,
	exportWrapper wrappers.ExportWrapper,
	scan *wrappers.ScanResponseModel,
	resultsParams map[string]string,
	agent string, featureflagsWrappers wrappers.FeatureFlagsWrapper) (results *wrappers.ScanResultsCollection, err error) {
	var resultsModel *wrappers.ScanResultsCollection
	var errorModel *wrappers.WebError

	resultsParams[commonParams.ScanIDQueryParam] = scan.ID
	_, sastRedundancy := resultsParams[commonParams.SastRedundancyFlag]

	scaHideDevAndTestDep := resultsParams[ScaExcludeResultTypesParam] == ScaDevAndTestExclusionParam

	resultsModel, errorModel, err = resultsWrapper.GetAllResultsByScanID(resultsParams)

	if err != nil {
		return nil, errors.Wrapf(err, "%s", failedListingResults)
	}
	if errorModel != nil {
		return nil, errors.Errorf("%s: CODE: %d, %s", failedListingResults, errorModel.Code, errorModel.Message)
	}

	if resultsModel != nil {
		if slices.Contains(scan.Engines, commonParams.SastType) && sastRedundancy {
			// Compute SAST results redundancy
			resultsModel = ComputeRedundantSastResults(resultsModel)
		}
		resultsModel, err = enrichScaResults(exportWrapper, scan, resultsModel, scaHideDevAndTestDep, featureflagsWrappers)
		if err != nil {
			return nil, err
		}

		if slices.Contains(scan.Engines, commonParams.ScsType) {
			resultsModel = filterScsResultsByAgent(resultsModel, agent)
		}

		resultsModel.ScanID = scan.ID
		return resultsModel, nil
	}
	return nil, nil
}

func enrichScaResults(
	exportWrapper wrappers.ExportWrapper,
	scan *wrappers.ScanResponseModel,
	resultsModel *wrappers.ScanResultsCollection,
	scaHideDevAndTestDep bool, featureflagWrapper wrappers.FeatureFlagsWrapper) (*wrappers.ScanResultsCollection, error) {
	if slices.Contains(scan.Engines, commonParams.ScaType) {
		scaExportDetails, err := services.GetExportPackage(exportWrapper, scan.ID, scaHideDevAndTestDep, featureflagWrapper)
		if err != nil {
			return nil, errors.Wrapf(err, "%s", failedListingResults)
		}
		scaPackageModel := parseScaExportPackage(scaExportDetails.Packages)
		scaTypeModel := parseExportScaVulnerability(scaExportDetails.ScaTypes)
		if scaPackageModel != nil {
			resultsModel = addPackageInformation(resultsModel, scaPackageModel, scaTypeModel)
		}
	}
	if slices.Contains(scan.Engines, commonParams.ContainersType) && !wrappers.IsContainersEnabled {
		resultsModel = removeResultsByType(resultsModel, commonParams.ContainersType)
	}
	return resultsModel, nil
}

func parseExportScaVulnerability(types []wrappers.ScaType) *[]wrappers.ScaTypeCollection {
	var scaTypes []wrappers.ScaTypeCollection
	for _, t := range types {
		scaTypes = append(scaTypes, wrappers.ScaTypeCollection(t))
	}
	return &scaTypes
}

func parseScaExportPackage(packages []wrappers.ScaPackage) *[]wrappers.ScaPackageCollection {
	var scaPackages []wrappers.ScaPackageCollection
	for _, pkg := range packages {
		pkg := pkg
		scaPackages = append(scaPackages, wrappers.ScaPackageCollection{
			ID:                      pkg.ID,
			Locations:               pkg.Locations,
			DependencyPathArray:     parsePackagePathToDependencyPath(&pkg),
			Outdated:                pkg.Outdated,
			IsDirectDependency:      pkg.IsDirectDependency,
			IsDevelopmentDependency: pkg.IsDevelopmentDependency,
			IsTestDependency:        pkg.IsTestDependency,
		})
	}
	return &scaPackages
}

func parsePackagePathToDependencyPath(pkg *wrappers.ScaPackage) [][]wrappers.DependencyPath {
	var dependencyPathArray [][]wrappers.DependencyPath
	for _, path := range pkg.PackagePathArray {
		var dependencyPath []wrappers.DependencyPath
		for _, dep := range path {
			dependencyPath = append(dependencyPath, wrappers.DependencyPath{
				ID:      dep.ID,
				Name:    dep.Name,
				Version: dep.Version,
			})
		}
		dependencyPathArray = append(dependencyPathArray, dependencyPath)
	}

	// We are doing this to maintain the same structure that was in risk-management api response
	// in risk-management, if the length of the dependency path array is 1, it will be the main package
	// in export service, if there are no dependencies, the package path array will be empty
	if len(dependencyPathArray) == 0 {
		appendMainPackageToDependencyPath(&dependencyPathArray, pkg)
	}
	return dependencyPathArray
}

func appendMainPackageToDependencyPath(dependencyPathArray *[][]wrappers.DependencyPath, pkg *wrappers.ScaPackage) {
	*dependencyPathArray = append(*dependencyPathArray, []wrappers.DependencyPath{{
		ID:            pkg.ID,
		Locations:     pkg.Locations,
		Name:          pkg.Name,
		IsDevelopment: pkg.IsDevelopmentDependency,
	}})
}

func removeResultsByType(model *wrappers.ScanResultsCollection, resultType string) *wrappers.ScanResultsCollection {
	var newResults []*wrappers.ScanResult
	for _, result := range model.Results {
		isResultType := result.Type == resultType
		if resultType == commonParams.SscsType {
			isResultType = strings.HasPrefix(result.Type, resultType)
		}
		if !isResultType {
			newResults = append(newResults, result)
		}
	}
	model.Results = newResults
	model.TotalCount = uint(len(newResults))
	return model
}

func exportSarifResults(targetFile string, results *wrappers.ScanResultsCollection) error {
	var err error
	var resultsJSON []byte
	log.Println("Creating SARIF Report: ", targetFile)
	var sarifResults = convertCxResultsToSarif(results)
	resultsJSON, err = json.Marshal(sarifResults)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingAll)
	}
	f, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to create target file  ", failedGettingAll)
	}
	_, _ = fmt.Fprintln(f, string(resultsJSON))
	_ = f.Close()
	return nil
}
func exportGlSastResults(targetFile string, results *wrappers.ScanResultsCollection, summary *wrappers.ResultSummary) error {
	log.Println("Creating gl-sast Report: ", targetFile)
	var glSast = new(wrappers.GlSastResultsCollection)
	glSast.Vulnerabilities = []wrappers.GlVulnerabilities{}
	err := addScanToGlSastReport(summary, glSast)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to add scan to gl-sast report", failedListingResults)
	}
	convertCxResultToGlSastVulnerability(results, glSast, summary)
	resultsJSON, err := json.Marshal(glSast)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to serialize gl-sast report ", failedListingResults)
	}
	f, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to create target file  ", failedListingResults)
	}
	defer f.Close()
	_, _ = fmt.Fprintln(f, string(resultsJSON))
	return nil
}

func exportGlScaResults(targetFile string, results *wrappers.ScanResultsCollection, summary *wrappers.ResultSummary) error {
	log.Println("Creating Gl-sca Report: ", targetFile)
	glScaResult := &wrappers.GlScaResultsCollection{
		Vulnerabilities:    []wrappers.GlScaDepVulnerabilities{}, // Initialize arrays to prevent GitLab schema validation errors.
		ScaDependencyFiles: []wrappers.ScaDependencyFile{},
	}
	err := addScanToGlScaReport(summary, glScaResult)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to denerate GL-Sca report ", failedListingResults)
	}
	convertCxResultToGlScaVulnerability(results, glScaResult)
	convertCxResultToGlScaFiles(results, glScaResult)
	resultsJSON, err := json.Marshal(glScaResult)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to serialize GL-Sca report ", failedListingResults)
	}
	f, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to create target file  ", failedListingResults)
	}
	_, _ = fmt.Fprintln(f, string(resultsJSON))
	defer f.Close()

	return nil
}

func addScanToGlScaReport(summary *wrappers.ResultSummary, glScaResult *wrappers.GlScaResultsCollection) error {
	createdAt, err := time.Parse(summaryCreatedAtLayout, summary.CreatedAt)
	if err != nil {
		return err
	}

	glScaResult.Schema = wrappers.ScaSchema
	glScaResult.Version = wrappers.SchemaVersion
	glScaResult.Scan.Analyzer.VendorGlSCA.VendorGlname = wrappers.AnalyzerScaID
	glScaResult.Scan.Analyzer.Name = wrappers.AnalyzerScaID
	glScaResult.Scan.Analyzer.Name = wrappers.AnalyzerScaID
	glScaResult.Scan.Analyzer.ID = wrappers.ScannerID
	glScaResult.Scan.Scanner.ID = wrappers.ScannerID
	glScaResult.Scan.Scanner.Name = wrappers.AnalyzerScaID
	glScaResult.Scan.Scanner.VendorGlSCA.VendorGlname = wrappers.AnalyzerScaID
	glScaResult.Scan.Status = commonParams.Success
	glScaResult.Scan.Type = wrappers.ScannerType
	glScaResult.Scan.StartTime = createdAt.Format(glTimeFormat)
	glScaResult.Scan.EndTime = createdAt.Format(glTimeFormat)
	glScaResult.Scan.Scanner.Name = wrappers.AnalyzerScaID
	glScaResult.Scan.Scanner.VersionGlSca = commonParams.Version
	glScaResult.Scan.Analyzer.VersionGlSca = commonParams.Version

	return nil
}

func addScanToGlSastReport(summary *wrappers.ResultSummary, glSast *wrappers.GlSastResultsCollection) error {
	createdAt, err := time.Parse(summaryCreatedAtLayout, summary.CreatedAt)
	if err != nil {
		return err
	}

	glSast.Scan = wrappers.ScanGlReport{}
	glSast.Schema = wrappers.SastSchema
	glSast.Version = wrappers.SastSchemaVersion
	glSast.Scan.Analyzer.URL = wrappers.AnalyzerURL
	glSast.Scan.Analyzer.Name = wrappers.VendorName
	glSast.Scan.Analyzer.Vendor.Name = wrappers.VendorName
	glSast.Scan.Analyzer.ID = wrappers.AnalyzerID
	glSast.Scan.Scanner.ID = wrappers.AnalyzerID
	glSast.Scan.Scanner.Name = wrappers.VendorName
	glSast.Scan.Status = commonParams.Success
	glSast.Scan.Type = commonParams.SastType
	glSast.Scan.StartTime = createdAt.Format(glTimeFormat)
	glSast.Scan.EndTime = createdAt.Format(glTimeFormat)
	glSast.Scan.Scanner.Vendor.Name = wrappers.VendorName
	glSast.Scan.Scanner.Version = commonParams.Version
	glSast.Scan.Analyzer.Version = commonParams.Version

	return nil
}
func exportSonarResults(targetFile string, results *wrappers.ScanResultsCollection) error {
	var err error
	var resultsJSON []byte
	log.Println("Creating SONAR Report: ", targetFile)
	var sonarResults = convertCxResultsToSonar(results)
	resultsJSON, err = json.Marshal(sonarResults)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingAll)
	}
	f, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to create target file  ", failedGettingAll)
	}
	_, _ = fmt.Fprintln(f, string(resultsJSON))
	_ = f.Close()
	return nil
}

// Function to decode HTML entities in the ScanResultsCollection
func decodeHTMLEntitiesInResults(results *wrappers.ScanResultsCollection) {
	for _, result := range results.Results {
		result.Description = html.UnescapeString(result.Description)
		result.DescriptionHTML = html.UnescapeString(result.DescriptionHTML)
		for _, node := range result.ScanResultData.Nodes {
			node.FullName = html.UnescapeString(node.FullName)
			node.Name = html.UnescapeString(node.Name)
		}
	}
}

func exportJSONResults(targetFile string, results *wrappers.ScanResultsCollection) error {
	decodeHTMLEntitiesInResults(results)
	var err error
	var resultsJSON []byte
	log.Println("Creating JSON Report: ", targetFile)
	resultsJSON, err = json.Marshal(results)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingAll)
	}
	f, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to create target file  ", failedGettingAll)
	}
	_, _ = fmt.Fprintln(f, string(resultsJSON))
	_ = f.Close()
	return nil
}

func exportJSONReportResults(jsonWrapper wrappers.ResultsJSONWrapper, summary *wrappers.ResultSummary, summaryRpt string, featureFlagsWrapper wrappers.FeatureFlagsWrapper) error {
	jsonReportsPayload := &wrappers.JSONReportsPayload{}
	pollingResp := &wrappers.JSONPollingResponse{}
	jsonReportsPayload.ReportName = reportNameImprovedScanReport

	jsonOptionsSections, jsonOptionsEngines := parseJSONOptions(summary.EnginesEnabled, jsonReportsPayload.ReportName)

	jsonReportsPayload.ReportType = CliType
	jsonReportsPayload.FileFormat = printer.FormatJSON
	jsonReportsPayload.Data.ScanID = summary.ScanID
	jsonReportsPayload.Data.ProjectID = summary.ProjectID
	jsonReportsPayload.Data.BranchName = summary.BranchName
	jsonReportsPayload.Data.Scanners = jsonOptionsEngines
	jsonReportsPayload.Data.Sections = jsonOptionsSections

	jsonReportID, webErr, err := jsonWrapper.GenerateJSONReport(jsonReportsPayload)
	if webErr != nil {
		return errors.Errorf("Error generating JSON report - %s", webErr.Message)
	}
	if err != nil {
		return errors.Errorf("Error generating JSON report - %s", err.Error())
	}
	log.Println("Generating JSON report")
	pollingResp.Status = startedStatus
	for pollingResp.Status == startedStatus || pollingResp.Status == requestedStatus {
		pollingResp, webErr, err = jsonWrapper.CheckJSONReportStatus(jsonReportID.ReportID)
		if err != nil || webErr != nil {
			return errors.Wrapf(err, "%v", webErr)
		}
		logger.PrintfIfVerbose("JSON report status: %s", pollingResp.Status)
		time.Sleep(delayValueForReport * time.Millisecond)
	}
	if pollingResp.Status != completedStatus {
		return errors.Errorf("JSON generating failed - Current status: %s", pollingResp.Status)
	}

	minioEnabled, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, wrappers.MinioEnabled)
	infoPathType := ""
	if minioEnabled.Status {
		infoPathType = jsonReportID.ReportID
	} else {
		infoPathType = pollingResp.URL
	}
	err = jsonWrapper.DownloadJSONReport(infoPathType, summaryRpt, minioEnabled.Status)
	if err != nil {
		return errors.Wrapf(err, "%s", "Failed downloading JSON report")
	}
	return nil
}

func parseJSONOptions(enabledEngines []string, reportName string) (jsonOptionsSections, jsonOptionsEngines []string) {
	jsonOptionsSections = []string{
		"ScanSummary",
		"ExecutiveSummary",
		"ScanResults",
	}

	var jsonOptionsEnginesMap = map[string]string{
		commonParams.ScaType:        "SCA",
		commonParams.SastType:       "SAST",
		commonParams.KicsType:       "KICS",
		commonParams.IacType:        "KICS",
		commonParams.ContainersType: "Containers",
		commonParams.ScsType:        "Microengines",
	}
	if jsonOptionsEngines == nil {
		for _, engine := range enabledEngines {
			if jsonOptionsEnginesMap[engine] != "" {
				jsonOptionsEngines = append(jsonOptionsEngines, jsonOptionsEnginesMap[engine])
			}
		}
	}

	if reportName == reportNameImprovedScanReport {
		jsonOptionsSections = translateReportSectionsForImproved(jsonOptionsSections)
	}

	return jsonOptionsSections, jsonOptionsEngines
}

func exportJSONSummaryResults(targetFile string, results *wrappers.ResultSummary) error {
	var err error
	var resultsJSON []byte
	log.Println("Creating summary JSON Report: ", targetFile)
	resultsJSON, err = json.Marshal(results)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to serialize results response ", failedGettingAll)
	}
	f, err := os.Create(targetFile)
	if err != nil {
		return errors.Wrapf(err, "%s: failed to create target file  ", failedGettingAll)
	}
	_, _ = fmt.Fprintln(f, string(resultsJSON))
	_ = f.Close()
	return nil
}

func exportPdfResults(pdfWrapper wrappers.ResultsPdfWrapper, summary *wrappers.ResultSummary, summaryRpt, formatPdfToEmail,
	pdfOptions string, featureFlagsWrapper wrappers.FeatureFlagsWrapper) error {
	pdfReportsPayload := &wrappers.PdfReportsPayload{}
	pollingResp := &wrappers.PdfPollingResponse{}
	pdfReportsPayload.ReportName = reportNameImprovedScanReport
	pdfOptionsSections, pdfOptionsEngines, err := parsePDFOptions(pdfOptions, summary.EnginesEnabled, pdfReportsPayload.ReportName)
	if err != nil {
		return err
	}
	pdfReportsPayload.ReportType = CliType
	pdfReportsPayload.FileFormat = printer.FormatPDF
	pdfReportsPayload.Data.ScanID = summary.ScanID
	pdfReportsPayload.Data.ProjectID = summary.ProjectID
	pdfReportsPayload.Data.BranchName = summary.BranchName
	pdfReportsPayload.Data.Scanners = pdfOptionsEngines
	pdfReportsPayload.Data.Sections = pdfOptionsSections

	// will generate pdf report and send it to the email list
	// instead of saving it to the file system
	if len(formatPdfToEmail) > 0 {
		emailList, validateErr := validateEmails(formatPdfToEmail)
		if validateErr != nil {
			return validateErr
		}
		pdfReportsPayload.ReportType = reportTypeEmail
		pdfReportsPayload.Data.Email = emailList
	}
	pdfReportID, webErr, err := pdfWrapper.GeneratePdfReport(pdfReportsPayload)
	if webErr != nil {
		return errors.Errorf("Error generating PDF report - %s", webErr.Message)
	}
	if err != nil {
		return errors.Errorf("Error generating PDF report - %s", err.Error())
	}
	if pdfReportsPayload.ReportType == reportTypeEmail {
		log.Println("Sending PDF report to: ", pdfReportsPayload.Data.Email)
		return nil
	}
	log.Println("Generating PDF report")
	pollingResp.Status = startedStatus
	for pollingResp.Status == startedStatus || pollingResp.Status == requestedStatus {
		pollingResp, webErr, err = pdfWrapper.CheckPdfReportStatus(pdfReportID.ReportID)
		if err != nil || webErr != nil {
			return errors.Wrapf(err, "%v", webErr)
		}
		logger.PrintfIfVerbose("PDF report status: %s", pollingResp.Status)
		time.Sleep(delayValueForReport * time.Millisecond)
	}
	if pollingResp.Status != completedStatus {
		return errors.Errorf("PDF generating failed - Current status: %s", pollingResp.Status)
	}

	minioEnabled, _ := wrappers.GetSpecificFeatureFlag(featureFlagsWrapper, wrappers.MinioEnabled)
	infoPathType := ""
	if minioEnabled.Status {
		infoPathType = pdfReportID.ReportID
	} else {
		infoPathType = pollingResp.URL
	}
	err = pdfWrapper.DownloadPdfReport(infoPathType, summaryRpt, minioEnabled.Status)
	if err != nil {
		return errors.Wrapf(err, "%s", "Failed downloading PDF report")
	}
	return nil
}

func parsePDFOptions(pdfOptions string, enabledEngines []string, reportName string) (pdfOptionsSections, pdfOptionsEngines []string, err error) {
	var pdfOptionsSectionsMap = map[string]string{
		"scansummary":      "ScanSummary",
		"executivesummary": "ExecutiveSummary",
		"scanresults":      "ScanResults",
	}

	var pdfOptionsEnginesMap = map[string]string{
		commonParams.ScaType:  "SCA",
		commonParams.SastType: "SAST",
		commonParams.KicsType: "KICS",
		commonParams.IacType:  "KICS",
	}

	pdfOptions = strings.ToLower(strings.ReplaceAll(pdfOptions, " ", ""))
	options := strings.Split(strings.ReplaceAll(pdfOptions, "\n", ""), ",")
	for _, s := range options {
		if pdfOptionsEnginesMap[s] != "" {
			pdfOptionsEngines = append(pdfOptionsEngines, pdfOptionsEnginesMap[s])
		} else if pdfOptionsSectionsMap[s] != "" {
			pdfOptionsSections = append(pdfOptionsSections, pdfOptionsSectionsMap[s])
		} else {
			return nil, nil, errors.Errorf("report option \"%s\" unavailable", s)
		}
	}
	if pdfOptionsEngines == nil {
		for _, engine := range enabledEngines {
			if pdfOptionsEnginesMap[engine] != "" {
				pdfOptionsEngines = append(pdfOptionsEngines, pdfOptionsEnginesMap[engine])
			}
		}
	}

	if reportName == reportNameImprovedScanReport {
		pdfOptionsSections = translateReportSectionsForImproved(pdfOptionsSections)
	}

	return pdfOptionsSections, pdfOptionsEngines, nil
}

func translateReportSectionsForImproved(sections []string) []string {
	var resultSections = make([]string, 0)

	var pdfOptionsSectionsImprovedTranslation = map[string][]string{
		"ScanSummary":      {"scan-information"},
		"ExecutiveSummary": {"results-overview"},
		"ScanResults":      {"scan-results", "categories", "resolved-results", "vulnerability-details"},
	}

	for _, section := range sections {
		if translatedSections := pdfOptionsSectionsImprovedTranslation[section]; translatedSections != nil {
			resultSections = append(resultSections, translatedSections...)
		}
	}

	return resultSections
}

func convertCxResultsToSarif(results *wrappers.ScanResultsCollection) *wrappers.SarifResultsCollection {
	var sarif = new(wrappers.SarifResultsCollection)
	sarif.Schema = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json"
	sarif.Version = "2.1.0"
	sarif.Runs = []wrappers.SarifRun{}
	sarif.Runs = append(sarif.Runs, createSarifRun(results))
	return sarif
}

func convertCxResultToGlSastVulnerability(results *wrappers.ScanResultsCollection, glSast *wrappers.GlSastResultsCollection, summary *wrappers.ResultSummary) {
	for _, result := range results.Results {
		if strings.TrimSpace(result.Type) == commonParams.SastType {
			glSast = parseGlSastVulnerability(result, glSast, summary)
		}
	}
}

func convertCxResultToGlScaVulnerability(results *wrappers.ScanResultsCollection, glScaResult *wrappers.GlScaResultsCollection) {
	for _, result := range results.Results {
		if strings.TrimSpace(result.Type) == commonParams.ScaType {
			glScaResult = parseGlscaVulnerability(result, glScaResult)
		}
	}
}

func convertCxResultToGlScaFiles(results *wrappers.ScanResultsCollection, glScaResult *wrappers.GlScaResultsCollection) {
	for _, result := range results.Results {
		if strings.TrimSpace(result.Type) == commonParams.ScaType {
			glScaResult = parseGlScaFiles(result, glScaResult)
		}
	}
}
func parseGlSastVulnerability(result *wrappers.ScanResult, glSast *wrappers.GlSastResultsCollection, summary *wrappers.ResultSummary) *wrappers.GlSastResultsCollection {
	hostName := parseURI(summary.BaseURI)

	queryName := result.ScanResultData.QueryName
	fileName := result.ScanResultData.Nodes[0].FileName
	lineNumber := strconv.FormatUint(uint64(result.ScanResultData.Nodes[0].Line), 10)
	startLine := result.ScanResultData.Nodes[0].Line
	endLine := result.ScanResultData.Nodes[0].Line + result.ScanResultData.Nodes[0].Length
	ID := fmt.Sprintf("%s:%s:%s", queryName, fileName, lineNumber)
	category := fmt.Sprintf("%s-%s", wrappers.VendorName, result.Type)
	message := fmt.Sprintf("%s@%s:%s", queryName, fileName, lineNumber)
	QueryDescriptionLink := fmt.Sprintf("%s/results/%s/%s/sast/description/%s/%s", hostName, summary.ScanID, summary.ProjectID, result.VulnerabilityDetails.CweID, result.ScanResultData.QueryID)

	glSast.Vulnerabilities = append(glSast.Vulnerabilities, wrappers.GlVulnerabilities{
		ID:          ID,
		Category:    category,
		Name:        queryName,
		Message:     message,
		Description: result.Description + "   \n" + QueryDescriptionLink,
		CVE:         ID,
		Severity:    cases.Title(language.English).String(result.Severity),
		Confidence:  cases.Title(language.English).String(result.Severity),
		Solution:    "",

		Scanner: wrappers.GlScanner{
			ID:   category,
			Name: category,
		},
		Identifiers: []wrappers.Identifier{
			{
				Type:  "cxOneScan",
				Name:  "CxOne Scan",
				URL:   summary.BaseURI,
				Value: result.ID,
			},
		},
		Links: make([]string, 0),
		Tracking: wrappers.Tracking{
			Type: "source",
			Items: []wrappers.Item{
				{
					Signatures: []wrappers.Signature{{Algorithm: result.Type + "-Algorithm ", Value: "NA"}},
					File:       fileName,
					EndLine:    endLine,
					StartLine:  startLine,
				},
			},
		},
		Flags: make([]wrappers.Flag, 0),
		Location: wrappers.Location{
			File:      fileName,
			StartLine: startLine,
			EndLine:   endLine,
		},
	})
	return glSast
}
func parseGlscaVulnerability(result *wrappers.ScanResult, glDependencyResult *wrappers.GlScaResultsCollection) *wrappers.GlScaResultsCollection {
	if result.ScanResultData.ScaPackageCollection != nil {
		glDependencyResult.Vulnerabilities = append(glDependencyResult.Vulnerabilities, wrappers.GlScaDepVulnerabilities{
			ID:          result.ID,
			Name:        result.VulnerabilityDetails.CveName,
			Description: result.Description,
			Severity:    cases.Title(language.English).String(result.Severity),
			Solution:    result.ScanResultData.RecommendedVersion,
			Identifiers: collectScaPackageData(result),
			Links:       collectScaPackageLinks(result),
			TrackingDep: wrappers.TrackingDep{
				Items: collectScaPackageItemsDep(result),
			},
			Flags: make([]string, 0),
			LocationDep: wrappers.GlScaDepVulnerabilityLocation{
				File: parseGlDependencyLocation(result),
				Dependency: wrappers.ScaDependencyLocation{
					Package:                      wrappers.PackageName{Name: result.ScanResultData.PackageIdentifier},
					ScaDependencyLocationVersion: "",
					Direct:                       result.ScanResultData.ScaPackageCollection.IsDirectDependency,
					ScaDependencyPath:            result.ScanResultData.Line,
				},
			},
		})
	}
	return glDependencyResult
}
func parseGlDependencyLocation(result *wrappers.ScanResult) string {
	var location string
	if result != nil && result.ScanResultData.ScaPackageCollection != nil && result.ScanResultData.ScaPackageCollection.Locations != nil {
		location = *result.ScanResultData.ScaPackageCollection.Locations[0]
	} else {
		location = ""
	}
	return location
}
func parseGlScaFiles(result *wrappers.ScanResult, glScaResult *wrappers.GlScaResultsCollection) *wrappers.GlScaResultsCollection {
	if result.ScanResultData.ScaPackageCollection != nil && result.ScanResultData.ScaPackageCollection.Locations != nil {
		glScaResult.ScaDependencyFiles = append(glScaResult.ScaDependencyFiles, wrappers.ScaDependencyFile{
			Path:           *result.ScanResultData.ScaPackageCollection.Locations[0],
			PackageManager: result.ScanResultData.ScaPackageCollection.ID,
			Dependencies:   collectScaFileLocations(result),
		})
	}
	return glScaResult
}
func collectScaFileLocations(result *wrappers.ScanResult) []wrappers.ScaDependencyLocation {
	allScaIdentifierLocations := []wrappers.ScaDependencyLocation{}
	for _, packageInfo := range result.ScanResultData.PackageData {
		allScaIdentifierLocations = append(allScaIdentifierLocations, wrappers.ScaDependencyLocation{
			Package: wrappers.PackageName{
				Name: packageInfo.Type,
			},
			ScaDependencyLocationVersion: packageInfo.URL,
			Direct:                       true,
			ScaDependencyPath:            result.ScanResultData.Line,
		})
	}
	return allScaIdentifierLocations
}
func collectScaPackageItemsDep(result *wrappers.ScanResult) []wrappers.ItemDep {
	allScaPackageItemDep := []wrappers.ItemDep{}
	allScaPackageItemDep = append(allScaPackageItemDep, wrappers.ItemDep{
		Signature: []wrappers.SignatureDep{{Algorithm: "SCA-Algorithm ", Value: "NA"}},
		File:      result.VulnerabilityDetails.CveName,
		EndLine:   0,
		StartLine: 0,
	})
	return allScaPackageItemDep
}
func collectScaPackageLinks(result *wrappers.ScanResult) []wrappers.LinkDep {
	allScaPackageLinks := []wrappers.LinkDep{}
	for _, packageInfo := range result.ScanResultData.PackageData {
		allScaPackageLinks = append(allScaPackageLinks, wrappers.LinkDep{
			Name: packageInfo.Type,
			URL:  packageInfo.URL,
		})
	}
	return allScaPackageLinks
}
func collectScaPackageData(result *wrappers.ScanResult) []wrappers.IdentifierDep {
	allIdentifierDep := []wrappers.IdentifierDep{}
	for _, packageInfo := range result.ScanResultData.PackageData {
		allIdentifierDep = append(allIdentifierDep, wrappers.IdentifierDep{
			Type:  packageInfo.Type,
			Value: packageInfo.URL,
			Name:  packageInfo.URL,
		})
	}
	return allIdentifierDep
}

func convertCxResultsToSonar(results *wrappers.ScanResultsCollection) *wrappers.ScanResultsSonar {
	var sonar = new(wrappers.ScanResultsSonar)
	sonar.Issues, sonar.Rules = parseSonar(results)
	return sonar
}

func createSarifRun(results *wrappers.ScanResultsCollection) wrappers.SarifRun {
	var sarifRun wrappers.SarifRun
	sarifRun.Tool.Driver.Name = wrappers.SarifName
	sarifRun.Tool.Driver.Version = wrappers.SarifVersion
	sarifRun.Tool.Driver.InformationURI = wrappers.SarifInformationURI
	sarifRun.Tool.Driver.Rules, sarifRun.Results = parseResults(results)
	return sarifRun
}

func parseResults(results *wrappers.ScanResultsCollection) ([]wrappers.SarifDriverRule, []wrappers.SarifScanResult) {
	var sarifRules = make([]wrappers.SarifDriverRule, 0)
	var sarifResults = make([]wrappers.SarifScanResult, 0)
	if results != nil {
		ruleIds := map[interface{}]bool{}
		for _, result := range results.Results {
			if rule := findRule(ruleIds, result); rule != nil {
				sarifRules = append(sarifRules, *rule)
			}
			if sarifResult := findResult(result); sarifResult != nil {
				sarifResults = append(sarifResults, sarifResult...)
			}
		}
	}
	return sarifRules, sarifResults
}

func parseSonar(results *wrappers.ScanResultsCollection) ([]wrappers.SonarIssues, []wrappers.SonarRules) {
	var sonarIssues []wrappers.SonarIssues
	var sonarRules []wrappers.SonarRules
	seenRuleIDs := make(map[string]bool) // Track already added rule IDs

	if results != nil {
		for _, result := range results.Results {
			var auxRules = initSonarRules(result)
			var auxIssue = initSonarIssue(result)

			if !seenRuleIDs[auxRules.ID] {
				sonarRules = append(sonarRules, auxRules)
				seenRuleIDs[auxRules.ID] = true
			}

			engineType := strings.TrimSpace(result.Type)

			if engineType == commonParams.SastType {
				auxIssue.PrimaryLocation = parseSonarPrimaryLocation(result)
				auxIssue.SecondaryLocations = parseSonarSecondaryLocations(result)
				sonarIssues = append(sonarIssues, auxIssue)
			} else if engineType == commonParams.KicsType {
				auxIssue.PrimaryLocation = parseLocationKics(result)
				sonarIssues = append(sonarIssues, auxIssue)
			} else if engineType == commonParams.ScaType {
				sonarIssuesByLocation := parseScaSonarLocations(result)
				sonarIssues = append(sonarIssues, sonarIssuesByLocation...)
			} else if wrappers.IsContainersEnabled && engineType == commonParams.ContainersType {
				auxIssue.PrimaryLocation = parseContainersSonar(result)
				sonarIssues = append(sonarIssues, auxIssue)
			} else if strings.HasPrefix(engineType, commonParams.SscsType) {
				sscsSonarIssue := parseSscsSonar(result, &auxIssue)
				sonarIssues = append(sonarIssues, sscsSonarIssue)
			}
		}
	}
	return sonarIssues, sonarRules
}

func parseContainersSonar(result *wrappers.ScanResult) wrappers.SonarLocation {
	var auxLocation wrappers.SonarLocation
	auxLocation.FilePath = result.ScanResultData.ImageFilePath
	auxLocation.Message = html.UnescapeString(result.Description)
	var textRange wrappers.SonarTextRange
	textRange.StartColumn = 1
	textRange.EndColumn = 2
	textRange.StartLine = 1
	textRange.EndLine = 2
	auxLocation.TextRange = textRange
	return auxLocation
}

func parseSscsSonar(result *wrappers.ScanResult, sonarIssue *wrappers.SonarIssues) wrappers.SonarIssues {
	sonarIssue.PrimaryLocation.FilePath = result.ScanResultData.Filename

	sonarIssue.PrimaryLocation.Message = html.UnescapeString(result.ScanResultData.Remediation)
	var textRange wrappers.SonarTextRange
	textRange.StartColumn = 1
	textRange.EndColumn = 2
	textRange.StartLine = result.ScanResultData.Line
	sonarIssue.PrimaryLocation.TextRange = textRange
	return *sonarIssue
}

func initSonarIssue(result *wrappers.ScanResult) wrappers.SonarIssues {
	var sonarIssue wrappers.SonarIssues
	engineType := strings.TrimSpace(result.Type)
	if engineType == commonParams.SastType {
		sonarIssue.RuleID = result.ScanResultData.LanguageName + " - " + result.ScanResultData.QueryName
	} else if engineType == commonParams.KicsType {
		sonarIssue.RuleID = result.ScanResultData.QueryName
	} else if engineType == commonParams.ScaType {
		sonarIssue.RuleID = result.ID
	} else if wrappers.IsContainersEnabled && engineType == commonParams.ContainersType {
		sonarIssue.RuleID = result.ID
	} else if strings.HasPrefix(engineType, commonParams.SscsType) {
		sonarIssue.RuleID = result.ID
	}

	sonarIssue.PrimaryLocation.Message = html.UnescapeString(result.Description)
	sonarIssue.EffortMinutes = 0

	return sonarIssue
}

func initSonarRules(result *wrappers.ScanResult) wrappers.SonarRules {
	var sonarRules wrappers.SonarRules
	var sonarImpacts wrappers.SonarImpacts

	sonarImpacts.Severity = sonarSeverities[result.Severity]
	sonarImpacts.SoftwareQuality = vulnerabilitySonar

	sonarRules.EngineID = result.Type
	sonarRules.CleanCodeAttribute = cleanCodeAttribute

	engineType := strings.TrimSpace(result.Type)
	if engineType == commonParams.SastType {
		sonarRules.Name = result.ScanResultData.QueryName
		sonarRules.Description = html.UnescapeString(result.Description)
		sonarRules.ID = result.ScanResultData.LanguageName + " - " + result.ScanResultData.QueryName
	} else if engineType == commonParams.KicsType {
		sonarRules.Name = result.ScanResultData.QueryName
		sonarRules.Description = html.UnescapeString(result.Description)
		sonarRules.ID = result.ScanResultData.QueryName
	} else if engineType == commonParams.ScaType {
		sonarRules.Name = result.ScanResultData.PackageIdentifier
		sonarRules.Description = html.UnescapeString(result.Description)
		sonarRules.ID = result.ID
	} else if wrappers.IsContainersEnabled && engineType == commonParams.ContainersType {
		sonarRules.Name = result.ScanResultData.ImageTag
		sonarRules.Description = html.UnescapeString(result.Description)
		sonarRules.ID = result.ID
	} else if strings.HasPrefix(engineType, commonParams.SscsType) {
		sonarRules.Name = result.ScanResultData.RuleName
		sonarRules.Description = html.UnescapeString(result.ScanResultData.RuleDescription)
		sonarRules.ID = result.ID
	}

	sonarRules.Impacts = []wrappers.SonarImpacts{sonarImpacts}

	return sonarRules
}

func parseScaSonarLocations(result *wrappers.ScanResult) []wrappers.SonarIssues {
	if result == nil || result.ScanResultData.ScaPackageCollection == nil || result.ScanResultData.ScaPackageCollection.Locations == nil {
		return []wrappers.SonarIssues{}
	}

	var issuesByLocation []wrappers.SonarIssues

	for _, location := range result.ScanResultData.ScaPackageCollection.Locations {
		issueByLocation := initSonarIssue(result)

		var primaryLocation wrappers.SonarLocation

		primaryLocation.FilePath = *location
		_, _, primaryLocation.Message = findRuleID(result)

		var textRange wrappers.SonarTextRange
		textRange.StartColumn = 1
		textRange.EndColumn = 2
		textRange.StartLine = 1
		textRange.EndLine = 2

		primaryLocation.TextRange = textRange

		issueByLocation.PrimaryLocation = primaryLocation

		issuesByLocation = append(issuesByLocation, issueByLocation)
	}

	return issuesByLocation
}

func parseLocationKics(results *wrappers.ScanResult) wrappers.SonarLocation {
	var auxLocation wrappers.SonarLocation
	auxLocation.FilePath = strings.TrimLeft(results.ScanResultData.Filename, "/")
	auxLocation.Message = html.UnescapeString(results.ScanResultData.Value)
	var auxTextRange wrappers.SonarTextRange
	auxTextRange.StartLine = results.ScanResultData.Line
	auxTextRange.StartColumn = 0
	auxTextRange.EndColumn = 1
	auxLocation.TextRange = auxTextRange
	return auxLocation
}

func parseSonarPrimaryLocation(results *wrappers.ScanResult) wrappers.SonarLocation {
	var auxLocation wrappers.SonarLocation
	// fill the details in the primary Location
	if len(results.ScanResultData.Nodes) > 0 {
		auxLocation.FilePath = strings.TrimLeft(results.ScanResultData.Nodes[0].FileName, "/")
		auxLocation.Message = html.UnescapeString(strings.ReplaceAll(results.ScanResultData.QueryName, "_", " "))
		auxLocation.TextRange = parseSonarTextRange(results.ScanResultData.Nodes[0])
	}
	return auxLocation
}

func parseSonarSecondaryLocations(results *wrappers.ScanResult) []wrappers.SonarLocation {
	var auxSecondaryLocations []wrappers.SonarLocation
	// Traverse all the rest of the scan result nodes into secondary location of sonar
	if len(results.ScanResultData.Nodes) >= 1 {
		for _, node := range results.ScanResultData.Nodes[1:] {
			var auxSecondaryLocation wrappers.SonarLocation
			auxSecondaryLocation.FilePath = strings.TrimLeft(node.FileName, "/")
			auxSecondaryLocation.Message = html.UnescapeString(strings.ReplaceAll(results.ScanResultData.QueryName, "_", " "))
			auxSecondaryLocation.TextRange = parseSonarTextRange(node)
			auxSecondaryLocations = append(auxSecondaryLocations, auxSecondaryLocation)
		}
	}
	return auxSecondaryLocations
}

func parseSonarTextRange(results *wrappers.ScanResultNode) wrappers.SonarTextRange {
	var auxTextRange wrappers.SonarTextRange
	auxTextRange.StartLine = results.Line
	startColumn := getSastStartColumn(results.Column)

	auxTextRange.StartColumn = startColumn
	auxTextRange.EndColumn = startColumn + results.Length

	if auxTextRange.StartColumn == auxTextRange.EndColumn {
		auxTextRange.EndColumn++
	}

	return auxTextRange
}

func findRule(ruleIds map[interface{}]bool, result *wrappers.ScanResult) *wrappers.SarifDriverRule {
	var sarifRule wrappers.SarifDriverRule
	sarifRule.ID, sarifRule.Name, _ = findRuleID(result)
	sarifRule.FullDescription = findFullDescription(result)
	sarifRule.Help = findHelp(result)
	sarifRule.HelpURI = findHelpURI(result)
	sarifRule.Properties = findProperties(result)

	if !ruleIds[sarifRule.ID] {
		ruleIds[sarifRule.ID] = true
		return &sarifRule
	}

	return nil
}

func getSastStartColumn(column uint) uint {
	if column == 0 {
		return 0
	}
	return column - 1
}

func findRuleID(result *wrappers.ScanResult) (ruleID, ruleName, shortMessage string) {
	caser := cases.Title(language.English)

	if result.ScanResultData.QueryID == nil && result.ScanResultData.RuleID == nil {
		return fmt.Sprintf("%s (%s)", result.ID, result.Type),
			caser.String(strings.ToLower(strings.ReplaceAll(result.ID, "-", ""))),
			html.UnescapeString(fmt.Sprintf("%s (%s)", result.ScanResultData.PackageIdentifier, result.ID))
	}

	if result.ScanResultData.RuleID != nil {
		ruleName = strings.ReplaceAll(result.ScanResultData.RuleName, "_", " ")
		return fmt.Sprintf("%s - %s (%s)", ruleName, *result.ScanResultData.RuleID, result.Type),
			ruleName,
			ruleName
	}

	ruleName = strings.ReplaceAll(result.ScanResultData.QueryName, "_", " ")
	return fmt.Sprintf("%v - %s (%s)", ruleName, result.ScanResultData.QueryID, result.Type),
		ruleName,
		ruleName
}

func findFullDescription(result *wrappers.ScanResult) wrappers.SarifDescription {
	var sarifDescription wrappers.SarifDescription
	sarifDescription.Text = findDescriptionText(result)
	return sarifDescription
}

func findHelp(result *wrappers.ScanResult) wrappers.SarifHelp {
	var sarifHelp wrappers.SarifHelp
	sarifHelp.Text = findHelpText(result)
	sarifHelp.Markdown = findHelpMarkdownText(result)

	return sarifHelp
}

func findHelpURI(result *wrappers.ScanResult) string {
	if strings.HasPrefix(result.Type, commonParams.SscsType) {
		if result.ScanResultData.RemediationLink != "" {
			return result.ScanResultData.RemediationLink
		}
	}

	return wrappers.SarifInformationURI
}

func findDescriptionText(result *wrappers.ScanResult) string {
	if result.Type == commonParams.KicsType {
		return fmt.Sprintf(
			"%s Value: %s Excepted value: %s",
			result.Description, result.ScanResultData.Value, result.ScanResultData.ExpectedValue,
		)
	} else if strings.HasPrefix(result.Type, commonParams.SscsType) {
		return result.ScanResultData.RuleDescription
	}

	return result.Description
}

func findHelpText(result *wrappers.ScanResult) string {
	if strings.HasPrefix(result.Type, commonParams.SscsType) {
		return findHelpMarkdownText(result)
	}

	return findDescriptionText(result)
}

func findHelpMarkdownText(result *wrappers.ScanResult) string {
	if result.Type == commonParams.KicsType {
		return fmt.Sprintf(
			"%s <br><br><strong>Value:</strong> %s <br><strong>Excepted value:</strong> %s",
			result.Description, result.ScanResultData.Value, result.ScanResultData.ExpectedValue,
		)
	} else if strings.HasPrefix(result.Type, commonParams.SscsType) {
		return result.ScanResultData.Remediation
	}

	return result.Description
}

func findProperties(result *wrappers.ScanResult) wrappers.SarifProperties {
	var sarifProperties wrappers.SarifProperties
	sarifProperties.ID, sarifProperties.Name, _ = findRuleID(result)
	sarifProperties.Description = findDescriptionText(result)
	sarifProperties.SecuritySeverity = securities[result.Severity]
	sarifProperties.Tags = []string{"security", "checkmarx", result.Type}
	return sarifProperties
}

func findSarifLevel(result *wrappers.ScanResult) string {
	level := map[string]string{
		infoCx:     infoLowSarif,
		lowCx:      infoLowSarif,
		mediumCx:   mediumSarif,
		highCx:     highSarif,
		criticalCx: highSarif,
	}
	return level[result.Severity]
}

func initSarifResult(result *wrappers.ScanResult) wrappers.SarifScanResult {
	var scanResult wrappers.SarifScanResult
	scanResult.RuleID, _, scanResult.Message.Text = findRuleID(result)
	scanResult.Level = findSarifLevel(result)
	scanResult.Locations = []wrappers.SarifLocation{}

	return scanResult
}

func findResult(result *wrappers.ScanResult) []wrappers.SarifScanResult {
	var scanResults []wrappers.SarifScanResult

	if len(result.ScanResultData.Nodes) > 0 {
		scanResults = parseSarifResultSast(result, scanResults)
	} else if result.Type == commonParams.KicsType {
		scanResults = parseSarifResultKics(result, scanResults)
	} else if result.Type == commonParams.ScaType {
		scanResults = parseSarifResultsSca(result, scanResults)
	} else if result.Type == commonParams.ContainersType && wrappers.IsContainersEnabled {
		scanResults = parseSarifResultsContainers(result, scanResults)
	} else if strings.HasPrefix(result.Type, commonParams.SscsType) {
		scanResults = parseSarifResultsSscs(result, scanResults)
	}

	if len(scanResults) > 0 {
		return scanResults
	}
	return nil
}

func parseSarifResultsContainers(result *wrappers.ScanResult, scanResults []wrappers.SarifScanResult) []wrappers.SarifScanResult {
	var scanResult = initSarifResult(result)
	var scanLocation wrappers.SarifLocation

	scanLocation.PhysicalLocation.ArtifactLocation.URI = result.ScanResultData.ImageFilePath
	scanLocation.PhysicalLocation.Region = &wrappers.SarifRegion{}
	scanLocation.PhysicalLocation.Region.StartLine = 1
	scanLocation.PhysicalLocation.Region.StartColumn = 1
	scanLocation.PhysicalLocation.Region.EndColumn = 2
	scanResult.Locations = append(scanResult.Locations, scanLocation)

	scanResults = append(scanResults, scanResult)
	return scanResults
}

func parseSarifResultsSca(result *wrappers.ScanResult, scanResults []wrappers.SarifScanResult) []wrappers.SarifScanResult {
	if result == nil || result.ScanResultData.ScaPackageCollection == nil || result.ScanResultData.ScaPackageCollection.Locations == nil {
		return scanResults
	}
	for _, location := range result.ScanResultData.ScaPackageCollection.Locations {
		var scanResult = initSarifResult(result)

		var scanLocation wrappers.SarifLocation
		scanLocation.PhysicalLocation.ArtifactLocation.URI = *location
		scanLocation.PhysicalLocation.Region = &wrappers.SarifRegion{}
		scanLocation.PhysicalLocation.Region.StartLine = 1
		scanLocation.PhysicalLocation.Region.StartColumn = 1
		scanLocation.PhysicalLocation.Region.EndColumn = 2
		scanResult.Locations = append(scanResult.Locations, scanLocation)

		scanResults = append(scanResults, scanResult)
	}
	return scanResults
}

func parseSarifResultKics(result *wrappers.ScanResult, scanResults []wrappers.SarifScanResult) []wrappers.SarifScanResult {
	var scanResult = initSarifResult(result)
	var scanLocation wrappers.SarifLocation

	scanLocation.PhysicalLocation.ArtifactLocation.URI = strings.Replace(
		result.ScanResultData.Filename,
		"/",
		"",
		1,
	)
	scanLocation.PhysicalLocation.Region = &wrappers.SarifRegion{}
	scanLocation.PhysicalLocation.Region.StartLine = result.ScanResultData.Line
	scanLocation.PhysicalLocation.Region.StartColumn = 1
	scanLocation.PhysicalLocation.Region.EndColumn = 2
	scanResult.Locations = append(scanResult.Locations, scanLocation)

	scanResults = append(scanResults, scanResult)
	return scanResults
}

func parseSarifResultSast(result *wrappers.ScanResult, scanResults []wrappers.SarifScanResult) []wrappers.SarifScanResult {
	if result == nil || result.ScanResultData.Nodes == nil {
		return scanResults
	}
	var scanResult = initSarifResult(result)

	for _, node := range result.ScanResultData.Nodes {
		var scanLocation wrappers.SarifLocation
		if len(node.FileName) >= sarifNodeFileLength {
			scanLocation.PhysicalLocation.ArtifactLocation.URI = node.FileName[1:]
			if node.Line <= 0 {
				continue
			}
			scanLocation.PhysicalLocation.Region = &wrappers.SarifRegion{}
			scanLocation.PhysicalLocation.Region.StartLine = node.Line
			column := node.Column
			length := node.Length
			scanLocation.PhysicalLocation.Region.StartColumn = column
			scanLocation.PhysicalLocation.Region.EndColumn = column + length

			scanResult.Locations = append(scanResult.Locations, scanLocation)
		}
	}

	scanResults = append(scanResults, scanResult)
	return scanResults
}

func parseSarifResultsSscs(result *wrappers.ScanResult, scanResults []wrappers.SarifScanResult) []wrappers.SarifScanResult {
	var scanResult = initSarifResult(result)
	scanResult.Message.Text = result.Description

	var scanLocation wrappers.SarifLocation

	trimOsSeparatorFromFileName(result)
	if result.Type == commonParams.SCSScorecardType && result.ScanResultData.Filename == noFileForScorecardResultString {
		scanLocation.PhysicalLocation.ArtifactLocation.URI = artifactLocationURIString
		scanLocation.PhysicalLocation.ArtifactLocation.Description = &wrappers.SarifMessage{}
		scanLocation.PhysicalLocation.ArtifactLocation.Description.Text = result.ScanResultData.Filename
	} else {
		scanLocation.PhysicalLocation.ArtifactLocation.URI = result.ScanResultData.Filename
	}

	scanLocation.PhysicalLocation.Region = &wrappers.SarifRegion{}
	scanLocation.PhysicalLocation.Region.StartLine = result.ScanResultData.Line
	scanLocation.PhysicalLocation.Region.StartColumn = 1
	scanLocation.PhysicalLocation.Region.EndColumn = 2
	if result.ScanResultData.Snippet != "" {
		scanLocation.PhysicalLocation.Region.Snippet = &wrappers.SarifSnippet{}
		scanLocation.PhysicalLocation.Region.Snippet.Text = result.ScanResultData.Snippet
	}

	scanResult.Locations = append(scanResult.Locations, scanLocation)

	var properties wrappers.SarifResultProperties
	properties.Severity = result.Severity
	properties.Validity = result.ScanResultData.Validity
	properties.IsInSource = result.ScanResultData.IsInSource
	properties.CommitURL = result.ScanResultData.CommitURL
	scanResult.Properties = &properties

	scanResults = append(scanResults, scanResult)
	return scanResults
}

func convertNotAvailableNumberToZero(summary *wrappers.ResultSummary) {
	if summary.KicsIssues == notAvailableNumber {
		summary.KicsIssues = 0
	} else if summary.SastIssues == notAvailableNumber {
		summary.SastIssues = 0
	} else if summary.ScaIssues == notAvailableNumber {
		summary.ScaIssues = 0
	} else if wrappers.IsContainersEnabled && *summary.ContainersIssues == notAvailableNumber {
		*summary.ContainersIssues = 0
	}
}

func buildAuxiliaryScaMaps(resultsModel *wrappers.ScanResultsCollection, scaPackageModel *[]wrappers.ScaPackageCollection,
	scaTypeModel *[]wrappers.ScaTypeCollection) (locationsByID map[string][]*string, typesByCVE map[string]wrappers.ScaTypeCollection) {
	locationsByID = make(map[string][]*string)
	typesByCVE = make(map[string]wrappers.ScaTypeCollection)
	// Create map to be used to populate locations for each package path
	for _, result := range resultsModel.Results {
		if result.Type == commonParams.ScaType {
			for _, packages := range *scaPackageModel {
				currentPackage := packages
				locationsByID[packages.ID] = currentPackage.Locations
			}
			for _, types := range *scaTypeModel {
				identifier := fmt.Sprintf("%s:%s", types.ID, types.PackageID)
				typesByCVE[identifier] = types
			}
		}
	}
	return locationsByID, typesByCVE
}

func buildScaType(typesByCVE map[string]wrappers.ScaTypeCollection, result *wrappers.ScanResult) string {
	identifier := buildVulnerabilityIdentifier(result)
	types, ok := typesByCVE[identifier]
	if ok && types.Type == "SupplyChain" {
		return "Supply Chain"
	}
	return "Vulnerability"
}

func buildScaState(typesByCVE map[string]wrappers.ScaTypeCollection, result *wrappers.ScanResult) string {
	identifier := buildVulnerabilityIdentifier(result)
	types, ok := typesByCVE[identifier]
	if ok && types.IsIgnored {
		return notExploitable
	}
	return result.State
}

func buildVulnerabilityIdentifier(result *wrappers.ScanResult) string {
	return fmt.Sprintf("%s:%s", result.ID, result.ScanResultData.PackageIdentifier)
}

func addPackageInformation(
	resultsModel *wrappers.ScanResultsCollection,
	scaPackageModel *[]wrappers.ScaPackageCollection,
	scaTypeModel *[]wrappers.ScaTypeCollection,
) *wrappers.ScanResultsCollection {
	locationsByID, typesByCVE := buildAuxiliaryScaMaps(resultsModel, scaPackageModel, scaTypeModel)
	scaPackageMap := buildScaPackageMap(*scaPackageModel)

	for _, result := range resultsModel.Results {
		if result.Type == commonParams.ScaType {
			processResult(result, locationsByID, typesByCVE, scaPackageMap)
		}
	}

	return resultsModel
}

func processResult(
	result *wrappers.ScanResult,
	locationsByID map[string][]*string,
	typesByCVE map[string]wrappers.ScaTypeCollection,
	scaPackageMap map[string]wrappers.ScaPackageCollection, // Updated parameter
) {
	const precision = 1

	currentID := result.ScanResultData.PackageIdentifier
	result.VulnerabilityDetails.CvssScore = util.RoundFloat(result.VulnerabilityDetails.CvssScore, precision)
	result.ScaType = buildScaType(typesByCVE, result)
	result.State = buildScaState(typesByCVE, result)

	updatePackages(result, scaPackageMap, locationsByID, currentID)
}

func updatePackages(
	result *wrappers.ScanResult,
	scaPackageMap map[string]wrappers.ScaPackageCollection,
	locationsByID map[string][]*string,
	currentID string,
) {
	packages, found := scaPackageMap[currentID]
	if !found {
		return
	}

	updateDependencyPaths(packages.DependencyPathArray, locationsByID)
	if !packages.SupportsQuickFix {
		packages.SupportsQuickFix = hasQuickFix(packages.DependencyPathArray)
	}

	if packages.IsDirectDependency {
		packages.TypeOfDependency = directDependencyType
	} else {
		packages.TypeOfDependency = indirectDependencyType
	}

	packages.FixLink = buildFixLink(result)
	result.ScanResultData.ScaPackageCollection = &packages
}

func hasQuickFix(dependencyPaths [][]wrappers.DependencyPath) bool {
	for i := range dependencyPaths {
		head := &dependencyPaths[i][0]
		if head.SupportsQuickFix {
			return true
		}
	}
	return false
}

func buildScaPackageMap(scaPackageModel []wrappers.ScaPackageCollection) map[string]wrappers.ScaPackageCollection {
	scaPackageMap := make(map[string]wrappers.ScaPackageCollection)
	for i := range scaPackageModel {
		scaPackageMap[scaPackageModel[i].ID] = scaPackageModel[i]
	}
	return scaPackageMap
}

func updateDependencyPaths(dependencyPaths [][]wrappers.DependencyPath, locationsByID map[string][]*string) {
	for i := range dependencyPaths {
		head := &dependencyPaths[i][0]
		head.Locations = locationsByID[head.ID]
		head.SupportsQuickFix = len(dependencyPaths[i]) == 1

		for _, location := range locationsByID[head.ID] {
			head.SupportsQuickFix = head.SupportsQuickFix && util.IsPackageFileSupported(*location)
		}
	}
}

func buildFixLink(result *wrappers.ScanResult) string {
	if result.ID != "" {
		return fmt.Sprint(fixLinkPrefix, result.ID)
	}
	return ""
}

func filterViolatedRules(policyModel wrappers.PolicyResponseModel) *wrappers.PolicyResponseModel {
	i := 0
	for _, policy := range policyModel.Policies {
		if len(policy.RulesViolated) > 0 {
			policyModel.Policies[i] = policy
			i++
		}
	}
	policyModel.Policies = policyModel.Policies[:i]
	return &policyModel
}

func trimOsSeparatorFromFileName(result *wrappers.ScanResult) {
	if result.ScanResultData.Filename != "" {
		result.ScanResultData.Filename = strings.TrimPrefix(result.ScanResultData.Filename, "/")
		result.ScanResultData.Filename = strings.TrimPrefix(result.ScanResultData.Filename, "\\")
	}
}

type ScannerResponse struct {
	ScanID    string `json:"ScanID,omitempty"`
	Name      string `json:"Name,omitempty"`
	Status    string `json:"Status,omitempty"`
	Details   string `json:"Details,omitempty"`
	ErrorCode string `json:"ErrorCode,omitempty"`
}

func parseURI(summaryBaseURI string) (hostName string) {
	parsedURL, err := url.Parse(summaryBaseURI)
	if err != nil {
		return ""
	}
	hostName = fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	return hostName
}

func printWarningIfIgnorePolicyOmiited() {
	fmt.Printf("\n            Warning: The --ignore-policy flag was not implemented because you don’t have the required permission.\n                     Only users with 'override-policy-management' permission can use this flag.                     \n\n")
}
