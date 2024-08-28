//go:build !integration

package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/checkmarx/ast-cli/internal/commands/util/printer"
	errorConstants "github.com/checkmarx/ast-cli/internal/constants/errors"
	params "github.com/checkmarx/ast-cli/internal/params"
	"github.com/checkmarx/ast-cli/internal/wrappers"
	"github.com/checkmarx/ast-cli/internal/wrappers/mock"
	"gotest.tools/assert"
)

const fileName = "cx_result"

const (
	resultsCommand      = "results"
	codeBashingCommand  = "codebashing"
	vulnerabilityValue  = "Reflected XSS All Clients"
	languageValue       = "PHP"
	cweValue            = "79"
	jsonValue           = "json"
	tableValue          = "table"
	listValue           = "list"
	secretDetectionLine = "| Secret Detection          0      5        3      2      0   Completed  |"
)

func flag(f string) string {
	return "--" + f
}

func TestResultHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "results")
}

func TestResultsExitCode_CompletedScan_PrintCorrectInfoToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{ID: "MOCK", Status: wrappers.ScanCompleted, Engines: []string{params.ScaType, params.SastType, params.KicsType}}
	results := getScannerResponse("", &model)
	assert.Equal(t, len(results), 1, "")
	assert.Equal(t, results[0].ScanID, "MOCK", "")
	assert.Equal(t, results[0].Status, wrappers.ScanCompleted, "")
}

func TestResultsExitCode_OnFailedKicsScanner_PrintCorrectFailedScannerInfoToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-kics-scanner-fail",
		Status: wrappers.ScanFailed,
		StatusDetails: []wrappers.StatusInfo{
			{
				Status:    wrappers.ScanFailed,
				Name:      "kics",
				Details:   "error message from kics scanner",
				ErrorCode: 1234,
			},
			{Status: wrappers.ScanFailed, Name: "general", Details: "timeout", ErrorCode: 1234},
		},
	}

	results := getScannerResponse("", &model)

	assert.Equal(t, len(results), 2, "Scanner results should be empty")
	assert.Equal(t, results[0].Name, "kics", "")
	assert.Equal(t, results[0].ErrorCode, "1234", "")
	assert.Equal(t, results[1].Name, "general", "")
	assert.Equal(t, results[1].ErrorCode, "1234", "")
	assert.Equal(t, results[1].Details, "timeout", "")
}

func TestResultsExitCode_OnFailedKicsAndScaScanners_PrintCorrectFailedScannersInfoToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-multiple-scanner-fails",
		Status: wrappers.ScanFailed,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 2344},
			{Status: wrappers.ScanFailed, Name: "sca", Details: "error message from sca scanner", ErrorCode: 4343},
			{Status: wrappers.ScanFailed, Name: "general", Details: "timeout", ErrorCode: 1234},
		},
	}

	results := getScannerResponse("", &model)

	assert.Equal(t, len(results), 3, "Scanner results should be empty")
	assert.Equal(t, results[0].Name, "kics", "")
	assert.Equal(t, results[0].ErrorCode, "2344", "")
	assert.Equal(t, results[1].Name, "sca", "")
	assert.Equal(t, results[1].ErrorCode, "4343", "")
	assert.Equal(t, results[2].Name, "general", "")
	assert.Equal(t, results[2].ErrorCode, "1234", "")
	assert.Equal(t, results[2].Details, "timeout", "")
}

func TestResultsExitCode_OnRequestedFailedScanner_PrintCorrectFailedScannerInfoToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-multiple-scanner-fails",
		Status: wrappers.ScanFailed,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 2344},
			{Status: wrappers.ScanFailed, Name: "sca", Details: "error message from sca scanner", ErrorCode: 4343},
			{Status: wrappers.ScanFailed, Name: "general", Details: "timeout", ErrorCode: 1234},
		},
	}

	results := getScannerResponse("sca", &model)

	assert.Equal(t, len(results), 1, "Scanner results should be empty")
	assert.Equal(t, results[0].Name, "sca", "")
	assert.Equal(t, results[0].ErrorCode, "4343", "")
}

func TestResultsExitCode_OnPartialScan_PrintOnlyFailedScannersInfoToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-sca-fail-partial-id",
		Status: wrappers.ScanPartial,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanCompleted, Name: "sast"},
			{Status: wrappers.ScanCanceled, Name: "sca", Details: "error message from sca scanner", ErrorCode: 4343},
			{Status: wrappers.ScanCompleted, Name: "general"},
		},
	}

	results := getScannerResponse("", &model)

	assert.Equal(t, len(results), 1, "Scanner results should be empty")
	assert.Equal(t, results[0].ScanID, "fake-scan-id-sca-fail-partial-id", "")
	assert.Equal(t, results[0].Status, "Partial", "")
}

