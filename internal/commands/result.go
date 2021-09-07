package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/checkmarxDev/ast-cli/internal/commands/util"
	resultsHelpers "github.com/checkmarxDev/sast-results/pkg/web/helpers"

	commonParams "github.com/checkmarxDev/ast-cli/internal/params"

	"github.com/checkmarxDev/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedListingResults = "Failed listing results"
	mediumLabel          = "medium"
	highLabel            = "high"
	lowLabel             = "low"
	sastTypeLabel        = "sast"
	kicsTypeLabel        = "infrastructure"
	scaTypeLabel         = "dependency"
)

var (
	filterResultsListFlagUsage = fmt.Sprintf(
		"Filter the list of results. Use ';' as the delimeter for arrays. Available filters are: %s",
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
			}, ",",
		),
	)
	scanAPIPath = ""
)

func NewResultCommand(resultsWrapper wrappers.ResultsWrapper) *cobra.Command {
	scanAPIPath = resultsWrapper.GetScaAPIPath()
	resultCmd := &cobra.Command{
		Use:   "result",
		Short: "Retrieve results",
		RunE:  runGetResultCommand(resultsWrapper),
	}
	addScanIDFlag(resultCmd, "ID to report on.")
	addResultFormatFlag(resultCmd, util.FormatJSON, util.FormatSummary, util.FormatSummaryConsole, util.FormatSarif)
	resultCmd.PersistentFlags().String(TargetFlag, "cx_result", "Output file")
	resultCmd.PersistentFlags().String(TargetPathFlag, ".", "Output Path")
	resultCmd.PersistentFlags().StringSlice(FilterFlag, []string{}, filterResultsListFlagUsage)
	return resultCmd
}

func getScanInfo(scanID string) (*wrappers.ResultSummary, error) {
	scansWrapper := wrappers.NewHTTPScansWrapper(scanAPIPath)
	scanInfo, errorModel, err := scansWrapper.GetByID(scanID)
	if err != nil {
		return nil, errors.Wrapf(err, "%s", failedGetting)
	}
	if errorModel != nil {
		return nil, errors.Errorf("%s: CODE: %d, %s", failedGetting, errorModel.Code, errorModel.Message)
	} else if scanInfo != nil {
		return &wrappers.ResultSummary{
			ScanID:       scanInfo.ID,
			Status:       string(scanInfo.Status),
			CreatedAt:    scanInfo.CreatedAt.Format("2006-01-02, 15:04:05"),
			ProjectID:    scanInfo.ProjectID,
			RiskStyle:    "",
			RiskMsg:      "",
			HighIssues:   0,
			MediumIssues: 0,
			LowIssues:    0,
			SastIssues:   0,
			KicsIssues:   0,
			ScaIssues:    0,
			Tags:         scanInfo.Tags,
		}, nil
	}
	return nil, err
}

func SummaryReport(results *wrappers.ScanResultsCollection, scanID string) (*wrappers.ResultSummary, error) {
	summary, err := getScanInfo(scanID)
	if err != nil {
		return nil, err
	}
	summary.TotalIssues = int(results.TotalCount)
	for _, result := range results.Results {
		countResult(summary, result)
	}
	if summary.HighIssues > 0 {
		summary.RiskStyle = highLabel
		summary.RiskMsg = "High Risk"
	} else if summary.MediumIssues > 0 {
		summary.RiskStyle = mediumLabel
		summary.RiskMsg = "Medium Risk"
	} else if summary.LowIssues > 0 {
		summary.RiskStyle = lowLabel
		summary.RiskMsg = "Low Risk"
	}
	return summary, nil
}

func countResult(summary *wrappers.ResultSummary, result *wrappers.ScanResult) {
	if result.Type == sastTypeLabel {
		summary.SastIssues++
	} else if result.Type == scaTypeLabel {
		summary.ScaIssues++
	} else if result.Type == kicsTypeLabel {
		summary.KicsIssues++
	}
	severity := strings.ToLower(result.Severity)
	if severity == highLabel {
		summary.HighIssues++
	} else if severity == lowLabel {
		summary.LowIssues++
	} else if severity == mediumLabel {
		summary.MediumIssues++
	}
}

