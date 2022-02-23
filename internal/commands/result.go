package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/MakeNowJust/heredoc"
	"github.com/checkmarx/ast-cli/internal/commands/util"

	commonParams "github.com/checkmarx/ast-cli/internal/params"

	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	failedListingResults     = "Failed listing results"
	failedListingCodeBashing = "Failed codebashing link"
	failedReadingParams      = "Failed reading flag values"
	mediumLabel              = "medium"
	highLabel                = "high"
	lowLabel                 = "low"
	sonarTypeLabel           = "_sonar"
	directoryPermission      = 0700
	infoSonar                = "INFO"
	lowSonar                 = "MINOR"
	mediumSonar              = "MAJOR"
	highSonar                = "CRITICAL"
	vulnerabilitySonar       = "VULNERABILITY"
	infoCx                   = "INFO"
	lowCx                    = "LOW"
	mediumCx                 = "MEDIUM"
	highCx                   = "HIGH"
)

var filterResultsListFlagUsage = fmt.Sprintf(
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
			commonParams.StateQueryParam,
		}, ",",
	),
)

// NewResultCommand - Deprecated command
func NewResultCommand(resultsWrapper wrappers.ResultsWrapper, scanWrapper wrappers.ScansWrapper) *cobra.Command {
	resultCmd := &cobra.Command{
		Use:   "result",
		Short: "Retrieve results",
		RunE:  runGetResultCommand(resultsWrapper, scanWrapper),
	}
	addScanIDFlag(resultCmd, "ID to report on.")
	addResultFormatFlag(resultCmd,
		util.FormatJSON, util.FormatSummary, util.FormatSummaryConsole, util.FormatSarif, util.FormatSummaryJSON)
	resultCmd.PersistentFlags().String(commonParams.TargetFlag, "cx_result", "Output file")
	resultCmd.PersistentFlags().String(commonParams.TargetPathFlag, ".", "Output Path")
	resultCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterResultsListFlagUsage)
	resultCmd.Deprecated = "please use 'results show' command instead."
	return resultCmd
}

func NewResultsCommand(resultsWrapper wrappers.ResultsWrapper, scanWrapper wrappers.ScansWrapper, codeBashingWrapper wrappers.CodeBashingWrapper) *cobra.Command {
	resultCmd := &cobra.Command{
		Use:   "results",
		Short: "Retrieve results",
		Annotations: map[string]string{
			"command:doc": heredoc.Doc(
				`
				https://checkmarx.atlassian.net/l/c/6NqgVMPM
			`,
			),
		},
	}
	showResultCmd := resultShowSubCommand(resultsWrapper, scanWrapper)
	codeBashingCmd := resultCodeBashing(codeBashingWrapper)
	resultCmd.AddCommand(
		showResultCmd,
		codeBashingCmd,
	)
	return resultCmd
}

func resultShowSubCommand(resultsWrapper wrappers.ResultsWrapper, scanWrapper wrappers.ScansWrapper) *cobra.Command {
	resultCmd := &cobra.Command{
		Use:   "show",
		Short: "Show results of a scan",
		Long:  "The show command enables the ability to show results about a requested scan in CxAST.",
		Example: heredoc.Doc(
			`
			$ cx results show --scan-id <scan Id>
		`,
		),
		RunE: runGetResultCommand(resultsWrapper, scanWrapper),
	}
	addScanIDFlag(resultCmd, "ID to report on.")
	addResultFormatFlag(resultCmd,
		util.FormatJSON, util.FormatSummary, util.FormatSummaryConsole, util.FormatSarif, util.FormatSummaryJSON)
	resultCmd.PersistentFlags().String(commonParams.TargetFlag, "cx_result", "Output file")
	resultCmd.PersistentFlags().String(commonParams.TargetPathFlag, ".", "Output Path")
	resultCmd.PersistentFlags().StringSlice(commonParams.FilterFlag, []string{}, filterResultsListFlagUsage)
	return resultCmd
}

func resultCodeBashing(codeBashingWrapper wrappers.CodeBashingWrapper) *cobra.Command {
	// Create a codeBashing wrapper
	resultCmd := &cobra.Command{
		Use:   "codebashing",
		Short: "Get codebashing lesson link",
		Long:  "The codebashing command enables the ability to retrieve the link about a specific vulnerability.",
		Example: heredoc.Doc(
			`
			$ cx codebashing --language <string> --vulnerabity-type <string> --cwe-id <string> --format <string>
		`,
		),
		RunE: runGetCodeBashingCommand(codeBashingWrapper),
	}
	resultCmd.PersistentFlags().String(commonParams.LanguageFlag, "", "Language")
	err := resultCmd.MarkPersistentFlagRequired(commonParams.LanguageFlag)
	if err != nil {
		log.Fatal(err)
	}
	resultCmd.PersistentFlags().String(commonParams.VulnerabilityTypeFlag, "", "Vulnerability Type")
	err = resultCmd.MarkPersistentFlagRequired(commonParams.VulnerabilityTypeFlag)
	if err != nil {
		log.Fatal(err)
	}
	resultCmd.PersistentFlags().String(commonParams.CweIDFlag, "", "CWE Id")
	err = resultCmd.MarkPersistentFlagRequired(commonParams.CweIDFlag)
	if err != nil {
		log.Fatal(err)
	}
	resultCmd.PersistentFlags().String(commonParams.FormatFlag, "json", "Format")
	return resultCmd
}