var executeCommand = func(t *testing.T, agent string) *wrappers.ScanResultsCollection {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.SCSEngineCLIEnabled, Status: true}

	_, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "SCS", "--report-format", "json", "--agent", agent)
	assert.NilError(t, err)

	file, err := os.Open(fileName + ".json")
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	fileContents, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var results wrappers.ScanResultsCollection
	err = json.Unmarshal(fileContents, &results)
	assert.NilError(t, err)
	return &results
}

func TestRunScsResultsShow_ASTCLI_AgentShouldShowAllResults(t *testing.T) {
	results := executeCommand(t, params.DefaultAgent)
	scsSecretDetectionFound := false
	scsScorecardFound := false
	for _, result := range results.Results {
		if result.Type == params.SCSSecretDetectionType {
			scsSecretDetectionFound = true
		}
		if result.Type == params.SCSScorecardType {
			scsScorecardFound = true
		}
		if scsSecretDetectionFound && scsScorecardFound {
			break
		}
	}
	assert.Assert(t, scsSecretDetectionFound && scsScorecardFound, "SCS results should be included for AST-CLI agent")
	assert.Assert(t, results.TotalCount == 2, "SCS Scorecard results should be excluded for VS Code agent")

	os.Remove(fileName + ".json")
}

func TestRunScsResultsShow_VSCode_AgentShouldNotShowScorecardResults(t *testing.T) {
	results := executeCommand(t, params.VSCodeAgent)
	for _, result := range results.Results {
		assert.Assert(t, result.Type != params.SCSScorecardType, "SCS Scorecard results should be excluded for VS Code agent")
	}
	assert.Assert(t, results.TotalCount == 1, "SCS Scorecard results should be excluded for VS Code agent")

	os.Remove(fileName + ".json")
}

func TestRunScsResultsShow_Other_AgentsShouldNotShowScsResults(t *testing.T) {
	results := executeCommand(t, "Jetbrains")
	for _, result := range results.Results {
		assert.Assert(t, result.Type != params.SCSScorecardType && result.Type != params.SCSSecretDetectionType, "SCS results should be excluded for other agents")
	}
	assert.Assert(t, results.TotalCount == 0, "SCS Scorecard results should be excluded")

	os.Remove(fileName + ".json")
}

func TestRunWithoutScsResults_Other_AgentsShouldNotShowScsResults(t *testing.T) {
	results := executeCommand(t, "Jetbrains")
	for _, result := range results.Results {
		assert.Assert(t, result.Type != params.SCSScorecardType && result.Type != params.SCSSecretDetectionType, "SCS results should be excluded for other agents")
	}
	assert.Assert(t, results.TotalCount == 7, "SCS Scorecard results should be excluded")

	os.Remove(fileName + ".json")
}

func TestRunNilResults_Other_AgentsShouldNotShowAnyResults(t *testing.T) {

	results := executeCommand(t, "Jetbrains")

	assert.Assert(t, results.TotalCount == 0, "SCS Scorecard results should be excluded")

	os.Remove(fileName + ".json")
}

func TestResultsExitCode_OnCanceledScan_PrintOnlyScanIDAndStatusCanceledToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-kics-fail-sast-canceled-id",
		Status: wrappers.ScanCanceled,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanCompleted, Name: "general"},
			{Status: wrappers.ScanCompleted, Name: "sast"},
			{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 6455},
		},
	}

	results := getScannerResponse("", &model)

	assert.Equal(t, len(results), 1, "Scanner results should be empty")
	assert.Equal(t, results[0].ScanID, "fake-scan-id-kics-fail-sast-canceled-id", "")
	assert.Equal(t, results[0].Status, wrappers.ScanCanceled, "")
}

func TestResultsExitCode_OnCanceledScanWithRequestedSuccessfulScanner_PrintOnlyScanIDAndStatusCanceledToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-kics-fail-sast-canceled-id",
		Status: wrappers.ScanCanceled,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanCompleted, Name: "general"},
			{Status: wrappers.ScanCompleted, Name: "sast"},
			{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 6455},
		},
	}

	results := getScannerResponse("sast", &model)

	assert.Equal(t, len(results), 1, "Scanner results should be empty")
	assert.Equal(t, results[0].ScanID, "fake-scan-id-kics-fail-sast-canceled-id", "")
	assert.Equal(t, results[0].Status, wrappers.ScanCanceled, "")
}