func writeHTMLSummary(targetFile string, summary *wrappers.ResultSummary) error {
	fmt.Println("Creating Summary Report: ", targetFile)
	summaryTemp, err := template.New("summaryTemplate").Parse(wrappers.SummaryTemplate)
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

func writeConsoleSummary(summary *wrappers.ResultSummary) error {
	fmt.Println("")
	fmt.Printf("         Created At: %s\n", summary.CreatedAt)
	fmt.Printf("               Risk: %s\n", summary.RiskMsg)
	fmt.Printf("         Project ID: %s\n", summary.ProjectID)
	fmt.Printf("            Scan ID: %s\n", summary.ScanID)
	fmt.Printf("       Total Issues: %d\n", summary.TotalIssues)
	fmt.Printf("        High Issues: %d\n", summary.HighIssues)
	fmt.Printf("      Medium Issues: %d\n", summary.MediumIssues)
	fmt.Printf("         Low Issues: %d\n", summary.LowIssues)

	fmt.Printf("         Kics Issues: %d\n", summary.KicsIssues)
	fmt.Printf("         Sast Issues: %d\n", summary.SastIssues)
	fmt.Printf("         SCA Issues: %d\n", summary.ScaIssues)

	return nil
}

func runGetResultCommand(resultsWrapper wrappers.ResultsWrapper) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		targetFile, _ := cmd.Flags().GetString(TargetFlag)
		targetPath, _ := cmd.Flags().GetString(TargetPathFlag)
		format, _ := cmd.Flags().GetString(TargetFormatFlag)
		scanID, _ := cmd.Flags().GetString(ScanIDFlag)
		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingResults)
		}
		return CreateScanReport(resultsWrapper, scanID, format, targetFile, targetPath, params)
	}
}

func CreateScanReport(
	resultsWrapper wrappers.ResultsWrapper,
	scanID string,
	reportTypes string,
	targetFile string,
	targetPath string,
	params map[string]string,
) error {
	if scanID == "" {
		return errors.Errorf("%s: Please provide a scan ID", failedListingResults)
	}
	err := createDirectory(targetPath)
	if err != nil {
		return err
	}
	results, err := ReadResults(resultsWrapper, scanID, params)
	if err != nil {
		return err
	}
	summary, err := SummaryReport(results, scanID)
	if err != nil {
		return err
	}
	reportList := strings.Split(reportTypes, ",")
	for _, reportType := range reportList {
		err = createReport(reportType, targetFile, targetPath, results, summary)
		if err != nil {
			return err
		}
	}
	return nil
}

func createReport(
	format string,
	targetFile string,
	targetPath string,
	results *wrappers.ScanResultsCollection,
	summary *wrappers.ResultSummary,
) error {
	if util.IsFormat(format, util.FormatSarif) {
		sarifRpt := createTargetName(targetFile, targetPath, "sarif")
		return exportSarifResults(sarifRpt, results)
	}
	if util.IsFormat(format, util.FormatJSON) {
		jsonRpt := createTargetName(targetFile, targetPath, "json")
		return exportJSONResults(jsonRpt, results)
	}
	if util.IsFormat(format, util.FormatSummaryConsole) {
		return writeConsoleSummary(summary)
	}
	if util.IsFormat(format, util.FormatSummary) {
		summaryRpt := createTargetName(targetFile, targetPath, "html")
		return writeHTMLSummary(summaryRpt, summary)
	}
	err := fmt.Errorf("bad report format %s", format)
	return err
}

func createTargetName(targetFile, targetPath, targetType string) string {
	return filepath.Join(targetPath, targetFile+"."+targetType)
}