func getScanInfo(scansWrapper wrappers.ScansWrapper, scanID string) (*wrappers.ResultSummary, error) {
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

func SummaryReport(
	scanWrapper wrappers.ScansWrapper,
	results *wrappers.ScanResultsCollection,
	scanID string,
) (*wrappers.ResultSummary, error) {
	summary, err := getScanInfo(scanWrapper, scanID)
	if err != nil {
		return nil, err
	}
	summary.BaseURI = wrappers.GetURL(fmt.Sprintf("projects/%s/overview", summary.ProjectID))
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
	engineType := strings.TrimSpace(result.Type)
	if engineType == commonParams.SastType {
		summary.SastIssues++
	} else if engineType == commonParams.ScaType {
		summary.ScaIssues++
	} else if engineType == commonParams.KicsType {
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
	log.Println("Creating Summary Report: ", targetFile)
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

	fmt.Printf("        Kics Issues: %d\n", summary.KicsIssues)
	fmt.Printf("      CxSAST Issues: %d\n", summary.SastIssues)
	fmt.Printf("       CxSCA Issues: %d\n", summary.ScaIssues)

	return nil
}

func runGetResultCommand(
	resultsWrapper wrappers.ResultsWrapper,
	scanWrapper wrappers.ScansWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		targetFile, _ := cmd.Flags().GetString(commonParams.TargetFlag)
		targetPath, _ := cmd.Flags().GetString(commonParams.TargetPathFlag)
		format, _ := cmd.Flags().GetString(commonParams.TargetFormatFlag)
		scanID, _ := cmd.Flags().GetString(commonParams.ScanIDFlag)
		params, err := getFilters(cmd)
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingResults)
		}
		return CreateScanReport(resultsWrapper, scanWrapper, scanID, format, targetFile, targetPath, params)
	}
}

func runGetCodeBashingCommand(
	codeBashingWrapper wrappers.CodeBashingWrapper,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		params := make(map[string]string)
		language, err := cmd.Flags().GetString(commonParams.LanguageFlag)
		if err != nil {
			return errors.Wrapf(err, "%s", failedReadingParams)
		}
		cwe, err := cmd.Flags().GetString(commonParams.CweIDFlag)
		if err != nil {
			return errors.Wrapf(err, "%s", failedReadingParams)
		}
		vulType, err := cmd.Flags().GetString(commonParams.VulnerabilityTypeFlag)
		if err != nil {
			return errors.Wrapf(err, "%s", failedReadingParams)
		}
		params["results"] = "[{\"lang\": \"" + language + "\", \"cwe_id\":\"CWE-" + cwe + "\", \"cxQueryName\":\"" + strings.ReplaceAll(vulType, " ", "_") + "\"}]"
		CodeBashingModel, errorModel, err := codeBashingWrapper.GetCodeBashingLinks(params)
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingCodeBashing)
		}
		if errorModel != nil {
			return errors.Wrapf(err, "%s", failedListingCodeBashing)
		}
		err = printByFormat(cmd, *CodeBashingModel)
		if err != nil {
			return errors.Wrapf(err, "%s", failedListingCodeBashing)
		}
		return nil
	}
}