func TestResultsExitCode_OnCanceledScanWithRequestedFailedScanner_PrintOnlyScanIDAndStatusCanceledToConsole(t *testing.T) {
	model := wrappers.ScanResponseModel{
		ID:     "fake-scan-id-kics-fail-sast-canceled-id",
		Status: wrappers.ScanCanceled,
		StatusDetails: []wrappers.StatusInfo{
			{Status: wrappers.ScanCompleted, Name: "general"},
			{Status: wrappers.ScanCompleted, Name: "sast"},
			{Status: wrappers.ScanFailed, Name: "kics", Details: "error message from kics scanner", ErrorCode: 6455},
		},
	}

	results := getScannerResponse("kics", &model)

	assert.Equal(t, len(results), 1, "Scanner results should be empty")
	assert.Equal(t, results[0].ScanID, "fake-scan-id-kics-fail-sast-canceled-id", "")
	assert.Equal(t, results[0].Status, wrappers.ScanCanceled, "")
}

func TestResultsExitCode_NoScanIdSent_FailCommandWithError(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "exit-code")
	assert.Equal(t, err.Error(), errorConstants.ScanIDRequired, "Wrong expected error message")
}

func TestResultsExitCode_OnErrorScan_FailCommandWithError(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "exit-code", "--scan-id", "fake-error-id")
	assert.Equal(t, err.Error(), "Failed showing a scan: fake error message", "Wrong expected error message")
}

func TestRunGetResultsByScanIdSarifFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sarif")
	// Remove generated sarif file
	removeFileBySuffix(t, printer.FormatSarif)
}
func TestRunGetResultsByScanIdSarifFormatWithContainers(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sarif")
	// Remove generated sarif file
	removeFileBySuffix(t, printer.FormatSarif)
}

func TestParseSarifEmptyResultSast(t *testing.T) {
	emptyResult := &wrappers.ScanResult{}
	result := parseSarifResultSast(emptyResult, nil)
	if result != nil {
		t.Errorf("Expected nil result for empty ScanResultData.Nodes, got %v", result)
	}
}

func TestRunGetResultsByScanIdSonarFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sonar")
	// Remove generated sonar file
	removeFile(t, fileName+"_"+printer.FormatSonar, printer.FormatJSON)
}

func TestRunGetResultsByScanIdSonarFormatWithContainers(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sonar")
	// Remove generated sonar file
	removeFile(t, fileName+"_"+printer.FormatSonar, printer.FormatJSON)
}

func TestRunGetResultsByScanIdJsonFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestDecodeHTMLEntitiesInResults(t *testing.T) {
	// Setup: Creating test data with HTML entities
	results := createTestScanResultsCollection()

	decodeHTMLEntitiesInResults(results)

	expectedFullName := `SomeClass<T>`
	expectedName := `Name with "quotes"`

	if results.Results[0].ScanResultData.Nodes[0].FullName != expectedFullName {
		t.Errorf("expected FullName to be %q, got %q", expectedFullName, results.Results[0].ScanResultData.Nodes[0].FullName)
	}

	if results.Results[0].ScanResultData.Nodes[0].Name != expectedName {
		t.Errorf("expected Name to be %q, got %q", expectedName, results.Results[0].ScanResultData.Nodes[0].Name)
	}
}

func TestRunGetResultsByScanIdJsonFormatWithContainers(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunGetResultsByScanIdJsonFormatWithSastRedundancy(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--sast-redundancy")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunGetResultsByScanIdSummaryJsonFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryJSON")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunGetResultsByScanIdSummaryJsonFormatWithContainers(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryJSON")

	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunGetResultsByScanIdSummaryHtmlFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryHTML")

	// Remove generated html file
	removeFileBySuffix(t, printer.FormatHTML)
}

func TestRunGetResultsByScanIdSummaryHtmlFormatWithContainers(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryHTML")

	// Remove generated html file
	removeFileBySuffix(t, printer.FormatHTML)
}

func TestRunGetResultsByScanIdSummaryConsoleFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
}

func TestRunGetResultsByScanIdSummaryMarkdownFormatWithContainers(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "markdown")
	// Remove generated md file
	removeFileBySuffix(t, "md")
}

func TestRunGetResultsByScanIdSummaryConsoleFormatWithContainers(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
}

func TestRunGetResultsByScanIdSummaryMarkdownFormat(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "markdown")
	// Remove generated md file
	removeFileBySuffix(t, "md")
}