func createDirectory(targetPath string) error {
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		fmt.Printf("\nOutput path not found: %s\n", targetPath)
		fmt.Printf("Creating directory: %s\n", targetPath)
		err = os.Mkdir(targetPath, 0600)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadResults(
	resultsWrapper wrappers.ResultsWrapper,
	scanID string,
	params map[string]string,
) (results *wrappers.ScanResultsCollection, err error) {
	var resultsModel *wrappers.ScanResultsCollection
	var errorModel *resultsHelpers.WebError
	params[commonParams.ScanIDQueryParam] = scanID
	resultsModel, errorModel, err = resultsWrapper.GetAllResultsByScanID(params)
	if err != nil {
		return nil, errors.Wrapf(err, "%s", failedListingResults)
	}
	if errorModel != nil {
		return nil, errors.Errorf("%s: CODE: %d, %s", failedListingResults, errorModel.Code, errorModel.Message)
	} else if resultsModel != nil {
		return resultsModel, nil
	}
	return nil, nil
}

func exportSarifResults(targetFile string, results *wrappers.ScanResultsCollection) error {
	var err error
	var resultsJSON []byte
	fmt.Println("Creating SARIF Report: ", targetFile)
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

func exportJSONResults(targetFile string, results *wrappers.ScanResultsCollection) error {
	var err error
	var resultsJSON []byte
	fmt.Println("Creating JSON Report: ", targetFile)
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

func convertCxResultsToSarif(results *wrappers.ScanResultsCollection) *wrappers.SarifResultsCollection {
	var sarif = new(wrappers.SarifResultsCollection)
	sarif.Schema = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json"
	sarif.Version = "2.1.0"
	sarif.Runs = []wrappers.SarifRun{}
	sarif.Runs = append(sarif.Runs, createSarifRun(results))
	return sarif
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
	var sarifRules []wrappers.SarifDriverRule
	var sarifResults []wrappers.SarifScanResult
	if results != nil {
		ruleIds := map[interface{}]bool{}
		for _, result := range results.Results {
			if rule := findRule(ruleIds, result); rule != nil {
				sarifRules = append(sarifRules, *rule)
			}

			if sarifResult := findResult(result); sarifResult != nil {
				sarifResults = append(sarifResults, *sarifResult)
			}
		}
	}
	return sarifRules, sarifResults
}

func findRule(ruleIds map[interface{}]bool, result *wrappers.ScanResult) *wrappers.SarifDriverRule {
	var sarifRule wrappers.SarifDriverRule

	if result.ScanResultData.QueryID == nil {
		sarifRule.ID = result.ID
	} else {
		sarifRule.ID = getRuleID(result.ScanResultData.QueryID)
	}

	if result.ScanResultData.QueryName != "" {
		sarifRule.Name = result.ScanResultData.QueryName
	}

	sarifRule.HelpURI = wrappers.SarifInformationURI

	if !ruleIds[sarifRule.ID] {
		ruleIds[sarifRule.ID] = true
		return &sarifRule
	}

	return nil
}

func findResult(result *wrappers.ScanResult) *wrappers.SarifScanResult {
	var scanResult wrappers.SarifScanResult
	scanResult.RuleID = getRuleID(result.ScanResultData.QueryID)
	scanResult.Message.Text = result.Comments.Comments
	scanResult.PartialFingerprints.PrimaryLocationLineHash = result.SimilarityID
	scanResult.Locations = []wrappers.SarifLocation{}

	for _, node := range result.ScanResultData.Nodes {
		var scanLocation wrappers.SarifLocation
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
	if len(scanResult.Locations) > 0 {
		return &scanResult
	}
	return nil
}

// getRuleID this method should be unnecessary when AST fixes the queryId field's type
func getRuleID(queryID interface{}) string {
	switch queryID.(type) {
	case float64:
		return fmt.Sprintf("%0.f", queryID)
	default:
		return fmt.Sprintf("%v", queryID)
	}
}