func CreateScanReport(
	resultsWrapper wrappers.ResultsWrapper,
	scanWrapper wrappers.ScansWrapper,
	scanID,
	reportTypes,
	targetFile,
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
	summary, err := SummaryReport(scanWrapper, results, scanID)
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
	format,
	targetFile,
	targetPath string,
	results *wrappers.ScanResultsCollection,
	summary *wrappers.ResultSummary,
) error {
	if util.IsFormat(format, util.FormatSarif) {
		sarifRpt := createTargetName(targetFile, targetPath, "sarif")
		return exportSarifResults(sarifRpt, results)
	}
	if util.IsFormat(format, util.FormatSonar) {
		sonarRpt := createTargetName(fmt.Sprintf("%s%s", targetFile, sonarTypeLabel), targetPath, "json")
		return exportSonarResults(sonarRpt, results)
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
	if util.IsFormat(format, util.FormatSummaryJSON) {
		summaryRpt := createTargetName(targetFile, targetPath, "json")
		return exportJSONSummaryResults(summaryRpt, summary)
	}
	err := fmt.Errorf("bad report format %s", format)
	return err
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
	scanID string,
	params map[string]string,
) (results *wrappers.ScanResultsCollection, err error) {
	var resultsModel *wrappers.ScanResultsCollection
	var errorModel *wrappers.WebError
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
func exportJSONResults(targetFile string, results *wrappers.ScanResultsCollection) error {
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

func convertCxResultsToSarif(results *wrappers.ScanResultsCollection) *wrappers.SarifResultsCollection {
	var sarif = new(wrappers.SarifResultsCollection)
	sarif.Schema = "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json"
	sarif.Version = "2.1.0"
	sarif.Runs = []wrappers.SarifRun{}
	sarif.Runs = append(sarif.Runs, createSarifRun(results))
	return sarif
}

func convertCxResultsToSonar(results *wrappers.ScanResultsCollection) *wrappers.ScanResultsSonar {
	var sonar = new(wrappers.ScanResultsSonar)
	sonar.Results = parseResultsSonar(results)
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

func parseResultsSonar(results *wrappers.ScanResultsCollection) []wrappers.SonarIssues {
	var sonarIssues []wrappers.SonarIssues
	// Match cx severity with sonar severity
	severities := map[string]string{
		infoCx:   infoSonar,
		lowCx:    lowSonar,
		mediumCx: mediumSonar,
		highCx:   highSonar,
	}
	if results != nil {
		for _, result := range results.Results {
			var auxIssue wrappers.SonarIssues
			auxIssue.Severity = severities[result.Severity]
			auxIssue.Type = vulnerabilitySonar
			auxIssue.EngineID = result.Type
			auxIssue.RuleID = result.ID
			auxIssue.EffortMinutes = 0

			engineType := strings.TrimSpace(result.Type)

			if engineType == commonParams.SastType {
				auxIssue.PrimaryLocation = parseSonarPrimaryLocation(result)
				auxIssue.SecondaryLocations = parseSonarSecondaryLocations(result)
				sonarIssues = append(sonarIssues, auxIssue)
			} else if engineType == commonParams.KicsType {
				auxIssue.PrimaryLocation = parseLocationKics(result)
				sonarIssues = append(sonarIssues, auxIssue)
			}
		}
	}
	return sonarIssues
}

func parseLocationKics(results *wrappers.ScanResult) wrappers.SonarLocation {
	var auxLocation wrappers.SonarLocation
	auxLocation.FilePath = strings.TrimLeft(results.ScanResultData.Filename, "/")
	auxLocation.Message = results.ScanResultData.Value
	var auxTextRange wrappers.SonarTextRange
	auxTextRange.StartLine = results.ScanResultData.Line
	auxTextRange.StartColumn = 1
	auxTextRange.EndColumn = 2
	auxLocation.TextRange = auxTextRange
	return auxLocation
}

func parseSonarPrimaryLocation(results *wrappers.ScanResult) wrappers.SonarLocation {
	var auxLocation wrappers.SonarLocation
	// fill the details in the primary Location
	if len(results.ScanResultData.Nodes) > 0 {
		auxLocation.FilePath = strings.TrimLeft(results.ScanResultData.Nodes[0].FileName, "/")
		auxLocation.Message = strings.ReplaceAll(results.ScanResultData.QueryName, "_", " ")
		auxLocation.TextRange = parseSonarTextRange(results.ScanResultData.Nodes[0])
	}
	return auxLocation
}

func parseSonarSecondaryLocations(results *wrappers.ScanResult) []wrappers.SonarLocation {
	var auxSecondaryLocations []wrappers.SonarLocation
	// Traverse all the rest of the scan result nodes into secondary location of sonar
	if len(results.ScanResultData.Nodes) > 1 {
		for _, node := range results.ScanResultData.Nodes[1:] {
			var auxSecondaryLocation wrappers.SonarLocation
			auxSecondaryLocation.FilePath = strings.TrimLeft(node.FileName, "/")
			auxSecondaryLocation.Message = strings.ReplaceAll(results.ScanResultData.QueryName, "_", " ")
			auxSecondaryLocation.TextRange = parseSonarTextRange(node)
			auxSecondaryLocations = append(auxSecondaryLocations, auxSecondaryLocation)
		}
	}
	return auxSecondaryLocations
}

func parseSonarTextRange(results *wrappers.ScanResultNode) wrappers.SonarTextRange {
	var auxTextRange wrappers.SonarTextRange
	auxTextRange.StartLine = results.Line
	auxTextRange.StartColumn = results.Column - 1
	auxTextRange.EndColumn = results.Column
	return auxTextRange
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
	scanResult.Message.Text = result.ScanResultData.QueryName
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