func createTestScanResultsCollection() *wrappers.ScanResultsCollection {
	return &wrappers.ScanResultsCollection{
		Results: []*wrappers.ScanResult{
			{
				Description:     "Vulnerability in SomeComponent",
				DescriptionHTML: "Description with quotes",
				ScanResultData: wrappers.ScanResultData{
					Nodes: []*wrappers.ScanResultNode{
						{
							FullName: "SomeClass&lt;T&gt;",
							Name:     "Name with &quot;quotes&quot;",
						},
					},
				},
			},
		},
	}
}

func removeFileBySuffix(t *testing.T, suffix string) {
	removeFile(t, fileName, suffix)
}

func removeFile(t *testing.T, prefix, suffix string) {
	err := os.Remove(fmt.Sprintf("%s.%s", prefix, suffix))
	assert.NilError(t, err, "Error removing file, check if report file created")
}

func TestRunGetResultsByScanIdPDFFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "pdf")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatPDF)
	// Remove generated pdf file
	removeFileBySuffix(t, printer.FormatPDF)
}

func TestRunGetResultsByScanIdPDFFormatWithContainers(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "pdf")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatPDF)
	// Remove generated pdf file
	removeFileBySuffix(t, printer.FormatPDF)
}

func TestRunGetResultsByScanIdWrongFormat(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "invalidFormat")
	assert.Equal(t, err.Error(), "bad report format invalidFormat", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithWrongFilterFormat(t *testing.T) {
	_ = execCmdNotNilAssertion(
		t,
		"results",
		"show",
		"--scan-id",
		"MOCK",
		"--report-format",
		"sarif",
		"--filter",
		"limit40",
	)
}

func TestRunGetResultsByScanIdWithMissingOrEmptyScanId(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "show")
	assert.Equal(t, err.Error(), "Failed listing results: Please provide a scan ID", "Wrong expected error message")

	err = execCmdNotNilAssertion(t, "results", "show", "--scan-id", "")
	assert.Equal(t, err.Error(), "Failed listing results: Please provide a scan ID", "Wrong expected error message")
}

func TestRunGetResultsByScanIdWithEmptyOutputPath(t *testing.T) {
	_ = execCmdNotNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--output-path", "")
}

func TestRunGetCodeBashingWithoutLanguage(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.CweIDFlag),
		cweValue,
		flag(params.VulnerabilityTypeFlag),
		vulnerabilityValue)
	assert.Equal(t, err.Error(), "required flag(s) \"language\" not set", "Wrong expected error message")
}

func TestRunGetCodeBashingWithoutVulnerabilityType(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.CweIDFlag),
		cweValue,
		flag(params.LanguageFlag),
		languageValue)
	assert.Equal(t, err.Error(), "required flag(s) \"vulnerability-type\" not set", "Wrong expected error message")
}

func TestRunGetCodeBashingWithoutCweId(t *testing.T) {
	err := execCmdNotNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.VulnerabilityTypeFlag),
		vulnerabilityValue,
		flag(params.LanguageFlag),
		languageValue)
	assert.Equal(t, err.Error(), "required flag(s) \"cwe-id\" not set", "Wrong expected error message")
}

func TestRunGetCodeBashingWithFormatJson(t *testing.T) {
	execCmdNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.VulnerabilityTypeFlag),
		vulnerabilityValue,
		flag(params.LanguageFlag),
		languageValue,
		flag(params.CweIDFlag),
		cweValue,
		flag(params.FormatFlag),
		jsonValue)
}

func TestRunGetCodeBashingWithFormatTable(t *testing.T) {
	execCmdNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.VulnerabilityTypeFlag),
		vulnerabilityValue,
		flag(params.LanguageFlag),
		languageValue,
		flag(params.CweIDFlag),
		cweValue,
		flag(params.FormatFlag),
		tableValue)
}

func TestRunGetCodeBashingWithFormatList(t *testing.T) {
	execCmdNilAssertion(
		t,
		resultsCommand,
		codeBashingCommand,
		flag(params.VulnerabilityTypeFlag),
		vulnerabilityValue,
		flag(params.LanguageFlag),
		languageValue,
		flag(params.CweIDFlag),
		cweValue,
		flag(params.FormatFlag),
		listValue)
}

func TestResultBflHelp(t *testing.T) {
	execCmdNilAssertion(t, "help", "results bfl")
}

func TestRunGetBflWithMissingOrEmptyScanIdAndQueryId(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "bfl")
	assert.Equal(t, err.Error(), "required flag(s) \"query-id\", \"scan-id\" not set")

	err = execCmdNotNilAssertion(t, "results", "bfl", "--scan-id", "")
	assert.Equal(t, err.Error(), "required flag(s) \"query-id\" not set")

	err = execCmdNotNilAssertion(t, "results", "bfl", "--query-id", "")
	assert.Equal(t, err.Error(), "required flag(s) \"scan-id\" not set")
}

func TestRunGetBflWithMultipleScanIdsAndQueryIds(t *testing.T) {
	err := execCmdNotNilAssertion(t, "results", "bfl", "--scan-id", "MOCK1,MOCK2", "--query-id", "MOCK")
	assert.Equal(t, err.Error(), "Multiple scan-ids are not allowed.")

	err = execCmdNotNilAssertion(t, "results", "bfl", "--scan-id", "MOCK1", "--query-id", "MOCK1,MOCK2")
	assert.Equal(t, err.Error(), "Multiple query-ids are not allowed.")
}

func TestRunGetBFLByScanIdAndQueryId(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "results", "bfl", "--scan-id", "MOCK", "--query-id", "MOCK")
	assert.NilError(t, err)
}

func TestRunGetBFLByScanIdAndQueryIdWithFormatJson(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "results", "bfl", "--scan-id", "MOCK", "--query-id", "MOCK", "--format", "JSON")
	assert.NilError(t, err)
}

func TestRunGetBFLByScanIdAndQueryIdWithFormatList(t *testing.T) {
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd, "results", "bfl", "--scan-id", "MOCK", "--query-id", "MOCK", "--format", "List")
	assert.NilError(t, err)
}

func TestRunGetResultsGeneratingPdfReportWithInvalidEmail(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.NewScanReportEnabled, Status: false}
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,invalid")
	assert.Equal(t, err.Error(), "report not sent, invalid email address: invalid", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithInvalidOptions(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.NewScanReportEnabled, Status: false}
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-options", "invalid")
	assert.Equal(t, err.Error(), "report option \"invalid\" unavailable", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithInvalidImprovedOptions(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.NewScanReportEnabled, Status: false}
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-options", "scan-information")
	assert.Equal(t, err.Error(), "report option \"scan-information\" unavailable", "Wrong expected error message")
}

func TestRunGetResultsGeneratingPdfReportWithEmailAndOptions(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.NewScanReportEnabled, Status: false}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,test@test.pt",
		"--report-pdf-options", "Iac-Security,Sast,Sca,ScanSummary")
	assert.NilError(t, err)
}

func TestRunGetResultsGeneratingPdfReportWithOptionsImprovedMappingHappens(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.NewScanReportEnabled, Status: true}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,test@test.pt",
		"--report-pdf-options", "Iac-Security,Sast,Sca,scansummary,scanresults")
	assert.NilError(t, err)
}

func TestRunGetResultsGeneratingPdfReportWithInvalidOptionsImproved(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.NewScanReportEnabled, Status: true}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--report-pdf-email", "ab@cd.pt,test@test.pt",
		"--report-pdf-options", "Iac-Security,Sast,Sca,scan-information")
	assert.Error(t, err, "report option \"scan-information\" unavailable")
}

func TestRunGetResultsGeneratingPdfReportWithOptions(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.NewScanReportEnabled, Status: false}
	cmd := createASTTestCommand()
	err := executeTestCommand(cmd,
		"results", "show",
		"--report-format", "pdf",
		"--scan-id", "MOCK",
		"--output-name", fileName,
		"--report-pdf-options", "Iac-Security,Sast,Sca,ScanSummary")
	defer func() {
		removeFileBySuffix(t, printer.FormatPDF)
		fmt.Println("test file removed!")
	}()
	assert.NilError(t, err)
	_, err = os.Stat(fmt.Sprintf("%s.%s", fileName, printer.FormatPDF))
	assert.NilError(t, err, "report file should exist: "+fileName+printer.FormatPDF)
}

func TestSBOMReportInvalidSBOMOption(t *testing.T) {
	err := execCmdNotNilAssertion(t,
		"results", "show",
		"--report-format", "sbom",
		"--scan-id", "MOCK",
		"--report-sbom-format", "invalid")
	assert.Equal(t, err.Error(), "invalid SBOM option: invalid", "Wrong expected error message")
}

func TestSBOMReportJson(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sbom")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatJSON)
	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatJSON))
}

func TestSBOMReportXML(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sbom", "--report-sbom-format", "CycloneDxXml")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatXML))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatXML)
	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatXML))
}

func TestSBOMReportJsonWithContainers(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sbom")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatJSON))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatJSON)
	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatJSON))
}

func TestSBOMReportXMLWithContainers(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "sbom", "--report-sbom-format", "CycloneDxXml")
	_, err := os.Stat(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatXML))
	assert.NilError(t, err, "Report file should exist for extension "+printer.FormatXML)
	// Remove generated json file
	os.Remove(fmt.Sprintf("%s.%s", fileName+"_"+printer.FormatSbom, printer.FormatXML))
}

func TestRunGetResultsByScanIdGLFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "gl-sast")
	// Run test for gl-sast report type
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatGLSast))
}
func TestRunResultsShow_ContainersFFIsOn_includeContainersResult(t *testing.T) {
	clearFlags()
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")
	assertContainersPresent(t, true)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}
func TestRunResultsShow_ContainersFFIsOff_excludeContainersResult(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: false}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json")
	assertContainersPresent(t, false)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}
func TestRunResultsShow_jetbrainsIsNotSupported_excludeContainersResult(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--agent", "jetbrains")
	assertContainersPresent(t, false)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunResultsShow_EclipseIsNotSupported_excludeContainersResult(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--agent", "Eclipse")
	assertContainersPresent(t, false)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunResultsShow_VsCodeIsNotSupported_excludeContainersResult(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--agent", "vs code")
	assertContainersPresent(t, false)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func TestRunResultsShow_VisualStudioIsNotSupported_excludeContainersResult(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: true}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "json", "--agent", "Visual Studio")
	assertContainersPresent(t, false)
	// Remove generated json file
	removeFileBySuffix(t, printer.FormatJSON)
}

func assertContainersPresent(t *testing.T, isContainersEnabled bool) {
	bytes, err := os.ReadFile(fileName + "." + printer.FormatJSON)
	assert.NilError(t, err, "Error reading file")
	// Unmarshal the JSON data into the ScanResultsCollection struct
	var scanResultsCollection *wrappers.ScanResultsCollection
	err = json.Unmarshal(bytes, &scanResultsCollection)
	assert.NilError(t, err, "Error unmarshalling JSON data")
	for _, scanResult := range scanResultsCollection.Results {
		if !isContainersEnabled && scanResult.Type == params.ContainersType {
			assert.Assert(t, false, "Containers result should not be present")
		} else if isContainersEnabled && scanResult.Type == params.ContainersType {
			return
		}
	}
	if isContainersEnabled {
		assert.Assert(t, false, "Containers result should be present")
	}
}
func TestRunGetResultsShow_ContainersFFOffAndResultsHasContainersResultsOnly_NilAssertion(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.ContainerEngineCLIEnabled, Status: false}
	execCmdNilAssertion(t, "results", "show", "--scan-id", "CONTAINERS_ONLY", "--report-format", "summaryConsole")
}
func TestRunGetResultsByScanIdGLSastAndAScaFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "gl-sast,gl-sca")
	// Run test for gl-sast report type
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatGLSast))
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatGLSca))
}

func TestRunGetResultsByScanIdGLScaFormat(t *testing.T) {
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK", "--report-format", "gl-sca")
	// Run test for gl-sca report type
	os.Remove(fmt.Sprintf("%s.%s", fileName, printer.FormatGLSca))
}

func Test_addPackageInformation(t *testing.T) {
	var dependencyPath = wrappers.DependencyPath{ID: "test-1"}
	var dependencyArray = [][]wrappers.DependencyPath{{dependencyPath}}
	resultsModel := &wrappers.ScanResultsCollection{
		Results: []*wrappers.ScanResult{
			{
				Type: "sca", // Assuming this matches commonParams.ScaType
				ScanResultData: wrappers.ScanResultData{
					PackageIdentifier: "pkg-123",
				},
				ID: "CVE-2021-23-424",
				VulnerabilityDetails: wrappers.VulnerabilityDetails{
					CvssScore: 5.0,
					CveName:   "cwe-789",
				},
			},
		},
	}
	scaPackageModel := &[]wrappers.ScaPackageCollection{
		{
			ID:                  "pkg-123",
			FixLink:             "",
			DependencyPathArray: dependencyArray,
		},
	}
	scaTypeModel := &[]wrappers.ScaTypeCollection{
		{}}

	resultsModel = addPackageInformation(resultsModel, scaPackageModel, scaTypeModel)

	expectedFixLink := "https://devhub.checkmarx.com/cve-details/CVE-2021-23-424"
	actualFixLink := resultsModel.Results[0].ScanResultData.ScaPackageCollection.FixLink
	assert.Equal(t, expectedFixLink, actualFixLink, "FixLink should match the result ID")
}

func TestRunGetResultsByScanIdGLSastFormat_NoVulnerabilities_Success(t *testing.T) {
	// Execute the command and perform nil assertion
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK_NO_VULNERABILITIES", "--report-format", "gl-sast")

	// Run test for gl-sast report type
	// Check if the file exists and vulnerabilities is empty, then delete the file
	if _, err := os.Stat(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSast)); err == nil {
		t.Logf("File exists: %s.%s", fileName, printer.FormatGLSast)
		resultsData, err := os.ReadFile(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSast))
		if err != nil {
			t.Logf("Failed to read file: %v", err)
		}

		var results wrappers.GlSastResultsCollection
		if err := json.Unmarshal(resultsData, &results); err != nil {
			t.Logf("Failed to unmarshal JSON: %v", err)
		}
		assert.Equal(t, len(results.Vulnerabilities), 0, "No vulnerabilities should be found")
		if err := os.Remove(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSast)); err != nil {
			t.Logf("Failed to delete file: %v", err)
		}
		t.Log("File deleted successfully.")
	}
}

func TestRunGetResultsByScanIdGLScaFormat_NoVulnerabilities_Success(t *testing.T) {
	// Execute the command and perform nil assertion
	execCmdNilAssertion(t, "results", "show", "--scan-id", "MOCK_NO_VULNERABILITIES", "--report-format", "gl-sca")

	// Run test for gl-sca report type
	// Check if the file exists and vulnerabilities is empty, then delete the file
	if _, err := os.Stat(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSca)); err == nil {
		t.Logf("File exists: %s.%s", fileName, printer.FormatGLSca)
		resultsData, err := os.ReadFile(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSca))
		if err != nil {
			t.Logf("Failed to read file: %v", err)
		}

		var results wrappers.GlScaResultsCollection
		if err := json.Unmarshal(resultsData, &results); err != nil {
			t.Logf("Failed to unmarshal JSON: %v", err)
		}
		assert.Equal(t, len(results.Vulnerabilities), 0, "No vulnerabilities should be found")
		if err := os.Remove(fmt.Sprintf("%s.%s-report.json", fileName, printer.FormatGLSca)); err != nil {
			t.Logf("Failed to delete file: %v", err)
		}
		t.Log("File deleted successfully.")
	}
}

func TestRunGetResultsByScanIdSummaryConsoleFormat_ScsNotScanned_ScsMissingInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = false
	mock.ScsScanPartial = false
	mock.ScorecardScanned = false
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.SCSEngineCLIEnabled, Status: true}
	buffer, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)

	stdoutString := buffer.String()
	fmt.Print(stdoutString)

	scsSummary := "| SCS               -       -        -       -      -       -       |"
	assert.Equal(t, strings.Contains(stdoutString, scsSummary), true,
		"Expected SCS summary:"+scsSummary)
	secretDetectionSummary := "Secret Detection"
	assert.Equal(t, !strings.Contains(stdoutString, secretDetectionSummary), true,
		"Expected Secret Detection summary to be missing:"+secretDetectionSummary)
	scorecardSummary := "Scorecard"
	assert.Equal(t, !strings.Contains(stdoutString, scorecardSummary), true,
		"Expected Scorecard summary to be missing:"+scorecardSummary)

	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSummaryConsoleFormat_ScsPartial_ScsPartialInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = true
	mock.ScorecardScanned = true
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.SCSEngineCLIEnabled, Status: true}

	buffer, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)

	stdoutString := buffer.String()
	ansiRegexp := regexp.MustCompile("\x1b\\[[0-9;]*[mK]")
	cleanString := ansiRegexp.ReplaceAllString(stdoutString, "")
	fmt.Print(stdoutString)

	TotalResults := "Total Results: 18"
	assert.Equal(t, strings.Contains(cleanString, TotalResults), true,
		"Expected: "+TotalResults)
	TotalSummary := "| TOTAL             0      10        5       3      0   Completed   |"
	assert.Equal(t, strings.Contains(cleanString, TotalSummary), true,
		"Expected TOTAL summary: "+TotalSummary)
	scsSummary := "| SCS               0       5        3       2      0   Partial     |"
	assert.Equal(t, strings.Contains(cleanString, scsSummary), true,
		"Expected SCS summary:"+scsSummary)
	secretDetectionSummary := secretDetectionLine
	assert.Equal(t, strings.Contains(cleanString, secretDetectionSummary), true,
		"Expected Secret Detection summary:"+secretDetectionSummary)
	scorecardSummary := " | Scorecard                 0      0        0      0      0   Failed     |"
	assert.Equal(t, strings.Contains(cleanString, scorecardSummary), true,
		"Expected Scorecard summary:"+scorecardSummary)

	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSummaryConsoleFormat_ScsScorecardNotScanned_ScorecardMissingInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = false
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.SCSEngineCLIEnabled, Status: true}

	buffer, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)

	stdoutString := buffer.String()
	fmt.Print(stdoutString)

	scsSummary := "| SCS               0       5        3       2      0   Completed   |"
	assert.Equal(t, strings.Contains(stdoutString, scsSummary), true,
		"Expected SCS summary:"+scsSummary)
	secretDetectionSummary := secretDetectionLine
	assert.Equal(t, strings.Contains(stdoutString, secretDetectionSummary), true,
		"Expected Secret Detection summary:"+secretDetectionSummary)
	scorecardSummary := "| Scorecard                 -      -        -      -      -       -      |"
	assert.Equal(t, strings.Contains(stdoutString, scorecardSummary), true,
		"Expected Scorecard summary:"+scorecardSummary)

	mock.SetScsMockVarsToDefault()
}

func TestRunGetResultsByScanIdSummaryConsoleFormat_SCSFlagNotEnabled_SCSMissingInReport(t *testing.T) {
	clearFlags()
	mock.HasScs = true
	mock.ScsScanPartial = false
	mock.ScorecardScanned = true
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.SCSEngineCLIEnabled, Status: false}

	buffer, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)

	stdoutString := buffer.String()
	fmt.Print(stdoutString)

	scsSummary := "| SCS"
	assert.Equal(t, !strings.Contains(stdoutString, scsSummary), true,
		"Expected SCS summary:"+scsSummary)
	secretDetectionSummary := "Secret Detection"
	assert.Equal(t, !strings.Contains(stdoutString, secretDetectionSummary), true,
		"Expected Secret Detection summary to be missing:"+secretDetectionSummary)
	scorecardSummary := "Scorecard"
	assert.Equal(t, !strings.Contains(stdoutString, scorecardSummary), true,
		"Expected Scorecard summary to be missing:"+scorecardSummary)

	mock.SetScsMockVarsToDefault()
}

func TestGetResultsSummaryConsoleFormatWithCriticalDisabled(t *testing.T) {
	clearFlags()
	mock.Flag = wrappers.FeatureFlagResponseModel{Name: wrappers.CVSSV3Enabled, Status: false}
	buffer, err := executeRedirectedOsStdoutTestCommand(createASTTestCommand(),
		"results", "show", "--scan-id", "MOCK", "--report-format", "summaryConsole")
	assert.NilError(t, err)

	stdoutString := buffer.String()
	fmt.Print(stdoutString)

	totalSummary := "| TOTAL           N/A       5        1       1      0   Completed   |"
	assert.Equal(t, strings.Contains(stdoutString, totalSummary), true,
		"Expected Total summary without critical:"+totalSummary)

	mock.SetScsMockVarsToDefault()
}

func Test_enhanceWithScanSummary(t *testing.T) {
	tests := []struct {
		name                string
		summary             *wrappers.ResultSummary
		results             *wrappers.ScanResultsCollection
		featureFlagsWrapper wrappers.FeatureFlagsWrapper
		expectedIssues      int
	}{
		{
			name:    "scan summary with no vulnerabilities",
			summary: createEmptyResultSummary(),
			results: &wrappers.ScanResultsCollection{
				Results:    nil,
				TotalCount: 0,
				ScanID:     "MOCK",
			},
			featureFlagsWrapper: mock.FeatureFlagsMockWrapper{},
			expectedIssues:      0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			enhanceWithScanSummary(tt.summary, tt.results, tt.featureFlagsWrapper)
			assert.Equal(t, tt.expectedIssues, tt.summary.TotalIssues)
		})
	}
}

func createEmptyResultSummary() *wrappers.ResultSummary {
	return &wrappers.ResultSummary{
		TotalIssues:    0,
		CriticalIssues: 0,
		HighIssues:     0,
		MediumIssues:   0,
		LowIssues:      0,
		InfoIssues:     0,
		SastIssues:     0,
		ScaIssues:      0,
		KicsIssues:     0,
		ScsIssues:      0,
		SCSOverview:    wrappers.SCSOverview{},
		APISecurity: wrappers.APISecResult{
			APICount:        0,
			TotalRisksCount: 0,
			Risks:           []int{0, 0, 0, 0},
			StatusCode:      0,
		},
		EnginesEnabled: []string{"sast", "sca", "kics", "containers"},
		EnginesResult: wrappers.EnginesResultsSummary{
			params.SastType: &wrappers.EngineResultSummary{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
				Info:     0,
			},
			params.ScaType: &wrappers.EngineResultSummary{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
				Info:     0,
			},
			params.KicsType: &wrappers.EngineResultSummary{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
				Info:     0,
			},
			params.APISecType: &wrappers.EngineResultSummary{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
			},
			params.ContainersType: &wrappers.EngineResultSummary{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
			},
		},
	}
}
